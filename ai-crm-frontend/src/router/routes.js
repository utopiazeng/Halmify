import {
  Avatar,
  Connection,
  DataAnalysis,
  DataBoard,
  Files,
  Finished,
  Histogram,
  Management,
  MessageBox,
  OfficeBuilding,
  Promotion,
  Sell,
  Service,
  Monitor,
  FolderOpened,
  ShoppingCart,
  SuitcaseLine
} from '@element-plus/icons-vue'

const LoginView = () => import('../views/LoginView.vue')
const ActivateAccountView = () => import('../views/ActivateAccountView.vue')
const NewPRDWorkspaceView = () => import('../views/NewPRDWorkspaceView.vue')
const SystemManagementView = () => import('../views/system/SystemManagementView.vue')
const TenantSettingsView = () => import('../views/system/TenantSettingsView.vue')
const IoTDevicesView = () => import('../views/system/IoTDevicesView.vue')
const CampaignsView = () => import('../views/domains/CampaignsView.vue')
const AccountsView = () => import('../views/domains/AccountsView.vue')
const ContactsView = () => import('../views/domains/ContactsView.vue')
const DashboardsView = () => import('../views/domains/DashboardsView.vue')
const DealsView = () => import('../views/domains/DealsView.vue')
const InsightsView = () => import('../views/domains/InsightsView.vue')
const TenantAIAnalysisView = () => import('../views/domains/TenantAIAnalysisView.vue')
const LeadsView = () => import('../views/domains/LeadsView.vue')
const OpportunitiesView = () => import('../views/domains/OpportunitiesView.vue')
const OrdersView = () => import('../views/OrdersEnhancedView.vue')
const ProductsView = () => import('../views/domains/ProductsView.vue')
const ProjectsView = () => import('../views/domains/ProjectsView.vue')
const TeamsView = () => import('../views/domains/TeamsView.vue')
const TicketsView = () => import('../views/domains/TicketsView.vue')
const WorkspaceView = () => import('../views/domains/WorkspaceView.vue')
const StorageView = () => import('../views/domains/StorageView.vue')
const PlatformOverviewView = () => import('../views/platform/PlatformOverviewView.vue')
const PlatformTenantsView = () => import('../views/platform/PlatformTenantsView.vue')
const PlatformSettingsView = () => import('../views/platform/PlatformSettingsView.vue')
const PlatformExpiringView = () => import('../views/platform/PlatformExpiringView.vue')
const PlatformRuntimeView = () => import('../views/platform/PlatformRuntimeView.vue')
const PlatformPlansView = () => import('../views/platform/PlatformPlansView.vue')
const PlatformAIAnalysisView = () => import('../views/platform/PlatformAIAnalysisView.vue')
const PlatformIoTFirmwareView = () => import('../views/platform/PlatformIoTFirmwareView.vue')
const IMCenterView = () => import('../views/im/IMCenterView.vue')
const ManufacturingPlaceholderView = () => import('../views/industry/ManufacturingPlaceholderView.vue')

export const authRoutes = [
  {
    path: '/',
    name: 'login',
    component: LoginView,
    meta: { layout: 'auth' }
  },
  {
    path: '/activate-account',
    name: 'activate-account',
    component: ActivateAccountView,
    meta: { layout: 'auth' }
  }
]

const routeMeta = (title, description, extra = {}) => ({
  title,
  description,
  titleKey: `nav.${title}`,
  descKey: description ? `navDesc.${description}` : '',
  ...extra
})

export const navigationRoutes = [
  {
    path: '/workspace',
    name: 'workspace',
    component: WorkspaceView,
    meta: routeMeta('客户主面板', '待办、客户动态、通知与跨模块 KPI。', { requiresAuth: true, icon: DataBoard, permission: 'workspace:view', menuPermission: 'menu:workspace', domain: 'workspace', group: 'workbench', order: 1 })
  },
  {
    path: '/projects',
    name: 'projects',
    component: ProjectsView,
    meta: routeMeta('客户项目', '项目、里程碑、交付物和客户协同空间。', { requiresAuth: true, icon: SuitcaseLine, permission: 'projects:view', menuPermission: 'menu:projects', domain: 'projects', group: 'service', order: 2 })
  },
  {
    path: '/products',
    name: 'products',
    component: ProductsView,
    meta: routeMeta('产品服务', '产品目录、套餐、定价、规格与库存。', { requiresAuth: true, icon: Files, permission: 'products:view', menuPermission: 'menu:products', domain: 'products', group: 'operation', order: 1 })
  },
  {
    path: '/campaigns',
    name: 'campaigns',
    component: CampaignsView,
    meta: routeMeta('市场活动', '活动创建、模板、预算 ROI、参与者与转化归因。', { requiresAuth: true, icon: Promotion, permission: 'campaigns:view', menuPermission: 'menu:campaigns', domain: 'campaigns', group: 'customer', order: 1 })
  },
  {
    path: '/contacts',
    name: 'contacts',
    component: ContactsView,
    meta: routeMeta('联系人', '联系人档案、互动时间轴、分级和去重。', { requiresAuth: true, icon: Avatar, permission: 'contacts:view', menuPermission: 'menu:contacts', domain: 'contacts', objectType: 'contact', group: 'customer', order: 4 })
  },
  {
    path: '/companies',
    name: 'companies',
    component: ContactsView,
    meta: routeMeta('公司', '公司档案、关联联系人、开放交易和最近互动。', { requiresAuth: true, icon: OfficeBuilding, permission: 'contacts:view', menuPermission: 'menu:companies', domain: 'contacts', objectType: 'company', group: 'customer', order: 3 })
  },
  {
    path: '/leads',
    name: 'leads',
    component: LeadsView,
    meta: routeMeta('线索', '多渠道捕获、分配回收、AI 评分、培育和转化。', { requiresAuth: true, icon: Connection, permission: 'leads:view', menuPermission: 'menu:leads', domain: 'leads', objectType: 'lead', group: 'customer', order: 2 })
  },
  {
    path: '/opportunities',
    name: 'opportunities',
    component: OpportunitiesView,
    meta: routeMeta('交易商机', '可视化管道、阶段门禁、预测和风险洞察。', { requiresAuth: true, icon: Sell, permission: 'opportunities:view', menuPermission: 'menu:opportunities', domain: 'opportunities', group: 'sales', order: 1 })
  },
  {
    path: '/deals',
    name: 'deals',
    component: DealsView,
    meta: routeMeta('成交记录', '成交金额、产品明细、归属分成、合同和发票。', { requiresAuth: true, icon: Finished, permission: 'deals:view', menuPermission: 'menu:deals', domain: 'deals', group: 'sales', order: 2 })
  },
  {
    path: '/orders',
    name: 'orders',
    component: OrdersView,
    meta: routeMeta('订单', '订单审核、支付、发货、发票和售后流转。', { requiresAuth: true, icon: ShoppingCart, permission: 'orders:view', menuPermission: 'menu:orders', domain: 'orders', group: 'sales', order: 3 })
  },
  {
    path: '/tickets',
    name: 'tickets',
    component: TicketsView,
    meta: routeMeta('服务工单', '分类、分配、SLA、处理记录和满意度。', { requiresAuth: true, icon: Service, permission: 'tickets:view', menuPermission: 'menu:tickets', domain: 'tickets', group: 'service', order: 1 })
  },
  {
    path: '/dashboards',
    name: 'dashboards',
    component: DashboardsView,
    meta: routeMeta('销售大屏', '实时业绩、漏斗、目标进度和趋势分析。', { requiresAuth: true, icon: Histogram, permission: 'dashboards:view', menuPermission: 'menu:dashboards', domain: 'dashboards', group: 'workbench', order: 2 })
  },
  {
    path: '/im',
    name: 'im',
    component: IMCenterView,
    meta: routeMeta('站内消息', '员工一对一实时沟通、在线状态与未读提醒。', { requiresAuth: true, icon: MessageBox, permission: 'im:view', menuPermission: 'menu:im', domain: 'im', group: 'workbench', order: 3 })
  },
  {
    path: '/storage',
    name: 'storage',
    component: StorageView,
    meta: routeMeta('存储空间', '查看租户磁盘用量、扩容到期状态和增购续费申请。', { requiresAuth: true, icon: FolderOpened, permission: 'storage:view', group: 'admin', order: 1 })
  },
  {
    path: '/system/tenant-settings',
    name: 'system-tenant-settings',
    component: TenantSettingsView,
    meta: routeMeta('租户设置', '设置业务默认币种、时区和租户级偏好。', { requiresAuth: true, icon: OfficeBuilding, permission: 'enterprise:view', group: 'admin', order: 2, tenantRequired: true })
  },
  {
    path: '/system/iot-devices',
    name: 'system-iot-devices',
    component: IoTDevicesView,
    meta: routeMeta('IoT设备', '管理租户IoT展示设备和销售大屏同步。', { requiresAuth: true, icon: Monitor, permission: 'iot:view', group: 'admin', order: 3, tenantRequired: true })
  },
  {
    path: '/teams',
    name: 'teams',
    component: TeamsView,
    meta: routeMeta('团队管理', '成员、汇报关系、角色矩阵、销售目标和工作量。', { requiresAuth: true, icon: Management, permission: 'teams:view', menuPermission: 'menu:teams', domain: 'teams', group: 'admin', order: 1, hidden: true })
  },
  {
    path: '/accounts',
    name: 'accounts',
    component: SystemManagementView,
    meta: routeMeta('账号权限', '账号、角色编码、模块权限和管理动作口径。', { requiresAuth: true, icon: Avatar, permission: 'accounts:manage', menuPermission: 'menu:accounts', domain: 'accounts', systemSection: 'user', group: 'admin', order: 3, hidden: true })
  },
  {
    path: '/accounts/overview',
    name: 'accounts-overview',
    component: AccountsView,
    meta: routeMeta('账号权限概览', '账号权限概览与模块权限速查。', { requiresAuth: true, icon: Avatar, permission: 'accounts:manage', menuPermission: 'menu:accounts', domain: 'accounts', group: 'admin', order: 5, hidden: true })
  },
  {
    path: '/insights',
    name: 'insights',
    component: InsightsView,
    meta: routeMeta('运营热点', '热点客户、行为热力图、高频问题、内容热度和风险洞察。', { requiresAuth: true, icon: DataAnalysis, permission: 'insights:view', menuPermission: 'menu:insights', domain: 'insights', group: 'operation', order: 2 })
  },
  {
    path: '/ai-analysis',
    name: 'ai-analysis',
    component: TenantAIAnalysisView,
    meta: routeMeta('AI经营分析', 'AI 汇总客户、销售、订单、服务和团队执行，生成管理层经营报告。', { requiresAuth: true, icon: DataAnalysis, permission: 'insights:view', menuPermission: 'menu:insights', domain: 'insights', group: 'operation', order: 3 })
  },
  {
    path: '/manufacturing',
    name: 'manufacturing',
    component: ManufacturingPlaceholderView,
    meta: routeMeta('industry.manufacturing.menu.title', 'industry.manufacturing.menu.desc', { requiresAuth: true, icon: Monitor, group: 'manufacturing', order: 1, titleKey: 'industry.manufacturing.menu.title', descKey: 'industry.manufacturing.menu.desc', industryPackage: 'manufacturing', manufacturingModule: 'customers', hidden: true })
  },
  {
    path: '/manufacturing/:module',
    name: 'manufacturing-module',
    component: ManufacturingPlaceholderView,
    meta: routeMeta('industry.manufacturing.menu.title', 'industry.manufacturing.menu.desc', { requiresAuth: true, icon: Monitor, group: 'manufacturing', order: 1, titleKey: 'industry.manufacturing.menu.title', descKey: 'industry.manufacturing.menu.desc', industryPackage: 'manufacturing', hidden: true })
  },
  {
    path: '/platform/overview',
    name: 'platform-overview',
    component: PlatformOverviewView,
    meta: routeMeta('平台概览', '查看租户运营摘要和平台容量健康。', { requiresAuth: true, icon: DataAnalysis, permission: 'platform:overview', group: 'platform', order: 1 })
  },
  {
    path: '/platform/tenants',
    name: 'platform-tenants',
    component: PlatformTenantsView,
    meta: routeMeta('租户管理', '开通租户、重置租户管理员并维护租户状态。', { requiresAuth: true, icon: OfficeBuilding, permission: 'platform:tenants', group: 'platform', order: 2 })
  },
  {
    path: '/platform/settings',
    name: 'platform-settings',
    component: PlatformSettingsView,
    meta: routeMeta('平台配置中心', '平台参数草稿、发布和回滚。', { requiresAuth: true, icon: Files, permission: 'platform:settings', group: 'platform', order: 3 })
  },
  {
    path: '/platform/plans',
    name: 'platform-plans',
    component: PlatformPlansView,
    meta: routeMeta('套餐管理', '维护基础版与Plus版套餐及模块清单。', { requiresAuth: true, icon: Management, permission: 'platform:settings', group: 'platform', order: 4 })
  },
  {
    path: '/platform/ai-analysis',
    name: 'platform-ai-analysis',
    component: PlatformAIAnalysisView,
    meta: routeMeta('AI分析账务', '查看租户 AI 分析开通申请、token 消耗和钱包余额。', { requiresAuth: true, icon: DataAnalysis, permission: 'platform:settings', group: 'platform', order: 5 })
  },
  {
    path: '/platform/expiring',
    name: 'platform-expiring',
    component: PlatformExpiringView,
    meta: routeMeta('到期提醒', '查看即将到期与已到期租户列表。', { requiresAuth: true, icon: Finished, permission: 'platform:tenants', group: 'platform', order: 6 })
  },
  {
    path: '/platform/runtime',
    name: 'platform-runtime',
    component: PlatformRuntimeView,
    meta: routeMeta('运行监控', '查看系统运行状态和文件节点健康。', { requiresAuth: true, icon: DataBoard, permission: 'platform:overview', group: 'platform', order: 7 })
  },
  {
    path: '/platform/iot-firmwares',
    name: 'platform-iot-firmwares',
    component: PlatformIoTFirmwareView,
    meta: routeMeta('IoT固件管理', '上传固件并按设备SN或租户发布终端升级。', { requiresAuth: true, icon: Monitor, permission: 'platform:settings', group: 'platform', order: 8 })
  },
  {
    path: '/platform/messages',
    name: 'platform-messages',
    component: IMCenterView,
    meta: routeMeta('platform.messages.title', 'platform.messages.desc', { requiresAuth: true, icon: MessageBox, permission: 'platform:overview', group: 'platform', order: 9, titleKey: 'platform.messages.title', descKey: 'platform.messages.desc' })
  },
  {
    path: '/system/user-center',
    name: 'system-user-center',
    component: SystemManagementView,
    meta: routeMeta('个人中心', '维护个人资料、密码和登录会话。', { requiresAuth: true, icon: Avatar, systemSection: 'user-center', group: 'profile', order: 99, hidden: true })
  },
  {
    path: '/system/user',
    name: 'system-user',
    component: SystemManagementView,
    meta: routeMeta('用户管理', '维护系统用户、角色关联、部门和岗位归属。', { requiresAuth: true, icon: Avatar, permission: 'accounts:manage', menuPermission: 'menu:accounts', systemSection: 'user', group: 'admin', order: 90 })
  },
  {
    path: '/system/role',
    name: 'system-role',
    component: SystemManagementView,
    meta: routeMeta('角色管理', '维护角色编码、状态和权限基础信息。', { requiresAuth: true, icon: Management, permission: 'accounts:manage', menuPermission: 'menu:accounts', systemSection: 'role', group: 'admin', order: 4 })
  },
  {
    path: '/system/department',
    name: 'system-department',
    component: SystemManagementView,
    meta: routeMeta('部门管理', '维护组织部门树、负责人和联系信息。', { requiresAuth: true, icon: OfficeBuilding, permission: 'system:department:view', systemSection: 'department', group: 'admin', order: 92 })
  },
  {
    path: '/system/post',
    name: 'system-post',
    component: SystemManagementView,
    meta: routeMeta('岗位管理', '维护岗位名称、编码、排序和状态。', { requiresAuth: true, icon: SuitcaseLine, permission: 'system:post:view', systemSection: 'post', group: 'admin', order: 93 })
  },
  {
    path: '/system/menu',
    name: 'system-menu',
    component: SystemManagementView,
    meta: routeMeta('菜单管理', '维护系统菜单、路由和权限标识。', { requiresAuth: true, icon: DataBoard, permission: 'accounts:manage', menuPermission: 'menu:accounts', systemSection: 'menu', group: 'admin', order: 94, hidden: true })
  },
  {
    path: '/system/log',
    name: 'system-log',
    component: SystemManagementView,
    meta: routeMeta('操作日志', '查看系统操作审计、导出和清理日志。', { requiresAuth: true, icon: DataAnalysis, permission: 'audit_log:view', systemSection: 'log', group: 'admin', order: 95, tenantRequired: true })
  },
  {
    path: '/system/login-log',
    name: 'system-login-log',
    component: SystemManagementView,
    meta: routeMeta('登录日志', '查看登录、刷新令牌和设备访问记录。', { requiresAuth: true, icon: DataAnalysis, permission: 'audit_log:view', systemSection: 'login-log', group: 'admin', order: 96, tenantRequired: true })
  },
  {
    path: '/system/security-log',
    name: 'system-security-log',
    component: SystemManagementView,
    meta: routeMeta('system.security.title', 'system.security.desc', { requiresAuth: true, icon: DataAnalysis, permission: 'audit_log:view', systemSection: 'security-log', group: 'admin', order: 97, tenantRequired: true, titleKey: 'system.security.title', descKey: 'system.security.desc' })
  },
  {
    path: '/crm-suite/:domain',
    name: 'crm-suite-domain',
    component: NewPRDWorkspaceView,
    meta: routeMeta('业务模块', '', { requiresAuth: true, icon: MessageBox, permission: 'workspace:view', menuPermission: 'menu:crm_suite_domain', hidden: true })
  }
]

export const routes = [...authRoutes, ...navigationRoutes]
