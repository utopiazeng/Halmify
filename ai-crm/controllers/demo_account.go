package controllers

import (
	"strings"

	"ai-crm/models"
)

const immutableDemoUsername = "testaccount"

func isImmutableDemoAccount(user models.User) bool {
	return strings.EqualFold(strings.TrimSpace(user.Username), immutableDemoUsername)
}
