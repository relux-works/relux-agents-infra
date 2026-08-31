package infra

// PiLifecycleLogStatus is bounded evidence. Paged results are lower bounds;
// only a fresh from-start scan with identical final even generations can
// publish WithinPolicy and SoakReady.
type PiLifecycleLogStatus struct {
	PolicySource          string                  `json:"policy_source"`
	AggregateRoot         string                  `json:"aggregate_root"`
	Policy                PiLifecycleLogRetention `json:"policy"`
	AggregateGeneration   uint64                  `json:"aggregate_generation"`
	LegacyGeneration      uint64                  `json:"legacy_generation"`
	ScanComplete          bool                    `json:"scan_complete"`
	ScanExhausted         bool                    `json:"scan_exhausted"`
	Continuation          string                  `json:"continuation,omitempty"`
	PageScope             string                  `json:"page_scope"`
	LowerBound            bool                    `json:"lower_bound"`
	ScanEntries           int                     `json:"scan_entries"`
	ScanControlBytes      int                     `json:"scan_control_bytes"`
	ManagedCount          int                     `json:"managed_count"`
	ManagedCommittedBytes int64                   `json:"managed_committed_bytes"`
	ManagedEnvelopeBytes  int64                   `json:"managed_envelope_bytes"`
	ActiveCount           int                     `json:"active_count"`
	ExpiredCount          int                     `json:"expired_count"`
	LegacyCount           int                     `json:"legacy_count"`
	LegacyBytes           int64                   `json:"legacy_bytes"`
	ForeignCount          int                     `json:"foreign_count"`
	ForeignBytes          int64                   `json:"foreign_bytes"`
	UnknownCount          int                     `json:"unknown_count"`
	Oldest                string                  `json:"oldest,omitempty"`
	Newest                string                  `json:"newest,omitempty"`
	RecoveryCount         int                     `json:"recovery_count"`
	PrunedCount           int                     `json:"pruned_count"`
	DroppedCount          int                     `json:"dropped_count"`
	Errors                []string                `json:"errors,omitempty"`
	WithinPolicy          bool                    `json:"within_policy"`
	SoakReady             bool                    `json:"soak_ready"`
}

const PiLegacyRetirementPlanSchema = "agents-infra.pi.lifecycle-legacy-retirement-plan.v1"

// PiLifecycleOperatorOptions resolves a configured managed profile without
// preparing project files, launching Pi, or contacting a shared runtime.
type PiLifecycleOperatorOptions struct {
	ProjectDir string
	Profile    string
	HomeDir    string
	CacheRoot  string
}

// PiLegacyEvidenceIdentity is the descriptor-relative filesystem authority
// projected by a legacy retirement dry-run. ChangeNsec and Size bind the plan;
// delete authority additionally requires the stable device/inode/mode/UID/link
// coordinates immediately before each mutation.
type PiLegacyEvidenceIdentity struct {
	Device     uint64 `json:"device"`
	Inode      uint64 `json:"inode"`
	Mode       uint32 `json:"mode"`
	UID        uint32 `json:"uid"`
	Links      uint64 `json:"links"`
	Size       int64  `json:"size"`
	ChangeNsec int64  `json:"change_nsec"`
}

type PiLegacyRetirementCandidate struct {
	Path          string                    `json:"path"`
	Scope         string                    `json:"scope"`
	RunStateKey   string                    `json:"run_state_key,omitempty"`
	Name          string                    `json:"name"`
	File          PiLegacyEvidenceIdentity  `json:"file"`
	LogsDirectory PiLegacyEvidenceIdentity  `json:"logs_directory"`
	RunsDirectory *PiLegacyEvidenceIdentity `json:"runs_directory,omitempty"`
	RunDirectory  *PiLegacyEvidenceIdentity `json:"run_directory,omitempty"`
}

// PiLegacyRetirementPlan is the exact, bounded external projection the
// operator must confirm. PlanHash covers every field that grants authority,
// including every candidate and directory identity.
type PiLegacyRetirementPlan struct {
	Schema              string                        `json:"schema"`
	DryRun              bool                          `json:"dry_run"`
	PolicySource        string                        `json:"policy_source"`
	PolicyDigest        string                        `json:"policy_digest"`
	ProjectStateKey     string                        `json:"project_state_key"`
	ProfileStateKey     string                        `json:"profile_state_key"`
	AggregateGeneration uint64                        `json:"aggregate_generation"`
	LegacyGeneration    uint64                        `json:"legacy_generation"`
	ProfileDirectory    PiLegacyEvidenceIdentity      `json:"profile_directory"`
	CandidateLimit      int                           `json:"candidate_limit"`
	ScanEntries         int                           `json:"scan_entries"`
	ScanControlBytes    int                           `json:"scan_control_bytes"`
	ScanComplete        bool                          `json:"scan_complete"`
	Candidates          []PiLegacyRetirementCandidate `json:"candidates"`
	PlanHash            string                        `json:"plan_hash"`
}

type PiLegacyRetirementResult struct {
	Schema           string               `json:"schema"`
	PlanHash         string               `json:"plan_hash"`
	LegacyGeneration uint64               `json:"legacy_generation"`
	RetiredCount     int                  `json:"retired_count"`
	Resumed          bool                 `json:"resumed"`
	Status           PiLifecycleLogStatus `json:"status"`
}
