package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/notify"
)

const (
	// settingNotifyEnabled 。：，
	settingNotifyEnabled = "notify_enabled"
	// settingNotifyPublicBaseURL 
	// （ https://artex.example.com）。。
	settingNotifyPublicBaseURL = "notify_public_base_url"
	// settingNotifyDigestMinutes （）。
	settingNotifyDigestMinutes = "notify_digest_interval_min"
)

const (
	notifyTick = 3 * time.Second
	// notifyBackgroundDisabledEnv      
	// (newNotifier(s).Run)  .   stepRealtime·stepDigest 
	notifyBackgroundDisabledEnv = "ARTEX_NOTIFY_BACKGROUND_DISABLED"
	notifyLease = 3 * time.Minute
	// notifyFanOutPerTick ，
	notifyFanOutPerTick = 200
	// notifyDefaultDigestMinutes 。
	notifyDefaultDigestMinutes = 30
	// notifyUnlimitedBurstPerTick 。
	notifyUnlimitedBurstPerTick = 50
	// notifyMaxSendsPerChannelPerTick 。
	//
	// （Run  goroutine ，tick ），**
	//
	// TestNotifyTickBudgetFitsWithinLease —— notifyLease、
	// notifySendTimeout 。
	notifyMaxSendsPerChannelPerTick = 5
	// notifySendTimeout 。，
	//  notifyLease， TestNotifyTickBudgetFitsWithinLease。
	notifySendTimeout = 30 * time.Second
)

// notifyBackoff ，。
// 3 （） db.MaxNotifyAttempts ，。
var notifyBackoff = []time.Duration{
	time.Second,
	5 * time.Second,
	30 * time.Second,
}

//  · . notification_deliveries.last_error  
// notify_api.go        「 」 (delivery-list.tsx)
const (
	errDeliveryChannelKindUnregistered = "unregistered channel type: %q"
	errDeliverySnapshotUnrenderable    = "        "
	errDeliveryChannelLengthCapped     = "message hit channel length cap; sent first %d, deferring the rest to next batch"
	errDeliveryNoDeliveredCount        = "channel did not report delivery count (delivered=%d)"
	errDeliveryRetryExhausted          = "failed after %d retries: %s"
	errDeliverySnapshotEmpty           = "  #%d     "
	errDeliverySnapshotParse           = "  #%d     : %w"
	errDeliveryBatchAllUnparseable     = "   %d   "
)

//
//  Scheduler ， goroutine （ server.New）。
// Scheduler  tick：（3 ），
type Notifier struct {
	s  *Server
	pg *db.DB

	mu      sync.Mutex
	buckets map[int64]*notifyBucket
}

// notifyBucket 。
//
type notifyBucket struct {
	tokens   float64
	lastFill time.Time
}

func newNotifier(s *Server) *Notifier {
	return &Notifier{s: s, pg: s.m.pg, buckets: map[int64]*notifyBucket{}}
}

// Run  ctx 。 server.New 。
func (n *Notifier) Run(ctx context.Context) {
	if n.pg == nil {
		return
	}
	t := time.NewTicker(notifyTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n.step(ctx)
		}
	}
}

//
func (n *Notifier) step(ctx context.Context) {
	if !n.enabled() {
		return
	}
	if _, _, err := n.pg.FanOutPendingEvents(ctx, notifyFanOutPerTick); err != nil {
		log.Printf("[notify] : %v", err)
		return
	}
	channels, err := n.pg.ListNotificationChannels(ctx)
	if err != nil {
		log.Printf("[notify] : %v", err)
		return
	}
	baseURL := n.publicBaseURL()
	for _, ch := range channels {
		if !ch.IsEnabled() {
			continue
		}
		//
		now := time.Now()
		if ch.Mode == db.NotifyModeDigest {
			tokens, claimLimit := digestTickPlan()
			if n.takeTokens(ch.ID, ch.RatePerMin, tokens, now) <= 0 {
				continue
			}
			n.stepDigest(ctx, ch, claimLimit, baseURL)
			continue
		}
		allow := n.takeTokens(ch.ID, ch.RatePerMin, notifyMaxSendsPerChannelPerTick, now)
		if allow <= 0 {
			continue
		}
		n.stepRealtime(ctx, ch, allow, baseURL)
	}
}

// digestTickPlan 。
//
//
//     rate_per_min  digest （）。
//
//  rate_per_min  digest ，
// （notifyMaxSendsPerChannelPerTick，）。
//  rate_per_min=20  3  tick  1 ，
// 「 30  1 」， db.MaxDigestBatchSize 。
//
// stepDigest， step ），
// TestDigestTickPlanDecouplesBatchSizeFromSendBudget 。
func digestTickPlan() (tokens, claimLimit int) {
	return 1, db.MaxDigestBatchSize
}

// stepRealtime ，。
func (n *Notifier) stepRealtime(ctx context.Context, ch *db.NotificationChannel, allow int, baseURL string) {
	deliveries, err := n.pg.ClaimRealtimeDeliveries(ctx, ch.ID, allow, notifyLease)
	if err != nil {
		log.Printf("[notify]  channel=%d: %v", ch.ID, err)
		return
	}
	if len(deliveries) == 0 {
		return
	}
	channel, cfg, ok := n.adapt(ch)
	if !ok {
		_ = n.pg.FailDeliveries(ctx, deliveryIDs(deliveries), fmt.Sprintf(errDeliveryChannelKindUnregistered, ch.Kind))
		return
	}
	for _, dl := range deliveries {
		msg, err := n.renderSingle(ctx, dl, baseURL)
		if err != nil {
			_ = n.pg.FailDeliveries(ctx, []int64{dl.ID}, err.Error())
			continue
		}
		n.send(ctx, channel, cfg, msg, []*db.NotificationDelivery{dl})
	}
}

func (n *Notifier) stepDigest(ctx context.Context, ch *db.NotificationChannel, allow int, baseURL string) {
	window := n.digestInterval()
	due, err := n.pg.DigestBatchDue(ctx, ch.ID, window)
	if err != nil {
		log.Printf("[notify]  channel=%d: %v", ch.ID, err)
		return
	}
	if !due {
		return
	}
	deliveries, err := n.pg.ClaimDigestBatch(ctx, ch.ID, allow, notifyLease)
	if err != nil {
		log.Printf("[notify]  channel=%d: %v", ch.ID, err)
		return
	}
	if len(deliveries) == 0 {
		return
	}
	channel, cfg, ok := n.adapt(ch)
	if !ok {
		_ = n.pg.FailDeliveries(ctx, deliveryIDs(deliveries), fmt.Sprintf(errDeliveryChannelKindUnregistered, ch.Kind))
		return
	}
	msg, included, err := n.renderBatch(ctx, deliveries, baseURL, int(window.Minutes()))
	if err != nil {
		_ = n.pg.FailDeliveries(ctx, deliveryIDs(deliveries), err.Error())
		return
	}
	if skipped := excludeDeliveries(deliveries, included); len(skipped) > 0 {
		reason := errDeliverySnapshotUnrenderable
		if fErr := n.pg.FailDeliveries(ctx, deliveryIDs(skipped), reason); fErr != nil {
			log.Printf("[notify]  channel=%s ids=%v: %v", ch.Kind, deliveryIDs(skipped), fErr)
		}
		log.Printf("[notify]  %d  channel=%d", len(skipped), ch.ID)
	}
	//  send：included[i]  msg.Items[i] ，
	n.send(ctx, channel, cfg, msg, included)
}

//
//
func (n *Notifier) send(ctx context.Context, channel notify.Channel, cfg map[string]any, msg notify.Message, deliveries []*db.NotificationDelivery) {
	sendCtx, cancel := context.WithTimeout(ctx, notifySendTimeout)
	defer cancel()
	delivered, err := channel.Send(sendCtx, cfg, msg)
	if err == nil && delivered > 0 {
		if delivered > len(deliveries) {
			log.Printf("[notify]  %d  %d channel=%s，",
				delivered, len(deliveries), channel.Kind())
			delivered = len(deliveries)
		}
		sent, rest := deliveries[:delivered], deliveries[delivered:]
		if err := n.pg.MarkDeliveriesSent(ctx, deliveryIDs(sent)); err != nil {
			log.Printf("[notify]  channel=%s ids=%v: %v", channel.Kind(), deliveryIDs(sent), err)
		}
		if len(rest) > 0 {
			//  DeferDeliveries  RescheduleDeliveries —— ，
			if err := n.pg.DeferDeliveries(ctx, deliveryIDs(rest),
				fmt.Sprintf(errDeliveryChannelLengthCapped, delivered)); err != nil {
				log.Printf("[notify]  channel=%s ids=%v: %v", channel.Kind(), deliveryIDs(rest), err)
			}
		}
		return
	}
	if err == nil {
		err = fmt.Errorf(errDeliveryNoDeliveredCount, delivered)
	}

	//
	//  `if maxAttempts(deliveries) >= MaxNotifyAttempts` ，
	// （attempts=1） failed——，
	permanent := notify.IsPermanent(err)
	var failIDs, exhaustedIDs []int64
	byDelay := map[time.Duration][]int64{}
	for _, dl := range deliveries {
		switch {
		case permanent:
			failIDs = append(failIDs, dl.ID)
		case dl.Attempts >= db.MaxNotifyAttempts:
			exhaustedIDs = append(exhaustedIDs, dl.ID)
		default:
			delay := notifyBackoff[min(dl.Attempts, len(notifyBackoff)-1)]
			byDelay[delay] = append(byDelay[delay], dl.ID)
		}
	}

	if len(failIDs) > 0 {
		if fErr := n.pg.FailDeliveries(ctx, failIDs, err.Error()); fErr != nil {
			log.Printf("[notify]  channel=%s ids=%v: %v", channel.Kind(), failIDs, fErr)
		}
	}
	if len(exhaustedIDs) > 0 {
		reason := fmt.Sprintf(errDeliveryRetryExhausted, db.MaxNotifyAttempts, err)
		if fErr := n.pg.FailDeliveries(ctx, exhaustedIDs, reason); fErr != nil {
			log.Printf("[notify]  channel=%s ids=%v: %v", channel.Kind(), exhaustedIDs, fErr)
		}
	}
	for delay, group := range byDelay {
		if rErr := n.pg.RescheduleDeliveries(ctx, group, delay, err.Error()); rErr != nil {
			log.Printf("[notify]  channel=%s ids=%v: %v", channel.Kind(), group, rErr)
		}
	}
	if len(failIDs)+len(exhaustedIDs) > 0 {
		log.Printf("[notify]  channel=%d kind=%s =%d =%d =%d: %s",
			deliveries[0].ChannelID, channel.Kind(), len(failIDs), len(exhaustedIDs), len(byDelay), err)
	}
}

// excludeDeliveries  all  keep （）。
func excludeDeliveries(all, keep []*db.NotificationDelivery) []*db.NotificationDelivery {
	inKeep := make(map[*db.NotificationDelivery]bool, len(keep))
	for _, dl := range keep {
		inKeep[dl] = true
	}
	var out []*db.NotificationDelivery
	for _, dl := range all {
		if !inKeep[dl] {
			out = append(out, dl)
		}
	}
	return out
}

func (n *Notifier) adapt(ch *db.NotificationChannel) (notify.Channel, map[string]any, bool) {
	channel, ok := notify.Get(ch.Kind)
	if !ok {
		return nil, nil, false
	}
	var cfg map[string]any
	if len(ch.Config) > 0 {
		_ = json.Unmarshal(ch.Config, &cfg)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return channel, cfg, true
}

// renderSingle 。
func (n *Notifier) renderSingle(ctx context.Context, dl *db.NotificationDelivery, baseURL string) (notify.Message, error) {
	snap, err := parseSnapshot(dl)
	if err != nil {
		return notify.Message{}, err
	}
	item, err := n.itemFor(ctx, snap, baseURL)
	if err != nil {
		return notify.Message{}, err
	}
	return notify.Message{Items: []notify.Item{item}, HomeURL: baseURL}, nil
}

//
//  included  msg.Items ****（ i  ↔  i ）。
func (n *Notifier) renderBatch(ctx context.Context, deliveries []*db.NotificationDelivery, baseURL string, windowMinutes int) (notify.Message, []*db.NotificationDelivery, error) {
	items := make([]notify.Item, 0, len(deliveries))
	included := make([]*db.NotificationDelivery, 0, len(deliveries))
	for _, dl := range deliveries {
		snap, err := parseSnapshot(dl)
		if err != nil {
			log.Printf("[notify]  delivery=%d: %v", dl.ID, err)
			continue
		}
		item, err := n.itemFor(ctx, snap, baseURL)
		if err != nil {
			return notify.Message{}, nil, err
		}
		items = append(items, item)
		included = append(included, dl)
	}
	if len(items) == 0 {
		return notify.Message{}, nil, fmt.Errorf(errDeliveryBatchAllUnparseable, len(deliveries))
	}
	return notify.Message{
		Items:         items,
		Batch:         true,
		WindowMinutes: windowMinutes,
		HomeURL:       baseURL,
	}, included, nil
}

func (n *Notifier) itemFor(ctx context.Context, snap notify.Snapshot, baseURL string) (notify.Item, error) {
	assets, err := n.pg.NotificationAssetNames(ctx, snap.AssetIDs)
	if err != nil {
		log.Printf("[notify]  finding=%d: %v", snap.FindingID, err)
	}
	item := notify.Item{
		FindingID:  snap.FindingID,
		Name:       snap.Name,
		VulnClass:  snap.VulnClass,
		Severity:   snap.Severity,
		Summary:    snap.Summary,
		Assets:     assets,
		FromStatus: snap.FromStatus,
		ToStatus:   snap.ToStatus,
	}
	if baseURL != "" {
		//  web/src/app/(main)/function/findings/detail/page.tsx，
		item.DetailURL = fmt.Sprintf("%s/function/findings/detail?id=%d", baseURL, snap.FindingID)
	}
	return item, nil
}

// takeTokens ** want **，。
//
//
//
func (n *Notifier) takeTokens(channelID int64, ratePerMin, want int, now time.Time) int {
	if want <= 0 {
		return 0
	}
	if ratePerMin <= 0 {
		return min(want, notifyUnlimitedBurstPerTick)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	b := n.buckets[channelID]
	if b == nil {
		b = &notifyBucket{tokens: float64(ratePerMin), lastFill: now}
		n.buckets[channelID] = b
	}
	if elapsed := now.Sub(b.lastFill).Seconds(); elapsed > 0 {
		b.tokens = minF(float64(ratePerMin), b.tokens+elapsed*float64(ratePerMin)/60)
		b.lastFill = now
	}
	take := min(int(b.tokens+1e-9), want)
	if take <= 0 {
		return 0
	}
	b.tokens -= float64(take)
	return take
}

func (n *Notifier) enabled() bool {
	return n.pg.GetBool(settingNotifyEnabled, true)
}

// publicBaseURL ，。
func (n *Notifier) publicBaseURL() string {
	v, ok, err := n.pg.GetSetting(settingNotifyPublicBaseURL)
	if err != nil || !ok {
		return ""
	}
	return trimTrailingSlash(v)
}

// digestInterval ，。
func (n *Notifier) digestInterval() time.Duration {
	v, ok, err := n.pg.GetSetting(settingNotifyDigestMinutes)
	if err != nil || !ok {
		return time.Duration(notifyDefaultDigestMinutes) * time.Minute
	}
	m := 0
	if _, err := fmt.Sscanf(v, "%d", &m); err != nil || m <= 0 {
		return time.Duration(notifyDefaultDigestMinutes) * time.Minute
	}
	return time.Duration(m) * time.Minute
}

// parseSnapshot 。
func parseSnapshot(dl *db.NotificationDelivery) (notify.Snapshot, error) {
	var snap notify.Snapshot
	if len(dl.Snapshot) == 0 {
		return snap, fmt.Errorf(errDeliverySnapshotEmpty, dl.ID)
	}
	if err := json.Unmarshal(dl.Snapshot, &snap); err != nil {
		return snap, fmt.Errorf(errDeliverySnapshotParse, dl.ID, err)
	}
	if snap.Kind == "" {
		snap.Kind = dl.EventKind
	}
	return snap, nil
}

func deliveryIDs(deliveries []*db.NotificationDelivery) []int64 {
	out := make([]int64, 0, len(deliveries))
	for _, dl := range deliveries {
		out = append(out, dl.ID)
	}
	return out
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
