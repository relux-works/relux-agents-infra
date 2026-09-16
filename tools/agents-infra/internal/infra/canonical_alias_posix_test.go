//go:build !windows

package infra

import (
	"bytes"
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

	// Only the live qwen-infra alias still delegates; the deprecated
	// openai/anthropic aliases refuse without touching the sibling.
	alias := filepath.Join(binDir, "qwen-infra")
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
	want := []string{project, "target", "qwen-infra", "--print-config", "--", "a b"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("alias record = %#v, want %#v", lines, want)
	}
}

func TestDeprecatedCanonicalAliasesRefuseWithoutSibling(t *testing.T) {
	for _, entrypoint := range []string{"openai-infra", "anthropic-infra"} {
		t.Run(entrypoint, func(t *testing.T) {
			message, ok := DeprecatedProviderMessage(entrypoint)
			if !ok {
				t.Fatalf("DeprecatedProviderMessage(%q) missing", entrypoint)
			}
			binDir := t.TempDir()
			project := t.TempDir()
			record := filepath.Join(t.TempDir(), "record")
			layout := Layout{Mode: ModeLocal, BinDir: binDir}
			// A sibling that would record delegation if the alias reached it.
			// The deprecated alias must refuse before any sibling lookup, so
			// the record must stay absent even when the sibling exists, and
			// the refusal must be identical when it is missing.
			mustWrite(t, filepath.Join(binDir, "agents-infra"), "#!/bin/sh\necho delegated >> \""+record+"\"\n")
			if err := installCanonicalTargetLaunchers(layout, io.Discard); err != nil {
				t.Fatalf("installCanonicalTargetLaunchers: %v", err)
			}
			for _, args := range [][]string{nil, {"--print-config", "-d", "--danger"}} {
				for _, sibling := range []string{"present", "missing"} {
					if sibling == "missing" {
						if err := os.Remove(filepath.Join(binDir, "agents-infra")); err != nil && !os.IsNotExist(err) {
							t.Fatal(err)
						}
					}
					cmd := exec.Command(filepath.Join(binDir, entrypoint), args...)
					cmd.Dir = project
					var stdout, stderr bytes.Buffer
					cmd.Stdout = &stdout
					cmd.Stderr = &stderr
					runErr := cmd.Run()
					exitErr, ok := runErr.(*exec.ExitError)
					if !ok || exitErr.ExitCode() != 1 {
						t.Fatalf("%s %q exit = %v, want exit 1 (stdout=%q stderr=%q)", entrypoint, args, runErr, stdout.String(), stderr.String())
					}
					if stdout.String() != "" {
						t.Fatalf("%s %q stdout = %q, want empty", entrypoint, args, stdout.String())
					}
					if stderr.String() != message+"\n" {
						t.Fatalf("%s %q stderr = %q, want exactly %q", entrypoint, args, stderr.String(), message+"\n")
					}
					if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
						t.Fatalf("%s %q reached the sibling target", entrypoint, args)
					}
				}
			}
		})
	}
}

func TestDeprecatedDirectProviderYoloAliasesRefuseWithoutSibling(t *testing.T) {
	for _, launcher := range directProviderYoloLaunchers {
		t.Run(launcher.name, func(t *testing.T) {
			message, ok := DeprecatedProviderMessage(launcher.name)
			if !ok {
				t.Fatalf("DeprecatedProviderMessage(%q) missing", launcher.name)
			}
			binDir := t.TempDir()
			project := t.TempDir()
			record := filepath.Join(t.TempDir(), "record")
			layout := Layout{Mode: ModeLocal, BinDir: binDir}
			mustWrite(t, filepath.Join(binDir, "agents-infra"), "#!/bin/sh\necho delegated >> \""+record+"\"\n")
			if err := installDirectProviderYoloLaunchers(layout, io.Discard); err != nil {
				t.Fatalf("installDirectProviderYoloLaunchers: %v", err)
			}
			args := []string{"--print-config", "", "a b", "line 1\nline 2", "tab\tvalue", "Հայերեն", "--", "-d"}
			for _, sibling := range []string{"present", "missing"} {
				if sibling == "missing" {
					if err := os.Remove(filepath.Join(binDir, "agents-infra")); err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
				}
				cmd := exec.Command(filepath.Join(binDir, launcher.name), args...)
				cmd.Dir = project
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				runErr := cmd.Run()
				exitErr, ok := runErr.(*exec.ExitError)
				if !ok || exitErr.ExitCode() != 1 {
					t.Fatalf("%s exit = %v, want exit 1 (stdout=%q stderr=%q)", launcher.name, runErr, stdout.String(), stderr.String())
				}
				if stdout.String() != "" {
					t.Fatalf("%s stdout = %q, want empty", launcher.name, stdout.String())
				}
				if stderr.String() != message+"\n" {
					t.Fatalf("%s stderr = %q, want exactly %q", launcher.name, stderr.String(), message+"\n")
				}
				if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
					t.Fatalf("%s reached the sibling target", launcher.name)
				}
			}
		})
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
