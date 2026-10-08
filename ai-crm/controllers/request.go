package controllers

import (
	"ai-crm/utils"
	"encoding/json"

	"github.com/beego/beego/v2/server/web/context"
)

func bindJSON(ctx *context.Context, target interface{}) bool {
	if err := json.Unmarshal(ctx.Input.RequestBody, target); err != nil {
		utils.BadRequestKey(ctx, "common.invalidRequest", nil)
		return false
	}

	return true
}
