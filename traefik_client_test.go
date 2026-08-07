package traefik

import (
	"net/url"
	"testing"
)

func TestClientHasTimeout(t *testing.T) {
	baseUrl, _ := url.Parse("http://traefik.example.org/api")

	for _, insecure := range []bool{false, true} {
		c, err := NewTraefikClient(&TraefikConfig{baseUrl: baseUrl, insecureSkipVerify: insecure})
		if err != nil {
			t.Fatal(err)
		}
		if c.client.Timeout <= 0 {
			t.Errorf("insecureSkipVerify=%v: no timeout, a blackholed connection would stall the refresh loop", insecure)
		}
	}
}
