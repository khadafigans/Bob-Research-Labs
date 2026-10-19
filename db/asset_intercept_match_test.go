package db

import "testing"

func rule(kind, pattern string, enabled bool) AssetInterceptRule {
	return AssetInterceptRule{Kind: kind, Pattern: pattern, Enabled: enabled}
}

func TestMatchAssetInterceptRules(t *testing.T) {
	cases := []struct {
		name    string
		rules   []AssetInterceptRule
		domains []string
		ips     []string
		urls    []string
		want    bool
		wantVal string
	}{
		{"", []AssetInterceptRule{rule("fuzzy_domain", ".gov.cn", true)},
			[]string{"www.beijing.gov.cn"}, nil, nil, true, "www.beijing.gov.cn"},
		{"", []AssetInterceptRule{rule("fuzzy_domain", ".edu", true)},
			[]string{"mit.edu"}, nil, nil, true, "mit.edu"},
		{"", []AssetInterceptRule{rule("exact_domain", "Example.com", true)},
			[]string{"example.com"}, nil, nil, true, "example.com"},
		{"", []AssetInterceptRule{rule("exact_domain", "example.com", true)},
			[]string{"a.example.com"}, nil, nil, false, ""},
		{"IP", []AssetInterceptRule{rule("exact_ip", "203.0.113.5", true)},
			nil, []string{"203.0.113.5"}, nil, true, "203.0.113.5"},
		{"IP", []AssetInterceptRule{rule("fuzzy_ip", "203.0.113.", true)},
			nil, []string{"203.0.113.99"}, nil, true, "203.0.113.99"},
		{"CIDR ", []AssetInterceptRule{rule("cidr", "192.168.0.0/16", true)},
			nil, []string{"192.168.5.20"}, nil, true, "192.168.5.20"},
		{"CIDR ", []AssetInterceptRule{rule("cidr", "192.168.0.0/16", true)},
			nil, []string{"10.0.0.1"}, nil, false, ""},
		{"URL", []AssetInterceptRule{rule("exact_url", "https://a.gov.cn/login", true)},
			nil, nil, []string{"https://a.gov.cn/login"}, true, "https://a.gov.cn/login"},
		{"URL", []AssetInterceptRule{rule("fuzzy_url", "/admin", true)},
			nil, nil, []string{"https://x.com/admin/panel"}, true, "https://x.com/admin/panel"},
		{"", []AssetInterceptRule{rule("fuzzy_domain", ".gov.cn", false)},
			[]string{"www.gov.cn"}, nil, nil, false, ""},
		{"", nil, []string{"www.gov.cn"}, nil, nil, false, ""},
		{"pattern", []AssetInterceptRule{rule("fuzzy_domain", "  ", true)},
			[]string{"www.gov.cn"}, nil, nil, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, val, ok := MatchAssetInterceptRules(c.rules, c.domains, c.ips, c.urls)
			if ok != c.want {
				t.Fatalf(" = %v,  %v (rule=%+v)", ok, c.want, r)
			}
			if ok && val != c.wantVal {
				t.Fatalf(" = %q,  %q", val, c.wantVal)
			}
		})
	}
}

func TestEvaluateAssetGate(t *testing.T) {
	block := []AssetInterceptRule{rule("fuzzy_domain", ".gov.cn", true)}
	allow := []AssetInterceptRule{rule("fuzzy_domain", "example.com", true)}

	if d := EvaluateAssetGate(block, allow, []string{"www.gov.cn"}, nil, nil); d.Allowed {
		t.Fatal("")
	}

	d := EvaluateAssetGate(block, allow, []string{"foo.other.com"}, nil, nil)
	if d.Allowed {
		t.Fatal("")
	}
	if d.Reason == "" {
		t.Fatal("")
	}

	if d := EvaluateAssetGate(block, allow, []string{"api.example.com"}, nil, nil); !d.Allowed {
		t.Fatal("")
	}

	if d := EvaluateAssetGate(block, nil, []string{"foo.other.com"}, nil, nil); !d.Allowed {
		t.Fatal("")
	}

	disabledAllow := []AssetInterceptRule{rule("fuzzy_domain", "example.com", false)}
	if d := EvaluateAssetGate(nil, disabledAllow, []string{"foo.other.com"}, nil, nil); !d.Allowed {
		t.Fatal("")
	}

	if d := EvaluateAssetGate(
		[]AssetInterceptRule{rule("fuzzy_domain", ".gov.cn", true)},
		[]AssetInterceptRule{rule("fuzzy_domain", ".gov.cn", true)},
		[]string{"www.gov.cn"}, nil, nil,
	); d.Allowed {
		t.Fatal("")
	}
}

func TestAssetInterceptCandidates(t *testing.T) {
	//  URL ：host ， fuzzy_domain 。
	a := &Asset{Type: "service", URL: "https://portal.beijing.gov.cn:8443/app"}
	domains, _, urls := a.interceptCandidates()
	if len(urls) != 1 || urls[0] != a.URL {
		t.Fatalf("urls = %v", urls)
	}
	found := false
	for _, d := range domains {
		if d == "portal.beijing.gov.cn" {
			found = true
		}
	}
	if !found {
		t.Fatalf("URL host : %v", domains)
	}
	r, _, ok := MatchAssetInterceptRules([]AssetInterceptRule{rule("fuzzy_domain", ".gov.cn", true)}, domains, nil, urls)
	if !ok {
		t.Fatalf(" URL  fuzzy_domain , rule=%+v", r)
	}

	// URL host  IP  IP ， CIDR 。
	b := &Asset{Type: "service", URL: "http://10.1.2.3/x"}
	_, ips, _ := b.interceptCandidates()
	if r, _, ok := MatchAssetInterceptRules([]AssetInterceptRule{rule("cidr", "10.0.0.0/8", true)}, nil, ips, nil); !ok {
		t.Fatalf("URL  IP  CIDR , ips=%v rule=%+v", ips, r)
	}
}
