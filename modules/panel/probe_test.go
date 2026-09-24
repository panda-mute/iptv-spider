package panel

import (
	"context"
	"testing"
)

func TestProbeJobLifecycleAndValidation(t *testing.T) {
	s := testService(t)

	// No channels to probe initially
	err := s.StartProbe(context.Background(), false)
	if err == nil {
		t.Fatal("expected error when no channels to probe, got nil")
	}

	// Add channel
	ch := Channel{
		Key:     "test-1",
		ID:      "1",
		Name:    "测试1",
		Group:   "卫视频道",
		URL:     "rtp://239.1.1.1:5140",
		Enabled: true,
	}
	if err := s.Store.Discover(ch); err != nil {
		t.Fatal(err)
	}

	status := s.ProbeStatus()
	if status["state"] != "idle" {
		t.Fatalf("expected idle probe state, got %v", status["state"])
	}

	if err := s.StartProbe(context.Background(), false); err != nil {
		t.Fatalf("StartProbe failed: %v", err)
	}

	runningStatus := s.ProbeStatus()
	if runningStatus["state"] != "running" && runningStatus["state"] != "completed" {
		t.Fatalf("expected running/completed state, got %v", runningStatus["state"])
	}

	s.StopProbe()
	stoppedStatus := s.ProbeStatus()
	if stoppedStatus["state"] == "running" {
		t.Fatalf("expected non-running state after stop, got %v", stoppedStatus["state"])
	}
}
