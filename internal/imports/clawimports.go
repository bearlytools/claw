package imports

import (
	"context"
	"fmt"
	"strings"

	"github.com/johnsiilver/halfpike"
	"github.com/johnsiilver/halfpike/line"
)

// Directive is a directive on where to store our external package renders.
type Directive struct {
	// Path is the path to the location in the git repo to store the imports.
	Path string
}

// Validate validates the Directive Path for format errors. This doesn't mean
// the path is good.
func (d Directive) Validate() error {
	if d.Path == "" {
		return fmt.Errorf("claw.imports Directive must have Path set")
	}
	if strings.HasPrefix(d.Path, "/") {
		return fmt.Errorf("claw.imports Directive.Path must not start with a /")
	}
	if strings.HasSuffix(d.Path, "/") {
		return fmt.Errorf("claw.imports Directive.Path must not end with a /")
	}
	return nil
}

/*
ClawImports represents a claw.imports file and includes halfpike methods to decode the file.
A claw.imports file looks like the following:

	directive {
		path: "claw/external/imports"
	}
*/
type ClawImports struct {
	// Directive informs where to put all external repository imports.
	Directive Directive
}

func (c *ClawImports) Validate() error {
	return c.Directive.Validate()
}

// Start is the start point for reading the claw.imports file.
func (c *ClawImports) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return c.ParseDirective
}

func (c *ClawImports) SkipLinesWithComments(p *halfpike.Parser) {
	l := p.Next()

	if strings.HasPrefix("//", l.Items[0].Val) {
		if p.EOF(l) {
			return
		}
		c.SkipLinesWithComments(p)
	} else {
		p.Backup()
	}
}

func (c *ClawImports) ParseDirective(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	c.SkipLinesWithComments(p)
	l := p.Next()

	// Get "directive {" line.
	if len(l.Items) < 3 {
		return p.Errorf("[Line %d] error: first line must be a 'Directive' statement", l.LineNum)
	}

	if l.Items[0].Val != "directive" {
		return p.Errorf("[Line %d] error: expect 'directive' keyword as first word, not %q", l.LineNum, l.Items[0].Val)
	}

	if l.Items[1].Val != "{" {
		return p.Errorf("Line[%d] error: expected '{' after 'directive', not %q", l.LineNum, l.Items[0].Val)
	}

	if err := commentOrEOL(l, 2); err != nil {
		return p.Errorf("%w", err)
	}

	// Get 'path: "path/to/imports"' line
	c.SkipLinesWithComments(p)
	l = p.Next()
	if len(l.Items) < 3 {
		return p.Errorf("[Line %d] error: expected a 'Directive' 'Path' statement", l.LineNum)
	}
	if l.Items[0].Val != "path:" {
		return p.Errorf("[Line %d] error: expect 'directive' keyword as first word, not %q", l.LineNum, l.Items[0].Val)
	}

	path := ""

	lex := line.New(halfpike.ItemJoin(l, 1, len(l.Items)))
	line.SkipAllSpaces(lex)
	item := lex.Next()
	if item.Type == line.ItemText {
		if item.HasPrefix(`"`) && item.HasSuffix(`"`) {
			path = item.Val
		} else if item.HasPrefix(`"`) {
			path += item.Val
			for {
				item = lex.Next()
				if item.Type == line.ItemEOF || item.Type == line.ItemEOL {
					return p.Errorf("[Line %d] error: 'path' assignment was no complete, missing ending \" character", l.LineNum)
				}
				if item.HasSuffix(`"`) {
					path += item.Val
					line.SkipAllSpaces(lex)
					item = lex.Next()
					switch item.Type {
					case line.ItemEOF, line.ItemEOL:
						break
					default:
						return p.Errorf("[Line %d] error: 'path' assignment has extra characters after ending \" character", l.LineNum)
					}
				}
				path += item.Val
			}
		} else {
			return p.Errorf("[Line %d] error: path arguement starts with \" character", l.LineNum)
		}
	} else {
		return p.Errorf("[Line %d] error: path arguement cannot start with anything but a \" character", l.LineNum)
	}

	// Get closing } .
	c.SkipLinesWithComments(p)
	l = p.Next()
	if strings.TrimSpace(l.Raw) != "}" {
		return p.Errorf("[Line %d] error: after path statement in directive there must be a closing }", l.LineNum)
	}

	c.Directive.Path = strings.Trim(path, `"`)

	return nil
}
