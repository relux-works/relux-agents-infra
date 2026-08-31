import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

/// F5: a value this program cannot understand must stop it.
///
/// The finding, driven through the shipped `benchmark-run`: a required
/// `context_75k` scenario with `"prefix_repeats": "2027"` — the count as a JSON
/// *string* — ran a **15-token** prompt instead of 16,232, and the invocation
/// exited 0 with `accepted: true`. `spec["prefix_repeats"] as? Int` is `nil` for
/// a string, and the old reader could not tell that from a field nobody wrote,
/// so it took the absence branch and dropped the prefix. Nothing downstream
/// could catch it: both records sealed honestly and the transcripts faithfully
/// recorded the wrong request.
///
/// So these tests are mostly negatives, and each one is a document that the
/// previous readers would have accepted and quietly mismeasured.
///
/// **Production call sites.** `BenchmarkScenarios.Suite.init(path:)` calls
/// ``PromptSuiteSchema/validate(data:knownScenarioNames:)`` and is itself called
/// at the top of `BenchmarkRunCommand.execute` — before the session directory
/// is created, before the equivalence verdict is read, and before either
/// runtime is launched. The scenario drivers then receive
/// ``PromptSuiteSchema/Scenario`` values and perform no casts at all.
/// `scripts/benchmark-gate-smoke.sh` sections 5 and 6 drive eight malformed
/// suites through the shipped subcommand and assert that each exits nonzero
/// having emitted no decision and created no session directory.
///
/// **F6 is the same rule one nesting level deeper.** A misspelled
/// `parameters.require` was read as the supported *absence* of `required`, the
/// tool-call parity check then demanded no argument back, and the shipped
/// binary printed `accepted: true`. `required` is mandatory now, and an
/// explicit `[]` is how a tool says it takes none.
///
/// **What these tests do not claim.** Unknown keys are refused at the document
/// object and at each scenario object; inside a tool declaration only
/// ``PromptSuiteSchema/validatedToolFields`` are checked. The JSON-Schema
/// parameter block below them is forwarded verbatim and unread — see
/// ``PromptSuiteSchema/unvalidatedByDesign``, pinned by
/// `toolValidationBoundaryIsPinned`. Revision 4 claimed unknown keys were
/// refused "at every level" while two levels went unchecked, which is how F6
/// got through; the claim here stops where the code does.
@Suite("prompt suite schema")
struct PromptSuiteSchemaTests {
    static let known: Set<String> = [
        "short_prompt", "long_prompt_8k", "tool_call", "multiturn_prefix_reuse",
        "stability_soak", "context_75k",
    ]

    /// A whole document around one scenario body, so each case differs only in
    /// the thing it is about.
    static func document(_ name: String, _ scenario: String, filler: String = "\"Filler. \"")
        -> Data
    {
        Data(
            """
            {
              "version": "test-1",
              "filler_paragraph": \(filler),
              "system_prompt": "You are a maintenance assistant.",
              "scenarios": {\(name.debugDescription): \(scenario)}
            }
            """.utf8)
    }

    static func validate(_ data: Data) throws -> PromptSuiteSchema.Suite {
        try PromptSuiteSchema.validate(data: data, knownScenarioNames: known)
    }

    /// Every fault, as `path: detail` strings, or `nil` if the suite validated.
    static func faults(_ data: Data) -> [String]? {
        do {
            _ = try validate(data)
            return nil
        } catch let failure as PromptSuiteSchema.Failure {
            guard case .malformed(let faults) = failure else { return ["unreadable"] }
            return faults.map(\.description)
        } catch {
            return ["unexpected \(error)"]
        }
    }

    static func refuses(_ data: Data, mentioning fragment: String) -> Bool {
        guard let faults = faults(data) else { return false }
        return faults.contains { $0.contains(fragment) }
    }

    // ================================================== the finding, exactly

    @Test("a prefix count written as a JSON string is refused, not read as absent")
    func stringPrefixRepeatsIsRefused() throws {
        let data = Self.document(
            "context_75k",
            """
            {"kind": "single", "prefix_repeats": "2027", "prompt": "Name three.",
             "max_tokens": 16}
            """)
        #expect(Self.refuses(data, mentioning: "scenarios.context_75k.prefix_repeats"))
        // The message has to distinguish `"2027"` from `2027`, because the file
        // does not distinguish them to the eye that wrote it.
        #expect(Self.refuses(data, mentioning: "the string \"2027\""))
    }

    /// The same document with the quotes removed is the suite this repository
    /// ships, and it must still validate. Without this, "refuse the string"
    /// could be satisfied by refusing every `prefix_repeats`.
    @Test("the same count written as a number is the shipped capacity scenario")
    func numericPrefixRepeatsIsAccepted() throws {
        let suite = try Self.validate(
            Self.document(
                "context_75k",
                """
                {"kind": "single", "prefix_repeats": 2027, "prompt": "Name three.",
                 "max_tokens": 16}
                """))
        let scenario = try #require(suite.scenarios["context_75k"])
        #expect(scenario.maxTokens == 16)
        #expect(scenario.body == .single(prompt: "Name three.", prefixRepeats: 2027))
    }

    @Test("a boolean, a fraction and a zero are all refused where a count belongs")
    func nonCountPrefixRepeatsAreRefused() throws {
        for literal in ["true", "20.5", "0", "-3", "null", "[]"] {
            let data = Self.document(
                "long_prompt_8k",
                """
                {"kind": "single", "prefix_repeats": \(literal), "prompt": "Name three."}
                """)
            #expect(
                Self.refuses(data, mentioning: "scenarios.long_prompt_8k.prefix_repeats"),
                "prefix_repeats: \(literal) was not refused")
        }
    }

    /// The typo case. `prefix_repeat` is not a wrong type — it is a field name
    /// the gate has never heard of, and ignoring it produces exactly the same
    /// 15-token prompt the finding reported.
    @Test("a misspelled field is refused rather than ignored")
    func unknownScenarioFieldIsRefused() throws {
        let data = Self.document(
            "context_75k",
            """
            {"kind": "single", "prefix_repeat": 2027, "prompt": "Name three."}
            """)
        #expect(Self.refuses(data, mentioning: "scenarios.context_75k.prefix_repeat"))
    }

    // ======================================= absences that are decisions

    /// `short_prompt` is the shipped scenario with no prefix at all, so this
    /// absence has to stay supported or the pinned suite stops validating.
    @Test("a scenario that writes no prefix count sends its prompt alone")
    func absentPrefixRepeatsIsSupported() throws {
        let suite = try Self.validate(
            Self.document("short_prompt", #"{"kind": "single", "prompt": "List two checks."}"#))
        let scenario = try #require(suite.scenarios["short_prompt"])
        #expect(scenario.body == .single(prompt: "List two checks.", prefixRepeats: nil))
        // And no cap either: absent means the driver's own default, which is
        // the number pinned into both records.
        #expect(scenario.maxTokens == nil)
    }

    // ============================================================ F6, exactly

    /// F6: `parameters.require` — one letter short — was read as the supported
    /// *absence* of `required`, so `requiredArguments` came out `[]`, the
    /// parity check in `BenchmarkScenarios.tool` demanded no argument back, a
    /// runtime that called the tool with `{}` passed, and the shipped
    /// `benchmark-run` printed `accepted: true`. The absence is gone: an
    /// explicit array is mandatory.
    @Test("a misspelled parameters.require is refused, not read as an absent required")
    func misspelledRequiredIsRefused() throws {
        let data = Self.document(
            "tool_call",
            """
            {"kind": "tool", "prompt": "Read it.",
             "tools": [{"type": "function", "function": {"name": "read_pressure",
               "parameters": {"type": "object",
                 "properties": {"vehicle": {"type": "integer"}},
                 "require": ["vehicle"]}}}]}
            """)
        #expect(
            Self.refuses(
                data,
                mentioning:
                    "scenarios.tool_call.tools[0].function.parameters.required: is required"))
    }

    /// The narrowing control that keeps the fix above from being "refuse every
    /// tool": a tool that genuinely takes no mandatory arguments says so, and
    /// is accepted with an empty demand list.
    @Test("an explicit empty required array is accepted and demands nothing back")
    func explicitEmptyRequiredIsSupported() throws {
        let suite = try Self.validate(
            Self.document(
                "tool_call",
                """
                {"kind": "tool", "prompt": "Read it.",
                 "tools": [{"type": "function", "function": {"name": "read_pressure",
                   "parameters": {"type": "object", "required": []}}}]}
                """))
        let scenario = try #require(suite.scenarios["tool_call"])
        guard case .tool(_, let tools) = scenario.body else {
            Issue.record("not a tool scenario")
            return
        }
        #expect(tools.count == 1)
        #expect(tools[0].name == "read_pressure")
        #expect(tools[0].requiredArguments.isEmpty)
    }

    @Test("an absent required array is refused rather than meaning no arguments")
    func absentRequiredIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document(
                    "tool_call",
                    """
                    {"kind": "tool", "prompt": "Read it.",
                     "tools": [{"type": "function", "function": {"name": "read_pressure",
                       "parameters": {"type": "object"}}}]}
                    """),
                mentioning:
                    "scenarios.tool_call.tools[0].function.parameters.required: is required"))
    }

    @Test("an absent parameters block is refused")
    func absentParametersIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document(
                    "tool_call",
                    """
                    {"kind": "tool", "prompt": "Read it.",
                     "tools": [{"type": "function", "function": {"name": "read_pressure"}}]}
                    """),
                mentioning: "scenarios.tool_call.tools[0].function.parameters: is required"))
    }

    @Test("a tool that is not a function declaration is refused")
    func nonFunctionToolTypeIsRefused() throws {
        for declaration in [
            #"{"function": {"name": "f", "parameters": {"required": []}}}"#,
            #"{"type": "code_interpreter", "function": {"name": "f", "parameters": {"required": []}}}"#,
            #"{"type": 7, "function": {"name": "f", "parameters": {"required": []}}}"#,
        ] {
            #expect(
                Self.refuses(
                    Self.document(
                        "tool_call",
                        #"{"kind": "tool", "prompt": "R", "tools": [\#(declaration)]}"#),
                    mentioning: "scenarios.tool_call.tools[0].type"),
                "\(declaration) was not refused for its type")
        }
    }

    /// The other half of the honesty claim: the parameter block below is an
    /// opaque JSON-Schema document carrying keywords this gate has never heard
    /// of, and it is forwarded **verbatim** rather than allowlisted. Refusing
    /// it would be the completeness promise revision 4 made and could not keep.
    @Test("an opaque JSON-Schema parameter block is forwarded verbatim, not graded")
    func opaqueParameterSchemaIsPreserved() throws {
        let suite = try Self.validate(
            Self.document(
                "tool_call",
                """
                {"kind": "tool", "prompt": "Read it.",
                 "tools": [{"type": "function", "function": {"name": "read_pressure",
                   "description": "Read coolant pressure.",
                   "strict": true,
                   "parameters": {"type": "object", "additionalProperties": false,
                     "$defs": {"id": {"type": "integer"}},
                     "properties": {"vehicle": {"$ref": "#/$defs/id"}},
                     "required": ["vehicle"]}}}]}
                """))
        let scenario = try #require(suite.scenarios["tool_call"])
        guard case .tool(_, let tools) = scenario.body else {
            Issue.record("not a tool scenario")
            return
        }
        #expect(tools[0].requiredArguments == ["vehicle"])
        let function = try #require(tools[0].value.objectValue?["function"]?.objectValue)
        #expect(function["strict"] == .bool(true))
        let parameters = try #require(function["parameters"]?.objectValue)
        #expect(parameters["additionalProperties"] == .bool(false))
        #expect(parameters["$defs"]?.objectValue?["id"] != nil)
    }

    /// The boundary of the claim, pinned so it cannot quietly widen again.
    @Test("the validated tool fields and the unvalidated surface are both pinned")
    func toolValidationBoundaryIsPinned() throws {
        #expect(
            PromptSuiteSchema.validatedToolFields == [
                "type", "function", "function.name", "function.parameters",
                "function.parameters.required",
            ])
        #expect(
            Set(PromptSuiteSchema.unvalidatedByDesign.keys) == [
                "version, comment",
                "<tool>.function.description",
                "<tool>.function.parameters.*",
                "<tool>.<other keys>",
            ])
        for (field, reason) in PromptSuiteSchema.unvalidatedByDesign {
            #expect(!reason.isEmpty, "\(field) has no recorded reason")
        }
    }

    /// The audit, asserted whole. Adding or removing a supported absence has to
    /// happen in the open, the same way ``RuntimeBenchmark/unpinnableConditions``
    /// is pinned — an absence is allowed to be a decision and not allowed to be
    /// an accident.
    @Test("the supported absences are exactly these five")
    func supportedAbsencesArePinned() throws {
        // Five, not six: F6 removed `tool.parameters.required`, because a
        // misspelled key is indistinguishable from an absent one and this one
        // bought an `accepted: true`.
        #expect(
            Set(PromptSuiteSchema.supportedAbsences.keys) == [
                "version",
                "comment",
                "<scenario>.max_tokens",
                "single.prefix_repeats",
                "multiturn.prefix_repeats",
            ])
        for (field, reason) in PromptSuiteSchema.supportedAbsences {
            #expect(!reason.isEmpty, "\(field) has no recorded reason")
        }
    }

    // ============================================ every other scenario reader

    @Test("a single scenario with no prompt is refused")
    func absentPromptIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document("short_prompt", #"{"kind": "single", "max_tokens": 32}"#),
                mentioning: "scenarios.short_prompt.prompt: is required"))
    }

    @Test("an empty or wrongly typed prompt is refused")
    func malformedPromptIsRefused() throws {
        for literal in ["\"\"", "42", "null", "[\"a\"]"] {
            #expect(
                Self.refuses(
                    Self.document("short_prompt", #"{"kind": "single", "prompt": \#(literal)}"#),
                    mentioning: "scenarios.short_prompt.prompt"),
                "prompt: \(literal) was not refused")
        }
    }

    /// `turns` defaulted to `[]`, which drove zero requests and then reported
    /// the scenario as succeeded with no exchanges at all.
    @Test("a multiturn scenario with no turns is refused")
    func absentOrEmptyTurnsAreRefused() throws {
        for literal in [nil, "[]", "\"only one\"", "[\"\"]", "[1, 2]"] {
            let body =
                literal.map { #"{"kind": "multiturn", "prefix_repeats": 215, "turns": \#($0)}"# }
                ?? #"{"kind": "multiturn", "prefix_repeats": 215}"#
            #expect(
                Self.refuses(
                    Self.document("multiturn_prefix_reuse", body),
                    mentioning: "scenarios.multiturn_prefix_reuse.turns"),
                "turns: \(literal ?? "absent") was not refused")
        }
    }

    @Test("a multiturn scenario with real turns validates")
    func multiturnValidates() throws {
        let suite = try Self.validate(
            Self.document(
                "multiturn_prefix_reuse",
                """
                {"kind": "multiturn", "prefix_repeats": 215, "max_tokens": 64,
                 "turns": ["First?", "Second?"]}
                """))
        let scenario = try #require(suite.scenarios["multiturn_prefix_reuse"])
        #expect(scenario.body == .multiturn(prefixRepeats: 215, turns: ["First?", "Second?"]))
    }

    /// `iterations` defaulted to `0`: the loop did nothing and the scenario
    /// reported success having performed no exchange.
    @Test("a soak with no iterations, or a string count, is refused")
    func malformedIterationsAreRefused() throws {
        for literal in [nil, "0", "\"20\"", "20.5", "true"] {
            let body =
                literal.map {
                    #"{"kind": "soak", "iterations": \#($0), "prompt_template": "Job {index}."}"#
                } ?? #"{"kind": "soak", "prompt_template": "Job {index}."}"#
            #expect(
                Self.refuses(
                    Self.document("stability_soak", body),
                    mentioning: "scenarios.stability_soak.iterations"),
                "iterations: \(literal ?? "absent") was not refused")
        }
    }

    /// `20.0` is the same whole number as `20`, and JSON does not distinguish
    /// them; `20.5` is not a count and is refused. The line is drawn at "can
    /// this be used as written", not at the spelling.
    @Test("a whole number written with a decimal point is still a count")
    func wholeNumberWithFractionPartIsACount() throws {
        let suite = try Self.validate(
            Self.document(
                "stability_soak",
                #"{"kind": "soak", "iterations": 20.0, "prompt_template": "Job {index}."}"#))
        #expect(
            suite.scenarios["stability_soak"]?.body
                == .soak(iterations: 20, promptTemplate: "Job {index}."))
    }

    /// The scenario exists to defeat the prompt cache. A template with no
    /// substitution sends one prompt twenty times, which measures the cache.
    @Test("a soak template that does not vary by index is refused")
    func staticSoakTemplateIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document(
                    "stability_soak",
                    #"{"kind": "soak", "iterations": 20, "prompt_template": "Summarise it."}"#),
                mentioning: "{index}"))
        let suite = try Self.validate(
            Self.document(
                "stability_soak",
                #"{"kind": "soak", "iterations": 20, "prompt_template": "Job {index}."}"#))
        #expect(
            suite.scenarios["stability_soak"]?.body
                == .soak(iterations: 20, promptTemplate: "Job {index}."))
    }

    @Test("a tool scenario whose declarations cannot be read is refused")
    func malformedToolsAreRefused() throws {
        let cases: [(String, String)] = [
            (#"{"kind": "tool", "prompt": "Read it."}"#, "tools: is required"),
            (#"{"kind": "tool", "prompt": "Read it.", "tools": []}"#, "tools"),
            (#"{"kind": "tool", "prompt": "Read it.", "tools": {}}"#, "tools"),
            (
                #"{"kind": "tool", "prompt": "Read it.", "tools": [{"type": "function"}]}"#,
                "tools[0].function"
            ),
            (
                #"{"kind": "tool", "prompt": "R", "tools": ["read_pressure"]}"#,
                "tools[0]"
            ),
            (
                #"""
                {"kind": "tool", "prompt": "R", "tools": [{"type": "function",
                 "function": {"name": "", "parameters": {"required": []}}}]}
                """#,
                "tools[0].function.name"
            ),
            (
                #"""
                {"kind": "tool", "prompt": "R", "tools": [{"type": "function",
                 "function": {"name": 7, "parameters": {"required": []}}}]}
                """#,
                "tools[0].function.name"
            ),
            (
                #"""
                {"kind": "tool", "prompt": "R", "tools": [{"type": "function",
                 "function": {"name": "f", "parameters": {"required": "vehicle"}}}]}
                """#,
                "tools[0].function.parameters.required"
            ),
            (
                #"""
                {"kind": "tool", "prompt": "R", "tools": [{"type": "function",
                 "function": {"name": "f", "parameters": {"required": ["vehicle", 7]}}}]}
                """#,
                "tools[0].function.parameters.required[1]"
            ),
            (
                #"""
                {"kind": "tool", "prompt": "R", "tools": [{"type": "function",
                 "function": {"name": "f", "parameters": ["required"]}}]}
                """#,
                "tools[0].function.parameters"
            ),
        ]
        for (body, fragment) in cases {
            #expect(
                Self.refuses(Self.document("tool_call", body), mentioning: fragment),
                "\(body) was not refused for \(fragment)")
        }
    }

    @Test("a well-formed tool scenario keeps the declaration verbatim")
    func toolScenarioValidates() throws {
        let suite = try Self.validate(
            Self.document(
                "tool_call",
                """
                {"kind": "tool", "prompt": "Read it.", "max_tokens": 32,
                 "tools": [{"type": "function", "function": {"name": "read_pressure",
                   "parameters": {"type": "object",
                     "properties": {"vehicle": {"type": "integer"}},
                     "required": ["vehicle"]}}}]}
                """))
        let scenario = try #require(suite.scenarios["tool_call"])
        guard case .tool(let prompt, let tools) = scenario.body else {
            Issue.record("not a tool scenario")
            return
        }
        #expect(prompt == "Read it.")
        #expect(tools[0].requiredArguments == ["vehicle"])
        // Forwarded unmodified: what is sent is what the suite wrote, and the
        // suite's bytes are what both records pin.
        let object = try #require(tools[0].value.objectValue)
        #expect(object["type"] == .string("function"))
        #expect(object["function"]?.objectValue?["name"] == .string("read_pressure"))
    }

    @Test("a bad output cap is refused wherever it appears")
    func malformedMaxTokensIsRefused() throws {
        for literal in ["\"16\"", "0", "-1", "16.5", "true"] {
            #expect(
                Self.refuses(
                    Self.document(
                        "short_prompt",
                        #"{"kind": "single", "prompt": "Go.", "max_tokens": \#(literal)}"#),
                    mentioning: "scenarios.short_prompt.max_tokens"),
                "max_tokens: \(literal) was not refused")
        }
    }

    // ================================================= kind and compatibility

    @Test("a missing or unrecognised kind is refused rather than skipped")
    func malformedKindIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document("short_prompt", #"{"prompt": "Go."}"#),
                mentioning: "scenarios.short_prompt.kind: is required"))
        #expect(
            Self.refuses(
                Self.document("short_prompt", #"{"kind": "singel", "prompt": "Go."}"#),
                mentioning: "scenarios.short_prompt.kind"))
        #expect(
            Self.refuses(
                Self.document("short_prompt", #"{"kind": 1, "prompt": "Go."}"#),
                mentioning: "scenarios.short_prompt.kind"))
    }

    /// A field that belongs to another kind is not an invented name — it is a
    /// field this driver would never read for this scenario.
    @Test("a field from another kind is refused")
    func scenarioIncompatibleFieldsAreRefused() throws {
        let cases: [(String, String, String)] = [
            (
                "short_prompt", #"{"kind": "single", "prompt": "Go.", "iterations": 20}"#,
                "iterations"
            ),
            (
                "short_prompt",
                #"{"kind": "single", "prompt": "Go.", "tools": [{"function": {"name": "f"}}]}"#,
                "tools"
            ),
            (
                "stability_soak",
                #"""
                {"kind": "soak", "iterations": 3, "prompt_template": "J {index}.",
                 "prefix_repeats": 4}
                """#,
                "prefix_repeats"
            ),
            (
                "multiturn_prefix_reuse",
                #"{"kind": "multiturn", "turns": ["a"], "prompt": "Go."}"#, "prompt"
            ),
        ]
        for (name, body, field) in cases {
            #expect(
                Self.refuses(Self.document(name, body), mentioning: "scenarios.\(name).\(field)"),
                "\(field) on \(name) was not refused")
        }
    }

    // ======================================================= the document

    /// `prefix(repeats:)` multiplies this string, so an empty filler makes
    /// every prefix count produce nothing — the finding's 15-token prompt by a
    /// second route, with no malformed field anywhere.
    @Test("an empty filler paragraph is refused")
    func emptyFillerIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document(
                    "long_prompt_8k",
                    #"{"kind": "single", "prefix_repeats": 215, "prompt": "Go."}"#,
                    filler: "\"\""),
                mentioning: "filler_paragraph"))
    }

    @Test("the document's own required fields are required")
    func documentFieldsAreRequired() throws {
        let one = #"{"kind": "single", "prompt": "Go."}"#
        let noFiller = Data(#"{"system_prompt": "S", "scenarios": {"short_prompt": \#(one)}}"#.utf8)
        #expect(Self.refuses(noFiller, mentioning: "filler_paragraph: is required"))
        let noSystem = Data(
            #"{"filler_paragraph": "F", "scenarios": {"short_prompt": \#(one)}}"#.utf8)
        #expect(Self.refuses(noSystem, mentioning: "system_prompt: is required"))
        let noScenarios = Data(#"{"filler_paragraph": "F", "system_prompt": "S"}"#.utf8)
        #expect(Self.refuses(noScenarios, mentioning: "scenarios: is required"))
        let emptyScenarios = Data(
            #"{"filler_paragraph": "F", "system_prompt": "S", "scenarios": {}}"#.utf8)
        #expect(Self.refuses(emptyScenarios, mentioning: "scenarios"))
        let listScenarios = Data(
            #"{"filler_paragraph": "F", "system_prompt": "S", "scenarios": []}"#.utf8)
        #expect(Self.refuses(listScenarios, mentioning: "scenarios"))
    }

    /// `version` and `comment` are recognised keys whose values are never read,
    /// and the shipped suite writes `"version": 1` as a number. A type rule
    /// there would refuse the pinned suite to buy nothing.
    @Test("the documentation fields are recognised but not typed")
    func documentationFieldsAreNotTyped() throws {
        let data = Data(
            #"""
            {"version": 1, "comment": ["a", "b"], "filler_paragraph": "F",
             "system_prompt": "S",
             "scenarios": {"short_prompt": {"kind": "single", "prompt": "Go."}}}
            """#.utf8)
        #expect(Self.faults(data) == nil, "\(Self.faults(data) ?? [])")
    }

    @Test("an unknown top-level field is refused")
    func unknownDocumentFieldIsRefused() throws {
        let data = Data(
            #"""
            {"filler_paragraph": "F", "system_prompt": "S", "filler_paragrahp": "F",
             "scenarios": {"short_prompt": {"kind": "single", "prompt": "Go."}}}
            """#.utf8)

        #expect(Self.refuses(data, mentioning: "filler_paragrahp"))
    }

    /// A scenario the driver's loop never visits is still pinned into both
    /// records by the suite digest, and then silently not run.
    @Test("a scenario this driver does not run is refused")
    func unknownScenarioNameIsRefused() throws {
        #expect(
            Self.refuses(
                Self.document("long_prompt_16k", #"{"kind": "single", "prompt": "Go."}"#),
                mentioning: "scenarios.long_prompt_16k"))
    }

    @Test("a scenario that is not an object is refused")
    func nonObjectScenarioIsRefused() throws {
        #expect(
            Self.refuses(Self.document("short_prompt", "\"single\""), mentioning: "short_prompt"))
    }

    /// Unreadable and malformed are different facts, kept apart at the top of
    /// the validator for the same reason every other reading in this gate keeps
    /// them apart.
    @Test("bytes that are not a JSON object are unreadable, not malformed")
    func unreadableIsItsOwnFact() throws {
        for bytes in ["not json at all", "[1, 2, 3]", "\"a suite\"", ""] {
            do {
                _ = try Self.validate(Data(bytes.utf8))
                Issue.record("\(bytes.debugDescription) validated")
            } catch let failure as PromptSuiteSchema.Failure {
                guard case .unreadable = failure else {
                    Issue.record("\(bytes.debugDescription) was reported as malformed")
                    continue
                }
            }
        }
    }

    /// An operator fixing a suite by launching two 28 GB runtimes per typo is
    /// not being helped.
    @Test("every fault is reported, not only the first")
    func allFaultsAreCollected() throws {
        let data = Data(
            #"""
            {"filler_paragraph": "", "system_prompt": "S",
             "scenarios": {
               "short_prompt": {"kind": "single", "prompt": "Go.", "max_tokens": "32"},
               "stability_soak": {"kind": "soak", "iterations": "20",
                                  "prompt_template": "Static."}}}
            """#.utf8)
        let faults = try #require(Self.faults(data))
        #expect(faults.count >= 4, "\(faults)")
        #expect(faults.contains { $0.hasPrefix("filler_paragraph") })
        #expect(faults.contains { $0.hasPrefix("scenarios.short_prompt.max_tokens") })
        #expect(faults.contains { $0.hasPrefix("scenarios.stability_soak.iterations") })
        #expect(faults.contains { $0.hasPrefix("scenarios.stability_soak.prompt_template") })
        // Sorted by path, so two runs of the same broken file report the same
        // thing in the same order.
        #expect(faults == faults.sorted())
    }

    // ============================================== the shipped suite, whole

    /// The pinned six-scenario suite this repository ships, in the shape
    /// `examples/benchmark-prompts.json` has. This is the positive that a
    /// narrowing mutant costs: any rule that makes an optional field mandatory
    /// reddens it.
    @Test("the shipped six-scenario suite validates")
    func shippedSuiteValidates() throws {
        let data = Data(
            #"""
            {
              "version": "benchmark-1",
              "comment": "The pinned suite.",
              "filler_paragraph": "Coolant loop maintenance notes. ",
              "system_prompt": "You are a maintenance assistant.",
              "scenarios": {
                "short_prompt": {"kind": "single", "prompt": "List two checks.",
                                 "max_tokens": 256},
                "long_prompt_8k": {"kind": "single", "prefix_repeats": 215,
                                   "prompt": "Summarise.", "max_tokens": 256},
                "context_75k": {"kind": "single", "prefix_repeats": 2027,
                                "prompt": "Name three.", "max_tokens": 16},
                "tool_call": {"kind": "tool", "prompt": "Use the tool.", "max_tokens": 256,
                  "tools": [{"type": "function", "function": {"name": "read_pressure",
                    "description": "Read coolant pressure.",
                    "parameters": {"type": "object",
                      "properties": {"vehicle": {"type": "integer"}},
                      "required": ["vehicle"]}}}]},
                "multiturn_prefix_reuse": {"kind": "multiturn", "prefix_repeats": 215,
                  "turns": ["First?", "Second?"], "max_tokens": 64},
                "stability_soak": {"kind": "soak", "iterations": 20,
                  "prompt_template": "Summarise inspection {index}.", "max_tokens": 64}
              }
            }
            """#.utf8)
        let suite = try Self.validate(data)
        #expect(Set(suite.scenarios.keys) == Self.known)
        #expect(suite.scenarios["context_75k"]?.kind == .single)
        #expect(suite.scenarios["stability_soak"]?.kind == .soak)
        #expect(suite.fillerParagraph == "Coolant loop maintenance notes. ")
    }
}
