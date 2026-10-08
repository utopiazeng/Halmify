package controllers

import (
	"ai-crm/utils"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"net/http"
)

type CRMController struct {
	web.Controller
}

func requireTenantUser(ctx *context.Context) (authContext, bool) {
	auth := currentAuth(ctx)
	if auth.TenantID == 0 {
		utils.Error(ctx, http.StatusForbidden, "缺少租户上下文")
		return auth, false
	}
	return auth, true
}
