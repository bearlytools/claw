package git

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/internal/imports/globalreplace"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/johnsiilver/halfpike"
)

// ClawFile holds a Claw file's content and version of the git repo.
type ClawFile struct {
	Content []byte
	Version string
	SHA256  string
}

// ModuleFile holds a claw.mod file's content.
type ModuleFile struct {
	Content []byte
	Exists  bool
}

// GetClawFile retrieves the .claw file for a package at a commit version. version can be
// an empty string and the .claw file will be for the latest commit.
func GetClawFile(ctx context.Context, pkgPath, version string) (ClawFile, error) {
	return getClawFile(ctx, pkgPath, version, 0)
}

// GetModuleFile retrieves the claw.mod file for a package at a commit version. version can be
// an empty string and the claw.mod file will be for the latest commit.
func GetModuleFile(ctx context.Context, pkgPath, version string) (ModuleFile, error) {
	log.Println("GetModuleFile pkgPath: ", pkgPath)
	localRepo, _, err := cloneRepo(pkgPath, version)
	if err != nil {
		return ModuleFile{}, err
	}

	insidePath := ""
	sp := strings.Split(pkgPath, "/")
	if len(sp) > 3 {
		insidePath = strings.Join(strings.Split(pkgPath, "/")[3:], "/")
	}
	clawDirPath := filepath.Join(localRepo, insidePath)

	modFilePath := filepath.Join(clawDirPath, "claw.mod")
	log.Println("modFilePath: ", modFilePath)

	b, err := os.ReadFile(modFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ModuleFile{Exists: false}, nil
		}
		return ModuleFile{}, fmt.Errorf("problem reading claw.mod file from git path %q: %w", modFilePath, err)
	}

	return ModuleFile{Content: b, Exists: true}, nil
}

func getClawFile(ctx context.Context, pkgPath, version string, depth int) (ClawFile, error) {
	if depth == 5 {
		return ClawFile{}, fmt.Errorf("had 5 global.replace redirects which exceeds our limit")
	}

	log.Println("pkgPath: ", pkgPath)
	localRepo, ver, err := cloneRepo(pkgPath, version)
	if err != nil {
		return ClawFile{}, err
	}
	if version == "" {
		version = ver
	}

	insidePath := ""
	sp := strings.Split(pkgPath, "/")
	if len(sp) > 3 {
		insidePath = strings.Join(strings.Split(pkgPath, "/")[3:], "/")
	}
	clawDirPath := filepath.Join(localRepo, insidePath)

	// We have a global.replace file, we need to parse it and the follow where it goes.
	grPath := filepath.Join(clawDirPath, "global.replace")
	if _, err := os.Stat(grPath); err == nil {
		b, err := os.ReadFile(grPath)
		if err != nil {
			return ClawFile{}, fmt.Errorf("package %q has a global.replace file, but we can't read it: %w", pkgPath, err)
		}
		gr := globalreplace.GlobalReplace{}

		if err := halfpike.Parse(ctx, conversions.ByteSlice2String(b), &gr); err != nil {
			return ClawFile{}, fmt.Errorf("package %q had global.replace file we could not parse: %w", pkgPath, err)
		}
		return getClawFile(ctx, gr.Replacement, version, depth+1)
	}

	log.Println("clawDirPath: ", clawDirPath)
	clawFile, err := FindClawFile(clawDirPath)
	if err != nil {
		return ClawFile{}, err
	}

	b, err := os.ReadFile(clawFile)
	if err != nil {
		return ClawFile{}, fmt.Errorf("problem reading .claw file from git path %q: %w", clawFile, err)
	}
	sum := sha256.Sum256(b)
	sha := fmt.Sprintf("%x", sum)
	return ClawFile{Content: b, Version: version, SHA256: sha}, nil
}

func cloneRepo(pkgPath string, version string) (repoPath, ver string, err error) {
	sp := strings.Split(pkgPath, "/")
	log.Println("repo: ", sp)
	// This is: [tmpdir]/claw/[host]/[repo]
	fp := filepath.Join(append(append([]string{}, os.TempDir(), "claw"), sp[0:3]...)...)

	var repo *git.Repository

	log.Println("fp: ", fp)
	var downloadRepo = true
	// This is already downloaded, so we can just update it.
	if _, err := os.Stat(fp); err == nil {
		repo, err = git.PlainOpen(fp)
		if err == nil {
			downloadRepo = false
			log.Println("successfully found: ", fp)
		} else {
			log.Printf("problem opening existing git repo at %q: %s", fp, err)
			if err := os.RemoveAll(fp); err != nil {
				log.Printf("cannot remove directory %s: %s", fp, err)
			}
		}
	}
	if downloadRepo {
		log.Println("have to fetch repo")
		domain := sp[0]
		user := sp[1]
		repoName := sp[2]
		// Clones the repository into the given dir, just as a normal git clone does.
		u, err := url.Parse(fmt.Sprintf("https://%s/%s/%s.git", domain, user, repoName))
		if err != nil {
			return "", "", fmt.Errorf("problem url parsing the git repo for package %q: %w", pkgPath, err)
		}
		repo, err = git.PlainClone(
			fp, false,
			&git.CloneOptions{
				URL: u.String(),
			},
		)
		if err != nil {
			return "", "", fmt.Errorf("problem cloning %q: %w", u.String(), err)
		}
		log.Println("cloned:", u.String())

		// We haven't downloaded it, so we need to clone it and checkout our version.
		err = os.MkdirAll(fp, 0700)
		if err != nil {
			return "", "", err
		}

		if err != nil {
			return "", "", fmt.Errorf("problem cloning repo for package %q: %w", pkgPath, err)
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", "", fmt.Errorf("problem getting repo worktree for package %q: %w", pkgPath, err)
	}

	if err = wt.Pull(&git.PullOptions{}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return "", "", fmt.Errorf("had issue doing a git pull for git repo at %q: %w", fp, err)

		}
	}

	if err := versionCheckout(repo, wt, version); err != nil {
		return "", "", err
	}

	// Since we didn't declare a version, we need to retrieve a reference hash.
	if version == "" {
		version, err = currentVersion(repo)
		if err != nil {
			return "", "", err
		}
	}

	return fp, version, nil
}

func currentVersion(repo *git.Repository) (string, error) {
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	// ... retrieves the commit history
	iter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return "", err
	}

	c, err := iter.Next()
	if err != nil {
		return "", fmt.Errorf("there do not appear to be any commits: %s", err)
	}
	return c.Hash.String(), nil
}

// versionCheckout will checkout the git repo at version. Version can be either the hash
// or a semantic version tag. If version == "", then we will not do a checkout of a particular
// version.
func versionCheckout(repo *git.Repository, wt *git.Worktree, version string) error {
	if version == "" {
		return nil
	}

	v := Version{}
	if err := v.FromString(version); err == nil {
		return semanticCheckout(repo, wt, version)
	}

	return hashCheckout(wt, version)
}

func semanticCheckout(repo *git.Repository, wt *git.Worktree, version string) error {
	iter, err := repo.Tags()
	if err != nil {
		return fmt.Errorf("problem getting git repo tags: %w", err)
	}

	for {
		ref, err := iter.Next()
		if err != nil {
			return fmt.Errorf("could not find version %q that was specified: %w", version, err)
		}
		if string(ref.Name()) == version {
			return hashCheckout(wt, ref.Hash().String())
		}
	}
	panic("should never get here")
}

func hashCheckout(wt *git.Worktree, version string) error {
	if err := wt.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(version)}); err != nil {
		return fmt.Errorf("had issue checking out verion %q: %w", version, err)
	}
	return nil
}

// FindClawFile will return the name of the .claw file at path.
func FindClawFile(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	found := []string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".claw") {
			found = append(found, e.Name())
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf(".claw file not found in %q", path)
	case 1:
		fp, err := filepath.Abs(filepath.Join(path, found[0]))
		if err != nil {
			panic(err)
		}
		return fp, nil
	}
	return "", fmt.Errorf("found multiple .claw files (not allowe) at %q", path)
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
	s = strings.TrimPrefix(s, "v")
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
