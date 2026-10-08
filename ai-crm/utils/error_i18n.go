package utils

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

var supportedErrorLocales = map[string]string{
	"zh": "zh-CN", "zh-cn": "zh-CN", "zh-hans": "zh-CN",
	"zh-tw": "zh-TW", "zh-hk": "zh-TW", "zh-hant": "zh-TW",
	"en": "en-US", "en-us": "en-US", "en-gb": "en-US",
	"es": "es-ES", "es-es": "es-ES",
	"fr": "fr-FR", "fr-fr": "fr-FR",
	"de": "de-DE", "de-de": "de-DE",
	"ja": "ja-JP", "ja-jp": "ja-JP",
	"ko": "ko-KR", "ko-kr": "ko-KR",
	"pt": "pt-PT", "pt-pt": "pt-PT", "pt-br": "pt-PT",
	"ar": "ar-SA", "ar-sa": "ar-SA",
	"ru": "ru-RU", "ru-ru": "ru-RU",
	"it": "it-IT", "it-it": "it-IT",
}

var zhErrorTextToKey = map[string]string{
	"请求参数错误":                "common.invalidRequest",
	"请求参数无效":                "common.invalidRequest",
	"无效的请求参数":               "common.invalidRequest",
	"缺少认证令牌":                "auth.tokenMissing",
	"无效的令牌":                 "auth.tokenInvalid",
	"登录会话已失效，请重新登录":         "auth.sessionExpired",
	"登录会话不存在，请重新登录":         "auth.sessionNotFound",
	"用户不存在或会话无效":            "auth.sessionInvalid",
	"首次登录需先修改密码":            "auth.passwordChangeRequired",
	"用户名或密码错误":              "auth.invalidCredentials",
	"用户账号已被禁用":              "auth.userDisabled",
	"验证码服务不可用":              "auth.captchaUnavailable",
	"请先完成滑动验证码":             "auth.captchaRequired",
	"滑动验证码无效，请重试":           "auth.captchaFailed",
	"滑动验证码已过期，请重试":          "auth.captchaExpired",
	"滑动验证码与当前设备不匹配":         "auth.captchaFailed",
	"滑动验证码已被使用，请刷新重试":       "auth.captchaExpired",
	"滑动验证码已失效，请刷新重试":        "auth.captchaExpired",
	"滑动验证未通过，请重试":           "auth.captchaFailed",
	"生成验证码令牌失败":             "auth.captchaTokenFailed",
	"登录防护繁忙，请稍后重试":          "auth.loginProtectionBusy",
	"登录尝试过于频繁，请稍后再试":        "auth.loginRateLimited",
	"系统已初始化，请由超管或企业管理员创建账号": "auth.systemInitialized",
	"用户名已存在":                "auth.usernameExists",
	"密码加密失败":                "auth.passwordHashFailed",
	"演示账号密码不可修改":            "auth.demoAccountPasswordLocked",
	"用户创建失败":                "auth.userCreateFailed",
	"缺少租户上下文":               "common.tenantContextMissing",
	"无权限访问":                 "common.permissionDenied",
	"无权限访问平台域接口":            "common.platformPermissionDenied",
	"仅平台超管可访问平台域接口":         "common.platformAdminOnly",
	"平台超管不可直接访问租户业务域":       "common.platformAdminTenantDenied",
	"当前租户未开通该功能模块":          "common.moduleNotEnabled",
	"租户不允许管理菜单":             "common.tenantMenuManageDenied",
	"TENANT_DISABLED":       "auth.tenantDisabled",
	"TENANT_EXPIRED":        "auth.tenantExpired",
	"业务对象不存在":               "common.objectNotFound",
	"无效的业务对象":               "common.invalidObject",
	"无效的业务域":                "common.invalidDomain",
	"数据不存在":                 "common.notFound",
	"记录不存在":                 "common.notFound",
	"文件不存在":                 "file.notFound",
	"暂无可用文件服务节点":            "file.noAvailableNode",
	"缺少file_id":             "file.missingFileID",
	"删除文件失败":                "file.deleteFailed",
	"导出失败":                  "common.exportFailed",
	"保存失败":                  "common.saveFailed",
	"保存视图失败":                "common.saveFailed",
	"加载失败":                  "common.loadFailed",
	"操作失败":                  "common.operationFailed",
	"创建失败":                  "common.createFailed",
	"更新失败":                  "common.updateFailed",
	"删除失败":                  "common.deleteFailed",
}

var errorMessages = map[string]map[string]string{
	"zh-CN": {
		"common.invalidRequest":              "请求参数错误",
		"common.notFound":                    "数据不存在",
		"common.objectNotFound":              "业务对象不存在",
		"common.invalidObject":               "无效的业务对象",
		"common.invalidDomain":               "无效的业务域",
		"common.permissionDenied":            "无权限访问",
		"common.platformPermissionDenied":    "无权限访问平台域接口",
		"common.platformAdminOnly":           "仅平台超管可访问平台域接口",
		"common.platformAdminTenantDenied":   "平台超管不可直接访问租户业务域",
		"common.tenantContextMissing":        "缺少租户上下文",
		"common.moduleNotEnabled":            "当前租户未开通该功能模块",
		"common.tenantMenuManageDenied":      "租户不允许管理菜单",
		"common.operationFailed":             "操作失败",
		"common.loadFailed":                  "加载失败",
		"common.saveFailed":                  "保存失败",
		"common.createFailed":                "创建失败",
		"common.updateFailed":                "更新失败",
		"common.deleteFailed":                "删除失败",
		"common.exportFailed":                "导出失败",
		"common.badRequest":                  "请求参数错误",
		"common.unauthorized":                "请先登录",
		"common.forbidden":                   "无权限访问",
		"common.internal":                    "服务器内部错误",
		"security.riskBlocked":               "访问行为触发安全风控，已阻断",
		"auth.tokenMissing":                  "缺少认证令牌",
		"auth.tokenInvalid":                  "无效的令牌",
		"auth.sessionExpired":                "登录会话已失效，请重新登录",
		"auth.sessionNotFound":               "登录会话不存在，请重新登录",
		"auth.sessionInvalid":                "用户不存在或会话无效",
		"auth.passwordChangeRequired":        "首次登录需先修改密码",
		"auth.invalidCredentials":            "用户名或密码错误",
		"auth.userDisabled":                  "用户账号已被禁用",
		"auth.tenantDisabled":                "租户已停用",
		"auth.tenantExpired":                 "租户已过期",
		"auth.captchaUnavailable":            "验证码服务不可用",
		"auth.captchaRequired":               "请先完成滑动验证码",
		"auth.captchaExpired":                "滑动验证码已失效，请刷新重试",
		"auth.captchaFailed":                 "滑动验证未通过，请重试",
		"auth.captchaTokenFailed":            "生成验证码令牌失败",
		"auth.loginProtectionBusy":           "登录防护繁忙，请稍后重试",
		"auth.loginRateLimited":              "登录尝试过于频繁，请稍后再试",
		"auth.systemInitialized":             "系统已初始化，请由超管或企业管理员创建账号",
		"auth.usernameExists":                "用户名已存在",
		"auth.passwordHashFailed":            "密码加密失败",
		"auth.demoAccountPasswordLocked":     "演示账号密码不可修改",
		"auth.userCreateFailed":              "用户创建失败",
		"file.notFound":                      "文件不存在",
		"file.noAvailableNode":               "暂无可用文件服务节点",
		"file.missingFileID":                 "缺少 file_id",
		"file.deleteFailed":                  "删除文件失败",
		"file.serviceStatusFailed":           "文件服务返回 {status}",
		"iotFirmware.error.missingFile":      "缺少固件升级包",
		"iotFirmware.error.invalidSize":      "固件升级包大小无效",
		"iotFirmware.error.invalidPackage":   "固件升级包无效：{reason}",
		"iotFirmware.error.shaMismatch":      "固件 SHA256 不匹配",
		"iotFirmware.error.tenantRequired":   "请选择租户",
		"iotFirmware.error.snsRequired":      "请选择设备SN",
		"iotFirmware.error.invalidTarget":    "升级目标无效",
		"iotFirmware.error.noMatchingDevice": "没有匹配的设备，请检查设备SN和固件型号是否一致",
	},
	"zh-TW": {
		"common.invalidRequest":              "請求參數錯誤",
		"common.notFound":                    "資料不存在",
		"common.objectNotFound":              "業務物件不存在",
		"common.invalidObject":               "無效的業務物件",
		"common.invalidDomain":               "無效的業務域",
		"common.permissionDenied":            "無權限訪問",
		"common.platformPermissionDenied":    "無權限訪問平台域介面",
		"common.platformAdminOnly":           "僅平台超管可訪問平台域介面",
		"common.platformAdminTenantDenied":   "平台超管不可直接訪問租戶業務域",
		"common.tenantContextMissing":        "缺少租戶上下文",
		"common.moduleNotEnabled":            "目前租戶未開通該功能模組",
		"common.tenantMenuManageDenied":      "租戶不允許管理選單",
		"common.operationFailed":             "操作失敗",
		"common.loadFailed":                  "載入失敗",
		"common.saveFailed":                  "儲存失敗",
		"common.createFailed":                "建立失敗",
		"common.updateFailed":                "更新失敗",
		"common.deleteFailed":                "刪除失敗",
		"common.exportFailed":                "匯出失敗",
		"common.badRequest":                  "請求參數錯誤",
		"common.unauthorized":                "請先登入",
		"common.forbidden":                   "無權限訪問",
		"common.internal":                    "伺服器內部錯誤",
		"security.riskBlocked":               "訪問行為觸發安全風控，已被阻斷",
		"auth.tokenMissing":                  "缺少認證令牌",
		"auth.tokenInvalid":                  "無效的令牌",
		"auth.sessionExpired":                "登入工作階段已失效，請重新登入",
		"auth.sessionNotFound":               "登入工作階段不存在，請重新登入",
		"auth.sessionInvalid":                "使用者不存在或工作階段無效",
		"auth.passwordChangeRequired":        "首次登入需先修改密碼",
		"auth.invalidCredentials":            "使用者名稱或密碼錯誤",
		"auth.userDisabled":                  "使用者帳號已停用",
		"auth.tenantDisabled":                "租戶已停用",
		"auth.tenantExpired":                 "租戶已過期",
		"auth.captchaUnavailable":            "驗證碼服務不可用",
		"auth.captchaRequired":               "請先完成滑動驗證碼",
		"auth.captchaExpired":                "滑動驗證碼已失效，請重新整理後重試",
		"auth.captchaFailed":                 "滑動驗證未通過，請重試",
		"auth.captchaTokenFailed":            "產生驗證碼令牌失敗",
		"auth.loginProtectionBusy":           "登入防護繁忙，請稍後重試",
		"auth.loginRateLimited":              "登入嘗試過於頻繁，請稍後再試",
		"auth.systemInitialized":             "系統已初始化，請由超管或企業管理員建立帳號",
		"auth.usernameExists":                "使用者名稱已存在",
		"auth.passwordHashFailed":            "密碼加密失敗",
		"auth.demoAccountPasswordLocked":     "示範帳號密碼不可修改",
		"auth.userCreateFailed":              "使用者建立失敗",
		"file.notFound":                      "檔案不存在",
		"file.noAvailableNode":               "暫無可用檔案服務節點",
		"file.missingFileID":                 "缺少 file_id",
		"file.deleteFailed":                  "刪除檔案失敗",
		"file.serviceStatusFailed":           "檔案服務返回 {status}",
		"iotFirmware.error.missingFile":      "缺少韌體升級包",
		"iotFirmware.error.invalidSize":      "韌體升級包大小無效",
		"iotFirmware.error.invalidPackage":   "韌體升級包無效：{reason}",
		"iotFirmware.error.shaMismatch":      "韌體 SHA256 不匹配",
		"iotFirmware.error.tenantRequired":   "請選擇租戶",
		"iotFirmware.error.snsRequired":      "請選擇裝置SN",
		"iotFirmware.error.invalidTarget":    "升級目標無效",
		"iotFirmware.error.noMatchingDevice": "沒有匹配的裝置，請檢查裝置SN和韌體型號是否一致",
	},
	"en-US": {
		"common.invalidRequest":              "Invalid request parameters",
		"common.notFound":                    "Data not found",
		"common.objectNotFound":              "Business object not found",
		"common.invalidObject":               "Invalid business object",
		"common.invalidDomain":               "Invalid business domain",
		"common.permissionDenied":            "You do not have permission to access this resource",
		"common.platformPermissionDenied":    "You do not have permission to access platform APIs",
		"common.platformAdminOnly":           "Only platform super admins can access platform APIs",
		"common.platformAdminTenantDenied":   "Platform super admins cannot directly access tenant business APIs",
		"common.tenantContextMissing":        "Tenant context is missing",
		"common.moduleNotEnabled":            "This tenant has not enabled the module",
		"common.tenantMenuManageDenied":      "Tenants cannot manage menus",
		"common.operationFailed":             "Operation failed",
		"common.loadFailed":                  "Failed to load",
		"common.saveFailed":                  "Failed to save",
		"common.createFailed":                "Failed to create",
		"common.updateFailed":                "Failed to update",
		"common.deleteFailed":                "Failed to delete",
		"common.exportFailed":                "Export failed",
		"common.badRequest":                  "Invalid request parameters",
		"common.unauthorized":                "Please sign in first",
		"common.forbidden":                   "You do not have permission to access this resource",
		"common.internal":                    "Internal server error",
		"security.riskBlocked":               "Access was blocked by security risk controls",
		"auth.tokenMissing":                  "Authentication token is missing",
		"auth.tokenInvalid":                  "Invalid token",
		"auth.sessionExpired":                "Your session has expired. Please sign in again",
		"auth.sessionNotFound":               "Login session was not found. Please sign in again",
		"auth.sessionInvalid":                "User does not exist or the session is invalid",
		"auth.passwordChangeRequired":        "You must change your password before continuing",
		"auth.invalidCredentials":            "Incorrect username or password",
		"auth.userDisabled":                  "This user account has been disabled",
		"auth.tenantDisabled":                "This tenant has been disabled",
		"auth.tenantExpired":                 "This tenant has expired",
		"auth.captchaUnavailable":            "Captcha service is unavailable",
		"auth.captchaRequired":               "Please complete slider verification first",
		"auth.captchaExpired":                "The slider verification has expired. Please refresh and try again",
		"auth.captchaFailed":                 "Slider verification failed. Please try again",
		"auth.captchaTokenFailed":            "Failed to generate captcha token",
		"auth.loginProtectionBusy":           "Login protection is busy. Please try again later",
		"auth.loginRateLimited":              "Too many login attempts. Please try again later",
		"auth.systemInitialized":             "The system has already been initialized. Ask a super admin or tenant admin to create accounts",
		"auth.usernameExists":                "Username already exists",
		"auth.passwordHashFailed":            "Failed to encrypt password",
		"auth.demoAccountPasswordLocked":     "The demo account password cannot be changed",
		"auth.userCreateFailed":              "Failed to create user",
		"file.notFound":                      "File not found",
		"file.noAvailableNode":               "No file service node is available",
		"file.missingFileID":                 "file_id is missing",
		"file.deleteFailed":                  "Failed to delete file",
		"file.serviceStatusFailed":           "File service returned {status}",
		"iotFirmware.error.missingFile":      "Firmware upgrade package is missing",
		"iotFirmware.error.invalidSize":      "Firmware upgrade package size is invalid",
		"iotFirmware.error.invalidPackage":   "Invalid firmware upgrade package: {reason}",
		"iotFirmware.error.shaMismatch":      "Firmware SHA256 does not match",
		"iotFirmware.error.tenantRequired":   "Select a tenant",
		"iotFirmware.error.snsRequired":      "Select at least one device SN",
		"iotFirmware.error.invalidTarget":    "Invalid upgrade target",
		"iotFirmware.error.noMatchingDevice": "No matching device found. Check that the device SN and firmware model match.",
	},
}

func init() {
	translations := map[string]map[string]string{
		"es-ES": {
			"auth.invalidCredentials":        "Usuario o contraseña incorrectos",
			"auth.captchaExpired":            "La verificación deslizante ha caducado. Actualiza e inténtalo de nuevo",
			"auth.captchaFailed":             "La verificación deslizante falló. Inténtalo de nuevo",
			"auth.demoAccountPasswordLocked": "La contraseña de la cuenta de demostración no se puede cambiar",
			"security.riskBlocked":           "El acceso fue bloqueado por controles de seguridad",
		},
		"fr-FR": {
			"auth.invalidCredentials":        "Nom d’utilisateur ou mot de passe incorrect",
			"auth.captchaExpired":            "La vérification par curseur a expiré. Actualisez et réessayez",
			"auth.captchaFailed":             "La vérification par curseur a échoué. Réessayez",
			"auth.demoAccountPasswordLocked": "Le mot de passe du compte de démonstration ne peut pas être modifié",
			"security.riskBlocked":           "L’accès a été bloqué par les contrôles de sécurité",
		},
		"de-DE": {
			"auth.invalidCredentials":        "Benutzername oder Passwort ist falsch",
			"auth.captchaExpired":            "Die Schieberegler-Prüfung ist abgelaufen. Bitte aktualisieren und erneut versuchen",
			"auth.captchaFailed":             "Schieberegler-Prüfung fehlgeschlagen. Bitte erneut versuchen",
			"auth.demoAccountPasswordLocked": "Das Passwort des Demokontos kann nicht geändert werden",
			"security.riskBlocked":           "Der Zugriff wurde durch Sicherheitskontrollen blockiert",
		},
		"ja-JP": {
			"auth.invalidCredentials":        "ユーザー名またはパスワードが正しくありません",
			"auth.captchaExpired":            "スライダー認証の有効期限が切れました。更新して再試行してください",
			"auth.captchaFailed":             "スライダー認証に失敗しました。再試行してください",
			"auth.demoAccountPasswordLocked": "デモアカウントのパスワードは変更できません",
			"security.riskBlocked":           "セキュリティ制御によりアクセスがブロックされました",
		},
		"ko-KR": {
			"auth.invalidCredentials":        "사용자 이름 또는 비밀번호가 올바르지 않습니다",
			"auth.captchaExpired":            "슬라이더 인증이 만료되었습니다. 새로고침 후 다시 시도하세요",
			"auth.captchaFailed":             "슬라이더 인증에 실패했습니다. 다시 시도하세요",
			"auth.demoAccountPasswordLocked": "데모 계정의 비밀번호는 변경할 수 없습니다",
			"security.riskBlocked":           "보안 위험 제어로 접근이 차단되었습니다",
		},
		"pt-PT": {
			"auth.invalidCredentials":        "Nome de utilizador ou palavra-passe incorretos",
			"auth.captchaExpired":            "A verificação deslizante expirou. Atualize e tente novamente",
			"auth.captchaFailed":             "A verificação deslizante falhou. Tente novamente",
			"auth.demoAccountPasswordLocked": "A palavra-passe da conta de demonstração não pode ser alterada",
			"security.riskBlocked":           "O acesso foi bloqueado pelos controlos de segurança",
		},
		"ar-SA": {
			"auth.invalidCredentials":        "اسم المستخدم أو كلمة المرور غير صحيحة",
			"auth.captchaExpired":            "انتهت صلاحية التحقق بالسحب. حدّث الصفحة وحاول مرة أخرى",
			"auth.captchaFailed":             "فشل التحقق بالسحب. حاول مرة أخرى",
			"auth.demoAccountPasswordLocked": "لا يمكن تغيير كلمة مرور الحساب التجريبي",
			"security.riskBlocked":           "تم حظر الوصول بواسطة ضوابط الأمان",
		},
		"ru-RU": {
			"auth.invalidCredentials":        "Неверное имя пользователя или пароль",
			"auth.captchaExpired":            "Проверка слайдером истекла. Обновите страницу и повторите попытку",
			"auth.captchaFailed":             "Проверка слайдером не пройдена. Попробуйте снова",
			"auth.demoAccountPasswordLocked": "Пароль демонстрационной учетной записи нельзя изменить",
			"security.riskBlocked":           "Доступ заблокирован правилами безопасности",
		},
		"it-IT": {
			"auth.invalidCredentials":        "Nome utente o password non corretti",
			"auth.captchaExpired":            "La verifica con cursore è scaduta. Aggiorna e riprova",
			"auth.captchaFailed":             "Verifica con cursore non riuscita. Riprova",
			"auth.demoAccountPasswordLocked": "La password dell'account demo non può essere modificata",
			"security.riskBlocked":           "L’accesso è stato bloccato dai controlli di sicurezza",
		},
	}
	for locale, overrides := range translations {
		messages := map[string]string{}
		for key, value := range errorMessages["en-US"] {
			messages[key] = value
		}
		for key, value := range overrides {
			messages[key] = value
		}
		errorMessages[locale] = messages
	}
}

func ResolveErrorMessageKey(message string, status int) (string, map[string]interface{}) {
	text := strings.TrimSpace(message)
	if text == "" {
		return ErrorKeyForStatus(status), nil
	}
	if strings.Contains(text, "文件服务返回") {
		return "file.serviceStatusFailed", map[string]interface{}{"status": lastNumber(text)}
	}
	if strings.HasPrefix(text, "删除文件失败") {
		return "file.deleteFailed", nil
	}
	if key := zhErrorTextToKey[text]; key != "" {
		return key, nil
	}
	if strings.Contains(text, "不存在") {
		return "common.notFound", nil
	}
	if strings.Contains(text, "无权限") || strings.Contains(text, "无权") {
		return "common.permissionDenied", nil
	}
	if strings.Contains(text, "不能为空") || strings.Contains(text, "无效") || strings.Contains(text, "格式") || strings.Contains(text, "必须") || strings.Contains(text, "请选择") || strings.Contains(text, "请填写") {
		return "common.invalidRequest", nil
	}
	if strings.Contains(text, "加载") || strings.Contains(text, "获取") || strings.Contains(text, "读取") {
		return "common.loadFailed", nil
	}
	if strings.Contains(text, "保存") {
		return "common.saveFailed", nil
	}
	if strings.Contains(text, "创建") || strings.Contains(text, "生成") {
		return "common.createFailed", nil
	}
	if strings.Contains(text, "更新") || strings.Contains(text, "变更") {
		return "common.updateFailed", nil
	}
	if strings.Contains(text, "删除") {
		return "common.deleteFailed", nil
	}
	if strings.Contains(text, "导出") {
		return "common.exportFailed", nil
	}
	return ErrorKeyForStatus(status), nil
}

func ErrorKeyForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "common.badRequest"
	case http.StatusUnauthorized:
		return "common.unauthorized"
	case http.StatusForbidden:
		return "common.forbidden"
	case http.StatusNotFound:
		return "common.notFound"
	default:
		if status >= 500 {
			return "common.internal"
		}
		return "common.operationFailed"
	}
}

func ErrorCodeFromKey(key string) string {
	code := strings.TrimSpace(key)
	if code == "" {
		return "COMMON_OPERATION_FAILED"
	}
	var builder strings.Builder
	var previous rune
	for _, current := range code {
		switch {
		case current == '.' || current == '-' || current == ' ':
			if builder.Len() > 0 && !strings.HasSuffix(builder.String(), "_") {
				builder.WriteByte('_')
			}
		case current >= 'A' && current <= 'Z':
			if builder.Len() > 0 && previous != 0 && previous != '_' && ((previous >= 'a' && previous <= 'z') || (previous >= '0' && previous <= '9')) {
				builder.WriteByte('_')
			}
			builder.WriteRune(current)
		default:
			builder.WriteRune(current)
		}
		previous = current
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "COMMON_OPERATION_FAILED"
	}
	return strings.ToUpper(result)
}

func LocalizedErrorMessage(ctx *context.Context, key string, params map[string]interface{}, legacyMessage string, status int) string {
	locale := RequestLocale(ctx)
	template := lookupErrorTemplate(locale, key)
	if template == "" {
		template = lookupErrorTemplate("zh-CN", key)
	}
	if template == "" {
		template = lookupErrorTemplate(locale, ErrorKeyForStatus(status))
	}
	if template == "" && legacyMessage != "" && locale == "zh-CN" {
		template = legacyMessage
	}
	if template == "" {
		template = lookupErrorTemplate("zh-CN", "common.operationFailed")
	}
	return interpolateErrorTemplate(template, params)
}

func RequestLocale(ctx *context.Context) string {
	if ctx == nil || ctx.Request == nil {
		return "zh-CN"
	}
	header := ctx.Request.Header.Get("Accept-Language")
	if header == "" {
		return "zh-CN"
	}
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if tag == "" {
			continue
		}
		normalized := supportedErrorLocales[strings.ToLower(tag)]
		if normalized != "" {
			return normalized
		}
		if dash := strings.Index(tag, "-"); dash > 0 {
			normalized = supportedErrorLocales[strings.ToLower(tag[:dash])]
			if normalized != "" {
				return normalized
			}
		}
	}
	return "zh-CN"
}

func lookupErrorTemplate(locale, key string) string {
	if messages := errorMessages[locale]; messages != nil {
		if value := messages[key]; value != "" {
			return value
		}
	}
	return ""
}

func interpolateErrorTemplate(template string, params map[string]interface{}) string {
	if len(params) == 0 {
		return template
	}
	result := template
	for key, value := range params {
		result = strings.ReplaceAll(result, "{"+key+"}", fmt.Sprint(value))
	}
	return result
}

func mergeLegacyErrorCode(data interface{}, fallback string) (interface{}, string) {
	errorCode := fallback
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data, errorCode
	}
	if raw, ok := dataMap["error_code"]; ok {
		if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
			errorCode = value
			if value == "TENANT_DISABLED" {
				dataMap["message_key"] = "auth.tenantDisabled"
			}
			if value == "TENANT_EXPIRED" {
				dataMap["message_key"] = "auth.tenantExpired"
			}
		}
	}
	return dataMap, errorCode
}

var numberPattern = regexp.MustCompile(`\d+`)

func lastNumber(text string) string {
	matches := numberPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}
