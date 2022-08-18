package globalreplace

import (
	"context"
	"fmt"
	"strings"

	"github.com/johnsiilver/halfpike"
)

// GlobalReplace represents the contents of a global.replace file.
type GlobalReplace struct {
	Replacement string
}

func (l *GlobalReplace) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return l.ParseWith
}

func (l *GlobalReplace) SkipLinesWithComments(p *halfpike.Parser) {
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

func (l *GlobalReplace) ParseWith(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'with ('", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("with", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	l.Replacement = line.Items[1].Val

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf(err.Error())
	}

	return l.FindNext
}

func (l *GlobalReplace) FindNext(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	l.SkipLinesWithComments(p)

	line := p.Next()
	if p.EOF(line) {
		return nil
	}
	return p.Errorf("[Line %d] do not understand this line", line.LineNum)
}

func (l *GlobalReplace) Validate() error {
	// Nothing really to test here.
	return nil
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

func commentOrEOL(line halfpike.Line, from int) error {
	if isComment(line.Items[from]) {
		return nil
	}

	if len(line.Items[from:]) > 1 {
		return fmt.Errorf("got item %q after %q, which was unexpected", halfpike.ItemJoin(line, from, len(line.Items)), halfpike.ItemJoin(line, 0, from))
	}

	return nil
}

func isComment(item halfpike.Item) bool {
	return strings.HasPrefix(item.Val, "//")
}
