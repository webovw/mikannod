package externalnodes

import (
	"strings"
	"testing"
)

func TestImportFullXrayJSONExtractsVLESSAndIgnoresLocalSections(t *testing.T) {
	raw := []byte(`{"remarks":"🇦🇪 Fast","dns":{"servers":["1.1.1.1"]},"inbounds":[{"protocol":"socks"}],"outbounds":[{"protocol":"vless","settings":{"address":"177.3.220.159","port":2083,"id":"ef988d56-9c18-408e-ae1f-a144a332ebce","encryption":"none"},"streamSettings":{"network":"tcp","security":"tls","tlsSettings":{"serverName":"edge.example","fingerprint":"chrome"}},"tag":"proxy"},{"protocol":"freedom","tag":"direct"},{"protocol":"blackhole","tag":"block"}]}`)
	got, err := Import(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d proxies", len(got))
	}
	p := got[0]
	if p.Name != "🇦🇪 Fast" || p.Server != "177.3.220.159" || p.Port != 2083 {
		t.Fatalf("proxy: %+v", p)
	}
	if !strings.HasPrefix(p.URI, "vless://") || !strings.Contains(p.URI, "security=tls") || p.Mihomo["type"] != "vless" {
		t.Fatalf("conversion: %+v", p)
	}
}

func TestImportReality(t *testing.T) {
	raw := []byte(`{"outbounds":[{"protocol":"vless","settings":{"address":"example.org","port":443,"id":"00000000-0000-4000-8000-000000000000"},"streamSettings":{"network":"tcp","security":"reality","realitySettings":{"serverName":"www.example.org","publicKey":"pk","shortId":"sid","fingerprint":"chrome"}}}]}`)
	got, err := Import(raw, "My own name")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "My own name" || !strings.Contains(got[0].URI, "security=reality") || got[0].Mihomo["reality-opts"] == nil {
		t.Fatalf("bad reality import: %+v", got[0])
	}
}

func TestImportRejectsNoUsableOutbound(t *testing.T) {
	if _, err := Import([]byte(`{"outbounds":[{"protocol":"freedom"}]}`), ""); err == nil {
		t.Fatal("expected no-proxy error")
	}
	if _, err := Import([]byte(`not json`), ""); err == nil {
		t.Fatal("expected JSON error")
	}
}
