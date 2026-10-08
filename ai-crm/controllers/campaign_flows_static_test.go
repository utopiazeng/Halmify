package controllers

import (
	"os"
	"strings"
	"testing"
)

func TestCampaignPerformanceDoesNotUseFuzzyAttribution(t *testing.T) {
	source, err := os.ReadFile("campaign_flows.go")
	if err != nil {
		t.Fatalf("read campaign_flows.go: %v", err)
	}
	code := string(source)
	start := strings.Index(code, "func campaignRelatedLeads")
	end := strings.Index(code, "func mergeCRMObjects")
	if start < 0 || end <= start {
		t.Fatalf("campaign related helper functions not found")
	}
	helpers := code[start:end]
	for _, forbidden := range []string{`"campaign_name"`, `"utm_campaign"`, `"utm_source", "source_channel"`, "payloadJSONKeyNumberLikeClause"} {
		if strings.Contains(helpers, forbidden) {
			t.Fatalf("campaign performance must not fuzzy-match attribution by %s", forbidden)
		}
	}
	if !strings.Contains(helpers, "payloadJSONKeyNumberEqualsClause") {
		t.Fatalf("campaign performance should keep strict campaign_id matching")
	}
}
