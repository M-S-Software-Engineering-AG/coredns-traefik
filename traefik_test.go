package traefik

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

// fakeClient returns a canned router list, standing in for a live Traefik API.
type fakeClient struct{ routers []HttpRouter }

func (f *fakeClient) GetHttpRouters() (*[]HttpRouter, error) { return &f.routers, nil }

// TestRetention proves a discovered host keeps being served for its retention
// window after it disappears from Traefik, is evicted once the window elapses,
// and that reappearing within the window cancels the pending eviction.
func TestRetention(t *testing.T) {
	baseURL, _ := url.Parse("http://traefik.example.org/api")
	ips := []net.IP{net.ParseIP("10.0.0.2")}
	fc := &fakeClient{}
	inst := &traefikInstance{
		config: &TraefikConfig{
			baseUrl:     baseURL,
			hostMatcher: regexp.MustCompile(`Host(SNI)?\(` + "`([^`]+)`" + `\)`),
			a:           &ips,
			ttl:         30,
			retention:   300,
			filter:      []string{"example.org."},
		},
		client:   fc,
		mappings: make(TraefikConfigEntryMap),
		evictAt:  make(map[string]time.Time),
	}
	const host = "whoami.example.org"
	router := HttpRouter{Rule: "Host(`" + host + "`)"}

	// Discovered while advertised.
	fc.routers = []HttpRouter{router}
	inst.refresh(false)
	if inst.getEntry(host) == nil {
		t.Fatal("host should be served once discovered")
	}

	// Gone from Traefik, but within the retention window it is still served.
	fc.routers = nil
	inst.refresh(false)
	if inst.getEntry(host) == nil {
		t.Fatal("host should linger during the retention window")
	}
	if _, pending := inst.evictAt[host]; !pending {
		t.Fatal("eviction should be pending")
	}

	// Reappears within the window: pending eviction is cancelled.
	fc.routers = []HttpRouter{router}
	inst.refresh(false)
	if _, pending := inst.evictAt[host]; pending {
		t.Fatal("eviction should be cancelled when the host reappears")
	}

	// Gone again, then the window elapses: evicted.
	fc.routers = nil
	inst.refresh(false)
	inst.evictAt[host] = time.Now().Add(-time.Second) // force the deadline into the past
	inst.refresh(false)
	if inst.getEntry(host) != nil {
		t.Fatal("host should be evicted after the retention window elapses")
	}

	// retention 0 evicts on the first refresh that no longer sees the host.
	inst.config.retention = 0
	fc.routers = []HttpRouter{router}
	inst.refresh(false)
	fc.routers = nil
	inst.refresh(false)
	if inst.getEntry(host) != nil {
		t.Fatal("with retention 0 the host should be evicted immediately")
	}
}

// TestMatchesFilter checks which discovered hosts a filter admits.
func TestMatchesFilter(t *testing.T) {
	cfg := &TraefikConfig{filter: []string{"example.org."}}
	cases := map[string]bool{
		"gitea.example.org":    true,
		"example.org":          true,
		"deep.sub.example.org": true,
		"example.net":          false,
		"notexample.org":       false,
	}
	for host, want := range cases {
		if got := cfg.matchesFilter(host); got != want {
			t.Errorf("matchesFilter(%q) = %v, want %v", host, got, want)
		}
	}
}

// TestServeDNSMultipleInstances proves a single Traefik handler answers for
// hosts discovered across more than one configured block.
func TestServeDNSMultipleInstances(t *testing.T) {
	cnameTarget := "traefik-a.example.org."
	instA := &traefikInstance{
		config: &TraefikConfig{ttl: 30},
		mappings: TraefikConfigEntryMap{
			"a.example.org": &TraefikConfigEntry{cname: &cnameTarget, ttl: 30},
		},
	}
	ips := []net.IP{net.ParseIP("10.0.0.5")}
	instB := &traefikInstance{
		config: &TraefikConfig{ttl: 30},
		mappings: TraefikConfigEntryMap{
			"b.example.org": &TraefikConfigEntry{a: &ips, ttl: 30},
		},
	}

	tk := &Traefik{
		Next:      test.ErrorHandler(),
		Instances: []*traefikInstance{instA, instB},
	}

	tests := []struct {
		qname  string
		qtype  uint16
		rcode  int
		answer int
	}{
		{"a.example.org.", dns.TypeA, dns.RcodeSuccess, 1}, // CNAME from instance A
		{"b.example.org.", dns.TypeA, dns.RcodeSuccess, 1}, // A from instance B
		{"unknown.example.org.", dns.TypeA, dns.RcodeNameError, 0},
	}

	for _, tc := range tests {
		r := new(dns.Msg)
		r.SetQuestion(tc.qname, tc.qtype)
		rec := dnstest.NewRecorder(&test.ResponseWriter{})

		if _, err := tk.ServeDNS(context.TODO(), rec, r); err != nil {
			t.Fatalf("%s: expected no error, got %v", tc.qname, err)
		}
		if rec.Msg.Rcode != tc.rcode {
			t.Errorf("%s: expected rcode %d, got %d", tc.qname, tc.rcode, rec.Msg.Rcode)
		}
		if len(rec.Msg.Answer) != tc.answer {
			t.Errorf("%s: expected %d answers, got %d", tc.qname, tc.answer, len(rec.Msg.Answer))
		}
	}
}
