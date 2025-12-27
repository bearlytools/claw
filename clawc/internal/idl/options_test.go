package idl

import (
	"github.com/gostdlib/base/context"
	"testing"

	"github.com/johnsiilver/halfpike"
	"github.com/kylelemons/godebug/pretty"
)

func TestOptions(t *testing.T) {
	tests := []struct {
		desc            string
		line            string
		parseOptKeyword bool
		want            Options
		err             bool
	}{
		{
			desc:            "Missing options keyword",
			line:            `[doNotDisturb("arg1", "arg2"), noCompressZeroValues()] // comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
			err: true,
		},
		{
			desc:            "Options is capitalized",
			line:            `Options [doNotDisturb("arg1", "arg2"), noCompressZeroValues()] // comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
			err: true,
		},
		{
			desc:            "Missing [",
			line:            `pptions doNotDisturb("arg1", "arg2"), noCompressZeroValues()] // comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
			err: true,
		},
		{
			desc:            "Missing ]",
			line:            `options [doNotDisturb("arg1", "arg2"), noCompressZeroValues() // comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
			err: true,
		},
		{
			desc:            "Comment is malformed",
			line:            `options [doNotDisturb("arg1", "arg2"), noCompressZeroValues()] / comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
			err: true,
		},
		{
			desc:            "Missing comma between options",
			line:            `options [doNotDisturb("arg1", "arg2") noCompressZeroValues()] / comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
			err: true,
		},
		{
			desc:            "Success",
			line:            `options [doNotDisturb("arg1", "arg2"), noCompressZeroValues()] // comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
		},
		{
			desc:            "Success",
			line:            `options [ doNotDisturb( "arg1" , "arg2" ), noCompressZeroValues()] // comment` + "\n",
			parseOptKeyword: true,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
		},
		{
			desc:            "Success",
			line:            `[ doNotDisturb( "arg1" , "arg2" ), noCompressZeroValues() ]// comment` + "\n",
			parseOptKeyword: false,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", nil}},
		},
		{
			desc:            "Success",
			line:            `[ doNotDisturb( "arg1" , "arg2" ), noCompressZeroValues("arg3" , "arg4",) ]// comment` + "\n",
			parseOptKeyword: false,
			want: Options{
				Option{"doNotDisturb", []string{"arg1", "arg2"}},
				Option{"noCompressZeroValues", []string{"arg3", "arg4"}}},
		},
	}

	for _, test := range tests {
		opt := Options{}

		l := &lineLexer{}
		halfpike.Parse(context.Background(), test.line, l)

		err := opt.parse(l.line, test.parseOptKeyword)
		switch {
		case err == nil && test.err:
			t.Errorf("TestOptions(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestOptions(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		if diff := pretty.Compare(test.want, opt); diff != "" {
			t.Errorf("TestOptions(%s): -want/+got:\n%s", test.desc, diff)
		}
	}
}

func TestValIsSet(t *testing.T) {
	if err := valIsSet(nil); err != nil {
		t.Fatalf("TestValIsSet: nil argument gave unexpected error: %s", err)
	}
	if err := valIsSet([]string{}); err != nil {
		t.Fatalf("TestValIsSet: []string{} argument gave unexpected error: %s", err)
	}
	if err := valIsSet([]string{"hello"}); err == nil {
		t.Fatalf("TestValIsSet: []string{\"hello\"} argument gave did not give error")
	}
}
