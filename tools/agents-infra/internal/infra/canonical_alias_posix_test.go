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

func TestDirectProviderYoloAliasesDelegateOnceAndPreserveCWDArgv(t *testing.T) {
	for _, launcher := range directProviderYoloLaunchers {
		t.Run(launcher.name, func(t *testing.T) {
			binDir := t.TempDir()
			project := t.TempDir()
			record := filepath.Join(t.TempDir(), "record")
			layout := Layout{Mode: ModeLocal, BinDir: binDir}
			mustWrite(t, filepath.Join(binDir, "agents-infra"), "#!/bin/sh\nprintf '%s\\0' \"$PWD\" \"$@\" > \""+record+"\"\n")
			if err := installDirectProviderYoloLaunchers(layout, io.Discard); err != nil {
				t.Fatalf("installDirectProviderYoloLaunchers: %v", err)
			}

			args := []string{"--print-config", "", "a b", "line 1\nline 2", "tab\tvalue", "Հայերեն", "--", "-d"}
			cmd := exec.Command(filepath.Join(binDir, launcher.name), args...)
			cmd.Dir = project
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", launcher.name, err, output)
			}
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			fields := bytes.Split(bytes.TrimSuffix(data, []byte{0}), []byte{0})
			got := make([]string, len(fields))
			for i, field := range fields {
				got[i] = string(field)
			}
			want := append([]string{project, "target-yolo", launcher.canonicalTarget}, args...)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("alias record = %#v, want %#v", got, want)
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
