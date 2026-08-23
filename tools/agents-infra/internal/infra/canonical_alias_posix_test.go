//go:build !windows

package infra

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCanonicalAliasReachesSiblingTargetAndPreservesCWDArgv(t *testing.T) {
	binDir := t.TempDir()
	project := t.TempDir()
	record := filepath.Join(t.TempDir(), "record")
	layout := Layout{Mode: ModeLocal, BinDir: binDir}
	mustWrite(t, filepath.Join(binDir, "agents-infra"), "#!/bin/sh\npwd > \""+record+"\"\nprintf '%s\\n' \"$@\" >> \""+record+"\"\n")
	if err := installCanonicalTargetLaunchers(layout, io.Discard); err != nil {
		t.Fatalf("installCanonicalTargetLaunchers: %v", err)
	}

	alias := filepath.Join(binDir, "openai-infra")
	cmd := exec.Command(alias, "--print-config", "--", "a b")
	cmd.Dir = project
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("alias: %v\n%s", err, output)
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{project, "target", "openai-infra", "--print-config", "--", "a b"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("alias record = %#v, want %#v", lines, want)
	}
}

func TestCanonicalAliasRefusesMissingAndNonRegularSibling(t *testing.T) {
	for _, state := range []string{"missing", "non_regular", "non_executable"} {
		t.Run(state, func(t *testing.T) {
			binDir := t.TempDir()
			layout := Layout{Mode: ModeLocal, BinDir: binDir}
			target := filepath.Join(binDir, "agents-infra")
			mustWrite(t, target, "#!/bin/sh\nexit 0\n")
			if err := installCanonicalTargetLaunchers(layout, io.Discard); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "missing", "non_regular":
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
			}
			if state == "non_regular" {
				if err := os.Mkdir(target, 0o755); err != nil {
					t.Fatal(err)
				}
			} else if state == "non_executable" {
				if err := os.Chmod(target, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			output, err := exec.Command(filepath.Join(binDir, "qwen-infra"), "--print-config").CombinedOutput()
			if err == nil || !strings.Contains(string(output), "missing or non-regular sibling") {
				t.Fatalf("alias error = %v output=%s", err, output)
			}
		})
	}
}
