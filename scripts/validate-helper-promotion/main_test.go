package main

import "testing"

func TestValidatePromotionMissingMarkers(t *testing.T) {
	failures := validatePromotion(
		"func Depends(keys ...string) {}",
		"",
		"",
	)
	if len(failures) == 0 {
		t.Fatal("expected failures when docs and migration markers are missing")
	}
}

func TestValidatePromotionPassesWithMarkers(t *testing.T) {
	code := "func Depends(keys ...string) {}\nfunc Untrack(fn func() error) error { return nil }"
	api := "`kit.Depends`\n`kit.Untrack`\nrender_load_helpers_test.go"
	migration := "`kit.Depends`\n`kit.Untrack`"

	failures := validatePromotion(code, api, migration)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}
