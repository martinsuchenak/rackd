package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// verify checks a request the way the webhook docs tell receivers to.
func verify(t *testing.T, secret string, r *http.Request, body []byte) {
	t.Helper()
	ts := r.Header.Get(HeaderTimestamp)
	if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
		t.Fatalf("timestamp header %q: %v", ts, err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(body)))
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if got := r.Header.Get(HeaderSignature); !hmac.Equal([]byte(got), []byte(want)) {
		t.Errorf("signature = %q, want %q", got, want)
	}
}

func TestSignCoversTimestampAndBody(t *testing.T) {
	body := []byte(`{"a":1}`)
	if Sign("s", 100, body) == Sign("s", 101, body) {
		t.Error("signature does not depend on the timestamp")
	}
	if Sign("s", 100, body) == Sign("s", 100, []byte(`{"a":2}`)) {
		t.Error("signature does not depend on the body")
	}
	// HMAC-SHA256("s", "100.{"a":1}"), computed independently.
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write([]byte(`100.{"a":1}`))
	if got, want := Sign("s", 100, body), "sha256="+hex.EncodeToString(mac.Sum(nil)); got != want {
		t.Errorf("Sign = %s, want %s", got, want)
	}
}

type received struct {
	header http.Header
	msg    Message
	body   []byte
}

func receiver(t *testing.T, secret string, status func(n int) int) (*httptest.Server, func() []received) {
	t.Helper()
	var mu sync.Mutex
	var got []received
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		verify(t, secret, r, body)
		var msg Message
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Errorf("body is not a Message: %v: %s", err, body)
		}
		mu.Lock()
		got = append(got, received{header: r.Header.Clone(), msg: msg, body: body})
		n := len(got)
		mu.Unlock()
		w.WriteHeader(status(n))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []received {
		mu.Lock()
		defer mu.Unlock()
		return append([]received(nil), got...)
	}
}

func TestWorkerDeliversPublishedEvents(t *testing.T) {
	ctx := context.Background()
	store := newWebhookTestStore(t)
	defer func() { _ = store.Close() }()

	srv, got := receiver(t, "topsecret", func(int) int { return http.StatusNoContent })
	if err := store.CreateWebhook(ctx, &model.Webhook{Name: "devices", URL: srv.URL, Secret: "topsecret", Active: true,
		Events: []model.EventType{model.EventTypeDeviceCreated}}); err != nil {
		t.Fatal(err)
	}
	worker := NewWorker(store, DefaultDeliveryConfig())
	worker.deliveryService.client = srv.Client() // the secure client refuses loopback
	worker.Start()
	defer worker.Stop()

	at := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	GetEventBus().PublishSync(model.EventTypeNetworkCreated, map[string]string{"id": "net-1"}) // not subscribed
	worker.handleEvent(model.Event{Type: model.EventTypeDeviceCreated, Timestamp: at, Payload: map[string]string{"id": "dev-1", "hostname": "srv01"}})

	waitUntil(t, func() bool { return len(got()) == 1 })
	r := got()[0]
	if r.msg.Event != model.EventTypeDeviceCreated || !r.msg.Timestamp.Equal(at) || r.msg.DeliveryID == "" {
		t.Errorf("message = %+v", r.msg)
	}
	if data, _ := r.msg.Data.(map[string]any); data["hostname"] != "srv01" {
		t.Errorf("data = %v", r.msg.Data)
	}
	if r.header.Get(HeaderEvent) != "device.created" || r.header.Get(HeaderDelivery) != r.msg.DeliveryID {
		t.Errorf("headers = %v", r.header)
	}

	waitUntil(t, func() bool {
		ds, _ := store.ListDeliveries(ctx, &model.DeliveryFilter{})
		return len(ds) == 1 && ds[0].Status == model.DeliveryStatusSuccess && ds[0].ResponseCode == http.StatusNoContent && ds[0].ID == r.msg.DeliveryID
	})
}

func TestRetryKeepsDeliveryIDAndResigns(t *testing.T) {
	ctx := context.Background()
	store := newWebhookTestStore(t)
	defer func() { _ = store.Close() }()

	srv, got := receiver(t, "k", func(n int) int {
		if n == 1 {
			return http.StatusServiceUnavailable
		}
		return http.StatusOK
	})
	webhook := &model.Webhook{Name: "retry", URL: srv.URL, Secret: "k", Active: true, Events: []model.EventType{model.EventTypeDeviceUpdated}}
	if err := store.CreateWebhook(ctx, webhook); err != nil {
		t.Fatal(err)
	}
	svc := NewDeliveryService(store, DeliveryConfig{MaxRetries: 3, RetryBackoff: time.Millisecond, HTTPTimeout: time.Second, RetentionDays: 1})
	svc.client = srv.Client()

	first, err := svc.Deliver(ctx, webhook, model.Event{Type: model.EventTypeDeviceUpdated, Payload: map[string]string{"id": "dev-1"}})
	if err == nil || first.Status != model.DeliveryStatusRetrying || first.ResponseCode != http.StatusServiceUnavailable {
		t.Fatalf("first attempt = %+v, %v", first, err)
	}
	time.Sleep(1100 * time.Millisecond) // the next attempt is due, and the timestamp has moved on
	if ok, _, err := svc.ProcessPendingRetries(ctx); err != nil || ok != 1 {
		t.Fatalf("retry: %d succeeded, %v", ok, err)
	}

	attempts := got()
	if len(attempts) != 2 {
		t.Fatalf("got %d attempts", len(attempts))
	}
	if attempts[0].msg.DeliveryID != first.ID || attempts[1].msg.DeliveryID != first.ID || attempts[1].header.Get(HeaderDelivery) != first.ID {
		t.Errorf("delivery IDs: %s, %s, record %s", attempts[0].msg.DeliveryID, attempts[1].msg.DeliveryID, first.ID)
	}
	if string(attempts[0].body) != string(attempts[1].body) {
		t.Error("retry changed the body")
	}
	if attempts[0].header.Get(HeaderTimestamp) == attempts[1].header.Get(HeaderTimestamp) {
		t.Error("retry reused the timestamp, so a receiver rejecting old timestamps would drop it")
	}
}

func TestWorkerStopUnsubscribes(t *testing.T) {
	store := newWebhookTestStore(t)
	defer func() { _ = store.Close() }()
	before := len(GetEventBus().snapshot())
	worker := NewWorker(store, DefaultDeliveryConfig())
	worker.Start()
	if len(GetEventBus().snapshot()) != before+1 {
		t.Fatal("Start did not subscribe")
	}
	worker.Stop()
	worker.Stop() // idempotent
	if len(GetEventBus().snapshot()) != before {
		t.Error("Stop did not unsubscribe")
	}
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatal("condition not met within 3s")
}

func TestDeliveryInterruptedByShutdownIsKeptForRetry(t *testing.T) {
	store := newWebhookTestStore(t)
	defer func() { _ = store.Close() }()

	arrived := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The server only notices the client hanging up once the body is read.
		_, _ = io.Copy(io.Discard, r.Body)
		close(arrived)
		<-r.Context().Done() // hold the request until the client gives up
	}))
	defer srv.Close()
	webhook := &model.Webhook{Name: "slow", URL: srv.URL, Active: true, Events: []model.EventType{model.EventTypeDeviceCreated}}
	if err := store.CreateWebhook(context.Background(), webhook); err != nil {
		t.Fatal(err)
	}
	svc := NewDeliveryService(store, DefaultDeliveryConfig())
	svc.client = srv.Client()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-arrived
		cancel() // shutdown while the request is in flight
	}()
	delivery, err := svc.Deliver(ctx, webhook, model.Event{Type: model.EventTypeDeviceCreated, Payload: "x"})
	if err == nil || delivery.Status != model.DeliveryStatusRetrying {
		t.Fatalf("delivery = %+v, %v", delivery, err)
	}
	stored, err := store.ListDeliveries(context.Background(), &model.DeliveryFilter{WebhookID: webhook.ID})
	if err != nil || len(stored) != 1 || stored[0].Status != model.DeliveryStatusRetrying {
		t.Errorf("stored deliveries = %+v, %v; the interrupted delivery must be kept for retry", stored, err)
	}
}
