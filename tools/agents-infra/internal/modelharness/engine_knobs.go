package modelharness

import (
	"errors"
	"fmt"
	"strconv"
)

// Engine names the inference-engine axis of a local-model profile,
// independently of the executable that hosts it (the environment axis) and of
// the model artifact it serves. It is never inferred from the executable
// path: two profiles that launch the identical executable with different
// engine values translate their knobs differently, and a profile that omits
// engine defaults to EngineMLXLM regardless of what executable it names.
type Engine string

const (
	EngineMLXLM    Engine = "mlx-lm"
	EngineLlamaCPP Engine = "llama-cpp"
	EngineMLXSwift Engine = "mlx-swift"

	defaultEngine = EngineMLXLM

	unboundedKVContextTokens = "unbounded"
)

var knownEngines = map[Engine]bool{
	EngineMLXLM:    true,
	EngineLlamaCPP: true,
	EngineMLXSwift: true,
}

// Knobs carries the canonical, engine-independent knob set from
// .research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md
// that is expressible as launch argv (Knobs 1-4 of that document). Knobs 5-10
// describe wire-protocol field naming, cache/memory telemetry, health
// semantics, and argv-parsing precedence: none of those has a launch-time
// argv spelling to translate, so none is represented here. They remain the
// concern of the engine adapter itself (TASK-260830-1e9gse), not of profile
// resolution.
type Knobs struct {
	KVContextTokens     string `toml:"kv_context_tokens,omitempty" json:"kv_context_tokens,omitempty"`
	PrefillChunkTokens  string `toml:"prefill_chunk_tokens,omitempty" json:"prefill_chunk_tokens,omitempty"`
	ReasoningEffort     string `toml:"reasoning_effort,omitempty" json:"reasoning_effort,omitempty"`
	SpeculativeDecoding string `toml:"speculative_decoding,omitempty" json:"speculative_decoding,omitempty"`
}

// knobFormatter renders one already-named knob's value into argv tokens for
// one engine, or refuses with an error describing why that engine cannot
// express the value.
type knobFormatter func(value string) ([]string, error)

// knobEngineTable is the one table every canonical knob is translated
// through. A knob absent from an engine's row is not valid for that engine;
// resolving a profile that sets it refuses naming both the knob and the
// engine rather than silently dropping it. Adding a new engine means adding
// its column to the rows it can express here — no caller of translateKnobs
// changes.
var knobEngineTable = map[string]map[Engine]knobFormatter{
	"kv_context_tokens": {
		EngineMLXLM:    formatKVContextTokensAsMaxKVSize,
		EngineMLXSwift: formatKVContextTokensAsMaxKVSize,
		EngineLlamaCPP: formatKVContextTokensAsCtxSize,
	},
	"prefill_chunk_tokens": {
		EngineMLXLM:    positiveIntFlagFormatter("--prefill-step-size"),
		EngineMLXSwift: positiveIntFlagFormatter("--prefill-step-size"),
		EngineLlamaCPP: positiveIntFlagFormatter("--ubatch-size"),
	},
	"reasoning_effort": {
		EngineMLXLM:    formatReasoningEffortAsChatTemplateArgs,
		EngineMLXSwift: reasoningEffortFlagFormatter,
		EngineLlamaCPP: reasoningEffortFlagFormatter,
	},
	// mlx-lm and mlx-swift expose no speculative-decoding launch flag at all
	// (spec Knob 4: "no speculation flag exercised", "not characterised by
	// this study; declared single-generation-at-a-time"). A profile that
	// sets speculative_decoding under either engine refuses: the knob is
	// valid only for llama-cpp.
	"speculative_decoding": {
		EngineLlamaCPP: formatSpeculativeDecoding,
	},
}

var validReasoningEfforts = map[string]bool{"low": true, "medium": true, "xhigh": true}

// speculativeDecodingKinds maps the profile-facing enum to llama-server's
// --spec-type spelling. "draft-model" speculation is deliberately absent: it
// needs a second, paired draft-model artifact this profile schema has no axis
// for yet, and forcing it through --spec-type would silently drop that
// requirement rather than refuse it.
var speculativeDecodingKinds = map[string]string{
	"ngram": "ngram-mod",
	"mtp":   "draft-mtp",
}

// translateKnobs renders every set field on knobs into argv tokens for
// engine, in a fixed order, or refuses on the first knob the engine cannot
// express. knobs may be nil, in which case it returns no tokens and no error.
func translateKnobs(engine Engine, knobs *Knobs) ([]string, error) {
	if knobs == nil {
		return nil, nil
	}
	ordered := []struct {
		name  string
		value string
	}{
		{"kv_context_tokens", knobs.KVContextTokens},
		{"prefill_chunk_tokens", knobs.PrefillChunkTokens},
		{"reasoning_effort", knobs.ReasoningEffort},
		{"speculative_decoding", knobs.SpeculativeDecoding},
	}
	var argv []string
	for _, knob := range ordered {
		if knob.value == "" {
			continue
		}
		tokens, err := translateKnob(engine, knob.name, knob.value)
		if err != nil {
			return nil, err
		}
		argv = append(argv, tokens...)
	}
	return argv, nil
}

func translateKnob(engine Engine, name, value string) ([]string, error) {
	perEngine, known := knobEngineTable[name]
	if !known {
		return nil, fmt.Errorf("unknown knob %q", name)
	}
	formatter, ok := perEngine[engine]
	if !ok {
		return nil, fmt.Errorf("knob %q is not valid for engine %q", name, engine)
	}
	tokens, err := formatter(value)
	if err != nil {
		return nil, fmt.Errorf("knob %q for engine %q: %w", name, engine, err)
	}
	return tokens, nil
}

func formatKVContextTokensAsMaxKVSize(value string) ([]string, error) {
	if value == unboundedKVContextTokens {
		return nil, nil
	}
	tokens, err := positiveInt(value)
	if err != nil {
		return nil, fmt.Errorf("must be a positive integer or %q: %w", unboundedKVContextTokens, err)
	}
	return []string{"--max-kv-size", strconv.Itoa(tokens)}, nil
}

func formatKVContextTokensAsCtxSize(value string) ([]string, error) {
	if value == unboundedKVContextTokens {
		return nil, errors.New(`"unbounded" has no finite --ctx-size equivalent`)
	}
	tokens, err := positiveInt(value)
	if err != nil {
		return nil, fmt.Errorf("must be a positive integer or %q: %w", unboundedKVContextTokens, err)
	}
	return []string{"--ctx-size", strconv.Itoa(tokens)}, nil
}

func positiveIntFlagFormatter(flag string) knobFormatter {
	return func(value string) ([]string, error) {
		tokens, err := positiveInt(value)
		if err != nil {
			return nil, fmt.Errorf("must be a positive integer: %w", err)
		}
		return []string{flag, strconv.Itoa(tokens)}, nil
	}
}

func formatReasoningEffortAsChatTemplateArgs(value string) ([]string, error) {
	if !validReasoningEfforts[value] {
		return nil, fmt.Errorf("must be one of low, medium, xhigh, got %q", value)
	}
	return []string{"--chat-template-args", fmt.Sprintf(`{"reasoning_effort": %q}`, value)}, nil
}

func reasoningEffortFlagFormatter(value string) ([]string, error) {
	if !validReasoningEfforts[value] {
		return nil, fmt.Errorf("must be one of low, medium, xhigh, got %q", value)
	}
	return []string{"--reasoning-effort", value}, nil
}

func formatSpeculativeDecoding(value string) ([]string, error) {
	if value == "off" {
		return nil, nil
	}
	kind, ok := speculativeDecodingKinds[value]
	if !ok {
		return nil, fmt.Errorf("must be one of off, ngram, mtp, got %q", value)
	}
	return []string{"--spec-type", kind}, nil
}

func positiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < 1 {
		return 0, fmt.Errorf("must be positive, got %d", parsed)
	}
	return parsed, nil
}
