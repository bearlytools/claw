package imports

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gostdlib/base/context"

	"github.com/johnsiilver/halfpike"

	"github.com/bearlytools/claw/clawc/internal/idl"
)

// LocalReplace represents the local.replace file and includes halfpike methods to decode the file.
type LocalReplace struct {
	Replace []Replace

	idl *idl.File
	fs  neededFS

	// testing is used to tell us that we are doing a test and don't want to actually do
	// validation that relies on idl.File and other things.
	testing bool
}

func NewLocalReplace(fs neededFS, f *idl.File) LocalReplace {
	return LocalReplace{fs: fs, idl: f}
}

// ReplaceMe returns the replacement for path. Use Replace.IsZero() to dermine if it was found or not.
func (l *LocalReplace) ReplaceMe(path string) Replace {
	for _, r := range l.Replace {
		if r.FromPath == path {
			return r
		}
	}
	return Replace{}
}

// Start is the start point for reading the IDL.
func (l *LocalReplace) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return l.FindNext
}

func (l *LocalReplace) SkipLinesWithComments(p *halfpike.Parser) {
	line := p.Next()

	if strings.HasPrefix("//", line.Items[0].Val) {
		if p.EOF(line) {
			return
		}
		l.SkipLinesWithComments(p)
	} else {
		p.Backup()
	}
}

func (l *LocalReplace) FindNext(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	l.SkipLinesWithComments(p)

	line := p.Next()

	switch strings.ToLower(line.Items[0].Val) {
	case "replace":
		if len(l.Replace) > 0 {
			return p.Errorf("[Line %d] error: duplicate 'replace' line found", line.LineNum)
		}
		p.Backup()
		return l.ParseReplace
	default:
		if p.EOF(line) {
			return nil
		}
		return p.Errorf("[Line %d] do not understand this line", line.LineNum)
	}
}

func (l *LocalReplace) ParseReplace(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'replace ('", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("replace", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf("%w", err)
	}

	for {
		line = p.Next()
		if p.EOF(line) {
			return p.Errorf("unexpected EOF before close of 'replace' directive")
		}
		if len(line.Items) < 2 {
			return p.Errorf("[Line %d] error: want either a ) or a replacement statement", line.LineNum)
		}
		if line.Items[0].Val == ")" {
			if len(l.Replace) == 0 {
				return p.Errorf("error: cannot have a 'replace' directive with no statements")
			}
			if err := commentOrEOL(line, 1); err != nil {
				return p.Errorf("%s", err.Error())
			}
			return l.FindNext
		}

		if err := l.parseReplaceLine(line); err != nil {
			return p.Errorf("[Line %d] error: %s", line.LineNum, err)
		}
	}
}

func (l *LocalReplace) parseReplaceLine(line halfpike.Line) error {
	if len(line.Items) < 4 {
		return fmt.Errorf("expected [package] => [/local/directory/path]")
	}
	if line.Items[1].Val != "=>" {
		return fmt.Errorf("expected second item to be =>, got %q", line.Items[1])
	}

	r := Replace{
		FromPath: line.Items[0].Val,
		ToPath:   line.Items[2].Val,
	}
	if err := commentOrEOL(line, 3); err != nil {
		return err
	}
	l.Replace = append(l.Replace, r)

	return nil
}

func (l *LocalReplace) Validate() error {
	if l.testing {
		return nil
	}

	for _, r := range l.Replace {
		if _, ok := l.idl.Imports.Imports[r.FromPath]; !ok {
			return fmt.Errorf("local.replace file has replace for package %q, but it is not found in the package imports", r.FromPath)
		}

		fi, err := os.Stat(r.ToPath)
		if err != nil {
			return fmt.Errorf("local.replace has replace for package %q with local directory %q, but that path had error: ", r.FromPath, r.ToPath)
		}
		if !fi.IsDir() {
			return fmt.Errorf("local.replace has replace for package %q with local directory %q, but that path is not a directory", r.FromPath, r.ToPath)
		}

		_, err = FindClawFile(l.fs, r.ToPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// FindClawFile will return the absolute path of the .claw file at path.
func FindClawFile(fs fs.ReadDirFS, path string) (string, error) {
	entries, err := fs.ReadDir(path)
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
