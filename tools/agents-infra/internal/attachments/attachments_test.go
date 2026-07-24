package attachments

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStageImagesAcceptsPathsAndManifestRefsWithRedactedMapping(t *testing.T) {
	root := t.TempDir()
	explicit := filepath.Join(root, "explicit.png")
	mustWriteBytes(t, explicit, []byte("explicit-image"))
	secretImage := filepath.Join(root, "sim-8912345678901234567.png")
	mustWriteBytes(t, secretImage, []byte("secret-image"))
	manifest := filepath.Join(root, "manifest.json")
	outDir := filepath.Join(root, "stage")
	writeManifestForTest(t, manifest, []map[string]any{
		{
			"id":         "photo-ref",
			"name":       filepath.Base(secretImage),
			"mime_type":  "image/png",
			"size_bytes": mustSize(t, secretImage),
			"local_path": secretImage,
		},
	})

	stdout := runHelper(t, []string{
		"stage-images",
		"--manifest", manifest,
		"--out-dir", outDir,
		explicit,
		"photo-ref",
	})

	payload := decodeObject(t, stdout)
	mappingPath := payload["mapping_path"].(string)
	assertExists(t, mappingPath)
	items := payload["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2\n%s", len(items), stdout)
	}
	if got := nestedString(items[0], "source", "kind"); got != "path" {
		t.Fatalf("first source kind = %q", got)
	}
	if got := nestedString(items[1], "source", "kind"); got != "manifest" {
		t.Fatalf("second source kind = %q", got)
	}
	if got := mustReadBytes(t, explicit); string(got) != "explicit-image" {
		t.Fatalf("explicit source mutated: %q", got)
	}
	if got := mustReadBytes(t, secretImage); string(got) != "secret-image" {
		t.Fatalf("manifest source mutated: %q", got)
	}

	mappingText := string(mustReadBytes(t, mappingPath))
	if strings.Contains(mappingText, "8912345678901234567") {
		t.Fatalf("mapping contains raw ICCID-like value:\n%s", mappingText)
	}
	if !strings.Contains(mappingText, "REDACTED_ICCID") {
		t.Fatalf("mapping missing redaction marker:\n%s", mappingText)
	}
	for _, raw := range items {
		item := raw.(map[string]any)
		staged := item["staged"].(map[string]any)
		assertExists(t, staged["path"].(string))
		if item["action"] != "copied" {
			t.Fatalf("action = %v, want copied", item["action"])
		}
		if item["source_read_only"] != true {
			t.Fatalf("source_read_only = %v, want true", item["source_read_only"])
		}
	}
}

func TestStageImagesAllFiltersManifestToImages(t *testing.T) {
	root := t.TempDir()
	image := filepath.Join(root, "image.png")
	mustWriteBytes(t, image, []byte("image"))
	text := filepath.Join(root, "notes.txt")
	mustWriteBytes(t, text, []byte("not an image"))
	manifest := filepath.Join(root, "manifest.json")
	writeManifestForTest(t, manifest, []map[string]any{
		{"id": "image", "name": filepath.Base(image), "mime_type": "image/png", "local_path": image},
		{"id": "text", "name": filepath.Base(text), "mime_type": "text/plain", "local_path": text},
	})

	stdout := runHelper(t, []string{"stage-images", "--manifest", manifest, "--out-dir", filepath.Join(root, "stage"), "--all"})

	payload := decodeObject(t, stdout)
	items := payload["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items length = %d, want 1\n%s", len(items), stdout)
	}
	if got := nestedString(items[0], "source", "manifest", "id"); got != "image" {
		t.Fatalf("manifest id = %q, want image", got)
	}
}

func TestHEICConverterPrefersSipsOnMacOS(t *testing.T) {
	fakeWhich := func(name string) (string, error) {
		switch name {
		case "sips":
			return "/usr/bin/sips", nil
		case "magick":
			return "/opt/homebrew/bin/magick", nil
		default:
			return "", os.ErrNotExist
		}
	}
	converter, executable, err := chooseHEICConverter(fakeWhich, "darwin")
	if err != nil {
		t.Fatalf("chooseHEICConverter: %v", err)
	}
	if converter != "sips" || executable != "/usr/bin/sips" {
		t.Fatalf("converter/executable = %q/%q", converter, executable)
	}
}

func TestHEICConverterUsesImageMagickFallback(t *testing.T) {
	fakeWhich := func(name string) (string, error) {
		if name == "magick" {
			return "/usr/local/bin/magick", nil
		}
		return "", os.ErrNotExist
	}
	converter, executable, err := chooseHEICConverter(fakeWhich, "linux")
	if err != nil {
		t.Fatalf("chooseHEICConverter: %v", err)
	}
	if converter != "imagemagick" || executable != "/usr/local/bin/magick" {
		t.Fatalf("converter/executable = %q/%q", converter, executable)
	}
}

func TestStageImagesHEICUsesDetectedPortableFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake magick is Unix-only")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	magick := filepath.Join(binDir, "magick")
	mustWriteBytes(t, magick, []byte("#!/bin/sh\ncp \"$1\" \"$2\"\n"))
	if err := os.Chmod(magick, 0o755); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	heic := filepath.Join(root, "sample.heic")
	mustWriteBytes(t, heic, []byte("fake-heic"))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("AGENTS_ATTACHMENTS_PLATFORM", "linux")

	stdout := runHelper(t, []string{"stage-images", "--out-dir", filepath.Join(root, "stage"), heic})

	payload := decodeObject(t, stdout)
	items := payload["items"].([]any)
	item := items[0].(map[string]any)
	if item["action"] != "normalized" {
		t.Fatalf("action = %v, want normalized", item["action"])
	}
	if got := nestedString(item, "normalization", "converter"); got != "imagemagick" {
		t.Fatalf("converter = %q, want imagemagick", got)
	}
	if got := mustReadBytes(t, nestedString(item, "staged", "path")); string(got) != "fake-heic" {
		t.Fatalf("staged bytes = %q", got)
	}
	if got := mustReadBytes(t, heic); string(got) != "fake-heic" {
		t.Fatalf("source bytes = %q", got)
	}
}

func TestMaterializeReadsCodexRolloutInputImages(t *testing.T) {
	root := t.TempDir()
	session := filepath.Join(root, "rollout-thread.jsonl")
	payload := []byte("fake-png")
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload)
	line := map[string]any{
		"type": "response_item",
		"payload": map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{"type": "input_text", "text": "<image name=[Evidence Shot]>"},
				{"type": "input_image", "image_url": dataURL},
			},
		},
	}
	lineBytes, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	mustWriteBytes(t, session, append(lineBytes, '\n'))
	manifest := filepath.Join(root, "manifest.json")
	outDir := filepath.Join(root, "attachments")

	stdout := runHelper(t, []string{"materialize", "--session", session, "--out-dir", outDir, "--manifest", manifest})

	if strings.TrimSpace(stdout) != manifest {
		t.Fatalf("stdout = %q, want %q", stdout, manifest)
	}
	manifestPayload := decodeObject(t, string(mustReadBytes(t, manifest)))
	attachments := manifestPayload["attachments"].([]any)
	if len(attachments) != 1 {
		t.Fatalf("attachments length = %d, want 1", len(attachments))
	}
	item := attachments[0].(map[string]any)
	if item["id"] != "attachment-001" || item["name"] != "Evidence-Shot.png" || item["mime_type"] != "image/png" {
		t.Fatalf("attachment item = %#v", item)
	}
	if got := mustReadBytes(t, item["local_path"].(string)); string(got) != "fake-png" {
		t.Fatalf("materialized bytes = %q", got)
	}
}

func TestFindRolloutPathUsesLegacyThreadIDSuffixMatching(t *testing.T) {
	home := t.TempDir()
	sessions := filepath.Join(home, ".codex", "sessions", "2026", "07", "21")
	matching := filepath.Join(sessions, "rollout-run-needle.jsonl")
	unrelated := filepath.Join(sessions, "rollout-run-needle-unrelated.jsonl")
	mustWriteBytes(t, matching, []byte("matching rollout"))
	mustWriteBytes(t, unrelated, []byte("unrelated rollout"))

	now := time.Now()
	if err := os.Chtimes(matching, now, now); err != nil {
		t.Fatalf("Chtimes matching rollout: %v", err)
	}
	if err := os.Chtimes(unrelated, now.Add(time.Second), now.Add(time.Second)); err != nil {
		t.Fatalf("Chtimes unrelated rollout: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	got, err := findRolloutPath("needle")
	if err != nil {
		t.Fatalf("findRolloutPath: %v", err)
	}
	if got != matching {
		t.Fatalf("findRolloutPath selected %q, want exact legacy suffix match %q", got, matching)
	}
}

func TestPathResolvesByIDNameAndLocalBasename(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "photo.png")
	mustWriteBytes(t, local, []byte("image"))
	manifest := filepath.Join(root, "manifest.json")
	writeManifestForTest(t, manifest, []map[string]any{
		{"id": "image-id", "name": "display.png", "mime_type": "image/png", "local_path": local},
	})

	for _, ref := range []string{"image-id", "display.png", "photo.png"} {
		stdout := runHelper(t, []string{"path", ref, "--manifest", manifest})
		if strings.TrimSpace(stdout) != local {
			t.Fatalf("path %s stdout = %q, want %q", ref, stdout, local)
		}
	}
}

func TestRunReportsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(nil, Options{Stdout: &stdout, Stderr: &stderr, Env: os.Getenv})
	if !IsUsageError(err) {
		t.Fatalf("err = %v, want usage error", err)
	}
	if code, ok := ExitCode(err); !ok || code != 2 {
		t.Fatalf("ExitCode(err) = %d/%t, want 2/true", code, ok)
	}
	if !strings.Contains(stderr.String(), "agents-attachments list") {
		t.Fatalf("stderr missing usage:\n%s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestPathReportsMissingMalformedAmbiguousAndLocalPathErrors(t *testing.T) {
	root := t.TempDir()
	missingManifest := filepath.Join(root, "missing.json")
	if err := runHelperError([]string{"path", "photo", "--manifest", missingManifest}); err == nil || !strings.Contains(err.Error(), "manifest not found") {
		t.Fatalf("missing manifest err = %v", err)
	}

	malformedManifest := filepath.Join(root, "malformed.json")
	mustWriteBytes(t, malformedManifest, []byte(`{"attachments":{}}`))
	if err := runHelperError([]string{"path", "photo", "--manifest", malformedManifest}); err == nil || !strings.Contains(err.Error(), "attachments field must be a list") {
		t.Fatalf("malformed manifest err = %v", err)
	}

	ambiguousManifest := filepath.Join(root, "ambiguous.json")
	writeManifestForTest(t, ambiguousManifest, []map[string]any{
		{"id": "one", "name": "same.png", "mime_type": "image/png", "local_path": filepath.Join(root, "one.png")},
		{"id": "two", "name": "same.png", "mime_type": "image/png", "local_path": filepath.Join(root, "two.png")},
	})
	if err := runHelperError([]string{"path", "same.png", "--manifest", ambiguousManifest}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous reference err = %v", err)
	}

	noLocalPathManifest := filepath.Join(root, "no-local-path.json")
	writeManifestForTest(t, noLocalPathManifest, []map[string]any{
		{"id": "photo", "name": "photo.png", "mime_type": "image/png"},
	})
	if err := runHelperError([]string{"path", "photo", "--manifest", noLocalPathManifest}); err == nil || !strings.Contains(err.Error(), "has no local_path") {
		t.Fatalf("no local_path err = %v", err)
	}
}

func TestStageImagesReportsInvalidInputs(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	writeManifestForTest(t, manifest, nil)
	if err := runHelperError([]string{"stage-images", "--manifest", manifest}); err == nil || !strings.Contains(err.Error(), "requires --all or at least one") {
		t.Fatalf("no input err = %v", err)
	}

	text := filepath.Join(root, "notes.txt")
	mustWriteBytes(t, text, []byte("not an image"))
	if err := runHelperError([]string{"stage-images", "--manifest", manifest, text}); err == nil || !strings.Contains(err.Error(), "not an image input") {
		t.Fatalf("non-image err = %v", err)
	}
}

func TestStageImagesHEICReportsMissingConverter(t *testing.T) {
	root := t.TempDir()
	heic := filepath.Join(root, "sample.heic")
	mustWriteBytes(t, heic, []byte("fake-heic"))
	t.Setenv("PATH", t.TempDir())
	t.Setenv("AGENTS_ATTACHMENTS_PLATFORM", "linux")

	err := runHelperError([]string{"stage-images", "--out-dir", filepath.Join(root, "stage"), heic})
	if err == nil || !strings.Contains(err.Error(), "HEIC normalization requires") {
		t.Fatalf("missing converter err = %v", err)
	}
}

func TestMaterializeReportsMalformedRolloutPayload(t *testing.T) {
	root := t.TempDir()
	session := filepath.Join(root, "rollout-thread.jsonl")
	line := map[string]any{
		"type": "response_item",
		"payload": map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{"type": "input_image", "image_url": "data:image/png;base64,%"},
			},
		},
	}
	lineBytes, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	mustWriteBytes(t, session, append(lineBytes, '\n'))

	err = runHelperError([]string{"materialize", "--session", session, "--manifest", filepath.Join(root, "manifest.json")})
	if err == nil || !strings.Contains(err.Error(), "illegal base64") {
		t.Fatalf("malformed rollout err = %v", err)
	}
}

func runHelper(t *testing.T, args []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if err := Run(args, Options{Stdout: &stdout, Stderr: &stderr, Env: os.Getenv}); err != nil {
		t.Fatalf("agents attachments failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}
	return stdout.String()
}

func runHelperError(args []string) error {
	var stdout, stderr bytes.Buffer
	return Run(args, Options{Stdout: &stdout, Stderr: &stderr, Env: os.Getenv})
}

func writeManifestForTest(t *testing.T, path string, attachments []map[string]any) {
	t.Helper()
	payload := map[string]any{"attachments": attachments}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	mustWriteBytes(t, path, append(data, '\n'))
}

func decodeObject(t *testing.T, data string) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal([]byte(data), &object); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}
	return object
}

func nestedString(value any, path ...string) string {
	current := value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	got, _ := current.(string)
	return got
}

func mustWriteBytes(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func mustReadBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return data
}

func mustSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s): %v", path, err)
	}
	return info.Size()
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
