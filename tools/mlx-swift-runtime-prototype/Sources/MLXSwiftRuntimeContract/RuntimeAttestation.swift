import Foundation

/// The context bound named by the running server on `GET /v1/models`.
///
/// Absence and an unreadable value are intentionally separate. Neither can be
/// satisfied by the launch argv: what the caller requested is not evidence of
/// what the running process honoured.
public enum RuntimeContextWindow: Sendable, Equatable, Codable {
    case reported(Int)
    case notReported
    case unread

    private enum CodingKeys: String, CodingKey { case state, length }
    private enum State: String, Codable { case reported, notReported, unread }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        switch try container.decode(State.self, forKey: .state) {
        case .reported:
            self = .reported(try container.decode(Int.self, forKey: .length))
        case .notReported:
            self = .notReported
        case .unread:
            self = .unread
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case .reported(let length):
            try container.encode(State.reported, forKey: .state)
            try container.encode(length, forKey: .length)
        case .notReported:
            try container.encode(State.notReported, forKey: .state)
        case .unread:
            try container.encode(State.unread, forKey: .state)
        }
    }

    /// Strictly decode the live models entry selected by model ID.
    public static func read(fromModelsEntry entry: [String: Any]) -> RuntimeContextWindow {
        guard let rawMeta = entry["meta"] else { return .notReported }
        guard let meta = rawMeta as? [String: Any] else { return .unread }
        guard let rawLength = meta["n_ctx"] else { return .notReported }
        guard !(rawLength is Bool), let number = rawLength as? NSNumber,
            CFNumberIsFloatType(number) == false
        else { return .unread }
        let length = number.intValue
        return length > 0 ? .reported(length) : .unread
    }

    /// The one observation rule consumed by context-policy decisions.
    ///
    /// `reported` is an observed value, `notReported` is an answered response
    /// where the fact was absent, and `unread` means the fact was not observed
    /// because the response could not be used. No caller-supplied value is a
    /// fourth state.
    var observation: FactObservation<Int> {
        switch self {
        case .reported(let length): .observed(length)
        case .notReported: .observedAbsent
        case .unread: .notObserved
        }
    }
}

/// One effective generation parameter reported by the running server.
///
/// The three states deliberately mirror ``RuntimeContextWindow``: an answered
/// omission is not the same fact as an unreadable response, and neither may be
/// replaced with a value decoded from another program's argv.
public enum RuntimeConfigurationValue: Sendable, Equatable, Codable {
    case reported(String)
    case notReported
    case unread

    private enum CodingKeys: String, CodingKey { case state, value }
    private enum State: String, Codable { case reported, notReported, unread }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        switch try container.decode(State.self, forKey: .state) {
        case .reported:
            self = .reported(try container.decode(String.self, forKey: .value))
        case .notReported:
            self = .notReported
        case .unread:
            self = .unread
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case .reported(let value):
            try container.encode(State.reported, forKey: .state)
            try container.encode(value, forKey: .value)
        case .notReported:
            try container.encode(State.notReported, forKey: .state)
        case .unread:
            try container.encode(State.unread, forKey: .state)
        }
    }

    var policyValue: String {
        switch self {
        case .reported(let value): value
        case .notReported: "not-reported"
        case .unread: "unread"
        }
    }
}

/// Effective pinned generation parameters read from `GET /v1/models`.
///
/// Production callers must use ``read(fromModelsEntry:)``. The launch argv is
/// retained as an assertion and for auditability, but never parsed as evidence
/// of what a different program ultimately configured.
public struct RuntimeGenerationConfiguration: Sendable, Equatable, Codable {
    public let prefillStepSize: RuntimeConfigurationValue
    public let reasoningEffort: RuntimeConfigurationValue

    public init(
        prefillStepSize: RuntimeConfigurationValue,
        reasoningEffort: RuntimeConfigurationValue
    ) {
        self.prefillStepSize = prefillStepSize
        self.reasoningEffort = reasoningEffort
    }

    public static let notReported = RuntimeGenerationConfiguration(
        prefillStepSize: .notReported, reasoningEffort: .notReported)
    public static let unread = RuntimeGenerationConfiguration(
        prefillStepSize: .unread, reasoningEffort: .unread)

    public static func reported(
        prefillStepSize: Int, reasoningEffort: String
    ) -> RuntimeGenerationConfiguration {
        RuntimeGenerationConfiguration(
            prefillStepSize: .reported(String(prefillStepSize)),
            reasoningEffort: .reported(reasoningEffort))
    }

    /// Strictly decode the live model entry selected by exact model ID.
    public static func read(fromModelsEntry entry: [String: Any])
        -> RuntimeGenerationConfiguration
    {
        guard let rawMeta = entry["meta"] else { return .notReported }
        guard let meta = rawMeta as? [String: Any] else { return .unread }
        guard let rawConfiguration = meta["runtime_config"] else { return .notReported }
        guard let configuration = rawConfiguration as? [String: Any] else { return .unread }

        let prefill: RuntimeConfigurationValue
        if let raw = configuration["prefill_step_size"] {
            if !(raw is Bool), let number = raw as? NSNumber,
                CFNumberIsFloatType(number) == false, number.intValue > 0
            {
                prefill = .reported(String(number.intValue))
            } else {
                prefill = .unread
            }
        } else {
            prefill = .notReported
        }

        let reasoning: RuntimeConfigurationValue
        if let raw = configuration["reasoning_effort"] {
            if let value = raw as? String, !value.isEmpty {
                reasoning = .reported(value)
            } else {
                reasoning = .unread
            }
        } else {
            reasoning = .notReported
        }
        return RuntimeGenerationConfiguration(
            prefillStepSize: prefill, reasoningEffort: reasoning)
    }
}

/// Whether a fact used by an admission decision was actually observed.
///
/// Runtime facts route through this type. No caller-supplied or argv-decoded
/// value is a fourth state.
enum FactObservation<Value: Sendable & Equatable>: Sendable, Equatable {
    case observed(Value)
    case observedAbsent
    case notObserved
}

/// What the benchmark invocation itself saw of one runtime's pass, written by
/// the process that launched that runtime and drove every request against it.
///
/// This type has been rewritten by three consecutive review findings on one
/// class of defect, and the shape of the fix changed each time.
///
/// * Revision 2 cross-checked ten provenance fields against the pins beside
///   them. Review handed the production gate a pair in which all ten were
///   consistent, all of them invented, and got `accepted=true`. A document was
///   being verified against itself.
/// * Revision 3 added this type, written by a separate `benchmark-attest
///   open|close` pair of subcommands against a live pid. Review then started
///   two placeholder HTTP servers that answered `GET /v1/models` and nothing
///   else, asked those same production subcommands to attest them — which they
///   did, correctly, because the processes were real — typed a set of
///   measurements beside them, and got `accepted=true` again in 7.2 seconds.
///   The gate had been made to certify its own blindness: it proved two
///   processes stayed alive, not that anything was measured.
/// * Revision 4 removes the construction rather than adding a clause to it.
///   There is no longer any way to ask this binary to attest a process the
///   caller supplies. An attestation exists only as a by-product of
///   `benchmark-run`, the single invocation that spawns the runtime, drives
///   every scenario against it, samples it, builds the record and judges the
///   pair. The caller supplies configuration; it supplies no measurement, and
///   there is no entry point through which one could be handed in.
///
/// What one invocation writes here, all of it read first-hand:
///
/// * the pid's executable path and its **start time**, from the kernel, of a
///   process this invocation spawned itself. Re-read before close, so a
///   recycled pid is refused.
/// * the SHA-256 of the executable the kernel says that pid is running.
/// * the SHA-256 of the launcher config file, read from disk.
/// * the model ID the runtime answered `/v1/models` with, over the wire.
/// * two readings of its own clock, bracketing the measured work.
/// * its own binary's SHA-256, so the binary that judges is the binary that
///   watched — the defect that separately invalidated revision 2's numbers,
///   where `19c54c5…` served and `3e5fdcc…` judged.
/// * ``transcriptDigest``: the seal over the record it built from the requests
///   it drove.
///
/// ``RuntimeBenchmark/admit(baseline:baselineAttestation:candidate:candidateAttestation:requiredScenarios:gateBinaryDigest:)``
/// requires the record to agree with all of it.
///
/// **The limit, stated rather than implied.** These are ordinary files owned by
/// the same user, so hand-authoring both a record and its attestation is still
/// physically possible. Two things changed, and only the second one matters.
/// The first: `benchmark-compare`, which reads such files, can no longer return
/// an acceptance at all — it replays an archived decision and its best outcome
/// is a reproduced rejection. The second: acceptance is reachable only from
/// `benchmark-run`, whose measurements come from exchanges it performed in the
/// same process, so producing a false acceptance now requires modifying and
/// rebuilding the gate rather than using it. That is a different class from
/// what review demonstrated three times, which was ordinary use of shipped
/// commands.
public struct RuntimeAttestation: Sendable, Equatable, Codable {
    /// Runtime identity this attestation is for. Must match the record's.
    public let runtime: String
    /// The pid the gate observed, as the driver resolved it for sampling.
    public let processID: Int
    /// Kernel-reported process start time. Re-read at ``close``: a pid that
    /// starts again under the same number is a different process, and an
    /// attestation that spanned two of them attests to neither.
    public let processStartUnixSeconds: Double
    /// Executable path the kernel says that pid is running.
    public let observedExecutablePath: String
    /// SHA-256 of the bytes at that path, computed by the gate.
    public let observedExecutableDigest: String
    /// Launcher config path and the digest the gate read off it.
    public let configPath: String
    public let configDigest: String
    /// Profile the gate was told the pass is running.
    public let profile: String
    /// Gate clock at ``open``.
    public let openedAtUnixSeconds: Double
    /// Gate clock at ``close``. `nil` means the gate never saw the pass end,
    /// which is not the same as a pass that ended and is never read as one.
    public let closedAtUnixSeconds: Double?
    /// Model ID the runtime listed when the gate asked it directly at close.
    /// `nil` until then.
    public let servedModelID: String?
    /// Context bound read from that same live models entry.
    public let observedContextWindow: RuntimeContextWindow
    /// Effective prefill and reasoning settings read from that same entry.
    public let observedGenerationConfiguration: RuntimeGenerationConfiguration?
    /// SHA-256 of the gate binary that wrote this document.
    public let gateBinaryDigest: String
    /// ``RuntimeBenchmark/transcriptDigest(of:)`` over the record this same
    /// invocation built from the requests it drove. `nil` until ``close``.
    ///
    /// This is the field that makes the document a witness to *work* rather
    /// than to a process. Everything else here says a process existed, ran a
    /// known executable and named a model; review's placeholder pair satisfied
    /// all of it while serving nothing. The seal cannot be satisfied that way,
    /// because the only code that computes it is the code that performed the
    /// exchanges it covers.
    public let transcriptDigest: String?

    public init(
        runtime: String,
        processID: Int,
        processStartUnixSeconds: Double,
        observedExecutablePath: String,
        observedExecutableDigest: String,
        configPath: String,
        configDigest: String,
        profile: String,
        openedAtUnixSeconds: Double,
        closedAtUnixSeconds: Double?,
        servedModelID: String?,
        observedContextWindow: RuntimeContextWindow = .notReported,
        observedGenerationConfiguration: RuntimeGenerationConfiguration? = nil,
        gateBinaryDigest: String,
        transcriptDigest: String?
    ) {
        self.runtime = runtime
        self.processID = processID
        self.processStartUnixSeconds = processStartUnixSeconds
        self.observedExecutablePath = observedExecutablePath
        self.observedExecutableDigest = observedExecutableDigest
        self.configPath = configPath
        self.configDigest = configDigest
        self.profile = profile
        self.openedAtUnixSeconds = openedAtUnixSeconds
        self.closedAtUnixSeconds = closedAtUnixSeconds
        self.servedModelID = servedModelID
        self.observedContextWindow = observedContextWindow
        self.observedGenerationConfiguration = observedGenerationConfiguration
        self.gateBinaryDigest = gateBinaryDigest
        self.transcriptDigest = transcriptDigest
    }

    /// The file one runtime's attestation lives at inside the directory.
    ///
    /// Shared by the gate that writes it, the driver that names the directory
    /// and the comparison that reads it, so none of the three can look in a
    /// place the others do not write.
    public static func fileName(runtime: String) -> String {
        "\(runtime).attestation.json"
    }
}
