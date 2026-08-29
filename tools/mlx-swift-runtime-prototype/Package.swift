// swift-tools-version: 6.1

import PackageDescription

// Revisions are pinned exactly: this prototype exists to report what a specific
// official MLX Swift LM revision does with a specific local model. A floating
// requirement would make every recorded measurement unreproducible.
let package = Package(
    name: "mlx-swift-runtime-prototype",
    platforms: [
        .macOS(.v15)
    ],
    products: [
        .executable(
            name: "mlx-swift-runtime-prototype",
            targets: ["mlx-swift-runtime-prototype"]),
        .library(
            name: "MLXSwiftRuntimeContract",
            targets: ["MLXSwiftRuntimeContract"]),
    ],
    dependencies: [
        .package(url: "https://github.com/ml-explore/mlx-swift-lm", exact: "3.31.4"),
        .package(url: "https://github.com/ml-explore/mlx-swift", exact: "0.31.6"),
        .package(url: "https://github.com/huggingface/swift-transformers", exact: "1.3.3"),
        .package(url: "https://github.com/apple/swift-nio", exact: "2.99.0"),
    ],
    targets: [
        // Pure-Swift OpenAI-contract layer: no MLX, no model, no network.
        // Everything that gates, validates or shapes a response lives here so
        // it can be tested without a 29 GiB model load.
        .target(
            name: "MLXSwiftRuntimeContract",
            path: "Sources/MLXSwiftRuntimeContract"),
        .executableTarget(
            name: "mlx-swift-runtime-prototype",
            dependencies: [
                "MLXSwiftRuntimeContract",
                .product(name: "MLXLLM", package: "mlx-swift-lm"),
                .product(name: "MLXVLM", package: "mlx-swift-lm"),
                .product(name: "MLXLMCommon", package: "mlx-swift-lm"),
                .product(name: "MLXHuggingFace", package: "mlx-swift-lm"),
                .product(name: "MLX", package: "mlx-swift"),
                // For `Module.modules()`: the condemned-worker teardown registers
                // this model's whole module tree weakly so it can say whose weights
                // came back. MLX's allocator counters are process-global and cannot.
                .product(name: "MLXNN", package: "mlx-swift"),
                .product(name: "Tokenizers", package: "swift-transformers"),
                .product(name: "NIOCore", package: "swift-nio"),
                .product(name: "NIOPosix", package: "swift-nio"),
                .product(name: "NIOHTTP1", package: "swift-nio"),
            ],
            path: "Sources/mlx-swift-runtime-prototype"),
        .testTarget(
            name: "MLXSwiftRuntimeContractTests",
            dependencies: ["MLXSwiftRuntimeContract"],
            path: "Tests/MLXSwiftRuntimeContractTests"),
    ]
)
