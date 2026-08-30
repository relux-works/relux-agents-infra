package infra

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/agentic"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
)

const processACleanupTimeout = 5 * time.Second

type ProcessATurnRunner interface {
	RunProcessATurn(context.Context, agentic.Plan) (managementpi.TurnResultInput, error)
}

type OSProcessATurnRunner struct{}

// BuildAndRunPiTurn is the production consumer boundary. It obtains the exact
// Process-A plan through vendorplugin.BuildLaunch, waits for the actual child,
// and delegates every schema-1 decision to the sole upstream classifier.
func BuildAndRunPiTurn(ctx context.Context, registry *vendorplugin.Registry, request vendorplugin.SpawnRequest, runner ProcessATurnRunner) (managementpi.TurnResult, error) {
	plan, err := vendorplugin.BuildLaunch(ctx, registry, request, agentic.LaunchModeExec)
	if err != nil {
		return managementpi.TurnResult{}, err
	}
	if runner == nil {
		runner = OSProcessATurnRunner{}
	}
	input, err := runner.RunProcessATurn(ctx, plan)
	if err != nil {
		return managementpi.TurnResult{}, err
	}
	return managementpi.ValidateTurnResult(input)
}

func (OSProcessATurnRunner) RunProcessATurn(ctx context.Context, plan agentic.Plan) (managementpi.TurnResultInput, error) {
	if err := ctx.Err(); err != nil {
		return managementpi.TurnResultInput{}, err
	}
	command := exec.Command(plan.Binary, plan.Argv...)
	command.Dir = plan.WorkDir
	command.Env = append([]string(nil), plan.Env...)
	if plan.Stdin.Attached {
		command.Stdin = bytes.NewReader(append([]byte(nil), plan.Stdin.Bytes...))
	}
	stdout := newBoundedProcessAStdout(managementpi.TurnResultMaxStdoutBytes)
	command.Stdout = stdout
	command.Stderr = io.Discard
	configureProcessACommand(command)
	if err := command.Start(); err != nil {
		return managementpi.TurnResultInput{}, err
	}

	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	waitErr, intervention, cleanup := waitForProcessA(ctx, command, waited)
	return managementpi.TurnResultInput{
		Stdout:          stdout.Bytes(),
		StdoutTruncated: stdout.Truncated(),
		Exit:            processAExit(waitErr),
		Intervention:    intervention,
		Cleanup:         cleanup,
	}, nil
}

func waitForProcessA(ctx context.Context, command *exec.Cmd, waited <-chan error) (error, managementpi.TurnIntervention, managementpi.ProcessACleanupOutcome) {
	select {
	case err := <-waited:
		return err, managementpi.TurnInterventionNone, managementpi.ProcessACleanupNotRequired
	case <-ctx.Done():
		select {
		case err := <-waited:
			return err, managementpi.TurnInterventionNone, managementpi.ProcessACleanupNotRequired
		default:
		}
		intervention := managementpi.TurnInterventionCancel
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			intervention = managementpi.TurnInterventionDeadline
		}
		waitErr, cleanup := stopProcessA(command, waited, processACleanupTimeout)
		return waitErr, intervention, cleanup
	}
}

type boundedProcessAStdout struct {
	mu        sync.Mutex
	limit     int
	data      []byte
	truncated bool
}

func newBoundedProcessAStdout(limit int) *boundedProcessAStdout {
	return &boundedProcessAStdout{limit: limit, data: make([]byte, 0, limit)}
}

func (b *boundedProcessAStdout) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		b.data = append(b.data, p[:remaining]...)
	}
	if remaining < len(p) {
		b.truncated = true
	}
	return len(p), nil
}

func (b *boundedProcessAStdout) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.data...)
}

func (b *boundedProcessAStdout) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}
