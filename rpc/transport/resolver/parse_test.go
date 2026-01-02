package resolver

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    Target
		wantErr bool
	}{
		{
			name:   "Success: bare address with port",
			target: "localhost:8080",
			want: Target{
				Scheme:   "passthrough",
				Endpoint: "localhost:8080",
			},
		},
		{
			name:   "Success: bare address without port",
			target: "localhost",
			want: Target{
				Scheme:   "passthrough",
				Endpoint: "localhost",
			},
		},
		{
			name:   "Success: unix socket path",
			target: "/var/run/app.sock",
			want: Target{
				Scheme:   "passthrough",
				Endpoint: "/var/run/app.sock",
			},
		},
		{
			name:   "Success: passthrough with empty authority",
			target: "passthrough:///localhost:8080",
			want: Target{
				Scheme:   "passthrough",
				Endpoint: "localhost:8080",
			},
		},
		{
			name:   "Success: dns with empty authority",
			target: "dns:///myservice.namespace:8080",
			want: Target{
				Scheme:   "dns",
				Endpoint: "myservice.namespace:8080",
			},
		},
		{
			name:   "Success: dns with authority",
			target: "dns://dns-server:53/myservice:8080",
			want: Target{
				Scheme:    "dns",
				Authority: "dns-server:53",
				Endpoint:  "myservice:8080",
			},
		},
		{
			name:   "Success: uppercase scheme normalized",
			target: "DNS:///myservice:8080",
			want: Target{
				Scheme:   "dns",
				Endpoint: "myservice:8080",
			},
		},
		{
			name:   "Success: custom scheme",
			target: "etcd://etcd-server:2379/services/myapp",
			want: Target{
				Scheme:    "etcd",
				Authority: "etcd-server:2379",
				Endpoint:  "services/myapp",
			},
		},
		{
			name:    "Error: empty target",
			target:  "",
			wantErr: true,
		},
		{
			name:    "Error: empty scheme",
			target:  "://authority/endpoint",
			wantErr: true,
		},
		{
			name:    "Error: missing endpoint after authority",
			target:  "dns://dns-server",
			wantErr: true,
		},
		{
			name:    "Error: empty endpoint",
			target:  "dns:///",
			wantErr: true,
		},
	}

	for _, test := range tests {
		got, err := Parse(test.target)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestParse](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestParse](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if diff := pretty.Compare(got, test.want); diff != "" {
			t.Errorf("[TestParse](%s): diff (-got +want):\n%s", test.name, diff)
		}
	}
}

func TestTargetString(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{
			name: "Success: without authority",
			target: Target{
				Scheme:   "dns",
				Endpoint: "myservice:8080",
			},
			want: "dns:///myservice:8080",
		},
		{
			name: "Success: with authority",
			target: Target{
				Scheme:    "dns",
				Authority: "dns-server:53",
				Endpoint:  "myservice:8080",
			},
			want: "dns://dns-server:53/myservice:8080",
		},
	}

	for _, test := range tests {
		got := test.target.String()
		if got != test.want {
			t.Errorf("[TestTargetString](%s): got %q, want %q", test.name, got, test.want)
		}
	}
}
