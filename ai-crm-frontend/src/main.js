import { createApp, defineAsyncComponent } from 'vue'
import { ElAlert } from 'element-plus/es/components/alert/index'
import { ElButton } from 'element-plus/es/components/button/index'
import { ElCard } from 'element-plus/es/components/card/index'
import { ElConfigProvider } from 'element-plus/es/components/config-provider/index'
import { ElDropdown, ElDropdownItem, ElDropdownMenu } from 'element-plus/es/components/dropdown/index'
import { ElForm, ElFormItem } from 'element-plus/es/components/form/index'
import { ElInput } from 'element-plus/es/components/input/index'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
import en from 'element-plus/es/locale/lang/en'
import es from 'element-plus/es/locale/lang/es'
import fr from 'element-plus/es/locale/lang/fr'
import de from 'element-plus/es/locale/lang/de'
import ja from 'element-plus/es/locale/lang/ja'
import ko from 'element-plus/es/locale/lang/ko'
import pt from 'element-plus/es/locale/lang/pt'
import ar from 'element-plus/es/locale/lang/ar'
import ru from 'element-plus/es/locale/lang/ru'
import it from 'element-plus/es/locale/lang/it'
import {
  Avatar,
  Connection,
  DataAnalysis,
  DataBoard,
  Files,
  Finished,
  FolderOpened,
  Histogram,
  Management,
  MessageBox,
  OfficeBuilding,
  Promotion,
  Sell,
  Service,
  ShoppingCart,
  SuitcaseLine
} from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import { ensureLocaleMessages, i18nState } from './i18n'
import { installDeleteClickGuard } from './utils/confirm'
import './style.css'
import './system-dialog-fixes.css'

const app = createApp(App)
installDeleteClickGuard()

const elementComponents = [
  ElAlert,
  ElButton,
  ElCard,
  ElConfigProvider,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElForm,
  ElFormItem,
  ElInput
]

elementComponents.forEach((component) => {
  app.use(component)
})

Object.entries({
  ElAvatar: () => import('element-plus/es/components/avatar/index').then((module) => module.ElAvatar),
  ElBadge: () => import('element-plus/es/components/badge/index').then((module) => module.ElBadge),
  ElBreadcrumb: () => import('element-plus/es/components/breadcrumb/index').then((module) => module.ElBreadcrumb),
  ElBreadcrumbItem: () => import('element-plus/es/components/breadcrumb/index').then((module) => module.ElBreadcrumbItem),
  ElCheckbox: () => import('element-plus/es/components/checkbox/index').then((module) => module.ElCheckbox),
  ElCheckboxGroup: () => import('element-plus/es/components/checkbox/index').then((module) => module.ElCheckboxGroup),
  ElCol: () => import('element-plus/es/components/col/index').then((module) => module.ElCol),
  ElCollapse: () => import('element-plus/es/components/collapse/index').then((module) => module.ElCollapse),
  ElCollapseItem: () => import('element-plus/es/components/collapse/index').then((module) => module.ElCollapseItem),
  ElDatePicker: () => import('element-plus/es/components/date-picker/index').then((module) => module.ElDatePicker),
  ElDialog: () => import('element-plus/es/components/dialog/index').then((module) => module.ElDialog),
  ElDivider: () => import('element-plus/es/components/divider/index').then((module) => module.ElDivider),
  ElDrawer: () => import('element-plus/es/components/drawer/index').then((module) => module.ElDrawer),
  ElEmpty: () => import('element-plus/es/components/empty/index').then((module) => module.ElEmpty),
  ElIcon: () => import('element-plus/es/components/icon/index').then((module) => module.ElIcon),
  ElInputNumber: () => import('element-plus/es/components/input-number/index').then((module) => module.ElInputNumber),
  ElMenu: () => import('element-plus/es/components/menu/index').then((module) => module.ElMenu),
  ElMenuItem: () => import('element-plus/es/components/menu/index').then((module) => module.ElMenuItem),
  ElOption: () => import('element-plus/es/components/select/index').then((module) => module.ElOption),
  ElOptionGroup: () => import('element-plus/es/components/select/index').then((module) => module.ElOptionGroup),
  ElPagination: () => import('element-plus/es/components/pagination/index').then((module) => module.ElPagination),
  ElPopover: () => import('element-plus/es/components/popover/index').then((module) => module.ElPopover),
  ElProgress: () => import('element-plus/es/components/progress/index').then((module) => module.ElProgress),
  ElRadioButton: () => import('element-plus/es/components/radio/index').then((module) => module.ElRadioButton),
  ElRadioGroup: () => import('element-plus/es/components/radio/index').then((module) => module.ElRadioGroup),
  ElRow: () => import('element-plus/es/components/row/index').then((module) => module.ElRow),
  ElSegmented: () => import('element-plus/es/components/segmented/index').then((module) => module.ElSegmented),
  ElSelect: () => import('element-plus/es/components/select/index').then((module) => module.ElSelect),
  ElSkeleton: () => import('element-plus/es/components/skeleton/index').then((module) => module.ElSkeleton),
  ElSlider: () => import('element-plus/es/components/slider/index').then((module) => module.ElSlider),
  ElSpace: () => import('element-plus/es/components/space/index').then((module) => module.ElSpace),
  ElSwitch: () => import('element-plus/es/components/switch/index').then((module) => module.ElSwitch),
  ElTabPane: () => import('element-plus/es/components/tabs/index').then((module) => module.ElTabPane),
  ElTable: () => import('element-plus/es/components/table/index').then((module) => module.ElTable),
  ElTableColumn: () => import('element-plus/es/components/table/index').then((module) => module.ElTableColumn),
  ElTabs: () => import('element-plus/es/components/tabs/index').then((module) => module.ElTabs),
  ElTag: () => import('element-plus/es/components/tag/index').then((module) => module.ElTag),
  ElTimeline: () => import('element-plus/es/components/timeline/index').then((module) => module.ElTimeline),
  ElTimelineItem: () => import('element-plus/es/components/timeline/index').then((module) => module.ElTimelineItem),
  ElTooltip: () => import('element-plus/es/components/tooltip/index').then((module) => module.ElTooltip),
  ElTreeSelect: () => import('element-plus/es/components/tree-select/index').then((module) => module.ElTreeSelect),
  ElUpload: () => import('element-plus/es/components/upload/index').then((module) => module.ElUpload)
}).forEach(([name, loader]) => {
  app.component(name, defineAsyncComponent(loader))
})

Object.entries({
  Avatar,
  Connection,
  DataAnalysis,
  DataBoard,
  Files,
  Finished,
  FolderOpened,
  Histogram,
  Management,
  MessageBox,
  OfficeBuilding,
  Promotion,
  Sell,
  Service,
  ShoppingCart,
  SuitcaseLine
}).forEach(([name, component]) => {
  app.component(name, component)
})

app.provide('elementLocales', {
  'zh-CN': zhCn,
  'zh-TW': zhTw,
  'en-US': en,
  'es-ES': es,
  'fr-FR': fr,
  'de-DE': de,
  'ja-JP': ja,
  'ko-KR': ko,
  'pt-PT': pt,
  'ar-SA': ar,
  'ru-RU': ru,
  'it-IT': it
})

const localeReady = i18nState.locale === 'zh-CN' ? Promise.resolve() : ensureLocaleMessages()

localeReady.finally(() => {
  app.use(router).mount('#app')
})
