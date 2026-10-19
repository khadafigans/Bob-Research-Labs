package notify

import (
	"errors"
	"strings"
	"testing"
)

func TestMaskedValueHidesBodyButKeepsTailHint(t *testing.T) {
	const secret = "https://oapi.dingtalk.com/robot/send?access_token=abcdef123456"
	got := MaskedValue(secret)
	if strings.Contains(got, "abcdef123456") {
		t.Fatalf(": %q", got)
	}
	if strings.Contains(got, "oapi.dingtalk.com") {
		t.Fatalf(": %q", got)
	}
	if !strings.HasSuffix(got, "123456") {
		t.Fatalf(" 6 : %q", got)
	}
	if !IsMasked(got) {
		t.Fatalf(" IsMasked : %q", got)
	}
}

func TestMaskedValueShortSecretGivesNoHint(t *testing.T) {
	for _, s := range []string{"abc", "abcdef", ""} {
		got := MaskedValue(s)
		if got != MaskedPrefix {
			t.Fatalf(" %d ， %q", len(s), got)
		}
		if s != "" && strings.Contains(got, s) {
			t.Fatalf(": %q", got)
		}
	}
}

func TestMaskConfigMasksOnlySecrets(t *testing.T) {
	cfg := map[string]any{
		"webhook": "https://example.com/hook?token=SECRETVALUE",
		"secret":  "SECtest123456",
		"port":    float64(587),
		"host":    "smtp.example.com",
	}
	masked := MaskConfig(KindDingTalk, cfg)
	for _, k := range []string{"webhook", "secret"} {
		s, _ := masked[k].(string)
		if !IsMasked(s) {
			t.Errorf("%s ， %q", k, s)
		}
	}
	if masked["port"] != float64(587) {
		t.Errorf(" port : %v", masked["port"])
	}
}

func TestMaskConfigUnknownKindReturnsEmpty(t *testing.T) {
	got := MaskConfig("nope", map[string]any{"webhook": "https://x/y?token=LEAK"})
	if len(got) != 0 {
		t.Fatalf("， %v", got)
	}
}

func TestMaskConfigDoesNotMutateInput(t *testing.T) {
	cfg := map[string]any{"webhook": "https://example.com/hook", "secret": "SECtest123456"}
	_ = MaskConfig(KindDingTalk, cfg)
	if IsMasked(cfg["secret"].(string)) {
		t.Fatal("MaskConfig ，")
	}
}

func TestMergeConfigKeepsStoredOnMaskedIncoming(t *testing.T) {
	stored := map[string]any{"webhook": "https://real/hook", "secret": "REALSECRET", "method": "POST"}
	//  method，+ method。
	incoming := map[string]any{
		"webhook": MaskedValue("https://real/hook"),
		"secret":  MaskedValue("REALSECRET"),
		"method":  "PUT",
	}
	got := MergeConfig(stored, incoming)
	if got["webhook"] != "https://real/hook" || got["secret"] != "REALSECRET" {
		t.Fatalf("， %v", got)
	}
	if got["method"] != "PUT" {
		t.Fatalf("， %v", got["method"])
	}
}

func TestMergeConfigEmptyStringClears(t *testing.T) {
	stored := map[string]any{"webhook": "https://real/hook", "secret": "REALSECRET"}
	got := MergeConfig(stored, map[string]any{"secret": ""})
	if _, ok := got["secret"]; ok {
		t.Fatalf("， %v", got)
	}
	if got["webhook"] != "https://real/hook" {
		t.Fatalf("， %v", got)
	}
}

func TestMergeConfigKeepsUnmentionedStoredKeys(t *testing.T) {
	stored := map[string]any{"host": "smtp.example.com", "port": float64(587), "password": "pw"}
	got := MergeConfig(stored, map[string]any{"port": float64(465)})
	if got["host"] != "smtp.example.com" || got["password"] != "pw" {
		t.Fatalf("， %v", got)
	}
	if got["port"] != float64(465) {
		t.Fatalf("， %v", got["port"])
	}
}

// TestPrepareConfigUpdateBlocksDestinationSwap ：
//
func TestPrepareConfigUpdateBlocksDestinationSwap(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		stored   map[string]any
		incoming map[string]any
		wantMissing string
	}{
		{
			name: " Webhook  Authorization ",
			kind: KindWebhook,
			stored: map[string]any{
				"url":     "https://legit.example.com/hook",
				"headers": map[string]any{"Authorization": "Bearer REAL-TOKEN"},
			},
			incoming:    map[string]any{"url": "https://attacker.tld/c"},
			wantMissing: "headers",
		},
		{
			name:        "Telegram  base_url  Bot Token ",
			kind:        KindTelegram,
			stored:      map[string]any{"bot_token": "123456:REAL", "chat_id": "1", "base_url": "https://api.telegram.org"},
			incoming:    map[string]any{"base_url": "https://attacker.tld"},
			wantMissing: "bot_token",
		},
		{
			name:        " SMTP ",
			kind:        KindEmail,
			stored:      map[string]any{"host": "smtp.corp.com", "port": 587, "password": "REALPW", "from": "a@b.c", "to": []any{"d@e.f"}},
			incoming:    map[string]any{"host": "smtp.attacker.tld"},
			wantMissing: "password",
		},
		{
			name:        " TLS ",
			kind:        KindEmail,
			stored:      map[string]any{"host": "smtp.corp.com", "port": 587, "tls": false, "password": "REALPW", "from": "a@b.c", "to": []any{"d@e.f"}},
			incoming:    map[string]any{"tls": true},
			wantMissing: "password",
		},
		{
			name:        " + ",
			kind:        KindTelegram,
			stored:      map[string]any{"bot_token": "123456:REAL", "chat_id": "1", "base_url": "https://api.telegram.org"},
			incoming:    map[string]any{"base_url": "https://attacker.tld", "bot_token": MaskedValue("123456:REAL")},
			wantMissing: "bot_token",
		},
		{
			name:        " Webhook ",
			kind:        KindDingTalk,
			stored:      map[string]any{"webhook": "https://oapi.dingtalk.com/robot/send?access_token=OLD", "secret": "REALSEC"},
			incoming:    map[string]any{"webhook": "https://attacker.tld/hook"},
			wantMissing: "secret",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := PrepareConfigUpdate(tc.kind, tc.stored, tc.incoming)
			if err == nil {
				t.Fatalf("，； %v", merged)
			}
			var target *ErrDestinationChangedWithoutCredentials
			if !errors.As(err, &target) {
				t.Fatalf("， %T: %v", err, err)
			}
			found := false
			for _, m := range target.Missing {
				if m == tc.wantMissing {
					found = true
				}
			}
			if !found {
				t.Fatalf(" %q， %v", tc.wantMissing, target.Missing)
			}
			if !strings.Contains(err.Error(), tc.wantMissing) {
				t.Errorf(" %q: %v", tc.wantMissing, err)
			}
		})
	}
}

// TestPrepareConfigUpdateAllowsLegitimateEdits ：，
func TestPrepareConfigUpdateAllowsLegitimateEdits(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		stored   map[string]any
		incoming map[string]any
	}{
		{
			name:     "（）",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://legit.example.com/hook", "headers": map[string]any{"Authorization": "Bearer REAL"}},
			incoming: map[string]any{"url": MaskedValue("https://legit.example.com/hook")},
		},
		{
			name:     "，",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://legit.example.com/hook", "method": "POST"},
			incoming: map[string]any{"method": "PUT"},
		},
		{
			name:     "****",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://old.example.com/hook", "headers": map[string]any{"Authorization": "Bearer OLD"}},
			incoming: map[string]any{"url": "https://new.example.com/hook", "headers": map[string]any{"Authorization": "Bearer NEW"}},
		},
		{
			name:     "",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://old.example.com/hook", "headers": map[string]any{"Authorization": "Bearer OLD"}},
			incoming: map[string]any{"url": "https://new.example.com/hook", "headers": ""},
		},
		{
			name:     "Telegram  chat_id（）",
			kind:     KindTelegram,
			stored:   map[string]any{"bot_token": "t", "chat_id": "1", "base_url": "https://api.telegram.org"},
			incoming: map[string]any{"chat_id": "-100200"},
		},
		{
			name:     "（）",
			kind:     KindEmail,
			stored:   map[string]any{"host": "smtp.corp.com", "port": 587, "password": "PW", "from": "a@b.c", "to": []any{"x@y.z"}},
			incoming: map[string]any{"to": []any{"new@y.z"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := PrepareConfigUpdate(tc.kind, tc.stored, tc.incoming)
			if err != nil {
				t.Fatalf(": %v", err)
			}
			if merged == nil {
				t.Fatal("")
			}
		})
	}
}

// TestPrepareConfigUpdatePortTypeTolerance ：
//  JSON number（float64）， float64，
func TestPrepareConfigUpdatePortTypeTolerance(t *testing.T) {
	stored := map[string]any{"host": "smtp.corp.com", "port": float64(587), "password": "PW"}
	if _, err := PrepareConfigUpdate(KindEmail, stored, map[string]any{"port": 587}); err != nil {
		t.Fatalf("（）: %v", err)
	}
	if _, err := PrepareConfigUpdate(KindEmail, stored, map[string]any{"port": 25}); err == nil {
		t.Fatal("")
	}
}

// TestPrepareConfigUpdateSurvivesRepeatedSaveWithBlankDestination 「
// 」：Telegram  base_url  API 。
//
//
//	 base_url:""（ config， MergeConfig）
//	→ ，MergeConfig 、delete 
//	→ ，incoming  ""、 stored ，「」
//
func TestPrepareConfigUpdateSurvivesRepeatedSaveWithBlankDestination(t *testing.T) {
	stored := map[string]any{"bot_token": "123:ABC", "chat_id": "-100", "base_url": ""}

	submit := func() map[string]any {
		return map[string]any{
			"bot_token": MaskedValue("123:ABC"),
			"chat_id":   "-100",
			"base_url":  "",
		}
	}

	merged, err := PrepareConfigUpdate(KindTelegram, stored, submit())
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if _, ok := merged["base_url"]; ok {
		t.Fatal("： MergeConfig ——「」")
	}

	merged2, err := PrepareConfigUpdate(KindTelegram, merged, submit())
	if err != nil {
		t.Fatalf("（）: %v", err)
	}
	if _, err := PrepareConfigUpdate(KindTelegram, merged2, submit()); err != nil {
		t.Fatalf(": %v", err)
	}
	if got := merged2["bot_token"]; got != "123:ABC" {
		t.Fatalf("Bot Token ， %v", got)
	}
}

// TestPrepareConfigUpdateStillGuardsBlankDestinationChanges 
// ——Telegram  Bot Token  URL ，
//  base_url  Token 。
func TestPrepareConfigUpdateStillGuardsBlankDestinationChanges(t *testing.T) {
	official := map[string]any{"bot_token": "123:ABC", "chat_id": "-100"}
	if _, err := PrepareConfigUpdate(KindTelegram, official, map[string]any{
		"bot_token": MaskedValue("123:ABC"),
		"base_url":  "https://tg-proxy.attacker.tld",
	}); err == nil {
		t.Fatal(" Token")
	}

	proxied := map[string]any{"bot_token": "123:ABC", "base_url": "https://proxy.internal/bot"}
	if _, err := PrepareConfigUpdate(KindTelegram, proxied, map[string]any{
		"bot_token": MaskedValue("123:ABC"),
		"base_url":  "",
	}); err == nil {
		t.Fatal("（ API）， Token")
	}
}

func TestDestinationKeysDeclaredForEveryKind(t *testing.T) {
	//  SecretKeys ：，PrepareConfigUpdate 。
	for kind, ch := range registry {
		if len(ch.DestinationKeys()) == 0 {
			t.Errorf(" %s ，", kind)
		}
		if len(ch.SecretKeys()) == 0 {
			t.Errorf(" %s ", kind)
		}
	}
}

func TestSecretKeysDeclaredForEveryKind(t *testing.T) {
	expect := map[string]bool{
		KindDingTalk: true, KindFeishu: true, KindWeCom: true,
		KindWebhook: true, KindTelegram: true, KindEmail: true,
	}
	for kind, ch := range registry {
		if !expect[kind] {
			t.Errorf(" %s ", kind)
			continue
		}
		if len(ch.SecretKeys()) == 0 {
			t.Errorf(" %s ，", kind)
		}
	}
}

// TestPrepareConfigUpdateRejectsMaskedInContainer ：
// ****（ webhook.headers ），
// MergeConfig 「」， "__masked__" 
func TestPrepareConfigUpdateRejectsMaskedInContainer(t *testing.T) {
	stored := map[string]any{
		"url":     "https://legit.example.com/hook",
		"headers": map[string]any{"Authorization": "Bearer REAL"},
	}
	incoming := map[string]any{
		"headers": map[string]any{"Authorization": MaskedPrefix},
	}
	if _, err := PrepareConfigUpdate(KindWebhook, stored, incoming); err == nil {
		t.Fatal("（）")
	}
	ok := map[string]any{"headers": map[string]any{"Authorization": "Bearer NEW"}}
	if _, err := PrepareConfigUpdate(KindWebhook, stored, ok); err != nil {
		t.Fatalf(": %v", err)
	}
}
