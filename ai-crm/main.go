package main

import (
	"ai-crm/controllers"
	"ai-crm/routers"
	"ai-crm/services"
	"ai-crm/utils"
	"time"

	"github.com/beego/beego/v2/server/web"
)

func main() {
	// 初始化日志
	utils.InitLogger()

	// 初始化数据库
	utils.InitDB()
	if err := utils.InitRedis(); err != nil {
		panic("Redis连接失败: " + err.Error())
	}
	services.InitIMRealtime()
	services.InitOperationLogger()
	services.InitFileserverGateway()
	services.StartFileserverProbeScheduler()
	controllers.StartAIPricingSyncScheduler()
	controllers.StartTenantAIModelSyncScheduler()
	controllers.StartTenantExpiryNoticeScheduler()
	go services.CleanupOperationLogsWithLock(services.OperationLogRetention())
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		services.WithSchedulerLock("im_cleanup_messages", func() {
			_ = services.NewIMService().CleanupExpiredMessages(180)
		})
		for range ticker.C {
			services.WithSchedulerLock("im_cleanup_messages", func() {
				_ = services.NewIMService().CleanupExpiredMessages(180)
			})
		}
	}()
	controllers.StartLeadAutomationScheduler()
	controllers.StartTicketSLANoticeScheduler()
	controllers.StartProjectReminderScheduler()
	controllers.StartOpportunityFollowUpScheduler()

	// 初始化路由
	routers.InitRouter()

	// 启动服务
	web.Run()
}
