package subs

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestExternalNodesJoinXrayAndMihomoSubscriptionsAndAuto(t *testing.T) {
	var y map[string]any
	if err := json.Unmarshal([]byte(`{"name":"🇦🇪 UAE","type":"vless","server":"edge.example","port":443,"uuid":"00000000-0000-4000-8000-000000000000","tls":true,"servername":"edge.example"}`), &y); err != nil {
		t.Fatal(err)
	}
	p := Profile{External: []ExternalProxy{{Name: "🇦🇪 UAE", URI: "vless://id@edge.example:443?security=tls#UAE", Mihomo: y, Auto: true}}}
	links, err := URIs(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(links, "vless://id@edge.example:443") {
		t.Fatalf("external URI missing: %s", links)
	}
	body, err := Mihomo(p, Groups{}, RoutingAll)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err = yaml.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	proxies := cfg["proxies"].([]any)
	if len(proxies) != 1 {
		t.Fatalf("proxies: %#v", proxies)
	}
	groups := cfg["proxy-groups"].([]any)
	auto := groups[1].(map[string]any)
	if !reflect.DeepEqual(auto["proxies"], []any{"🇦🇪 UAE"}) {
		t.Fatalf("Auto group: %#v", auto["proxies"])
	}
}

func TestExternalNodeCanBeExcludedFromAuto(t *testing.T) {
	y := map[string]any{"name": "External", "type": "vless", "server": "edge.example", "port": 443, "uuid": "id"}
	p := Profile{External: []ExternalProxy{{Name: "External", URI: "vless://id@edge.example:443#External", Mihomo: y, Auto: false}}}
	body, err := Mihomo(p, Groups{}, RoutingAll)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err = yaml.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	groups := cfg["proxy-groups"].([]any)
	auto := groups[1].(map[string]any)
	if !reflect.DeepEqual(auto["proxies"], []any{"DIRECT"}) {
		t.Fatalf("disabled external server leaked into Auto: %#v", auto["proxies"])
	}
}
