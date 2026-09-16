package infra

import "strings"

// Deprecated provider-launch messages. This is the single source of truth:
// the Go dispatcher (main.deprecatedProviderError) and every generated shell
// wrapper bake in these exact bytes, so the installed entrypoints cannot drift
// from the compiled binary.
const (
	deprecatedCodexMessage          = "agents-infra codex is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."
	deprecatedClaudeMessage         = "agents-infra claude is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."
	deprecatedOpenAIInfraMessage    = "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."
	deprecatedAnthropicInfraMessage = "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."
	deprecatedOpenAIDangeMessage    = "openai-dange is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."
	deprecatedAnthropicDangeMessage = "anthropic-dange is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."
)

// DeprecatedProviderMessage reports the exact one-line stderr message for a
// retired provider-launch entrypoint. The second result is false for live
// entrypoints such as qwen-infra.
func DeprecatedProviderMessage(entrypoint string) (string, bool) {
	switch entrypoint {
	case "codex":
		return deprecatedCodexMessage, true
	case "claude":
		return deprecatedClaudeMessage, true
	case "openai-infra":
		return deprecatedOpenAIInfraMessage, true
	case "anthropic-infra":
		return deprecatedAnthropicInfraMessage, true
	case "openai-dange":
		return deprecatedOpenAIDangeMessage, true
	case "anthropic-dange":
		return deprecatedAnthropicDangeMessage, true
	default:
		return "", false
	}
}

// escapeCmdEcho escapes a literal message for a Windows cmd `echo ... 1>&2`
// line. Redirection metacharacters must be caret-escaped or cmd would parse
// the `<args>` in the deprecation notice as a redirection.
func escapeCmdEcho(message string) string {
	replacer := strings.NewReplacer(
		"^", "^^",
		"<", "^<",
		">", "^>",
		"&", "^&",
		"|", "^|",
	)
	return replacer.Replace(message)
}

// posixShellQuote single-quotes a literal for POSIX sh. Single quotes are the
// only character that needs escaping inside them, so this stays correct even
// if a future message gains characters that are special in double quotes.
func posixShellQuote(message string) string {
	return "'" + strings.ReplaceAll(message, "'", `'\''`) + "'"
}

// posixDeprecationRefusalBody is a complete deprecated-alias wrapper: it
// prints the exact migration line to stderr and exits 1 without touching the
// filesystem, the network, or any sibling target.
func posixDeprecationRefusalBody(message string) string {
	return "#!/usr/bin/env sh\nset -eu\nprintf '%s\\n' " + posixShellQuote(message) + " >&2\nexit 1\n"
}

// cmdDeprecationRefusalBody is the Windows equivalent of
// posixDeprecationRefusalBody.
func cmdDeprecationRefusalBody(message string) string {
	return "@echo off\r\necho " + escapeCmdEcho(message) + " 1>&2\r\nexit /b 1\r\n"
}

// posixCLIWrapperDeprecationGuard refuses deprecated `agents-infra`
// subcommands before the wrapper creates its build directory or invokes the
// Go toolchain. It mirrors the Go dispatcher exactly: `codex` and `claude`
// refuse for any trailing args, `target` refuses only for the two deprecated
// aliases, and `target-yolo` reports the installed dange surface name.
func posixCLIWrapperDeprecationGuard() string {
	return "case \"${1:-}\" in\n" +
		"  codex)\n" +
		"    printf '%s\\n' " + posixShellQuote(deprecatedCodexMessage) + " >&2\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"  claude)\n" +
		"    printf '%s\\n' " + posixShellQuote(deprecatedClaudeMessage) + " >&2\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"  target)\n" +
		"    case \"${2:-}\" in\n" +
		"      openai-infra)\n" +
		"        printf '%s\\n' " + posixShellQuote(deprecatedOpenAIInfraMessage) + " >&2\n" +
		"        exit 1\n" +
		"        ;;\n" +
		"      anthropic-infra)\n" +
		"        printf '%s\\n' " + posixShellQuote(deprecatedAnthropicInfraMessage) + " >&2\n" +
		"        exit 1\n" +
		"        ;;\n" +
		"    esac\n" +
		"    ;;\n" +
		"  target-yolo)\n" +
		"    case \"${2:-}\" in\n" +
		"      openai-infra)\n" +
		"        printf '%s\\n' " + posixShellQuote(deprecatedOpenAIDangeMessage) + " >&2\n" +
		"        exit 1\n" +
		"        ;;\n" +
		"      anthropic-infra)\n" +
		"        printf '%s\\n' " + posixShellQuote(deprecatedAnthropicDangeMessage) + " >&2\n" +
		"        exit 1\n" +
		"        ;;\n" +
		"    esac\n" +
		"    ;;\n" +
		"esac\n"
}

// cmdCLIWrapperDeprecationGuard is the Windows cmd equivalent of
// posixCLIWrapperDeprecationGuard. Comparisons are case-sensitive like the Go
// dispatcher; `%~1` strips any caller quoting around the subcommand name.
func cmdCLIWrapperDeprecationGuard() string {
	return "if \"%~1\"==\"codex\" (\r\n" +
		"  echo " + escapeCmdEcho(deprecatedCodexMessage) + " 1>&2\r\n" +
		"  exit /b 1\r\n" +
		")\r\n" +
		"if \"%~1\"==\"claude\" (\r\n" +
		"  echo " + escapeCmdEcho(deprecatedClaudeMessage) + " 1>&2\r\n" +
		"  exit /b 1\r\n" +
		")\r\n" +
		"if \"%~1\"==\"target\" (\r\n" +
		"  if \"%~2\"==\"openai-infra\" (\r\n" +
		"    echo " + escapeCmdEcho(deprecatedOpenAIInfraMessage) + " 1>&2\r\n" +
		"    exit /b 1\r\n" +
		"  )\r\n" +
		"  if \"%~2\"==\"anthropic-infra\" (\r\n" +
		"    echo " + escapeCmdEcho(deprecatedAnthropicInfraMessage) + " 1>&2\r\n" +
		"    exit /b 1\r\n" +
		"  )\r\n" +
		")\r\n" +
		"if \"%~1\"==\"target-yolo\" (\r\n" +
		"  if \"%~2\"==\"openai-infra\" (\r\n" +
		"    echo " + escapeCmdEcho(deprecatedOpenAIDangeMessage) + " 1>&2\r\n" +
		"    exit /b 1\r\n" +
		"  )\r\n" +
		"  if \"%~2\"==\"anthropic-infra\" (\r\n" +
		"    echo " + escapeCmdEcho(deprecatedAnthropicDangeMessage) + " 1>&2\r\n" +
		"    exit /b 1\r\n" +
		"  )\r\n" +
		")\r\n"
}
