package infra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The guards in this file answer three questions that a passing launch test
// cannot: can the agents-infra observation adapter reach a live runtime, can
// the production consumer classify a Pi turn without the sole upstream
// classifier, and does either plane branch on a Pi/Qwen/MLX/model identity.
//
// Each guard discovers its plane instead of trusting a fixed filename list, so
// a helper moved into a new same-package file is scanned rather than skipped.
// The exact bypass shape covered is the one that a filename list admits: a
// production declaration in a *different* file, reached from the plane's entry
// declaration by a package-level call or by a method on the plane's own type.

const (
	observerAssemblyFile = "agents_management_registry.go"
	observerPlaneFile    = "agents_management_observer.go"
	consumerPlaneFile    = "agents_management_process_a.go"
)

// observationPlaneSeeds are the exact declarations agents-management calls
// through vendorplugin.EngineObservationAdapter. Everything they can reach in
// this package is part of the observation plane.
var observationPlaneSeeds = []string{
	"NewSanitizedEngineObservationAdapter",
	"ObserveEngine",
	"EngineObservationAdapterDeclaration",
}

// consumerPlaneSeeds are the exact declarations that build the Pi launch, run
// Process A, and hand the capture to the sole classifier.
var consumerPlaneSeeds = []string{
	"BuildAndRunPiTurn",
	"RunProcessATurn",
	"waitForProcessA",
	"stopProcessA",
	"processAExit",
	"configureProcessACommand",
}

type packageIndex struct {
	dir      string
	fset     *token.FileSet
	files    map[string]*ast.File
	funcs    map[string][]string // bare function name -> files declaring it
	methods  map[string][]string // "Recv.Method" -> files declaring it
	byRecv   map[string][]string // receiver type -> method names
	packages map[string]bool     // imported package aliases seen anywhere
}

// indexProductionPackage parses every non-test Go source in the package,
// including files behind a foreign build constraint, so a bypass cannot hide on
// another platform.
func indexProductionPackage(t *testing.T, dir string) *packageIndex {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	index := &packageIndex{
		dir:      dir,
		fset:     token.NewFileSet(),
		files:    map[string]*ast.File{},
		funcs:    map[string][]string{},
		methods:  map[string][]string{},
		byRecv:   map[string][]string{},
		packages: map[string]bool{},
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(index.fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		index.files[name] = file
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if function.Recv == nil || len(function.Recv.List) == 0 {
				index.funcs[function.Name.Name] = append(index.funcs[function.Name.Name], name)
				continue
			}
			recv := receiverTypeName(function.Recv.List[0].Type)
			index.methods[recv+"."+function.Name.Name] = append(index.methods[recv+"."+function.Name.Name], name)
			index.byRecv[recv] = append(index.byRecv[recv], function.Name.Name)
		}
	}
	if len(index.files) == 0 {
		t.Fatal("plane discovery found no production sources; the guard would pass vacuously")
	}
	return index
}

func receiverTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(typed.X)
	case *ast.Ident:
		return typed.Name
	case *ast.IndexExpr:
		return receiverTypeName(typed.X)
	}
	return ""
}

// plane returns every production file reachable from the seed declarations.
// Reachability follows package-level calls by bare identifier and methods
// declared on any receiver type the plane already owns, which is exactly how a
// "called helper in another file" bypass has to be written.
func (index *packageIndex) plane(t *testing.T, seeds []string, ownedReceivers []string) []string {
	t.Helper()
	reached := map[string]bool{}
	files := map[string]bool{}
	receivers := map[string]bool{}
	for _, recv := range ownedReceivers {
		receivers[recv] = true
	}

	var visitDecl func(decl *ast.FuncDecl, file string)
	queue := []*ast.FuncDecl{}
	enqueue := func(name string) {
		for _, file := range index.funcs[name] {
			key := "func:" + name + ":" + file
			if reached[key] {
				continue
			}
			reached[key] = true
			files[file] = true
			queue = append(queue, index.declIn(file, "", name))
		}
		for recv := range receivers {
			for _, file := range index.methods[recv+"."+name] {
				key := "method:" + recv + "." + name + ":" + file
				if reached[key] {
					continue
				}
				reached[key] = true
				files[file] = true
				queue = append(queue, index.declIn(file, recv, name))
			}
		}
	}
	visitDecl = func(decl *ast.FuncDecl, _ string) {
		if decl == nil || decl.Body == nil {
			return
		}
		ast.Inspect(decl.Body, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.Ident:
				if len(index.funcs[typed.Name]) > 0 {
					enqueue(typed.Name)
				}
			case *ast.SelectorExpr:
				for recv := range receivers {
					if len(index.methods[recv+"."+typed.Sel.Name]) > 0 {
						enqueue(typed.Sel.Name)
					}
				}
			}
			return true
		})
	}

	for _, seed := range seeds {
		enqueue(seed)
	}
	if len(files) == 0 {
		t.Fatalf("plane seeds %v matched no production declaration", seeds)
	}
	for len(queue) > 0 {
		decl := queue[0]
		queue = queue[1:]
		visitDecl(decl, "")
	}

	discovered := make([]string, 0, len(files))
	for file := range files {
		discovered = append(discovered, file)
	}
	sort.Strings(discovered)
	return discovered
}

func (index *packageIndex) declIn(file, recv, name string) *ast.FuncDecl {
	parsed := index.files[file]
	if parsed == nil {
		return nil
	}
	for _, decl := range parsed.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Name.Name != name {
			continue
		}
		if recv == "" {
			if function.Recv == nil {
				return function
			}
			continue
		}
		if function.Recv != nil && len(function.Recv.List) > 0 && receiverTypeName(function.Recv.List[0].Type) == recv {
			return function
		}
	}
	return nil
}

func (index *packageIndex) imports(file string) []string {
	parsed := index.files[file]
	if parsed == nil {
		return nil
	}
	paths := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

// Production call site: vendorplugin.BuildLaunch calls
// (*SanitizedEngineObservationAdapter).ObserveEngine, which may only ask the injected
// SanitizedEngineObservationReader. Nothing it reaches in this package may open
// a process, socket, environment, or user configuration itself.
func TestObservationPlaneCannotReachLiveRuntime(t *testing.T) {
	index := indexProductionPackage(t, ".")
	plane := index.plane(t, observationPlaneSeeds, []string{"SanitizedEngineObservationAdapter"})
	if !containsFile(plane, observerPlaneFile) {
		t.Fatalf("plane %v lost its own entry file; discovery is not fail-closed", plane)
	}
	forbidden := []string{"os", "os/exec", "os/user", "net", "net/http", "net/url", "syscall", "golang.org/x/sys/unix", "os/signal"}
	for _, file := range plane {
		for _, path := range index.imports(file) {
			for _, denied := range forbidden {
				if path == denied {
					t.Fatalf("observation plane file %s imports %s; a live read cannot originate inside the adapter", file, denied)
				}
			}
		}
	}
}

// Production call site: BuildAndRunPiTurn. Everything it reaches must hand the
// bounded capture to pi.ValidateTurnResult exactly once and must own no
// schema-1 parser of its own.
func TestConsumerPlaneCannotParseAroundSoleClassifier(t *testing.T) {
	index := indexProductionPackage(t, ".")
	plane := index.plane(t, consumerPlaneSeeds, []string{"OSProcessATurnRunner", "boundedProcessAStdout"})
	if !containsFile(plane, consumerPlaneFile) {
		t.Fatalf("plane %v lost its own entry file; discovery is not fail-closed", plane)
	}
	calls := 0
	for _, file := range plane {
		source, err := os.ReadFile(filepath.Join(".", file))
		if err != nil {
			t.Fatal(err)
		}
		text := string(source)
		calls += strings.Count(text, "managementpi.ValidateTurnResult(")
		for _, forbidden := range []string{"encoding/json", "json.Unmarshal", "json.NewDecoder", "json.Decoder"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("consumer plane file %s contains permissive schema-1 parser %q", file, forbidden)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("consumer plane calls the sole classifier %d times, want exactly 1 (plane=%v)", calls, plane)
	}
}

// AC6. Neither plane may condition on a Pi, Qwen, MLX, vendor, or model
// identity. Dispatch is carried by the registry declaration data assembled in
// agents_management_registry.go, never by a literal in the launch path.
func TestGenericPlanesContainNoIdentityBranch(t *testing.T) {
	index := indexProductionPackage(t, ".")
	planes := map[string][]string{
		"observation": index.plane(t, observationPlaneSeeds, []string{"SanitizedEngineObservationAdapter"}),
		"consumer":    index.plane(t, consumerPlaneSeeds, []string{"OSProcessATurnRunner", "boundedProcessAStdout"}),
	}
	forbidden := []string{"qwen", "mlx", "alibaba", "llama", "gguf", "safetensors", "local-models", "qwen-infra"}
	for name, plane := range planes {
		for _, file := range plane {
			if file == observerAssemblyFile {
				t.Fatalf("%s plane reached the concrete assembly file; identity data must not be on the launch path", name)
			}
			source, err := os.ReadFile(filepath.Join(".", file))
			if err != nil {
				t.Fatal(err)
			}
			lowered := strings.ToLower(string(source))
			for _, literal := range forbidden {
				if strings.Contains(lowered, literal) {
					t.Fatalf("%s plane file %s mentions identity literal %q", name, file, literal)
				}
			}
		}
	}
}

// The concrete assembly is allowed to *declare* and to *refuse* a Pi/MLX
// identity. It is not allowed to *dispatch* on one: a conditional that selects
// between two launch behaviours by identity literal is the shape a second
// dispatch authority takes, so only a plain fail-closed refusal is admitted.
func TestConcreteAssemblyDeclaresIdentityWithoutDispatchingOnIt(t *testing.T) {
	index := indexProductionPackage(t, ".")
	file := index.files[observerAssemblyFile]
	if file == nil {
		t.Fatalf("%s is missing; the assembly guard would pass vacuously", observerAssemblyFile)
	}
	inspected := 0
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SwitchStmt:
			if typed.Tag != nil && conditionHasStringLiteral(typed.Tag) {
				t.Errorf("%s switches on an identity literal at %s", observerAssemblyFile, index.fset.Position(typed.Pos()))
			}
			for _, clause := range typed.Body.List {
				caseClause, ok := clause.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, expr := range caseClause.List {
					if conditionHasStringLiteral(expr) {
						t.Errorf("%s dispatches on identity case at %s", observerAssemblyFile, index.fset.Position(expr.Pos()))
					}
				}
			}
		case *ast.IfStmt:
			if !conditionHasStringLiteral(typed.Cond) {
				return true
			}
			inspected++
			if typed.Else != nil || !isPlainRefusal(typed.Body) {
				t.Errorf("%s selects behaviour on an identity literal at %s; only a plain fail-closed refusal is admitted",
					observerAssemblyFile, index.fset.Position(typed.Pos()))
			}
		}
		return true
	})
	if inspected == 0 {
		t.Fatalf("%s declared no identity refusal; the assembly guard would pass vacuously", observerAssemblyFile)
	}
}

func conditionHasStringLiteral(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if ok && literal.Kind == token.STRING && literal.Value != `""` {
			found = true
		}
		return !found
	})
	return found
}

// isPlainRefusal reports whether a block is exactly one return statement, which
// is the only body an identity conditional may carry in the assembly.
func isPlainRefusal(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}
	_, ok := body.List[0].(*ast.ReturnStmt)
	return ok
}

// AC4. Process-B election, lease, restart, quarantine, and rotation stay owned
// by agents-infra, and the generic observation/consumer planes may not reach
// any of them: the adapter reads sanitized values and Process-A cleanup signals
// only its own process group.
func TestProcessBLifecycleStaysOwnedByAgentsInfraAndOffTheGenericPlanes(t *testing.T) {
	index := indexProductionPackage(t, ".")
	owned := []string{"RunSharedRuntimeBroker", "SetSharedRuntimeManualQuarantine", "SharedRuntimeStatusReport"}
	for _, name := range owned {
		if len(index.funcs[name]) == 0 {
			t.Fatalf("Process-B lifecycle entry point %s left agents-infra; ownership moved or the guard is stale", name)
		}
	}
	lifecycle := map[string]bool{}
	for name := range index.funcs {
		lowered := strings.ToLower(name)
		for _, marker := range []string{"sharedruntime", "broker", "quarantine", "lease", "rotat"} {
			if strings.Contains(lowered, marker) {
				lifecycle[name] = true
			}
		}
	}
	if len(lifecycle) == 0 {
		t.Fatal("no Process-B lifecycle declaration discovered; the ownership guard would pass vacuously")
	}
	planes := map[string][]string{
		"observation": index.plane(t, observationPlaneSeeds, []string{"SanitizedEngineObservationAdapter"}),
		"consumer":    index.plane(t, consumerPlaneSeeds, []string{"OSProcessATurnRunner", "boundedProcessAStdout"}),
	}
	for name, plane := range planes {
		for _, file := range plane {
			source, err := os.ReadFile(filepath.Join(".", file))
			if err != nil {
				t.Fatal(err)
			}
			for symbol := range lifecycle {
				if strings.Contains(string(source), symbol) {
					t.Fatalf("%s plane file %s reaches Process-B lifecycle symbol %s; the generic plane must not become the broker", name, file, symbol)
				}
			}
		}
	}
}

// The acceptance evidence for this contract is static and fake by
// construction. This guard discovers the contract's own test sources instead of
// listing them, so a case added later cannot quietly reach a live runtime,
// model, socket, network endpoint, broker, or the user's real configuration.
func TestContractTestsContactNoLiveRuntime(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	discovered := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}
		if !strings.HasPrefix(name, "agents_management_") && !strings.HasPrefix(name, "pi_turn_result") {
			continue
		}
		discovered++
		// Compare parsed imports and identifiers, not raw text: a substring
		// scan would match this guard's own denial list.
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		forbiddenImports := map[string]bool{"net": true, "net/http": true, "net/url": true, "os/user": true, "os/signal": true}
		for _, path := range (&packageIndex{files: map[string]*ast.File{name: parsed}}).imports(name) {
			if forbiddenImports[path] {
				t.Fatalf("contract test %s imports %s; the acceptance evidence must stay static and fake", name, path)
			}
		}
		forbiddenSymbols := map[string]bool{
			"RunSharedRuntimeBroker": true, "StopSharedRuntime": true,
			"SetSharedRuntimeManualQuarantine": true, "SharedRuntimeStatusReport": true,
			"UserHomeDir": true, "UserCacheDir": true, "Dial": true,
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if ok && forbiddenSymbols[ident.Name] {
				t.Errorf("contract test %s reaches live state through %s at %s", name, ident.Name, fset.Position(ident.Pos()))
			}
			return true
		})
	}
	if discovered < 3 {
		t.Fatalf("no-live discovery found only %d contract test files; the guard would pass vacuously", discovered)
	}
}

func containsFile(files []string, want string) bool {
	for _, file := range files {
		if file == want {
			return true
		}
	}
	return false
}
