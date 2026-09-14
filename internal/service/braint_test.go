package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendStatus_Success(t *testing.T) {
	var onlineCalled int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agent/online" {
			atomic.AddInt32(&onlineCalled, 1)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_ = os.Setenv("LARAVEL_API_URL", ts.URL+"/api/v1/agent")
	defer os.Unsetenv("LARAVEL_API_URL")

	payload := HearbeatPayload{
		AgentName: "Test Bot",
		Version:   "1.0.0",
		Interval:  "10 Min",
		Status:    "Active",
	}

	err := SendStatus("/online", payload)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if atomic.LoadInt32(&onlineCalled) != 1 {
		t.Fatalf("expected 1 online call, got %d", onlineCalled)
	}
}

func TestSendStatus_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_ = os.Setenv("LARAVEL_API_URL", ts.URL+"/api/v1/agent")
	defer os.Unsetenv("LARAVEL_API_URL")

	payload := HearbeatPayload{
		AgentName: "Test Bot",
	}

	err := SendStatus("/online", payload)
	if err == nil {
		t.Fatalf("expected error for 500 status code, got nil")
	}
}

func TestHeartbeatWorker_AutoReconnect(t *testing.T) {
	var onlineCalled int32
	var heartbeatCalled int32
	var serverEnabled int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&serverEnabled) == 0 {
			// Simulate server down
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		if r.URL.Path == "/api/v1/agent/online" {
			atomic.AddInt32(&onlineCalled, 1)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/v1/agent/heartbeat" {
			atomic.AddInt32(&heartbeatCalled, 1)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_ = os.Setenv("LARAVEL_API_URL", ts.URL+"/api/v1/agent")
	defer os.Unsetenv("LARAVEL_API_URL")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	payload := HearbeatPayload{
		AgentName: "Test Bot",
		Version:   "1.0.0",
		Status:    "Active",
	}

	// Server is initially DOWN
	atomic.StoreInt32(&serverEnabled, 0)
	SetLaravelConnected(false)

	go StartHeartbeatWorker(ctx, payload)

	// Wait 1 second - server is down, so IsLaravelConnected must be false
	time.Sleep(1 * time.Second)
	if IsLaravelConnected() {
		t.Fatalf("expected isLaravelConnected to be false initially")
	}

	// Now start/enable the server!
	atomic.StoreInt32(&serverEnabled, 1)

	// Heartbeat worker checks reconnect every 5s, let's wait for it to reconnect
	deadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(deadline) {
		if IsLaravelConnected() && atomic.LoadInt32(&onlineCalled) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !IsLaravelConnected() {
		t.Fatalf("expected IsLaravelConnected to be true after server became available")
	}
	if atomic.LoadInt32(&onlineCalled) < 1 {
		t.Fatalf("expected at least 1 online call, got %d", onlineCalled)
	}
}
