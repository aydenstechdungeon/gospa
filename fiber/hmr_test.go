package fiber

import "testing"

func TestDetermineUpdateType_GospaIsTemplateSafe(t *testing.T) {
	mgr := NewHMRManager(HMRConfig{})
	updateType, reloadReason := mgr.determineUpdateType("routes/+page.gospa")
	if updateType != "template" {
		t.Fatalf("expected template update type, got %q", updateType)
	}
	if reloadReason != "template-safe" {
		t.Fatalf("expected template-safe reload reason, got %q", reloadReason)
	}
}
