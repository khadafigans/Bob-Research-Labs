package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/notify"
)

//
// ：** Notifier.step()**，
//  stepRealtime/stepDigest。 step() ——
//
//
// ：stepRealtime/stepDigest 、，

// notifyFixture 。
type notifyFixture struct {
	s       *Server
	pg      *db.DB
	request func(method, path, body string) *httptest.ResponseRecorder
	n       *Notifier
	//  task/exploration：，。
	taskID int64
	expID  int64
	cleanupMark int64
}

func newNotifyFixture(t *testing.T) *notifyFixture {
	t.Helper()
	// 「」 notify  ssrf_test.go 。
	t.Setenv(notify.AllowLocalTargetsEnv, "1")
	//   (3 tick) .    stepRealtime·
	//   )  .   trafficEvidenceServer   
	t.Setenv(notifyBackgroundDisabledEnv, "1")
	s, _, request := trafficEvidenceServer(t)
	pg := s.m.pg

	//  task： trafficEvidenceServer  task  exploration id，
	task, err := s.m.CreateTask("", "", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := strconv.ParseInt(task.ID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pg.Exec(`DELETE FROM tasks WHERE id=$1`, taskID) })

	var mark int64
	if err := pg.QueryRow(`SELECT COALESCE(max(id),0) FROM notification_events`).Scan(&mark); err != nil {
		t.Fatal(err)
	}
	//
	// ：FanOutPendingEvents ****，
	// 。 trafficEvidenceServer 
	if _, err := pg.Exec(`UPDATE notification_events SET fanned_out = true WHERE id <= $1 AND NOT fanned_out`, mark); err != nil {
		t.Fatal(err)
	}

	f := &notifyFixture{s: s, pg: pg, request: request, n: newNotifier(s), taskID: taskID, expID: task.ExpID, cleanupMark: mark}
	t.Cleanup(func() {
		if _, err := pg.Exec(`DELETE FROM notification_events WHERE id > $1`, f.cleanupMark); err != nil {
			t.Logf(": %v", err)
		}
	})
	if err := pg.SetBool(settingNotifyEnabled, true); err != nil {
		t.Fatal(err)
	}
	return f
}

// record ， finding id。
func (f *notifyFixture) record(t *testing.T, vulnclass, severity string) int64 {
	t.Helper()
	out, err := f.s.evidenceStore().Record(context.Background(), db.RecordFindingInput{
		TaskID:        f.taskID,
		ExplorationID: f.expID,
		Worker:        "test",
		VulnClass:     vulnclass,
		Name:          vulnclass,
		Severity:      severity,
		Summary:       vulnclass + " ",
		Evidence:      "poc",
	}, nil)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	return out.FindingID
}

// channel （ stepX ）。
func (f *notifyFixture) channel(t *testing.T, id int64) *db.NotificationChannel {
	t.Helper()
	ch, err := f.pg.NotificationChannelByID(context.Background(), id)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	return ch
}

func (f *notifyFixture) deliver(t *testing.T, chID int64, baseURL string) {
	t.Helper()
	ctx := context.Background()
	if _, _, err := f.pg.FanOutPendingEvents(ctx, 500); err != nil {
		t.Fatalf(": %v", err)
	}
	f.n.stepRealtime(ctx, f.channel(t, chID), 50, baseURL)
}

// createChannel  HTTP ，。
func (f *notifyFixture) createChannel(t *testing.T, payload map[string]any) int64 {
	t.Helper()
	raw, _ := json.Marshal(payload)
	r := f.request("POST", "/api/notify/channels", string(raw))
	if r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	var res struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &res); err != nil || res.ID == 0 {
		t.Fatalf(": %s (%v)", r.Body, err)
	}
	t.Cleanup(func() { f.pg.Exec(`DELETE FROM notification_channels WHERE id=$1`, res.ID) })
	return res.ID
}

type fakeWebhook struct {
	*httptest.Server
	mu     sync.Mutex
	bodies []map[string]any
}

func newFakeWebhook(t *testing.T) *fakeWebhook {
	t.Helper()
	f := &fakeWebhook{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.bodies = append(f.bodies, body)
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeWebhook) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bodies)
}

func (f *fakeWebhook) body(t *testing.T, i int) map[string]any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if i >= len(f.bodies) {
		t.Fatalf(" %d ， %d ", len(f.bodies), i)
	}
	return f.bodies[i]
}

func (f *fakeWebhook) last(t *testing.T) map[string]any {
	t.Helper()
	if f.count() == 0 {
		t.Fatal("")
	}
	return f.body(t, f.count()-1)
}

// markdownText ，：
//  markdown  `text`、ActionCard  `text`、 markdown  `content`。
func markdownText(t *testing.T, body map[string]any) string {
	t.Helper()
	for _, key := range []string{"markdown", "actionCard"} {
		section, ok := body[key].(map[string]any)
		if !ok {
			continue
		}
		for _, field := range []string{"text", "content"} {
			if s, ok := section[field].(string); ok && s != "" {
				return s
			}
		}
	}
	t.Fatalf(": %v", body)
	return ""
}

// agePendingBatch ，。
func (f *notifyFixture) agePendingBatch(t *testing.T, chID int64) {
	t.Helper()
	if _, err := f.pg.Exec(`UPDATE notification_deliveries SET created_at = now() - interval '2 hours'
WHERE channel_id=$1 AND state=$2`, chID, db.NotifyStatePending); err != nil {
		t.Fatal(err)
	}
}

func TestNotifyEndToEndRealtimeDelivery(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
	})
	f.record(t, "SQL", "high")
	f.deliver(t, chID, "")

	if hook.count() != 1 {
		t.Fatalf(" 1 ， %d", hook.count())
	}
	text := markdownText(t, hook.last(t))
	//   ). ""·"Summary"    —  notify.SeverityLabel("high")
	for _, want := range []string{"SQL", "", "Summary"} {
		if !strings.Contains(text, want) {
			t.Fatalf(" %q:\n%s", want, text)
		}
	}
	var pending int
	if err := f.pg.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE channel_id=$1 AND state <> $2`,
		chID, db.NotifyStateSent).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf(" %d  sent", pending)
	}
}

func TestNotifyChannelAPIMasksSecretsAndPreservesOnUpdate(t *testing.T) {
	f := newNotifyFixture(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": "https://oapi.dingtalk.com/robot/send?access_token=abc123456", "secret": "SECabcdef123456"},
	})

	r := f.request("GET", "/api/notify/channels", "")
	if r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	if strings.Contains(r.Body.String(), "abc123456") || strings.Contains(r.Body.String(), "SECabcdef123456") {
		t.Fatalf(": %s", r.Body)
	}
	var listed struct {
		Channels []struct {
			ID         int64          `json:"id"`
			Config     map[string]any `json:"config"`
			SecretKeys []string       `json:"secret_keys"`
		} `json:"channels"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	var mine *struct {
		ID         int64          `json:"id"`
		Config     map[string]any `json:"config"`
		SecretKeys []string       `json:"secret_keys"`
	}
	for i := range listed.Channels {
		if listed.Channels[i].ID == chID {
			mine = &listed.Channels[i]
		}
	}
	if mine == nil {
		t.Fatal("")
	}
	if !notify.IsMasked(fmt.Sprint(mine.Config["webhook"])) || !notify.IsMasked(fmt.Sprint(mine.Config["secret"])) {
		t.Fatalf(": %v", mine.Config)
	}
	if len(mine.SecretKeys) == 0 {
		t.Fatal("")
	}

	body, _ := json.Marshal(map[string]any{
		"name":   "",
		"config": map[string]any{"webhook": fmt.Sprint(mine.Config["webhook"]), "secret": fmt.Sprint(mine.Config["secret"])},
	})
	if r := f.request("PATCH", fmt.Sprintf("/api/notify/channels/%d", chID), string(body)); r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	cfg := f.channelConfig(t, chID)
	if cfg["webhook"] != "https://oapi.dingtalk.com/robot/send?access_token=abc123456" {
		t.Fatalf(": %v", cfg["webhook"])
	}
	if cfg["secret"] != "SECabcdef123456" {
		t.Fatalf(" secret : %v", cfg["secret"])
	}
	if f.channel(t, chID).Name != "" {
		t.Fatal("")
	}

	body, _ = json.Marshal(map[string]any{"config": map[string]any{"secret": ""}})
	if r := f.request("PATCH", fmt.Sprintf("/api/notify/channels/%d", chID), string(body)); r.Code != 200 {
		t.Fatalf(" secret  %d: %s", r.Code, r.Body)
	}
	if _, still := f.channelConfig(t, chID)["secret"]; still {
		t.Fatal(" secret")
	}
}

func (f *notifyFixture) channelConfig(t *testing.T, id int64) map[string]any {
	t.Helper()
	var cfg map[string]any
	if err := json.Unmarshal(f.channel(t, id).Config, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestNotifyChannelAPICreateValidation(t *testing.T) {
	f := newNotifyFixture(t)
	cases := []struct {
		name    string
		payload map[string]any
		wantSub string
	}{
		// wantSub  server/notify_api.go      
		// (notifyErrKindInvalidFmt·notifyErrNameMissing·notifyErrModeInvalid   ).
		// webhook   notify/dingtalk.go  Validate   ,  webhook 
		//     (notify/dingtalk_feishu_wecom_localized_test.go
		//  "Webhook URL is required"·"Invalid Webhook URL"  ) 
		{"", map[string]any{"name": "x", "kind": "nope", "config": map[string]any{}}, "   "},
		{"", map[string]any{"kind": notify.KindDingTalk, "config": map[string]any{"webhook": "https://e.com/h"}}, "  "},
		{" webhook", map[string]any{"name": "x", "kind": notify.KindDingTalk, "config": map[string]any{}}, "Webhook URL is required"},
		{"webhook ", map[string]any{"name": "x", "kind": notify.KindDingTalk, "config": map[string]any{"webhook": "file:///etc/passwd"}}, "Invalid Webhook URL"},
		{"", map[string]any{"name": "x", "kind": notify.KindDingTalk, "mode": "sometimes", "config": map[string]any{"webhook": "https://e.com/h"}}, "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(tc.payload)
			r := f.request("POST", "/api/notify/channels", string(raw))
			if r.Code != 400 {
				t.Fatalf(" 400， %d: %s", r.Code, r.Body)
			}
			if !strings.Contains(r.Body.String(), tc.wantSub) {
				t.Fatalf(" %q， %s", tc.wantSub, r.Body)
			}
		})
	}
	if r := f.request("DELETE", "/api/notify/channels/99999999", ""); r.Code != 404 {
		t.Fatalf(" 404， %d", r.Code)
	}
}

func TestNotifyFilterBlocksBelowThreshold(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
		"filter": map[string]any{"min_severity": "critical"},
	})
	f.record(t, "", "low")
	if _, _, err := f.pg.FanOutPendingEvents(context.Background(), 500); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.pg.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE channel_id=$1`, chID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("， %d ", n)
	}
	f.n.stepRealtime(context.Background(), f.channel(t, chID), 50, "")
	if hook.count() != 0 {
		t.Fatal("")
	}
}

func TestNotifyDigestBatchesMultipleFindingsIntoOneMessage(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"mode":   db.NotifyModeDigest,
		"config": map[string]any{"webhook": hook.URL},
	})
	for i := 0; i < 3; i++ {
		f.record(t, fmt.Sprintf("%d", i+1), "high")
	}
	ctx := context.Background()
	if _, _, err := f.pg.FanOutPendingEvents(ctx, 500); err != nil {
		t.Fatal(err)
	}
	ch := f.channel(t, chID)

	f.n.stepDigest(ctx, ch, 50, "")
	if hook.count() != 0 {
		t.Fatal("")
	}

	f.agePendingBatch(t, chID)
	f.n.stepDigest(ctx, ch, 50, "")
	if got := hook.count(); got != 1 {
		t.Fatalf("， %d ", got)
	}
	text := markdownText(t, hook.last(t))
	//     markdown.go  "** N new finding(s) N**"
	//  (digestInterval  30 WindowMinutes>0).
	if !strings.Contains(text, "") || !strings.Contains(text, "3 new finding(s)") {
		t.Fatalf("/:\n%s", text)
	}
	for i := 1; i <= 3; i++ {
		if !strings.Contains(text, fmt.Sprintf("%d", i)) {
			t.Fatalf(" %d :\n%s", i, text)
		}
	}
	var distinct, total int
	if err := f.pg.QueryRow(`SELECT count(DISTINCT batch_id), count(*) FROM notification_deliveries WHERE channel_id=$1`, chID).Scan(&distinct, &total); err != nil {
		t.Fatal(err)
	}
	if total != 3 || distinct != 1 {
		t.Fatalf(" batch_id， distinct=%d total=%d", distinct, total)
	}
}

func TestNotifyDisabledChannelDoesNotSend(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":    "",
		"kind":    notify.KindDingTalk,
		"enabled": false,
		"config":  map[string]any{"webhook": hook.URL},
	})
	f.record(t, "", "critical")
	if _, _, err := f.pg.FanOutPendingEvents(context.Background(), 500); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.pg.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE channel_id=$1`, chID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("， %d ", n)
	}
}

func TestNotifyStatusChangeDelivery(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
		"filter": map[string]any{"on_status_change": true},
	})
	finding := f.record(t, "", "high")
	r := f.request("PATCH", fmt.Sprintf("/api/exploration/findings/%d", finding), `{"status":"fixed"}`)
	if r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	f.deliver(t, chID, "")

	// ：fixed ；finding_created 。
	found := false
	for i := 0; i < hook.count(); i++ {
		text := markdownText(t, hook.body(t, i))
		// Status change  markdown.go  "**Status change**: %s → %s" , 
		// notify.StatusLabel  (fixed → "Fixed").
		if strings.Contains(text, "Status change") && strings.Contains(text, "Fixed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("「 → 」（ %d ）", hook.count())
	}
}

func TestNotifyStatusChangeSuppressedByDefault(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
	})
	finding := f.record(t, "", "high")
	if r := f.request("PATCH", fmt.Sprintf("/api/exploration/findings/%d", finding), `{"status":"false_positive"}`); r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	if _, _, err := f.pg.FanOutPendingEvents(context.Background(), 500); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.pg.QueryRow(`SELECT count(*) FROM notification_deliveries d
JOIN notification_events e ON e.id = d.event_id
WHERE d.channel_id=$1 AND e.kind=$2`, chID, notify.EventFindingStatusChanged).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("， %d ", n)
	}
}

func TestNotifyTestMessageEndpoint(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
	})
	if r := f.request("POST", fmt.Sprintf("/api/notify/channels/%d/test", chID), ""); r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	if hook.count() != 1 {
		t.Fatalf(" 1 ， %d", hook.count())
	}
	//    notify_api.go  notifyTestName("  ·   ").
	if text := markdownText(t, hook.last(t)); !strings.Contains(text, "") {
		t.Fatalf(": %s", text)
	}
	badID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": "http://127.0.0.1:1/hook"},
	})
	if r := f.request("POST", fmt.Sprintf("/api/notify/channels/%d/test", badID), ""); r.Code != 502 {
		t.Fatalf(" 502， %d: %s", r.Code, r.Body)
	}
}

func TestNotifyDeliveriesHistoryAndRetry(t *testing.T) {
	f := newNotifyFixture(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": "http://127.0.0.1:1/hook"},
	})
	f.record(t, "", "high")
	ctx := context.Background()
	if _, _, err := f.pg.FanOutPendingEvents(ctx, 500); err != nil {
		t.Fatal(err)
	}
	ch := f.channel(t, chID)
	for i := 0; i < db.MaxNotifyAttempts; i++ {
		f.n.stepRealtime(ctx, ch, 50, "")
		if _, err := f.pg.Exec(`UPDATE notification_deliveries SET next_attempt_at = now() - interval '1 minute' WHERE channel_id=$1`, chID); err != nil {
			t.Fatal(err)
		}
	}
	var state string
	if err := f.pg.QueryRow(`SELECT state FROM notification_deliveries WHERE channel_id=$1`, chID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != db.NotifyStateFailed {
		t.Fatalf(" failed， %s", state)
	}

	r := f.request("GET", fmt.Sprintf("/api/notify/deliveries?channel_id=%d&state=failed", chID), "")
	if r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	var hist struct {
		Deliveries []struct {
			ID        int64  `json:"id"`
			State     string `json:"state"`
			LastError string `json:"last_error"`
			Attempts  int    `json:"attempts"`
			Title     string `json:"title"`
		} `json:"deliveries"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &hist); err != nil {
		t.Fatal(err)
	}
	if hist.Total != 1 || len(hist.Deliveries) != 1 {
		t.Fatalf(" 1 ， total=%d len=%d", hist.Total, len(hist.Deliveries))
	}
	if hist.Deliveries[0].LastError == "" {
		t.Fatal("，")
	}
	if hist.Deliveries[0].Attempts < db.MaxNotifyAttempts {
		t.Fatalf("， %d", hist.Deliveries[0].Attempts)
	}
	if hist.Deliveries[0].Title != "" {
		t.Fatalf("， %q", hist.Deliveries[0].Title)
	}

	if r := f.request("POST", fmt.Sprintf("/api/notify/deliveries/%d/retry", hist.Deliveries[0].ID), ""); r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	var attempts int
	if err := f.pg.QueryRow(`SELECT state, attempts FROM notification_deliveries WHERE id=$1`, hist.Deliveries[0].ID).Scan(&state, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != db.NotifyStatePending || attempts != 0 {
		t.Fatalf(" pending  attempts=0， %s/%d", state, attempts)
	}
}

func TestNotifyMetaAndSettingsRoundTrip(t *testing.T) {
	f := newNotifyFixture(t)
	r := f.request("GET", "/api/notify/meta", "")
	if r.Code != 200 {
		t.Fatalf("meta : %s", r.Body)
	}
	var meta struct {
		Kinds []struct {
			Kind       string   `json:"kind"`
			SecretKeys []string `json:"secret_keys"`
		} `json:"kinds"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if len(meta.Kinds) != len(notify.Kinds()) {
		t.Fatalf("meta  %d ， %d", len(notify.Kinds()), len(meta.Kinds))
	}
	for _, k := range meta.Kinds {
		if len(k.SecretKeys) == 0 {
			t.Errorf(" %s ", k.Kind)
		}
	}

	if r := f.request("PUT", "/api/settings", `{"notify_public_base_url":"https://artex.example.com/","notify_digest_interval_min":15,"notify_enabled":true}`); r.Code != 200 {
		t.Fatalf(" %d: %s", r.Code, r.Body)
	}
	t.Cleanup(func() {
		f.pg.Exec(`DELETE FROM settings WHERE key IN ($1,$2)`, settingNotifyPublicBaseURL, settingNotifyDigestMinutes)
	})
	payload := f.s.settingsPayload()
	if payload["notify_public_base_url"] != "https://artex.example.com" {
		t.Fatalf(": %v", payload["notify_public_base_url"])
	}
	if payload["notify_digest_interval_min"] != 15 {
		t.Fatalf(": %v", payload["notify_digest_interval_min"])
	}

	for _, body := range []string{
		`{"notify_public_base_url":"ftp://x"}`,
		`{"notify_digest_interval_min":0}`,
		`{"notify_digest_interval_min":99999}`,
	} {
		if r := f.request("PUT", "/api/settings", body); r.Code != 400 {
			t.Errorf("%s  400， %d", body, r.Code)
		}
	}
}

// TestNotifyDeepLinkUsesPublicBaseURL ： public_base_url 
func TestNotifyDeepLinkUsesPublicBaseURL(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
	})
	finding := f.record(t, "", "high")
	f.deliver(t, chID, "https://artex.example.com")

	body := hook.last(t)
	card, _ := body["actionCard"].(map[string]any)
	if card == nil {
		t.Fatalf(" ActionCard， msgtype=%v", body["msgtype"])
	}
	want := fmt.Sprintf("https://artex.example.com/function/findings/detail?id=%d", finding)
	if card["singleURL"] != want {
		t.Fatalf("\n %s\n %v", want, card["singleURL"])
	}
}

// TestNotifyNoDeepLinkWithoutBaseURL ：
// （ localhost ）， markdown。
func TestNotifyNoDeepLinkWithoutBaseURL(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindDingTalk,
		"config": map[string]any{"webhook": hook.URL},
	})
	f.record(t, "", "high")
	f.deliver(t, chID, "")

	body := hook.last(t)
	if body["msgtype"] != "markdown" {
		t.Fatalf(" markdown， %v", body["msgtype"])
	}
	//    markdown.go  "[View details](URL)"   —  
	if text := markdownText(t, body); strings.Contains(text, "View details") {
		t.Fatalf(":\n%s", text)
	}
}

// TestNotifyDigestSegmentsAndDefersRemainder 「」。
//
//
func TestNotifyDigestSegmentsAndDefersRemainder(t *testing.T) {
	f := newNotifyFixture(t)
	hook := newFakeWebhook(t)
	chID := f.createChannel(t, map[string]any{
		"name":   "",
		"kind":   notify.KindWeCom,
		"mode":   db.NotifyModeDigest,
		"config": map[string]any{"webhook": hook.URL},
	})
	const total = 60
	longName := strings.Repeat("", 6)
	for i := 0; i < total; i++ {
		f.record(t, longName+strconv.Itoa(i+1), "high")
	}
	ctx := context.Background()
	if _, _, err := f.pg.FanOutPendingEvents(ctx, 500); err != nil {
		t.Fatal(err)
	}
	f.agePendingBatch(t, chID)
	ch := f.channel(t, chID)

	f.n.stepDigest(ctx, ch, 50, "")
	if hook.count() != 1 {
		t.Fatalf("， %d", hook.count())
	}

	var sent, pending int
	if err := f.pg.QueryRow(`SELECT
    count(*) FILTER (WHERE state=$2),
    count(*) FILTER (WHERE state=$3)
  FROM notification_deliveries WHERE channel_id=$1`, chID, db.NotifyStateSent, db.NotifyStatePending).
		Scan(&sent, &pending); err != nil {
		t.Fatal(err)
	}
	if sent == 0 {
		t.Fatal("")
	}
	if pending == 0 {
		t.Fatalf(" %d  4096 ，；sent=%d", total, sent)
	}
	if sent+pending != total {
		t.Fatalf("：sent=%d pending=%d total=%d（=）", sent, pending, total)
	}
	//   markdown.go  "(   N …  N next message …)"  .
	if text := markdownText(t, hook.last(t)); !strings.Contains(text, "") {
		t.Fatalf(":\n%.400s", text)
	}

	var maxAttempts int
	if err := f.pg.QueryRow(`SELECT COALESCE(max(attempts),0) FROM notification_deliveries
WHERE channel_id=$1 AND state=$2`, chID, db.NotifyStatePending).Scan(&maxAttempts); err != nil {
		t.Fatal(err)
	}
	if maxAttempts > 0 {
		t.Fatalf("（）， attempts=%d", maxAttempts)
	}

	rounds := 0
	for {
		var undelivered int
		if err := f.pg.QueryRow(`SELECT count(*) FROM notification_deliveries
WHERE channel_id=$1 AND state <> $2 AND state <> $3`, chID, db.NotifyStateSent, db.NotifyStateFailed).
			Scan(&undelivered); err != nil {
			t.Fatal(err)
		}
		if undelivered == 0 {
			break
		}
		rounds++
		if rounds > total+5 {
			t.Fatalf("： %d  %d ", rounds, undelivered)
		}
		before := hook.count()
		f.n.stepDigest(ctx, ch, 50, "")
		if hook.count() == before {
			t.Fatalf(" %d ， %d ", rounds, undelivered)
		}
	}
	if rounds < 2 {
		t.Fatalf(" 4096  %d ，， %d ", total, rounds)
	}
	var failed int
	if err := f.pg.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE channel_id=$1 AND state=$2`,
		chID, db.NotifyStateFailed).Scan(&failed); err != nil {
		t.Fatal(err)
	}
	if failed != 0 {
		t.Fatalf("，， %d", failed)
	}
}

// TestNotifyBackoffTableMatchesAttemptBudget 。
//
// （db.MaxNotifyAttempts）（notifyBackoff）：
func TestNotifyBackoffTableMatchesAttemptBudget(t *testing.T) {
	if len(notifyBackoff) != db.MaxNotifyAttempts {
		t.Fatalf("(%d)(%d)——",
			len(notifyBackoff), db.MaxNotifyAttempts)
	}
	for i := 1; i < len(notifyBackoff); i++ {
		if notifyBackoff[i] < notifyBackoff[i-1] {
			t.Fatalf("： %d  %v <  %d  %v",
				i, notifyBackoff[i], i-1, notifyBackoff[i-1])
		}
	}
}

// TestNotifyRateLimitDoesNotConsumeRetryBudget 「」。
func TestNotifyRateLimitDoesNotConsumeRetryBudget(t *testing.T) {
	n := &Notifier{buckets: map[int64]*notifyBucket{}}
	now := time.Now()
	if got := n.takeTokens(1, 1, notifyMaxSendsPerChannelPerTick, now); got != 1 {
		t.Fatalf(" 1  1 ， %d", got)
	}
	if got := n.takeTokens(1, 1, notifyMaxSendsPerChannelPerTick, now.Add(time.Millisecond)); got != 0 {
		t.Fatalf(" 0， %d", got)
	}
	if got := n.takeTokens(1, 1, notifyMaxSendsPerChannelPerTick, now.Add(30*time.Second)); got != 0 {
		t.Fatalf("， %d", got)
	}
	if got := n.takeTokens(1, 1, notifyMaxSendsPerChannelPerTick, now.Add(time.Minute)); got != 1 {
		t.Fatalf(" 1 ， %d", got)
	}
	if got := n.takeTokens(2, 0, notifyUnlimitedBurstPerTick+10, now); got != notifyUnlimitedBurstPerTick {
		t.Fatalf(" %d， %d", notifyUnlimitedBurstPerTick, got)
	}
	if got := n.takeTokens(1, 1, notifyMaxSendsPerChannelPerTick, now.Add(time.Millisecond)); got != 0 {
		t.Fatalf(" 1 ， %d", got)
	}
}

// TestNotifyTakeTokensKeepsUnusedTokens 「 want 」。
//
func TestNotifyTakeTokensKeepsUnusedTokens(t *testing.T) {
	n := &Notifier{buckets: map[int64]*notifyBucket{}}
	now := time.Now()
	if got := n.takeTokens(1, 100, 5, now); got != 5 {
		t.Fatalf("want=5  5 ， %d", got)
	}
	if got := n.takeTokens(1, 100, 95, now); got != 95 {
		t.Fatalf("（ 95）， %d——", got)
	}
	if got := n.takeTokens(1, 100, 1, now); got != 0 {
		t.Fatalf("， 0， %d", got)
	}
	n2 := &Notifier{buckets: map[int64]*notifyBucket{}}
	if got := n2.takeTokens(1, 20, 0, now); got != 0 {
		t.Fatalf("want=0  0， %d", got)
	}
	if got := n2.takeTokens(1, 20, 20, now); got != 20 {
		t.Fatalf("want=0 ， 20， %d", got)
	}
}

// TestDigestTickPlanDecouplesBatchSizeFromSendBudget 。
//
// （ stepDigest  limit，
func TestDigestTickPlanDecouplesBatchSizeFromSendBudget(t *testing.T) {
	tokens, claimLimit := digestTickPlan()
	if tokens != 1 {
		t.Fatalf("， 1 ， %d", tokens)
	}
	if claimLimit != db.MaxDigestBatchSize {
		t.Fatalf(" db.MaxDigestBatchSize=%d， %d",
			db.MaxDigestBatchSize, claimLimit)
	}
	if claimLimit <= notifyMaxSendsPerChannelPerTick {
		t.Fatalf(" %d  %d ——"+
			"「」，「」",
			claimLimit, notifyMaxSendsPerChannelPerTick)
	}
}

// TestNotifyTickBudgetFitsWithinLease 。
//
// （notifyMaxSendsPerChannelPerTick）：
func TestNotifyTickBudgetFitsWithinLease(t *testing.T) {
	worst := time.Duration(notifyMaxSendsPerChannelPerTick) * notifySendTimeout
	if worst >= notifyLease {
		t.Fatalf(" %v  %v"+
			"（notifyMaxSendsPerChannelPerTick=%d × notifySendTimeout=%v）——"+
			"",
			worst, notifyLease, notifyMaxSendsPerChannelPerTick, notifySendTimeout)
	}
}
