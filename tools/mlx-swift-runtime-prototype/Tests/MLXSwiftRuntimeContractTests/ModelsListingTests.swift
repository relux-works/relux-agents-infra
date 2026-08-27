import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("models listing readiness gate")
struct ModelsListingTests {
    static let modelID = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"

    static func advertisedIDs(_ listing: ModelsListing) -> [String] {
        guard case .array(let entries)? = listing.body.objectValue?["data"] else { return [] }
        return entries.compactMap {
            guard case .string(let id)? = $0.objectValue?["id"] else { return nil }
            return id
        }
    }

    @Test("a ready runtime advertises exactly the configured model")
    func advertisesWhenReady() {
        let listing = ModelsListing.make(modelID: Self.modelID, readiness: .ready, created: 1)
        #expect(listing.status == 200)
        #expect(Self.advertisedIDs(listing) == [Self.modelID])
        // The managed launcher parses `object == "list"` before scanning data.
        #expect(listing.body.objectValue?["object"] == .string("list"))
    }

    @Test(
        "a runtime that is not ready never advertises the model",
        arguments: [
            RuntimeReadiness.loading,
            RuntimeReadiness.shuttingDown,
            RuntimeReadiness.failed("metal OOM"),
        ])
    func withholdsModelUntilResident(readiness: RuntimeReadiness) {
        // This is the readiness gate. The launcher retries on 503 and accepts
        // a 200 whose data[] contains the exact model ID; publishing the ID
        // while the weights are still loading would make the launcher report
        // ready before the first token could ever be produced.
        let listing = ModelsListing.make(modelID: Self.modelID, readiness: readiness, created: 1)
        #expect(listing.status == 503)
        #expect(Self.advertisedIDs(listing).isEmpty)
        #expect(!Self.advertisedIDs(listing).contains(Self.modelID))
    }

    @Test("a failed load reports the reason alongside the empty list")
    func reportsLoadFailure() {
        let listing = ModelsListing.make(
            modelID: Self.modelID, readiness: .failed("unsupported model type"), created: 1)
        #expect(listing.status == 503)
        #expect(Self.advertisedIDs(listing).isEmpty)
        guard case .object(let error)? = listing.body.objectValue?["error"] else {
            Issue.record("expected an error object on a failed load")
            return
        }
        #expect(error["code"] == .string("model_load_failed"))
        #expect(error["message"] == .string("unsupported model type"))
    }

    @Test("even a status-blind poller cannot find the model before it is ready")
    func bodyAloneIsSufficient() throws {
        // Narrowing check: the gate must hold on the body, not only the status
        // code, because a client that ignores 503 still scans data[].
        for readiness in [RuntimeReadiness.loading, .shuttingDown, .failed("x")] {
            let listing = ModelsListing.make(
                modelID: Self.modelID, readiness: readiness, created: 1)
            let encoded = try JSONEncoding.string(listing.body)
            #expect(!encoded.contains(Self.modelID))
        }
    }

    @Test("the model path is not slash-escaped on the wire")
    func doesNotEscapeSlashes() throws {
        // The launcher compares the advertised ID against a configured absolute
        // path; `\/` escaping would break that comparison.
        let listing = ModelsListing.make(modelID: Self.modelID, readiness: .ready, created: 1)
        let encoded = try JSONEncoding.string(listing.body)
        #expect(encoded.contains(Self.modelID))
        #expect(!encoded.contains("\\/"))
    }
}
