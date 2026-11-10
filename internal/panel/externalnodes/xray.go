// Package externalnodes imports client-side Xray JSON outbounds as subscription proxies.
// It deliberately ignores inbounds, DNS, routing, policy and server-side settings: only
// the outbound connection parameters are meaningful on a subscriber's device.
package externalnodes

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"mikan/internal/hostname"
)

type Document struct {
	Outbounds []Outbound `json:"outbounds"`
	Remarks   string     `json:"remarks"`
}
type Outbound struct {
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
	Settings struct {
		Address    string `json:"address"`
		Port       int    `json:"port"`
		ID         string `json:"id"`
		Password   string `json:"password"`
		Encryption string `json:"encryption"`
		Flow       string `json:"flow"`
		Method     string `json:"method"`
		VNext      []struct {
			Address string `json:"address"`
			Port    int    `json:"port"`
			Users   []struct {
				ID       string `json:"id"`
				Security string `json:"security"`
				AlterID  int    `json:"alterId"`
				Level    int    `json:"level"`
				Flow     string `json:"flow"`
			} `json:"users"`
		} `json:"vnext"`
		Servers []struct {
			Address  string `json:"address"`
			Port     int    `json:"port"`
			Password string `json:"password"`
			Method   string `json:"method"`
		} `json:"servers"`
	} `json:"settings"`
	Stream struct {
		Network  string `json:"network"`
		Security string `json:"security"`
		TLS      struct {
			ServerName    string   `json:"serverName"`
			Fingerprint   string   `json:"fingerprint"`
			ALPN          []string `json:"alpn"`
			AllowInsecure bool     `json:"allowInsecure"`
		} `json:"tlsSettings"`
		Reality struct {
			ServerName  string `json:"serverName"`
			Fingerprint string `json:"fingerprint"`
			PublicKey   string `json:"publicKey"`
			ShortID     string `json:"shortId"`
			SpiderX     string `json:"spiderX"`
		} `json:"realitySettings"`
		WS struct {
			Path    string            `json:"path"`
			Headers map[string]string `json:"headers"`
		} `json:"wsSettings"`
		GRPC struct {
			ServiceName string `json:"serviceName"`
			MultiMode   bool   `json:"multiMode"`
		} `json:"grpcSettings"`
		TCP struct {
			Header struct {
				Type string `json:"type"`
			} `json:"header"`
		} `json:"tcpSettings"`
		XHTTP struct {
			Path string `json:"path"`
			Host string `json:"host"`
			Mode string `json:"mode"`
		} `json:"xhttpSettings"`
		HTTPUpgrade struct {
			Path string `json:"path"`
			Host string `json:"host"`
		} `json:"httpupgradeSettings"`
	} `json:"streamSettings"`
}

type Proxy struct {
	Tag      string         `json:"tag,omitempty"`
	Name     string         `json:"name"`
	Protocol string         `json:"protocol"`
	Server   string         `json:"server"`
	Port     int            `json:"port"`
	URI      string         `json:"uri"`
	Mihomo   map[string]any `json:"mihomo"`
}

func Import(raw []byte, customName string) ([]Proxy, error) {
	var doc Document
	if len(raw) == 0 || len(raw) > 2<<20 {
		return nil, errors.New("JSON пустой или превышает 2 МБ")
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("некорректный JSON: %w", err)
	}
	if len(doc.Outbounds) == 0 {
		return nil, errors.New("в JSON нет массива outbounds")
	}
	var out []Proxy
	for _, ob := range doc.Outbounds {
		p, err := convert(ob)
		if err != nil { // direct/block/dns and unsupported outbounds are not VPN servers
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errors.New("не найдено поддерживаемых VPN outbounds (VLESS, VMess, Trojan, Shadowsocks)")
	}
	for i := range out {
		name := strings.TrimSpace(customName)
		if name == "" {
			name = strings.TrimSpace(doc.Remarks)
		}
		if name == "" {
			name = strings.TrimSpace(out[i].Tag)
		}
		if name == "" {
			name = out[i].Server
		}
		if len(out) > 1 {
			tag := strings.TrimSpace(out[i].Tag)
			if tag != "" {
				name += " · " + tag
			} else {
				name += " " + strconv.Itoa(i+1)
			}
		}
		out[i].Name = name
		out[i].Mihomo["name"] = name
	}
	return out, nil
}

func convert(o Outbound) (Proxy, error) {
	typ := strings.ToLower(o.Protocol)
	s := o.Settings
	stream := o.Stream
	server, port := s.Address, s.Port
	user := s.ID
	switch typ {
	case "vmess", "vless":
		if len(s.VNext) > 0 {
			server = s.VNext[0].Address
			port = s.VNext[0].Port
			if len(s.VNext[0].Users) > 0 {
				user = s.VNext[0].Users[0].ID
				if s.VNext[0].Users[0].Flow != "" {
					s.Flow = s.VNext[0].Users[0].Flow
				}
			}
		}
	case "trojan":
		if len(s.Servers) > 0 {
			server = s.Servers[0].Address
			port = s.Servers[0].Port
			s.Password = s.Servers[0].Password
		}
	case "shadowsocks":
		if len(s.Servers) > 0 {
			server = s.Servers[0].Address
			port = s.Servers[0].Port
			s.Method = s.Servers[0].Method
			s.Password = s.Servers[0].Password
		}
	case "vless", "trojan":
	default:
		return Proxy{}, errors.New("unsupported outbound")
	}
	if !hostname.Valid(server) || port < 1 || port > 65535 {
		return Proxy{}, errors.New("missing server or port")
	}
	y := map[string]any{"name": o.Tag, "type": typ, "server": server, "port": port}
	q := url.Values{}
	switch typ {
	case "vless":
		if user == "" {
			return Proxy{}, errors.New("missing UUID")
		}
		y["uuid"] = user
		q.Set("encryption", "none")
		if s.Flow != "" {
			y["flow"] = s.Flow
			q.Set("flow", s.Flow)
		}
	case "trojan":
		if s.Password == "" {
			s.Password = user
		}
		if s.Password == "" {
			return Proxy{}, errors.New("missing password")
		}
		y["password"] = s.Password
	case "vmess":
		if user == "" {
			return Proxy{}, errors.New("missing UUID")
		}
		alter := 0
		sec := "auto"
		if len(s.VNext) > 0 && len(s.VNext[0].Users) > 0 {
			alter = s.VNext[0].Users[0].AlterID
			if s.VNext[0].Users[0].Security != "" {
				sec = s.VNext[0].Users[0].Security
			}
		}
		y["uuid"] = user
		y["alterId"] = alter
		y["cipher"] = sec
	case "shadowsocks":
		if s.Method == "" || s.Password == "" {
			return Proxy{}, errors.New("missing method/password")
		}
		y["cipher"] = s.Method
		y["password"] = s.Password
	}
	if stream.Network == "" {
		stream.Network = "tcp"
	}
	switch stream.Network {
	case "tcp":
		y["network"] = "tcp"
	case "ws":
		y["network"] = "ws"
		y["ws-opts"] = map[string]any{"path": defaultPath(stream.WS.Path), "headers": stream.WS.Headers}
	case "grpc":
		y["network"] = "grpc"
		y["grpc-opts"] = map[string]any{"grpc-service-name": stream.GRPC.ServiceName}
	case "httpupgrade":
		y["network"] = "ws"
		y["ws-opts"] = map[string]any{"path": defaultPath(stream.HTTPUpgrade.Path), "headers": map[string]string{"Host": stream.HTTPUpgrade.Host}}
	case "xhttp":
		y["network"] = "xhttp"
		y["xhttp-opts"] = map[string]any{"path": defaultPath(stream.XHTTP.Path), "host": stream.XHTTP.Host, "mode": stream.XHTTP.Mode}
	default:
		return Proxy{}, fmt.Errorf("unsupported transport %q", stream.Network)
	}
	tlsOn := stream.Security == "tls" || stream.Security == "reality"
	sni := stream.TLS.ServerName
	fp := stream.TLS.Fingerprint
	if stream.Security == "reality" {
		if stream.Reality.PublicKey == "" || stream.Reality.ShortID == "" || stream.Reality.ServerName == "" {
			return Proxy{}, errors.New("incomplete REALITY settings")
		}
		sni = stream.Reality.ServerName
		fp = stream.Reality.Fingerprint
		y["tls"] = true
		y["servername"] = sni
		y["reality-opts"] = map[string]any{"public-key": stream.Reality.PublicKey, "short-id": stream.Reality.ShortID}
		if fp != "" {
			y["client-fingerprint"] = fp
		}
		q.Set("security", "reality")
		q.Set("sni", sni)
		q.Set("pbk", stream.Reality.PublicKey)
		q.Set("sid", stream.Reality.ShortID)
		if stream.Reality.SpiderX != "" {
			q.Set("spx", stream.Reality.SpiderX)
		}
	} else if tlsOn {
		y["tls"] = true
		if sni != "" {
			y["servername"] = sni
			q.Set("sni", sni)
		}
		if fp != "" {
			y["client-fingerprint"] = fp
		}
		if len(stream.TLS.ALPN) > 0 {
			y["alpn"] = stream.TLS.ALPN
		}
		q.Set("security", "tls")
	}
	if stream.Network == "ws" {
		q.Set("type", "ws")
		q.Set("path", defaultPath(stream.WS.Path))
		if h := stream.WS.Headers["Host"]; h != "" {
			q.Set("host", h)
		}
	} else if stream.Network == "grpc" {
		q.Set("type", "grpc")
		q.Set("serviceName", stream.GRPC.ServiceName)
	} else if stream.Network == "xhttp" {
		q.Set("type", "xhttp")
		q.Set("path", defaultPath(stream.XHTTP.Path))
		if stream.XHTTP.Host != "" {
			q.Set("host", stream.XHTTP.Host)
		}
		if stream.XHTTP.Mode != "" {
			q.Set("mode", stream.XHTTP.Mode)
		}
	} else if stream.Network == "httpupgrade" {
		q.Set("type", "httpupgrade")
		q.Set("path", defaultPath(stream.HTTPUpgrade.Path))
		if stream.HTTPUpgrade.Host != "" {
			q.Set("host", stream.HTTPUpgrade.Host)
		}
	} else {
		q.Set("type", "tcp")
	}
	name := strings.TrimSpace(o.Tag)
	if name == "" {
		name = server
	}
	var uri string
	switch typ {
	case "vless":
		uri = "vless://" + url.User(user).String() + "@" + server + ":" + strconv.Itoa(port) + "?" + q.Encode() + "#" + url.QueryEscape(name)
	case "trojan":
		uri = "trojan://" + url.UserPassword(s.Password, "").String() + "@" + server + ":" + strconv.Itoa(port) + "?" + q.Encode() + "#" + url.QueryEscape(name)
	case "vmess":
		raw, _ := json.Marshal(map[string]any{"v": "2", "ps": name, "add": server, "port": port, "id": user, "aid": y["alterId"], "scy": y["cipher"], "net": stream.Network, "type": "none", "host": stream.WS.Headers["Host"], "path": stream.WS.Path, "tls": map[bool]string{true: "tls", false: ""}[tlsOn], "sni": sni, "fp": fp})
		uri = "vmess://" + base64.StdEncoding.EncodeToString(raw)
	case "shadowsocks":
		cred := base64.RawURLEncoding.EncodeToString([]byte(s.Method + ":" + s.Password))
		uri = "ss://" + cred + "@" + server + ":" + strconv.Itoa(port) + "#" + url.QueryEscape(name)
	}
	return Proxy{Tag: o.Tag, Protocol: typ, Server: server, Port: port, URI: uri, Mihomo: y}, nil
}
func defaultPath(s string) string {
	if s == "" {
		return "/"
	}
	return s
}
