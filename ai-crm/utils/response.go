package utils

import (
	"net/http"

	"github.com/beego/beego/v2/server/web/context"
)

// Response 统一响应结构
type Response struct {
	Code          int                    `json:"code"`
	Message       string                 `json:"message"`
	Data          interface{}            `json:"data"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	MessageKey    string                 `json:"message_key,omitempty"`
	MessageParams map[string]interface{} `json:"message_params,omitempty"`
}

// Success 成功响应
func Success(ctx *context.Context, data interface{}) {
	ctx.Output.SetStatus(http.StatusOK)
	if err := ctx.Output.JSON(Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}, false, false); err != nil {
		LogError("JSON响应写出失败: %v", err)
	}
}

// Error 错误响应
func Error(ctx *context.Context, code int, message string) {
	ErrorWithData(ctx, code, message, nil)
}

func ErrorWithData(ctx *context.Context, code int, message string, data interface{}) {
	key, params := ResolveErrorMessageKey(message, code)
	ErrorKeyWithLegacy(ctx, code, key, params, message, data)
}

func ErrorKey(ctx *context.Context, code int, key string, params map[string]interface{}) {
	ErrorKeyWithLegacy(ctx, code, key, params, "", nil)
}

func ErrorKeyWithData(ctx *context.Context, code int, key string, params map[string]interface{}, data interface{}) {
	ErrorKeyWithLegacy(ctx, code, key, params, "", data)
}

func ErrorKeyWithLegacy(ctx *context.Context, code int, key string, params map[string]interface{}, legacyMessage string, data interface{}) {
	if params == nil {
		params = map[string]interface{}{}
	}
	if key == "" {
		key = ErrorKeyForStatus(code)
	}
	errorCode := ErrorCodeFromKey(key)
	data, errorCode = mergeLegacyErrorCode(data, errorCode)
	if legacyMessage != "" {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if _, exists := dataMap["legacy_message"]; !exists {
				dataMap["legacy_message"] = legacyMessage
			}
			data = dataMap
		} else {
			data = map[string]interface{}{"legacy_message": legacyMessage}
		}
	}
	message := LocalizedErrorMessage(ctx, key, params, legacyMessage, code)
	ctx.Output.SetStatus(code)
	if err := ctx.Output.JSON(Response{
		Code:          code,
		Message:       message,
		Data:          data,
		ErrorCode:     errorCode,
		MessageKey:    key,
		MessageParams: params,
	}, false, false); err != nil {
		LogError("JSON错误响应写出失败: %v", err)
	}
}

// BadRequest 错误请求响应
func BadRequest(ctx *context.Context, message string) {
	Error(ctx, http.StatusBadRequest, message)
}

// Unauthorized 未授权响应
func Unauthorized(ctx *context.Context, message string) {
	Error(ctx, http.StatusUnauthorized, message)
}

// Forbidden 禁止访问响应
func Forbidden(ctx *context.Context, message string) {
	Error(ctx, http.StatusForbidden, message)
}

// NotFound 未找到响应
func NotFound(ctx *context.Context, message string) {
	Error(ctx, http.StatusNotFound, message)
}

// InternalServerError 服务器内部错误响应
func InternalServerError(ctx *context.Context, message string) {
	Error(ctx, http.StatusInternalServerError, message)
}

func BadRequestKey(ctx *context.Context, key string, params map[string]interface{}) {
	ErrorKey(ctx, http.StatusBadRequest, key, params)
}

func UnauthorizedKey(ctx *context.Context, key string, params map[string]interface{}) {
	ErrorKey(ctx, http.StatusUnauthorized, key, params)
}

func ForbiddenKey(ctx *context.Context, key string, params map[string]interface{}) {
	ErrorKey(ctx, http.StatusForbidden, key, params)
}

func NotFoundKey(ctx *context.Context, key string, params map[string]interface{}) {
	ErrorKey(ctx, http.StatusNotFound, key, params)
}
