package controllers

import (
	"ai-crm/services"
	"ai-crm/utils"
	"errors"
	"net/http"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

type IndustryPackageController struct {
	web.Controller
}

type industryPackageUpdateRequest struct {
	PackageCode string                 `json:"package_code"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
}

func (c *IndustryPackageController) ListPackages() {
	if _, ok := requirePlatformOrTenantUser(c.Ctx); !ok {
		return
	}
	items, err := services.ListIndustryPackages()
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取行业包列表失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *IndustryPackageController) GetTenantIndustryPackage() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	assignment, exists, err := services.TenantIndustryPackage(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取租户行业包失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"package": assignment, "exists": exists})
}

func (c *IndustryPackageController) UpdatePlatformTenantIndustryPackage() {
	if _, ok := requirePlatformAdmin(c.Ctx); !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req industryPackageUpdateRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	assignment, err := services.SetTenantIndustryPackage(tenantID, services.IndustryPackageUpsert{
		PackageCode: req.PackageCode,
		Enabled:     req.Enabled,
		Config:      req.Config,
	})
	if err != nil {
		writeIndustryPackageError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"package": assignment})
}

func (c *IndustryPackageController) ExtensionsManifest() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	manifest, err := services.TenantIndustryManifest(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取行业扩展清单失败")
		return
	}
	utils.Success(c.Ctx, manifest)
}

func (c *IndustryPackageController) ManufacturingStatus() {
	if !c.requireManufacturingEnabled() {
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"package_code": services.IndustryPackageManufacturing,
		"enabled":      true,
		"status":       "placeholder",
	})
}

func (c *IndustryPackageController) ManufacturingGate() {
	if !c.requireManufacturingEnabled() {
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"package_code": services.IndustryPackageManufacturing,
		"enabled":      true,
		"status":       "placeholder",
		"path":         c.Ctx.Input.URL(),
	})
}

func (c *IndustryPackageController) requireManufacturingEnabled() bool {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return false
	}
	if !services.IsIndustryPackageEnabled(auth.TenantID, services.IndustryPackageManufacturing) {
		writeIndustryPackageError(c.Ctx, errors.New(services.IndustryPackageErrorDisabled))
		return false
	}
	return true
}

func requirePlatformOrTenantUser(ctx *context.Context) (authContext, bool) {
	auth := currentAuth(ctx)
	if auth.IsPlatformAdmin() || auth.TenantID > 0 {
		return auth, true
	}
	utils.Forbidden(ctx, "缺少租户或平台上下文")
	return auth, false
}

func writeIndustryPackageError(ctx *context.Context, err error) {
	message := err.Error()
	status := http.StatusBadRequest
	switch message {
	case services.IndustryPackageErrorDisabled:
		status = http.StatusForbidden
	case services.IndustryPackageErrorNotFound:
		status = http.StatusNotFound
	case services.IndustryPackageErrorConflict:
		status = http.StatusConflict
	default:
		message = "行业包操作失败"
	}
	utils.ErrorWithData(ctx, status, message, map[string]interface{}{"error_code": message})
}
