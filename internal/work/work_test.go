package work

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Work
		wantErr bool
	}{
		{
			name: "Success: basic claw.work file",
			content: `repo github.com/bearlytools/claw

vendorDir claw_vendor
`,
			want: &Work{
				Repo:      "github.com/bearlytools/claw",
				VendorDir: "claw_vendor",
			},
			wantErr: false,
		},
		{
			name: "Success: with comments",
			content: `// This is a claw.work file
repo github.com/example/project

// Vendor directory configuration
vendorDir vendor
`,
			want: &Work{
				Repo:      "github.com/example/project",
				VendorDir: "vendor",
			},
			wantErr: false,
		},
		{
			name: "Success: inline comments",
			content: `repo github.com/example/project // repo path
vendorDir my_vendor // custom vendor dir
`,
			want: &Work{
				Repo:      "github.com/example/project",
				VendorDir: "my_vendor",
			},
			wantErr: false,
		},
		{
			name: "Success: nested vendor directory",
			content: `repo github.com/example/project
vendorDir build/vendor
`,
			want: &Work{
				Repo:      "github.com/example/project",
				VendorDir: "build/vendor",
			},
			wantErr: false,
		},
		{
			name:    "Error: missing repo",
			content: `vendorDir claw_vendor`,
			wantErr: true,
		},
		{
			name: "Error: missing vendorDir",
			content: `repo github.com/example/project
`,
			wantErr: true,
		},
		{
			name:    "Error: empty file",
			content: ``,
			wantErr: true,
		},
		{
			name: "Error: duplicate vendorDir",
			content: `repo github.com/example/project
vendorDir vendor1
vendorDir vendor2
`,
			wantErr: true,
		},
		{
			name: "Error: unknown directive",
			content: `repo github.com/example/project
vendorDir vendor
unknown something
`,
			wantErr: true,
		},
		{
			name: "Error: absolute vendorDir path",
			content: `repo github.com/example/project
vendorDir /absolute/path
`,
			wantErr: true,
		},
		{
			name: "Error: repo with double slashes",
			content: `repo github.com//example/project
vendorDir vendor
`,
			wantErr: true,
		},
		{
			name: "Error: wrong case vendordir",
			content: `repo github.com/example/project
vendordir vendor
`,
			wantErr: true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		got, err := Parse(ctx, []byte(test.content))

		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestParse(%s)]: got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestParse(%s)]: got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("[TestParse(%s)]: -want +got:\n%s", test.name, diff)
		}
	}
}

func TestValidateModuleInRepo(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		repoPath   string
		wantErr    bool
	}{
		{
			name:       "Success: module equals repo",
			modulePath: "github.com/bearlytools/claw",
			repoPath:   "github.com/bearlytools/claw",
			wantErr:    false,
		},
		{
			name:       "Success: module is child of repo",
			modulePath: "github.com/bearlytools/claw/internal/work",
			repoPath:   "github.com/bearlytools/claw",
			wantErr:    false,
		},
		{
			name:       "Success: module is direct child",
			modulePath: "github.com/bearlytools/claw/sub",
			repoPath:   "github.com/bearlytools/claw",
			wantErr:    false,
		},
		{
			name:       "Error: module not related to repo",
			modulePath: "github.com/other/project",
			repoPath:   "github.com/bearlytools/claw",
			wantErr:    true,
		},
		{
			name:       "Error: module is partial prefix match",
			modulePath: "github.com/bearlytools/clawsome",
			repoPath:   "github.com/bearlytools/claw",
			wantErr:    true,
		},
		{
			name:       "Error: repo is child of module (reversed)",
			modulePath: "github.com/bearlytools",
			repoPath:   "github.com/bearlytools/claw",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		err := ValidateModuleInRepo(test.modulePath, test.repoPath)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestValidateModuleInRepo(%s)]: got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestValidateModuleInRepo(%s)]: got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}
	}
}

func TestWorkValidate(t *testing.T) {
	tests := []struct {
		name    string
		work    Work
		wantErr bool
	}{
		{
			name: "Success: valid work",
			work: Work{
				Repo:      "github.com/example/project",
				VendorDir: "vendor",
			},
			wantErr: false,
		},
		{
			name: "Error: empty repo",
			work: Work{
				Repo:      "",
				VendorDir: "vendor",
			},
			wantErr: true,
		},
		{
			name: "Error: empty vendorDir",
			work: Work{
				Repo:      "github.com/example/project",
				VendorDir: "",
			},
			wantErr: true,
		},
		{
			name: "Error: absolute vendorDir",
			work: Work{
				Repo:      "github.com/example/project",
				VendorDir: "/absolute/path",
			},
			wantErr: true,
		},
		{
			name: "Error: repo with double slashes",
			work: Work{
				Repo:      "github.com//example",
				VendorDir: "vendor",
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := test.work.Validate()

		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestWorkValidate(%s)]: got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestWorkValidate(%s)]: got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}
	}
}
