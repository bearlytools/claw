package imports

import (
	"github.com/gostdlib/base/context"
	"embed"
	"testing"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/johnsiilver/halfpike"
	"github.com/kylelemons/godebug/pretty"
)

//go:embed testing/localreplace/*
var embedFS embed.FS

func TestLocalReplace(t *testing.T) {
	tests := []struct {
		desc     string
		filePath string
		want     LocalReplace
		err      bool
	}{
		{
			desc:     "Missing source",
			filePath: "testing/localreplace/missing_source.replace",
			err:      true,
		},
		{
			desc:     "Missing replacement",
			filePath: "testing/localreplace/missing_replace.replace",
			err:      true,
		},
		{
			desc:     "Bad replace, not all lowercase",
			filePath: "testing/localreplace/bad_capital.replace",
			err:      true,
		},
		{
			desc:     "Missing ( in replace (",
			filePath: "testing/localreplace/missing(.replace",
			err:      true,
		},
		{
			desc:     "Missing ) in replace ()",
			filePath: "testing/localreplace/missing).replace",
			err:      true,
		},
		{
			desc:     "Bad comment like //comment instead of // comment",
			filePath: "testing/localreplace/bad_comment.replace",
			err:      true,
		},
		{
			desc:     "Success",
			filePath: "testing/localreplace/good.replace",
			want: LocalReplace{
				Replace: []Replace{
					{FromPath: "github.com/johnsiilver/whatever", ToPath: "../some/directory/somewhere"},
					{FromPath: "github.com/djustice/something", ToPath: "../some/directory/somewhere/else"},
				},
			},
		},
	}

	for _, test := range tests {
		b, err := embedFS.ReadFile(test.filePath)
		if err != nil {
			panic(err)
		}

		got := LocalReplace{testing: true}
		err = halfpike.Parse(context.Background(), conversions.ByteSlice2String(b), &got)
		switch {
		case err == nil && test.err:
			t.Errorf("TestLocalReplace(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestLocalReplace(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		if diff := pretty.Compare(test.want.Replace, got.Replace); diff != "" {
			t.Errorf("TestLocalReplace(%s): -want/+got:\n%s", test.desc, diff)
		}
	}
}
