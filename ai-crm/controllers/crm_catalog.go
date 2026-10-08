package controllers

import (
	"ai-crm/utils"
	"strings"
)

type prdDomain struct {
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ObjectTypes []string               `json:"object_types"`
	Permission  string                 `json:"permission"`
	Statuses    []string               `json:"statuses"`
	Template    map[string]interface{} `json:"template"`
}

type prdCapability struct {
	Code        string `json:"code"`
	Section     string `json:"section"`
	Domain      string `json:"domain"`
	ObjectType  string `json:"object_type"`
	Capability  string `json:"capability"`
	Implemented bool   `json:"implemented"`
}

type prdFieldSpec struct {
	Domain     string                 `json:"domain"`
	ObjectType string                 `json:"object_type"`
	Title      string                 `json:"title"`
	Required   []string               `json:"required"`
	Optional   []string               `json:"optional"`
	Defaults   map[string]interface{} `json:"defaults"`
	Rules      []string               `json:"rules"`
	Acceptance []string               `json:"acceptance"`
}

type prdAction struct {
	Key         string                 `json:"key"`
	Domain      string                 `json:"domain"`
	ObjectType  string                 `json:"object_type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Payload     map[string]interface{} `json:"payload"`
}

var crmDomains = []prdDomain{
	{"enterprise", "企业管理", "企业信息、组织架构、销售流程、模块字段权限、数据隔离规则", []string{"company_config", "department", "sales_process", "stage", "field_permission", "data_scope_rule", "hr_sync"}, "enterprise:view", []string{"active", "disabled", "draft"}, map[string]interface{}{"timezone": "Asia/Shanghai", "currency": defaultBusinessCurrencyCode, "date_format": "YYYY-MM-DD", "language": "zh-CN"}},
	{"projects", "客户项目管理", "客户项目、里程碑、看板、交付物、客户协同空间", []string{"project", "milestone", "project_view", "deliverable", "customer_space", "project_deal_rule"}, "projects:view", []string{"planning", "active", "at_risk", "completed"}, map[string]interface{}{"manager": "", "milestones": []string{}, "progress": 0, "customer_visible": false}},
	{"products", "产品与服务管理", "产品目录、服务套餐、定价、规格、库存、订单产品行", []string{"product", "category", "service_package", "pricing_rule", "variant", "inventory", "order_line"}, "products:view", []string{"on_sale", "stopped", "testing"}, map[string]interface{}{"sku": "", "unit": "", "tax_rate": 0, "stock": 0, "stock_threshold": 0}},
	{"campaigns", "市场活动管理", "活动、模板、预算ROI、任务、参与者、转化分析", []string{"campaign", "campaign_template", "campaign_budget", "campaign_task", "participant", "attribution"}, "campaigns:view", []string{"draft", "scheduled", "running", "finished"}, map[string]interface{}{"type": "webinar", "budget_items": []interface{}{}, "utm_source": ""}},
	{"contacts", "联系人管理", "联系人、公司、互动历史、客户分级、画像、提醒、导入导出、去重", []string{"contact", "company", "interaction", "customer_grade", "profile", "anniversary", "import_export", "dedupe_job"}, "contacts:view", []string{"active", "inactive", "merged"}, map[string]interface{}{"email": "", "phone": "", "company": "", "source": "manual", "grade": "normal"}},
	{"deals", "成交记录管理", "成交金额、日期、产品明细、负责人归属、合同发票、阶段时间线、拒绝原因", []string{"closed_deal", "revenue_line", "deal_date", "owner_split", "contract_invoice", "deal_timeline", "loss_reason"}, "deals:view", []string{"closed_won", "closed_lost", "refunded"}, map[string]interface{}{"contract_no": "", "invoice_no": "", "owner_split": []interface{}{}, "line_items": []interface{}{}}},
	{"leads", "线索管理", "多渠道线索、分配回收、AI评分、跟进计划、培育、转化", []string{"lead", "assignment_rule", "recycle_rule", "lead_score", "follow_sequence", "nurture_rule", "conversion"}, "leads:view", []string{"new", "pending_assign", "following", "mql", "sdr_following", "sql", "nurturing", "converted", "invalid"}, map[string]interface{}{"company_name": "", "contact_name": "", "phone": "", "email": "", "source_channel": "manual", "utm_source": "", "intent_product": "", "temperature": "cold", "next_action": ""}},
	{"dashboards", "销售大屏与仪表盘", "实时看板、团队排行、销售漏斗、目标进度、趋势图、大屏模式", []string{"dashboard", "leaderboard", "funnel", "target_progress", "trend_chart", "big_screen"}, "dashboards:view", []string{"active", "archived"}, map[string]interface{}{"widgets": []interface{}{}, "refresh_seconds": 30}},
	{"teams", "团队管理", "团队成员、汇报关系、角色矩阵、销售目标、工作量", []string{"team_member", "reporting_line", "role_matrix", "sales_goal", "workload"}, "teams:view", []string{"active", "inactive"}, map[string]interface{}{"manager_id": 0, "skills": []interface{}{}, "max_load": 0}},
	{"insights", "运营热点追踪", "热点客户/产品、行为热力图、高频问题、内容热点、归因、交易风险、交易复盘", []string{"hot_customer", "hot_product", "behavior_heatmap", "faq_cluster", "content_hotspot", "touch_attribution", "deal_risk", "deal_review"}, "insights:view", []string{"open", "watching", "resolved", "draft", "pending_review", "published", "archived"}, map[string]interface{}{"risk_score": 0, "reason": "", "suggestion": ""}},
	{"opportunities", "交易商机管理", "可视化管道、阶段推进、报价单、预测、竞品信息、活动关联、Revenue Intelligence、风险预警", []string{"opportunity", "quote", "pipeline", "stage_gate", "forecast", "competition_note", "deal_activity", "revenue_intelligence", "risk_alert"}, "opportunities:view", []string{"initial_contact", "need_confirmed", "proposal", "negotiation", "won", "lost", "on_hold", "draft", "sent", "accepted", "rejected", "expired"}, map[string]interface{}{"stage": "initial_contact", "probability": 10, "line_items": []interface{}{}, "competitors": []interface{}{}}},
	{"tickets", "服务工单管理", "工单创建、分类、分配、SLA、处理记录、满意度、报表", []string{"ticket", "ticket_category", "ticket_assignment", "sla_rule", "ticket_note", "satisfaction", "ticket_report"}, "tickets:view", []string{"new", "assigned", "processing", "waiting_customer", "resolved", "closed", "reopened"}, map[string]interface{}{"category": "", "priority": "normal", "customer_grade": "normal", "sla_due_at": "", "public_reply": ""}},
	{"orders", "订单管理", "订单生成、审核、支付、发货、发票、历史查询、售后", []string{"order", "order_approval", "payment", "shipment", "invoice", "order_history", "return_request"}, "orders:view", []string{"draft", "pending_review", "confirmed", "partial_shipped", "shipped", "completed", "canceled"}, map[string]interface{}{"payment_status": "pending", "shipment_no": "", "invoice_type": "", "line_items": []interface{}{map[string]interface{}{"product_id": 0, "quantity": 1, "unit_price": 0}}}},
	{"workspace", "客户主面板", "待办、客户动态、通知中心、快捷入口、小组件、KPI卡片", []string{"task", "customer_activity", "mail_notification", "shortcut_group", "widget_layout", "kpi_card"}, "workspace:view", []string{"open", "done", "read", "archived"}, map[string]interface{}{"source": "", "due_at": "", "priority": "normal"}},
}

var crmFieldSpecs = []prdFieldSpec{
	{"enterprise", "company_config", "企业信息配置", []string{"full_name", "timezone", "currency", "date_format", "language"}, []string{"short_name", "logo_url"}, map[string]interface{}{"timezone": "Asia/Shanghai", "currency": defaultBusinessCurrencyCode, "date_format": "YYYY-MM-DD", "language": "zh-CN"}, []string{"仅管理员可编辑", "修改即时写入操作日志"}, []string{"时区和货币配置保存后可从 /api/v1/prd/settings 查询", "操作日志保留修改记录"}},
	{"enterprise", "department", "组织架构设置", []string{"department_code", "manager"}, []string{"parent_department_id", "hr_sync", "children"}, map[string]interface{}{"hr_sync": "manual"}, []string{"存在子部门或关联人员时禁止删除"}, []string{"删除有子部门的部门返回 400", "可通过 department_id 建立树状层级"}},
	{"enterprise", "sales_process", "销售流程自定义", []string{"stages"}, []string{"stage_required_fields", "warning_days"}, map[string]interface{}{"stages": []string{"pipeline", "proposal", "negotiation", "won"}}, []string{"阶段推进必须满足必填字段"}, []string{"缺少阶段必填字段时更新商机返回 400"}},
	{"enterprise", "field_permission", "模块/字段权限基础设置", []string{"domain", "hidden", "readonly"}, []string{"bypass_user_ids"}, map[string]interface{}{"hidden": []string{}, "readonly": []string{}}, []string{"隐藏字段列表查询时脱敏", "只读字段禁止非管理员写入"}, []string{"非管理员写入只读字段返回 400"}},
	{"enterprise", "data_scope_rule", "数据隔离规则", []string{"mode"}, []string{"conditions", "piercing_user_ids"}, map[string]interface{}{"mode": "shared"}, []string{"支持 owner、department、team、region、condition"}, []string{"开启 owner 后普通用户仅可见本人数据"}},
	{"projects", "project", "客户项目创建", []string{"manager", "milestones"}, []string{"customer_visible", "progress", "project_tags"}, map[string]interface{}{"progress": 0, "customer_visible": false}, []string{"订单履约可自动生成项目", "项目完成可回写关联交易阶段"}, []string{"订单履约接口返回 project", "完成项目后项目状态为 completed"}},
	{"projects", "milestone", "里程碑跟踪", []string{"planned_date", "completion_rule"}, []string{"warning_days", "confirmed_by_customer"}, map[string]interface{}{"completion_rule": "manual"}, []string{"最多 30 个里程碑", "到期预警进入工作台"}, []string{"里程碑记录可关联项目"}},
	{"projects", "deliverable", "交付物管理", []string{"file_name", "version"}, []string{"approver_ids", "customer_downloadable"}, map[string]interface{}{"version": "V1.0", "customer_downloadable": false}, []string{"支持审批发起", "客户可见性由 payload 控制"}, []string{"交付物可调用对象审批接口"}},
	{"products", "product", "产品目录管理", []string{"sku", "category", "unit"}, []string{"images", "description", "tax_rate", "stock", "stock_threshold"}, map[string]interface{}{"unit": "套", "stock": 0, "stock_threshold": 0}, []string{"未填 SKU 时自动生成", "订单履约时校验库存"}, []string{"产品创建后 code/payload.sku 不为空", "库存不足时订单履约返回 500/错误信息"}},
	{"products", "service_package", "服务套餐配置", []string{"billing_type", "base_items"}, []string{"addons", "mutual_exclusion"}, map[string]interface{}{"billing_type": "monthly"}, []string{"支持一次性、订阅和用量计费"}, []string{"套餐记录可保存互斥规则"}},
	{"products", "pricing_rule", "定价策略", []string{"price_type", "standard_price"}, []string{"tiers", "customer_id", "discount_threshold"}, map[string]interface{}{"price_type": "standard"}, []string{"客户价优先于阶梯价，阶梯价优先于标准价", "低价折扣可发起审批"}, []string{"定价记录可发起审批实例"}},
	{"campaigns", "campaign", "活动创建", []string{"type", "budget_items", "owner"}, []string{"landing_page", "utm_source", "plan_id"}, map[string]interface{}{"type": "webinar", "budget_items": []interface{}{}}, []string{"活动可捕获线索", "预算 ROI 从交易归因汇总"}, []string{"活动捕获线索接口返回 lead 和 task"}},
	{"campaigns", "campaign_template", "活动模板", []string{"layout", "email_template"}, []string{"variables", "thank_you_page"}, map[string]interface{}{"variables": []string{"活动名称", "日期"}}, []string{"模板可生成活动任务"}, []string{"模板记录可创建并复用"}},
	{"contacts", "contact", "联系人档案管理", []string{"email", "phone", "company"}, []string{"title", "source", "grade", "custom_fields"}, map[string]interface{}{"source": "manual", "grade": "normal"}, []string{"邮箱查重", "支持字段权限"}, []string{"联系人合并后辅记录状态为 merged"}},
	{"contacts", "interaction", "互动历史时间轴", []string{"interaction_type", "summary", "happened_at"}, []string{"source_module", "related_object_id"}, map[string]interface{}{"interaction_type": "note"}, []string{"统一承载邮件、电话、会议、工单、活动记录"}, []string{"互动记录可与任意业务对象建立关系"}},
	{"contacts", "company", "公司管理", []string{"industry"}, []string{"open_deal_amount", "recent_interaction", "grade"}, map[string]interface{}{"grade": "normal"}, []string{"公司可关联多个联系人"}, []string{"关系接口能查询公司联系人关联"}},
	{"deals", "closed_deal", "成交记录", []string{"signed_at", "line_items"}, []string{"contract_no", "invoice_no", "owner_split", "loss_reason"}, map[string]interface{}{"line_items": []interface{}{}}, []string{"赢单自动生成成交记录", "成交可生成订单"}, []string{"商机赢单接口返回 deal，订单从成交记录生成"}},
	{"leads", "lead", "线索管理", []string{"email", "utm_source"}, []string{"title", "company", "product_page_visits", "form_completeness", "next_action"}, map[string]interface{}{"intent": "medium", "product_page_visits": 0}, []string{"自动评分", "支持分配、回收、转化"}, []string{"线索评分接口返回 score 和 breakdown", "转化后线索状态为 converted"}},
	{"leads", "follow_sequence", "跟进计划", []string{"steps"}, []string{"ab_version", "personalization"}, map[string]interface{}{"steps": []interface{}{}}, []string{"步骤支持邮件、电话、等待"}, []string{"计划记录可保存多步骤 payload"}},
	{"dashboards", "dashboard", "实时销售数据看板", []string{"widgets"}, []string{"refresh_seconds", "time_filter"}, map[string]interface{}{"refresh_seconds": 30}, []string{"KPI 由统一对象实时汇总"}, []string{"驾驶舱接口返回 kpi"}},
	{"accounts", "user_account", "用户账号管理", []string{"name", "email", "status"}, []string{"phone", "sso_provider", "ad_sync"}, map[string]interface{}{"status": "active"}, []string{"账号接口管理真实 users 表"}, []string{"用户列表接口返回账号数据"}},
	{"accounts", "mfa_policy", "多因素认证", []string{"required", "methods"}, []string{"scope"}, map[string]interface{}{"required": false, "methods": []string{"totp"}}, []string{"策略作为租户配置保存"}, []string{"配置接口可保存 mfa_policy"}},
	{"teams", "team_member", "团队成员信息", []string{"email", "skills"}, []string{"hire_date", "current_load", "max_load"}, map[string]interface{}{"skills": []interface{}{}, "max_load": 25}, []string{"技能用于线索分配", "负载超阈值停止分配"}, []string{"团队成员 payload 可记录负载"}},
	{"teams", "reporting_line", "汇报关系", []string{"manager_id"}, []string{"dotted_manager_id"}, map[string]interface{}{}, []string{"调整后目标归属可重新计算"}, []string{"关系记录可表达直属和虚线汇报"}},
	{"insights", "deal_risk", "交易风险洞察", []string{"opportunity_id", "reason", "risk_score"}, []string{"suggestion"}, map[string]interface{}{"risk_score": 70}, []string{"商机风险生成任务推送主面板"}, []string{"风险接口返回 risk 和 task"}},
	{"insights", "faq_cluster", "高频问题库", []string{"question", "answer"}, []string{"keywords", "source_ticket_ids"}, map[string]interface{}{"source": "ticket_cluster"}, []string{"工单可生成知识条目"}, []string{"工单生成知识接口返回 article"}},
	{"insights", "deal_review", "交易复盘知识", []string{"review_type", "opportunity_id", "summary"}, []string{"win_reasons", "loss_reasons", "lessons", "reusable_playbook", "avoid_next_time", "tags", "keywords"}, map[string]interface{}{"source": "opportunity_review", "review_type": "won"}, []string{"商机可创建或更新赢单/输单复盘", "发布后进入知识库阅读"}, []string{"同一商机只保留一个复盘对象", "发布状态复盘可在知识库检索"}},
	{"opportunities", "opportunity", "可视化管道", []string{"stage", "probability", "next_activity_date"}, []string{"forecast_category", "proposal_url", "quote_amount", "competitors"}, map[string]interface{}{"stage": "pipeline", "probability": 0.2}, []string{"阶段门禁校验", "赢单生成成交和订单"}, []string{"阶段推进接口返回 updated opportunity", "赢单接口返回 deal/order"}},
	{"opportunities", "quote", "报价单", []string{"line_items", "quote_amount"}, []string{"quote_no", "valid_until", "sent_at", "accepted_at"}, map[string]interface{}{"quote_status": "draft", "line_items": []interface{}{}}, []string{"从商机生成报价单", "发送和接受报价都会写入商机时间线"}, []string{"报价发送后商机可进入谈判/评审", "报价可生成订单草稿"}},
	{"opportunities", "deal_activity", "活动关联", []string{"activity_type", "summary"}, []string{"next_step", "happened_at"}, map[string]interface{}{"activity_type": "call"}, []string{"活动进入交易时间线"}, []string{"阶段推进会生成 deal_activity"}},
	{"tickets", "ticket", "工单创建", []string{"category", "priority", "customer_grade"}, []string{"public_reply", "internal_note", "sla_due_at"}, map[string]interface{}{"priority": "normal", "customer_grade": "normal"}, []string{"创建时自动计算 SLA", "关闭后记录满意度"}, []string{"工单创建 payload 包含 sla_due_at", "关闭低分会生成回访待办"}},
	{"tickets", "ticket_note", "处理记录", []string{"visibility", "content"}, []string{"agent_id"}, map[string]interface{}{"visibility": "internal"}, []string{"内部备注客户不可见，公开回复客户可见"}, []string{"处理记录可关联工单"}},
	{"orders", "order", "订单管理", []string{"line_items", "payment_status"}, []string{"shipment_no", "invoice_type", "return_reason"}, map[string]interface{}{"payment_status": "pending", "line_items": []interface{}{}}, []string{"支付、发货、开票、售后独立留痕", "服务类订单可履约成项目"}, []string{"支付接口更新 payment_status", "发货接口写入 shipment_no"}},
	{"orders", "invoice", "发票管理", []string{"invoice_no", "invoice_type"}, []string{"tax_no", "sent_to"}, map[string]interface{}{"invoice_type": "special_vat"}, []string{"订单可生成发票信息并锁定关键字段"}, []string{"开票接口返回 invoice payload"}},
	{"workspace", "task", "待办任务列表", []string{"source", "due_at", "priority"}, []string{"related_domain", "related_id"}, map[string]interface{}{"priority": "normal"}, []string{"完成后即时刷新驾驶舱"}, []string{"完成接口返回 status=done"}},
	{"workspace", "mail_notification", "邮件通知中心", []string{"category", "content"}, []string{"read_at", "source"}, map[string]interface{}{"category": "system"}, []string{"支持标记已读"}, []string{"标记已读接口返回 status=read"}},
	{"workspace", "widget_layout", "自定义小组件", []string{"widgets"}, []string{"layout_name"}, map[string]interface{}{"widgets": []interface{}{}}, []string{"布局作为对象保存，可多套"}, []string{"创建 widget_layout 后主面板可查询"}},
	{"workspace", "kpi_card", "KPI 卡片", []string{"metric"}, []string{"drilldown_domain"}, map[string]interface{}{"metric": "won_amount"}, []string{"点击 KPI 可下钻到对应业务域"}, []string{"驾驶舱接口返回 KPI 聚合值"}},
}

var crmActions = []prdAction{
	{"campaign_capture_lead", "campaigns", "campaign", "活动捕获线索", "从活动参与者直接生成线索和主面板待办", "POST", "/api/v1/prd/flows/campaigns/{id}/capture-lead", map[string]interface{}{"name": "活动线索", "email": "lead@example.com", "company": "客户公司"}},
	{"campaign_attribution", "campaigns", "campaign", "生成活动归因", "汇总活动关联线索和成交金额，生成多触点归因洞察", "POST", "/api/v1/prd/flows/campaigns/{id}/attribution", map[string]interface{}{}},
	{"lead_score", "leads", "lead", "AI 线索评分", "按邮箱域名、职位、页面访问、表单完整度计算 0-100 分并生成跟进任务", "POST", "/api/v1/prd/flows/leads/{id}/score", map[string]interface{}{}},
	{"lead_assign", "leads", "lead", "线索分配", "把线索分配给负责人，同时创建跟进待办", "POST", "/api/v1/prd/flows/leads/{id}/assign", map[string]interface{}{"owner_id": 1, "reason": "manual_assign"}},
	{"lead_recycle", "leads", "lead", "线索回收", "超时未跟进的线索退回公共池", "POST", "/api/v1/prd/flows/leads/{id}/recycle", map[string]interface{}{"reason": "超时未跟进"}},
	{"lead_convert", "leads", "lead", "线索转联系人/交易", "把线索转为联系人和交易商机，并保留关联关系", "POST", "/api/v1/prd/flows/leads/{id}/convert", map[string]interface{}{"company_name": "客户公司", "deal_name": "新交易", "amount": 100000}},
	{"contact_merge", "contacts", "contact", "联系人去重合并", "合并重复联系人并迁移关联关系", "POST", "/api/v1/prd/flows/contacts/merge", map[string]interface{}{"primary_id": 1, "duplicate_ids": []uint{2}, "reason": "duplicate_email"}},
	{"insight_hotspots", "insights", "hot_customer", "生成热点洞察", "按成交金额和产品库存/订单数据生成热点客户与产品", "POST", "/api/v1/prd/flows/insights/hotspots", map[string]interface{}{}},
	{"opportunity_stage", "opportunities", "opportunity", "阶段推进", "按销售流程门禁推进商机阶段并写入活动时间线", "POST", "/api/v1/prd/flows/opportunities/{id}/stage", map[string]interface{}{"stage": "proposal", "comment": "进入方案阶段"}},
	{"opportunity_quote", "opportunities", "quote", "生成报价单", "从商机产品明细生成报价单，报价发送后可进入谈判/评审", "POST", "/api/v1/prd/flows/opportunities/{id}/quote", map[string]interface{}{"reason": "给客户第一版报价"}},
	{"quote_send", "opportunities", "quote", "发送报价", "记录报价已发送和发送说明，写入商机时间线", "POST", "/api/v1/prd/flows/quotes/{id}/send", map[string]interface{}{"reason": "已发给客户确认"}},
	{"quote_accept", "opportunities", "quote", "接受报价", "客户接受报价，可继续转为订单草稿", "POST", "/api/v1/prd/flows/quotes/{id}/accept", map[string]interface{}{"reason": "客户确认接受"}},
	{"quote_order", "opportunities", "quote", "报价生成订单", "把已接受报价的明细和金额生成订单草稿", "POST", "/api/v1/prd/flows/quotes/{id}/order", map[string]interface{}{}},
	{"opportunity_win", "opportunities", "opportunity", "商机赢单", "赢单后生成成交记录，订单从成交记录继续生成", "POST", "/api/v1/prd/flows/opportunities/{id}/win", map[string]interface{}{}},
	{"opportunity_review", "opportunities", "opportunity", "保存商机复盘", "为商机创建或更新赢单/输单复盘草稿", "POST", "/api/v1/prd/flows/opportunities/{id}/review", map[string]interface{}{"review_type": "won", "summary": "关键结论"}},
	{"opportunity_review_submit", "opportunities", "opportunity", "提交复盘审核", "把商机复盘提交为待审核状态", "POST", "/api/v1/prd/flows/opportunity-reviews/{id}/submit", map[string]interface{}{}},
	{"opportunity_review_publish", "insights", "deal_review", "发布交易复盘", "审核通过后发布到知识库供员工阅读", "POST", "/api/v1/prd/flows/opportunity-reviews/{id}/publish", map[string]interface{}{}},
	{"opportunity_risk", "opportunities", "opportunity", "生成风险洞察", "根据停滞和互动频率生成风险记录与经理待办", "POST", "/api/v1/prd/flows/opportunities/{id}/risk", map[string]interface{}{}},
	{"deal_order", "deals", "closed_deal", "成交生成订单", "把成交产品明细带入订单", "POST", "/api/v1/prd/flows/deals/{id}/order", map[string]interface{}{}},
	{"order_pay", "orders", "order", "标记支付", "记录支付流水、方式、金额和时间", "POST", "/api/v1/prd/flows/orders/{id}/pay", map[string]interface{}{"payment_no": "PAY-001", "payment_method": "manual", "paid_amount": 100000}},
	{"order_ship", "orders", "order", "订单发货", "写入物流单号和承运商", "POST", "/api/v1/prd/flows/orders/{id}/ship", map[string]interface{}{"shipment_no": "SHP-001", "carrier": "manual"}},
	{"order_invoice", "orders", "order", "订单开票", "记录发票号、类型、税号并更新订单历史", "POST", "/api/v1/prd/flows/orders/{id}/invoice", map[string]interface{}{"invoice_no": "INV-001", "invoice_type": "special_vat"}},
	{"order_payment_plan", "orders", "order", "保存付款计划", "把分期节点、到期日和逾期金额写入订单", "POST", "/api/v1/prd/flows/orders/{id}/payment-plan", map[string]interface{}{"payment_plan": []map[string]interface{}{{"name": "首付款", "amount": 50000, "due_date": "2026-06-01"}}}},
	{"order_return", "orders", "order", "退换货售后", "发起退换货并生成售后工单", "POST", "/api/v1/prd/flows/orders/{id}/return", map[string]interface{}{"reason": "客户申请换货"}},
	{"order_fulfill", "orders", "order", "订单履约成项目", "扣减库存并生成客户项目", "POST", "/api/v1/prd/flows/orders/{id}/fulfill", map[string]interface{}{}},
	{"order_ticket", "orders", "order", "订单生成工单", "从订单创建售后服务工单", "POST", "/api/v1/prd/flows/orders/{id}/ticket", map[string]interface{}{}},
	{"project_complete", "projects", "project", "完成项目", "项目完成并回写关联交易/动态", "POST", "/api/v1/prd/flows/projects/{id}/complete", map[string]interface{}{}},
	{"ticket_note", "tickets", "ticket", "记录处理过程", "新增内部备注或公开回复", "POST", "/api/v1/prd/flows/tickets/{id}/note", map[string]interface{}{"visibility": "internal", "content": "已跟进"}},
	{"ticket_knowledge", "tickets", "ticket", "工单生成知识", "从高频问题工单生成知识条目", "POST", "/api/v1/prd/flows/tickets/{id}/knowledge", map[string]interface{}{}},
	{"ticket_knowledge_suggest", "tickets", "ticket", "推荐知识", "按工单分类和描述推荐已有知识条目", "GET", "/api/v1/prd/flows/tickets/{id}/knowledge-suggestions", map[string]interface{}{}},
	{"ticket_auto_assign", "tickets", "ticket", "自动分派工单", "按问题类型和客服负载自动选择处理人", "POST", "/api/v1/prd/flows/tickets/{id}/auto-assign", map[string]interface{}{"reason": "按规则自动分派"}},
	{"ticket_transfer", "tickets", "ticket", "工单内部转派", "一线支持无法解决时转派给二线专家并保留处理记录", "POST", "/api/v1/prd/flows/tickets/{id}/transfer", map[string]interface{}{"owner_id": 2, "reason": "需二线技术介入"}},
	{"ticket_create_bug", "tickets", "ticket", "工单转研发Bug", "把工单问题转为研发Bug任务并建立关联", "POST", "/api/v1/prd/flows/tickets/{id}/bug", map[string]interface{}{"name": "研发Bug：xxx", "severity": "high", "module": "billing", "description": "复现步骤...", "reason": "缺陷排查", "owner_id": 1}},
	{"ticket_sync_bug_status", "tickets", "ticket", "研发Bug状态回写", "回写Bug状态到工单并自动调整工单状态（也支持研发任务更新后自动回写）", "POST", "/api/v1/prd/flows/tickets/{id}/bug-status", map[string]interface{}{"status": "fixed", "reason": "代码已修复并发布", "sync_bug": true}},
	{"ticket_escalation_queue", "tickets", "ticket", "SLA升级队列", "列出已超时或高优先级待升级工单", "GET", "/api/v1/prd/flows/tickets/escalation-queue", map[string]interface{}{}},
	{"ticket_transfer_queue", "tickets", "ticket", "转派队列", "查看一线/二线/产品经理队列中的待处理工单", "GET", "/api/v1/prd/flows/tickets/transfer-queue", map[string]interface{}{"limit": 50, "group_by": "queue"}},
	{"ticket_transfer_stats", "tickets", "ticket", "转派统计", "统计转派后处理效率、超时和重开情况", "GET", "/api/v1/prd/flows/tickets/transfer-stats", map[string]interface{}{"days": 30}},
	{"ticket_sla_escalate", "tickets", "ticket", "SLA升级", "逾期或高优先级工单升级并创建经理待办", "POST", "/api/v1/prd/flows/tickets/{id}/escalate", map[string]interface{}{"reason": "SLA 即将超时", "force": true}},
	{"ticket_close", "tickets", "ticket", "关闭并评价", "关闭工单，低满意度自动创建回访任务", "POST", "/api/v1/prd/flows/tickets/{id}/close", map[string]interface{}{"csat": 5, "nps": 9, "comment": "已解决"}},
	{"workspace_done", "workspace", "task", "完成待办", "完成工作台任务并刷新 KPI", "POST", "/api/v1/prd/flows/workspace/{id}/done", map[string]interface{}{}},
	{"workspace_read", "workspace", "mail_notification", "通知已读", "标记邮件/系统通知为已读", "POST", "/api/v1/prd/flows/workspace/{id}/read", map[string]interface{}{}},
	{"module_access_preview", "enterprise", "field_permission", "权限预览", "按当前用户实时预览各模块读写权限等级", "GET", "/api/v1/prd/module-access-preview", map[string]interface{}{}},
	{"approval_pending_summary", "workspace", "task", "审批待办汇总", "统计待审批总量与超时审批", "GET", "/api/v1/prd/approvals/pending-summary", map[string]interface{}{"overdue_hours": 48}},
	{"integration_health", "enterprise", "hr_sync", "集成健康检查", "汇总各集成配置启用状态和最近执行结果", "GET", "/api/v1/prd/integrations/health", map[string]interface{}{}},
	{"next_best_actions", "workspace", "kpi_card", "下一步建议", "基于商机/工单状态自动生成优先处理建议", "GET", "/api/v1/prd/next-actions", map[string]interface{}{"limit": 20}},
}

var crmCapabilities = []prdCapability{
	{"3.1.1", "企业信息配置", "enterprise", "company_config", "object+settings", true},
	{"3.1.2", "组织架构设置", "enterprise", "department", "object+relation+delete_guard", true},
	{"3.1.3", "销售流程自定义", "enterprise", "sales_process", "settings+stage_gate", true},
	{"3.1.4", "模块/字段权限基础设置", "enterprise", "field_permission", "settings+field_acl+rbac", true},
	{"3.1.5", "数据隔离规则", "enterprise", "data_scope_rule", "settings+data_scope", true},
	{"3.2.1", "客户项目创建", "projects", "project", "object+order_fulfill_flow", true},
	{"3.2.2", "里程碑跟踪", "projects", "milestone", "object", true},
	{"3.2.3", "项目进度看板", "projects", "project_view", "object+payload", true},
	{"3.2.4", "交付物管理", "projects", "deliverable", "object+approval", true},
	{"3.2.5", "客户协同空间", "projects", "customer_space", "object+portal_payload", true},
	{"3.2.6", "项目与交易关联", "projects", "project_deal_rule", "relation+complete_flow", true},
	{"3.3.1", "产品目录管理", "products", "product", "object+import_export", true},
	{"3.3.2", "服务套餐配置", "products", "service_package", "object", true},
	{"3.3.3", "定价策略", "products", "pricing_rule", "object+approval", true},
	{"3.3.4", "产品规格管理", "products", "variant", "object", true},
	{"3.3.5", "库存状态监控", "products", "inventory", "order_inventory_deduction", true},
	{"3.3.6", "产品与服务关联订单", "products", "order_line", "order_line_payload", true},
	{"3.4.1", "活动创建", "campaigns", "campaign", "object", true},
	{"3.4.2", "活动模板", "campaigns", "campaign_template", "object", true},
	{"3.4.3", "预算与ROI分析", "campaigns", "campaign_budget", "object+payload", true},
	{"3.4.4", "活动任务分配", "campaigns", "campaign_task", "object", true},
	{"3.4.5", "参与者追踪", "campaigns", "participant", "object+lead_capture_flow", true},
	{"3.4.6", "活动线索转化分析", "campaigns", "attribution", "relation+payload+attribution_flow", true},
	{"3.5.1", "联系人档案管理", "contacts", "contact", "object+field_acl", true},
	{"3.5.2", "互动历史时间轴", "contacts", "interaction", "object+relations", true},
	{"3.5.3", "公司管理", "contacts", "company", "object+relations", true},
	{"3.5.4", "客户分级", "contacts", "customer_grade", "object+payload", true},
	{"3.5.5", "客户画像", "contacts", "profile", "object+payload", true},
	{"3.5.6", "生日/纪念日提醒", "contacts", "anniversary", "object+workspace_task", true},
	{"3.5.7", "批量导入导出", "contacts", "import_export", "import_export_api", true},
	{"3.5.8", "数据去重", "contacts", "dedupe_job", "merge_flow", true},
	{"3.6.1", "成交金额", "deals", "revenue_line", "object+amount", true},
	{"3.6.2", "成交日期", "deals", "deal_date", "payload", true},
	{"3.6.3", "成交产品明细", "deals", "closed_deal", "line_items_payload", true},
	{"3.6.4", "负责人归属", "deals", "owner_split", "payload", true},
	{"3.6.5", "合同/发票关联", "deals", "contract_invoice", "payload", true},
	{"3.6.6", "成交阶段时间线", "deals", "deal_timeline", "object+relation", true},
	{"3.6.7", "拒绝原因分析", "deals", "loss_reason", "object+payload", true},
	{"3.7.1", "多渠道线索捕获", "leads", "lead", "object+campaign_capture_flow", true},
	{"3.7.2", "线索分配与回收", "leads", "assignment_rule", "assign+recycle_flow", true},
	{"3.7.3", "AI线索评分", "leads", "lead_score", "score_field+payload", true},
	{"3.7.4", "跟进计划", "leads", "follow_sequence", "object", true},
	{"3.7.5", "线索培育", "leads", "nurture_rule", "object", true},
	{"3.7.6", "线索转联系人/交易", "leads", "conversion", "convert_flow", true},
	{"3.8.1", "实时销售数据看板", "dashboards", "dashboard", "cockpit+kpi", true},
	{"3.8.2", "团队业绩排行", "dashboards", "leaderboard", "object", true},
	{"3.8.3", "销售漏斗", "dashboards", "funnel", "object+summary", true},
	{"3.8.4", "目标进度跟踪", "dashboards", "target_progress", "object", true},
	{"3.8.5", "趋势图表", "dashboards", "trend_chart", "object", true},
	{"3.8.6", "大屏模式", "dashboards", "big_screen", "object", true},
	{"3.9.1", "用户账号管理", "accounts", "user_account", "accounts_api", true},
	{"3.9.2", "角色定义", "accounts", "role", "rbac_api", true},
	{"3.9.3", "数据访问权限分配", "accounts", "field_acl", "rbac+settings", true},
	{"3.9.4", "多因素认证", "accounts", "mfa_policy", "policy_object", true},
	{"3.9.5", "登录日志", "accounts", "login_log", "operation_log_api", true},
	{"3.9.6", "操作审计日志", "accounts", "audit_log_policy", "operation_log_api", true},
	{"3.10.1", "团队成员信息", "teams", "team_member", "object", true},
	{"3.10.2", "汇报关系", "teams", "reporting_line", "object+relation", true},
	{"3.10.3", "角色视图", "teams", "role_matrix", "object+rbac", true},
	{"3.10.4", "销售目标设定", "teams", "sales_goal", "object", true},
	{"3.10.5", "工作量视图", "teams", "workload", "object", true},
	{"3.11.1", "热点客户/产品分析", "insights", "hot_customer", "object+hotspot_flow", true},
	{"3.11.2", "销售行为热力图", "insights", "behavior_heatmap", "object", true},
	{"3.11.3", "高频问题库", "insights", "faq_cluster", "ticket_knowledge_flow", true},
	{"3.11.4", "内容热点追踪", "insights", "content_hotspot", "object", true},
	{"3.11.5", "多触点收益归因", "insights", "touch_attribution", "object+relation", true},
	{"3.11.6", "交易风险洞察", "insights", "deal_risk", "risk_flow", true},
	{"3.12.1", "可视化管道", "opportunities", "pipeline", "object+status", true},
	{"3.12.2", "阶段推进", "opportunities", "stage_gate", "settings+validation", true},
	{"3.12.3", "预计金额/日期", "opportunities", "forecast", "amount+payload", true},
	{"3.12.4", "竞争信息备注", "opportunities", "competition_note", "payload", true},
	{"3.12.5", "活动关联", "opportunities", "deal_activity", "relation", true},
	{"3.12.6", "Revenue Intelligence洞察", "opportunities", "revenue_intelligence", "insight_object", true},
	{"3.12.7", "交易风险预警", "opportunities", "risk_alert", "risk_flow", true},
	{"3.13.1", "工单创建", "tickets", "ticket", "object+portal", true},
	{"3.13.2", "工单分类", "tickets", "ticket_category", "object+payload", true},
	{"3.13.3", "工单分配", "tickets", "ticket_assignment", "object+payload", true},
	{"3.13.4", "SLA监控", "tickets", "sla_rule", "settings+auto_due+escalation_flow", true},
	{"3.13.5", "处理记录", "tickets", "ticket_note", "object+payload", true},
	{"3.13.6", "满意度评价", "tickets", "satisfaction", "close_flow", true},
	{"3.13.7", "工单报表", "tickets", "ticket_report", "object", true},
	{"3.14.1", "订单生成", "orders", "order", "deal_order_flow", true},
	{"3.14.2", "订单审核流程", "orders", "order_approval", "approval_api", true},
	{"3.14.3", "支付状态跟踪", "orders", "payment", "pay_flow", true},
	{"3.14.4", "发货管理", "orders", "shipment", "ship_flow", true},
	{"3.14.5", "发票管理", "orders", "invoice", "payload", true},
	{"3.14.6", "订单历史查询", "orders", "order_history", "object+export", true},
	{"3.15.1", "待办任务列表", "workspace", "task", "cockpit+done_flow", true},
	{"3.15.2", "近期客户动态时间轴", "workspace", "customer_activity", "cockpit", true},
	{"3.15.3", "邮件通知中心", "workspace", "mail_notification", "object", true},
	{"3.15.4", "快捷入口分组", "workspace", "shortcut_group", "object", true},
	{"3.15.5", "自定义小组件", "workspace", "widget_layout", "object", true},
	{"3.15.6", "KPI卡片", "workspace", "kpi_card", "cockpit+kpi", true},
}

func (c *CRMController) ListDomains() {
	utils.Success(c.Ctx, map[string]interface{}{"items": crmDomains})
}

func (c *CRMController) Coverage() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domainFilter := strings.TrimSpace(c.Ctx.Input.Query("domain"))
	items := make([]prdCapability, 0, len(crmCapabilities))
	byDomain := map[string]map[string]interface{}{}
	for _, capability := range crmCapabilities {
		if domainFilter != "" && capability.Domain != domainFilter {
			continue
		}
		items = append(items, capability)
		stats := byDomain[capability.Domain]
		if stats == nil {
			stats = map[string]interface{}{"domain": capability.Domain, "total": 0, "implemented": 0}
			byDomain[capability.Domain] = stats
		}
		stats["total"] = stats["total"].(int) + 1
		if capability.Implemented {
			stats["implemented"] = stats["implemented"].(int) + 1
		}
	}
	summary := make([]map[string]interface{}, 0, len(byDomain))
	for _, domain := range crmDomains {
		if stats := byDomain[domain.Key]; stats != nil {
			total := stats["total"].(int)
			implemented := stats["implemented"].(int)
			stats["name"] = domain.Name
			stats["coverage_rate"] = float64(implemented) / float64(total)
			summary = append(summary, stats)
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "summary": summary, "tenant_id": auth.TenantID})
}

func (c *CRMController) Schema() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domainFilter := strings.TrimSpace(c.Ctx.Input.Query("domain"))
	items := make([]prdFieldSpec, 0, len(crmFieldSpecs))
	for _, spec := range crmFieldSpecs {
		if domainFilter != "" && spec.Domain != domainFilter {
			continue
		}
		items = append(items, spec)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "tenant_id": auth.TenantID})
}

func (c *CRMController) Actions() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domainFilter := strings.TrimSpace(c.Ctx.Input.Query("domain"))
	items := make([]prdAction, 0, len(crmActions))
	for _, action := range crmActions {
		if domainFilter != "" && action.Domain != domainFilter {
			continue
		}
		items = append(items, action)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "tenant_id": auth.TenantID})
}
