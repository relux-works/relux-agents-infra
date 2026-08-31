import Foundation

/// The prompt suite, validated completely before anything is launched.
///
/// This is F5, and it is the same sentence the gate has now been rebuilt around
/// five times: **a value this program cannot understand must stop it, not
/// default it.** Every previous round applied the rule to something the gate
/// read *out* of a runtime — an attestation, a `/v1/models` entry, a `/slots`
/// answer. This one applies it to the input that decides what the gate is
/// measuring in the first place.
///
/// The finding, driven through the shipped `benchmark-run`: a required
/// `context_75k` scenario carrying `"prefix_repeats": "2027"` — the count as a
/// JSON *string* — reached
///
/// ```swift
/// if let repeats = spec["prefix_repeats"] as? Int { … }
/// ```
///
/// which is false for a string, so the branch was skipped, the 16,232-token
/// prefix was never built, and a **15-token** prompt was measured. The pass
/// succeeded, both records sealed honestly, the transcripts were faithful, and
/// `benchmark-run` exited 0 with `accepted: true`. The capacity scenario — the
/// one that decides whether a runtime can serve the context this whole
/// evaluation exists to measure — was hollow while the decision read as earned.
/// Sealing does not help here: the gate seals and scores the wrong request
/// perfectly.
///
/// So the readers are gone. There is no `as?` cast left in
/// `BenchmarkScenarios`: the suite is decoded into ``JSONValue``, which
/// distinguishes a JSON string from a JSON number from a JSON boolean exactly,
/// validated in full, and handed to the scenario drivers as
/// ``PromptSuiteSchema/Scenario`` values that cannot carry a field the gate did
/// not understand. A suite that fails validation never reaches a launch.
///
/// **Absent and malformed are two different facts**, the same distinction
/// ``RuntimeContextWindow`` makes for `meta.n_ctx`. An optional field that is
/// simply not there is a supported shape and is listed by name in
/// ``supportedAbsences``; an optional field that is *present* and wrongly
/// typed, out of range, or incompatible with the scenario's kind is a fault.
/// Unknown keys are faults too — `prefix_repeat` for `prefix_repeats` is the
/// same 15-token prompt with a typo instead of a type error, and nothing
/// downstream could tell the difference.
///
/// **Where that stops, stated rather than implied.** Revision 4 of this file
/// claimed unknown keys were refused "at every level" while two levels went
/// unchecked, and F6 walked straight through the gap: a misspelled
/// `parameters.require` was read as the supported *absence* of `required`, the
/// tool-call parity check then demanded no arguments back, and the shipped
/// binary printed `accepted: true`. So the claim this file makes now is the
/// narrow one it can keep:
///
/// * unknown keys are refused **at the document object** and **at each
///   scenario object**, kind-scoped;
/// * inside a tool declaration, exactly the fields in ``validatedToolFields``
///   are checked, and `function.parameters.required` is **required** — an
///   explicit `[]` is how a tool says it needs no arguments;
/// * everything else in a declaration, above all the JSON-Schema parameter
///   block, is forwarded to the runtime **verbatim and unread**. That is
///   listed by name in ``unvalidatedByDesign``, because an allowlist for
///   arbitrary JSON Schema is a promise this gate cannot keep, and a
///   completeness claim that is not true is worse than the gap it hides.
///
/// Every fault in the document is collected rather than the first one thrown,
/// because an operator fixing a suite by launching two 28 GB runtimes per typo
/// is not being helped.
public enum PromptSuiteSchema {
    /// One thing wrong with the document, at the place it is wrong.
    public struct Fault: Sendable, Equatable, CustomStringConvertible {
        public let path: String
        public let detail: String

        public init(path: String, detail: String) {
            self.path = path
            self.detail = detail
        }

        public var description: String { "\(path): \(detail)" }
    }

    /// Why a suite was refused. Unreadable and malformed are kept apart for the
    /// same reason every other reading in this gate keeps them apart.
    public enum Failure: Error, Equatable, CustomStringConvertible {
        /// The bytes are not JSON, or their root is not an object. Nothing
        /// could be validated, so nothing is reported per-field.
        case unreadable(String)
        /// The document parsed and is wrong. Every fault, sorted by path.
        case malformed([Fault])

        public var description: String {
            switch self {
            case .unreadable(let detail):
                return detail
            case .malformed(let faults):
                return "the prompt suite is malformed in \(faults.count) place(s): "
                    + faults.map(\.description).joined(separator: "; ")
            }
        }
    }

    /// The four scenario shapes this gate can drive. A `kind` outside this set
    /// is a fault rather than a scenario that is quietly not run — the previous
    /// driver returned `nil` for an unknown kind and the loop skipped it, so a
    /// misspelled kind removed a required scenario from a pass without saying
    /// anything.
    public enum Kind: String, Sendable, Equatable, CaseIterable {
        case single
        case tool
        case multiturn
        case soak
    }

    /// One declared tool, kept whole so the request sends exactly what the
    /// suite wrote, with the two fields the parity check reads pulled out and
    /// validated up front rather than at measurement time.
    public struct ToolDeclaration: Sendable, Equatable {
        public let name: String
        public let requiredArguments: [String]
        /// The element verbatim, forwarded to the runtime unmodified.
        public let value: JSONValue
    }

    /// The kind-specific fields, after validation. There is no case here that
    /// can hold a value the gate did not understand.
    public enum Body: Sendable, Equatable {
        case single(prompt: String, prefixRepeats: Int?)
        case tool(prompt: String, tools: [ToolDeclaration])
        case multiturn(prefixRepeats: Int?, turns: [String])
        case soak(iterations: Int, promptTemplate: String)
    }

    public struct Scenario: Sendable, Equatable {
        public let name: String
        public let kind: Kind
        /// Absent means the driver's own default, which is pinned into both
        /// records as `maxOutputTokens`. See ``supportedAbsences``.
        public let maxTokens: Int?
        public let body: Body
    }

    public struct Suite: Sendable, Equatable {
        public let fillerParagraph: String
        public let systemPrompt: String
        public let scenarios: [String: Scenario]
    }

    // ------------------------------------------------------------- the audit

    /// Every optional field, and what its absence is taken to mean.
    ///
    /// This is the audit the rework brief asked for, written where it is
    /// enforced rather than in a document that drifts from it. An absence that
    /// is genuinely supported is fine — it just has to be a decision. A test
    /// asserts this dictionary whole, so adding or removing a supported absence
    /// happens in the open, exactly as
    /// ``RuntimeBenchmark/unpinnableConditions`` is pinned.
    ///
    /// Every field this gate *reads* and that is not listed here is required.
    /// In particular `prompt`, `turns`, `iterations`, `prompt_template`,
    /// `tools`, `kind`, `filler_paragraph`, `system_prompt`, `scenarios` and —
    /// since F6 — `tool.parameters` and `tool.parameters.required` have **no**
    /// supported absence: each of them previously defaulted to `""`, `[]` or
    /// `0`, and each of those defaults produces a scenario that measures
    /// nothing, or checks nothing, and reports success. Fields this gate does
    /// not read are neither required nor refused; they are named in
    /// ``unvalidatedByDesign``.
    public static let supportedAbsences: [String: String] = [
        "version": "documentation only, and its value is never read; the suite is pinned into "
            + "both records by SHA-256 over its bytes",
        "comment": "documentation only, and its value is never read",
        "<scenario>.max_tokens": "the driver's own default output cap, which is the value pinned "
            + "into both records as maxOutputTokens",
        "single.prefix_repeats": "no filler prefix; the scenario's prompt is sent alone. This is "
            + "short_prompt's shape",
        "multiturn.prefix_repeats": "no shared prefix before the first turn",
    ]

    /// What this validator deliberately does **not** check, named so the claim
    /// above stops exactly where the code does.
    ///
    /// F6 was found because revision 4 claimed unknown keys were refused "at
    /// every level" while two levels went unchecked. The honest claim is the
    /// narrow one: unknown keys are refused **at the document object and at
    /// each scenario object**, and inside a tool declaration only the named
    /// fields below are validated. A JSON-Schema parameter block is an opaque
    /// document this gate forwards to the runtime verbatim and has no business
    /// grading, so it is passed through unread rather than allowlisted --
    /// an allowlist for arbitrary JSON Schema is a promise that cannot be kept.
    ///
    /// The consequence is stated rather than hidden: a misspelling inside the
    /// parameter block that is *not* one of the fields below reaches the
    /// runtime unremarked. `required` is the only key in there the gate itself
    /// reads, which is why it is the only one made mandatory.
    ///
    /// Pinned whole by a test, the same treatment ``supportedAbsences`` and
    /// ``RuntimeBenchmark/unpinnableConditions`` get.
    public static let unvalidatedByDesign: [String: String] = [
        "version, comment": "recognised as keys; their values are never read and are not typed",
        "<tool>.function.description": "forwarded verbatim; the gate reads nothing out of it",
        "<tool>.function.parameters.*": "every key other than \"required\" -- \"type\", "
            + "\"properties\", \"additionalProperties\", \"$defs\", \"anyOf\" and any other "
            + "JSON-Schema keyword -- is forwarded to the runtime verbatim and is not inspected. "
            + "Only \"required\" is read by this gate, by the tool-call parity check",
        "<tool>.<other keys>": "any key beside \"type\" and \"function\" is forwarded verbatim",
    ]

    /// The tool-declaration fields the benchmark itself depends on, and which
    /// are therefore validated before launch. Everything else in a declaration
    /// is ``unvalidatedByDesign``.
    ///
    /// `function.parameters.required` is **mandatory** rather than optional,
    /// and that is F6: a misspelled `parameters.require` read as the supported
    /// *absence* of `required` left the parity check demanding no arguments
    /// back, so a runtime that called the tool with an empty argument object
    /// passed. An explicit `[]` still means "this tool deliberately requires no
    /// arguments"; it just has to be written down.
    public static let validatedToolFields: [String] = [
        "type", "function", "function.name", "function.parameters",
        "function.parameters.required",
    ]

    /// The soak template's substitution. A template without it sends the same
    /// prompt on every iteration, which lets the prompt cache serve the repeats
    /// the scenario exists to prevent.
    public static let soakIndexToken = "{index}"

    private static let documentKeys: Set<String> = [
        "version", "comment", "filler_paragraph", "system_prompt", "scenarios",
    ]

    private static func scenarioKeys(_ kind: Kind) -> Set<String> {
        switch kind {
        case .single: return ["kind", "prompt", "max_tokens", "prefix_repeats"]
        case .tool: return ["kind", "prompt", "max_tokens", "tools"]
        case .multiturn: return ["kind", "turns", "max_tokens", "prefix_repeats"]
        case .soak: return ["kind", "iterations", "prompt_template", "max_tokens"]
        }
    }

    // -------------------------------------------------------- the validation

    /// The whole document, or every reason it is not usable.
    ///
    /// - Parameter knownScenarioNames: the names the driver can actually run.
    ///   A scenario the driver has never heard of is a fault rather than dead
    ///   weight, because it is pinned into both records as part of the suite
    ///   digest and then silently never executed.
    public static func validate(
        data: Data, knownScenarioNames: Set<String>
    ) throws -> Suite {
        let root: JSONValue
        do {
            root = try JSONDecoder().decode(JSONValue.self, from: data)
        } catch {
            throw Failure.unreadable("the prompt suite is not readable as JSON: \(error)")
        }
        guard case .object(let document) = root else {
            throw Failure.unreadable(
                "the prompt suite's root is \(typeName(root)), and a suite is a JSON object")
        }

        var faults: [Fault] = []
        unknownKeys(document, allowed: documentKeys, at: "", into: &faults)
        // `version` and `comment` are recognised as keys and their values are
        // deliberately not constrained: nothing is read out of them, and a rule
        // that cannot change what the gate measures only makes it look more
        // careful than it is. The key recognition above is what matters here,
        // because a misspelled *field name* is the hazard, not a version
        // written as `1` rather than `"1"`.
        // Required non-empty, and the emptiness matters rather than being
        // tidiness: `prefix(repeats:)` multiplies this string, so an empty
        // filler makes every prefix_repeats count produce nothing at all --
        // F5's 15-token prompt by a second route.
        let filler = requiredNonEmptyString(
            document, "filler_paragraph", at: "filler_paragraph", into: &faults)
        let system = requiredNonEmptyString(
            document, "system_prompt", at: "system_prompt", into: &faults)

        var scenarios: [String: Scenario] = [:]
        switch document["scenarios"] {
        case .none:
            faults.append(
                Fault(path: "scenarios", detail: "is required and is not present"))
        case .some(.object(let entries)) where entries.isEmpty:
            faults.append(
                Fault(path: "scenarios", detail: "is present and declares no scenario"))
        case .some(.object(let entries)):
            for name in entries.keys.sorted() {
                let path = "scenarios.\(name)"
                guard knownScenarioNames.contains(name) else {
                    faults.append(
                        Fault(
                            path: path,
                            detail: "is not a scenario this driver runs; the known scenarios are "
                                + knownScenarioNames.sorted().joined(separator: ", ")))
                    continue
                }
                guard case .object(let entry) = entries[name] ?? .null else {
                    faults.append(
                        Fault(
                            path: path,
                            detail: "is \(typeName(entries[name] ?? .null)), and a scenario is a "
                                + "JSON object"))
                    continue
                }
                if let scenario = scenario(name: name, entry: entry, at: path, into: &faults) {
                    scenarios[name] = scenario
                }
            }
        case .some(let other):
            faults.append(
                Fault(
                    path: "scenarios",
                    detail: "is \(typeName(other)), and scenarios is a JSON object keyed by "
                        + "scenario name"))
        }

        guard faults.isEmpty, let filler, let system else {
            throw Failure.malformed(faults.sorted { ($0.path, $0.detail) < ($1.path, $1.detail) })
        }
        return Suite(fillerParagraph: filler, systemPrompt: system, scenarios: scenarios)
    }

    private static func scenario(
        name: String, entry: [String: JSONValue], at path: String, into faults: inout [Fault]
    ) -> Scenario? {
        guard let raw = entry["kind"] else {
            faults.append(Fault(path: "\(path).kind", detail: "is required and is not present"))
            return nil
        }
        guard case .string(let spelling) = raw, let kind = Kind(rawValue: spelling) else {
            faults.append(
                Fault(
                    path: "\(path).kind",
                    detail: "is \(literal(raw)), and a kind is one of "
                        + Kind.allCases.map(\.rawValue).joined(separator: ", ")))
            return nil
        }
        // Kind-scoped, so a field that belongs to another kind is refused for
        // what it is: `iterations` on a `single` scenario is not an unknown key
        // the author invented, it is a field the driver would never read.
        unknownKeys(entry, allowed: scenarioKeys(kind), at: path, into: &faults)
        let before = faults.count
        let maxTokens = optionalPositiveInt(
            entry, "max_tokens", at: "\(path).max_tokens", into: &faults)

        var body: Body?
        switch kind {
        case .single:
            let prompt = requiredNonEmptyString(
                entry, "prompt", at: "\(path).prompt", into: &faults)
            let repeats = optionalPositiveInt(
                entry, "prefix_repeats", at: "\(path).prefix_repeats", into: &faults)
            if let prompt { body = .single(prompt: prompt, prefixRepeats: repeats) }
        case .tool:
            let prompt = requiredNonEmptyString(
                entry, "prompt", at: "\(path).prompt", into: &faults)
            let tools = toolDeclarations(entry, at: "\(path).tools", into: &faults)
            if let prompt, let tools { body = .tool(prompt: prompt, tools: tools) }
        case .multiturn:
            let repeats = optionalPositiveInt(
                entry, "prefix_repeats", at: "\(path).prefix_repeats", into: &faults)
            let turns = requiredStrings(
                entry, "turns", at: "\(path).turns", emptyMeans: nil, into: &faults)
            if let turns { body = .multiturn(prefixRepeats: repeats, turns: turns) }
        case .soak:
            let iterations = requiredPositiveInt(
                entry, "iterations", at: "\(path).iterations", into: &faults)
            let template = requiredNonEmptyString(
                entry, "prompt_template", at: "\(path).prompt_template", into: &faults)
            if let template, !template.contains(soakIndexToken) {
                faults.append(
                    Fault(
                        path: "\(path).prompt_template",
                        detail: "does not contain \(soakIndexToken.debugDescription), so every "
                            + "iteration would send the same prompt and the prompt cache would "
                            + "serve the repeats this scenario exists to prevent"))
            } else if let iterations, let template {
                body = .soak(iterations: iterations, promptTemplate: template)
            }
        }
        guard let body, faults.count == before else { return nil }
        return Scenario(name: name, kind: kind, maxTokens: maxTokens, body: body)
    }

    // ----------------------------------------------------------- the readers

    private static func unknownKeys(
        _ object: [String: JSONValue], allowed: Set<String>, at path: String,
        into faults: inout [Fault]
    ) {
        for key in object.keys.sorted() where !allowed.contains(key) {
            faults.append(
                Fault(
                    path: path.isEmpty ? key : "\(path).\(key)",
                    detail: "is not a field this gate reads; a field it cannot understand is "
                        + "refused rather than ignored, because an ignored field and a "
                        + "misspelled one look identical from here. The fields read at this "
                        + "level are " + allowed.sorted().joined(separator: ", ")))
        }
    }

    private static func requiredNonEmptyString(
        _ object: [String: JSONValue], _ key: String, at path: String, into faults: inout [Fault]
    ) -> String? {
        guard let raw = object[key] else {
            faults.append(Fault(path: path, detail: "is required and is not present"))
            return nil
        }
        guard case .string(let value) = raw else {
            faults.append(
                Fault(path: path, detail: "is \(literal(raw)), and this field is a string"))
            return nil
        }
        guard !value.isEmpty else {
            faults.append(
                Fault(
                    path: path,
                    detail: "is present and empty; an empty value here measures something other "
                        + "than what the suite describes"))
            return nil
        }
        return value
    }

    /// A required array of non-empty strings.
    ///
    /// - Parameter emptyMeans: what an explicitly empty array is taken to mean,
    ///   or `nil` when an empty array is itself a fault. `turns: []` drove zero
    ///   requests and then reported the scenario as succeeded, so it is a
    ///   fault; `required: []` is a legitimate statement that a tool takes no
    ///   mandatory arguments, and the point of F6 is that it has to be
    ///   *stated* rather than inferred from a key that is not there.
    private static func requiredStrings(
        _ object: [String: JSONValue], _ key: String, at path: String, emptyMeans: String?,
        into faults: inout [Fault]
    ) -> [String]? {
        guard let raw = object[key] else {
            faults.append(Fault(path: path, detail: "is required and is not present"))
            return nil
        }
        guard case .array(let elements) = raw else {
            faults.append(
                Fault(
                    path: path, detail: "is \(literal(raw)), and this field is an array of strings")
            )
            return nil
        }
        if elements.isEmpty, emptyMeans == nil {
            faults.append(
                Fault(
                    path: path,
                    detail: "is present and empty; a scenario with no turns performs no exchange "
                        + "and would report success having measured nothing"))
            return nil
        }
        var values: [String] = []
        var sound = true
        for (index, element) in elements.enumerated() {
            guard case .string(let value) = element, !value.isEmpty else {
                faults.append(
                    Fault(
                        path: "\(path)[\(index)]",
                        detail: "is \(literal(element)), and each entry is a non-empty string"))
                sound = false
                continue
            }
            values.append(value)
        }
        return sound ? values : nil
    }

    /// Absent is a supported shape; present and unusable is not.
    ///
    /// The difference is the whole finding. `as? Int` on the string `"2027"`
    /// returns `nil`, which the old reader could not tell from a field nobody
    /// wrote, so it took the absence branch and dropped the prefix.
    private static func optionalPositiveInt(
        _ object: [String: JSONValue], _ key: String, at path: String, into faults: inout [Fault]
    ) -> Int? {
        guard let raw = object[key] else { return nil }
        return positiveInt(raw, at: path, into: &faults)
    }

    private static func requiredPositiveInt(
        _ object: [String: JSONValue], _ key: String, at path: String, into faults: inout [Fault]
    ) -> Int? {
        guard let raw = object[key] else {
            faults.append(Fault(path: path, detail: "is required and is not present"))
            return nil
        }
        return positiveInt(raw, at: path, into: &faults)
    }

    private static func positiveInt(
        _ raw: JSONValue, at path: String, into faults: inout [Fault]
    ) -> Int? {
        guard case .int(let value) = raw else {
            faults.append(
                Fault(
                    path: path,
                    detail: "is \(literal(raw)), and this field is a whole number. A JSON string, "
                        + "boolean or fraction is a value this gate cannot use, not a field that "
                        + "was left out"))
            return nil
        }
        guard value >= 1 else {
            faults.append(
                Fault(
                    path: path,
                    detail: "is \(value), and this field counts something that must happen at "
                        + "least once; a count of zero or less would drive no work and still "
                        + "report a result"))
            return nil
        }
        return value
    }

    private static func toolDeclarations(
        _ object: [String: JSONValue], at path: String, into faults: inout [Fault]
    ) -> [ToolDeclaration]? {
        guard let raw = object["tools"] else {
            faults.append(Fault(path: path, detail: "is required and is not present"))
            return nil
        }
        guard case .array(let elements) = raw, !elements.isEmpty else {
            faults.append(
                Fault(
                    path: path,
                    detail: "is \(literal(raw)), and this field is a non-empty array of OpenAI "
                        + "tool declarations"))
            return nil
        }
        var declarations: [ToolDeclaration] = []
        var sound = true
        for (index, element) in elements.enumerated() {
            let elementPath = "\(path)[\(index)]"
            guard case .object(let declaration) = element else {
                faults.append(
                    Fault(
                        path: elementPath,
                        detail: "is \(literal(element)), and a tool declaration is a JSON object"))
                sound = false
                continue
            }
            // Only the fields this benchmark itself reads are checked; see
            // `validatedToolFields` for the list and `unvalidatedByDesign` for
            // where that stops. The declaration is forwarded verbatim, so the
            // rest of the JSON Schema is the suite author's business.
            switch declaration["type"] {
            case .some(.string("function")):
                break
            case .none:
                faults.append(
                    Fault(
                        path: "\(elementPath).type",
                        detail: "is required and is not present; this gate drives OpenAI function "
                            + "tools, and the parity check reads a called function's name back "
                            + "out of the response"))
                sound = false
            case .some(let other):
                faults.append(
                    Fault(
                        path: "\(elementPath).type",
                        detail: "is \(literal(other)), and \"function\" is the only tool type "
                            + "whose call this gate can check parity for"))
                sound = false
            }
            guard case .object(let function)? = declaration["function"] else {
                faults.append(
                    Fault(
                        path: "\(elementPath).function",
                        detail: "is \(literal(declaration["function"] ?? .null)), and the parity "
                            + "check reads the declared function's name and required arguments "
                            + "out of this object"))
                sound = false
                continue
            }
            var name: String?
            if case .string(let value)? = function["name"], !value.isEmpty {
                name = value
            } else {
                faults.append(
                    Fault(
                        path: "\(elementPath).function.name",
                        detail: "is \(literal(function["name"] ?? .null)), and the parity check "
                            + "compares the runtime's called function against this name"))
                sound = false
            }
            // F6: `required` is mandatory. It used to be optional, and a
            // misspelled `parameters.require` was therefore read as the
            // supported absence of `required` -- the parity check then demanded
            // no argument back, an empty argument object passed, and the
            // shipped binary reported `accepted: true`. An explicit `[]` still
            // says "no mandatory arguments"; it just has to say it.
            var required: [String]?
            switch function["parameters"] {
            case .some(.object(let parameters)):
                required = requiredStrings(
                    parameters, "required",
                    at: "\(elementPath).function.parameters.required",
                    emptyMeans: "this tool deliberately requires no arguments", into: &faults)
                if required == nil { sound = false }
            case .none:
                faults.append(
                    Fault(
                        path: "\(elementPath).function.parameters",
                        detail: "is required and is not present; the parity check reads the "
                            + "declared required arguments out of it, and a tool that takes none "
                            + "says so with an explicit \"required\": []"))
                sound = false
            case .some(let other):
                faults.append(
                    Fault(
                        path: "\(elementPath).function.parameters",
                        detail: "is \(literal(other)), and a JSON-Schema parameter block is an "
                            + "object"))
                sound = false
            }
            if let name, let required {
                declarations.append(
                    ToolDeclaration(name: name, requiredArguments: required, value: element))
            }
        }
        return sound ? declarations : nil
    }

    // ------------------------------------------------------------- reporting

    private static func typeName(_ value: JSONValue) -> String {
        switch value {
        case .null: return "null"
        case .bool: return "a boolean"
        case .int: return "a whole number"
        case .double: return "a fractional number"
        case .string: return "a string"
        case .array: return "an array"
        case .object: return "an object"
        }
    }

    /// What the document actually said, so a refusal names the offending value
    /// rather than only its type. `"2027"` and `2027` have to be
    /// distinguishable in the message, because they are indistinguishable in
    /// the file to the eye that wrote them.
    private static func literal(_ value: JSONValue) -> String {
        switch value {
        case .null: return "null"
        case .bool(let flag): return "the boolean \(flag)"
        case .int(let number): return "the whole number \(number)"
        case .double(let number): return "the fractional number \(number)"
        case .string(let text): return "the string \(text.debugDescription)"
        case .array(let elements): return "an array of \(elements.count) element(s)"
        case .object(let entries): return "an object of \(entries.count) key(s)"
        }
    }
}
