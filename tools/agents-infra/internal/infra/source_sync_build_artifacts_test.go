package infra

import "testing"

// A source tree that contains a SwiftPM package also contains that package's
// `.build` directory once anyone has compiled it. Those products are
// machine-local, can be hundreds of megabytes, and some bundle resources are
// written mode 0400, so copying them into an install root fails with
// "permission denied" and takes the whole `setup local` run down with it.
func TestShouldSkipExcludesSwiftPMBuildProducts(t *testing.T) {
	skipped := []struct {
		rel   string
		isDir bool
	}{
		{"tools/mlx-swift-runtime-prototype/.build", true},
		{"tools/mlx-swift-runtime-prototype/.build/release/binary", false},
		{
			"tools/mlx-swift-runtime-prototype/.build/arm64-apple-macosx/release/" +
				"swift-crypto_Crypto.bundle/PrivacyInfo.xcprivacy",
			false,
		},
		{".build", true},
		{".build/checkouts/mlx-swift/Package.swift", false},
	}
	for _, entry := range skipped {
		if !shouldSkip(entry.rel, entry.isDir) {
			t.Fatalf("shouldSkip(%q, %v) = false, want true", entry.rel, entry.isDir)
		}
	}
}

// The exclusion must match the `.build` path component exactly. A substring or
// prefix match would also swallow real source directories, and a package's
// `Sources`/`Tests` trees must still install.
func TestShouldSkipKeepsSwiftPackageSources(t *testing.T) {
	kept := []struct {
		rel   string
		isDir bool
	}{
		{"tools/mlx-swift-runtime-prototype/Package.swift", false},
		{"tools/mlx-swift-runtime-prototype/Sources/Contract/Gate.swift", false},
		{"tools/mlx-swift-runtime-prototype/Tests/ContractTests/GateTests.swift", false},
		{"tools/mlx-swift-runtime-prototype/scripts/smoke.sh", false},
		{"tools/build", true},
		{"tools/build/main.go", false},
		{"tools/.buildkite/pipeline.yml", false},
		{"tools/prebuild/.build-notes.md", false},
	}
	for _, entry := range kept {
		if shouldSkip(entry.rel, entry.isDir) {
			t.Fatalf("shouldSkip(%q, %v) = true, want false", entry.rel, entry.isDir)
		}
	}
}

// The same package cannot be built by SwiftPM at all: mlx-swift's Metal shaders
// require `xcodebuild`, which writes a multi-gigabyte `DerivedData` tree into
// the source directory. It is the same hazard as `.build` and must be excluded
// for the same reason.
func TestShouldSkipExcludesXcodeDerivedData(t *testing.T) {
	skipped := []struct {
		rel   string
		isDir bool
	}{
		{"tools/mlx-swift-runtime-prototype/DerivedData", true},
		{
			"tools/mlx-swift-runtime-prototype/DerivedData/Build/Products/Release/" +
				"mlx-swift-runtime-prototype",
			false,
		},
		{
			"tools/mlx-swift-runtime-prototype/DerivedData/Build/Products/Release/" +
				"mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib",
			false,
		},
		{"tools/mlx-swift-runtime-prototype/DerivedData/SourcePackages/checkouts/mlx-swift/Package.swift", false},
		{"DerivedData", true},
	}
	for _, entry := range skipped {
		if !shouldSkip(entry.rel, entry.isDir) {
			t.Fatalf("shouldSkip(%q, %v) = false, want true", entry.rel, entry.isDir)
		}
	}
}

// The DerivedData exclusion must match a whole path component. A substring or
// prefix match would swallow real source paths that merely start with or
// contain the same letters.
func TestShouldSkipKeepsPathsNamedLikeDerivedData(t *testing.T) {
	kept := []struct {
		rel   string
		isDir bool
	}{
		{"tools/agents-infra/internal/infra/DerivedDataPolicy.go", false},
		{"tools/agents-infra/internal/DerivedDataNotes/readme.md", false},
		{"docs/derivedData/notes.md", false},
		{"tools/mlx-swift-runtime-prototype/Sources/DerivedDataCheck.swift", false},
	}
	for _, entry := range kept {
		if shouldSkip(entry.rel, entry.isDir) {
			t.Fatalf("shouldSkip(%q, %v) = true, want false", entry.rel, entry.isDir)
		}
	}
}
