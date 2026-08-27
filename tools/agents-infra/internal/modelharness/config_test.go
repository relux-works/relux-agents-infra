package modelharness

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolveLocalProfile(t *testing.T) {
	config := writeConfig(t, `
[profiles.qwen-local]
mode = "local"
executable = "/bin/echo"
argv = ["serve", "--host", "{host}", "--port", "{port}"]
`)
	plan, err := Resolve(config, "qwen-local", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if plan.Mode != "local" || plan.Executable != "/bin/echo" || plan.Endpoint != "http://127.0.0.1:18011/v1" {
		t.Fatalf("plan = %#v", plan)
	}
	wantArgv := []string{"serve", "--host", "127.0.0.1", "--port", "18011"}
	if !reflect.DeepEqual(plan.Argv, wantArgv) {
		t.Fatalf("argv = %#v, want %#v", plan.Argv, wantArgv)
	}
}

func TestResolveLocalStressPolicy(t *testing.T) {
	config := writeConfig(t, `
[profiles.qwen-local]
mode = "local"
executable = "/bin/echo"
argv = ["serve", "--host", "{host}", "--port", "{port}"]

[profiles.qwen-local.stress]
prompt_tokens = 50000
max_output_tokens = 1
startup_timeout_seconds = 120
request_timeout_seconds = 600
sample_interval_milliseconds = 250
`)
	plan, err := Resolve(config, "qwen-local", "127.0.0.1", 18011)
	if err != nil {
		t.Fatal(err)
	}
	want := &StressPolicy{PromptTokens: 50000, MaxOutputTokens: 1, StartupTimeoutSeconds: 120, RequestTimeoutSeconds: 600, SampleIntervalMilliseconds: 250}
	if !reflect.DeepEqual(plan.Stress, want) {
		t.Fatalf("stress policy=%#v want=%#v", plan.Stress, want)
	}
}

func TestResolveLocalSupervisionPolicy(t *testing.T) {
	config := writeConfig(t, `
[profiles.qwen-local]
mode = "local"
executable = "/bin/echo"
argv = ["serve", "--host", "{host}", "--port", "{port}"]

[profiles.qwen-local.supervision]
fatal_output_substrings = ["Resource limit (499000) exceeded", "Exception in thread"]
restart_on_failure = true
max_restarts = 3
restart_window_seconds = 3600
restart_delay_milliseconds = 1000
`)
	plan, err := Resolve(config, "qwen-local", "127.0.0.1", 18011)
	if err != nil {
		t.Fatal(err)
	}
	want := &SupervisionPolicy{
		FatalOutputSubstrings:    []string{"Resource limit (499000) exceeded", "Exception in thread"},
		RestartOnFailure:         true,
		MaxRestarts:              3,
		RestartWindowSeconds:     3600,
		RestartDelayMilliseconds: 1000,
	}
	if !reflect.DeepEqual(plan.Supervision, want) {
		t.Fatalf("supervision policy=%#v want=%#v", plan.Supervision, want)
	}
}

func TestResolveSSHProfile(t *testing.T) {
	config := writeConfig(t, `
[profiles.qwen-remote]
mode = "ssh"
ssh_executable = "/usr/bin/ssh"
ssh_target = "dedicated-mac"
remote_executable = "/Users/remote/.local/bin/model-harness"
remote_config = "/Users/remote/.config/model-harness/config.toml"
remote_profile = "qwen-tiny"
remote_host = "127.0.0.1"
remote_port = 18012
`)
	plan, err := Resolve(config, "qwen-remote", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if plan.Mode != "ssh" || plan.Executable != "/usr/bin/ssh" || plan.Remote == nil {
		t.Fatalf("plan = %#v", plan)
	}
	if len(plan.Argv) == 0 || plan.Argv[0] != "-tt" {
		t.Fatalf("ssh run must force a remote PTY for disconnect cleanup: %#v", plan.Argv)
	}
	joined := strings.Join(plan.Argv, " ")
	for _, want := range []string{
		"127.0.0.1:18011:127.0.0.1:18012",
		"dedicated-mac",
		"'/Users/remote/.local/bin/model-harness' 'run' 'qwen-tiny'",
		"'--config' '/Users/remote/.config/model-harness/config.toml'",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv %q missing %q", joined, want)
		}
	}
}

func TestResolveRejectsUnsafeOrIncompleteProfiles(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "non-loopback remote",
			body: `
[profiles.remote]
mode = "ssh"
ssh_target = "host"
remote_executable = "/bin/model-harness"
remote_profile = "tiny"
remote_host = "0.0.0.0"
remote_port = 18011
`,
			want: "remote_host must equal 127.0.0.1",
		},
		{
			name: "embedded placeholder",
			body: `
[profiles.local]
mode = "local"
executable = "/bin/echo"
argv = ["--host={host}", "--port", "{port}"]
`,
			want: "placeholders must be whole argv tokens",
		},
		{
			name: "unknown field",
			body: `
[profiles.local]
mode = "local"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]
shell = true
`,
			want: "strict mode",
		},
		{
			name: "incomplete stress policy",
			body: `
[profiles.local]
mode = "local"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.local.stress]
prompt_tokens = 50000
`,
			want: "stress.max_output_tokens",
		},
		{
			name: "ssh stress policy",
			body: `
[profiles.remote]
mode = "ssh"
ssh_target = "host"
remote_executable = "/bin/model-harness"
remote_profile = "tiny"
remote_host = "127.0.0.1"
remote_port = 18011

[profiles.remote.stress]
prompt_tokens = 50000
max_output_tokens = 1
startup_timeout_seconds = 120
request_timeout_seconds = 600
sample_interval_milliseconds = 250
`,
			want: "stress is currently supported only for local mode",
		},
		{
			name: "empty fatal output substring",
			body: `
[profiles.local]
mode = "local"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.local.supervision]
fatal_output_substrings = [""]
max_restarts = 3
restart_window_seconds = 3600
restart_delay_milliseconds = 1000
`,
			want: "fatal_output_substrings",
		},
		{
			name: "ssh supervision policy",
			body: `
[profiles.remote]
mode = "ssh"
ssh_target = "host"
remote_executable = "/bin/model-harness"
remote_profile = "tiny"
remote_host = "127.0.0.1"
remote_port = 18011

[profiles.remote.supervision]
fatal_output_substrings = ["fatal"]
max_restarts = 3
restart_window_seconds = 3600
restart_delay_milliseconds = 1000
`,
			want: "supervision is currently supported only for local mode",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(writeConfig(t, test.body), strings.Split(test.body, "[profiles.")[1][:strings.Index(strings.Split(test.body, "[profiles.")[1], "]")], "127.0.0.1", 18011)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestResolveRejectsNonLoopbackPresentedEndpoint(t *testing.T) {
	config := writeConfig(t, `
[profiles.local]
mode = "local"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]
`)
	_, err := Resolve(config, "local", "0.0.0.0", 18011)
	if err == nil || !strings.Contains(err.Error(), "host must equal 127.0.0.1") {
		t.Fatalf("error = %v", err)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
