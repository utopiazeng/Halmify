package controllers

import (
	"testing"
	"time"
)

func TestBuildEntitlementQuoteFromStateAllowsRenewalWithoutAdditionalQuantity(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	expireAt := now.AddDate(0, 6, 0)

	quote, err := buildEntitlementQuoteFromState("storage", 0, 1, now, 3, &expireAt, 900, nil)
	if err != nil {
		t.Fatalf("expected renewal quote without additional quantity, got error: %v", err)
	}
	if quote.AdditionalQuantity != 0 {
		t.Fatalf("additional quantity = %d, want 0", quote.AdditionalQuantity)
	}
	if quote.TargetQuantity != 3 {
		t.Fatalf("target quantity = %d, want current quantity 3", quote.TargetQuantity)
	}
	if quote.DueCents <= 0 {
		t.Fatalf("due cents = %d, want positive renewal amount", quote.DueCents)
	}
}

func TestBuildEntitlementQuoteFromStateRejectsZeroRenewalWithoutCurrentEntitlement(t *testing.T) {
	_, err := buildEntitlementQuoteFromState("seats", 0, 1, time.Now(), 0, nil, 0, nil)
	if err == nil {
		t.Fatal("expected zero renewal without current entitlement to fail")
	}
}

func TestBuildEntitlementQuoteFromStateRejectsNegativeAdditionalQuantity(t *testing.T) {
	_, err := buildEntitlementQuoteFromState("storage", -1, 1, time.Now(), 3, nil, 0, nil)
	if err == nil {
		t.Fatal("expected negative additional quantity to fail")
	}
}

func TestBuildEntitlementQuoteFromStateUsesOneDollarPerSeatMonth(t *testing.T) {
	quote, err := buildEntitlementQuoteFromState("seats", 0, 1, time.Now(), 2, nil, 0, nil)
	if err != nil {
		t.Fatalf("expected seat renewal quote, got error: %v", err)
	}
	if quote.UnitPriceCents != 100 {
		t.Fatalf("seat monthly unit price cents = %d, want 100", quote.UnitPriceCents)
	}
	if quote.TargetGrossCents != 2400 {
		t.Fatalf("target gross cents = %d, want 2400", quote.TargetGrossCents)
	}
}
