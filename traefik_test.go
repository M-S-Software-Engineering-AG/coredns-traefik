package traefik

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

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
