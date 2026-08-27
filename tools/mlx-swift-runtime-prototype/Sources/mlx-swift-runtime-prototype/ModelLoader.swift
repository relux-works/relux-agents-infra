import Foundation
import MLX
import MLXHuggingFace
import MLXLLM
import MLXLMCommon
import MLXNN
import MLXSwiftRuntimeContract
import MLXVLM
import Tokenizers

struct LoadedModel: @unchecked Sendable {
    let container: ModelContainer
    let modelType: String
    let factory: String
    let loadSeconds: Double

    /// What these weights cost MLX, measured as the `activeMemory` delta across
    /// the load that produced them.
    ///
    /// Measured here and nowhere else, because here is the only place both
    /// sides of the delta exist. It is what makes a condemned worker's teardown
    /// checkable without a threshold somebody picked: the release is observed
    /// when MLX has stopped calling at least this many bytes active. See
    /// ``GenerationBatchRecovery/weightsReleased(_:)``.
    ///
    /// Can legitimately read `0` — a load that never reached the GPU, or an MLX
    /// build whose counters stay flat. That is a failure to read, not a model
    /// that cost nothing, and the release gate refuses to attest from it rather
    /// than treating every later reading as satisfying it.
    let weightFootprintBytes: Int

    /// Every `Module` in this model's tree, held weakly.
    ///
    /// Populated here because here is the only place the tree is reachable
    /// before the engine takes ownership, and because a registry populated
    /// anywhere else would be a registry the teardown could find empty. It is
    /// what lets the condemned-worker teardown say something about *this
    /// model's* weights instead of about the process's byte total: MLX's
    /// allocator counters are process-global and cannot attribute a byte, and
    /// review defeated a gate built on them alone.
    ///
    /// See ``GenerationBatchRecovery/WeightReleaseObservation/liveWeightOwners``.
    let weightOwners: WeightOwnerRegistry
}

enum ModelLoadError: Error, CustomStringConvertible {
    case unreadableConfig(path: String, reason: String)
    case missingModelType(path: String)
    case noFactoryAccepted(modelType: String, failures: [String])

    var description: String {
        switch self {
        case .unreadableConfig(let path, let reason):
            return "could not read \(path): \(reason)"
        case .missingModelType(let path):
            return "\(path) has no \"model_type\" field"
        case .noFactoryAccepted(let modelType, let failures):
            return
                "no MLX Swift LM factory could load model_type \(modelType.debugDescription): "
                + failures.joined(separator: " | ")
        }
    }
}

enum ModelLoader {
    /// Read `model_type` straight from the model directory.
    ///
    /// Reported separately from the load result so a load failure can still name
    /// the architecture that was refused — that name is the whole point of the
    /// unsupported-architecture gap list.
    static func modelType(atPath path: String) throws -> String {
        let configURL = URL(fileURLWithPath: path).appendingPathComponent("config.json")
        let data: Data
        do {
            data = try Data(contentsOf: configURL)
        } catch {
            throw ModelLoadError.unreadableConfig(
                path: configURL.path, reason: String(describing: error))
        }
        let root = try? JSONDecoder().decode(MLXSwiftRuntimeContract.JSONValue.self, from: data)
        guard case .string(let modelType)? = root?.objectValue?["model_type"] else {
            throw ModelLoadError.missingModelType(path: configURL.path)
        }
        return modelType
    }

    /// Load the model from a local directory.
    ///
    /// The factories are referenced directly rather than through
    /// `NSClassFromString` trampolines: a statically linked executable that
    /// never names `VLMModelFactory` can have the trampoline class stripped, and
    /// the resulting "no factory available" error would look like an
    /// unsupported architecture instead of a link problem.
    ///
    /// The *order* is not `ModelFactoryRegistry`'s. It comes from
    /// ``RuntimeOptions/modelFactory`` and defaults to text-only, because for
    /// `model_type` `qwen3_5` both factories accept the directory and only the
    /// text one evaluates a long prompt in chunks. Taking the registry's
    /// vision-first order here would silently decide the runtime's long-context
    /// behaviour from a default nothing in this executable's HTTP contract asks
    /// for.
    ///
    /// No `Downloader` is linked anywhere in this executable, so a rejected or
    /// incomplete local directory cannot silently become a Hub download of a
    /// different model.
    static func load(
        path: String,
        preference: RuntimeOptions.ModelFactoryPreference = .textOnly
    ) async throws -> LoadedModel {
        let modelType = try modelType(atPath: path)
        let directory = URL(fileURLWithPath: path)
        let tokenizerLoader = #huggingFaceTokenizerLoader()

        let loaders: [String: () async throws -> ModelContainer] = [
            "MLXVLM.VLMModelFactory": {
                try await VLMModelFactory.shared.loadContainer(
                    from: directory, using: tokenizerLoader)
            },
            "MLXLLM.LLMModelFactory": {
                try await LLMModelFactory.shared.loadContainer(
                    from: directory, using: tokenizerLoader)
            },
        ]
        let candidates: [(name: String, load: () async throws -> ModelContainer)] =
            preference.factoryOrder.compactMap { name in
                loaders[name].map { (name, $0) }
            }

        var failures: [String] = []
        for candidate in candidates {
            let started = DispatchTime.now()
            // Taken per candidate rather than once up front: a factory that
            // rejects the directory may still have allocated and released
            // buffers before refusing, and folding that into the accepted
            // candidate's delta would overstate the weights the teardown then
            // has to wait for.
            let activeBefore = Memory.snapshot().activeMemory
            do {
                let container = try await candidate.load()
                let elapsed =
                    Double(DispatchTime.now().uptimeNanoseconds - started.uptimeNanoseconds) / 1e9
                // Closed immediately after the load and before anything else
                // runs, so nothing between the two readings can be charged to
                // the weights.
                let footprint = max(0, Memory.snapshot().activeMemory - activeBefore)
                // The whole module tree, root included. `modules()` flattens it
                // for us, and every weight array in this model is a stored
                // property of one of these objects -- which is precisely why
                // "all of them are gone" is a claim about this model rather
                // than about the process.
                let owners = WeightOwnerRegistry()
                await container.perform { context in
                    owners.register(context.model.modules())
                }
                return LoadedModel(
                    container: container, modelType: modelType, factory: candidate.name,
                    loadSeconds: elapsed,
                    weightFootprintBytes: footprint,
                    weightOwners: owners)
            } catch {
                failures.append("\(candidate.name): \(error)")
                StandardOutput.shared.log(
                    "\(candidate.name) rejected model_type \(modelType): \(error)")
            }
        }
        throw ModelLoadError.noFactoryAccepted(modelType: modelType, failures: failures)
    }
}
