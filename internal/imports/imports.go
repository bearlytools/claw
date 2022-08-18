// Package imports provides handling of claw.mod, local.replace and global.replace files.
package imports

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/internal/idl"
	"github.com/bearlytools/claw/internal/imports/git"
	"github.com/bearlytools/claw/internal/vcs"

	osfs "github.com/gopherfs/fs/io/os"

	"github.com/johnsiilver/halfpike"
)

// ImportFlow is a list of imports that have been imported in this import path.
// It is used when going down the tree of imports to make sure we do not see ourselves
// already in the path, which would indicate a cyclic import.
type ImportFlow []string

func (i ImportFlow) String() string {
	b := strings.Builder{}
	b.WriteString("cyclic import detected:\n")
	for x, imp := range i {
		if x == 0 {
			b.WriteString(fmt.Sprintf("\t%s", imp))
			continue
		}
		b.WriteString(fmt.Sprintf("-->\n\t%s", imp))
	}
	return b.String()
}

type importKey string

var impKey = importKey("clawImports")

// ExtractImports extracts a list of packages that have been imported in this import
// chain from the Context object. A nil return indicates you are at the first file.
func ExtractImports(ctx context.Context) ImportFlow {
	a := ctx.Value(impKey)
	if a == nil {
		return nil
	}
	return a.(ImportFlow)
}

// AppendImports extracts the ImportFlow in the Context and appends pkgPath to it
// and returns the new Context.
func AppendImports(ctx context.Context, pkgPath string) context.Context {
	l := ExtractImports(ctx)
	l = append(l, pkgPath)
	return context.WithValue(ctx, impKey, l)
}

type vcsGit interface {
	InRepo(pkgPath string) bool
	Root() string
	Origin() string
	Abs(p string) (string, error)
}

type neededFS interface {
	fs.ReadFileFS
	fs.ReadDirFS
	fs.StatFS
}

// Config just holds the overall union of claw.mod, local.replace and global.replace.
type Config struct {
	Root *idl.File
	// Imports is a mapping of all package paths to their IDL. If the idl.File == nil but
	// there is a key, then there is a replacement, check the LocalReplace.
	Imports       map[string]*idl.File
	Module        *Module
	LocalReplace  LocalReplace
	GlobalReplace map[string]Replace

	fs          neededFS
	git         vcsGit
	getClawFile func(ctx context.Context, pkgPath string, version string) (git.ClawFile, error)
}

// NewConfig creates a new Config.
func NewConfig() *Config {
	fs, err := osfs.New()
	if err != nil {
		panic("can't access OS: " + err.Error())
	}

	return &Config{
		Imports:       map[string]*idl.File{},
		GlobalReplace: map[string]Replace{},
		fs:            fs,
		getClawFile:   git.GetClawFile,
	}
}

// InRootRepo will determine if pkgPath is in the root file's repo.
func (c *Config) InRootRepo(pkgPath string) bool {
	if c.git == nil {
		panic("need git support for this")
	}
	return c.git.InRepo(pkgPath)
}

// RootDir returns the root directory for the repo of the root file.
func (c *Config) RootDir() string {
	if c.git == nil {
		panic("need git support for this, so you must call Read() first")
	}
	return c.git.Root()
}

// Abs returns the absolute path to p in the root file git repo.
func (c *Config) Abs(p string) (string, error) {
	if c.git == nil {
		panic("need git support for this, so you must call Read() first")
	}
	return c.git.Abs(p)
}

// Read reads the .claw file at clawFilePath, any claw.mod files found, any local.replace file,
// and uses it to build up our Config with all the files that are imported and that they import
// until we error or have all the files needed to begin building our Claw language files.
func (c *Config) Read(ctx context.Context, clawFilePath string) error {
	dir := filepath.Dir(clawFilePath)

	if c.git == nil {
		// Add git if its a git repo.
		git, err := vcs.NewGit(dir)
		if err == nil {
			c.git = git
		}
	}

	if _, err := c.fs.Stat(filepath.Join(dir, "global.replace")); err == nil {
		return fmt.Errorf("cannot compile claw files for directory with global.replace")
	}

	content, err := c.fs.ReadFile(clawFilePath)
	if err != nil {
		return fmt.Errorf("error: problem reading file %s: %s", clawFilePath, err)
	}
	file := idl.New()

	if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), file); err != nil {
		return err
	}

	if err := file.Validate(); err != nil {
		return fmt.Errorf("problem validating root file %q: %w", clawFilePath, err)
	}

	clawMod := filepath.Join(dir, "claw.mod")
	if err := c.readConfig(ctx, clawMod); err != nil {
		return fmt.Errorf("problem reading module file %q: %w", clawMod, err)
	}

	clawLocalReplace := filepath.Join(dir, "local.replace")
	if err := c.readLocalReplace(ctx, clawLocalReplace, c.Root); err != nil {
		return fmt.Errorf("problem reading local.replace: %w", err)
	}

	// Add all our replacements to our Imports, but set them to nil.
	for _, r := range c.LocalReplace.Replace {
		c.Imports[r.ToPath] = nil
	}

	for _, imp := range file.Imports.Imports {
		log.Println("import: ", imp.Path)
		var r Replace
		// We don't need to grab this if it has already been gotten.
		if _, ok := c.Imports[file.FullPath]; ok {
			r = c.LocalReplace.ReplaceMe(imp.Path)
			if r.IsZero() { // We already have it and its not a replacement.
				continue
			}

			// This import is locally replaced, let's see if we have the replacement already loaded.
			if c.Imports[r.ToPath] != nil {
				continue
			}

			// This is a stack copy, shouldn't affect the map entry.
			// We change it to the replacement and now its time to go replace stuff.
			imp.Name = path.Base(r.ToPath)
			imp.Path = r.ToPath
		}

		// Add our import path to a copy of the list and send it down.
		ctx = AppendImports(ctx, c.Module.Path)
		log.Printf("@Read(): %#+v", imp)
		if err := c.read(ctx, imp.Path); err != nil {
			return err
		}
	}
	file.FullPath = c.Module.Path
	c.Root = file
	c.Imports[c.Module.Path] = file

	// Now that we've done one pass and built our idl.File entries, we now need to go
	// back and update all the FullPath entries and attach all external identifiers
	// to the external identifier's idl.File.
	return c.populateExternals()
}

// populateExternals loops through all idl.File entries and updates their .Externals
// to reference the now parsed dependencies.
func (c *Config) populateExternals() error {
	for k, imp := range c.Imports {
		imp.FullPath = k
		if err := c.populateIDLExternals(imp); err != nil {
			return err
		}
	}
	return nil
}

// populateIDLExternals populates a single idl.File's External dependencies.
func (c *Config) populateIDLExternals(i *idl.File) error {
	for varType, idlImp := range i.External {
		if idlImp != nil {
			continue
		}
		// External names are [package].[type]
		sp := strings.Split(varType, ".")
		entry, err := i.Imports.ByPkgName(sp[0])
		if err != nil {
			// This is a defense in depth entry, this should never get to this point
			// and not have this set.
			return fmt.Errorf("package %q had type %q that we couldn't locate an imported package for", i.FullPath, varType)
		}
		i.External[varType] = c.Imports[entry.Path]
	}
	return nil
}

func (c *Config) readConfig(ctx context.Context, clawModPath string) error {
	// If a claw.mod file exists, read it in.
	fi, err := c.fs.Stat(clawModPath)
	if err == nil {
		if fi.IsDir() {
			return fmt.Errorf("there is a claw.mod directory, which is not allowed")
		}
		m := &Module{}
		content, err := c.fs.ReadFile(clawModPath)
		if err != nil {
			return fmt.Errorf("error: problem reading file %s: %s", clawModPath, err)
		}
		if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), m); err != nil {
			return err
		}
		c.Module = m
		return nil
	} else {
		if err != fs.ErrNotExist {
			return fmt.Errorf("problem reading claw.mod file: %w", err)
		}
		return err
	}
}

func (c *Config) readLocalReplace(ctx context.Context, localReplacePath string, f *idl.File) error {
	lr := NewLocalReplace(c.fs, f)
	fi, err := c.fs.Stat(localReplacePath)
	if err == nil {
		if fi.IsDir() {
			return fmt.Errorf("there is a local.replace directory, which is not allowed")
		}

		content, err := c.fs.ReadFile(localReplacePath)
		if err != nil {
			return fmt.Errorf("error: problem reading file %s: %s", localReplacePath, err)
		}
		if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), &lr); err != nil {
			return err
		}
		c.LocalReplace = lr
	} else {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("problem reading local.replace file: %w", err)
		}
	}
	return nil
}

func (c *Config) read(ctx context.Context, pkgPath string) error {
	// Detect cyclic imports using eBGP ASN detection methodology.
	// Aka... you can't see yourself in the path.
	l := ExtractImports(ctx)
	for _, i := range l {
		if i == pkgPath {
			l = append(l, pkgPath) // So the cyclic path shows up.
			return errors.New(l.String())
		}
	}
	ctx = AppendImports(ctx, pkgPath)

	if c.git == nil {
		return fmt.Errorf("for the moment, you must be in a git repo for clawc to work")
	}

	if c.git.InRepo(pkgPath) {
		localPath := strings.Join(strings.Split(pkgPath, c.git.Origin())[1:], "")
		localPath = strings.TrimPrefix(localPath, "/")
		localPath = filepath.Join(c.git.Root(), localPath)

		log.Println("reading local: ", pkgPath)
		return c.readLocal(ctx, pkgPath, localPath)
	}

	log.Println("reading remote: ", pkgPath)
	return c.readRemote(ctx, pkgPath)
}

func (c *Config) readLocal(ctx context.Context, pkgPath, localPath string) error {
	clawFile, err := FindClawFile(c.fs, localPath)
	if err != nil {
		return err
	}
	log.Println("local clawfile is: ", clawFile)
	content, err := c.fs.ReadFile(clawFile)
	if err != nil {
		return fmt.Errorf("could not read package(%s) that is local to the git repo at path %q", pkgPath, clawFile)
	}

	file := idl.New()

	if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), file); err != nil {
		return fmt.Errorf("problem parsing Claw package %q: %w", pkgPath, err)
	}

	if err := file.Validate(); err != nil {
		return fmt.Errorf("problem parsing Claw package %q: %w", pkgPath, err)
	}

	c.Imports[pkgPath] = file
	for _, imp := range file.Imports.Imports {
		log.Printf("@readLocal(): %#+v", imp.Path)
		if err := c.read(ctx, imp.Path); err != nil {
			return err
		}
	}
	log.Println("I made it")
	return nil
}

func (c *Config) readRemote(ctx context.Context, pkgPath string) error {
	cf, err := c.getClawFile(ctx, pkgPath, "")
	if err != nil {
		return err
	}

	log.Println("claw file content:\n", string(cf.Content))
	file := idl.New()

	if err := halfpike.Parse(ctx, conversions.ByteSlice2String(cf.Content), file); err != nil {
		return err
	}

	if err := file.Validate(); err != nil {
		return err
	}
	file.RepoVersion = cf.Version
	file.SHA256 = cf.SHA256

	c.Imports[pkgPath] = file
	for _, imp := range file.Imports.Imports {
		log.Printf("@readRemote(): %#+v", imp)
		if err := c.read(ctx, imp.Path); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) Validate(f *idl.File) error {
	if c.Module != nil {
		for _, req := range c.Module.Required {
			if _, ok := f.Imports.Imports[req.Path]; !ok {
				return fmt.Errorf("module file has required import %q that is not in the package", req.Path)
			}
		}
	}

	return nil
}

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) IsZero() bool {
	if v.Major == 0 && v.Minor == 0 && v.Patch == 0 {
		return true
	}
	return false
}

func (v *Version) FromString(s string) error {
	s = s[1:]
	l := strings.Split(s, ".")
	if len(l) != 3 {
		return fmt.Errorf("a version must have a major, minor and patch version")
	}
	n, err := strconv.Atoi(l[0])
	if err != nil {
		return fmt.Errorf("version's major number was not a number: %s", err)
	}
	v.Major = n
	n, err = strconv.Atoi(l[1])
	if err != nil {
		return fmt.Errorf("version's minor number was not a number: %s", err)
	}
	v.Minor = n
	n, err = strconv.Atoi(l[2])
	if err != nil {
		return fmt.Errorf("version's patch number was not a number: %s", err)
	}
	v.Patch = n
	return nil
}

type Require struct {
	// Path is the path the package is located at.
	Path string
	// Version is the version of the package we want. If this is the zero value,
	// we will use the ID.
	Version Version
	// ID is whatever the version control's ID is going to be.
	ID string
}

type ACL struct {
	Path string
}

type Replace struct {
	FromPath    string
	FromVersion Version
	ToPath      string
	ToVersion   Version
}

func (r Replace) IsZero() bool {
	return reflect.ValueOf(r).IsZero()
}
