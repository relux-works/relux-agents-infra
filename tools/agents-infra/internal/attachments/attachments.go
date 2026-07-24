package attachments

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const (
	ManifestEnv             = "AGENTS_ATTACHMENTS_MANIFEST"
	DefaultManifest         = ".temp/agents-attachments-manifest.json"
	DefaultOutDirName       = "agents-attachments"
	DefaultImageStageDir    = ".temp/image-intake"
	DefaultImageMappingName = "image-stage-map.json"
)

type Options struct {
	Stdout io.Writer
	Stderr io.Writer
	Env    func(string) string
}

type Attachment map[string]any

type stageSource struct {
	Kind     string
	Input    string
	Path     string
	MIMEType string
	Manifest *stageManifestSource
}

type stageManifestSource struct {
	ID        string
	Name      string
	LocalPath string
}

var (
	dataURLRE   = regexp.MustCompile(`(?s)^data:([^;]+);base64,(.+)$`)
	imageNameRE = regexp.MustCompile(`(?i)<image\s+name=\[([^\]]+)\]>`)

	imageExtensions = map[string]bool{
		".bmp":  true,
		".gif":  true,
		".heic": true,
		".heif": true,
		".jpeg": true,
		".jpg":  true,
		".png":  true,
		".tif":  true,
		".tiff": true,
		".webp": true,
	}
	heicExtensions = map[string]bool{
		".heic": true,
		".heif": true,
	}

	sensitiveRedactions = []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`\b89\d{17,20}\b`), "[REDACTED_ICCID]"},
		{regexp.MustCompile(`\b\d{14,15}\b`), "[REDACTED_IMSI]"},
		{regexp.MustCompile(`(?i)\b(api[_-]?key|secret|token|password|passwd|pwd|qr)[=:][^\s/\\]+`), `${1}=[REDACTED_SECRET]`},
		{regexp.MustCompile(`\b[A-Fa-f0-9]{32,}\b`), "[REDACTED_HEX_SECRET]"},
		{regexp.MustCompile(`\b[A-Za-z0-9_-]{40,}\b`), "[REDACTED_KEYLIKE_SECRET]"},
	}
)

func Run(args []string, opts Options) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Env == nil {
		opts.Env = os.Getenv
	}
	manifestPath, strippedArgs, err := resolveManifestPath(args, opts.Env)
	if err != nil {
		return err
	}
	if len(strippedArgs) == 0 {
		_, _ = fmt.Fprint(opts.Stderr, Usage())
		return exitCodeError{code: 2}
	}

	command := strippedArgs[0]
	switch command {
	case "materialize":
		return cmdMaterialize(manifestPath, strippedArgs[1:], opts)
	case "stage-images":
		return cmdStageImages(manifestPath, strippedArgs[1:], opts)
	}

	attachments, err := loadAttachments(manifestPath)
	if err != nil {
		return err
	}
	switch command {
	case "list":
		return cmdList(attachments, opts.Stdout)
	case "show":
		if len(strippedArgs) != 2 {
			_, _ = fmt.Fprint(opts.Stderr, Usage())
			return exitCodeError{code: 2}
		}
		return cmdShow(attachments, strippedArgs[1], opts.Stdout)
	case "path":
		if len(strippedArgs) != 2 {
			_, _ = fmt.Fprint(opts.Stderr, Usage())
			return exitCodeError{code: 2}
		}
		return cmdPath(attachments, strippedArgs[1], opts.Stdout)
	default:
		_, _ = fmt.Fprint(opts.Stderr, Usage())
		return exitCodeError{code: 2}
	}
}

func Usage() string {
	return "Usage:\n" +
		"  agents-attachments list [--manifest PATH]\n" +
		"  agents-attachments show <id-or-name> [--manifest PATH]\n" +
		"  agents-attachments path <id-or-name> [--manifest PATH]\n" +
		"  agents-attachments materialize [--thread-id ID] [--session PATH] [--out-dir DIR] [--manifest PATH]\n" +
		"  agents-attachments stage-images [--manifest PATH] [--out-dir DIR] [--mapping PATH] [--all] <path-or-id-or-name>...\n"
}

type exitCodeError struct {
	code int
}

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func IsUsageError(err error) bool {
	var target exitCodeError
	return errors.As(err, &target) && target.code == 2
}

func ExitCode(err error) (int, bool) {
	var target exitCodeError
	if errors.As(err, &target) {
		return target.code, true
	}
	return 0, false
}

func resolveManifestPath(args []string, env func(string) string) (string, []string, error) {
	var manifest string
	stripped := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--manifest":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--manifest requires a path")
			}
			manifest = args[i+1]
			i++
		case strings.HasPrefix(arg, "--manifest="):
			manifest = strings.TrimPrefix(arg, "--manifest=")
		default:
			stripped = append(stripped, arg)
		}
	}
	if manifest == "" {
		manifest = env(ManifestEnv)
	}
	if manifest == "" {
		manifest = DefaultManifest
	}
	return expandUser(manifest), stripped, nil
}

func loadAttachments(path string) ([]Attachment, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("manifest not found: %s", path)
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}

	var items []any
	switch value := raw.(type) {
	case []any:
		items = value
	case map[string]any:
		rawAttachments, ok := value["attachments"]
		if !ok {
			items = nil
			break
		}
		var listOK bool
		items, listOK = rawAttachments.([]any)
		if !listOK {
			return nil, fmt.Errorf("attachments field must be a list in %s", path)
		}
	default:
		return nil, fmt.Errorf("unsupported manifest shape in %s", path)
	}

	result := make([]Attachment, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			result = append(result, Attachment(object))
		}
	}
	return result, nil
}

func attachmentKey(item Attachment, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func findAttachment(attachments []Attachment, needle string) (Attachment, error) {
	var matches []Attachment
	for _, item := range attachments {
		if attachmentKey(item, "id") == needle || attachmentKey(item, "name") == needle {
			matches = append(matches, item)
			continue
		}
		localPath := attachmentKey(item, "local_path")
		if localPath != "" && filepath.Base(localPath) == needle {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("attachment not found: %s", needle)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("attachment reference is ambiguous: %s", needle)
	}
	return matches[0], nil
}

func cmdList(attachments []Attachment, out io.Writer) error {
	for _, item := range attachments {
		_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\n",
			attachmentKey(item, "id"),
			attachmentKey(item, "name"),
			attachmentKey(item, "mime_type"),
			attachmentKey(item, "local_path"),
		)
	}
	return nil
}

func cmdShow(attachments []Attachment, needle string, out io.Writer) error {
	item, err := findAttachment(attachments, needle)
	if err != nil {
		return err
	}
	return writeJSON(out, item)
}

func cmdPath(attachments []Attachment, needle string, out io.Writer) error {
	item, err := findAttachment(attachments, needle)
	if err != nil {
		return err
	}
	localPath := attachmentKey(item, "local_path")
	if localPath == "" {
		return fmt.Errorf("attachment has no local_path: %s", needle)
	}
	_, _ = fmt.Fprintln(out, localPath)
	return nil
}

func cmdStageImages(manifestPath string, args []string, opts Options) error {
	outDir, mappingPath, allImages, refs, err := resolveStageOptions(args)
	if err != nil {
		return err
	}
	if mappingPath == "" {
		mappingPath = filepath.Join(outDir, DefaultImageMappingName)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mappingPath), 0o755); err != nil {
		return err
	}
	sources, err := resolveStageSources(manifestPath, allImages, refs)
	if err != nil {
		return err
	}
	seenNames := map[string]bool{}
	items := make([]map[string]any, 0, len(sources))
	for i, source := range sources {
		item, err := stageOneImage(source, outDir, i+1, seenNames)
		if err != nil {
			return err
		}
		items = append(items, item)
	}

	resolvedOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	resolvedMappingPath, err := filepath.Abs(mappingPath)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"version":      1,
		"out_dir":      resolvedOutDir,
		"mapping_path": resolvedMappingPath,
		"items":        items,
	}

	if err := writeJSONFile(mappingPath, payload); err != nil {
		return err
	}
	return writeJSON(opts.Stdout, payload)
}

func resolveStageOptions(args []string) (string, string, bool, []string, error) {
	outDir := DefaultImageStageDir
	var mappingPath string
	allImages := false
	var refs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--out-dir":
			if i+1 >= len(args) {
				return "", "", false, nil, fmt.Errorf("--out-dir requires a path")
			}
			outDir = expandUser(args[i+1])
			i++
		case strings.HasPrefix(arg, "--out-dir="):
			outDir = expandUser(strings.TrimPrefix(arg, "--out-dir="))
		case arg == "--mapping":
			if i+1 >= len(args) {
				return "", "", false, nil, fmt.Errorf("--mapping requires a path")
			}
			mappingPath = expandUser(args[i+1])
			i++
		case strings.HasPrefix(arg, "--mapping="):
			mappingPath = expandUser(strings.TrimPrefix(arg, "--mapping="))
		case arg == "--all":
			allImages = true
		case strings.HasPrefix(arg, "--"):
			return "", "", false, nil, fmt.Errorf("unknown option for stage-images: %s", arg)
		default:
			refs = append(refs, arg)
		}
	}
	if !allImages && len(refs) == 0 {
		return "", "", false, nil, fmt.Errorf("stage-images requires --all or at least one path/id/name")
	}
	return outDir, mappingPath, allImages, refs, nil
}

func resolveStageSources(manifestPath string, allImages bool, refs []string) ([]stageSource, error) {
	var attachments []Attachment
	var attachmentsLoaded bool
	sources := []stageSource{}
	if allImages {
		var err error
		attachments, err = loadAttachments(manifestPath)
		if err != nil {
			return nil, err
		}
		attachmentsLoaded = true
		for _, item := range attachments {
			localPath := attachmentKey(item, "local_path")
			if localPath == "" {
				continue
			}
			path := expandUser(localPath)
			mimeType := attachmentKey(item, "mime_type")
			if mimeType == "" {
				mimeType = guessMimeForPath(path)
			}
			if isImageCandidate(path, mimeType) {
				ref := attachmentKey(item, "id")
				if ref == "" {
					ref = attachmentKey(item, "name")
				}
				sources = append(sources, attachmentToStageSource(item, ref))
			}
		}
	}
	for _, ref := range refs {
		path := expandUser(ref)
		if _, err := os.Stat(path); err == nil {
			sources = append(sources, stageSource{
				Kind:     "path",
				Input:    ref,
				Path:     path,
				MIMEType: guessMimeForPath(path),
			})
			continue
		}
		if !attachmentsLoaded {
			var err error
			attachments, err = loadAttachments(manifestPath)
			if err != nil {
				return nil, err
			}
			attachmentsLoaded = true
		}
		item, err := findAttachment(attachments, ref)
		if err != nil {
			return nil, err
		}
		sources = append(sources, attachmentToStageSource(item, ref))
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no image inputs found")
	}
	return sources, nil
}

func attachmentToStageSource(item Attachment, ref string) stageSource {
	localPath := attachmentKey(item, "local_path")
	return stageSource{
		Kind:     "manifest",
		Input:    ref,
		Path:     expandUser(localPath),
		MIMEType: attachmentKey(item, "mime_type"),
		Manifest: &stageManifestSource{
			ID:        attachmentKey(item, "id"),
			Name:      attachmentKey(item, "name"),
			LocalPath: localPath,
		},
	}
}

func stageOneImage(source stageSource, outDir string, index int, seenNames map[string]bool) (map[string]any, error) {
	if source.Path == "" {
		return nil, fmt.Errorf("attachment has no local_path: %s", redactSensitiveText(source.Input))
	}
	info, err := os.Stat(source.Path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("image source not found: %s", redactSensitiveText(source.Path))
	}
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("image source is not a file: %s", redactSensitiveText(source.Path))
	}

	src, err := resolveExistingPath(source.Path)
	if err != nil {
		return nil, err
	}
	mimeType := source.MIMEType
	if mimeType == "" {
		mimeType = guessMimeForPath(src)
	}
	if !isImageCandidate(src, mimeType) {
		label := source.Input
		if label == "" {
			label = src
		}
		return nil, fmt.Errorf("not an image input: %s", redactSensitiveText(label))
	}

	normalizeHEIC := isHEICImage(src, mimeType)
	stageName := stagedNameForSource(src, mimeType, index, normalizeHEIC)
	stageName = uniquifyStageName(stageName, outDir, seenNames)
	stagedPath := filepath.Join(outDir, stageName)

	var normalization map[string]any
	action := "copied"
	outputMIME := mimeType
	if normalizeHEIC {
		normalization, err = runHEICNormalization(src, stagedPath)
		if err != nil {
			return nil, err
		}
		action = "normalized"
		outputMIME = "image/png"
	} else {
		if err := copyRegularFile(src, stagedPath); err != nil {
			return nil, err
		}
		if outputMIME == "" {
			outputMIME = guessMimeForPath(stagedPath)
		}
	}

	sourceInfo := map[string]any{
		"kind":        source.Kind,
		"input":       redactSensitiveText(source.Input),
		"label":       redactSensitiveText(filepath.Base(src)),
		"path":        redactSensitiveText(src),
		"path_sha256": sha256String(src),
	}
	if source.Manifest != nil {
		sourceInfo["manifest"] = map[string]any{
			"id":         redactSensitiveText(source.Manifest.ID),
			"name":       redactSensitiveText(source.Manifest.Name),
			"local_path": redactSensitiveText(source.Manifest.LocalPath),
		}
	}

	stagedAbs, err := filepath.Abs(stagedPath)
	if err != nil {
		return nil, err
	}
	item := map[string]any{
		"source":           sourceInfo,
		"source_read_only": true,
		"original": map[string]any{
			"sha256":     sha256File(src),
			"size_bytes": info.Size(),
			"mime_type":  mimeType,
		},
		"staged": map[string]any{
			"path":       stagedAbs,
			"filename":   filepath.Base(stagedPath),
			"sha256":     sha256File(stagedPath),
			"size_bytes": mustFileSize(stagedPath),
			"mime_type":  outputMIME,
		},
		"action": action,
	}
	if normalization != nil {
		item["normalization"] = normalization
	}
	return item, nil
}

func chooseHEICConverter(which func(string) (string, error), platform string) (string, string, error) {
	if platform == "" {
		platform = os.Getenv("AGENTS_ATTACHMENTS_PLATFORM")
	}
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform == "darwin" {
		if path, err := which("sips"); err == nil && path != "" {
			return "sips", path, nil
		}
	}
	if path, err := which("magick"); err == nil && path != "" {
		return "imagemagick", path, nil
	}
	if path, err := which("convert"); err == nil && path != "" {
		return "imagemagick-convert", path, nil
	}
	return "", "", fmt.Errorf("HEIC normalization requires macOS sips or ImageMagick (magick/convert) on PATH")
}

func runHEICNormalization(src, dst string) (map[string]any, error) {
	converter, executable, err := chooseHEICConverter(exec.LookPath, "")
	if err != nil {
		return nil, err
	}
	var argv []string
	if converter == "sips" {
		argv = []string{"-s", "format", "png", src, "--out", dst}
	} else {
		argv = []string{src, dst}
	}
	cmd := exec.Command(executable, argv...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return nil, fmt.Errorf("HEIC normalization failed for %s using %s: %s", redactSensitiveText(filepath.Base(src)), converter, redactSensitiveText(detail))
	}
	return map[string]any{"from": "heic", "to": "png", "converter": converter}, nil
}

func stagedNameForSource(src, mimeType string, index int, normalizeHEIC bool) string {
	rawName := filepath.Base(src)
	outputMIME := mimeType
	if normalizeHEIC {
		rawName = strings.TrimSuffix(rawName, filepath.Ext(rawName)) + ".png"
		outputMIME = "image/png"
	}
	safeLabel := redactSensitiveText(rawName)
	safeName := sanitizeName(safeLabel, fmt.Sprintf("image-%03d", index), outputMIME)
	return fmt.Sprintf("%03d-%s", index, safeName)
}

func uniquifyStageName(name, outDir string, seenNames map[string]bool) string {
	candidate := name
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if stem == "" {
		stem = "image"
	}
	suffix := filepath.Ext(name)
	index := 2
	for seenNames[candidate] || pathExists(filepath.Join(outDir, candidate)) {
		candidate = fmt.Sprintf("%s-%d%s", stem, index, suffix)
		index++
	}
	seenNames[candidate] = true
	return candidate
}

func cmdMaterialize(manifestPath string, args []string, opts Options) error {
	threadID, sessionPath, outDir, err := resolveMaterializeOptions(args, manifestPath, opts.Env)
	if err != nil {
		return err
	}
	if sessionPath == "" {
		if threadID == "" {
			return fmt.Errorf("materialize requires --thread-id, --session, or CODEX_THREAD_ID")
		}
		sessionPath, err = findRolloutPath(threadID)
		if err != nil {
			return err
		}
	}
	attachments, err := materializeFromCodexRollout(sessionPath, outDir)
	if err != nil {
		return err
	}
	if len(attachments) == 0 {
		return fmt.Errorf("no input_image attachments found in %s", sessionPath)
	}
	if err := writeManifest(manifestPath, attachments, sessionPath); err != nil {
		return err
	}
	resolvedManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(opts.Stdout, resolvedManifest)
	return nil
}

func resolveMaterializeOptions(args []string, manifestPath string, env func(string) string) (string, string, string, error) {
	threadID := env("CODEX_THREAD_ID")
	sessionPath := ""
	outDir := filepath.Join(filepath.Dir(manifestPath), DefaultOutDirName)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--thread-id":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--thread-id requires a value")
			}
			threadID = args[i+1]
			i++
		case strings.HasPrefix(arg, "--thread-id="):
			threadID = strings.TrimPrefix(arg, "--thread-id=")
		case arg == "--session":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--session requires a path")
			}
			sessionPath = expandUser(args[i+1])
			i++
		case strings.HasPrefix(arg, "--session="):
			sessionPath = expandUser(strings.TrimPrefix(arg, "--session="))
		case arg == "--out-dir":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--out-dir requires a path")
			}
			outDir = expandUser(args[i+1])
			i++
		case strings.HasPrefix(arg, "--out-dir="):
			outDir = expandUser(strings.TrimPrefix(arg, "--out-dir="))
		default:
			return "", "", "", fmt.Errorf("unknown option for materialize: %s", arg)
		}
	}
	return threadID, sessionPath, outDir, nil
}

func findRolloutPath(threadID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sessionsRoot := filepath.Join(home, ".codex", "sessions")
	if !pathExists(sessionsRoot) {
		return "", fmt.Errorf("codex sessions directory not found: %s", sessionsRoot)
	}
	type match struct {
		path    string
		modTime int64
	}
	var matches []match
	err = filepath.WalkDir(sessionsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "rollout-") && strings.HasSuffix(base, threadID+".jsonl") {
			info, err := d.Info()
			if err != nil {
				return err
			}
			matches = append(matches, match{path: path, modTime: info.ModTime().UnixNano()})
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})
	if len(matches) == 0 {
		return "", fmt.Errorf("no rollout file found for thread_id=%s", threadID)
	}
	return matches[0].path, nil
}

func materializeFromCodexRollout(sessionPath, outDir string) ([]Attachment, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	contentLists, err := rolloutUserMessageContents(sessionPath)
	if err != nil {
		return nil, err
	}
	attachments := []Attachment{}
	seenLocalNames := map[string]bool{}
	seenDisplayNames := map[string]bool{}
	counter := 1
	for _, content := range contentLists {
		var hintName string
		for _, rawItem := range content {
			item, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}
			itemType, _ := item["type"].(string)
			if itemType == "input_text" {
				if text, ok := item["text"].(string); ok {
					if hinted := extractHintName(text); hinted != "" {
						hintName = hinted
					}
				}
				continue
			}
			if itemType != "input_image" {
				continue
			}
			imageURL, ok := item["image_url"].(string)
			if !ok || !strings.HasPrefix(imageURL, "data:") {
				continue
			}
			mimeType, payload, err := splitDataURL(imageURL)
			if err != nil {
				return nil, err
			}
			sum := sha256.Sum256(payload)
			fallbackStem := fmt.Sprintf("attachment-%03d", counter)
			displayName := sanitizeName(hintName, fallbackStem, mimeType)
			displayName = uniquifyName(displayName, seenDisplayNames)
			localName := fmt.Sprintf("%03d-%s", counter, displayName)
			for seenLocalNames[localName] {
				counter++
				fallbackStem = fmt.Sprintf("attachment-%03d", counter)
				displayName = sanitizeName(hintName, fallbackStem, mimeType)
				displayName = uniquifyName(displayName, seenDisplayNames)
				localName = fmt.Sprintf("%03d-%s", counter, displayName)
			}
			seenLocalNames[localName] = true
			localPath := filepath.Join(outDir, localName)
			if err := os.WriteFile(localPath, payload, 0o644); err != nil {
				return nil, err
			}
			resolvedLocalPath, err := filepath.Abs(localPath)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, Attachment{
				"id":         fallbackStem,
				"name":       displayName,
				"mime_type":  mimeType,
				"size_bytes": len(payload),
				"sha256":     hex.EncodeToString(sum[:]),
				"local_path": resolvedLocalPath,
				"source":     "codex-rollout",
				"metadata": map[string]any{
					"session_path": sessionPath,
				},
			})
			counter++
			hintName = ""
		}
	}
	return attachments, nil
}

func rolloutUserMessageContents(sessionPath string) ([][]any, error) {
	file, err := os.Open(sessionPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var contents [][]any
	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	for {
		var object map[string]any
		if err := decoder.Decode(&object); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		if object["type"] != "response_item" {
			continue
		}
		payload, ok := object["payload"].(map[string]any)
		if !ok || payload["type"] != "message" || payload["role"] != "user" {
			continue
		}
		content, ok := payload["content"].([]any)
		if ok {
			contents = append(contents, content)
		}
	}
	return contents, nil
}

func extractHintName(text string) string {
	match := imageNameRE.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func splitDataURL(value string) (string, []byte, error) {
	match := dataURLRE.FindStringSubmatch(value)
	if len(match) < 3 {
		return "", nil, fmt.Errorf("unsupported attachment payload: expected data URL")
	}
	mimeType := strings.TrimSpace(match[1])
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	encoded := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, match[2])
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, err
	}
	return mimeType, payload, nil
}

func writeManifest(path string, attachments []Attachment, sessionPath string) error {
	payload := map[string]any{
		"attachments":  attachments,
		"source":       "codex-rollout",
		"session_path": sessionPath,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeJSONFile(path, payload)
}

func guessExtension(mimeType string) string {
	if mimeType == "image/jpeg" {
		return ".jpg"
	}
	extensions, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(extensions) > 0 {
		for _, ext := range extensions {
			if ext == ".jpe" {
				return ".jpg"
			}
			return ext
		}
	}
	return ""
}

func sanitizeName(name, fallbackStem, mimeType string) string {
	ext := guessExtension(mimeType)
	raw := strings.TrimSpace(name)
	if raw == "" {
		raw = fallbackStem
	}
	if filepath.Ext(raw) == "" && ext != "" {
		raw += ext
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range raw {
		valid := r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
		if valid {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	safe := strings.Trim(builder.String(), ".-")
	if safe == "" {
		safe = fallbackStem
		if ext != "" {
			safe += ext
		}
	}
	if filepath.Ext(safe) == "" && ext != "" {
		safe += ext
	}
	return safe
}

func uniquifyName(name string, seenNames map[string]bool) string {
	candidate := name
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if stem == "" {
		stem = "attachment"
	}
	suffix := filepath.Ext(name)
	index := 2
	for seenNames[candidate] {
		candidate = fmt.Sprintf("%s-%d%s", stem, index, suffix)
		index++
	}
	seenNames[candidate] = true
	return candidate
}

func redactSensitiveText(value string) string {
	redacted := value
	for _, rule := range sensitiveRedactions {
		redacted = rule.pattern.ReplaceAllString(redacted, rule.replacement)
	}
	return redacted
}

func guessMimeForPath(path string) string {
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if before, _, ok := strings.Cut(mimeType, ";"); ok {
		mimeType = before
	}
	return mimeType
}

func isImageCandidate(path, mimeType string) bool {
	return strings.HasPrefix(strings.ToLower(mimeType), "image/") || imageExtensions[strings.ToLower(filepath.Ext(path))]
}

func isHEICImage(path, mimeType string) bool {
	normalized := strings.ToLower(mimeType)
	return normalized == "image/heic" || normalized == "image/heif" || heicExtensions[strings.ToLower(filepath.Ext(path))]
}

func sha256File(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	digest := sha256.New()
	_, _ = io.Copy(digest, file)
	return hex.EncodeToString(digest.Sum(nil))
}

func sha256String(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func mustFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func resolveExistingPath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Abs(resolved)
	}
	return filepath.Abs(path)
}

func copyRegularFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func expandUser(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func writeJSON(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func writeJSONFile(path string, payload any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSON(file, payload)
}
