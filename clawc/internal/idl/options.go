package idl

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/johnsiilver/halfpike"
	lexline "github.com/johnsiilver/halfpike/line"
)

type validateOptArgs func(args []string) error

var fileOptions = map[string]validateOptArgs{
	"NoZeroValueCompression": valNoZeroValueCompression,
}

func valNoZeroValueCompression(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("NoZeroValueCompression takes no arguments")
	}
	return nil
}

var optionsDL = lexline.DecodeList{
	LeftConstraint:  `[`,
	RightConstraint: `]`,
	Separator:       `,`,
	EntryQuote:      `"`,
}

// Options holds a set of options defined.
type Options []Option

func (o *Options) parse(line halfpike.Line, optionsKeyword bool) error {
	if optionsKeyword {
		if err := caseSensitiveCheck("options", line.Items[0].Val); err != nil {
			return err
		}
	}

	lex := lexline.New(line.Raw)

	// Get "options" out of the way.
	if optionsKeyword {
		for {
			i := lex.Next()
			if i.Type == lexline.ItemSpace {
				continue
			}
			if i.Val != "options" {
				return fmt.Errorf("parseOptions() received an line that did not start with 'options' keyword")
			}
			break
		}
	}

	// Find our opening [ character.
	lexline.SkipAllSpaces(lex)
	i := lex.Next()
	if !strings.HasPrefix(i.Val, "[") {
		return fmt.Errorf("'options' keyword followed by invalid characters: %s", i.Val)
	}
	lex.Backup()
	if err := o.parseOptions(lex); err != nil {
		return err
	}
	return nil
}

func (o *Options) parseOptions(lex *lexline.Lexer) error {
	buff := strings.Builder{}
	for {
		i := lex.Next()
		if i.Type == lexline.ItemEOF || i.Type == lexline.ItemEOL {
			break
		}
		buff.WriteString(i.Val)
	}
	reader := strings.NewReader(buff.String())
	buff.Reset()

	foundLeft := false
	foundRight := false
	justFinishedOption := false
	read := 0
	var last rune
	var lastNotSpace rune

	for reader.Len() > 0 {
		if foundRight {
			break
		}

		err := func() error {
			if read == 1 {
				if !foundLeft {
					return fmt.Errorf("bug: something wrong here, didn't find [")
				}
			}
			r, _, _ := reader.ReadRune()
			read++
			defer func() {
				if justFinishedOption {
					justFinishedOption = false
					return
				}
				last = r
				if !unicode.IsSpace(r) {
					lastNotSpace = r
				}
			}()

			switch {
			case r == '[':
				if foundRight {
					return fmt.Errorf("cannot have two [")
				}
				foundLeft = true
				return nil
			case r == ']':
				if !foundLeft {
					return fmt.Errorf("cannot have ] before [")
				}
				if len(*o) == 0 {
					return fmt.Errorf("cannot have 0 options")
				}
				foundRight = true
				return nil
			case r == '(':
				name := buff.String()
				buff.Reset()
				if len(name) == 0 {
					return fmt.Errorf("cannot have an option with no name")
				}
				var opt Option
				var err error
				opt, last, lastNotSpace, err = o.parseArgs(Option{Name: name}, reader)
				if err != nil {
					return err
				}
				*o = append(*o, opt)
				justFinishedOption = true
				return nil
			case r == ',':
				if len(*o) == 0 {
					return fmt.Errorf("found entry separator(,) before an entry")
				}
				if lastNotSpace != ')' {
					return fmt.Errorf("found entry separator(,) after '%c' which isn't correct", lastNotSpace)
				}
				return nil
			case unicode.IsSpace(r):
				switch {
				case unicode.IsSpace(last):
					return nil
				case last == ',', last == ')':
					return nil
				case lastNotSpace == '[':
					return nil
				}
				return fmt.Errorf("cannot have a space character after an option name and before (")
			case unicode.IsLetter(r) || unicode.IsNumber(r):
				buff.WriteRune(r)
				return nil
			default:
				return fmt.Errorf("unsupported character '%c' in option name", r)
			}
		}()
		if err != nil {
			return err
		}
	}

	if reader.Len() == 0 {
		if !foundRight {
			return fmt.Errorf("options didn't have closing ]")
		}
	}

	// Make sure we only have spaces or comments after the close ]
	for reader.Len() > 0 {
		r, _, _ := reader.ReadRune()
		if unicode.IsSpace(r) {
			continue
		}
		if r == '/' {
			r, _, err := reader.ReadRune()
			if err != nil {
				return err
			}
			if r == '/' {
				return nil
			}
		}
		return fmt.Errorf("after option close ], only space or comment is legal")
	}
	return nil
}

func (o *Options) parseArgs(opt Option, reader *strings.Reader) (option Option, last, lastNotSpace rune, err error) {
	foundRight := false
	var inQuote bool
	buff := strings.Builder{}

	for reader.Len() > 0 {
		if foundRight {
			return opt, last, lastNotSpace, nil
		}
		err := func() error {
			r, _, _ := reader.ReadRune()
			defer func() {
				last = r
				if !unicode.IsSpace(r) {
					lastNotSpace = r
				}
			}()

			switch {
			case unicode.IsSpace(r):
				if inQuote {
					buff.WriteRune(r)
				}
				return nil
			case r == ',':
				if inQuote {
					buff.WriteRune(r)
					return nil
				}
				if len(opt.Args) == 0 {
					return fmt.Errorf("cannot have , after %s(", opt.Name)
				}
				if lastNotSpace != '"' {
					return fmt.Errorf("comma in wrong place")
				}
				return nil
			case r == '"':
				if inQuote {
					opt.Args = append(opt.Args, buff.String())
					buff.Reset()
				}
				inQuote = !inQuote
				return nil
			case unicode.IsLetter(r) || unicode.IsNumber(r):
				buff.WriteRune(r)
				return nil
			case r == ')':
				if inQuote {
					return fmt.Errorf("option %s had arg without closing \"", opt.Name)
				}
				foundRight = true
				return nil
			default:
				return fmt.Errorf("unsupported character in arg to %s(): '%c'", opt.Name, r)
			}
		}()
		if err != nil {
			return opt, 0, 0, err
		}
	}
	if !foundRight {
		return opt, 0, 0, fmt.Errorf("option %s had arg without closing )", opt.Name)
	}
	return opt, 0, 0, fmt.Errorf("option %s did not have a closing )", opt.Name)
}
