package controllers

import (
	"testing"

	"ai-crm/models"
)

func TestIsImmutableDemoAccount(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{name: "demo account", username: "testaccount", want: true},
		{name: "case and whitespace", username: " TestAccount ", want: true},
		{name: "ordinary account", username: "sales-admin", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isImmutableDemoAccount(models.User{Username: test.username}); got != test.want {
				t.Fatalf("isImmutableDemoAccount(%q) = %t, want %t", test.username, got, test.want)
			}
		})
	}
}
