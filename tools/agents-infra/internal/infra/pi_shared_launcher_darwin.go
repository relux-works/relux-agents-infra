//go:build darwin

package infra

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type SharedRuntimeLauncherOptions struct {
	RuntimeKey     string
	ProfileProject string
	ProfileName    string
	HomeDir        string
	CacheRoot      string
	Environ        []string
}

var (
	sharedRuntimeSetNonblock   = unix.SetNonblock
	sharedRuntimeExecve        = unix.Exec
	sharedAuthEvidenceObserver func(sharedAuthDecodeEvidence)
)

func RunSharedRuntimeLauncher(options SharedRuntimeLauncherOptions) error {
	var stat unix.Stat_t
	if err := unix.Fstat(3, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFIFO {
		if err == nil {
			err = errors.New("authorization descriptor is not a FIFO")
		}
		return sharedRuntimeReason("runtime_launch_unauthorized", "no_authorization_descriptor", err)
	}

	resolved, err := resolveSharedProfile(options.ProfileProject, options.HomeDir, options.CacheRoot, options.ProfileName)
	if err != nil {
		return &SharedRuntimeError{Code: "runtime_launch_identity_unresolvable", Err: err}
	}
	if resolved.RuntimeKey != options.RuntimeKey {
		return sharedRuntimeError("runtime_launch_identity_mismatch", errors.New("recomputed runtime key differs from --runtime-key"))
	}
	execPlanDigest := SharedRuntimeExecPlanDigest(resolved.Profile, resolved.Paths.RuntimeCWD)

	if err := sharedRuntimeSetNonblock(3, true); err != nil {
		return sharedRuntimeReason("runtime_launch_unauthorized", "deadline_unavailable", err)
	}
	raw, reason, err := readSharedAuthorizationFrame(3, time.Duration(resolved.Profile.Runtime.StartupTimeoutSeconds)*time.Second)
	if err != nil {
		if reason == "protocol_violation" {
			return sharedRuntimeReason("protocol_violation", "frame_unparseable", err)
		}
		return sharedRuntimeReason("runtime_launch_unauthorized", reason, err)
	}
	frame, evidence, err := decodeSharedRuntimeAuthorizationFrame(raw)
	if err != nil {
		if sharedAuthEvidenceObserver != nil {
			sharedAuthEvidenceObserver(evidence)
		}
		return err
	}
	comparisons := []struct {
		name  string
		equal bool
	}{
		{name: "schema", equal: frame.Schema == sharedRuntimeAuthSchema},
		{name: "protocol_version", equal: frame.ProtocolVersion == SharedRuntimeProtocolVersion},
		{name: "launcher_pid", equal: frame.LauncherPID == os.Getpid()},
		{name: "runtime_key", equal: frame.RuntimeKey == options.RuntimeKey},
		{name: "exec_plan_digest", equal: frame.ExecPlanDigest == execPlanDigest},
	}
	for _, comparison := range comparisons {
		evidence.ComparedFields = append(evidence.ComparedFields, comparison.name)
		if !comparison.equal {
			evidence.Decision = "refuse"
			if sharedAuthEvidenceObserver != nil {
				sharedAuthEvidenceObserver(evidence)
			}
			return sharedRuntimeMismatch("runtime_authorization_mismatch", comparison.name)
		}
	}
	if sharedAuthEvidenceObserver != nil {
		sharedAuthEvidenceObserver(evidence)
	}
	environ := options.Environ
	if environ == nil {
		environ = os.Environ()
	}
	argv := append([]string{resolved.Profile.Runtime.Executable}, resolved.Profile.Runtime.Argv...)
	if err := sharedRuntimeExecve(resolved.Profile.Runtime.Executable, argv, environ); err != nil {
		return sharedRuntimeError("runtime_start_failed", err)
	}
	return nil
}

func readSharedAuthorizationFrame(fd int, timeout time.Duration) ([]byte, string, error) {
	deadline := time.Now().Add(timeout)
	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	temporary := make([]byte, 4096)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, "authorization_deadline", errors.New("authorization deadline elapsed")
		}
		milliseconds := int(remaining / time.Millisecond)
		if milliseconds < 1 {
			milliseconds = 1
		}
		pollFDs := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN | unix.POLLHUP | unix.POLLERR}}
		ready, err := unix.Poll(pollFDs, milliseconds)
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			return nil, "protocol_violation", err
		}
		if ready == 0 {
			continue
		}
		count, err := unix.Read(fd, temporary)
		if count > 0 {
			if buffer.Len()+count > sharedRuntimeMaxFrameBytes {
				return nil, "protocol_violation", errors.New("authorization frame exceeds 65536 bytes")
			}
			buffer.Write(temporary[:count])
			if newline := bytes.IndexByte(buffer.Bytes(), '\n'); newline >= 0 {
				if len(bytes.TrimSpace(buffer.Bytes()[newline+1:])) != 0 {
					return nil, "protocol_violation", errors.New("content follows authorization frame")
				}
				return append([]byte(nil), buffer.Bytes()[:newline]...), "", nil
			}
		}
		if err != nil && !errors.Is(err, syscall.EAGAIN) && !errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, "protocol_violation", err
		}
		if count == 0 {
			if buffer.Len() == 0 {
				return nil, "broker_died_before_authorizing", io.EOF
			}
			return nil, "protocol_violation", fmt.Errorf("authorization frame ended before newline: %w", io.ErrUnexpectedEOF)
		}
	}
}
