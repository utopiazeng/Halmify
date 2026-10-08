<template>
  <div class="system-page campaign-page">
    <section class="system-card campaign-workbench-hero">
      <div>
        <span class="section-eyebrow">{{ t('domain.auto.64c3207c46') }}</span>
        <h2>{{ t('domain.auto.c0d4ab58c1') }}</h2>
        <p>{{ t('domain.auto.47cbe5500d') }}</p>
      </div>
      <div class="campaign-hero-visual" aria-hidden="true">
        <svg viewBox="0 0 280 150" role="img">
          <defs>
            <linearGradient id="campaignLeadGradient" x1="0" x2="1" y1="0" y2="1">
              <stop offset="0" stop-color="#dbeafe" />
              <stop offset="1" stop-color="#dcfce7" />
            </linearGradient>
          </defs>
          <rect x="16" y="18" width="248" height="112" rx="18" fill="url(#campaignLeadGradient)" />
          <path d="M52 84 C86 36, 128 38, 154 76 S210 112, 236 54" fill="none" stroke="#2563eb" stroke-width="6" stroke-linecap="round" />
          <circle cx="54" cy="84" r="13" fill="#2563eb" />
          <circle cx="154" cy="76" r="13" fill="#7c3aed" />
          <circle cx="236" cy="54" r="13" fill="#16a34a" />
          <rect x="42" y="104" width="54" height="8" rx="4" fill="#93c5fd" />
          <rect x="118" y="102" width="70" height="8" rx="4" fill="#c4b5fd" />
          <rect x="200" y="80" width="38" height="8" rx="4" fill="#86efac" />
        </svg>
      </div>
    </section>

    <section class="campaign-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="campaign-operations-grid">
      <div class="system-card campaign-list-card">
        <div class="list-card-search">
          <el-form :model="filters" label-width="72px" class="system-search-form">
            <el-form-item :label="t('domain.auto.cc1b21e800')">
              <el-input v-model="filters.keyword" :placeholder="t('domain.auto.c4067f055b')" clearable />
            </el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')">
              <el-select v-model="filters.status" :placeholder="t('domain.auto.dba277df58')" clearable>
                <el-option :label="t('domain.auto.0f436818c0')" value="draft" />
                <el-option :label="t('domain.auto.6f1b157a55')" value="scheduled" />
                <el-option :label="t('domain.auto.6f1972e48e')" value="running" />
                <el-option :label="t('domain.auto.20391328c8')" value="finished" />
              </el-select>
            </el-form-item>
            <div class="system-search-actions">
              <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
              <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
            </div>
          </el-form>
        </div>
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.7031d1096d') }}</el-button>
            <el-button @click="goCreateLeadFromCampaign(selectedCampaign)">{{ t('domain.auto.ba7a55fbaa') }}</el-button>
            <el-button :disabled="!selectedCampaign" @click="openDetailDrawer(selectedCampaign)">{{ t('domain.auto.faea8c1db9') }}</el-button>
          </div>
          <div class="toolbar-icons">
            <el-tooltip :content="t('domain.auto.38108eaa1d')">
              <button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="refreshAll">
                <Icon icon="ri:refresh-line" />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('domain.auto.56292a48ef')">
              <button class="system-icon-btn" :aria-label="t('domain.auto.56292a48ef')" @click="columnDrawer = true">
                <Icon icon="ri:settings-3-line" />
              </button>
            </el-tooltip>
          </div>
        </div>

        <el-table
          :data="rows"
          class="system-table campaign-table"
          row-key="id"
          v-loading="loading"
          highlight-current-row
          :current-row-key="selectedCampaign?.id"
          :row-class-name="campaignRowClassName"
          @row-click="selectCampaign"
        >
          <el-table-column v-if="visibleColumn('name')" prop="name" :label="t('domain.auto.b2548636f0')" min-width="220">
            <template #default="{ row }">
              <div class="record-title">
                <strong>{{ row.name }} <el-tag v-if="objectVisibility(row) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
                <span>{{ campaignTypeLabel(row.object_type, parsePayload(row.payload_json).type) }} · {{ sourceLabel(sourceForRow(row)) }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('status')" :label="t('domain.auto.62e951a692')" width="110">
            <template #default="{ row }"><el-tag :type="statusType(row.status)" effect="light">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('time')" :label="t('domain.auto.abe0ecdb04')" min-width="170">
            <template #default="{ row }">{{ campaignTimeRange(row) }}</template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('progress')" :label="t('domain.auto.32e920a5df')" min-width="150">
            <template #default="{ row }">
              <el-progress :percentage="campaignTimeProgress(row).percent" :status="campaignTimeProgress(row).status" />
              <span class="table-subtext">{{ campaignTimeProgress(row).label }}</span>
            </template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('leads')" :label="t('domain.auto.8de0b8fb1d')" width="90">
            <template #default="{ row }">{{ metricFor(row).lead_count || 0 }}</template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('opportunities')" :label="t('domain.auto.24aef3c29e')" width="110">
            <template #default="{ row }">{{ metricFor(row).opportunity_count || 0 }}</template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('revenue')" :label="t('domain.auto.02720419d3')" width="150">
            <template #default="{ row }">{{ formatAmount(metricFor(row).revenue_amount || 0, row.currency) }}</template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('owner')" :label="t('domain.auto.974d383f36')" width="140">
            <template #default="{ row }">{{ ownerName(row.owner_id) }}</template>
          </el-table-column>
          <el-table-column v-if="visibleColumn('next')" :label="t('domain.auto.ea0ef2ae72')" min-width="150">
            <template #default="{ row }">
              <el-button size="small" plain @click.stop="runRowNextAction(row)">{{ nextActionLabel(row) }}</el-button>
            </template>
          </el-table-column>
          <el-table-column :label="t('campaign.actions')" width="260" fixed="right">
            <template #default="{ row }">
              <div class="campaign-row-actions" @click.stop>
                <el-button size="small" type="primary" plain @click="openDetailDrawer(row)">{{ t('domain.auto.faea8c1db9') }}</el-button>
                <el-button size="small" plain @click="openChainDrawer(row)">{{ t('domain.auto.f3a298ce49') }}</el-button>
                <el-button v-if="canEditRecord(row)" size="small" plain @click="openEdit(row)">{{ t('domain.auto.a7f814c0a4') }}</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div class="system-pagination">
          <span>{{ t('campaign.totalPrefix') }} {{ pagination.total }} {{ t('campaign.totalSuffix') }}</span>
          <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
        </div>
      </div>

    </section>

    <el-drawer v-model="chainDrawer" :title="t('domain.auto.95757e6adc')" size="720px">
      <div ref="chainCardRef" class="system-card campaign-chain-card" :class="{ 'campaign-chain-highlight': chainHighlight }">
        <div class="chart-panel-head">
          <div>
            <strong>{{ t('domain.auto.87d6a70c91') }}</strong>
            <span>{{ selectedCampaign?.name || t('domain.auto.495ff59ce0') }}</span>
          </div>
          <el-tag size="small">{{ sourceLabel(selectedCampaignSource) }}</el-tag>
        </div>
        <div v-if="selectedCampaign" class="campaign-chain-flow">
          <button v-for="item in chainRows" :key="item.key" class="campaign-chain-node" @click="openChainNode(item.key)">
            <span>{{ item.label }}</span>
            <strong>{{ item.count }}</strong>
            <em>{{ formatAmount(item.amount, selectedCampaign.currency) }}</em>
          </button>
        </div>
        <el-empty v-else :description="t('domain.auto.e3e3c1a0e7')" />
        <div class="campaign-chain-actions">
          <el-button size="small" type="primary" plain :disabled="!selectedCampaign" @click="goCreateLeadFromCampaign(selectedCampaign)">{{ t('domain.auto.ba7a55fbaa') }}</el-button>
          <el-button size="small" plain :disabled="!selectedCampaign" @click="openRelatedDrawer('opportunities')">{{ t('domain.auto.d23ef6ed4e') }}</el-button>
          <el-button v-if="canEditRecord(selectedCampaign)" size="small" plain :disabled="!selectedCampaign" @click="generateAttribution(selectedCampaign)">{{ t('domain.auto.63e37cbb0d') }}</el-button>
          <el-button size="small" plain :disabled="!selectedCampaign" @click="advancedDrawer = true">{{ t('domain.auto.e85f465b0c') }}</el-button>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="detailDrawer" :title="t('domain.auto.1970283ed1')" size="860px">
      <div class="system-card campaign-detail-card">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.3a12ff3c37') }}</strong>
          <el-tag>{{ selectedCampaign?.name || t('domain.auto.495ff59ce0') }}</el-tag>
        </div>
        <div class="campaign-guide">
          <strong>{{ t('domain.auto.74b4902b59') }}</strong>
          <span>{{ t('domain.auto.598e5e510a') }}</span>
        </div>
        <el-tabs v-model="activeResultTab" class="campaign-result-tabs">
          <el-tab-pane :label="t('domain.auto.8de0b8fb1d')" name="leads">
            <div class="campaign-related-list">
              <button v-for="item in relatedLeads" :key="item.id" class="campaign-related-item" @click="openLead(item)">
                <strong>{{ item.name }}</strong>
                <span>{{ statusLabel(item.status) }} · {{ leadContactText(item) }}</span>
                <em>{{ t('domain.auto.f7acefd2d4') }}</em>
              </button>
              <el-empty v-if="!relatedLeads.length" :description="t('domain.auto.3adfd5e1e9')" />
            </div>
          </el-tab-pane>
          <el-tab-pane :label="t('domain.auto.c4beee8569')" name="opportunities">
            <div class="campaign-related-list">
              <button v-for="item in relatedOpportunities" :key="item.id" class="campaign-related-item" @click="goObject('opportunities', item)">
                <strong>{{ item.name }}</strong>
                <span>{{ statusLabel(item.status) }} · {{ ownerName(item.owner_id) }}</span>
                <em>{{ formatAmount(item.amount, item.currency) }}</em>
              </button>
              <el-empty v-if="!relatedOpportunities.length" :description="t('domain.auto.23e12e977b')" />
            </div>
          </el-tab-pane>
          <el-tab-pane :label="t('domain.auto.4483df6e28')" name="deals">
            <div class="campaign-related-list">
              <button v-for="item in relatedDeals" :key="item.id" class="campaign-related-item" @click="goObject('deals', item)">
                <strong>{{ item.name }}</strong>
                <span>{{ statusLabel(item.status) }} · {{ ownerName(item.owner_id) }}</span>
                <em>{{ formatAmount(item.amount, item.currency) }}</em>
              </button>
              <el-empty v-if="!relatedDeals.length" :description="t('domain.auto.2b829504f6')" />
            </div>
          </el-tab-pane>
          <el-tab-pane :label="t('domain.auto.8ef538db71')" name="orders">
            <div class="campaign-related-list">
              <button v-for="item in relatedOrders" :key="item.id" class="campaign-related-item" @click="goObject('orders', item)">
                <strong>{{ item.name }}</strong>
                <span>{{ statusLabel(item.status) }} {{ t('domain.auto.7def2c55b6') }} {{ formatAmount(parsePayload(item.payload_json).paid_amount || 0, item.currency) }}</span>
                <em>{{ formatAmount(item.amount, item.currency) }}</em>
              </button>
              <el-empty v-if="!relatedOrders.length" :description="t('domain.auto.410c13a01f')" />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>

      <div class="system-card campaign-detail-card campaign-drawer-section">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.1970283ed1') }}</strong>
          <div class="campaign-card-actions">
            <el-button v-if="canFollowRecord(selectedCampaign)" size="small" plain :disabled="!selectedCampaign" @click="openFollowUp">{{ t('domain.auto.da835b347c') }}</el-button>
            <el-button v-if="canEditRecord(selectedCampaign)" size="small" plain :disabled="!selectedCampaign" @click="openEdit(selectedCampaign)">{{ t('domain.auto.7719bdd192') }}</el-button>
          </div>
        </div>
        <div v-if="selectedCampaign" class="lead-summary">
          <div><span>{{ t('domain.auto.b2548636f0') }}</span><strong>{{ selectedCampaign.name }}</strong></div>
          <div><span>{{ t('domain.auto.62e951a692') }}</span><strong>{{ statusLabel(selectedCampaign.status) }}</strong></div>
          <div><span>{{ t('domain.auto.e4e46c7235') }}</span><strong>{{ campaignTypeLabel(selectedCampaign.object_type, selectedCampaignPayload.type) }}</strong></div>
          <div><span>{{ t('domain.auto.c63f79e636') }}</span><strong>{{ sourceLabel(selectedCampaignSource) }}</strong></div>
          <div><span>{{ t('domain.auto.e8868af6eb') }}</span><strong>{{ formatDate(selectedCampaign.start_at) }}</strong></div>
          <div><span>{{ t('domain.auto.a0bb9f49ab') }}</span><strong>{{ formatDate(selectedCampaign.end_at) }}</strong></div>
          <div><span>{{ t('domain.auto.0dcf0e012a') }}</span><strong>{{ formatAmount(campaignBudget(selectedCampaign), selectedCampaign.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(selectedCampaign.owner_id) }}</strong></div>
          <div><span>CPL</span><strong>{{ formatAmount(performanceSummary.cpl || 0, selectedCampaign.currency) }}</strong></div>
          <div><span>ROI</span><strong>{{ percent(performanceSummary.roi || 0) }}</strong></div>
        </div>
        <ObjectAccessPanel
          v-if="selectedCampaign"
          domain="campaigns"
          :record="selectedCampaign"
          :users="users"
          :can-manage="canManageRecordACL(selectedCampaign)"
          :object-label="t('domain.auto.5fa95b1b73')"
          @updated="refreshSelectedCampaign"
        />
        <el-empty v-else :description="t('campaign.selectCampaign')" />
      </div>

      <div class="system-card campaign-detail-card campaign-drawer-section">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.c625b7c816') }}</strong>
          <el-button v-if="canFollowRecord(selectedCampaign)" size="small" type="primary" plain :disabled="!selectedCampaign" @click="openFollowUp">{{ t('domain.auto.da835b347c') }}</el-button>
        </div>
        <div v-if="campaignTimeline.length" class="campaign-timeline-list">
          <div v-for="item in campaignTimeline" :key="item.id" class="campaign-timeline-item">
            <strong>{{ timelineTitle(item) }}</strong>
            <small class="timeline-operator">{{ timelineOperator(item) }}</small>
            <span>{{ timelineContent(item) }}</span>
            <em>{{ formatTime(item.created_at) }}</em>
          </div>
        </div>
        <el-empty v-else :description="t('domain.auto.983795ea28')" />
      </div>

      <div class="system-card campaign-detail-card campaign-drawer-section">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.4ea4d4727d') }}</strong>
          <el-upload v-if="canFollowRecord(selectedCampaign) && selectedCampaign" :show-file-list="false" :http-request="uploadCampaignFile" :disabled="fileUploading">
            <el-button size="small" plain :loading="fileUploading">{{ t('domain.auto.287d6fd3a9') }}</el-button>
          </el-upload>
        </div>
        <div v-if="campaignFiles.length" class="campaign-file-list">
          <div v-for="item in campaignFiles" :key="item.id" class="campaign-file-item">
            <div>
              <strong>{{ filePayload(item).file_name || item.name }}</strong>
              <span>{{ fileCategoryLabel(filePayload(item).category) }} · {{ fileSizeText(filePayload(item).file_size) }} · {{ formatTime(filePayload(item).uploaded_at || item.created_at) }}</span>
            </div>
            <div class="campaign-file-actions">
              <el-button link type="primary" @click="downloadCampaignFile(item)">{{ t('domain.auto.d668db53bc') }}</el-button>
              <el-button v-if="canEditRecord(selectedCampaign)" link type="danger" @click="deleteCampaignFile(item)">{{ t('domain.auto.3755f56f2f') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else :description="t('domain.auto.22c998afca')" />
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.623e38c094') : t('domain.auto.7031d1096d')" width="960px" class="campaign-form-dialog" destroy-on-close>
      <el-form :model="form" label-position="top" class="lead-dialog-form campaign-dialog-form">
        <el-tabs v-model="activeFormTab">
          <el-tab-pane :label="t('domain.auto.41654e0268')" name="base">
            <div class="campaign-guide wide-field">
              <strong>{{ t('domain.auto.abb44a0cef') }}</strong>
              <span>{{ t('domain.auto.7ba17b3ca9') }}</span>
            </div>
            <el-form-item :label="t('domain.auto.2b020286d1')" required>
              <el-input v-model="form.name" :placeholder="t('domain.auto.88f2ffb655')" />
            </el-form-item>
            <el-form-item :label="t('domain.auto.0bd77b74e3')">
              <el-select v-model="form.payload.type" class="full-input" @change="applyCampaignTemplate">
                <el-option v-for="item in campaignTypeOptions" :key="item.value" :label="item.label" :value="item.value">
                  <span>{{ item.label }}</span>
                  <span class="product-option-meta">{{ item.desc }}</span>
                </el-option>
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')">
              <el-select v-model="form.status" class="full-input">
                <el-option :label="t('domain.auto.0f436818c0')" value="draft" />
                <el-option :label="t('domain.auto.6f1b157a55')" value="scheduled" />
                <el-option :label="t('domain.auto.6f1972e48e')" value="running" />
                <el-option :label="t('domain.auto.20391328c8')" value="finished" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.974d383f36')">
              <el-select v-model="form.owner_id" clearable filterable class="full-input">
                <el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.e941d410f4')">
              <el-select v-model="form.company_id" class="full-input" clearable filterable :placeholder="t('domain.auto.42379aae38')" @change="applyCompany">
                <el-option v-for="item in companyOptions" :key="item.id" :label="companyOptionLabel(item)" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.2425bd4bc1')">
              <el-select v-model="form.contact_id" class="full-input" clearable filterable :placeholder="t('domain.auto.56a1a3ddad')" @change="applyContact">
                <el-option v-for="item in contactOptions" :key="item.id" :label="contactOptionLabel(item)" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.e8868af6eb')">
              <el-date-picker v-model="form.start_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" :placeholder="t('domain.auto.c4e356be18')" />
            </el-form-item>
            <el-form-item :label="t('domain.auto.a0bb9f49ab')">
              <el-date-picker v-model="form.end_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" :placeholder="t('domain.auto.f1dc195d14')" />
            </el-form-item>
            <el-form-item :label="t('domain.auto.be59a54e35')" class="wide-field">
              <ObjectAccessEditor
                v-model:visibility="form.payload.visibility"
                v-model:entries="formAccessEntries"
                :users="users"
                :public-label="t('domain.auto.49d409a41b')"
                :private-label="t('domain.auto.ff80493967')"
              />
            </el-form-item>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.e7f0c16259')" name="channel">
            <div class="campaign-guide wide-field">
              <strong>{{ t('domain.auto.128646180f') }}</strong>
              <span>{{ t('domain.auto.4f82d84200') }}</span>
            </div>
            <el-form-item :label="t('domain.auto.c3f47175c6')">
              <el-select v-model="form.payload.source_template" class="full-input" @change="applySourceTemplate">
                <el-option v-for="item in campaignTypeOptions" :key="item.value" :label="item.label" :value="item.value">
                  <span>{{ item.label }}</span>
                  <span class="product-option-meta">{{ item.sourceHelp }}</span>
                </el-option>
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.58a68a4f39')">
              <el-select v-model="form.payload.source_channel" filterable allow-create class="full-input">
                <el-option v-for="item in sourceChannelOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.9efd200e70')">
              <el-select v-model="form.payload.utm_source" filterable allow-create class="full-input">
                <el-option-group v-for="group in utmOptionGroups" :key="group.label" :label="group.label">
                  <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
                </el-option-group>
              </el-select>
            </el-form-item>
            <el-collapse v-model="sourceAdvancedOpen" class="campaign-advanced-collapse wide-field">
              <el-collapse-item :title="t('domain.auto.3a26497969')" name="source">
                <div class="campaign-inline-help">{{ t('domain.auto.6cc3a780d9') }}</div>
                <div class="campaign-form-grid">
                  <el-form-item :label="t('domain.auto.4ac31a8bb7')">
                    <el-select v-model="form.payload.utm_medium" filterable allow-create class="full-input">
                      <el-option v-for="item in mediumOptions" :key="item.value" :label="item.label" :value="item.value" />
                    </el-select>
                  </el-form-item>
                  <el-form-item :label="t('domain.auto.68a5f45822')">
                    <el-input v-model="form.payload.utm_campaign" :placeholder="t('domain.auto.1ed1d73417')" />
                  </el-form-item>
                </div>
              </el-collapse-item>
            </el-collapse>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.914555989d')" name="budget">
            <div class="campaign-guide wide-field">
              <strong>{{ t('domain.auto.59fc821df5') }}</strong>
              <span>{{ t('domain.auto.a843f50e06') }}</span>
            </div>
            <el-form-item :label="t('domain.auto.a81ab5e100')">
              <el-select v-model="form.currency" class="full-input">
                <el-option v-for="item in campaignCurrencyOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.0dcf0e012a')">
              <div class="campaign-money-input"><el-input-number v-model="form.payload.budget" :min="0" class="full-input" /><span>{{ form.currency || defaultBusinessCurrency() }}</span></div>
            </el-form-item>
            <el-form-item :label="t('domain.auto.581a15421e')">
              <div class="campaign-money-input"><el-input-number v-model="form.payload.cost" :min="0" class="full-input" /><span>{{ form.currency || defaultBusinessCurrency() }}</span></div>
            </el-form-item>
            <el-form-item :label="t('domain.auto.13c87e76a1')">
              <el-input-number v-model="form.payload.target_leads" :min="0" class="full-input" />
            </el-form-item>
            <el-form-item :label="t('domain.auto.447905b5c7')">
              <div class="campaign-money-input"><el-input-number v-model="form.payload.target_revenue" :min="0" class="full-input" /><span>{{ form.currency || defaultBusinessCurrency() }}</span></div>
            </el-form-item>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.4ea4d4727d')" name="material">
            <div class="campaign-guide wide-field">
              <strong>{{ t('domain.auto.7957cd0ee1') }}</strong>
              <span>{{ t('domain.auto.c30299fd4d') }}</span>
            </div>
            <el-form-item :label="t('domain.auto.1ed8160dcb')">
              <el-input v-model="form.payload.landing_page" placeholder="https://example.com/campaign" />
            </el-form-item>
            <div class="campaign-upload-panel wide-field">
              <div>
                <strong>{{ t('domain.auto.1fa785bcc9') }}</strong>
                <span>{{ t('domain.auto.7b4abfb002') }}</span>
              </div>
              <el-upload :show-file-list="false" :http-request="uploadCampaignFile" :disabled="fileUploading">
                <el-button type="primary" plain :loading="fileUploading">{{ t('domain.auto.287d6fd3a9') }}</el-button>
              </el-upload>
              <div v-if="campaignFiles.length" class="campaign-file-list wide-field">
                <div v-for="item in campaignFiles" :key="item.id" class="campaign-file-item">
                  <div>
                    <strong>{{ filePayload(item).file_name || item.name }}</strong>
                    <span>{{ fileCategoryLabel(filePayload(item).category) }} · {{ fileSizeText(filePayload(item).file_size) }}</span>
                  </div>
                  <div class="campaign-file-actions">
                    <el-button link type="primary" @click="downloadCampaignFile(item)">{{ t('domain.auto.d668db53bc') }}</el-button>
                    <el-button link type="danger" @click="deleteCampaignFile(item)">{{ t('domain.auto.3755f56f2f') }}</el-button>
                  </div>
                </div>
              </div>
            </div>
            <el-form-item :label="t('domain.auto.e0361480e3')" class="wide-field">
              <el-input v-model="form.payload.note" type="textarea" :rows="3" :placeholder="t('domain.auto.08c043654b')" />
            </el-form-item>
            <el-form-item :label="t('domain.auto.6ce74cec96')" class="wide-field">
              <el-select v-model="form.payload.attribution_model" clearable class="full-input">
                <el-option v-for="item in attributionModelOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="formDialog = false">{{ t('campaign.cancel') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('campaign.submit') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="followUpDialog" :title="t('domain.auto.769abf8b2b')" width="620px" destroy-on-close>
      <el-form :model="followUpForm" label-width="104px" class="lead-dialog-form">
        <el-form-item :label="t('domain.auto.43d2da0170')">
          <el-select v-model="followUpForm.type" class="full-input">
            <el-option v-for="item in followUpTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.163aec9194')" required class="wide-field">
          <el-input v-model="followUpForm.content" type="textarea" :rows="4" :placeholder="t('domain.auto.cfedbafc0d')" />
        </el-form-item>
        <el-form-item :label="t('domain.auto.ab2583981e')">
          <el-input v-model="followUpForm.next_action" :placeholder="t('domain.auto.228ba21097')" />
        </el-form-item>
        <el-form-item :label="t('domain.auto.6af6db8140')">
          <el-date-picker v-model="followUpForm.next_follow_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" :placeholder="t('domain.auto.53e32830a5')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="followUpDialog = false">{{ t('campaign.cancel') }}</el-button>
        <el-button type="primary" @click="submitFollowUp">{{ t('campaign.submit') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="leadDrawer" :title="t('domain.auto.0e3f367fc7')" size="420px">
      <div v-if="leadRecord" class="lead-summary">
        <div><span>{{ t('campaign.lead') }}</span><strong>{{ leadRecord.name }}</strong></div>
        <div><span>{{ t('domain.auto.62e951a692') }}</span><strong>{{ statusLabel(leadRecord.status) }}</strong></div>
        <div><span>{{ t('campaign.email') }}</span><strong>{{ parsePayload(leadRecord.payload_json).email || '-' }}</strong></div>
        <div><span>{{ t('campaign.phone') }}</span><strong>{{ parsePayload(leadRecord.payload_json).phone || '-' }}</strong></div>
        <div><span>{{ t('domain.auto.c63f79e636') }}</span><strong>{{ sourceLabel(parsePayload(leadRecord.payload_json).source_channel || parsePayload(leadRecord.payload_json).utm_source) }}</strong></div>
      </div>
    </el-drawer>

    <el-drawer v-model="relatedDrawer" :title="relatedDrawerTitle" size="520px">
      <div class="campaign-related-list">
        <button v-for="item in relatedDrawerRows" :key="item.id" class="campaign-related-item" @click="goObject(relatedDrawerType, item)">
          <strong>{{ item.name }}</strong>
          <span>{{ statusLabel(item.status) }} · {{ ownerName(item.owner_id) }}</span>
          <em>{{ formatAmount(item.amount, item.currency) }}</em>
        </button>
        <el-empty v-if="!relatedDrawerRows.length" :description="t('domain.auto.646d03bffd')" />
      </div>
    </el-drawer>

    <el-drawer v-model="advancedDrawer" :title="t('domain.auto.2bd5435541')" size="680px">
      <div class="campaign-guide">
        <strong>{{ t('domain.auto.936ed05d8e') }}</strong>
        <span>{{ t('domain.auto.af3f04c9d7') }}</span>
      </div>
      <div class="chart-panel-head">
        <strong>{{ t('domain.auto.e44813af8b') }}</strong>
        <el-tag size="small">{{ selectedAttribution?.name || t('domain.auto.75ea3ef06e') }}</el-tag>
      </div>
      <div class="campaign-channel-list">
        <div v-for="item in channelRows" :key="item.channel" class="campaign-channel-row">
          <span>{{ sourceLabel(item.channel) }}</span>
          <div><i :style="{ width: chartWidth(item.revenue, maxChannelRevenue) }"></i></div>
          <strong>{{ formatAmount(item.revenue, selectedCampaign?.currency || defaultBusinessCurrency()) }}</strong>
        </div>
      </div>
      <el-empty v-if="!channelRows.length" :description="t('domain.auto.8d90ef4614')" />
      <el-divider>{{ t('domain.auto.dfbce24cae') }}</el-divider>
      <el-table :data="touchRows" class="system-table" size="small">
        <el-table-column :label="t('campaign.stage')" prop="stage" min-width="110" />
        <el-table-column :label="t('campaign.channel')" min-width="110"><template #default="{ row }">{{ sourceLabel(row.channel) }}</template></el-table-column>
        <el-table-column :label="t('campaign.touch')" prop="touch" min-width="150" />
        <el-table-column :label="t('campaign.weight')" width="90"><template #default="{ row }">{{ percent(row.weight) }}</template></el-table-column>
        <el-table-column :label="t('campaign.revenue')" width="130"><template #default="{ row }">{{ formatAmount(row.revenue, selectedCampaign?.currency || defaultBusinessCurrency()) }}</template></el-table-column>
      </el-table>
    </el-drawer>

    <el-drawer v-model="columnDrawer" :title="t('domain.auto.56292a48ef')" size="320px">
      <div class="column-settings">
        <div class="column-settings-head">
          <el-button link type="primary" @click="columnSelections = columnOptions.map((item) => item.key)">{{ t('campaign.selectAll') }}</el-button>
          <el-button link type="primary" @click="resetColumns">{{ t('campaign.resetDefault') }}</el-button>
        </div>
        <el-checkbox-group v-model="columnSelections" class="column-checkboxes" @change="saveColumns">
          <el-checkbox v-for="item in columnOptions" :key="item.key" :label="item.key">{{ item.label }}</el-checkbox>
        </el-checkbox-group>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canEditRecord, canFollowRecord, canManageDomain, canManageRecordACL, currentUser } from '../../permissions'
import ObjectAccessPanel from '../../components/ObjectAccessPanel.vue'
import ObjectAccessEditor from '../../components/ObjectAccessEditor.vue'
import MetricCard from '../../components/MetricCard.vue'
import { currencyOptions as campaignCurrencyOptions, formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { downloadProtectedFile, unwrapFileUploadResponse } from '../../utils/download'
import { timelineOperatorText } from '../../utils/timeline'
import { useCustomerChainOptions } from '../../utils/customerChain'
import { apiErrorMessage } from '../../utils/api-error'

const router = useRouter()
const rows = ref([])
const allRows = ref([])
const users = ref([])
const selectedCampaign = ref(null)
const selectedAttribution = ref(null)
const relatedLeads = ref([])
const performance = ref({})
const campaignMetrics = ref({})
const loading = ref(false)
const fileUploading = ref(false)
const formDialog = ref(false)
const followUpDialog = ref(false)
const columnDrawer = ref(false)
const chainDrawer = ref(false)
const detailDrawer = ref(false)
const leadDrawer = ref(false)
const relatedDrawer = ref(false)
const advancedDrawer = ref(false)
const editingId = ref(null)
const leadRecord = ref(null)
const relatedDrawerType = ref('opportunities')
const chainCardRef = ref(null)
const chainHighlight = ref(false)
const filters = reactive({ keyword: '', status: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
const formAccessEntries = ref([])
const {
  companyOptions,
  contactOptions,
  loadCustomerOptions,
  companyOptionLabel,
  contactOptionLabel,
  applyCompany,
  applyContact,
  syncCustomerChainFromRecord,
  normalizeCustomerChainForSubmit
} = useCustomerChainOptions(form)
const followUpForm = reactive(defaultFollowUpForm())
const activeFormTab = ref('base')
const activeResultTab = ref('leads')
const sourceAdvancedOpen = ref([])
const LEGACY_ACL_UPDATE_SUMMARY = '\u66f4\u65b0\u534f\u4f5c\u6743\u9650'
const LEGACY_ACTIVITY_SUMMARY_PREFIX = '\u6d3b\u52a8\u52a8\u6001\uff1a'
const columnOptions = [
  { key: 'name', label: t('domain.auto.b2548636f0') },
  { key: 'status', label: t('domain.auto.62e951a692') },
  { key: 'time', label: t('domain.auto.abe0ecdb04') },
  { key: 'progress', label: t('domain.auto.32e920a5df') },
  { key: 'budget', label: t('domain.auto.0dcf0e012a') },
  { key: 'leads', label: t('domain.auto.8de0b8fb1d') },
  { key: 'opportunities', label: t('domain.auto.24aef3c29e') },
  { key: 'revenue', label: t('domain.auto.02720419d3') },
  { key: 'owner', label: t('domain.auto.974d383f36') },
  { key: 'next', label: t('domain.auto.ea0ef2ae72') }
]
const columnSelections = ref(loadColumns())

const campaignTypeOptions = [
  { value: 'webinar', label: t('domain.auto.1f536d3b67'), desc: t('domain.auto.ba9930bf4d'), source_channel: 'campaign', utm_source: 'webinar', utm_medium: 'event', utm_campaign: 'webinar', sourceHelp: t('domain.auto.d9d6a14c1f') },
  { value: 'ads', label: t('domain.auto.3cb249b95e'), desc: t('domain.auto.1410f33348'), source_channel: 'ads', utm_source: 'google_ads', utm_medium: 'cpc', utm_campaign: 'paid_campaign', sourceHelp: t('domain.auto.21afee0d92') },
  { value: 'whitepaper', label: t('domain.auto.785e4caa19'), desc: t('domain.auto.b8ec0790f2'), source_channel: 'content', utm_source: 'whitepaper', utm_medium: 'content', utm_campaign: 'whitepaper', sourceHelp: t('domain.auto.a54ac019cc') },
  { value: 'event', label: t('domain.auto.201f7cace7'), desc: t('domain.auto.cf041e98da'), source_channel: 'event', utm_source: 'expo_event', utm_medium: 'offline', utm_campaign: 'offline_event', sourceHelp: t('domain.auto.3ba6674ba9') },
  { value: 'social', label: t('domain.auto.b2f3c23b7b'), desc: t('domain.auto.7aef6c34c7'), source_channel: 'social', utm_source: 'linkedin', utm_medium: 'social', utm_campaign: 'social_campaign', sourceHelp: t('domain.auto.e507b0e3a5') },
  { value: 'email', label: t('domain.auto.b773308aff'), desc: t('domain.auto.051b2f135c'), source_channel: 'email', utm_source: 'email_campaign', utm_medium: 'email', utm_campaign: 'email_campaign', sourceHelp: t('domain.auto.cbea95d982') },
  { value: 'partner', label: t('domain.auto.6df294494b'), desc: t('domain.auto.88bda72abc'), source_channel: 'partner', utm_source: 'referral', utm_medium: 'partner', utm_campaign: 'partner_referral', sourceHelp: t('domain.auto.85886c728a') },
  { value: 'phone', label: t('domain.auto.0bed4c2ea9'), desc: t('domain.auto.e1578f103e'), source_channel: 'phone', utm_source: 'sales_call', utm_medium: 'call', utm_campaign: 'phone_outbound', sourceHelp: t('domain.auto.b65a684f47') },
  { value: 'repeat', label: t('domain.auto.958c027a5a'), desc: t('domain.auto.a6fc57db07'), source_channel: 'customer', utm_source: 'existing_customer', utm_medium: 'crm', utm_campaign: 'customer_growth', sourceHelp: t('domain.auto.c03b2729ae') },
  { value: 'other', label: t('domain.auto.1a26edf94a'), desc: t('domain.auto.e1b0cc289f'), source_channel: 'other', utm_source: 'other', utm_medium: 'other', utm_campaign: 'other_campaign', sourceHelp: t('domain.auto.30c1600b9b') }
]
const sourceChannelOptions = [
  { value: 'campaign', label: t('domain.auto.5fa95b1b73') },
  { value: 'ads', label: t('domain.auto.3cb249b95e') },
  { value: 'website', label: t('domain.auto.5b5b925ba4') },
  { value: 'content', label: t('domain.auto.19e548ea66') },
  { value: 'event', label: t('domain.auto.201f7cace7') },
  { value: 'social', label: t('domain.auto.b2f3c23b7b') },
  { value: 'email', label: t('domain.auto.b773308aff') },
  { value: 'partner', label: t('domain.auto.6df294494b') },
  { value: 'phone', label: t('domain.auto.0bed4c2ea9') },
  { value: 'customer', label: t('domain.auto.958c027a5a') },
  { value: 'organic_search', label: t('domain.auto.8803fa49b4') },
  { value: 'direct', label: t('domain.auto.3400e6edb1') },
  { value: 'other', label: t('domain.auto.1a26edf94a') }
]
const utmOptionGroups = [
  { label: t('domain.auto.975b40f704'), options: [{ value: 'google_ads', label: 'Google Ads' }, { value: 'bing_ads', label: 'Bing Ads' }, { value: 'meta_ads', label: 'Meta/Facebook Ads' }, { value: 'linkedin_ads', label: 'LinkedIn Ads' }, { value: 'tiktok_ads', label: 'TikTok Ads' }] },
  { label: t('domain.auto.cb77f9f63b'), options: [{ value: 'website_form', label: t('domain.auto.a2af045711') }, { value: 'landing_page', label: t('domain.auto.1ed8160dcb') }, { value: 'webinar', label: t('domain.auto.454170ab01') }, { value: 'whitepaper', label: t('domain.auto.785e4caa19') }, { value: 'expo_event', label: t('domain.auto.2fce6b3fa6') }] },
  { label: t('domain.auto.8ba30a6c1f'), options: [{ value: 'linkedin', label: 'LinkedIn' }, { value: 'facebook', label: 'Facebook' }, { value: 'whatsapp', label: 'WhatsApp' }, { value: 'line', label: 'Line' }, { value: 'referral', label: t('domain.auto.b5ea49f77d') }] },
  { label: t('domain.auto.25927bb889'), options: [{ value: 'email_campaign', label: t('domain.auto.b773308aff') }, { value: 'newsletter', label: 'Newsletter' }, { value: 'remarketing', label: t('domain.auto.358005f41b') }, { value: 'sales_call', label: t('domain.auto.0bed4c2ea9') }, { value: 'existing_customer', label: t('domain.auto.958c027a5a') }, { value: 'direct', label: t('domain.auto.3400e6edb1') }, { value: 'organic_search', label: t('domain.auto.8803fa49b4') }] }
]
const mediumOptions = [
  { value: 'cpc', label: t('domain.auto.8139c0dfac') },
  { value: 'paid_social', label: t('domain.auto.a0a89c642e') },
  { value: 'social', label: t('domain.auto.1b5ab45cca') },
  { value: 'email', label: t('domain.auto.1c8e464184') },
  { value: 'event', label: t('domain.auto.2900996c93') },
  { value: 'offline', label: t('domain.auto.bbed4c838b') },
  { value: 'partner', label: t('domain.auto.6df294494b') },
  { value: 'call', label: t('domain.auto.f39c6b629f') },
  { value: 'content', label: t('domain.auto.19e548ea66') },
  { value: 'crm', label: t('domain.auto.2d07a8ee2a') },
  { value: 'organic', label: t('domain.auto.d482e2f763') },
  { value: 'other', label: t('domain.auto.1a26edf94a') }
]
const attributionModelOptions = [
  { value: 'position_time_decay', label: t('domain.auto.81c70c6c27') },
  { value: 'linear', label: t('domain.auto.cc60cebf14') },
  { value: 'first_touch', label: t('domain.auto.f2948fca4b') },
  { value: 'last_touch', label: t('domain.auto.d1621fc5f9') }
]
const followUpTypeOptions = [
  { value: 'execution', label: t('domain.auto.2c1a8b4e89') },
  { value: 'call', label: t('domain.auto.be56921d24') },
  { value: 'email', label: t('domain.auto.4404c87bd9') },
  { value: 'meeting', label: t('domain.auto.33e9651895') },
  { value: 'material_update', label: t('domain.auto.ff025b54a5') },
  { value: 'review', label: t('domain.auto.4d3d73b2d4') },
  { value: 'note', label: t('domain.auto.e0361480e3') }
]
const fileCategoryOptions = [
  { value: 'brochure', label: t('domain.auto.07c09427b9') },
  { value: 'poster', label: t('domain.auto.e63a92c582') },
  { value: 'whitepaper', label: t('domain.auto.a7761629bb') },
  { value: 'video', label: t('domain.auto.fa4e33b698') },
  { value: 'quote', label: t('domain.auto.269b089854') },
  { value: 'landing', label: t('domain.auto.a038b6bd9a') },
  { value: 'creative', label: t('domain.auto.24f8acfbaf') },
  { value: 'other', label: t('domain.auto.1a26edf94a') }
]

const campaignRows = computed(() => allRows.value.filter((item) => item.object_type === 'campaign'))
const canManage = computed(() => canManageDomain('campaigns'))
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))
const attributionRows = computed(() => allRows.value.filter((item) => item.object_type === 'attribution'))
const latestAttributionPayload = computed(() => parsePayload(selectedAttribution.value?.payload_json))
const performanceSummary = computed(() => performance.value?.summary || {})
const selectedCampaignPayload = computed(() => parsePayload(selectedCampaign.value?.payload_json))
const selectedCampaignSource = computed(() => selectedCampaignPayload.value.source_channel || selectedCampaignPayload.value.utm_source || '')
const relatedOpportunities = computed(() => performance.value?.opportunities || [])
const relatedDeals = computed(() => performance.value?.deals || [])
const relatedOrders = computed(() => performance.value?.orders || [])
const campaignTimeline = computed(() => performance.value?.timeline || [])
const campaignFiles = computed(() => performance.value?.files || [])
const chainRows = computed(() => performance.value?.chain || [])
const latestSummary = computed(() => ({ ...(latestAttributionPayload.value.summary || {}), ...performanceSummary.value }))
const touchRows = computed(() => latestAttributionPayload.value.touches || [])
const channelRows = computed(() => latestSummary.value.channels || [])
const maxChannelRevenue = computed(() => Math.max(1, ...channelRows.value.map((item) => Number(item.revenue || 0))))
const pageMetricSummary = computed(() => rows.value.reduce((summary, row) => {
  const metric = metricFor(row)
  summary.leads += Number(metric.lead_count || 0)
  summary.opportunities += Number(metric.opportunity_count || 0)
  summary.revenue += Number(metric.revenue_amount || 0)
  summary.cost += Number(metric.campaign_cost || campaignBudget(row) || 0)
  return summary
}, { leads: 0, opportunities: 0, revenue: 0, cost: 0 }))
const kpiCards = computed(() => {
  const summary = pageMetricSummary.value
  const roi = summary.cost > 0 ? (summary.revenue - summary.cost) / summary.cost : 0
  return [
    { key: 'campaigns', title: t('domain.auto.b993dea3ac'), value: pagination.total || rows.value.length, icon: 'ri:megaphone-line', tone: 'blue', help: t('domain.auto.57ef2795c2') },
    { key: 'leads', title: t('domain.auto.ce73a9b731'), value: summary.leads, icon: 'ri:user-add-line', tone: 'green', help: t('domain.auto.d9d8f264bf') },
    { key: 'opportunities', title: t('domain.auto.24aef3c29e'), value: summary.opportunities, icon: 'ri:briefcase-4-line', tone: 'purple', help: t('domain.auto.13fa2c6ff6') },
    { key: 'revenue', title: t('domain.auto.02720419d3'), value: formatAmount(summary.revenue, selectedCampaign.value?.currency || defaultBusinessCurrency()), icon: 'ri:money-dollar-circle-line', tone: 'orange', help: t('domain.auto.e880eca84d') },
    { key: 'roi', title: 'ROI', value: percent(roi), icon: 'ri:funds-line', tone: roi >= 0 ? 'green' : 'red', help: t('domain.auto.7d44bcc07e') }
  ]
})
const attentionItems = computed(() => buildAttentionItems())
const relatedDrawerRows = computed(() => {
  if (relatedDrawerType.value === 'leads') return relatedLeads.value
  if (relatedDrawerType.value === 'deals') return relatedDeals.value
  if (relatedDrawerType.value === 'orders') return relatedOrders.value
  return relatedOpportunities.value
})
const relatedDrawerTitle = computed(() => {
  const titles = {
    leads: t('domain.auto.0e3f367fc7'),
    opportunities: t('domain.auto.24aef3c29e'),
    deals: t('domain.auto.6a30a471cf'),
    orders: t('domain.auto.8ef538db71')
  }
  return titles[relatedDrawerType.value] || t('domain.auto.3dc6f1edc5')
})

onMounted(async () => {
  await Promise.all([loadUsers(), loadCustomerOptions(), refreshAll()])
})

async function loadUsers() {
  try {
    const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } })
    users.value = res.data?.items || []
    if (!users.value.length) {
      const fallback = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200 } })
      users.value = fallback.data?.items || []
    }
  } catch {
    const me = currentUser()
    users.value = me?.id ? [me] : []
  }
  const me = currentUser()
  if (me?.id && !users.value.some((item) => Number(item.id) === Number(me.id))) {
    users.value = [{ ...me, name: me.name || me.username }, ...users.value]
  }
}

async function refreshAll() {
  await loadAllRows()
  await loadPage()
}

async function loadAllRows() {
  try {
    const res = await api.get('/api/v1/prd/campaigns/objects', { params: { page: 1, size: 200 } })
    allRows.value = res.data?.items || []
    if (selectedCampaign.value) {
      selectedCampaign.value = campaignRows.value.find((item) => item.id === selectedCampaign.value.id) || null
    }
    syncSelectedAttribution()
    await loadPerformance()
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.8e82b4c36e')))
  }
}

async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/campaigns/objects', { params: cleanParams({ keyword: filters.keyword, status: filters.status, object_type: 'campaign', page: pagination.page, size: pagination.size }) })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
    await loadMetricsForRows(rows.value)
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.f6b7a41e56')))
  } finally {
    loading.value = false
  }
}

async function loadMetricsForRows(items) {
  const campaigns = (items || []).filter((item) => item.object_type === 'campaign')
  if (!campaigns.length) return
  const next = { ...campaignMetrics.value }
  await Promise.all(campaigns.map(async (row) => {
    try {
      const res = await api.get(`/api/v1/prd/flows/campaigns/${row.id}/performance`)
      next[row.id] = res.data || {}
    } catch {
      next[row.id] = next[row.id] || { summary: {} }
    }
  }))
  campaignMetrics.value = next
}

async function selectCampaign(row) {
  if (!row) return
  if (row.object_type === 'attribution') {
    selectedAttribution.value = row
    const campaignID = Number(parsePayload(row.payload_json).campaign_id || 0)
    selectedCampaign.value = campaignRows.value.find((item) => item.id === campaignID) || selectedCampaign.value
  } else if (row.object_type === 'campaign') {
    selectedCampaign.value = row
    syncSelectedAttribution()
  }
  await loadPerformance()
}

function campaignRowClassName({ row }) {
  return selectedCampaign.value?.id === row.id ? 'campaign-row-selected' : ''
}

function syncSelectedAttribution() {
  if (!selectedCampaign.value) {
    selectedAttribution.value = attributionRows.value[0] || null
    return
  }
  selectedAttribution.value = attributionRows.value.find((item) => Number(parsePayload(item.payload_json).campaign_id) === selectedCampaign.value.id) || null
}

async function loadPerformance() {
  if (!selectedCampaign.value) {
    relatedLeads.value = []
    performance.value = {}
    return
  }
  try {
    const res = await api.get(`/api/v1/prd/flows/campaigns/${selectedCampaign.value.id}/performance`)
    performance.value = res.data || {}
    relatedLeads.value = res.data?.leads || []
    campaignMetrics.value = { ...campaignMetrics.value, [selectedCampaign.value.id]: res.data || {} }
  } catch {
    performance.value = {}
    relatedLeads.value = []
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, defaultForm())
  formAccessEntries.value = []
  activeFormTab.value = 'base'
  sourceAdvancedOpen.value = []
  formDialog.value = true
  if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions()
}

async function openEdit(row) {
  if (!row || row.object_type !== 'campaign') return
  selectedCampaign.value = row
  await loadPerformance()
  editingId.value = row.id
  Object.assign(form, { object_type: 'campaign', name: row.name, code: row.code || '', status: row.status || 'running', owner_id: row.owner_id || 0, company_id: row.company_id || 0, contact_id: row.contact_id || 0, amount: row.amount || 0, currency: row.currency || defaultBusinessCurrency(), score: row.score || 0, start_at: normalizeDateTimeForInput(row.start_at), end_at: normalizeDateTimeForInput(row.end_at), tags: splitTags(row.tags), payload: { ...defaultForm().payload, ...parsePayload(row.payload_json) } })
  syncCustomerChainFromRecord(row)
  if (!form.payload.visibility) form.payload.visibility = objectVisibility(row)
  if (!form.payload.type) form.payload.type = 'webinar'
  if (!form.payload.source_template) form.payload.source_template = matchingTemplateValue(form.payload) || form.payload.type
  if (form.payload.budget === undefined) form.payload.budget = row.amount || 0
  formAccessEntries.value = await loadObjectACLEditEntries(row.id)
  activeFormTab.value = 'base'
  sourceAdvancedOpen.value = []
  formDialog.value = true
  if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions()
}

function prepareCampaignForm() {
  if (!String(form.name || '').trim()) {
    ElMessage.error(t('domain.auto.41d38959b7'))
    return false
  }
  form.object_type = 'campaign'
  form.name = String(form.name || '').trim()
  form.currency = form.currency || defaultBusinessCurrency()
  form.amount = Number(form.payload.budget || 0)
  if (form.start_at && form.end_at && new Date(form.start_at.replace(' ', 'T')).getTime() > new Date(form.end_at.replace(' ', 'T')).getTime()) {
    ElMessage.error(t('domain.auto.c16dacab6c'))
    return false
  }
  form.payload.private = form.payload.visibility === 'private'
  form.payload.campaign_name = form.name
  normalizeCustomerChainForSubmit()
  if (form.payload.visibility !== 'private') formAccessEntries.value = []
  return true
}

function buildCampaignRowFromForm(id, existing = {}) {
  return {
    ...existing,
    id: Number(id),
    object_type: 'campaign',
    name: form.name,
    code: form.code || existing.code || '',
    status: form.status || existing.status || 'running',
    owner_id: Number(form.owner_id || existing.owner_id || currentUser().id || 0),
    company_id: Number(form.company_id || existing.company_id || form.payload.company_id || 0),
    contact_id: Number(form.contact_id || existing.contact_id || form.payload.contact_id || 0),
    amount: Number(form.amount || 0),
    currency: form.currency || existing.currency || defaultBusinessCurrency(),
    start_at: form.start_at || existing.start_at || '',
    end_at: form.end_at || existing.end_at || '',
    score: Number(form.score || existing.score || 0),
    tags: Array.isArray(form.tags) ? form.tags.join(',') : form.tags,
    payload_json: JSON.stringify(form.payload || {})
  }
}

async function persistCampaignForm() {
  if (!prepareCampaignForm()) return null
  let savedID = Number(editingId.value || 0)
  let savedObject = null
  if (editingId.value) {
    await api.put(`/api/v1/prd/campaigns/objects/${editingId.value}`, form)
    savedObject = buildCampaignRowFromForm(editingId.value, selectedCampaign.value || {})
  } else {
    const res = await api.post('/api/v1/prd/campaigns/objects', form)
    savedObject = res.data?.object || res.data?.item || null
    savedID = Number(savedObject?.id || res.data?.id || 0)
    if (savedID) {
      editingId.value = savedID
      savedObject = savedObject ? { ...buildCampaignRowFromForm(savedID), ...savedObject } : buildCampaignRowFromForm(savedID)
    }
  }
  if (savedID) await saveObjectACLEditEntries(savedID)
  return savedObject || (savedID ? buildCampaignRowFromForm(savedID) : null)
}

async function submitForm() {
  try {
    const saved = await persistCampaignForm()
    if (!saved) return
    ElMessage.success(t('domain.auto.3c5be5f2a1'))
    formDialog.value = false
    await refreshAll()
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.b412eb86ab')))
  }
}

async function loadObjectACLEditEntries(objectID) {
  if (!objectID) return []
  try {
    const res = await api.get(`/api/v1/prd/campaigns/objects/${objectID}/acl`)
    return (res.data?.items || []).map((item) => ({
      uid: `acl_${item.user_id}_${item.id || Date.now()}`,
      user_id: item.user_id,
      access_level: item.access_level || 'view'
    }))
  } catch {
    return []
  }
}

async function saveObjectACLEditEntries(objectID) {
  const entries = form.payload.visibility === 'private' ? normalizeAccessEntries(formAccessEntries.value) : []
  await api.put(`/api/v1/prd/campaigns/objects/${objectID}/acl`, { entries })
}

function normalizeAccessEntries(entries) {
  const seen = new Set()
  return (entries || [])
    .filter((item) => item.user_id)
    .map((item) => ({ user_id: Number(item.user_id), access_level: item.access_level || 'view' }))
    .filter((item) => {
      if (seen.has(item.user_id)) return false
      seen.add(item.user_id)
      return true
    })
}

async function generateAttribution(row) {
  if (!row || row.object_type !== 'campaign') return
  try {
    const res = await api.post(`/api/v1/prd/flows/campaigns/${row.id}/attribution`)
    ElMessage.success(t('domain.auto.427f22549e'))
    await refreshAll()
    const attribution = res.data?.attribution
    if (attribution) selectedAttribution.value = attribution
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.a9a87690ee')))
  }
}

async function exportObjects() {
  try {
    const res = await api.get('/api/v1/prd/campaigns/objects/export')
    const count = res.data?.count || res.data?.items?.length || 0
    ElMessage.success(`${t('domain.auto.35f670828a')}${count}${t('domain.auto.bce2ef6151')}`)
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.6753381261')))
  }
}

async function refreshSelectedCampaign(id) {
  await refreshAll()
  selectedCampaign.value = campaignRows.value.find((item) => Number(item.id) === Number(id)) || selectedCampaign.value
}

function resetFilters() {
  Object.assign(filters, { keyword: '', status: '' })
  pagination.page = 1
  loadPage()
}

function openLead(row) {
  leadRecord.value = row
  leadDrawer.value = true
}

function openRelatedDrawer(type) {
  relatedDrawerType.value = type
  relatedDrawer.value = true
}

async function openDetailDrawer(row) {
  if (!row || row.object_type !== 'campaign') {
    ElMessage.warning(t('domain.auto.865fe354ef'))
    return
  }
  try {
    await selectCampaign(row)
    detailDrawer.value = true
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.cb67a55aba')))
  }
}

async function openChainDrawer(row) {
  if (!row || row.object_type !== 'campaign') {
    ElMessage.warning(t('domain.auto.865fe354ef'))
    return
  }
  try {
    chainDrawer.value = true
    await selectCampaign(row)
    await scrollToChain()
    ElMessage.success(t('domain.auto.a9a5330572'))
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.fa03c64ea1')))
  }
}

async function viewCampaignChain(row) {
  await openChainDrawer(row)
}

function handleCampaignMoreCommand(command) {
  if (!selectedCampaign.value) {
    ElMessage.warning(t('domain.auto.865fe354ef'))
    return
  }
  if (command === 'detail') return openDetailDrawer(selectedCampaign.value)
  if (command === 'follow') return openFollowUp()
  if (command === 'upload') {
    detailDrawer.value = true
    ElMessage.info(t('domain.auto.602c3597ba'))
    return
  }
  if (command === 'review') return generateAttribution(selectedCampaign.value)
  if (command === 'advanced') {
    advancedDrawer.value = true
  }
}

async function ensureCampaignSavedBeforeUpload() {
  if (formDialog.value && !editingId.value) {
    try {
      const saved = await persistCampaignForm()
      if (!saved) return null
      if (!saved?.id) throw new Error(t('domain.auto.890f92344a'))
      selectedCampaign.value = saved
      ElMessage.success(t('domain.auto.8f25a0ffa0'))
      await refreshSelectedCampaign(saved.id)
      await loadPerformance()
      return selectedCampaign.value || saved
    } catch (error) {
      ElMessage.error(errorMessage(error, t('domain.auto.64798a8e89')))
      return null
    }
  }
  if (formDialog.value && editingId.value) {
    const row = selectedCampaign.value?.id === Number(editingId.value)
      ? selectedCampaign.value
      : buildCampaignRowFromForm(editingId.value, selectedCampaign.value || {})
    selectedCampaign.value = row
    return row
  }
  if (selectedCampaign.value?.id) return selectedCampaign.value
  ElMessage.warning(t('domain.auto.865fe354ef'))
  return null
}

async function scrollToChain() {
  await nextTick()
  chainCardRef.value?.scrollIntoView?.({ behavior: 'smooth', block: 'center' })
  chainHighlight.value = true
  window.setTimeout(() => { chainHighlight.value = false }, 1400)
}

function openChainNode(key) {
  if (key === 'leads') return openRelatedDrawer('leads')
  if (key === 'opportunities') return openRelatedDrawer('opportunities')
  if (key === 'deals') return openRelatedDrawer('deals')
  if (key === 'orders' || key === 'payments') return openRelatedDrawer('orders')
}

function goObject(domain, row) {
  const routeNames = { leads: 'leads', opportunities: 'opportunities', deals: 'deals', orders: 'orders' }
  const name = routeNames[domain]
  if (!name) return
  router.push({ name, query: { id: row.id, campaign_id: selectedCampaign.value?.id } })
}

function runAttentionAction(item) {
  if (item.actionType === 'create') return openCreate()
  if (item.actionType === 'create_lead') return goCreateLeadFromCampaign(selectedCampaign.value)
  if (item.actionType === 'leads') return openRelatedDrawer('leads')
  if (item.actionType === 'opportunities') return openRelatedDrawer('opportunities')
  if (item.actionType === 'review') return generateAttribution(selectedCampaign.value)
  if (item.actionType === 'orders') return openRelatedDrawer('orders')
  if (item.actionType === 'detail') return openDetailDrawer(selectedCampaign.value)
  if (item.actionType === 'chain') return openChainDrawer(selectedCampaign.value)
}

function runRowNextAction(row) {
  const action = nextActionType(row)
  if (action === 'create_lead') return goCreateLeadFromCampaign(row)
  if (action === 'review') return generateAttribution(row)
  if (action === 'opportunities') return goRelatedFromCampaign(row, 'opportunities')
  if (action === 'follow_opportunities') return goRelatedFromCampaign(row, 'opportunities')
  if (action === 'orders') return goRelatedFromCampaign(row, 'orders')
  if (action === 'chain') return openChainDrawer(row)
  return openDetailDrawer(row)
}

function goCreateLeadFromCampaign(row) {
  if (!row || row.object_type !== 'campaign') {
    router.push({ name: 'leads', query: { create: '1', source_channel: 'campaign' } })
    return
  }
  const p = parsePayload(row.payload_json)
  router.push({
    name: 'leads',
    query: {
      campaign_id: row.id,
      campaign_name: row.name,
      company_id: p.company_id || row.company_id || '',
      contact_id: p.contact_id || row.contact_id || '',
      source_channel: p.source_channel || 'campaign',
      utm_source: p.utm_source || '',
      utm_medium: p.utm_medium || '',
      utm_campaign: p.utm_campaign || slugify(row.name || `campaign_${row.id}`)
    }
  })
}

function goRelatedFromCampaign(row, domain) {
  if (!row?.id) return
  router.push({ name: domain, query: { campaign_id: row.id } })
}

function applyCampaignTemplate(value) {
  const template = findTemplate(value)
  if (!template) return
  form.payload.source_template = template.value
  fillSourcePayload(form.payload, template, form.name)
}

function applySourceTemplate(value) {
  const template = findTemplate(value)
  if (!template) return
  fillSourcePayload(form.payload, template, form.name)
}

function fillSourcePayload(target, template, name) {
  target.source_channel = template.source_channel
  target.utm_source = template.utm_source
  target.utm_medium = template.utm_medium
  target.utm_campaign = slugify(name || template.utm_campaign || template.value)
}

function findTemplate(value) {
  return campaignTypeOptions.find((item) => item.value === value)
}

function matchingTemplateValue(payload) {
  const source = String(payload.source_channel || '')
  const utm = String(payload.utm_source || '')
  return campaignTypeOptions.find((item) => item.source_channel === source && item.utm_source === utm)?.value || ''
}

function buildAttentionItems() {
  if (!rows.value.length) {
    return [{ key: 'empty', icon: 'ri:add-circle-line', title: t('domain.auto.9fa3aaf087'), desc: t('domain.auto.5bcad1dfed'), action: t('domain.auto.7031d1096d'), actionType: 'create' }]
  }
  if (!selectedCampaign.value) {
    return [{ key: 'select', icon: 'ri:cursor-line', title: t('domain.auto.495ff59ce0'), desc: t('domain.auto.c28070968a'), action: t('domain.auto.f3a298ce49'), actionType: 'none' }]
  }
  const summary = performanceSummary.value
  const items = []
  const time = campaignTimeProgress(selectedCampaign.value)
  if (time.key === 'not_started') {
    items.push({ key: 'not-started', icon: 'ri:calendar-schedule-line', title: t('domain.auto.21dc4a18ab'), desc: `${time.label}${t('domain.auto.cad167590d')}`, action: t('domain.auto.faea8c1db9'), actionType: 'detail' })
  }
  if (time.key === 'ending_soon') {
    items.push({ key: 'ending-soon', icon: 'ri:alarm-warning-line', title: t('domain.auto.199b3ccc19'), desc: t('domain.auto.51c413b8e5'), action: t('domain.auto.f3a298ce49'), actionType: 'chain' })
  }
  if (time.key === 'ended') {
    items.push({ key: 'ended', icon: 'ri:flag-line', title: t('domain.auto.54de1348a6'), desc: t('domain.auto.e87d103ffa'), action: t('domain.auto.4eb78d99e0'), actionType: 'review' })
  }
  if (!Number(summary.lead_count || 0)) {
    items.push({ key: 'no-leads', icon: 'ri:user-add-line', title: t('domain.auto.9ae30cc75d'), desc: t('domain.auto.8d70982171'), action: t('domain.auto.47a1b14404'), actionType: 'create_lead' })
  }
  if (Number(summary.lead_count || 0) > 0 && !Number(summary.opportunity_count || 0)) {
    items.push({ key: 'no-opportunity', icon: 'ri:exchange-dollar-line', title: t('domain.auto.e774cd79b4'), desc: t('domain.auto.7aaed544a5'), action: t('domain.auto.d444c14883'), actionType: 'leads' })
  }
  if (Number(summary.campaign_cost || 0) > 0 && !Number(summary.revenue_amount || 0)) {
    items.push({ key: 'cost-no-revenue', icon: 'ri:funds-line', title: t('domain.auto.7ea8d1d13d'), desc: t('domain.auto.b0833b614c'), action: t('domain.auto.b4f5c14081'), actionType: 'opportunities' })
  }
  if (Number(summary.opportunity_count || 0) > 0 && !Number(summary.order_count || 0)) {
    items.push({ key: 'opportunity-no-order', icon: 'ri:shopping-cart-line', title: t('domain.auto.36867be29d'), desc: t('domain.auto.6a050866d7'), action: t('domain.auto.b1f8a261b9'), actionType: 'opportunities' })
  }
  if (!items.length) {
    items.push({ key: 'healthy', icon: 'ri:check-double-line', title: t('domain.auto.f16fc741d1'), desc: t('domain.auto.2d29332b5f'), action: t('domain.auto.4eb78d99e0'), actionType: 'review' })
  }
  return items
}

function defaultForm() {
  return { object_type: 'campaign', name: '', code: '', status: 'running', owner_id: Number(currentUser().id || 0), company_id: 0, contact_id: 0, amount: 0, currency: defaultBusinessCurrency(), start_at: '', end_at: '', score: 0, tags: [], payload: { company_id: undefined, company: '', company_name: '', contact_id: undefined, contact: '', contact_name: '', type: 'webinar', source_template: 'webinar', source_channel: 'campaign', utm_source: 'webinar', utm_medium: 'event', utm_campaign: 'webinar', budget: 0, cost: 0, target_leads: 0, target_revenue: 0, landing_page: '', note: '', attribution_model: 'position_time_decay', visibility: 'public' } }
}

function defaultFollowUpForm() {
  return { type: 'execution', content: '', next_action: '', next_follow_at: '' }
}

function openFollowUp() {
  if (!selectedCampaign.value) {
    ElMessage.warning(t('domain.auto.865fe354ef'))
    return
  }
  Object.assign(followUpForm, defaultFollowUpForm())
  followUpDialog.value = true
}

async function submitFollowUp() {
  if (!String(followUpForm.content || '').trim()) {
    ElMessage.error(t('domain.auto.01dedbf8a8'))
    return
  }
  if (!selectedCampaign.value?.id) return
  try {
    await api.post(`/api/v1/prd/flows/campaigns/${selectedCampaign.value.id}/follow-up`, {
      ...followUpForm,
      next_follow_at: normalizeDateTimeForAPI(followUpForm.next_follow_at)
    })
    ElMessage.success(t('domain.auto.0c7fb979d2'))
    followUpDialog.value = false
    await loadPerformance()
    await refreshSelectedCampaign(selectedCampaign.value.id)
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.2a5c37667d')))
  }
}

async function uploadCampaignFile(options) {
  const row = await ensureCampaignSavedBeforeUpload()
  if (!row?.id) {
    options.onError?.(new Error(t('domain.auto.2ccaf84a5b')))
    return
  }
  fileUploading.value = true
  try {
    const data = new FormData()
    data.append('file', options.file)
    const upload = await api.post('/api/v1/files/upload', data)
    const info = unwrapFileUploadResponse(upload)
    await api.post(`/api/v1/prd/flows/campaigns/${row.id}/files`, {
      file_id: info.file_id,
      file_name: info.original_name || info.file_name || options.file.name,
      file_size: Number(info.file_size || info.size || options.file.size || 0),
      file_type: options.file.type || info.file_type || '',
      storage_node_id: info.storage_node_id || '',
      download_url: info.download_url || `/api/v1/files/${info.file_id}/download`,
      category: inferCampaignFileCategory(options.file),
      visibility: 'internal'
    })
    ElMessage.success(t('domain.auto.5cddc47f4d'))
    await loadPerformance()
    options.onSuccess?.(info)
  } catch (error) {
    options.onError?.(error)
    ElMessage.error(errorMessage(error, t('domain.auto.68a811e394')))
  } finally {
    fileUploading.value = false
  }
}

async function downloadCampaignFile(file) {
  const p = filePayload(file)
  const fileID = p.file_id
  try {
    await downloadProtectedFile(p.download_url || `/api/v1/files/${fileID}/download`, p.file_name || file.name || t('domain.auto.6641b49102'))
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.d35ff84ea2')))
  }
}

async function deleteCampaignFile(file) {
  if (!selectedCampaign.value?.id || !file?.id) return
  try {
    await api.delete(`/api/v1/prd/flows/campaigns/${selectedCampaign.value.id}/files/${file.id}`)
    ElMessage.success(t('domain.auto.e927eb7eab'))
    await loadPerformance()
  } catch (error) {
    ElMessage.error(errorMessage(error, t('domain.auto.7d991ec57a')))
  }
}

function parsePayload(value) {
  try { return typeof value === 'string' ? JSON.parse(value || '{}') : (value || {}) } catch { return {} }
}
function objectVisibility(row) { const p = parsePayload(row?.payload_json); return p.visibility === 'private' || p.access_visibility === 'private' || p.private === true || p.private === 'true' ? 'private' : 'public' }
function campaignBudget(row) {
  const p = parsePayload(row?.payload_json)
  if (Number(p.budget || 0) > 0) return Number(p.budget || 0)
  if (Number(p.cost || 0) > 0) return Number(p.cost || 0)
  return Number(row?.amount || 0)
}
function metricFor(row) {
  return campaignMetrics.value[row?.id]?.summary || {}
}
function filePayload(row) { return parsePayload(row?.payload_json) }
function timelineTitle(row) {
  const p = parsePayload(row?.payload_json)
  if (p.event_type === 'acl_update') return t('campaign.timeline.aclUpdate')
  if (p.event_type === 'campaign_follow_up') return t('campaign.timeline.followUp')
  const type = followUpTypeOptions.find((item) => item.value === p.interaction_type)?.label
  return `${type || t('domain.auto.c625b7c816')} · ${translateCampaignTimelineText(p.summary || row?.name || '')}`
}
function timelineContent(row) {
  const p = parsePayload(row?.payload_json)
  const content = translateCampaignTimelineText(p.content || p.reason || p.summary || row?.name || '')
  const next = p.next_action ? `${t('domain.auto.9c9db830ea')}${p.next_action}` : ''
  return `${content}${next}`
}
function timelineOperator(row) { return timelineOperatorText(row, users.value, formatTime) }
function translateCampaignTimelineText(value) {
  const text = String(value || '').trim()
  if (!text) return ''
  if (text === LEGACY_ACL_UPDATE_SUMMARY) return t('campaign.timeline.aclUpdate')
  if (text.startsWith(LEGACY_ACTIVITY_SUMMARY_PREFIX)) return `${t('campaign.timeline.followUp')}: ${text.slice(LEGACY_ACTIVITY_SUMMARY_PREFIX.length)}`
  return text
}
function fileCategoryLabel(value) {
  return fileCategoryOptions.find((item) => item.value === value)?.label || t('domain.auto.1a26edf94a')
}
function fileSizeText(value) {
  const size = Number(value || 0)
  if (!size) return t('domain.auto.23165c3fc3')
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${size} B`
}
function inferCampaignFileCategory(file) {
  const name = String(file?.name || '').toLowerCase()
  const type = String(file?.type || '').toLowerCase()
  if (type.startsWith('video/')) return 'video'
  if (name.includes('whitepaper') || name.includes(t('domain.auto.a7761629bb'))) return 'whitepaper'
  if (name.includes('poster') || name.includes(t('domain.auto.e63a92c582'))) return 'poster'
  if (name.includes('quote') || name.includes(t('domain.auto.78208f311a'))) return 'quote'
  if (name.includes('landing') || name.includes(t('domain.auto.1ed8160dcb'))) return 'landing'
  if (type.startsWith('image/')) return 'poster'
  return 'brochure'
}
function sourceForRow(row) {
  const p = parsePayload(row?.payload_json)
  return p.source_channel || p.utm_source || ''
}
function leadContactText(row) {
  const p = parsePayload(row?.payload_json)
  return p.company_name || p.company || p.email || p.phone || '-'
}
function nextActionLabel(row) {
  const action = nextActionType(row)
  return ({
    detail: t('domain.auto.faea8c1db9'),
    create_lead: t('domain.auto.47a1b14404'),
    opportunities: t('domain.auto.efe9f773a9'),
    follow_opportunities: t('domain.auto.1656deb4f7'),
    orders: t('domain.auto.1656deb4f7'),
    review: t('domain.auto.e7ccb58735'),
    chain: t('domain.auto.f3a298ce49')
  }[action] || t('domain.auto.f3a298ce49'))
}
function nextActionType(row) {
  const time = campaignTimeProgress(row)
  if (time.key === 'not_started') return 'detail'
  if (time.key === 'ending_soon') return 'chain'
  if (time.key === 'ended') return 'review'
  const metric = metricFor(row)
  if (!Number(metric.lead_count || 0)) return 'create_lead'
  if (!Number(metric.opportunity_count || 0)) return 'opportunities'
  if (!Number(metric.order_count || 0)) return 'follow_opportunities'
  if (Number(metric.revenue_amount || 0) > 0) return 'review'
  return 'chain'
}
function campaignTimeRange(row) {
  const start = formatDate(row?.start_at)
  const end = formatDate(row?.end_at)
  if (start === '-' && end === '-') return t('domain.auto.4164a0f4bd')
  return `${start} ~ ${end}`
}
function campaignTimeProgress(row) {
  const start = parseDateValue(row?.start_at)
  const end = parseDateValue(row?.end_at)
  const now = new Date()
  if (!start && !end) return { key: 'missing', label: t('domain.auto.4164a0f4bd'), percent: 0, status: undefined }
  if (start && now < start) return { key: 'not_started', label: `${daysBetween(now, start)}${t('domain.auto.edd654bf27')}`, percent: 0, status: undefined }
  if (end && now > end) return { key: 'ended', label: t('domain.auto.20391328c8'), percent: 100, status: 'success' }
  if (start && end && end > start) {
    const percent = Math.min(99, Math.max(1, Math.round(((now.getTime() - start.getTime()) / (end.getTime() - start.getTime())) * 100)))
    const leftDays = daysBetween(now, end)
    if (leftDays <= 3) return { key: 'ending_soon', label: `${leftDays}${t('domain.auto.4ad4c7a016')}`, percent, status: 'warning' }
    return { key: 'running', label: `${t('domain.auto.6f1972e48e')}${percent}%`, percent, status: undefined }
  }
  if (start) return { key: 'running', label: t('domain.auto.6b9afe6255'), percent: 50, status: undefined }
  return { key: 'scheduled_end', label: `${t('domain.auto.bb7de82aa3')}${formatDate(row?.end_at)}`, percent: 0, status: undefined }
}
function daysBetween(from, to) {
  const start = from instanceof Date ? from : new Date(from)
  const end = to instanceof Date ? to : new Date(to)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return 0
  return Math.max(0, Math.ceil((end.getTime() - start.getTime()) / 86400000))
}
function parseDateValue(value) {
  if (!value) return null
  const date = new Date(String(value).replace(' ', 'T'))
  return Number.isNaN(date.getTime()) ? null : date
}
function normalizeDateTimeForInput(value) {
  if (!value) return ''
  const date = parseDateValue(value)
  if (!date) return String(value).slice(0, 19).replace('T', ' ')
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
}
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function visibleColumn(key) { return columnSelections.value.includes(key) }
function loadColumns() {
  const defaults = columnOptions.map((item) => item.key)
  try {
    const stored = JSON.parse(localStorage.getItem('campaign_columns') || '[]')
    const allowed = new Set(defaults)
    const normalized = stored.filter((item) => allowed.has(item))
    for (const key of ['time', 'progress']) {
      if (normalized.length && !normalized.includes(key)) normalized.splice(Math.min(normalized.indexOf('status') + 1 || normalized.length, normalized.length), 0, key)
    }
    return normalized.length ? normalized : defaults
  } catch {
    return defaults
  }
}
function saveColumns() { localStorage.setItem('campaign_columns', JSON.stringify(columnSelections.value)) }
function resetColumns() { columnSelections.value = columnOptions.map((item) => item.key); saveColumns() }
function campaignTypeLabel(objectType, type) {
  if (objectType === 'attribution') return t('domain.auto.782295b896')
  return campaignTypeOptions.find((item) => item.value === type)?.label || type || objectType
}
function sourceLabel(value) {
  const source = String(value || '').trim()
  if (!source) return '-'
  const channel = sourceChannelOptions.find((item) => item.value === source)
  if (channel) return channel.label
  for (const group of utmOptionGroups) {
    const option = group.options.find((item) => item.value === source)
    if (option) return option.label
  }
  return source
}
function statusLabel(value) { return ({ draft: t('domain.auto.0f436818c0'), scheduled: t('domain.auto.6f1b157a55'), running: t('domain.auto.6f1972e48e'), finished: t('domain.auto.20391328c8'), active: t('domain.auto.d4e9ca3dd4'), open: t('domain.auto.256783b7d4'), new: t('domain.auto.ce73a9b731'), converted: t('domain.auto.45d2a4da13'), invalid: t('domain.auto.93736531ce'), closed_won: t('domain.auto.128b1b1447'), archived: t('domain.auto.5cfbea2b76'), draft_order: t('domain.auto.66e0dbde28'), confirmed: t('domain.auto.d9fea67ad2'), completed: t('domain.auto.e99b48a29b') }[value] || value || '-') }
function statusType(value) { return ({ running: 'success', active: 'success', finished: 'info', scheduled: 'primary', draft: 'warning', invalid: 'danger', converted: 'success' }[value] || 'info') }
function formatAmount(amount, currency = defaultBusinessCurrency()) {
  return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency()))
}
function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString()
}
function formatDate(value) {
  if (!value) return '-'
  const date = parseDateValue(value)
  if (!date) return String(value).slice(0, 16).replace('T', ' ')
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
function normalizeDateTimeForAPI(value) {
  if (!value) return ''
  const text = String(value).trim()
  if (!text) return ''
  if (text.includes('T')) return text
  const date = new Date(text.replace(' ', 'T'))
  return Number.isNaN(date.getTime()) ? text : date.toISOString()
}
function ownerName(ownerID) {
  const id = Number(ownerID || 0)
  if (!id) return '-'
  const user = users.value.find((item) => Number(item.id) === id)
  return user ? userDisplayName(user) : `#${id}`
}
function userDisplayName(user) {
  const label = user.name || user.username || user.email || `#${user.id}`
  return user.department_name ? `${label} · ${user.department_name}` : label
}
function percent(value) {
  if (value === undefined || value === null || value === '') return '0%'
  return `${Math.round(Number(value || 0) * 100)}%`
}
function chartWidth(value, max) {
  return `${Math.max(8, Math.round((Number(value || 0) / Number(max || 1)) * 100))}%`
}
function slugify(value) {
  const text = String(value || '').trim().toLowerCase()
  const slug = text.replace(/[^a-z0-9\u4e00-\u9fa5]+/gi, '_').replace(/^_+|_+$/g, '')
  return slug || 'campaign'
}
function errorMessage(error, fallback) {
  return apiErrorMessage(error, fallback)
}
</script>

<style scoped>
:deep(.campaign-dialog-form .el-form-item__label) {
  min-height: 20px;
  height: auto;
  margin-bottom: 6px;
  padding: 0;
  line-height: 1.35;
  white-space: normal;
  overflow-wrap: anywhere;
  text-align: left;
}
</style>
