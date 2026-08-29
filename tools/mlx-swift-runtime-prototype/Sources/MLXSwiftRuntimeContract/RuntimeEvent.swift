import Foundation

/// A structured line the runtime writes to stdout.
///
/// `model-harness` forwards a managed child's stdout and stderr unchanged and
/// matches literal fatal substrings against them, so one JSON object per line is
/// the natural place to report load time and resident memory: it survives the
/// forwarding path and stays greppable.
public struct RuntimeEvent: Sendable, Equatable {
    public let name: String
    public let fields: [String: JSONValue]

    public init(name: String, fields: [String: JSONValue]) {
        self.name = name
        self.fields = fields
    }

    public var json: JSONValue {
        var object = fields
        object["event"] = .string(name)
        return .object(object)
    }

    public func line() throws -> String {
        try JSONEncoding.string(json)
    }

    /// Memory readings are optional on purpose: a failed `task_info` call is
    /// reported as `null`, never as `0`. A zero would read as "the model uses
    /// no memory", which is a measurement this runtime cannot make.
    public static func modelLoaded(
        modelID: String,
        modelPath: String,
        loadSeconds: Double,
        residentBytes: UInt64?,
        physicalFootprintBytes: UInt64?,
        modelType: String
    ) -> RuntimeEvent {
        RuntimeEvent(
            name: "model_loaded",
            fields: [
                "model_id": .string(modelID),
                "model_path": .string(modelPath),
                "model_type": .string(modelType),
                "load_seconds": .double((loadSeconds * 1000).rounded() / 1000),
                "resident_bytes": bytes(residentBytes),
                "resident_mib": mebibytes(residentBytes),
                "physical_footprint_bytes": bytes(physicalFootprintBytes),
                "physical_footprint_mib": mebibytes(physicalFootprintBytes),
            ])
    }

    private static func bytes(_ value: UInt64?) -> JSONValue {
        value.map { .int(Int(clamping: $0)) } ?? .null
    }

    private static func mebibytes(_ value: UInt64?) -> JSONValue {
        value.map { .double((Double($0) / 1_048_576 * 10).rounded() / 10) } ?? .null
    }
}

/// Server-sent-event framing for streaming completions.
public enum ServerSentEvent {
    public static func data(_ body: JSONValue) throws -> String {
        "data: \(try JSONEncoding.string(body))\n\n"
    }

    public static let done = "data: [DONE]\n\n"
}
