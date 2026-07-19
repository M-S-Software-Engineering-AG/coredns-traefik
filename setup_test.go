package traefik

import (
	"testing"

	"github.com/coredns/caddy"
)

// TestSetup tests the various things that should be parsed by setup, including
// multiple traefik blocks in one server block.
func TestSetup(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		shouldErr    bool
		numInstances int
	}{
		{"no records", `traefik`, true, 0},
		{"too many args", `traefik https://traefik.example.org/api extra`, true, 0},
		{"cname", `traefik https://traefik.example.org/api {
			cname traefik.example.org
		}`, false, 1},
		{"a record", `traefik https://traefik.example.org/api {
			a 10.0.0.2
		}`, false, 1},
		{"cname and a are mutually exclusive", `traefik https://traefik.example.org/api {
			cname traefik.example.org
			a 10.0.0.2
		}`, true, 0},
		{"unknown property", `traefik https://traefik.example.org/api {
			bogus value
		}`, true, 0},
		{"two blocks", `traefik https://a.example.org/api {
			cname a.example.org
		}
		traefik https://b.example.org/api {
			a 10.0.0.3
		}`, false, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := caddy.NewTestController("dns", tc.input)
			traefik, err := createPlugin(c)
			if tc.shouldErr && err == nil {
				t.Fatalf("Expected error, but got none")
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
			if !tc.shouldErr && len(traefik.Instances) != tc.numInstances {
				t.Fatalf("Expected %d instances, but got %d", tc.numInstances, len(traefik.Instances))
			}
		})
	}
}
