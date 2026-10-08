package controllers

import "testing"

func TestNormalizeDeepSeekPricingSnapshotUsesFlashOnlyAndCurrentMargin(t *testing.T) {
	t.Setenv("AI_MODEL", defaultDeepSeekModel)
	snapshot := aiPricingSnapshot{
		Provider:   deepSeekProvider,
		Currency:   defaultAICurrency,
		MarginRate: 0.10,
		Aliases: map[string]string{
			"deepseek-reasoner": "deepseek-v4-pro",
		},
		Models: []aiPricingModel{
			{
				Model:       defaultDeepSeekModel,
				DisplayName: deepSeekModelDisplayName(defaultDeepSeekModel),
				Rates: []aiPricingRate{
					{Metric: "input_cache_hit", OfficialPriceMicrosPerMillion: 2800, TenantPriceMicrosPerMillion: 3080},
					{Metric: "input_cache_miss", OfficialPriceMicrosPerMillion: 140000, TenantPriceMicrosPerMillion: 154000},
					{Metric: "output", OfficialPriceMicrosPerMillion: 280000, TenantPriceMicrosPerMillion: 308000},
				},
			},
			{
				Model:       "deepseek-v4-pro",
				DisplayName: "DeepSeek V4 Pro",
				Rates: []aiPricingRate{
					{Metric: "output", OfficialPriceMicrosPerMillion: 870000, TenantPriceMicrosPerMillion: 957000},
				},
			},
		},
	}

	normalized := normalizeDeepSeekPricingSnapshot(snapshot)
	if normalized.MarginRate != aiPricingMarginRate {
		t.Fatalf("margin = %v, want %v", normalized.MarginRate, aiPricingMarginRate)
	}
	if len(normalized.Models) != 1 {
		t.Fatalf("models = %d, want 1: %#v", len(normalized.Models), normalized.Models)
	}
	model := normalized.Models[0]
	if model.Model != defaultDeepSeekModel {
		t.Fatalf("model = %q, want %q", model.Model, defaultDeepSeekModel)
	}
	if _, ok := normalized.Aliases["deepseek-v4-pro"]; ok {
		t.Fatalf("pro alias should not be returned: %#v", normalized.Aliases)
	}
	if normalized.Aliases["deepseek-reasoner"] != defaultDeepSeekModel {
		t.Fatalf("aliases not normalized: %#v", normalized.Aliases)
	}
	rates := map[string]aiPricingRate{}
	for _, rate := range model.Rates {
		rates[rate.Metric] = rate
	}
	if rates["input_cache_hit"].TenantPriceMicrosPerMillion != 3220 {
		t.Fatalf("cache-hit price = %d, want 3220", rates["input_cache_hit"].TenantPriceMicrosPerMillion)
	}
	if rates["input_cache_miss"].TenantPriceMicrosPerMillion != 161000 {
		t.Fatalf("cache-miss price = %d, want 161000", rates["input_cache_miss"].TenantPriceMicrosPerMillion)
	}
	if rates["output"].TenantPriceMicrosPerMillion != 322000 {
		t.Fatalf("output price = %d, want 322000", rates["output"].TenantPriceMicrosPerMillion)
	}
}

func TestPricingModelUsesConfiguredModel(t *testing.T) {
	t.Setenv("AI_MODEL", "deepseek-v4-pro")
	snapshot := aiPricingSnapshot{
		Provider: deepSeekProvider,
		Currency: defaultAICurrency,
		Models: []aiPricingModel{
			{
				Model: "deepseek-v4-pro",
				Rates: []aiPricingRate{
					{Metric: "output", OfficialPriceMicrosPerMillion: 870000, TenantPriceMicrosPerMillion: 1000000},
				},
			},
		},
	}

	model := pricingModel(snapshot, "deepseek-v4-pro")
	if model.Model != "deepseek-v4-pro" {
		t.Fatalf("model = %q, want deepseek-v4-pro", model.Model)
	}
	output := pricingRate(model, "output")
	if output.TenantPriceMicrosPerMillion != 1000500 {
		t.Fatalf("output price = %d, want 1000500", output.TenantPriceMicrosPerMillion)
	}
}

func TestNewAIPricingRateUsesFifteenPercentMargin(t *testing.T) {
	rate := newAIPricingRate("output", "输出 token", amountFloatToMicros(0.28))
	if rate.TenantPriceMicrosPerMillion != 322000 {
		t.Fatalf("tenant price = %d, want 322000", rate.TenantPriceMicrosPerMillion)
	}
	if rate.TenantPriceAmount != "0.322000" {
		t.Fatalf("tenant amount = %q, want 0.322000", rate.TenantPriceAmount)
	}
}

func TestParseDeepSeekPricingHTMLUsesConfiguredModelColumn(t *testing.T) {
	html := `<table>
		<tr><td colspan="2">MODEL</td><td>deepseek-v4-flash<sup>(1)</sup></td><td>deepseek-v4-pro</td></tr>
		<tr><td rowspan="3">PRICING</td><td>1M INPUT TOKENS (CACHE HIT)</td><td>$0.0028</td><td>$0.003625</td></tr>
		<tr><td>1M INPUT TOKENS (CACHE MISS)</td><td>$0.14</td><td>$0.435</td></tr>
		<tr><td>1M OUTPUT TOKENS</td><td>$0.28</td><td>$0.87</td></tr>
	</table>`
	prices, err := parseDeepSeekPricingHTML(html, "deepseek-v4-pro")
	if err != nil {
		t.Fatalf("parse pricing: %v", err)
	}
	if prices["deepseek-v4-pro"]["input_cache_hit"] != 3625 {
		t.Fatalf("cache-hit = %d, want 3625", prices["deepseek-v4-pro"]["input_cache_hit"])
	}
	if prices["deepseek-v4-pro"]["input_cache_miss"] != 435000 {
		t.Fatalf("cache-miss = %d, want 435000", prices["deepseek-v4-pro"]["input_cache_miss"])
	}
	if prices["deepseek-v4-pro"]["output"] != 870000 {
		t.Fatalf("output = %d, want 870000", prices["deepseek-v4-pro"]["output"])
	}
}
