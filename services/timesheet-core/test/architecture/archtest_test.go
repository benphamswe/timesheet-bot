// Package architecture_test enforces the DDD and Clean Architecture dependency
// rules (docs/adr/0001-ddd-package-by-bounded-context.md) as tests, so a
// violation fails the build instead of waiting for review. It scans the
// production packages under internal/ and cmd/ only; test/ is never scanned.
package architecture_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const module = "timesheet-bot"

// moduleRoot is services/timesheet-core, relative to this package directory
// (test/architecture), which is where go test runs it.
const moduleRoot = "../.."

// scannedDirs are the production source trees, relative to moduleRoot.
var scannedDirs = []string{"internal", "cmd"}

// rule forbids packages whose directory matches from importing paths matching any of deny.
// allow, if set, is an allowlist: every non-stdlib import must match it.
type rule struct {
	name  string
	from  *regexp.Regexp
	allow *regexp.Regexp
	deny  []*regexp.Regexp
}

var rules = []rule{
	{
		name:  "domain and kernel import only the stdlib, the kernel and their own context",
		from:  regexp.MustCompile(`^internal/(shared/kernel|[a-z]+/domain)$`),
		allow: regexp.MustCompile(`^` + module + `/internal/shared/kernel$`),
		deny:  []*regexp.Regexp{regexp.MustCompile(`^(net(/|$)|database/|os/exec$)`)},
	},
	{
		name: "application layers do not depend on infrastructure, interfaces, platform or cmd",
		from: regexp.MustCompile(`^internal/[a-z]+/application$`),
		deny: []*regexp.Regexp{
			regexp.MustCompile(`/infrastructure(/|$)`),
			regexp.MustCompile(`^` + module + `/internal/(interfaces|platform)(/|$)`),
			regexp.MustCompile(`^` + module + `/cmd(/|$)`),
		},
	},
	{
		name: "infrastructure does not depend on interfaces or cmd",
		from: regexp.MustCompile(`^internal/[a-z]+/infrastructure(/.*)?$`),
		deny: []*regexp.Regexp{
			regexp.MustCompile(`^` + module + `/internal/interfaces(/|$)`),
			regexp.MustCompile(`^` + module + `/cmd(/|$)`),
		},
	},
	{
		name: "interfaces never build adapters; only cmd/timesheet wires infrastructure",
		from: regexp.MustCompile(`^internal/interfaces(/.*)?$`),
		deny: []*regexp.Regexp{regexp.MustCompile(`/infrastructure(/|$)`)},
	},
	{
		name: "production code never imports the test tree",
		from: regexp.MustCompile(`^(internal|cmd)(/.*)?$`),
		deny: []*regexp.Regexp{regexp.MustCompile(`^` + module + `/test(/|$)`)},
	},
}

// ownContext lets a domain package import packages of its own bounded context.
func ownContext(dir, imp string) bool {
	parts := strings.Split(dir, "/") // internal/<ctx>/domain
	return len(parts) >= 2 && strings.HasPrefix(imp, module+"/internal/"+parts[1]+"/domain")
}

func isStdlib(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(first, ".") && first != module
}

// imports returns the non-test imports per package directory under scannedDirs,
// relative to the module root.
func imports(t *testing.T) map[string][]string {
	t.Helper()
	root, err := filepath.Abs(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	byDir := map[string][]string{}
	fset := token.NewFileSet()
	for _, dir := range scannedDirs {
		if err := walkImports(fset, root, dir, byDir); err != nil {
			t.Fatal(err)
		}
	}
	return byDir
}

// walkImports adds the non-test imports of every package under root/dir to byDir.
func walkImports(fset *token.FileSet, root, dir string, byDir map[string][]string) error {
	return filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "testdata" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for _, spec := range f.Imports {
			p, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("%s: import %s: %w", path, spec.Path.Value, err)
			}
			byDir[rel] = append(byDir[rel], p)
		}
		return nil
	})
}

func TestDependencyRules(t *testing.T) {
	t.Parallel()
	byDir := imports(t)
	for _, r := range rules {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			checked := 0
			for dir, imps := range byDir {
				if !r.from.MatchString(dir) {
					continue
				}
				checked++
				for _, imp := range imps {
					if r.allow != nil && !isStdlib(imp) && !r.allow.MatchString(imp) && !ownContext(dir, imp) {
						t.Errorf("%s imports %s, which is not allowed", dir, imp)
					}
					for _, d := range r.deny {
						if d.MatchString(imp) {
							t.Errorf("%s imports %s", dir, imp)
						}
					}
				}
			}
			// A rule that matches no package passes vacuously; fail instead, so a
			// renamed directory or a broken pattern cannot switch the rule off.
			if checked == 0 {
				t.Errorf("rule %q matched no package (from %s); the walker or the pattern is broken", r.name, r.from)
			}
		})
	}
}

var (
	wallClockCall = regexp.MustCompile(`\btime\.Now\(\)`)
	clockFreeDir  = regexp.MustCompile(`^internal/[a-z]+/(domain|application)/`)
)

func TestDomainHasNoWallClock(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(moduleRoot)
	if err != nil {
		t.Fatalf("module root: %v", err)
	}
	checked := 0
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !clockFreeDir.MatchString(rel) {
			return nil
		}
		checked++
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if wallClockCall.Match(src) {
			t.Errorf("%s calls time.Now(); take a kernel.Clock or a now parameter", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if checked == 0 {
		t.Errorf("no domain or application files found under %s; the walker or the pattern is broken", root)
	}
}
