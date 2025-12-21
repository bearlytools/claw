package imports

import (
	"github.com/gostdlib/base/context"
	"testing"

	"github.com/johnsiilver/halfpike"
	"github.com/kylelemons/godebug/pretty"
)

func TestClawImports(t *testing.T) {
	tests := []struct {
		desc    string
		content string
		want    ClawImports
		err     bool
	}{
		{
			desc: "Error: missing directive",
			content: `
			{
				path: ""claw/external/imports"
			}
			`,
			err: true,
		},
		{
			desc: "Error: path starts with /",
			content: `
			directive {
				path: "/claw/external/imports"
			}
			`,
			err: true,
		},
		{
			desc: "Error: path ends with /",
			content: `
			directive {
				path: "claw/external/imports/"
			}
			`,
			err: true,
		},
		{
			desc: "Error: path ends with /",
			content: `
			directive {
				path: "claw/external/imports/"
			}
			`,
			err: true,
		},
		{
			desc: "Error: path line has a comma at the end",
			content: `
			directive {
				path: "claw/external/imports/",
			}
			`,
			err: true,
		},
		{
			desc: "Success",
			content: `
			// Comment here
			directive {
				path: "claw/external/imports"
			}
			// Comment here
			`,
			want: ClawImports{
				Directive: Directive{Path: "claw/external/imports"},
			},
		},
	}

	for _, test := range tests {
		got := ClawImports{}
		err := halfpike.Parse(context.Background(), test.content, &got)
		switch {
		case err == nil && test.err:
			t.Errorf("TestClawImports(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestClawImports(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("TestClawImports(%s): -want/+got:\n%s", test.desc, diff)
		}
	}
}
