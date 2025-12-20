// Package work provides parsing for claw.work files which define workspace configuration.
package work

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gostdlib/base/context"
	"github.com/johnsiilver/halfpike"

	"github.com/bearlytools/claw/internal/conversions"
)

const workFileName = "claw.work"

// Work represents a claw.work file configuration.
type Work struct {
	// Repo is the repository path (e.g., "github.com/bearlytools/claw").
	Repo string
	// VendorDir is the vendor directory name relative to the claw.work file location.
	VendorDir string
}

// Validate checks that the Work configuration is valid.
func (w *Work) Validate() error {
	if w.Repo == "" {
		return fmt.Errorf("repo is required in claw.work")
	}
	if w.VendorDir == "" {
		return fmt.Errorf("vendorDir is required in claw.work")
	}
	if strings.Contains(w.Repo, "//") {
		return fmt.Errorf("repo cannot contain //")
	}
	if filepath.IsAbs(w.VendorDir) {
		return fmt.Errorf("vendorDir must be a relative path, got %q", w.VendorDir)
	}
	return nil
}

// ValidateModuleInRepo checks that a module path is equal to or a child of the repo path.
func ValidateModuleInRepo(modulePath, repoPath string) error {
	if !strings.HasPrefix(modulePath, repoPath) {
		return fmt.Errorf("module %q must be equal to or a child of repo %q", modulePath, repoPath)
	}
	if len(modulePath) > len(repoPath) && modulePath[len(repoPath)] != '/' {
		return fmt.Errorf("module %q must be equal to or a child of repo %q", modulePath, repoPath)
	}
	return nil
}

// Start is the start point for reading the claw.work file.
func (w *Work) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return w.ParseRepo
}

func (w *Work) skipLinesWithComments(p *halfpike.Parser) {
	l := p.Next()

	if len(l.Items) > 0 && strings.HasPrefix(l.Items[0].Val, "//") {
		if p.EOF(l) {
			return
		}
		w.skipLinesWithComments(p)
	} else {
		p.Backup()
	}
}

// ParseRepo parses the repo directive which must be the first non-comment line.
func (w *Work) ParseRepo(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	w.skipLinesWithComments(p)

	l := p.Next()

	if len(l.Items) < 3 {
		return p.Errorf("[Line %d] error: first directive must be 'repo <path>'", l.LineNum)
	}

	if l.Items[0].Val != "repo" {
		return p.Errorf("[Line %d] error: expect 'repo' keyword as first directive, not %q", l.LineNum, l.Items[0].Val)
	}

	w.Repo = l.Items[1].Val
	if err := commentOrEOL(l, 2); err != nil {
		return p.Errorf("[Line %d] %s", l.LineNum, err.Error())
	}

	return w.FindNext
}

// FindNext scans for the next directive to parse.
func (w *Work) FindNext(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	w.skipLinesWithComments(p)

	line := p.Next()

	if p.EOF(line) {
		return nil
	}

	switch strings.ToLower(line.Items[0].Val) {
	case "vendordir":
		if w.VendorDir != "" {
			return p.Errorf("[Line %d] error: duplicate 'vendorDir' directive found", line.LineNum)
		}
		p.Backup()
		return w.ParseVendorDir
	default:
		return p.Errorf("[Line %d] error: unknown directive %q, expected 'vendorDir'", line.LineNum, line.Items[0].Val)
	}
}

// ParseVendorDir parses the vendorDir directive.
func (w *Work) ParseVendorDir(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	l := p.Next()

	if len(l.Items) < 3 {
		return p.Errorf("[Line %d] error: expected 'vendorDir <path>'", l.LineNum)
	}

	if err := caseSensitiveCheck("vendorDir", l.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %s", l.LineNum, err.Error())
	}

	w.VendorDir = l.Items[1].Val
	if err := commentOrEOL(l, 2); err != nil {
		return p.Errorf("[Line %d] %s", l.LineNum, err.Error())
	}

	return w.FindNext
}

func caseSensitiveCheck(want string, item string) error {
	if item != want {
		if strings.EqualFold(item, want) {
			return fmt.Errorf("%q keyword found, but it is required to be %q", item, want)
		}
		return fmt.Errorf("got: %q, want: %q", item, want)
	}
	return nil
}

func isComment(item halfpike.Item) bool {
	return strings.HasPrefix(item.Val, "//")
}

func commentOrEOL(line halfpike.Line, from int) error {
	if from >= len(line.Items) {
		return nil
	}

	if isComment(line.Items[from]) {
		return nil
	}

	if len(line.Items[from:]) > 1 {
		return fmt.Errorf("got item %q after %q, which was unexpected", halfpike.ItemJoin(line, from, len(line.Items)), halfpike.ItemJoin(line, 0, from))
	}

	return nil
}

// Parse parses the content of a claw.work file.
func Parse(ctx context.Context, content []byte) (*Work, error) {
	w := &Work{}
	if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), w); err != nil {
		return nil, fmt.Errorf("failed to parse claw.work: %w", err)
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	return w, nil
}

// FindWork searches for a claw.work file starting from startDir and walking up
// the directory tree. Returns the parsed Work, the directory containing claw.work,
// and any error. Returns an error if no claw.work file is found.
func FindWork(ctx context.Context, startDir string) (*Work, string, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	dir := absStart
	for {
		workPath := filepath.Join(dir, workFileName)
		content, err := os.ReadFile(workPath)
		if err == nil {
			w, parseErr := Parse(ctx, content)
			if parseErr != nil {
				return nil, "", fmt.Errorf("failed to parse %s: %w", workPath, parseErr)
			}
			return w, dir, nil
		}

		if !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("failed to read %s: %w", workPath, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, "", fmt.Errorf("claw.work file not found (searched from %s to root)", absStart)
		}
		dir = parent
	}
}
