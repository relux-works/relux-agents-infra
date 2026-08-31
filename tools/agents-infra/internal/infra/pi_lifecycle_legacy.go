//go:build !windows

package infra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var (
	// Deterministic production-entry seams used only by filesystem authority
	// tests. Production leaves both nil.
	piLifecycleBeforeLegacyCandidateUnlink func(parentFD int, name string, index int) error
	piLifecycleAfterLegacyCandidateRename  func(index int) error
	piLifecycleAfterLegacyCandidateUnlink  func(index int) error
)

type piLifecycleLegacyGeneration struct {
	SchemaVersion       int                           `json:"schema_version"`
	Generation          uint64                        `json:"generation"`
	State               string                        `json:"state"`
	Scope               string                        `json:"scope"`
	OperationID         string                        `json:"operation_id,omitempty"`
	StartedAt           string                        `json:"started_at,omitempty"`
	PlanHash            string                        `json:"plan_hash,omitempty"`
	PolicySource        string                        `json:"policy_source,omitempty"`
	PolicyDigest        string                        `json:"policy_digest,omitempty"`
	AggregateGeneration uint64                        `json:"aggregate_generation,omitempty"`
	ProfileDirectory    PiLegacyEvidenceIdentity      `json:"profile_directory,omitempty"`
	Candidates          []PiLegacyRetirementCandidate `json:"candidates,omitempty"`
	NextCandidate       int                           `json:"next_candidate,omitempty"`
	CandidateRenamed    bool                          `json:"candidate_renamed,omitempty"`
	Retired             int                           `json:"retired,omitempty"`
}

type piLifecycleOperatorProfile struct {
	Project      string
	ProfileName  string
	Profile      PiProfile
	PolicySource string
	Paths        PiStatePaths
}

func resolvePiLifecycleOperator(options PiLifecycleOperatorOptions) (piLifecycleOperatorProfile, error) {
	project, err := CanonicalProjectDir(options.ProjectDir)
	if err != nil {
		return piLifecycleOperatorProfile{}, err
	}
	homeDir := options.HomeDir
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return piLifecycleOperatorProfile{}, err
		}
	}
	composite, err := loadCompositeProjectConfig(ancestorDirsRootFirst(project), filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName))
	if err != nil {
		return piLifecycleOperatorProfile{}, piError("invalid_project_configuration", err)
	}
	profileName := options.Profile
	if profileName == "" && composite.PiPrimarySession.Profile.Present {
		profileName = composite.PiPrimarySession.Profile.Value
	}
	if profileName == "" {
		return piLifecycleOperatorProfile{}, piError("unknown_pi_profile", errors.New("no managed Pi profile is selected"))
	}
	profile, ok := composite.PiProfiles[profileName]
	if !ok {
		return piLifecycleOperatorProfile{}, piError("unknown_pi_profile", fmt.Errorf("unknown Pi profile %q", profileName))
	}
	if err := ValidatePiStateKeyCollisions(composite.PiProfiles); err != nil {
		return piLifecycleOperatorProfile{}, err
	}
	paths, err := ResolvePiStatePaths(options.CacheRoot, project, profileName)
	if err != nil {
		return piLifecycleOperatorProfile{}, err
	}
	return piLifecycleOperatorProfile{Project: project, ProfileName: profileName, Profile: profile, PolicySource: profile.Source, Paths: paths}, nil
}

func PiLifecycleOperatorStatus(ctx context.Context, options PiLifecycleOperatorOptions, continuation string) (PiLifecycleLogStatus, error) {
	resolved, err := resolvePiLifecycleOperator(options)
	if err != nil {
		return PiLifecycleLogStatus{}, err
	}
	return PiLifecycleStatus(ctx, resolved.Paths, resolved.Profile.LifecycleLogRetention, resolved.PolicySource, continuation)
}

func PiLegacyRetirementOperatorDryRun(ctx context.Context, options PiLifecycleOperatorOptions) (PiLegacyRetirementPlan, error) {
	resolved, err := resolvePiLifecycleOperator(options)
	if err != nil {
		return PiLegacyRetirementPlan{}, err
	}
	return PiLegacyRetirementDryRun(ctx, resolved.Paths, resolved.Profile.LifecycleLogRetention, resolved.PolicySource)
}

func PiLegacyRetirementOperatorConfirm(ctx context.Context, options PiLifecycleOperatorOptions, confirmation string) (PiLegacyRetirementResult, error) {
	resolved, err := resolvePiLifecycleOperator(options)
	if err != nil {
		return PiLegacyRetirementResult{}, err
	}
	return PiLegacyRetire(ctx, resolved.Paths, resolved.Profile.LifecycleLogRetention, resolved.PolicySource, confirmation)
}

func piLegacyIdentityFromStat(st unix.Stat_t) PiLegacyEvidenceIdentity {
	return PiLegacyEvidenceIdentity{
		Device: uint64(st.Dev), Inode: st.Ino, Mode: uint32(st.Mode), UID: st.Uid,
		Links: uint64(st.Nlink), Size: st.Size, ChangeNsec: piLifecycleStatChangeNsec(st),
	}
}

func piLegacyDirectoryIdentityMatches(expected PiLegacyEvidenceIdentity, observed unix.Stat_t, bindChange bool) bool {
	actual := piLegacyIdentityFromStat(observed)
	if expected.Device != actual.Device || expected.Inode != actual.Inode || expected.Mode != actual.Mode || expected.UID != actual.UID {
		return false
	}
	return !bindChange || expected.Links == actual.Links && expected.Size == actual.Size && expected.ChangeNsec == actual.ChangeNsec
}

func piLegacyFileIdentityMatches(expected PiLegacyEvidenceIdentity, observed unix.Stat_t, bindChange bool) bool {
	actual := piLegacyIdentityFromStat(observed)
	if expected.Device != actual.Device || expected.Inode != actual.Inode || expected.Mode != actual.Mode || expected.UID != actual.UID || expected.Links != actual.Links || expected.Size != actual.Size {
		return false
	}
	return !bindChange || expected.ChangeNsec == actual.ChangeNsec
}

func validatePiLegacyFileStat(st unix.Stat_t) error {
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0o777 != 0o600 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || st.Size < 0 {
		return errors.New("legacy candidate is not a mode-0600, effective-UID-owned, singly-linked regular file")
	}
	return nil
}

func readPiLifecycleLegacyGeneration(rootFD int, budget *piLifecycleBudget) (piLifecycleLegacyGeneration, error) {
	encoded, _, err := readPiLifecycleControl(rootFD, "legacy-generation.json", budget)
	if err != nil {
		return piLifecycleLegacyGeneration{}, err
	}
	var generation piLifecycleLegacyGeneration
	if err := decodePiLifecycleControl(encoded, &generation); err != nil {
		return generation, err
	}
	if err := validatePiLifecycleLegacyGeneration(generation); err != nil {
		return generation, err
	}
	return generation, nil
}

func validatePiLifecycleLegacyGeneration(generation piLifecycleLegacyGeneration) error {
	if generation.SchemaVersion != piLifecycleSchemaVersion || generation.Scope != "legacy" || generation.State != "even" && generation.State != "odd" || (generation.State == "even") != (generation.Generation&1 == 0) || generation.Retired < 0 {
		return errors.New("legacy generation schema, scope, state, parity, or counters are invalid")
	}
	if generation.State == "even" {
		zeroIdentity := PiLegacyEvidenceIdentity{}
		if generation.OperationID != "" || generation.StartedAt != "" || generation.PlanHash != "" || generation.PolicySource != "" || generation.PolicyDigest != "" || generation.AggregateGeneration != 0 || generation.ProfileDirectory != zeroIdentity || len(generation.Candidates) != 0 || generation.NextCandidate != 0 || generation.CandidateRenamed {
			return errors.New("even legacy generation carries retirement authority")
		}
		return nil
	}
	if len(generation.OperationID) != 32 || len(generation.PlanHash) != 64 || generation.PolicySource == "" || len(generation.PolicyDigest) != 64 || generation.AggregateGeneration&1 != 0 || len(generation.Candidates) == 0 || generation.NextCandidate < 0 || generation.NextCandidate > len(generation.Candidates) || generation.CandidateRenamed && generation.NextCandidate == len(generation.Candidates) {
		return errors.New("odd legacy generation retirement coordinates are invalid")
	}
	for _, encoded := range []string{generation.OperationID, generation.PlanHash, generation.PolicyDigest} {
		if _, err := hex.DecodeString(encoded); err != nil {
			return errors.New("odd legacy generation contains non-hex authority")
		}
	}
	started, err := time.Parse(time.RFC3339Nano, generation.StartedAt)
	if err != nil || started.UTC().Format(time.RFC3339Nano) != generation.StartedAt {
		return errors.New("odd legacy generation started_at is invalid")
	}
	if err := validatePiLegacyDirectoryIdentity(generation.ProfileDirectory); err != nil {
		return err
	}
	previous := ""
	for _, candidate := range generation.Candidates {
		if err := validatePiLegacyRetirementCandidate(candidate); err != nil {
			return err
		}
		if candidate.Path <= previous {
			return errors.New("legacy candidates are not strictly sorted")
		}
		previous = candidate.Path
	}
	return nil
}

func validatePiLegacyDirectoryIdentity(identity PiLegacyEvidenceIdentity) error {
	if identity.Device == 0 || identity.Inode == 0 || identity.Mode&unix.S_IFMT != unix.S_IFDIR || identity.Mode&0o777 != 0o700 || identity.UID != uint32(os.Geteuid()) || identity.Links == 0 {
		return errors.New("legacy directory identity is invalid")
	}
	return nil
}

func validatePiLegacyRetirementCandidate(candidate PiLegacyRetirementCandidate) error {
	if candidate.Name == "" || candidate.Name == "." || candidate.Name == ".." || strings.ContainsAny(candidate.Name, `/\\`) || !strings.HasSuffix(candidate.Name, ".jsonl") {
		return errors.New("legacy candidate name is invalid")
	}
	if candidate.Scope != "profile" && candidate.Scope != "run" {
		return errors.New("legacy candidate scope is invalid")
	}
	wantPath := "logs/" + candidate.Name
	if candidate.Scope == "profile" {
		if candidate.RunStateKey != "" || candidate.RunsDirectory != nil || candidate.RunDirectory != nil {
			return errors.New("profile legacy candidate carries run authority")
		}
	} else {
		if !piStateKeyPattern.MatchString(candidate.RunStateKey) || candidate.RunsDirectory == nil || candidate.RunDirectory == nil {
			return errors.New("run legacy candidate authority is incomplete")
		}
		wantPath = "runs/" + candidate.RunStateKey + "/logs/" + candidate.Name
		if err := validatePiLegacyDirectoryIdentity(*candidate.RunsDirectory); err != nil {
			return err
		}
		if err := validatePiLegacyDirectoryIdentity(*candidate.RunDirectory); err != nil {
			return err
		}
	}
	if candidate.Path != wantPath {
		return errors.New("legacy candidate path is not canonical")
	}
	if err := validatePiLegacyDirectoryIdentity(candidate.LogsDirectory); err != nil {
		return err
	}
	if candidate.File.Device == 0 || candidate.File.Inode == 0 || candidate.File.Mode&unix.S_IFMT != unix.S_IFREG || candidate.File.Mode&0o777 != 0o600 || candidate.File.UID != uint32(os.Geteuid()) || candidate.File.Links != 1 || candidate.File.Size < 0 {
		return errors.New("legacy candidate file identity is invalid")
	}
	return nil
}

func PiLegacyRetirementDryRun(ctx context.Context, paths PiStatePaths, policy PiLifecycleLogRetention, policySource string) (PiLegacyRetirementPlan, error) {
	return buildPiLegacyRetirementPlan(ctx, paths, policy, policySource)
}

func buildPiLegacyRetirementPlan(ctx context.Context, paths PiStatePaths, policy PiLifecycleLogRetention, policySource string) (PiLegacyRetirementPlan, error) {
	plan := PiLegacyRetirementPlan{
		Schema: PiLegacyRetirementPlanSchema, DryRun: true, PolicySource: policySource,
		PolicyDigest: piLifecyclePolicyDigest(policy), ProjectStateKey: paths.ProjectStateKey,
		ProfileStateKey: paths.ProfileStateKey, CandidateLimit: policy.MaxMutationsPerOperation / 2,
		Candidates: []PiLegacyRetirementCandidate{},
	}
	if plan.CandidateLimit < 1 {
		return plan, piError("lifecycle_legacy_plan_exhausted", errors.New("max_mutations_per_operation must permit one fenced rename and unlink"))
	}
	opCtx, cancel := piLifecycleOperationContext(ctx, policy.StatusTimeoutSeconds)
	defer cancel()
	profileFD, err := openPiAggregateProfileRoot(paths)
	if err != nil {
		return plan, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(profileFD)
	profileStat, err := validatePiLifecycleDirAt(profileFD, ".", profileFD)
	if err != nil {
		return plan, piError("lifecycle_log_evidence_unknown", err)
	}
	plan.ProfileDirectory = piLegacyIdentityFromStat(profileStat)
	rootFD, rootStat, entriesFD, entriesStat, err := openPiLifecycleStatusAuthority(profileFD)
	if err != nil {
		return plan, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(rootFD)
	defer unix.Close(entriesFD)
	budget := &piLifecycleBudget{policy: policy}
	aggregateStart, legacyStart, err := readPiLifecycleGenerationPair(rootFD, budget)
	if err != nil || aggregateStart.State != "even" || legacyStart.State != "even" {
		if err == nil {
			err = errors.New("legacy retirement requires stable even generations")
		}
		return plan, piError("lifecycle_log_evidence_unknown", err)
	}
	if legacyStart.Generation > ^uint64(0)-2 {
		return plan, piError("lifecycle_legacy_plan_exhausted", errors.New("legacy generation cannot fence and complete another retirement"))
	}
	plan.AggregateGeneration, plan.LegacyGeneration = aggregateStart.Generation, legacyStart.Generation
	candidates, err := collectPiLegacyRetirementCandidates(opCtx, profileFD, budget)
	if err != nil {
		return plan, err
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
	if len(candidates) > plan.CandidateLimit {
		return plan, piError("lifecycle_legacy_plan_exhausted", fmt.Errorf("%d candidates exceed bounded plan limit %d; inspect with paginated status", len(candidates), plan.CandidateLimit))
	}
	aggregateFinal, legacyFinal, err := readPiLifecycleGenerationPair(rootFD, budget)
	plan.ScanEntries, plan.ScanControlBytes = budget.entries, budget.controlBytes
	if err != nil || aggregateFinal != aggregateStart || !reflect.DeepEqual(legacyFinal, legacyStart) {
		if err == nil {
			err = errors.New("lifecycle generation changed during legacy dry-run")
		}
		return plan, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := revalidatePiLifecycleStatusAuthority(profileFD, rootFD, rootStat, entriesFD, entriesStat); err != nil {
		return plan, piError("lifecycle_log_evidence_unknown", err)
	}
	profileFinal, err := validatePiLifecycleDirAt(profileFD, ".", profileFD)
	if err != nil || !piLegacyDirectoryIdentityMatches(plan.ProfileDirectory, profileFinal, true) {
		return plan, piError("lifecycle_log_evidence_unknown", errors.New("profile directory changed during legacy dry-run"))
	}
	plan.Candidates, plan.ScanComplete = candidates, true
	hash, err := hashPiLegacyRetirementPlan(plan)
	if err != nil {
		return plan, err
	}
	plan.PlanHash = hash
	if legacyStart.Retired > int(^uint(0)>>1)-len(candidates) {
		return plan, piError("lifecycle_legacy_plan_exhausted", errors.New("legacy retired counter cannot represent the complete plan"))
	}
	prospective := piLifecycleLegacyGeneration{
		SchemaVersion: piLifecycleSchemaVersion, Generation: legacyStart.Generation + 1, State: "odd", Scope: "legacy",
		OperationID: strings.Repeat("0", 32), StartedAt: "9999-12-31T23:59:59.999999999Z",
		PlanHash: plan.PlanHash, PolicySource: plan.PolicySource, PolicyDigest: plan.PolicyDigest, AggregateGeneration: plan.AggregateGeneration,
		ProfileDirectory: plan.ProfileDirectory, Candidates: append([]PiLegacyRetirementCandidate(nil), candidates...), Retired: legacyStart.Retired,
	}
	persistedStates := []piLifecycleLegacyGeneration{prospective}
	if len(candidates) > 0 {
		renameProgress := prospective
		renameProgress.NextCandidate = len(candidates) - 1
		renameProgress.CandidateRenamed = true
		renameProgress.Retired += len(candidates) - 1
		unlinkedProgress := prospective
		unlinkedProgress.NextCandidate = len(candidates)
		unlinkedProgress.Retired += len(candidates)
		persistedStates = append(persistedStates, renameProgress, unlinkedProgress)
	}
	for _, persisted := range persistedStates {
		if _, err := encodePiLifecycleControl(persisted); err != nil {
			return plan, piError("lifecycle_legacy_plan_exhausted", fmt.Errorf("complete retirement authority does not fit the bounded control document: %w", err))
		}
	}
	return plan, nil
}

func hashPiLegacyRetirementPlan(plan PiLegacyRetirementPlan) (string, error) {
	projection := plan
	projection.DryRun = false
	projection.PlanHash = ""
	encoded, err := json.Marshal(projection)
	if err != nil {
		return "", piError("lifecycle_legacy_plan_refused", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func collectPiLegacyRetirementCandidates(ctx context.Context, profileFD int, budget *piLifecycleBudget) ([]PiLegacyRetirementCandidate, error) {
	var candidates []PiLegacyRetirementCandidate
	appendLogs := func(parentFD int, directoryName, scope, runKey string, runsIdentity, runIdentity *PiLegacyEvidenceIdentity) error {
		logsFD, _, err := openPiLifecycleDirectoryAt(parentFD, directoryName)
		if errors.Is(err, syscall.ENOENT) {
			return nil
		}
		if err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		defer unix.Close(logsFD)
		var logsStat unix.Stat_t
		if err := unix.Fstat(logsFD, &logsStat); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		logsIdentity := piLegacyIdentityFromStat(logsStat)
		_, complete, scanErr := scanPiLifecycleDirectoryPage(ctx, logsFD, 0, budget, func(entry piLifecycleDirent) error {
			var st unix.Stat_t
			if err := unix.Fstatat(logsFD, entry.Name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return piError("lifecycle_log_evidence_unknown", err)
			}
			if !strings.HasSuffix(entry.Name, ".jsonl") || validatePiLegacyFileStat(st) != nil {
				return piError("lifecycle_log_evidence_unknown", fmt.Errorf("unproven legacy evidence %q", entry.Name))
			}
			path := "logs/" + entry.Name
			if scope == "run" {
				path = "runs/" + runKey + "/logs/" + entry.Name
			}
			candidate := PiLegacyRetirementCandidate{Path: path, Scope: scope, RunStateKey: runKey, Name: entry.Name, File: piLegacyIdentityFromStat(st), LogsDirectory: logsIdentity}
			if runsIdentity != nil {
				copy := *runsIdentity
				candidate.RunsDirectory = &copy
			}
			if runIdentity != nil {
				copy := *runIdentity
				candidate.RunDirectory = &copy
			}
			candidates = append(candidates, candidate)
			return nil
		})
		if scanErr != nil {
			return scanErr
		}
		if !complete {
			return piError("lifecycle_log_scan_exhausted", errors.New("legacy dry-run scan is incomplete"))
		}
		if err := revalidatePiLifecycleDirectoryAt(parentFD, directoryName, logsFD, piLifecycleDirectoryIdentity{Device: logsIdentity.Device, Inode: logsIdentity.Inode, ChangeNsec: logsIdentity.ChangeNsec}); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		return nil
	}
	if err := appendLogs(profileFD, "logs", "profile", "", nil, nil); err != nil {
		return nil, err
	}
	runsFD, runsDirectory, err := openPiLifecycleDirectoryAt(profileFD, "runs")
	if errors.Is(err, syscall.ENOENT) {
		return candidates, nil
	}
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(runsFD)
	var runsStat unix.Stat_t
	if err := unix.Fstat(runsFD, &runsStat); err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	runsIdentity := piLegacyIdentityFromStat(runsStat)
	_, complete, scanErr := scanPiLifecycleDirectoryPage(ctx, runsFD, 0, budget, func(entry piLifecycleDirent) error {
		if !piStateKeyPattern.MatchString(entry.Name) {
			return piError("lifecycle_log_evidence_unknown", fmt.Errorf("unproven run evidence %q", entry.Name))
		}
		runFD, runDirectory, err := openPiLifecycleDirectoryAt(runsFD, entry.Name)
		if err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		defer unix.Close(runFD)
		var runStat unix.Stat_t
		if err := unix.Fstat(runFD, &runStat); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		runIdentity := piLegacyIdentityFromStat(runStat)
		if err := appendLogs(runFD, "logs", "run", entry.Name, &runsIdentity, &runIdentity); err != nil {
			return err
		}
		return revalidatePiLifecycleDirectoryAt(runsFD, entry.Name, runFD, runDirectory)
	})
	if scanErr != nil {
		return nil, scanErr
	}
	if !complete {
		return nil, piError("lifecycle_log_scan_exhausted", errors.New("legacy runs dry-run scan is incomplete"))
	}
	if err := revalidatePiLifecycleDirectoryAt(profileFD, "runs", runsFD, runsDirectory); err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	return candidates, nil
}

func PiLegacyRetire(ctx context.Context, paths PiStatePaths, policy PiLifecycleLogRetention, policySource, confirmation string) (PiLegacyRetirementResult, error) {
	result := PiLegacyRetirementResult{Schema: PiLegacyRetirementPlanSchema}
	if len(confirmation) != 64 {
		return result, piError("lifecycle_legacy_confirmation_mismatch", errors.New("retirement requires the exact 64-hex dry-run plan hash"))
	}
	if _, err := hex.DecodeString(confirmation); err != nil || confirmation != strings.ToLower(confirmation) {
		return result, piError("lifecycle_legacy_confirmation_mismatch", errors.New("retirement confirmation is not lowercase hexadecimal"))
	}
	opCtx, cancel := piLifecycleOperationContext(ctx, policy.MaintenanceTimeoutSeconds)
	defer cancel()
	profileFD, rootFD, foreground, retention, err := openAndLockPiLegacyRetirementAuthority(opCtx, paths)
	if err != nil {
		return result, err
	}
	defer unix.Close(profileFD)
	defer unix.Close(rootFD)
	defer foreground.Close()
	defer retention.Close()

	legacy, err := readPiLifecycleLegacyGeneration(rootFD, nil)
	if err != nil {
		return result, piError("lifecycle_log_evidence_unknown", err)
	}
	resumed := legacy.State == "odd"
	if resumed {
		if legacy.PlanHash != confirmation || legacy.PolicySource != policySource || legacy.PolicyDigest != piLifecyclePolicyDigest(policy) {
			return result, piError("lifecycle_legacy_confirmation_mismatch", errors.New("confirmation does not match the resumable odd legacy plan"))
		}
	} else {
		plan, planErr := buildPiLegacyRetirementPlan(opCtx, paths, policy, policySource)
		if planErr != nil {
			return result, planErr
		}
		if plan.PlanHash != confirmation {
			return result, piError("lifecycle_legacy_confirmation_mismatch", errors.New("confirmation does not match the current full legacy plan"))
		}
		if len(plan.Candidates) == 0 {
			status, statusErr := PiLifecycleStatus(opCtx, paths, policy, policySource, "")
			result.PlanHash, result.LegacyGeneration, result.Status = plan.PlanHash, plan.LegacyGeneration, status
			return result, statusErr
		}
		legacy, err = beginPiLegacyRetirement(rootFD, plan)
		if err != nil {
			return result, err
		}
	}
	plannedCount := len(legacy.Candidates)
	if err := resumePiLegacyRetirement(opCtx, profileFD, rootFD, policy, &legacy); err != nil {
		return result, err
	}
	status, statusErr := PiLifecycleStatus(opCtx, paths, policy, policySource, "")
	result.PlanHash = confirmation
	result.LegacyGeneration = legacy.Generation
	result.RetiredCount = plannedCount
	result.Resumed = resumed
	result.Status = status
	return result, statusErr
}

func openAndLockPiLegacyRetirementAuthority(ctx context.Context, paths PiStatePaths) (int, int, *os.File, *os.File, error) {
	profileFD, err := openPiAggregateProfileRoot(paths)
	if err != nil {
		return -1, -1, nil, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	rootFD, err := unix.Openat(profileFD, "lifecycle-logs", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		unix.Close(profileFD)
		return -1, -1, nil, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	if _, err := validatePiLifecycleDirAt(profileFD, "lifecycle-logs", rootFD); err != nil {
		unix.Close(rootFD)
		unix.Close(profileFD)
		return -1, -1, nil, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	foreground, err := acquirePiLifecycleLock(ctx, rootFD, "foreground.lock")
	if err != nil {
		unix.Close(rootFD)
		unix.Close(profileFD)
		return -1, -1, nil, nil, err
	}
	retention, err := acquirePiLifecycleLock(ctx, rootFD, "retention.lock")
	if err != nil {
		foreground.Close()
		unix.Close(rootFD)
		unix.Close(profileFD)
		return -1, -1, nil, nil, err
	}
	return profileFD, rootFD, foreground, retention, nil
}

func beginPiLegacyRetirement(rootFD int, plan PiLegacyRetirementPlan) (piLifecycleLegacyGeneration, error) {
	current, err := readPiLifecycleLegacyGeneration(rootFD, nil)
	if err != nil || current.State != "even" || current.Generation != plan.LegacyGeneration || current.Generation > ^uint64(0)-2 {
		if err == nil {
			err = errors.New("legacy generation changed before retirement authorization")
		}
		return current, piError("lifecycle_log_evidence_unknown", err)
	}
	var nonce [16]byte
	if _, err := io.ReadFull(piLifecycleRandom, nonce[:]); err != nil {
		return current, piError("lifecycle_legacy_plan_refused", err)
	}
	operation := piLifecycleLegacyGeneration{
		SchemaVersion: piLifecycleSchemaVersion, Generation: current.Generation + 1, State: "odd", Scope: "legacy",
		OperationID: hex.EncodeToString(nonce[:]), StartedAt: piLifecycleNow().UTC().Format(time.RFC3339Nano),
		PlanHash: plan.PlanHash, PolicySource: plan.PolicySource, PolicyDigest: plan.PolicyDigest, AggregateGeneration: plan.AggregateGeneration,
		ProfileDirectory: plan.ProfileDirectory, Candidates: append([]PiLegacyRetirementCandidate(nil), plan.Candidates...), Retired: current.Retired,
	}
	if err := validatePiLifecycleLegacyGeneration(operation); err != nil {
		return current, piError("lifecycle_legacy_plan_refused", err)
	}
	encoded, err := encodePiLifecycleControl(operation)
	if err != nil {
		return current, piError("lifecycle_legacy_plan_exhausted", err)
	}
	if err := writePiLifecycleControlAtomic(rootFD, "legacy-generation.json", encoded); err != nil {
		return current, err
	}
	return operation, nil
}

func resumePiLegacyRetirement(ctx context.Context, profileFD, rootFD int, policy PiLifecycleLogRetention, generation *piLifecycleLegacyGeneration) error {
	budget := &piLifecycleBudget{policy: policy}
	for generation.NextCandidate < len(generation.Candidates) {
		if err := piLifecycleCheck(ctx); err != nil {
			return err
		}
		candidate := generation.Candidates[generation.NextCandidate]
		// Open on stable device/inode/mode/UID authority first. The directory
		// ctime may already reflect our own crash-interrupted rename or unlink;
		// the exact pre-first-mutation ctime is rebound below when the original
		// source is still present.
		logsFD, err := openAndRevalidatePiLegacyCandidateParent(profileFD, generation.ProfileDirectory, candidate, true)
		if err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		tombstone := piLegacyRetirementTombstone(generation.OperationID, generation.NextCandidate)
		originalFD, originalPresent, originalErr := openPiLegacyCandidateFile(logsFD, candidate.Name, candidate.File, true)
		tombstoneFD, tombstonePresent, tombstoneErr := openPiLegacyCandidateFile(logsFD, tombstone, candidate.File, false)
		if originalErr != nil || tombstoneErr != nil || originalPresent && tombstonePresent {
			if originalFD >= 0 {
				unix.Close(originalFD)
			}
			if tombstoneFD >= 0 {
				unix.Close(tombstoneFD)
			}
			unix.Close(logsFD)
			return piError("lifecycle_log_evidence_unknown", errors.New("legacy candidate and tombstone state is ambiguous"))
		}
		if !originalPresent && !tombstonePresent {
			unix.Close(logsFD)
			if !generation.CandidateRenamed {
				return piError("lifecycle_log_evidence_unknown", errors.New("legacy candidate disappeared without persisted rename authority"))
			}
			generation.NextCandidate++
			generation.CandidateRenamed = false
			generation.Retired++
			if err := writePiLifecycleLegacyGeneration(rootFD, *generation); err != nil {
				return err
			}
			continue
		}
		if tombstonePresent && !generation.CandidateRenamed {
			// The exact operation-bound tombstone proves that this confirmed
			// retirement performed the rename. Persist equivalent authority
			// before unlink so a second crash cannot turn legitimate dual
			// absence into an unrecoverable unknown state.
			generation.CandidateRenamed = true
			if err := writePiLifecycleLegacyGeneration(rootFD, *generation); err != nil {
				unix.Close(tombstoneFD)
				unix.Close(logsFD)
				return err
			}
		}
		if originalPresent {
			if generation.CandidateRenamed {
				unix.Close(originalFD)
				unix.Close(logsFD)
				return piError("lifecycle_log_evidence_unknown", errors.New("legacy source reappeared after fenced rename"))
			}
			if err := budget.mutate(1); err != nil {
				unix.Close(originalFD)
				unix.Close(logsFD)
				return err
			}
			if piLifecycleBeforeLegacyCandidateUnlink != nil {
				if err := piLifecycleBeforeLegacyCandidateUnlink(logsFD, candidate.Name, generation.NextCandidate); err != nil {
					unix.Close(originalFD)
					unix.Close(logsFD)
					return piError("lifecycle_log_evidence_unknown", err)
				}
			}
			if err := revalidatePiLegacyMutationAuthority(profileFD, rootFD, logsFD, candidate.Name, originalFD, *generation, candidate, true); err != nil {
				unix.Close(originalFD)
				unix.Close(logsFD)
				return piError("lifecycle_log_evidence_unknown", err)
			}
			if err := unix.Renameat(logsFD, candidate.Name, logsFD, tombstone); err != nil {
				unix.Close(originalFD)
				unix.Close(logsFD)
				return piError("lifecycle_log_evidence_unknown", err)
			}
			unix.Close(originalFD)
			if piLifecycleAfterLegacyCandidateRename != nil {
				if err := piLifecycleAfterLegacyCandidateRename(generation.NextCandidate); err != nil {
					unix.Close(logsFD)
					return err
				}
			}
			generation.CandidateRenamed = true
			if err := writePiLifecycleLegacyGeneration(rootFD, *generation); err != nil {
				unix.Close(logsFD)
				return err
			}
			tombstoneFD, tombstonePresent, err = openPiLegacyCandidateFile(logsFD, tombstone, candidate.File, false)
			if err != nil || !tombstonePresent {
				unix.Close(logsFD)
				return piError("lifecycle_log_evidence_unknown", errors.New("fenced legacy tombstone is unreadable"))
			}
		}
		if err := budget.mutate(1); err != nil {
			unix.Close(tombstoneFD)
			unix.Close(logsFD)
			return err
		}
		if err := revalidatePiLegacyMutationAuthority(profileFD, rootFD, logsFD, tombstone, tombstoneFD, *generation, candidate, false); err != nil {
			unix.Close(tombstoneFD)
			unix.Close(logsFD)
			return piError("lifecycle_log_evidence_unknown", err)
		}
		if err := unix.Unlinkat(logsFD, tombstone, 0); err != nil {
			unix.Close(tombstoneFD)
			unix.Close(logsFD)
			return piError("lifecycle_log_evidence_unknown", err)
		}
		unix.Close(tombstoneFD)
		unix.Close(logsFD)
		if piLifecycleAfterLegacyCandidateUnlink != nil {
			if err := piLifecycleAfterLegacyCandidateUnlink(generation.NextCandidate); err != nil {
				return err
			}
		}
		generation.NextCandidate++
		generation.CandidateRenamed = false
		generation.Retired++
		if err := writePiLifecycleLegacyGeneration(rootFD, *generation); err != nil {
			return err
		}
	}
	completed := piLifecycleLegacyGeneration{SchemaVersion: piLifecycleSchemaVersion, Generation: generation.Generation + 1, State: "even", Scope: "legacy", Retired: generation.Retired}
	if err := writePiLifecycleLegacyGeneration(rootFD, completed); err != nil {
		return err
	}
	*generation = completed
	return nil
}

func piLegacyRetirementTombstone(operationID string, index int) string {
	return fmt.Sprintf(".retiring-%s-%d.jsonl", operationID, index)
}

func writePiLifecycleLegacyGeneration(rootFD int, generation piLifecycleLegacyGeneration) error {
	if err := validatePiLifecycleLegacyGeneration(generation); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	encoded, err := encodePiLifecycleControl(generation)
	if err != nil {
		return piError("lifecycle_legacy_plan_exhausted", err)
	}
	return writePiLifecycleControlAtomic(rootFD, "legacy-generation.json", encoded)
}

func openAndRevalidatePiLegacyCandidateParent(profileFD int, profileIdentity PiLegacyEvidenceIdentity, candidate PiLegacyRetirementCandidate, afterMutation bool) (int, error) {
	profileStat, err := validatePiLifecycleDirAt(profileFD, ".", profileFD)
	if err != nil || !piLegacyDirectoryIdentityMatches(profileIdentity, profileStat, !afterMutation) {
		return -1, errors.New("legacy profile directory authority changed")
	}
	parentFD := profileFD
	opened := []int{}
	closeOpened := func() {
		for _, fd := range opened {
			unix.Close(fd)
		}
	}
	if candidate.Scope == "run" {
		runsFD, _, err := openPiLifecycleDirectoryAt(profileFD, "runs")
		if err != nil {
			return -1, err
		}
		opened = append(opened, runsFD)
		var st unix.Stat_t
		if err := unix.Fstat(runsFD, &st); err != nil || !piLegacyDirectoryIdentityMatches(*candidate.RunsDirectory, st, !afterMutation) {
			closeOpened()
			return -1, errors.New("legacy runs directory authority changed")
		}
		runFD, _, err := openPiLifecycleDirectoryAt(runsFD, candidate.RunStateKey)
		if err != nil {
			closeOpened()
			return -1, err
		}
		opened = append(opened, runFD)
		if err := unix.Fstat(runFD, &st); err != nil || !piLegacyDirectoryIdentityMatches(*candidate.RunDirectory, st, !afterMutation) {
			closeOpened()
			return -1, errors.New("legacy run directory authority changed")
		}
		parentFD = runFD
	}
	logsFD, _, err := openPiLifecycleDirectoryAt(parentFD, "logs")
	if err != nil {
		closeOpened()
		return -1, err
	}
	var logsStat unix.Stat_t
	if err := unix.Fstat(logsFD, &logsStat); err != nil || !piLegacyDirectoryIdentityMatches(candidate.LogsDirectory, logsStat, !afterMutation) {
		unix.Close(logsFD)
		closeOpened()
		return -1, fmt.Errorf("legacy logs directory authority changed: after_mutation=%t expected=%+v observed=%+v err=%v", afterMutation, candidate.LogsDirectory, piLegacyIdentityFromStat(logsStat), err)
	}
	for _, fd := range opened {
		unix.Close(fd)
	}
	return logsFD, nil
}

func openPiLegacyCandidateFile(logsFD int, name string, expected PiLegacyEvidenceIdentity, bindChange bool) (int, bool, error) {
	fd, err := unix.Openat(logsFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, err
	}
	var descriptor, path unix.Stat_t
	if err := unix.Fstat(fd, &descriptor); err != nil {
		unix.Close(fd)
		return -1, false, err
	}
	if err := unix.Fstatat(logsFD, name, &path, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		unix.Close(fd)
		return -1, false, err
	}
	if validatePiLegacyFileStat(descriptor) != nil || !piLegacyFileIdentityMatches(expected, descriptor, bindChange) || descriptor.Dev != path.Dev || descriptor.Ino != path.Ino || descriptor.Mode != path.Mode || descriptor.Uid != path.Uid || descriptor.Nlink != path.Nlink || descriptor.Size != path.Size {
		unix.Close(fd)
		return -1, false, errors.New("legacy candidate identity changed")
	}
	return fd, true, nil
}

func revalidatePiLegacyMutationAuthority(profileFD, rootFD, logsFD int, name string, fileFD int, generation piLifecycleLegacyGeneration, candidate PiLegacyRetirementCandidate, bindChange bool) error {
	current, err := readPiLifecycleLegacyGeneration(rootFD, nil)
	if err != nil || !reflect.DeepEqual(current, generation) {
		return errors.New("legacy generation changed immediately before mutation")
	}
	aggregate, _, err := readPiLifecycleGenerationPair(rootFD, nil)
	if err != nil || aggregate.State != "even" || aggregate.Generation != generation.AggregateGeneration {
		return errors.New("aggregate generation changed immediately before legacy mutation")
	}
	bindDirectoryChange := bindChange && generation.NextCandidate == 0 && !generation.CandidateRenamed
	proofLogsFD, err := openAndRevalidatePiLegacyCandidateParent(profileFD, generation.ProfileDirectory, candidate, !bindDirectoryChange)
	if err != nil {
		return err
	}
	defer unix.Close(proofLogsFD)
	var heldLogsStat, proofLogsStat unix.Stat_t
	if err := unix.Fstat(logsFD, &heldLogsStat); err != nil {
		return err
	}
	if err := unix.Fstat(proofLogsFD, &proofLogsStat); err != nil || heldLogsStat.Dev != proofLogsStat.Dev || heldLogsStat.Ino != proofLogsStat.Ino {
		return errors.New("legacy directory path no longer names the held mutation authority")
	}
	profileStat, err := validatePiLifecycleDirAt(profileFD, ".", profileFD)
	if err != nil || !piLegacyDirectoryIdentityMatches(generation.ProfileDirectory, profileStat, false) {
		return errors.New("profile directory changed immediately before legacy mutation")
	}
	var logsStat unix.Stat_t
	if err := unix.Fstat(logsFD, &logsStat); err != nil || !piLegacyDirectoryIdentityMatches(candidate.LogsDirectory, logsStat, false) {
		return errors.New("legacy logs directory changed immediately before mutation")
	}
	var fileStat, pathStat unix.Stat_t
	if err := unix.Fstat(fileFD, &fileStat); err != nil || validatePiLegacyFileStat(fileStat) != nil || !piLegacyFileIdentityMatches(candidate.File, fileStat, bindChange) {
		return errors.New("legacy file descriptor changed immediately before mutation")
	}
	if err := unix.Fstatat(logsFD, name, &pathStat, unix.AT_SYMLINK_NOFOLLOW); err != nil || fileStat.Dev != pathStat.Dev || fileStat.Ino != pathStat.Ino || fileStat.Mode != pathStat.Mode || fileStat.Uid != pathStat.Uid || fileStat.Nlink != pathStat.Nlink || fileStat.Size != pathStat.Size {
		return errors.New("legacy file path changed immediately before mutation")
	}
	return nil
}
