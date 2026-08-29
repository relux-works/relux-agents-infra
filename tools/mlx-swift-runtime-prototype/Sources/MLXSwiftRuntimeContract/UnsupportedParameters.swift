import Foundation

/// Request fields this runtime does not implement.
///
/// Every entry is a field whose presence would change the result if it were
/// honoured. They are refused rather than dropped: a caller that sets `stop` or
/// `response_format` and gets a normal 200 back has been told the constraint was
/// applied when it was not.
///
/// Fields that are merely *inert* (for example `user`) are not listed — ignoring
/// metadata misrepresents nothing.
public enum UnsupportedParameters {
    /// Refused whenever the field is present and non-null.
    public static let alwaysRefused: [String] = [
        "stop",
        "logit_bias",
        "top_logprobs",
        "response_format",
        // The `<think>` splitter is configured from the startup template state.
        // Letting a request flip `enable_thinking` would misfile the answer as
        // reasoning, so template kwargs are refused instead of forwarded.
        "chat_template_kwargs",
    ]

    /// Detect unsupported fields in a decoded request body.
    ///
    /// Returned in a stable order so the refusal message is deterministic.
    public static func present(in object: [String: JSONValue]) -> [String] {
        var found: [String] = []

        for field in alwaysRefused where (object[field] ?? .null) != .null {
            found.append(field)
        }

        // `n` is only a problem when more than one choice is requested.
        switch object["n"] {
        case .some(.int(let count)) where count != 1:
            found.append("n")
        case .some(.double(let count)) where count != 1:
            found.append("n")
        default:
            break
        }

        // `logprobs` is inert when explicitly disabled.
        if case .some(.bool(true)) = object["logprobs"] {
            found.append("logprobs")
        }

        // `tool_choice` is honoured only in its default form: the runtime cannot
        // force or forbid a call, so any other value would be a false promise.
        switch object["tool_choice"] {
        case .none, .some(.null), .some(.string("auto")):
            break
        default:
            found.append("tool_choice")
        }

        return found.sorted()
    }
}
