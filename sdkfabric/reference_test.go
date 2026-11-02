package sdkfabric_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// This file has no build tag and imports only the standard library, so the
// package's static boundary is checked even where the gofabric-tagged binding
// is not compiled.

const fabricModule = "github.com/SukramJ/go-fabric/"

func parseSource(t *testing.T, name string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func buildTag(f *ast.File) string {
	for _, group := range f.Comments {
		if group.Pos() >= f.Package {
			break
		}
		for _, c := range group.List {
			if tag, ok := strings.CutPrefix(c.Text, "//go:build "); ok {
				return tag
			}
		}
	}
	return ""
}

func imports(t *testing.T, f *ast.File) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		out[name] = path
	}
	return out
}

// selectors returns every package-qualified identifier the file uses, as
// "import/path.Name".
func selectors(t *testing.T, f *ast.File) map[string]bool {
	t.Helper()
	names := imports(t, f)
	out := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				if path, ok := names[id.Name]; ok {
					out[path+"."+sel.Sel.Name] = true
				}
			}
		}
		return true
	})
	return out
}

// TestAA12StaticBoundary pins what the production binding may reach: only the
// tagged SDK packages it binds, no network, file system, test support or SDK
// transport package, and no SDK call that constructs a node or a network
// advertiser. Node construction is reachable only through the injected
// launcher.
func TestAA12StaticBoundary(t *testing.T) {
	f := parseSource(t, "reference.go")
	if got := buildTag(f); got != "gofabric" {
		t.Fatalf("reference.go build constraint %q, want gofabric", got)
	}
	allowed := map[string]bool{
		"context": true, "errors": true, "fmt": true, "strings": true, "sync": true, "time": true,
		"github.com/Project-Helianthus/helianthus-matter-binding/adoption": true,
		fabricModule + "cluster/spec/bridgeddevicebasicinformation":        true,
		fabricModule + "cluster/spec/descriptor":                           true,
		fabricModule + "cluster/spec/electricalpowermeasurement":           true,
		fabricModule + "cluster/spec/powertopology":                        true,
		fabricModule + "device":                                            true,
		fabricModule + "mdns":                                              true,
		fabricModule + "node":                                              true,
	}
	for _, path := range imports(t, f) {
		if !allowed[path] {
			t.Errorf("reference.go imports %q", path)
		}
	}
	used := selectors(t, f)
	for _, banned := range []string{
		fabricModule + "node.New", fabricModule + "node.Run",
		fabricModule + "mdns.NewZeroconf", fabricModule + "mdns.NewSubtypeResponder", fabricModule + "mdns.NewNoop",
	} {
		if used[banned] {
			t.Errorf("reference.go calls %s", banned)
		}
	}
	for _, required := range []string{
		fabricModule + "device.NewElectricalSensor",
		fabricModule + "device.ElectricalSensorRevision",
		fabricModule + "cluster/spec/powertopology.Revision",
		fabricModule + "cluster/spec/electricalpowermeasurement.Revision",
	} {
		if !used[required] {
			t.Errorf("reference.go does not bind %s", required)
		}
	}
}

// TestAA12StaticFixture pins the fixture leg to the in-memory site: the tagged
// test never asks fabrictest for loopback sockets and launches only through
// the site's AddNode route.
func TestAA12StaticFixture(t *testing.T) {
	f := parseSource(t, "fabrictest_test.go")
	if got := buildTag(f); got != "gofabric" {
		t.Fatalf("fabrictest_test.go build constraint %q, want gofabric", got)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if kv, ok := n.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Loopback" {
				t.Errorf("fabrictest_test.go sets Loopback")
			}
		}
		return true
	})
	used := selectors(t, f)
	if used[fabricModule+"node.New"] || used[fabricModule+"node.Run"] {
		t.Error("fabrictest_test.go constructs a node outside the site")
	}
	adds := 0
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "AddNode" || sel.Sel.Name == "AddDevice") {
			adds++
			if sel.Sel.Name != "AddNode" {
				t.Error("fabrictest_test.go launches through AddDevice")
			}
		}
		return true
	})
	if adds != 1 {
		t.Errorf("fabrictest_test.go has %d site launch calls, want 1", adds)
	}
}
