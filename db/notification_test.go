package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Autumn-27/artex/notify"
)

//  PostgreSQL（）。 SQL 
// FOR UPDATE SKIP LOCKED、make_interval、JSONB、 IN(...) ，

func notifyTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// newTestChannel ，。
func newTestChannel(t *testing.T, d *DB, kind, mode string, filter string) *NotificationChannel {
	t.Helper()
	if filter == "" {
		filter = `{}`
	}
	ch := &NotificationChannel{
		Name:       "-" + t.Name(),
		Kind:       kind,
		Mode:       mode,
		Config:     json.RawMessage(`{"webhook":"https://example.com/hook"}`),
		Filter:     json.RawMessage(filter),
		RatePerMin: 100,
	}
	id, err := d.SaveNotificationChannel(context.Background(), ch)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM notification_channels WHERE id=$1`, id) })
	ch.ID = id
	return ch
}

// addTestEvent （ finding），。
func addTestEvent(t *testing.T, d *DB, kind string, findingID int64, snap notify.Snapshot) int64 {
	t.Helper()
	snap.Kind = kind
	snap.FindingID = findingID
	id, err := d.AddNotificationEvent(context.Background(), kind, findingID, snap)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM notification_events WHERE id=$1`, id) })
	return id
}

func TestNotificationAssetNamesResolvesAndPreservesOrder(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	insertAsset := func(query, value string) int64 {
		t.Helper()
		var id int64
		if err := d.QueryRow(query, value).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	domID := insertAsset(`INSERT INTO assets(type, domain) VALUES('subdomain',$1) RETURNING id`, "a.example.com")
	ipID := insertAsset(`INSERT INTO assets(type, ip) VALUES('ip',$1) RETURNING id`, "10.1.2.3")
	svcID := insertAsset(`INSERT INTO assets(type, url) VALUES('service',$1) RETURNING id`, "https://a.example.com/admin")
	t.Cleanup(func() {
		d.Exec(`DELETE FROM assets WHERE id IN ($1,$2,$3)`, domID, ipID, svcID)
	})

	got, err := d.NotificationAssetNames(ctx, []int64{svcID, 999999999, domID, ipID, svcID})
	if err != nil {
		t.Fatalf(": %v", err)
	}
	want := []string{"https://a.example.com/admin", "a.example.com", "10.1.2.3"}
	if len(got) != len(want) {
		t.Fatalf("， %v  %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("/， %v  %v", want, got)
		}
	}
}

// TestRecordNotificationEventTxUnwindsOnFailure ：
//  notification_events （ false ），
//  ①  false ②  aborted ，。
//
// "current transaction is aborted" ——「
//
//  **ROLLBACK  COMMIT**：ALTER TABLE  PG ，
func TestRecordNotificationEventTxUnwindsOnFailure(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	if _, err := d.Exec(`ALTER TABLE notification_events DROP CONSTRAINT IF EXISTS notify_test_never`); err != nil {
		t.Fatal(err)
	}

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck // ，

	if _, err := tx.ExecContext(ctx, `ALTER TABLE notification_events ADD CONSTRAINT notify_test_never CHECK (false) NOT VALID`); err != nil {
		t.Fatalf(": %v", err)
	}
	if RecordNotificationEventTx(ctx, tx, notify.EventFindingCreated, 1, notify.Snapshot{Severity: "high"}) {
		t.Fatal("")
	}
	var one int
	if err := tx.QueryRowContext(ctx, `SELECT 1`).Scan(&one); err != nil {
		t.Fatalf("（）: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf(": %v", err)
	}
	var exists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='notify_test_never')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("，")
	}
}

func TestFanOutRoutesEventsByFilter(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	all := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	onlyCritical := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{"min_severity":"critical"}`)
	sqlOnly := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{"vulnclass_include":["SQL"]}`)

	highSQL := addTestEvent(t, d, notify.EventFindingCreated, 1001, notify.Snapshot{Severity: "high", VulnClass: "SQL"})
	lowXSS := addTestEvent(t, d, notify.EventFindingCreated, 1002, notify.Snapshot{Severity: "low", VulnClass: "XSS"})
	criticalXSS := addTestEvent(t, d, notify.EventFindingCreated, 1003, notify.Snapshot{Severity: "critical", VulnClass: "XSS"})

	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatalf(": %v", err)
	}

	cases := []struct {
		name    string
		eventID int64
		channel int64
		want    bool
	}{
		{" high", highSQL, all.ID, true},
		{" low", lowXSS, all.ID, true},
		{" high", highSQL, onlyCritical.ID, false},
		{" critical", criticalXSS, onlyCritical.ID, true},
		{"SQL SQL", highSQL, sqlOnly.ID, true},
		{"SQL XSS", lowXSS, sqlOnly.ID, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var exists bool
			if err := d.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM notification_deliveries WHERE event_id=$1 AND channel_id=$2)`,
				tc.eventID, tc.channel).Scan(&exists); err != nil {
				t.Fatal(err)
			}
			if exists != tc.want {
				t.Fatalf(":  %v  %v", tc.want, exists)
			}
		})
	}

	events, deliveries, err := d.FanOutPendingEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if events != 0 || deliveries != 0 {
		t.Fatalf("， events=%d deliveries=%d", events, deliveries)
	}
}

// TestFanOutMarksEventsWithNoMatchingChannel 「」。
func TestFanOutMarksEventsWithNoMatchingChannel(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	pick := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{"vulnclass_include":[""]}`)
	_ = pick

	ev := addTestEvent(t, d, notify.EventFindingCreated, 2001, notify.Snapshot{Severity: "high", VulnClass: "XSS"})
	_, deliveries, err := d.FanOutPendingEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if deliveries != 0 {
		t.Fatalf("， %d", deliveries)
	}
	var fanned bool
	if err := d.QueryRowContext(ctx, `SELECT fanned_out FROM notification_events WHERE id=$1`, ev).Scan(&fanned); err != nil {
		t.Fatal(err)
	}
	if !fanned {
		t.Fatal("，")
	}
}

func TestClaimRealtimeDeliveriesHonorsLeaseAndMode(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	realtime := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	digest := newTestChannel(t, d, notify.KindDingTalk, NotifyModeDigest, `{}`)

	addTestEvent(t, d, notify.EventFindingCreated, 3001, notify.Snapshot{Severity: "high", VulnClass: "XSS"})
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}

	//  realtime ， digest 。
	got, err := d.ClaimRealtimeDeliveries(ctx, realtime.ID, 10, time.Minute)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if len(got) != 1 {
		t.Fatalf(" 1 ， %d", len(got))
	}
	if got[0].State != NotifyStateSending || got[0].Attempts != 1 {
		t.Fatalf(" sending  attempts=1， state=%s attempts=%d", got[0].State, got[0].Attempts)
	}
	if got[0].Channel == nil || len(got[0].Channel.Config) == 0 {
		t.Fatal("，")
	}
	if got[0].FindingID != 3001 {
		t.Fatalf("finding id ， %d", got[0].FindingID)
	}

	again, err := d.ClaimRealtimeDeliveries(ctx, realtime.ID, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("， %d ", len(again))
	}

	left, err := d.ClaimRealtimeDeliveries(ctx, digest.ID, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf(" digest ， %d ", len(left))
	}
}

// TestClaimExpiredLeaseRecovers ： sending
func TestClaimExpiredLeaseRecovers(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	addTestEvent(t, d, notify.EventFindingCreated, 4001, notify.Snapshot{Severity: "high"})
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	first, err := d.ClaimRealtimeDeliveries(ctx, ch.ID, 10, time.Minute)
	if err != nil || len(first) != 1 {
		t.Fatalf(": %v (%d )", err, len(first))
	}
	if _, err := d.Exec(`UPDATE notification_deliveries SET next_attempt_at = now() - interval '1 minute' WHERE id=$1`, first[0].ID); err != nil {
		t.Fatal(err)
	}
	second, err := d.ClaimRealtimeDeliveries(ctx, ch.ID, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Fatalf(" sending ， %d ", len(second))
	}
	if second[0].Attempts != 2 {
		t.Fatalf("， %d", second[0].Attempts)
	}
}

func TestClaimSkipsDisabledChannel(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	addTestEvent(t, d, notify.EventFindingCreated, 5001, notify.Snapshot{Severity: "high"})
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if err := d.SetNotificationChannelEnabled(ctx, ch.ID, false); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := d.QueryRow(`SELECT state FROM notification_deliveries WHERE channel_id=$1`, ch.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != NotifyStateSkipped {
		t.Fatalf(" skipped， %s", state)
	}
	got, err := d.ClaimRealtimeDeliveries(ctx, ch.ID, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("， %d ", len(got))
	}
}

func TestDigestBatchDueAndStableBatchID(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeDigest, `{}`)
	for i := 0; i < 3; i++ {
		addTestEvent(t, d, notify.EventFindingCreated, int64(6000+i), notify.Snapshot{Severity: "high"})
	}
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}

	due, err := d.DigestBatchDue(ctx, ch.ID, 30*time.Minute)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if due {
		t.Fatal("")
	}

	if _, err := d.Exec(`UPDATE notification_deliveries SET created_at = now() - interval '40 minutes' WHERE channel_id=$1`, ch.ID); err != nil {
		t.Fatal(err)
	}
	due, err = d.DigestBatchDue(ctx, ch.ID, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		t.Fatal("")
	}

	batch, err := d.ClaimDigestBatch(ctx, ch.ID, MaxDigestBatchSize, time.Minute)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if len(batch) != 3 {
		t.Fatalf(" 3 ， %d ", len(batch))
	}
	if batch[0].BatchID == nil {
		t.Fatal(" batch_id，")
	}
	firstBatchID := *batch[0].BatchID
	for _, dl := range batch {
		if dl.BatchID == nil || *dl.BatchID != firstBatchID {
			t.Fatalf(" batch_id， %v vs %d", dl.BatchID, firstBatchID)
		}
	}

	// ****，batch_id （COALESCE ）：
	//
	allIDs := make([]int64, 0, len(batch))
	for _, dl := range batch {
		allIDs = append(allIDs, dl.ID)
	}
	if err := d.RescheduleDeliveries(ctx, allIDs, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`UPDATE notification_deliveries SET next_attempt_at = now() - interval '1 minute' WHERE channel_id=$1`, ch.ID); err != nil {
		t.Fatal(err)
	}
	reclaimed, err := d.ClaimDigestBatch(ctx, ch.ID, MaxDigestBatchSize, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(reclaimed) != 3 {
		t.Fatalf(" 3 ， %d", len(reclaimed))
	}
	if reclaimed[0].BatchID == nil || *reclaimed[0].BatchID != firstBatchID {
		t.Fatalf(" batch_id  %d， %v", firstBatchID, reclaimed[0].BatchID)
	}
}

func TestDeliveryStateTransitions(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	addTestEvent(t, d, notify.EventFindingCreated, 7001, notify.Snapshot{Severity: "high"})
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	got, err := d.ClaimRealtimeDeliveries(ctx, ch.ID, 10, time.Minute)
	if err != nil || len(got) != 1 {
		t.Fatalf(": %v (%d)", err, len(got))
	}
	id := got[0].ID

	if err := d.RescheduleDeliveries(ctx, []int64{id}, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	var state, lastErr string
	if err := d.QueryRow(`SELECT state, last_error FROM notification_deliveries WHERE id=$1`, id).Scan(&state, &lastErr); err != nil {
		t.Fatal(err)
	}
	if state != NotifyStatePending || lastErr != "" {
		t.Fatalf(" pending ， state=%s err=%q", state, lastErr)
	}

	if err := d.FailDeliveries(ctx, []int64{id}, ""); err != nil {
		t.Fatal(err)
	}
	if err := d.QueryRow(`SELECT state FROM notification_deliveries WHERE id=$1`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != NotifyStateFailed {
		t.Fatalf(" failed， %s", state)
	}

	if err := d.RetryNotificationDelivery(ctx, id); err != nil {
		t.Fatalf(": %v", err)
	}
	var attempts int
	var next time.Time
	if err := d.QueryRow(`SELECT state, attempts, next_attempt_at FROM notification_deliveries WHERE id=$1`, id).Scan(&state, &attempts, &next); err != nil {
		t.Fatal(err)
	}
	if state != NotifyStatePending || attempts != 0 {
		t.Fatalf(" pending  attempts=0， state=%s attempts=%d", state, attempts)
	}
	if next.After(time.Now().Add(time.Second)) {
		t.Fatal("")
	}

	if err := d.MarkDeliveriesSent(ctx, []int64{id}); err != nil {
		t.Fatal(err)
	}
	if err := d.RetryNotificationDelivery(ctx, id); err == nil {
		t.Fatal("")
	}
}

func TestListNotificationDeliveriesPagingAndFilter(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	for i := 0; i < 5; i++ {
		addTestEvent(t, d, notify.EventFindingCreated, int64(8000+i), notify.Snapshot{Severity: "high", Name: ""})
	}
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ClaimRealtimeDeliveries(ctx, ch.ID, 10, time.Minute); err != nil {
		t.Fatal(err)
	}

	page1, total, err := d.ListNotificationDeliveries(ctx, NotificationDeliveryFilter{ChannelID: ch.ID, State: NotifyStateSending}, 1, 2)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if total != 5 {
		t.Fatalf(" 5， %d", total)
	}
	if len(page1) != 2 {
		t.Fatalf(" 2 ， %d", len(page1))
	}
	page2, _, err := d.ListNotificationDeliveries(ctx, NotificationDeliveryFilter{ChannelID: ch.ID, State: NotifyStateSending}, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || page2[0].ID >= page1[0].ID {
		t.Fatalf("， page1[0]=%d page2[0]=%d", page1[0].ID, page2[0].ID)
	}
	if page1[0].ChannelName == "" || page1[0].FindingID == 0 {
		t.Fatalf(": %+v", page1[0])
	}

	pending, totalPending, err := d.ListNotificationDeliveries(ctx, NotificationDeliveryFilter{ChannelID: ch.ID, State: NotifyStatePending}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if totalPending != 0 || len(pending) != 0 {
		t.Fatalf(" pending ， %d  (total=%d)", len(pending), totalPending)
	}
}

func TestSetFindingStatusWithNotifyOnlyEmitsOnRealChange(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	tk, err := d.CreateTask("", "", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(tk.ID)
	es := d.Exploration(tk.ExplorationID)
	f, err := es.RecordFinding(ctx, RecordFindingInput{
		TaskID: tk.ID, Worker: "test", VulnClass: "SQL", Name: "",
		Severity: "high", Summary: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM notification_events WHERE finding_id=$1`, f.FindingID) })

	//  finding_created ，。
	var base int
	if err := d.QueryRow(`SELECT count(*) FROM notification_events WHERE finding_id=$1`, f.FindingID).Scan(&base); err != nil {
		t.Fatal(err)
	}
	if base < 1 {
		t.Fatal("")
	}

	from, found, notified, err := d.SetFindingStatusWithNotify(ctx, f.FindingID, "pending")
	if err != nil || !found {
		t.Fatalf(": found=%v err=%v", found, err)
	}
	if notified {
		t.Fatal("")
	}
	if from != "pending" {
		t.Fatalf(" pending， %q", from)
	}

	from, found, notified, err = d.SetFindingStatusWithNotify(ctx, f.FindingID, "fixed")
	if err != nil || !found {
		t.Fatalf(": found=%v err=%v", found, err)
	}
	if !notified {
		t.Fatal("")
	}
	if from != "pending" {
		t.Fatalf("from  pending， %q", from)
	}
	var snapshot []byte
	if err := d.QueryRow(`SELECT snapshot FROM notification_events WHERE finding_id=$1 AND kind=$2`,
		f.FindingID, notify.EventFindingStatusChanged).Scan(&snapshot); err != nil {
		t.Fatalf(": %v", err)
	}
	var snap notify.Snapshot
	if err := json.Unmarshal(snapshot, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.FromStatus != "pending" || snap.ToStatus != "fixed" {
		t.Fatalf(": %s → %s", snap.FromStatus, snap.ToStatus)
	}
	if snap.VulnClass != "SQL" || snap.Severity != "high" || snap.Name != "" {
		t.Fatalf(": %+v", snap)
	}
	var status string
	if err := d.QueryRow(`SELECT status FROM findings WHERE id=$1`, f.FindingID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "fixed" {
		t.Fatalf(" fixed， %s", status)
	}

	if _, found, _, err := d.SetFindingStatusWithNotify(ctx, 999999999, "fixed"); err != nil || found {
		t.Fatalf(" found=false ， found=%v err=%v", found, err)
	}
}

func TestNotificationStatsSnapshot(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	addTestEvent(t, d, notify.EventFindingCreated, 9001, notify.Snapshot{Severity: "high"})
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	stats, err := d.NotificationStatsSnapshot(ctx)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if stats.Channels < 1 || stats.ChannelsOn < 1 {
		t.Fatalf(": %+v", stats)
	}
	if stats.Pending < 1 {
		t.Fatalf(": %+v", stats)
	}
	if stats.BacklogAgeMS < 0 || stats.BacklogAgeMS > int64(time.Hour/time.Millisecond) {
		t.Fatalf(": %d ms", stats.BacklogAgeMS)
	}
	_ = ch
}

func TestNotificationChannelCRUDRoundTrip(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	ch := &NotificationChannel{
		Name:       "CRUD ",
		Kind:       notify.KindEmail,
		Mode:       NotifyModeDigest,
		Config:     json.RawMessage(`{"host":"smtp.example.com","port":587,"from":"a@b.c","to":["x@y.z"]}`),
		Filter:     json.RawMessage(`{"min_severity":"medium","on_status_change":true}`),
		RatePerMin: 42,
	}
	id, err := d.SaveNotificationChannel(ctx, ch)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM notification_channels WHERE id=$1`, id) })

	got, err := d.NotificationChannelByID(ctx, id)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if got.Mode != NotifyModeDigest || got.RatePerMin != 42 || got.Name != "CRUD " {
		t.Fatalf(": %+v", got)
	}
	if !got.IsEnabled() {
		t.Fatal("")
	}
	var cfg map[string]any
	if err := json.Unmarshal(got.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "smtp.example.com" {
		t.Fatalf(": %v", cfg)
	}
	var filter notify.Filter
	if err := json.Unmarshal(got.Filter, &filter); err != nil {
		t.Fatal(err)
	}
	if filter.MinSeverity != "medium" || !filter.OnStatusChange {
		t.Fatalf(": %+v", filter)
	}

	got.Name = ""
	off := false
	got.Enabled = &off
	if _, err := d.SaveNotificationChannel(ctx, got); err != nil {
		t.Fatal(err)
	}
	after, err := d.NotificationChannelByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Name != "" || after.IsEnabled() {
		t.Fatalf(": %+v", after)
	}

	if err := d.DeleteNotificationChannel(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := d.NotificationChannelByID(ctx, id); err != ErrNotificationChannelNotFound {
		t.Fatalf(" ErrNotificationChannelNotFound， %v", err)
	}
	if err := d.DeleteNotificationChannel(ctx, id); err != ErrNotificationChannelNotFound {
		t.Fatalf("， %v", err)
	}
}

// TestSaveNotificationChannelKeepsExplicitZeroRate ：
//
//  bug：SaveNotificationChannel  `if RatePerMin <= 0 {  }`，
// 、UI 、takeTokens 「0=」，
//  server （ notifyCreateChannel），db 。
func TestSaveNotificationChannelKeepsExplicitZeroRate(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	unlimited := &NotificationChannel{
		Name: "", Kind: notify.KindDingTalk, RatePerMin: 0,
		Config: json.RawMessage(`{"webhook":"https://example.com/h"}`),
	}
	id, err := d.SaveNotificationChannel(ctx, unlimited)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM notification_channels WHERE id=$1`, id) })
	got, err := d.NotificationChannelByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.RatePerMin != 0 {
		t.Fatalf(" 0 ，， %d", got.RatePerMin)
	}
	if got.Mode != NotifyModeRealtime {
		t.Fatalf(" realtime， %s", got.Mode)
	}

	bad := &NotificationChannel{
		Name: "", Kind: notify.KindDingTalk, RatePerMin: -1,
		Config: json.RawMessage(`{"webhook":"https://example.com/h"}`),
	}
	if _, err := d.SaveNotificationChannel(ctx, bad); err == nil {
		t.Fatal("")
	}
}

// TestDeleteChannelCascadesDeliveries ：
func TestDeleteChannelCascadesDeliveries(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeRealtime, `{}`)
	ev := addTestEvent(t, d, notify.EventFindingCreated, 9101, notify.Snapshot{Severity: "high"})
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := d.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE channel_id=$1`, ch.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before == 0 {
		t.Fatal("：")
	}
	if err := d.DeleteNotificationChannel(ctx, ch.ID); err != nil {
		t.Fatal(err)
	}
	var after int
	if err := d.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE channel_id=$1`, ch.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Fatalf("， %d ", after)
	}
	var evExists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM notification_events WHERE id=$1)`, ev).Scan(&evExists); err != nil {
		t.Fatal(err)
	}
	if !evExists {
		t.Fatal("")
	}
}

// TestClaimDigestBatchHonorsCallerLimit ：
// ——allow  takeTokens ，
// rate_per_min  digest 。 limit 。
func TestClaimDigestBatchHonorsCallerLimit(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()
	ch := newTestChannel(t, d, notify.KindDingTalk, NotifyModeDigest, `{}`)
	for i := 0; i < 10; i++ {
		addTestEvent(t, d, notify.EventFindingCreated, int64(7000+i), notify.Snapshot{Severity: "high"})
	}
	if _, _, err := d.FanOutPendingEvents(ctx, 100); err != nil {
		t.Fatal(err)
	}
	got, err := d.ClaimDigestBatch(ctx, ch.ID, 3, time.Minute)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if len(got) != 3 {
		t.Fatalf(" 3 ， %d", len(got))
	}
	if got, err := d.ClaimDigestBatch(ctx, ch.ID, 0, time.Minute); err != nil || len(got) != 0 {
		t.Fatalf(" 0  0 ， %d  err=%v", len(got), err)
	}
}

// TestFinishFindingRetestEmitsStatusChange ：
// —— on_status_change 
//
func TestFinishFindingRetestEmitsStatusChange(t *testing.T) {
	d := notifyTestDB(t)
	ctx := context.Background()

	tk, err := d.CreateTask("", "", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(tk.ID)
	es := d.Exploration(tk.ExplorationID)
	f, err := es.RecordFinding(ctx, RecordFindingInput{
		TaskID: tk.ID, Worker: "test", VulnClass: "SQL", Name: "",
		Severity: "high", Summary: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM notification_events WHERE finding_id=$1`, f.FindingID) })

	rt, _, _, err := d.CreateFindingRetest(ctx, f.FindingID, "")
	if err != nil {
		t.Fatal(err)
	}
	if rt.ConversationID == nil {
		t.Fatal("")
	}
	if ok, err := d.StartFindingRetest(ctx, rt.ID); err != nil || !ok {
		t.Fatalf(": ok=%v err=%v", ok, err)
	}
	if err := d.RecordFindingRetestResult(ctx, *rt.ConversationID, "fixed", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := d.FinishFindingRetest(rt.ID, "completed", ""); err != nil {
		t.Fatalf(": %v", err)
	}

	var status string
	if err := d.QueryRow(`SELECT status FROM findings WHERE id=$1`, f.FindingID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != FindingFixed {
		t.Fatalf(" fixed， %s", status)
	}

	var snapshot []byte
	err = d.QueryRow(`SELECT snapshot FROM notification_events WHERE finding_id=$1 AND kind=$2 ORDER BY id DESC LIMIT 1`,
		f.FindingID, notify.EventFindingStatusChanged).Scan(&snapshot)
	if err != nil {
		t.Fatalf("（ on_status_change ）: %v", err)
	}
	var snap notify.Snapshot
	if err := json.Unmarshal(snapshot, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.FromStatus != "pending" || snap.ToStatus != FindingFixed {
		t.Fatalf(": %s → %s", snap.FromStatus, snap.ToStatus)
	}
	if snap.Name != "" || snap.Severity != "high" {
		t.Fatalf(": %+v", snap)
	}
}
