package imports

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bearlytools/claw/internal/imports/git"
	memfs "github.com/gopherfs/fs/io/mem/simple"
	"github.com/kylelemons/godebug/pretty"
)

//go:embed testing/config/*
var importsFS embed.FS

func mustReadFile(p string) []byte {
	b, err := importsFS.ReadFile(p)
	if err != nil {
		panic(fmt.Sprintf("mustReadFile could not read %q: %s", p, err))
	}
	return b
}

type fakeVCSGit struct {
	root   string
	origin string
}

func (f fakeVCSGit) InRepo(pkgPath string) bool {
	pkgPath = path.Clean(pkgPath)
	return strings.HasPrefix(pkgPath, f.origin)
}

func (f fakeVCSGit) Root() string {
	return f.root
}

func (f fakeVCSGit) Origin() string {
	return f.origin
}

func (f fakeVCSGit) Abs(p string) (string, error) {
	p = path.Clean(p)

	// If p == "github.com/johnsiilver/claw/subdir/claw"
	// and g.Origin() == "github.com/johnsiilver/claw"
	// sp == ["/subdir/claw"]
	sp := strings.Split(p, f.origin)
	if len(sp) < 1 {
		return "", fmt.Errorf("git.Abs(%q) results in nothing(origin: %q", p, f.origin)
	}
	s := strings.Join(sp[1:], "/")
	s = strings.TrimPrefix(s, "/")
	// Now we have s == "subdir/claw"
	// g.Root() == "/usr/someone/trees/claw/"
	s = filepath.Join(f.root, s)
	var err error
	s, err = filepath.Abs(s)
	if err != nil {
		return "", fmt.Errorf("git.Abs(%q) error: %w", p, err)
	}

	// Nowe should have s == "/usr/someone/trees/claw/subdir/claw"
	return s, nil
}

type fakeGetClawFile struct {
	m map[string]git.ClawFile
}

func (f fakeGetClawFile) getClawFile(ctx context.Context, pkgPath string, version string) (git.ClawFile, error) {
	cf, ok := f.m[pkgPath]
	if !ok {
		return git.ClawFile{}, fmt.Errorf("%q not found", pkgPath)
	}
	if cf.Version != version {
		return git.ClawFile{}, fmt.Errorf("%q not found", pkgPath)
	}
	cf.SHA256 = fmt.Sprintf("%x", sha256.Sum256(cf.Content))
	return cf, nil
}

type fakeGetModuleFile struct {
	m map[string]git.ModuleFile
}

func (f fakeGetModuleFile) getModuleFile(ctx context.Context, pkgPath string, version string) (git.ModuleFile, error) {
	mf, ok := f.m[pkgPath]
	if !ok {
		return git.ModuleFile{Exists: false}, nil
	}
	return mf, nil
}

type pathContent struct {
	path    string
	content []byte
}

// TestConfig does a some basic tests of reading all our files and generating our data.
// It does a test of our cyclic import detection.
func TestConfig(t *testing.T) {
	tests := []struct {
		desc           string
		vcsGit         fakeVCSGit
		files          []pathContent
		getClawFile    func(ctx context.Context, pkgPath string, version string) (git.ClawFile, error)
		getModuleFile  func(ctx context.Context, pkgPath string, version string) (git.ModuleFile, error)
		err            bool
		errMsgContains string
	}{
		{
			desc: "Test recursion error",
			vcsGit: fakeVCSGit{
				root:   "/user/name/trees/bearlytools/claw",
				origin: "github.com/bearlytools/claw",
			},
			files: []pathContent{
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/claw.mod", mustReadFile("testing/config/vehicles.mod")},
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/vehicles.claw", mustReadFile("testing/config/vehicles.claw")},
				// This file which is depended on by vehicles imports vehicles.
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/manufacturers/claw.mod", mustReadFile("testing/config/manufacturers.mod")},
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/manufacturers/manufacturers.claw", mustReadFile("testing/config/recursive_manufacturers.claw")},
			},
			getClawFile: fakeGetClawFile{
				m: map[string]git.ClawFile{
					"github.com/bearlytools/test_claw_imports/trucks": {
						Content: mustReadFile("testing/config/trucks.claw"),
					},
					"github.com/bearlytools/test_claw_imports/cars/claw": {
						Content: mustReadFile("testing/config/cars.claw"),
					},
				},
			}.getClawFile,
			getModuleFile: fakeGetModuleFile{
				m: map[string]git.ModuleFile{
					"github.com/bearlytools/test_claw_imports/trucks": {
						Content: mustReadFile("testing/config/trucks.mod"),
						Exists:  true,
					},
					"github.com/bearlytools/test_claw_imports/cars/claw": {
						Content: mustReadFile("testing/config/cars.mod"),
						Exists:  true,
					},
				},
			}.getModuleFile,
			errMsgContains: "cyclic import detected",
			err:            true,
		},
		{
			desc: "Test success",
			vcsGit: fakeVCSGit{
				root:   "/user/name/trees/bearlytools/claw",
				origin: "github.com/bearlytools/claw",
			},
			files: []pathContent{
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/claw.mod", mustReadFile("testing/config/vehicles.mod")},
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/vehicles.claw", mustReadFile("testing/config/vehicles.claw")},
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/manufacturers/claw.mod", mustReadFile("testing/config/manufacturers.mod")},
				{"/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/manufacturers/manufacturers.claw", mustReadFile("testing/config/manufacturers.claw")},
			},
			getClawFile: fakeGetClawFile{
				m: map[string]git.ClawFile{
					"github.com/bearlytools/test_claw_imports/trucks": {
						Content: mustReadFile("testing/config/trucks.claw"),
					},
					"github.com/bearlytools/test_claw_imports/cars/claw": {
						Content: mustReadFile("testing/config/cars.claw"),
					},
				},
			}.getClawFile,
			getModuleFile: fakeGetModuleFile{
				m: map[string]git.ModuleFile{
					"github.com/bearlytools/test_claw_imports/trucks": {
						Content: mustReadFile("testing/config/trucks.mod"),
						Exists:  true,
					},
					"github.com/bearlytools/test_claw_imports/cars/claw": {
						Content: mustReadFile("testing/config/cars.mod"),
						Exists:  true,
					},
				},
			}.getModuleFile,
		},
	}

	for _, test := range tests {
		testFS := memfs.New()
		for _, f := range test.files {
			if err := testFS.WriteFile(f.path, f.content, 0600); err != nil {
				panic(err)
			}
		}

		config := NewConfig()
		config.fs = testFS
		config.getClawFile = test.getClawFile
		config.getModuleFile = test.getModuleFile
		config.git = test.vcsGit

		err := config.Read(context.Background(), "/user/name/trees/bearlytools/claw/testing/imports/vehicles/claw/vehicles.claw")
		switch {
		case err == nil && test.err:
			t.Errorf("TestConfig(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestConfig(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			if test.errMsgContains != "" {
				if !strings.Contains(err.Error(), test.errMsgContains) {
					t.Errorf("TestConfig(%s): expected error message to contain %q, but was %q", test.desc, test.errMsgContains, err.Error())
				}
			}
			continue
		}

		pconfig := pretty.Config{IncludeUnexported: false, TrackCycles: true}
		if diff := pconfig.Compare(wantConfig, config); diff != "" {
			t.Errorf("TestConfig(%s): -want/+got:\n%s", test.desc, diff)
		}
		//litter.Config.HidePrivateFields = true
		//litter.Config.DisablePointerReplacement = true
		//litter.Dump(config)
	}
}
