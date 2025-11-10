package imports

import (
	"context"
	"fmt"
	"log"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/johnsiilver/halfpike"

	"github.com/bearlytools/claw/internal/idl"
)

// Module represents a claw.mod file and includes halpike methods to decode the file.
type Module struct {
	// Path is the module path.
	Path string
	// Required is a list of specific versions of packages that must be imported.
	Required []Require
	// Replace is a list of packages that should be replaced with a different location.
	Replace []Replace
	// ACLs are a list of ACLs that are provided. If it is set to acls = public, then
	// the first and only ACL will be *.
	ACLs []ACL
}

func (m *Module) Validate() error {
	if err := ValidModuleName(m.Path); err != nil {
		return err
	}
	return nil
}

// Start is the start point for reading the IDL.
func (m *Module) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return m.ParseModule
}

func (m *Module) SkipLinesWithComments(p *halfpike.Parser) {
	l := p.Next()

	if strings.HasPrefix("//", l.Items[0].Val) {
		if p.EOF(l) {
			return
		}
		m.SkipLinesWithComments(p)
	} else {
		p.Backup()
	}
}

func (m *Module) ParseModule(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	l := p.Next()

	if len(l.Items) < 3 {
		return p.Errorf("[Line %d] error: first line must be a 'module' statement", l.LineNum)
	}

	if l.Items[0].Val != "module" {
		return p.Errorf("[Line %d] error: expect 'module' keyword as first word, not %q", l.LineNum, l.Items[0].Val)
	}

	m.Path = l.Items[1].Val
	if err := commentOrEOL(l, 2); err != nil {
		return p.Errorf("%s", err.Error())
	}

	return m.FindNext
}

// FindNext is used to scan lines until we find the next thing to parse and direct
// to the halfpike.ParseFn responsible.
func (m *Module) FindNext(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	m.SkipLinesWithComments(p)

	line := p.Next()

	switch strings.ToLower(line.Items[0].Val) {
	case "require":
		if len(m.Required) >= 0 {
			return p.Errorf("[Line %d] error: duplicate 'require' line found", line.LineNum)
		}
		p.Backup()
		return m.ParseRequire
	case "replace":
		if len(m.Replace) > 0 {
			return p.Errorf("[Line %d] error: duplicate 'replace' line found", line.LineNum)
		}
		p.Backup()
		return m.ParseReplace
	case "acls":
		if len(m.ACLs) != 0 {
			return p.Errorf("[Line %d] error: duplicate 'acls' line found", line.LineNum)
		}
		p.Backup()
		return m.ParseACLs
	default:
		if p.EOF(line) {
			return nil
		}
		log.Println("what is this: ", line)
		return p.Errorf("[Line %d] do not understand this line", line.LineNum)
	}
}

func (m *Module) ParseRequire(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'require ('", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("require", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf("%s", err.Error())
	}

	for {
		line = p.Next()
		if p.EOF(line) {
			return p.Errorf("unexpected EOF before close of 'require' directive")
		}
		if len(line.Items) < 2 {
			return p.Errorf("[Line %d] error: want either a ) or a require statement", line.LineNum)
		}
		if line.Items[0].Val == ")" {
			if len(m.Required) == 0 {
				return p.Errorf("error: cannot have a 'required' directive with no statements")
			}
			if err := commentOrEOL(line, 1); err != nil {
				return p.Errorf("%s", err.Error())
			}
			return m.FindNext
		}

		if err := m.parseRequireLine(line); err != nil {
			return p.Errorf("[Line %d] error: %s", line.LineNum, err)
		}
	}
}

var verRegex = regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)`)

func (m *Module) parseRequireLine(line halfpike.Line) error {
	if len(line.Items) < 3 {
		return fmt.Errorf("expected [package] v[semantic version]")
	}

	// This means that it wasn't a v1.3.2, so it must be a commit string.
	if verRegex.MatchString(line.Items[1].Val) {
		v := Version{}
		if err := v.FromString(line.Items[1].Val); err != nil {
			return err
		}
		r := Require{
			Path:    line.Items[0].Val,
			Version: v,
		}
		m.Required = append(m.Required, r)
		return nil
	}

	// Okay, that must mean that it was a commit string.
	r := Require{
		Path: line.Items[0].Val,
		ID:   line.Items[1].Val,
	}
	m.Required = append(m.Required, r)

	if err := commentOrEOL(line, 2); err != nil {
		return err
	}
	return nil
}

func (m *Module) ParseReplace(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'replace ('", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("replace", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf("%s", err.Error())
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
			if len(m.Replace) == 0 {
				return p.Errorf("error: cannot have a 'replace' directive with no statements")
			}
			if err := commentOrEOL(line, 1); err != nil {
				return p.Errorf("%s", err.Error())
			}
			return m.FindNext
		}

		if err := m.parseReplaceLine(line); err != nil {
			return p.Errorf("[Line %d] error: %s", line.LineNum, err)
		}
	}
}

func (m *Module) parseReplaceLine(line halfpike.Line) error {
	if len(line.Items) < 4 {
		return fmt.Errorf("expected [package] => [package]")
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
	m.Replace = append(m.Replace, r)
	return nil
}

func (m *Module) ParseACLs(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'acls (' or 'acls = \"public\"'", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("acls", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	if line.Items[1].Val == "=" {
		if line.Items[2].Val == "public" {
			m.ACLs = append(m.ACLs, ACL{Path: "*"})
			if err := commentOrEOL(line, 3); err != nil {
				return p.Errorf("%s", err.Error())
			}
			return m.FindNext
		}
		return p.Errorf("[Line %d] error: if 'acls' followed by '=', the next item must be 'public'", line.LineNum)
	}

	if line.Items[1].Val != "(" {
		return p.Errorf("[Line %d] error: if 'acls' can followed by '= public' or '(', not %q", line.LineNum, line.Items[1].Val)
	}

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf("%s", err.Error())
	}

	for {
		line = p.Next()
		if p.EOF(line) {
			return p.Errorf("unexpected EOF before close of 'acls' directive")
		}
		if len(line.Items) < 2 {
			return p.Errorf("[Line %d] error: want either a ) or an acl statement", line.LineNum)
		}
		if line.Items[0].Val == ")" {
			if len(m.ACLs) == 0 {
				return p.Errorf("error: cannot have a 'acls' directive with no statements")
			}
			if err := commentOrEOL(line, 1); err != nil {
				return p.Errorf("%s", err.Error())
			}
			return m.FindNext
		}

		index := strings.Index(line.Items[0].Val, "*")
		if index != -1 || index != (len(line.Items[0].Val)-1) {
			return p.Errorf("an ACL can only have * as the last character in the ACL")
		}
		m.ACLs = append(m.ACLs, ACL{Path: line.Items[0].Val})
	}
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
	if isComment(line.Items[from]) {
		return nil
	}

	if len(line.Items[from:]) > 1 {
		return fmt.Errorf("got item %q after %q, which was unexpected", halfpike.ItemJoin(line, from, len(line.Items)), halfpike.ItemJoin(line, 0, from))
	}

	return nil
}

// ValidModuleName determines if a module name is valid.
func ValidModuleName(module string) error {
	pkgName := path.Base(module)
	if err := idl.ValidPackage(pkgName); err != nil {
		return err
	}

	if strings.Contains(module, "//") {
		return fmt.Errorf("module cannot have //")
	}

	for _, r := range pkgName {
		if unicode.IsLetter(r) {
			continue
		}
		if unicode.IsNumber(r) {
			continue
		}
		switch r {
		case '_', '-', '/':
			continue
		}
		return fmt.Errorf("module name contains character %v which is invalid for a module name", r)
	}
	return nil
}
