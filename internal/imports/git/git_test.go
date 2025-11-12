package git

import (
	"testing"
)

func TestVersionFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr bool
	}{
		{
			name:  "version with v prefix",
			input: "v1.2.3",
			want:  Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "version without v prefix",
			input: "1.2.3",
			want:  Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "version with zeros",
			input: "v0.0.0",
			want:  Version{Major: 0, Minor: 0, Patch: 0},
		},
		{
			name:  "large version numbers",
			input: "v10.20.30",
			want:  Version{Major: 10, Minor: 20, Patch: 30},
		},
		{
			name:    "invalid format - missing patch",
			input:   "v1.2",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "v1.2.3.4",
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric major",
			input:   "va.2.3",
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric minor",
			input:   "v1.b.3",
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric patch",
			input:   "v1.2.c",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Version{}
			err := v.FromString(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("FromString() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("FromString() unexpected error: %v", err)
				return
			}

			if v.Major != tt.want.Major || v.Minor != tt.want.Minor || v.Patch != tt.want.Patch {
				t.Errorf("FromString() = {Major: %d, Minor: %d, Patch: %d}, want {Major: %d, Minor: %d, Patch: %d}",
					v.Major, v.Minor, v.Patch, tt.want.Major, tt.want.Minor, tt.want.Patch)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		v1   Version
		v2   Version
		want int
	}{
		{
			name: "v1 < v2 by major",
			v1:   Version{Major: 1, Minor: 0, Patch: 0},
			v2:   Version{Major: 2, Minor: 0, Patch: 0},
			want: -1,
		},
		{
			name: "v1 > v2 by major",
			v1:   Version{Major: 2, Minor: 0, Patch: 0},
			v2:   Version{Major: 1, Minor: 0, Patch: 0},
			want: 1,
		},
		{
			name: "v1 < v2 by minor",
			v1:   Version{Major: 1, Minor: 1, Patch: 0},
			v2:   Version{Major: 1, Minor: 2, Patch: 0},
			want: -1,
		},
		{
			name: "v1 > v2 by minor",
			v1:   Version{Major: 1, Minor: 2, Patch: 0},
			v2:   Version{Major: 1, Minor: 1, Patch: 0},
			want: 1,
		},
		{
			name: "v1 < v2 by patch",
			v1:   Version{Major: 1, Minor: 2, Patch: 3},
			v2:   Version{Major: 1, Minor: 2, Patch: 4},
			want: -1,
		},
		{
			name: "v1 > v2 by patch",
			v1:   Version{Major: 1, Minor: 2, Patch: 4},
			v2:   Version{Major: 1, Minor: 2, Patch: 3},
			want: 1,
		},
		{
			name: "v1 == v2",
			v1:   Version{Major: 1, Minor: 2, Patch: 3},
			v2:   Version{Major: 1, Minor: 2, Patch: 3},
			want: 0,
		},
		{
			name: "zero versions equal",
			v1:   Version{Major: 0, Minor: 0, Patch: 0},
			v2:   Version{Major: 0, Minor: 0, Patch: 0},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v1.Compare(tt.v2)
			if got != tt.want {
				t.Errorf("Version.Compare() = %d, want %d", got, tt.want)
			}
		})
	}
}
