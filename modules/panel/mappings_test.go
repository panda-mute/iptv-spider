package panel

import (
	"path/filepath"
	"testing"
)

func TestDefaultKeywordMappingsAndCustomization(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "panel.json")
	store, err := OpenStore(storePath, DefaultSettings())
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	// 1. Verify default mappings exist
	mappings := store.Mappings()
	if len(mappings) != len(DefaultMappings()) {
		t.Fatalf("expected %d default mappings, got %d", len(DefaultMappings()), len(mappings))
	}

	mSports, found := store.FindMapping("体育频道")
	if !found || mSports.TargetName != "五星体育" || mSports.TargetID != "8" {
		t.Fatalf("expected 体育频道 -> 五星体育, got %+v", mSports)
	}

	mSportsHD, found := store.FindMapping("体育频道HD")
	if !found || mSportsHD.TargetName != "五星体育HD" || mSportsHD.TargetID != "108" {
		t.Fatalf("expected 体育频道HD -> 五星体育HD, got %+v", mSportsHD)
	}

	// 2. Test ApplyMapping on channel with 体育频道
	c1 := Channel{
		Name:    "体育频道",
		URL:     "igmp://233.18.204.6:5140",
		Group:   "待识别",
		Enabled: true,
	}
	if !store.ApplyMapping(&c1) {
		t.Fatalf("expected ApplyMapping to return true for 体育频道")
	}
	if c1.Name != "五星体育" || c1.ID != "8" || c1.Group != "本地" || c1.Logo != "/logos/五星体育.png" {
		t.Fatalf("ApplyMapping did not update channel properly: %+v", c1)
	}

	// 3. Test Service.Channels uses mapping
	s := NewService(store)
	s.LogosDir = filepath.Join(tmpDir, "logos")
	_ = s.EnsurePresetLogos()
	if err := store.PutChannel(Channel{
		ID:      "scan-1",
		Name:    "体育频道",
		URL:     "rtp://239.1.1.1:5140",
		Enabled: true,
	}); err != nil {
		t.Fatalf("PutChannel failed: %v", err)
	}
	channels, err := s.Channels()
	if err != nil {
		t.Fatalf("Channels() failed: %v", err)
	}
	if len(channels) == 0 || channels[0].Name != "五星体育" {
		t.Fatalf("expected channel name to be mapped to 五星体育, got %+v", channels)
	}

	// 4. Test Service.FindLocalLogo follows mapping
	logo := s.FindLocalLogo("体育频道")
	if logo != "/logos/五星体育.png" {
		t.Fatalf("expected FindLocalLogo(体育频道) = /logos/五星体育.png, got %q", logo)
	}

	// 5. Test user customization: change mapping for 体育频道 to custom target
	err = store.SaveMapping(ChannelMapping{
		Keyword:    "体育频道",
		TargetName: "广东体育",
		TargetID:   "99",
		Group:      "体育专区",
	})
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}

	cCustom := Channel{Name: "体育频道"}
	store.ApplyMapping(&cCustom)
	if cCustom.Name != "广东体育" || cCustom.ID != "99" || cCustom.Group != "体育专区" {
		t.Fatalf("expected user customized mapping to apply, got %+v", cCustom)
	}

	// 6. Test user deletes mapping: 体育频道 is no longer modified
	err = store.DeleteMapping("体育频道")
	if err != nil {
		t.Fatalf("DeleteMapping failed: %v", err)
	}
	cDeleted := Channel{Name: "体育频道"}
	if store.ApplyMapping(&cDeleted) {
		t.Fatalf("expected ApplyMapping to return false after mapping deleted")
	}
	if cDeleted.Name != "体育频道" {
		t.Fatalf("channel name should remain unchanged when mapping deleted")
	}

	// 7. Test ResetMappings: restores default mappings
	err = store.ResetMappings()
	if err != nil {
		t.Fatalf("ResetMappings failed: %v", err)
	}
	mReset, found := store.FindMapping("体育频道")
	if !found || mReset.TargetName != "五星体育" {
		t.Fatalf("expected ResetMappings to restore 体育频道 -> 五星体育, got %+v", mReset)
	}
}
