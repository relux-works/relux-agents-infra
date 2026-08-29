//go:build !windows

package infra

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type sharedResolvedProfile struct {
	Project       string
	HomeDir       string
	ProfileName   string
	Profile       PiProfile
	Sharing       PiRuntimeSharing
	RuntimeKey    string
	ProfileDigest string
	Paths         SharedRuntimePaths
}

func resolveSharedProfile(projectDir, homeDir, cacheRoot, profileName string) (sharedResolvedProfile, error) {
	project, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return sharedResolvedProfile{}, err
	}
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return sharedResolvedProfile{}, err
		}
	}
	composite, err := loadCompositeProjectConfig(ancestorDirsRootFirst(project), filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName))
	if err != nil {
		return sharedResolvedProfile{}, piError("invalid_project_configuration", err)
	}
	if profileName == "" && composite.PiPrimarySession.Profile.Present {
		profileName = composite.PiPrimarySession.Profile.Value
	}
	if profileName == "" {
		return sharedResolvedProfile{}, sharedRuntimeError("shared_runtime_not_configured", errors.New("no managed Pi profile is selected"))
	}
	profile, ok := composite.PiProfiles[profileName]
	if !ok {
		return sharedResolvedProfile{}, piError("unknown_pi_profile", fmt.Errorf("unknown Pi profile %q", profileName))
	}
	if profile.Runtime.Sharing == nil || profile.Runtime.Sharing.Mode != "shared" {
		return sharedResolvedProfile{}, sharedRuntimeError("shared_runtime_not_configured", fmt.Errorf("Pi profile %q is not configured for shared runtime mode", profileName))
	}
	runtimeKey, profileDigest := SharedRuntimeKey(profile)
	paths, err := ResolveSharedRuntimePaths(cacheRoot, runtimeKey)
	if err != nil {
		return sharedResolvedProfile{}, err
	}
	return sharedResolvedProfile{
		Project:       project,
		HomeDir:       homeDir,
		ProfileName:   profileName,
		Profile:       profile,
		Sharing:       *profile.Runtime.Sharing,
		RuntimeKey:    runtimeKey,
		ProfileDigest: profileDigest,
		Paths:         paths,
	}, nil
}

func environmentValue(environ []string, name string) string {
	prefix := name + "="
	for _, item := range environ {
		if len(item) >= len(prefix) && item[:len(prefix)] == prefix {
			return item[len(prefix):]
		}
	}
	return ""
}

func scrubSharedRuntimeEnvironment(environ []string) []string {
	result := make([]string, 0, len(environ))
	for _, item := range environ {
		name := item
		if index := indexByte(item, '='); index >= 0 {
			name = item[:index]
		}
		if len(name) >= 3 && name[:3] == "PI_" {
			continue
		}
		if name == "HF_ENDPOINT" || name == "MODEL_ENDPOINT" || name == "GGML_BACKEND_PATH" || name == "LLAMA_API_KEY" {
			continue
		}
		if hasAnyPrefixFold(name, "DYLD_", "LD_", "NODE_", "BUN_", "LLAMA_ARG_") {
			continue
		}
		result = append(result, item)
	}
	return result
}

func indexByte(value string, target byte) int {
	for i := range value {
		if value[i] == target {
			return i
		}
	}
	return -1
}

func hasAnyPrefixFold(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(value) < len(prefix) {
			continue
		}
		match := true
		for i := range prefix {
			left := value[i]
			if left >= 'a' && left <= 'z' {
				left -= 'a' - 'A'
			}
			if left != prefix[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
