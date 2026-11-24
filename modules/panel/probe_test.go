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

func TestSetResolutionPersistsOnImportedChannel(t *testing.T) {
	s := testService(t)
	imp := Channel{
		ID:         "51",
		Name:       "CCTV-1",
		OperatorID: "ch-cctv1",
		Group:      "央视",
		URL:        "igmp://239.10.0.1:5140",
		Enabled:    true,
		Source:     "iptv",
	}
	if err := s.Store.Import([]Channel{imp}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SetResolution(imp, "1920x1080", true); err != nil {
		t.Fatal(err)
	}
	got, err := s.Channel("ch-cctv1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Resolution != "1920x1080" {
		t.Fatalf("expected imported channel resolution 1920x1080, got %q", got.Resolution)
	}
}
