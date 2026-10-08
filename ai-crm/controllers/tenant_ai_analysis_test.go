package controllers

import "testing"

func TestTenantAnalysisCostUsesDeepSeekTokenBuckets(t *testing.T) {
	pricing := tenantAnalysisPricing{
		InputHitRate:  aiPricingRate{TenantPriceMicrosPerMillion: 100},
		InputMissRate: aiPricingRate{TenantPriceMicrosPerMillion: 1000},
		OutputRate:    aiPricingRate{TenantPriceMicrosPerMillion: 2000},
		Currency:      "USD",
	}
	usage := tenantAnalysisUsage{
		CachedInputTokens:   1000,
		UncachedInputTokens: 2000,
		OutputTokens:        3000,
		PromptTokens:        3000,
		CompletionTokens:    3000,
		TotalTokens:         6000,
	}
	got := calculateTenantAnalysisCostMicros(usage, pricing)
	want := int64(9)
	if got != want {
		t.Fatalf("cost = %d, want %d", got, want)
	}
}

func TestTenantAnalysisCostDefaultsInputToCacheMiss(t *testing.T) {
	pricing := tenantAnalysisPricing{
		InputHitRate:  aiPricingRate{TenantPriceMicrosPerMillion: 1},
		InputMissRate: aiPricingRate{TenantPriceMicrosPerMillion: 1000},
		OutputRate:    aiPricingRate{TenantPriceMicrosPerMillion: 2000},
		Currency:      "USD",
	}
	usage := tenantAnalysisUsage{
		PromptTokens:     1500,
		CompletionTokens: 500,
		TotalTokens:      2000,
	}
	got := calculateTenantAnalysisCostMicros(usage, pricing)
	want := int64(3)
	if got != want {
		t.Fatalf("cost = %d, want %d", got, want)
	}
}

func TestTenantAnalysisPriceSnapshotKeepsTenantRates(t *testing.T) {
	pricing := tenantAnalysisPricing{
		Snapshot:      aiPricingSnapshot{Provider: deepSeekProvider, Currency: "USD", SourceURL: "https://example.test", SourceStatus: "synced", SyncedAt: "2026-05-24T00:00:00Z", MarginRate: aiPricingMarginRate},
		Model:         aiPricingModel{Model: "deepseek-v4-flash"},
		Provider:      deepSeekProvider,
		ModelName:     "deepseek-v4-flash",
		Currency:      "USD",
		InputHitRate:  aiPricingRate{TenantPriceMicrosPerMillion: 11, OfficialPriceAmount: "0.000010", TenantPriceAmount: "0.000011"},
		InputMissRate: aiPricingRate{TenantPriceMicrosPerMillion: 22, OfficialPriceAmount: "0.000020", TenantPriceAmount: "0.000022"},
		OutputRate:    aiPricingRate{TenantPriceMicrosPerMillion: 33, OfficialPriceAmount: "0.000030", TenantPriceAmount: "0.000033"},
	}
	snapshot := tenantAnalysisPriceSnapshot(pricing)
	if snapshot["cached_input_price_micros"] != int64(11) || snapshot["uncached_input_price_micros"] != int64(22) || snapshot["output_price_micros"] != int64(33) {
		t.Fatalf("snapshot tenant rates not preserved: %#v", snapshot)
	}
	if snapshot["source_url"] != "https://example.test" || snapshot["synced_at"] != "2026-05-24T00:00:00Z" {
		t.Fatalf("snapshot source fields not preserved: %#v", snapshot)
	}
}
