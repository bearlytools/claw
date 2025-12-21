package vcs

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	libpath "path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/bearlytools/claw/clawc/languages/go/conversions"
	"github.com/pkg/errors"
)

var (
	ErrNotInstalled = fmt.Errorf("not installed")
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v *Version) FromString(s string) error {
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

func (v Version) meetsMin() bool {
	if v.Major < 2 {
		return false
	}
	if v.Minor < 18 {
		return false
	}
	return true
}

// Git provides some information on git.
type Git struct {
	gitCmd string
	path   string
	ver    Version
}

// NewGit returns a new version of Git. It will error if git is not installed or at the
// correct version required. Path is the path your are testing, which will be set to
// our working directory for all commands.
func NewGit(path string) (*Git, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrNotInstalled
	}

	g := &Git{gitCmd: gitPath, path: path}
	if !g.Using() {
		return nil, err
	}

	if err := g.version(); err != nil {
		return nil, err
	}

	if !g.ver.meetsMin() {
		return nil, fmt.Errorf("do not support git version < 2.18.0")
	}
	return g, nil
}

// Using returns true if the directory you wanted is using git.
func (g *Git) Using() bool {
	// git rev-parse --is-inside-work-tree
	cmd := exec.Command(g.gitCmd, "rev-parse", "--is-inside-work-tree")
	cmd.Dir = g.path

	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("had error trying to run git: %s", err))
	}
	b = bytes.TrimSpace(b)
	switch conversions.ByteSlice2String(b) {
	case "true":
		return true
	}
	return false
}

func (g *Git) Version() Version { return g.ver }

var verRE = regexp.MustCompile(`\d+\.\d+\.\d+`)

func (g *Git) version() error {
	// git rev-parse --is-inside-work-tree
	cmd := exec.Command(g.gitCmd, "version")
	cmd.Dir = g.path

	b, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	m := verRE.Find(b)
	if m == nil {
		return fmt.Errorf("version string not found in 'git version' output(%s)", string(b))
	}
	v := Version{}
	if err := v.FromString(conversions.ByteSlice2String(m)); err != nil {
		return err
	}
	g.ver = v
	return nil
}

// Root returns the current git root (ignoring submodules).
func (g *Git) Root() string {
	// git rev-parse --git-dir
	cmd := exec.Command(g.gitCmd, "rev-parse", "--git-dir")
	cmd.Dir = g.path

	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("had error trying to run git: %s", err))
	}

	sp := bytes.Split(b, []byte(".git"))
	if len(sp) != 2 {
		panic(fmt.Errorf("'git rev-parse --git-dir returned output that wasn't expected: %s", string(b)))
	}
	return conversions.ByteSlice2String(sp[0])
}

// Origin returns the origin's path without http:// or https:// and without .git at the end.
func (g *Git) Origin() string {
	cmd := exec.Command(g.gitCmd, "config", "--get", "remote.origin.url")
	cmd.Dir = g.path
	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	s := strings.TrimSpace(conversions.ByteSlice2String(b))
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u.Host + strings.TrimSuffix(u.Path, ".git")
}

// InRepo let's you know if the path is in the repo or not by looking for it in the filesystem.
func (g *Git) InRepo(pkgPath string) bool {
	pkgPath = libpath.Clean(pkgPath)
	return strings.HasPrefix(pkgPath, g.Origin())
}

// Abs will return the absolute path of path p in the filesystem if p is in this git repo.
// If the pkg that this path belongs to is .InRepo() === false, this will give useless output.
func (g *Git) Abs(p string) (string, error) {
	p = libpath.Clean(p)

	// If p == "github.com/johnsiilver/claw/subdir/claw"
	// and g.Origin() == "github.com/johnsiilver/claw"
	// sp == ["/subdir/claw"]
	sp := strings.Split(p, g.Origin())
	if len(sp) < 1 {
		return "", fmt.Errorf("git.Abs(%q) results in nothing(origin: %q", p, g.Origin())
	}
	s := strings.Join(sp[1:], "/")
	s = strings.TrimPrefix(s, "/")
	// Now we have s == "subdir/claw"
	// g.Root() == "/usr/someone/trees/claw/"
	s = filepath.Join(g.Root(), s)
	var err error
	s, err = filepath.Abs(s)
	if err != nil {
		return "", fmt.Errorf("git.Abs(%q) error: %w", p, err)
	}

	// Nowe should have s == "/usr/someone/trees/claw/subdir/claw"
	return s, nil
}

///////////////////////////////////////////////////
// This section below is borrowed from https://github.com/golang/dep/pull/395/files#diff-d06b1f6d53d1d1307a7a266f4d9a8d89e5d77aa22877b00bb8fb1900e02d74c8
// where they replaced the non-working filepath.HasPrefix with one that works.
///////////////////////////////////////////////////

func IsDir(name string) (bool, error) {
	// TODO: lstat?
	fi, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !fi.IsDir() {
		return false, errors.Errorf("%q is not a directory", name)
	}
	return true, nil
}

// HasFilepathPrefix will determine if "path" starts with "prefix" from
// the point of view of a filesystem.
//
// Unlike filepath.HasPrefix, this function is path-aware, meaning that
// it knows that two directories /foo and /foobar are not the same
// thing, and therefore HasFilepathPrefix("/foobar", "/foo") will return
// false.
//
// This function also handles the case where the involved filesystems
// are case-insensitive, meaning /foo/bar and /Foo/Bar correspond to the
// same file. In that situation HasFilepathPrefix("/Foo/Bar", "/foo")
// will return true. The implementation is *not* OS-specific, so a FAT32
// filesystem mounted on Linux will be handled correctly.
func HasFilepathPrefix(path, prefix string) bool {
	if filepath.VolumeName(path) != filepath.VolumeName(prefix) {
		return false
	}

	var dn string

	if isDir, err := IsDir(path); err != nil {
		return false
	} else if isDir {
		dn = path
	} else {
		dn = filepath.Dir(path)
	}

	dn = strings.TrimSuffix(dn, string(os.PathSeparator))
	prefix = strings.TrimSuffix(prefix, string(os.PathSeparator))

	dirs := strings.Split(dn, string(os.PathSeparator))[1:]
	prefixes := strings.Split(prefix, string(os.PathSeparator))[1:]

	if len(prefixes) > len(dirs) {
		return false
	}

	var d, p string

	for i := range prefixes {
		// need to test each component of the path for
		// case-sensitiveness because on Unix we could have
		// something like ext4 filesystem mounted on FAT
		// mountpoint, mounted on ext4 filesystem, i.e. the
		// problematic filesystem is not the last one.
		if isCaseSensitiveFilesystem(filepath.Join(d, dirs[i])) {
			d = filepath.Join(d, dirs[i])
			p = filepath.Join(p, prefixes[i])
		} else {
			d = filepath.Join(d, strings.ToLower(dirs[i]))
			p = filepath.Join(p, strings.ToLower(prefixes[i]))
		}

		if p != d {
			return false
		}
	}

	return true
}

// genTestFilename returns a string with at most one rune case-flipped.
//
// The transformation is applied only to the first rune that can be
// reversibly case-flipped, meaning:
//
// * A lowercase rune for which it's true that lower(upper(r)) == r
// * An uppercase rune for which it's true that upper(lower(r)) == r
//
// All the other runes are left intact.
func genTestFilename(str string) string {
	flip := true
	return strings.Map(func(r rune) rune {
		if flip {
			if unicode.IsLower(r) {
				u := unicode.ToUpper(r)
				if unicode.ToLower(u) == r {
					r = u
					flip = false
				}
			} else if unicode.IsUpper(r) {
				l := unicode.ToLower(r)
				if unicode.ToUpper(l) == r {
					r = l
					flip = false
				}
			}
		}
		return r
	}, str)
}

// isCaseSensitiveFilesystem determines if the filesystem where dir
// exists is case sensitive or not.
//
// CAVEAT: this function works by taking the last component of the given
// path and flipping the case of the first letter for which case
// flipping is a reversible operation (/foo/Bar → /foo/bar), then
// testing for the existence of the new filename. There are two
// possibilities:
//
// 1. The alternate filename does not exist. We can conclude that the
// filesystem is case sensitive.
//
// 2. The filename happens to exist. We have to test if the two files
// are the same file (case insensitive file system) or different ones
// (case sensitive filesystem).
//
// If the input directory is such that the last component is composed
// exclusively of case-less codepoints (e.g.  numbers), this function will
// return false.
func isCaseSensitiveFilesystem(dir string) bool {
	alt := filepath.Join(filepath.Dir(dir),
		genTestFilename(filepath.Base(dir)))

	dInfo, err := os.Stat(dir)
	if err != nil {
		return true
	}

	aInfo, err := os.Stat(alt)
	if err != nil {
		return true
	}

	return !os.SameFile(dInfo, aInfo)
}
