#!/usr/bin/env node

import fs from 'fs'
import path from 'path'

const args = new Set(process.argv.slice(2))
const fail = args.has('--fail')
const allSrc = args.has('--all-src')
const root = path.resolve(process.cwd(), allSrc ? 'src' : 'src/views')
const textExt = new Set(['.vue', '.js', '.ts'])
const ignoreDirs = new Set(['assets', '__tests__', '__mocks__'])
const ignoreFiles = new Set(['src/i18n.js', 'src/i18n-dom.js', 'src/i18n-strict-locales.js'])
const nonHanLocales = new Set(['en-US', 'es-ES', 'fr-FR', 'de-DE', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT'])
const simplifiedSensitiveLocales = new Set(['ko-KR'])
const localeScriptRules = {
  'zh-CN': new Set(['han']),
  'zh-TW': new Set(['han']),
  'ja-JP': new Set(['han', 'kana']),
  'ko-KR': new Set(['hangul']),
  'ar-SA': new Set(['arabic']),
  'ru-RU': new Set(['cyrillic']),
  'en-US': new Set(),
  'es-ES': new Set(),
  'fr-FR': new Set(),
  'de-DE': new Set(),
  'pt-PT': new Set(),
  'it-IT': new Set()
}
const scriptPatterns = {
  han: /[\u3400-\u9FFF]/,
  kana: /[\u3040-\u30FF]/,
  hangul: /[\uAC00-\uD7AF]/,
  arabic: /[\u0600-\u06FF]/,
  cyrillic: /[\u0400-\u04FF]/
}
const placeholderPattern = /^(Missing translation|CRM interface text|Interface text|Text unavailable|Label|Etiqueta|Libellé|Bezeichnung|ラベル|레이블|تسمية|Метка|Etichetta|標籤|Falta traducción|Traduction manquante|Fehlende Übersetzung|翻訳がありません|번역 없음|Tradução em falta|الترجمة مفقودة|Отсутствует перевод|Traduzione mancante)(\s+[a-f0-9]{4})?$/i
const pendingTranslationPattern = /^__PENDING_TRANSLATION__:/i
const codeLikeFallbackPattern = /(^domain auto [a-f0-9]+$|^domain\.auto\.[a-f0-9]+$|^[a-z]+ [a-z]+ [a-f0-9]{4,}$)/i
const genericFallbackValues = new Set([
  'Details',
  'Detalles',
  'Détails',
  'Details',
  '詳細',
  '상세',
  'Detalhes',
  'التفاصيل',
  'Сведения',
  'Dettagli',
  '詳情',
  'Access settings',
  'Admin settings',
  'Lead handling',
  'Deal progress',
  'Order finance',
  'Campaign results',
  'Project progress',
  'Contact details',
  'Customer health',
  'Ticket handling',
  'Product inventory',
  'Business information',
  'Interface control',
  'Confirm action',
  'Confirm deletion',
  'Please enter the required information',
  'Please select an option',
  'No data available',
  'Operation failed',
  'Operation completed',
  'Permission and access settings',
  'Notification and task reminder',
  'Service ticket handling',
  'File and material management',
  'Order fulfillment and finance',
  'Opportunity and deal progress',
  'Lead follow-up and conversion',
  'Campaign performance and attribution',
  'Customer profile and health',
  'Project delivery progress',
  'Product and inventory management',
  'System administration',
  'Analytics and dashboard',
  'Contact and company details',
  'Help and guidance'
])
const genericFieldFallbackByLocale = {
  'en-US': new Set(['Field', 'Info', 'Object', 'Item']),
  'es-ES': new Set(['Campo', 'Elemento']),
  'fr-FR': new Set(['Champ', 'Élément', 'Element']),
  'de-DE': new Set(['Feld', 'Element']),
  'ja-JP': new Set(['フィールド', '項目']),
  'ko-KR': new Set(['필드', '항목']),
  'pt-PT': new Set(['Campo', 'Item', 'Elemento']),
  'ar-SA': new Set(['حقل', 'عنصر']),
  'ru-RU': new Set(['Поле', 'Элемент']),
  'it-IT': new Set(['Campo', 'Elemento'])
}
const forbiddenVisibleFragments = [
  'Create Details',
  'Details name',
  'Order finance Details',
  'PublicOpportunity',
  'PrivateOpportunity',
  'BatchDeleteInvalidLateLead',
  'Interface text',
  'CRM interface text',
  'UI text',
  'No Field',
  'Enter Field',
  'Select Field',
  'Field name',
  'Field is required',
  'Field management',
  'CampaignLead',
  'NotificationType',
  'MaintainCustomer',
  'AssignLead',
  'WonRevenue',
  'CampaignTime',
  'CampaignContact',
  'CampaignActivity',
  'CampaignCustomer'
]
const forbiddenCamelFragments = /\b(LeadCreate|GenerateOrderDraft|PrivateProject|PublicProject|PrivateLead|PublicLead|PrivateOrder|PublicOrder|PrivateDeal|PublicDeal|PrivateCampaign|PublicCampaign|SystemMessage|SourceCampaign|ViewLead|ViewCompany|ViewContact|ViewOpportunity|HandleOrder|HandleTicket|RelatedOrder|RelatedCompany|CurrentInventory|OpportunityAmount|OrderAmount|ProjectAmount|TargetAmount|DealAmount|LeadStatus|MenuType|DeviceType|AccountType|RoleCode|RoleID|BugStatus|AllLead|AllOrder|AllProject|AllTicket|AllStatus|AllType|EnabledAI|EnabledUser|OrderSave|ProjectSave|OpportunitySave|DealSave|SaveRule|SaveTemplate|SaveScore|SavePermissions|CreateReminder|CreateProjectReminder|CreateOpportunityReminder|ReminderTime|ReminderObject|NextReminder|NextFollow-up|AttributionRevenue|ChannelRevenue|TargetCustomer|CustomerType|SourcePlatform|PotentialCustomer|NormalUser|NormalRole|RelatedRole|PermissionRole|PermissionsRole)\b/
const leakedFallbackPattern = /^(Details|Detalles|Détails|التفاصيل|Сведения|Dettagli|詳細|상세|詳情)\s+[a-f0-9]{4}$/i
const shortUiPattern = /\.(button|action|label|title|status|type|name|placeholder|filter|column|tab|tag|menu|nav|header|empty)$/i

bootstrapBrowserGlobals()

const { i18nState, localeOptions, t, applyExternalMessages, ensureLocaleMessages } = await import('../src/i18n.js')
const translatePack = readTranslatePack()
applyExternalMessages(translatePack)

function bootstrapBrowserGlobals() {
  const storage = new Map()
  globalThis.localStorage = globalThis.localStorage || {
    getItem(key) {
      return storage.has(key) ? storage.get(key) : null
    },
    setItem(key, value) {
      storage.set(key, String(value))
    },
    removeItem(key) {
      storage.delete(key)
    },
    clear() {
      storage.clear()
    }
  }
  globalThis.navigator = globalThis.navigator || {
    language: 'en-US',
    languages: ['en-US']
  }
  globalThis.document = globalThis.document || {
    createElement() {
      return {
        content: {
          firstChild: null
        },
        innerHTML: '',
        cloneNode() {
          return this
        }
      }
    },
    createElementNS() {
      return this.createElement()
    },
    documentElement: {
      lang: '',
      dir: ''
    }
  }
  globalThis.window = globalThis.window || globalThis
}

function readTranslatePack() {
  const file = path.resolve(process.cwd(), 'src/translate.md')
  if (!fs.existsSync(file)) return null
  const markdown = fs.readFileSync(file, 'utf8')
  const match = markdown.match(/```json\s*([\s\S]*?)```/i)
  try {
    return JSON.parse(match ? match[1] : markdown)
  } catch {
    return null
  }
}

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name.startsWith('.')) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (!ignoreDirs.has(entry.name)) walk(full, files)
      continue
    }
    if (textExt.has(path.extname(entry.name))) files.push(full)
  }
  return files
}

function stripComments(content) {
  return content
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/\/\/.*$/gm, '')
    .replace(/<!--[\s\S]*?-->/g, '')
}

function collectKeys() {
  const keys = new Set()
  for (const file of walk(root)) {
    const relative = path.relative(process.cwd(), file)
    if (ignoreFiles.has(relative)) continue
    const source = stripComments(fs.readFileSync(file, 'utf8'))
    const re = /\bt\s*\(\s*['"]([^'"]+)['"]/g
    for (const match of source.matchAll(re)) keys.add(match[1])
  }
  return [...keys].sort()
}

const keys = collectKeys()
const issues = []

for (const locale of localeOptions.map((item) => item.value)) {
  await ensureLocaleMessages(locale)
  i18nState.locale = locale
  for (const key of keys) {
    const value = String(t(key) || '').trim()
    validateValue({ locale, key, value, source: 'runtime' })
  }
}

const translateMessages = translatePack?.messages || {}
for (const locale of localeOptions.map((item) => item.value)) {
  const localeMessages = translateMessages[locale]
  if (!localeMessages || typeof localeMessages !== 'object') continue
  for (const [key, rawValue] of Object.entries(localeMessages)) {
    validateValue({ locale, key, value: String(rawValue || '').trim(), source: 'translate.md' })
  }
}

console.log(`[i18n-runtime] scanned keys: ${keys.length}`)
console.log(`[i18n-runtime] locales: ${localeOptions.length}`)
console.log(`[i18n-runtime] issues: ${issues.length}`)
for (const issue of issues.slice(0, 80)) {
  console.log(`- ${issue.locale} ${issue.key}: ${issue.reason} -> ${issue.value}`)
}

if (fail && issues.length) process.exit(1)

function compactLimitFor(locale, key) {
  if (!shortUiPattern.test(key)) return 120
  if (/\.(button|action|label|status|type|column|tab|tag|menu|nav|header)$/.test(key)) return 48
  return 72
}

function validateValue({ locale, key, value, source }) {
  if (!value) {
    issues.push({ locale, key, value, reason: `${source}: empty` })
    return
  }
  if (value === key || /^domain\.auto\.[a-f0-9]+$/i.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: unresolved key` })
    return
  }
  if (codeLikeFallbackPattern.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: code-like fallback` })
    return
  }
  if (placeholderPattern.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: placeholder` })
    return
  }
  if (pendingTranslationPattern.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: pending translation` })
    return
  }
  if (genericFallbackValues.has(value)) {
    issues.push({ locale, key, value, reason: `${source}: generic fallback` })
    return
  }
  if (genericFieldFallbackByLocale[locale]?.has(value)) {
    issues.push({ locale, key, value, reason: `${source}: generic field fallback` })
    return
  }
  if (forbiddenVisibleFragments.some((item) => value.includes(item))) {
    issues.push({ locale, key, value, reason: `${source}: forbidden visible fallback` })
    return
  }
  if (locale === 'en-US' && forbiddenCamelFragments.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: concatenated fallback` })
    return
  }
  if (leakedFallbackPattern.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: leaked fallback key` })
    return
  }
  if (nonHanLocales.has(locale) && /[\u3400-\u9FFF]/.test(stripProtectedSegments(value))) {
    issues.push({ locale, key, value, reason: `${source}: contains Han text` })
    return
  }
  const wrongScript = wrongScriptForLocale(value, locale)
  if (wrongScript) {
    issues.push({ locale, key, value, reason: `${source}: contains ${wrongScript} text` })
    return
  }
  if (simplifiedSensitiveLocales.has(locale) && containsSimplifiedChineseText(value)) {
    issues.push({ locale, key, value, reason: `${source}: contains simplified Chinese wording` })
    return
  }
  if (/\\\\\\\\\\\\\\\\/.test(value)) {
    issues.push({ locale, key, value, reason: `${source}: escaped newline pollution` })
    return
  }
  if (shortUiPattern.test(key) && value.length > compactLimitFor(locale, key)) {
    issues.push({ locale, key, value, reason: `${source}: too long for compact UI` })
  }
}

function stripProtectedSegments(value = '') {
  return String(value)
    .replace(/https?:\/\/\S+/g, '')
    .replace(/\$\{[^}]*\}/g, '')
    .replace(/\{[^}]*\}/g, '')
    .replace(/\bR_[A-Z0-9_]+\b/g, '')
    .replace(/\b[A-Z]{2,}\b/g, '')
    .replace(/\b[A-Za-z]+(?:\.[A-Za-z0-9]+)+\b/g, '')
}

function wrongScriptForLocale(value, locale) {
  const allowed = localeScriptRules[locale] || new Set()
  const stripped = stripProtectedSegments(value)
  for (const [script, pattern] of Object.entries(scriptPatterns)) {
    if (!allowed.has(script) && pattern.test(stripped)) return script
  }
  return ''
}

function containsSimplifiedChineseText(value = '') {
  return /[们义乌乐习乡书买乱争于亏云亚产亩亲亵亿仅从仑仓仪们价众优伙会伞伟传伤伦伪体余佣佥侠侣侥侦侧侨侩侪侬俣俦俨俩俪俭债倾偬偻偾偿傥傧储傩儿兑兖兰关兴养兽冁内冈册写军农冯冲决况冻净凄准凉减凑凛几凤凫凭凯击凿刍划刘则刚创删别刬刭刹剂剀剐剑剥剧劝办务劢动励劲劳势勋匀匦匮区医华协单卖卢卤卧卫却卺厂厅历厉压厌厍厕厢厣厦厨厩厮县参叆叇双发变叙叠叶号叹叽吁后吓吕吗吣吨听启吴呒呓呕呖呗员呙呛呜咏咙咛咝咤响哑哒哓哔哕哗哙哜哝哟唛唠唡唢唤啧啬啭啮啰啴啸喷喽喾嗫嗳嘘嘤嘱噜嚣团园囱围囵国图圆圣圹场坂坏块坚坛坝坞坟坠垄垅垆垒垦垩垫垭垯垱垲垴埘埙埚埯堑堕墙壮声壳壶处备复够头夸夹夺奁奂奋奖奥妆妇妈妩妪妫姗姜娄娅娆娇娈娱娲娴婳婴婵婶媪嫒嫔嫱嬷孙学孪宁宝实宠审宪宫宽宾寝对寻导寿将尔尘尝尧尴尸尽层屃屉届属屡屦屿岁岂岖岗岘岙岚岛岭岳岽岿峃峡峣峤峥峦崂崃崄崭嵘嵚嵝巅巩巯币帅师帐帘帜带帧帮帱帻帼幂并广庄庆庐庑库应庙庞废庼廪开异弃张弥弪弯弹强归当录彟彦彻径徕忆忏忧忾怀态怂怃怄怅怆怜总怼怿恋恒恳恶恸恹恺恻恼恽悦悬悭悮悯惊惧惨惩惫惬惭惮惯愠愤愦愿慑懑懒懔戆戋戏戗战戬户扎扑扦执扩扪扫扬扰抚抛抟抠抡抢护报担拟拢拣拥拦拧拨择挂挚挛挜挝挞挟挠挡挢挣挤挥挦捞损捡换捣据捻掳掴掷掸掺揽揿搀搁搂搅携摄摅摆摇摈摊撄撑撵撷撸撺擞攒敌敛数斋斓斗斩断无旧时旷旸昙昼显晋晒晓晔晕晖暂术朴机杀杂权条来杨杩杰极构枞枢枣枥枧枨枪枫枭柜柠柽栀栅标栈栉栊栋栌栎栏树栖样栾桊桠桡桢档桤桥桦桧桨桩梦梼梾检棂椁椟椠椤椭楼榄榅榇榈榉槚槛槟槠横樯樱橥橱橹橼檐檩欢欤欧歼殁殇残殒殓殚殡殴毁毂毕毙毡毵氇气氢氩氲汇汉汤汹沟没沣沤沥沦沧沪泞泪泶泷泸泺泻泼泽泾洁洒洼浃浅浆浇浈浉浊测浍济浏浐浑浒浓浔涂涛涝涞涟涠涡涢涣涤润涧涨涩渊渌渍渎渐渑渔渖渗温湾湿溃溅溆溇滗滚滞滟滠满滢滤滥滦滨滩滪漤潆潇潋潍潜潴澜濑濒灏灭灯灵灾灿炀炉炖炜炝点炼炽烁烂烃烛烟烦烧烨烩烫烬热焕焖焘煴爱爷牍牦牵牺犊状犷犸犹狈狝狞独狭狮狯狰狱狲猃猎猕猡猪猫猬献獭玑玙玛玮环现玱玺珉珏珐珑珰珲琎琏琐琼瑶瑷璇璎瓒瓯电画畅畴疖疗疟疠疡疬疮疯疱疴痈痉痒痖痨痪痫瘅瘆瘗瘘瘪瘫瘾瘿癞癣癫皑皱皲盏盐监盖盗盘眍眦眬着睁睐睑瞒瞩矫矶矾矿砀码砖砗砚砜砺砻砾础硁硕硖硗硙确硷碍碛碜碱礼祎祢祯祷祸禀禄禅离秃秆种积称秽秾稆税稳穑穷窃窍窑窜窝窥窦窭竖竞笃笋笔笕笺笼笾筚筛筜筝筹签简箓箦箧箨箩箪箫篑篓篮篱簖籁籴类籼粜粝粤粪粮糁糇紧絷纟纠纡红纣纤纥约级纨纩纪纫纬纭纯纰纱纲纳纵纶纷纸纹纺纽纾线绀绁绂练组绅细织终绉绊绋绌绍绎经绑绒结绔绕绖绘给绚绛络绝绞统绠绡绢绣绤绥绦继绩绪绫续绮绯绱绲绳维绵绶绷绸绹绺绻综绽绿缀缁缂缃缄缅缆缇缈缉缋缌缎缏缑缒缓缔缕编缗缘缙缚缛缜缝缟缠缡缢缣缤缥缦缧缨缩缪缫缬缭缮缯缰缱缲缳缴罂网罗罚罢罴羁羟羡翘耢耧耸耻聂聋职聍联聩聪肃肠肤肷肾肿胀胁胆胜胧胨胪胫胶脉脍脏脐脑脓脔脚脱脶脸腊腌腘腭腻腼腽腾膑臜舆舣舰舱舻艰艳艺节芈芗芜芦苁苇苈苋苌苍苎苏苧苹范茎茏茑茔茕茧荆荐荙荚荛荜荞荟荠荡荣荤荥荦荧荨荩荪荫荬荭荮药莅莜莱莲莳莴莶获莸莹莺莼萚萝萤营萦萧萨葱蒇蒉蒋蒌蓝蓟蓠蓣蓥蓦蔷蔹蔺蔼蕲蕴薮藓虏虑虚虫虬虮虽虾虿蚀蚁蚂蚕蚝蚬蛊蛎蛏蛮蛰蛱蛲蛳蛴蜕蜗蜡蝇蝈蝉蝎蝼蝾螀螨蟏衅衔补衬衮袄袅袆袜袭装裆裈裤裢裣褛褴襁襕见观规觅视觇览觉觊觋觌觎觏觐觑觞触觯誉誊讠计订讣认讥讦讧讨让讪讫训议讯记讲讳讴讵讶讷许讹论讼讽设访诀证诂诃评诅识诈诉诊诋诌词诎诏诐译诒诓诔试诖诗诘诙诚诛诜话诞诟诠诡询诣诤该详诧诨诩诫诬语诮误诰诱诲诳说诵诶请诸诹诺读诽课诿谀谁调谄谅谆谈谊谋谌谍谎谏谐谑谒谓谔谕谖谗谘谙谚谛谜谝谞谟谠谡谢谣谤谥谦谧谨谩谪谫谬谭谮谯谰谱谲谳谴谵贝贞负贡财责贤败账货质贩贪贫贬购贮贯贰贱贲贳贴贵贷贸费贺贻贼贽贾贿赀赁赂赃资赅赆赇赈赉赊赋赌赎赏赐赔赖赘赚赛赞赠赡赢赣赵赶趋趱跃跄跞践跶跷跸跹跻踊踌踪踬踯蹑蹒蹰蹿躏躜躯车轧轨轩轪轫转轭轮软轰轱轲轳轴轵轶轷轸轹轺轻轼载轾轿辀辁辂较辄辅辆辇辈辉辊辋辌辍辎辏辐辑输辔辕辖辗辘辙辚辞辩辫边辽达迁过迈运还这进远违连迟迩迳迹适选逊递逻遗遥邓邝邬邮邹邺邻郏郐郑郓郦郧郸酝酦酱酽酾酿释里鉴銮錾钆钇针钉钊钋钌钍钎钏钐钒钓钔钕钗钙钚钛钜钝钞钟钠钢钣钤钥钦钧钨钩钪钫钬钭钮钯钰钱钲钳钴钵钹钺钻钼钽钾钿铀铁铂铃铄铅铆铈铉铊铋铌铍铎铐铑铒铕铗铙铚铛铜铝铞铟铠铡铢铣铤铥铦铧铨铩铪铫铭铮铯铰铱铲铳铴铵银铷铸铹铺铻铼铽链铿销锁锂锃锄锅锆锇锈锉锋锌锍锎锐锑锒锓锔锕锖锗错锚锛锜锞锟锡锢锣锤锥锦锧锨锩锫锬锭键锯锰锱锲锴锵锶锷锸锹锺锻锼锾锿镀镁镂镃镄镅镆镇镈镉镊镌镍镎镏镐镑镒镓镔镕镖镗镘镙镚镛镜镝镞镟镠镡镢镣镤镥镦镧镨镩镪镫镬镭镯镰镱镲镳镴长门闩闪闫闭问闯闰闲间闵闷闸闹闺闻闼闽闾阀阁阂阃阄阅阆阈阉阊阋阌阍阎阏阐阑阒阔阕阖阗阘阙阚队阳阴阵阶际陆陇陈陉陕陧陨险随隐隶隽难雏雠雳雾霁霉霭靓静靥鞑鞒鞯韦韧韩韪韫韬韵页顶顷项顺须顼顽顾顿颀颁颂预颅领颇颈颉颊颌颍颎颏颐频颓颔颖颗题颚额颞颟颠颡颢颤颥颦颧风飐飑飒飓飕飖飗飘飙飚飞飨餍饣饥饧饨饩饪饫饬饭饮饯饰饱饲饳饴饵饶饷饸饹饺饻饼饽饿馁馂馃馄馅馆馈馊馋馌馍馎馏馐馑馒馓馔馕马驭驮驯驰驱驳驴驵驶驷驸驹驻驼驽驾驿骀骁骂骄骅骆骇骈骉骊骋验骏骐骑骒骓骖骗骘骚骛骜骝骞骟骠骡骢骣骤骥骧髅髋髌鬓魇魉鱼鱽鱾鱿鲀鲁鲂鲅鲆鲇鲈鲋鲍鲎鲐鲑鲒鲔鲕鲚鲛鲜鲞鲟鲠鲡鲢鲣鲤鲥鲦鲧鲨鲩鲫鲭鲮鲰鲱鲲鲳鲴鲵鲶鲷鲸鲹鲺鲻鲼鲽鲾鳀鳃鳄鳅鳇鳊鳋鳌鳍鳎鳏鳐鳑鳒鳓鳔鳕鳖鳗鳘鳙鳜鳝鳞鳟鳢鸟鸠鸡鸢鸣鸥鸦鸧鸨鸩鸪鸫鸬鸭鸯鸱鸲鸳鸵鸶鸷鸸鸹鸺鸽鸾鸿鹁鹂鹃鹄鹅鹆鹇鹈鹉鹊鹋鹌鹍鹏鹐鹑鹒鹓鹔鹕鹖鹗鹘鹚鹛鹜鹞鹟鹠鹡鹢鹣鹤鹥鹦鹧鹨鹩鹪鹫鹬鹭鹯鹰鹱鹲鹳鹴鹾麦麸黄黉黡黩黪黾鼋鼍鼗鼹齄齐齑齿龀龁龂龄龅龆龇龈龉龊龋龌龙龚龛龟]/.test(String(value || ''))
}
