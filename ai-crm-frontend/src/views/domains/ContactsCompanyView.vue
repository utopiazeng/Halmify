<template>
  <div class="system-page contact-company-page">
    <section ref="tableCardRef" class="system-card">
      <div class="list-card-search">
        <el-form :model="filters" label-width="72px" class="system-search-form">
          <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="searchPlaceholder" clearable /></el-form-item>
          <el-form-item :label="t('domain.auto.62e951a692')">
            <el-select v-model="filters.status" clearable :placeholder="t('domain.auto.dba277df58')">
              <el-option :label="t('domain.auto.d4e9ca3dd4')" value="active" />
              <el-option :label="t('domain.auto.d989e55188')" value="inactive" />
              <el-option :label="t('domain.auto.9c8acd3d30')" value="merged" />
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
          <el-button v-if="canManage" type="primary" plain @click="openCreate">{{ activeType === 'company' ? t('domain.auto.dfc213f3cd') : t('domain.auto.ae11d6a9cf') }}</el-button>
        </div>
        <div class="toolbar-icons">
          <el-tooltip :content="t('domain.auto.38108eaa1d')"><button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage"><Icon icon="ri:refresh-line" /></button></el-tooltip>
        </div>
      </div>

      <el-table :data="rows" class="system-table" row-key="id" v-loading="loading">
        <el-table-column prop="name" :label="activeType === 'company' ? t('domain.auto.94c4852ffb') : t('domain.auto.2425bd4bc1')" min-width="180">
          <template #default="{ row }">
            <div class="record-title">
              <strong>{{ row.name }}</strong>
              <span>{{ payload(row).company || payload(row).industry || row.code || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column v-if="activeType === 'contact'" :label="t('domain.auto.50f7daf7b1')" min-width="170"><template #default="{ row }">{{ contactCompanyName(row) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.4a81eba22d')" min-width="140"><template #default="{ row }">{{ activeType === 'company' ? optionLabel(payload(row).industry, companyIndustryOptions) : (payload(row).title || '-') }}</template></el-table-column>
        <el-table-column :label="activeType === 'company' ? ct('companyPhone') : t('domain.auto.9c01ad0911')" min-width="130"><template #default="{ row }">{{ payload(row).phone || '-' }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.9ed627bcf6')" min-width="170"><template #default="{ row }">{{ payload(row).email || payload(row).website || '-' }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.7fbcb1b0a3')" width="120"><template #default="{ row }">{{ contactRoleGradeLabel(row) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.974d383f36')" width="130"><template #default="{ row }">{{ ownerName(row.owner_id) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.62e951a692')" width="110"><template #default="{ row }"><el-tag :type="row.status === 'merged' ? 'warning' : row.status === 'inactive' ? 'info' : 'success'" effect="light">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('domain.auto.093dea88c9')" min-width="150"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.f3ea6d345e')" width="168" fixed="right">
          <template #default="{ row }">
            <div class="row-actions three" @click.stop>
              <button class="action-btn view" @click="openDetail(row)"><Icon icon="ri:eye-line" /></button>
              <button v-if="canManage" class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
              <button v-if="canManage && row.status === 'inactive'" class="action-btn add" :title="t('domain.auto.3c3a525acc')" @click="restoreOne(row)"><Icon icon="ri:restart-line" /></button>
              <button v-else-if="canManage" class="action-btn danger" :title="t('domain.auto.d989e55188')" @click="inactiveOne(row)"><Icon icon="ri:forbid-2-line" /></button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="system-pagination">
        <span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span>
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
      </div>
    </section>

    <el-drawer v-model="detailDrawer" :size="detailDrawerSize" :title="detailRecord?.name || t('domain.auto.4f55ee1e68')" destroy-on-close>
      <div v-if="detailRecord" class="contact-overview-drawer">
        <section class="contact-identity-card">
          <div class="contact-identity-main">
            <div>
              <div class="contact-title-row">
                <h2>{{ detailRecord.name }}</h2>
                <el-tag :type="detailRecord.status === 'inactive' ? 'info' : detailRecord.status === 'merged' ? 'warning' : 'success'" effect="light">
                  {{ statusLabel(detailRecord.status) }}
                </el-tag>
              </div>
              <p>{{ isCompanyDetail ? (optionLabel(detailPayload.industry, companyIndustryOptions) || t('domain.auto.3b61280829')) : (detailPayload.title || t('domain.auto.a4a310f6bf')) }}</p>
            </div>
            <div class="detail-actions contact-actions">
              <el-button v-if="canManage" @click="openInteraction(detailRecord)">{{ t('domain.auto.f19c3343bd') }}</el-button>
              <el-button v-if="canManage" @click="openEdit(detailRecord)">{{ t('domain.auto.7719bdd192') }}</el-button>
              <el-button v-if="canManage && detailRecord.status === 'inactive'" type="success" plain @click="restoreOne(detailRecord)">{{ t('domain.auto.3c3a525acc') }}</el-button>
              <el-button v-else-if="canManage" type="danger" plain @click="inactiveOne(detailRecord)">{{ t('domain.auto.d989e55188') }}</el-button>
            </div>
          </div>
          <div class="contact-identity-grid">
            <div><span>{{ isCompanyDetail ? t('domain.auto.7c67482ff1') : t('domain.auto.9c01ad0911') }}</span><strong>{{ isCompanyDetail ? (detailPayload.website || '-') : (detailPayload.phone || '-') }}</strong></div>
            <div><span>{{ isCompanyDetail ? ct('companyPhone') : t('domain.auto.9ed627bcf6') }}</span><strong>{{ isCompanyDetail ? (detailPayload.phone || '-') : (detailPayload.email || '-') }}</strong></div>
            <div v-if="isCompanyDetail"><span>{{ t('domain.auto.285cf8097b') }}</span><strong>{{ [detailPayload.headquarters, optionLabel(detailPayload.size, companySizeOptions) || detailPayload.employee_count_range].filter(Boolean).join(' · ') || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(detailRecord.owner_id) }}</strong></div>
            <div><span>{{ t('domain.auto.50f7daf7b1') }}</span><strong>{{ isCompanyDetail ? detailRecord.name : detailCompanyName }}</strong></div>
            <div><span>{{ t('domain.auto.78acf9e064') }}</span><strong>{{ formatTime(contactSummary.last_interaction_at) }}</strong></div>
            <div><span>{{ t('domain.auto.093dea88c9') }}</span><strong>{{ formatTime(detailRecord.updated_at) }}</strong></div>
          </div>
        </section>

        <section class="contact-kpi-grid">
          <div v-for="item in kpiCards" :key="item.key" class="contact-kpi-card">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <small>{{ item.hint }}</small>
          </div>
        </section>

        <el-tabs v-model="detailActiveTab" class="contact-overview-tabs">
          <el-tab-pane :label="t('domain.auto.c122e1758c')" name="overview">
            <div class="contact-overview-grid">
              <section class="contact-overview-section">
                <div class="contact-section-head">
                  <h3>{{ t('domain.auto.8bb166165f') }}</h3>
                </div>
                <div class="summary-metrics">
                  <div><span>{{ t('domain.auto.260617751a') }}</span><strong>{{ Number(contactSummary.active_opportunities || 0) }}</strong></div>
                  <div><span>{{ t('domain.auto.5ac60d781c') }}</span><strong>{{ formatAmount(contactSummary.closed_revenue || 0) }}</strong></div>
                  <div><span>{{ t('domain.auto.75b438b16f') }}</span><strong>{{ formatAmount(contactSummary.open_receivable || 0) }}</strong></div>
                  <div><span>{{ t('domain.auto.6d45caa7fe') }}</span><strong>{{ Number(contactSummary.ticket_processing || 0) }}</strong></div>
                </div>
              </section>
              <section class="contact-overview-section">
                <div class="contact-section-head">
                  <h3>{{ t('domain.auto.1dae2d8032') }}</h3>
                </div>
                <div class="timeline-list compact">
                  <div v-for="item in limited(related.recent_timeline, 6)" :key="`${item.domain}-${item.id}-${item.happened_at}`" class="timeline-row">
                    <span>{{ formatTime(item.happened_at) }}</span>
                    <strong>{{ timelineDomainLabel(item.domain, item.object_type) }} · {{ item.title }}</strong>
                    <small>{{ item.summary || statusLabel(item.status) }}</small>
                    <small class="timeline-operator">{{ timelineOperator(item) }}</small>
                  </div>
                </div>
                <el-empty v-if="!related.recent_timeline.length" :description="t('domain.auto.1118af3208')" />
              </section>
              <section v-if="isCompanyDetail" class="contact-overview-section wide">
                <div class="contact-section-head">
                  <h3>{{ t('domain.auto.a1d3c05d38') }}</h3>
                </div>
                <div class="company-profile-grid">
                  <div v-for="item in companyProfileFields" :key="item.key" class="company-profile-item">
                    <span>{{ item.label }}</span>
                    <strong>{{ formatResearchValue(item.value) || '-' }}</strong>
                  </div>
                </div>
              </section>
            </div>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.c62381d171')" name="company">
            <div class="contact-tab-grid">
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ isCompanyDetail ? t('domain.auto.15581973d3') : t('domain.auto.f2e41d72cf') }}</h3><el-button link type="primary" @click="goDomain(isCompanyDetail ? 'contacts' : 'companies')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(isCompanyDetail ? related.contacts : related.companies, 10)" :key="`relation-${item.id}`" class="overview-item">
                    <div>
                      <strong>{{ item.name }}</strong>
                      <span>{{ payload(item).title || payload(item).industry || payload(item).phone || payload(item).email || payload(item).website || '-' }}</span>
                    </div>
                    <el-button v-if="canManage && !isCompanyDetail && item.object_type === 'company'" size="small" type="danger" link @click.stop="unlinkCompany(item)">{{ t('domain.auto.25a139280f') }}</el-button>
                  </div>
                </div>
                <el-empty v-if="!(isCompanyDetail ? related.contacts : related.companies).length" :description="t('domain.auto.302a237f33')" />
              </section>
              <section v-if="isCompanyDetail" class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.9f3dea450a') }}</h3></div>
                <div class="research-completeness">
                  <el-progress :percentage="researchCompleteness.percentage" />
                  <div class="research-missing">
                    <el-tag v-for="item in researchCompleteness.missing" :key="item" size="small" type="info">{{ item }}</el-tag>
                    <span v-if="!researchCompleteness.missing.length">{{ t('domain.auto.dd41449e60') }}</span>
                  </div>
                </div>
              </section>
            </div>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.6a1ec27547')" name="marketing">
            <div class="contact-tab-grid">
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.9ac5196bb4') }}</h3><el-button link type="primary" @click="goDomain('campaigns')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.campaigns, 10)" :key="`campaign-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ statusLabel(item.status) }} · {{ payload(item).source_channel || payload(item).utm_source || payload(item).channel || '-' }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.campaigns.length" :description="t('domain.auto.559c4db795')" />
              </section>
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.45795e74d2') }}</h3><el-button link type="primary" @click="goDomain('leads')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.leads, 10)" :key="`lead-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ statusLabel(item.status) }} · {{ payload(item).source_channel || payload(item).utm_source || '-' }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.leads.length" :description="t('domain.auto.31eb486d95')" />
              </section>
            </div>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.8bb389d54a')" name="sales">
            <div class="contact-tab-grid">
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.6c0a936ea1') }}</h3><el-button link type="primary" @click="goDomain('opportunities')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.opportunities, 10)" :key="`opp-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ statusLabel(item.status) }} · {{ formatAmount(item.amount, item.currency) }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.opportunities.length" :description="t('domain.auto.306dbaf443')" />
              </section>
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.d93a9092e1') }}</h3><el-button link type="primary" @click="goDomain('deals')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.quotes, 5)" :key="`quote-${item.id}`" class="overview-item">
                    <div><strong>{{ item.code || item.name }}</strong><span>{{ t('domain.auto.d126dd6f0e') }} {{ statusLabel(item.status) }} · {{ formatAmount(item.amount, item.currency) }}</span></div>
                  </div>
                  <div v-for="item in limited(related.deals, 5)" :key="`deal-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ t('domain.auto.e95e4170cd') }} {{ statusLabel(item.status) }} · {{ formatAmount(item.amount, item.currency) }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.quotes.length && !related.deals.length" :description="t('domain.auto.da28038b30')" />
              </section>
              <section class="contact-overview-section wide">
                <div class="contact-section-head"><h3>{{ t('domain.auto.4e2dded77b') }}</h3><el-button link type="primary" @click="goDomain('orders')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.orders, 10)" :key="`order-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ statusLabel(item.status) }} · {{ formatAmount(item.amount, item.currency) }} {{ t('domain.auto.740fb44dab') }} {{ formatAmount(payload(item).receivable_balance || 0, item.currency) }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.orders.length" :description="t('domain.auto.98972e444e')" />
              </section>
            </div>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.9721c5a0e1')" name="service">
            <div class="contact-tab-grid">
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.60ac0b974a') }}</h3><el-button link type="primary" @click="goDomain('projects')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.projects, 10)" :key="`project-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ statusLabel(item.status) }} · {{ payload(item).project_manager || payload(item).stage || '-' }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.projects.length" :description="t('domain.auto.b48cd25620')" />
              </section>
              <section class="contact-overview-section">
                <div class="contact-section-head"><h3>{{ t('domain.auto.84963a014d') }}</h3><el-button link type="primary" @click="goDomain('tickets')">{{ t('domain.auto.f7acefd2d4') }}</el-button></div>
                <div class="overview-list">
                  <div v-for="item in limited(related.tickets, 10)" :key="`ticket-${item.id}`" class="overview-item">
                    <div><strong>{{ item.name }}</strong><span>{{ statusLabel(item.status) }} · {{ payload(item).category || '-' }}</span></div>
                  </div>
                </div>
                <el-empty v-if="!related.tickets.length" :description="t('domain.auto.b6abcb6a67')" />
              </section>
              <section class="contact-overview-section wide">
                <div class="contact-section-head"><h3>{{ t('domain.auto.97c9507761') }}</h3></div>
                <div class="summary-metrics">
                  <div><span>{{ t('domain.auto.6dd05275f7') }}</span><strong>{{ Number(serviceSummary.ticket_last_30d || 0) }}</strong></div>
                  <div><span>{{ t('domain.auto.6d45caa7fe') }}</span><strong>{{ Number(serviceSummary.ticket_processing || 0) }}</strong></div>
                  <div><span>{{ t('domain.auto.fd589af938') }}</span><strong>{{ formatCSAT(serviceSummary.avg_csat) }}</strong></div>
                  <div><span>{{ t('domain.auto.9297c3859f') }}</span><strong>{{ Number(contactSummary.project_count || 0) }}</strong></div>
                </div>
              </section>
            </div>
          </el-tab-pane>

          <el-tab-pane :label="t('domain.auto.b938829a98')" name="timeline">
            <div class="timeline-list">
              <div v-for="item in related.interactions" :key="`interaction-${item.id}`" class="timeline-row">
                <span>{{ formatTime(item.created_at) }}</span>
                <strong>{{ item.name }}</strong>
                <small>{{ payload(item).summary || payload(item).next_action || '-' }}</small>
                <small class="timeline-operator">{{ timelineOperator(item) }}</small>
              </div>
            </div>
            <el-empty v-if="!related.interactions.length" :description="t('domain.auto.96b919608c')" />
          </el-tab-pane>

          <el-tab-pane
            v-if="showManufacturingDetailTab"
            :label="t('industry.manufacturing.customer360.detailTab')"
            name="manufacturing"
          >
            <mfg-customer360-panel
              :customer-id="Number(detailRecord.id || 0)"
              :customer-name="detailRecord.name || ''"
              embedded
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="contactFormTitle" width="min(1180px, 96vw)" destroy-on-close>
      <el-alert v-if="duplicateCandidates.length" class="contact-duplicate-alert" type="warning" show-icon :closable="false">
        <template #title>{{ t('domain.auto.523e40a074') }} {{ duplicateCandidates.length }} {{ t('domain.auto.3b71004489') }}</template>
        <div class="duplicate-list">
          <div v-for="item in duplicateCandidates" :key="item.id" class="duplicate-item">
            <span>{{ item.name }} · {{ payload(item).phone || payload(item).email || payload(item).website || statusLabel(item.status) }}</span>
            <button class="action-btn view" @click="openDetail(item)"><Icon icon="ri:eye-line" /></button>
          </div>
        </div>
      </el-alert>
      <el-form :model="form" label-width="124px" class="lead-dialog-form" :class="{ 'company-dialog-form': activeType === 'company' }">
        <template v-if="activeType === 'contact'">
          <el-form-item :label="t('domain.auto.be4c2616b1')" required><el-input v-model="form.name" /></el-form-item>
          <el-form-item :label="t('domain.auto.a2a92e50a1')"><el-input v-model="form.payload.title" /></el-form-item>
          <el-form-item :label="t('domain.auto.9c01ad0911')"><el-input v-model="form.payload.phone" /></el-form-item>
          <el-form-item :label="t('domain.auto.9ed627bcf6')"><el-input v-model="form.payload.email" /></el-form-item>
          <el-form-item :label="t('domain.auto.f2e41d72cf')">
            <el-select
              v-model="form.payload.company"
              class="full-input"
              filterable
              allow-create
              default-first-option
              clearable
              reserve-keyword
              :loading="companyOptionsLoading"
              :placeholder="t('domain.auto.4854406f4e')"
              @focus="loadCompanyOptions()"
              @change="handleContactCompanyChange"
            >
              <el-option v-for="item in companyOptions" :key="item.id" :label="companyOptionLabel(item)" :value="item.name">
                <div class="company-option">
                  <strong>{{ item.name }}</strong>
                  <span>{{ payload(item).industry || payload(item).website || t('domain.auto.5d80806d7f') }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item :label="t('domain.auto.f33c7dc8a6')"><el-select v-model="form.payload.decision_role"><el-option :label="t('domain.auto.9bed2d370d')" value="decision_maker" /><el-option :label="t('domain.auto.ff5691b83d')" value="influencer" /><el-option :label="t('domain.auto.4a7f444ac5')" value="user" /></el-select></el-form-item>
        </template>
        <template v-else>
          <el-tabs v-model="companyFormTab" class="company-form-tabs">
            <el-tab-pane :label="t('domain.auto.41654e0268')" name="base">
              <div v-if="companyFormTab === 'base'" class="company-form-grid">
                <el-form-item :label="t('domain.auto.e1272432a4')" required><el-input v-model="form.name" /></el-form-item>
                <el-form-item :label="t('domain.auto.7c67482ff1')">
                  <div class="company-website-row">
                    <el-input v-model="form.payload.website" placeholder="https://example.com" />
                  </div>
                </el-form-item>
                <el-form-item :label="t('domain.auto.b313f81655')">
                  <el-select v-model="form.payload.industry" class="full-input" filterable allow-create default-first-option clearable :placeholder="ct('selectOrInputIndustry')">
                    <el-option v-for="item in companyIndustryOptions" :key="item.value" :label="ct(item.labelKey)" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('domain.auto.675a254004')">
                  <el-select v-model="form.payload.size" class="full-input" filterable allow-create default-first-option clearable :placeholder="ct('selectOrInputSize')">
                    <el-option v-for="item in companySizeOptions" :key="item.value" :label="ct(item.labelKey)" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="ct('companyPhone')"><el-input v-model="form.payload.phone" :placeholder="ct('companyPhonePlaceholder')" /></el-form-item>
                <el-form-item :label="t('domain.auto.2425bd4bc1')">
                  <el-select
                    v-model="form.payload.contact_name"
                    class="full-input"
                    filterable
                    remote
                    clearable
                    :remote-method="loadContactOptions"
                    :loading="contactOptionsLoading"
                    :placeholder="ct('selectOrInputContact')"
                    @focus="loadContactOptions()"
                    @change="handleCompanyContactChange"
                  >
                    <el-option v-for="item in contactOptions" :key="item.id" :label="contactOptionLabel(item)" :value="item.name">
                      <div class="company-option">
                        <strong>{{ item.name }}</strong>
                        <span>{{ payload(item).title || payload(item).phone || payload(item).email || '-' }}</span>
                      </div>
                    </el-option>
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('domain.auto.c3785b2f81')">
                  <el-select v-model="form.payload.customer_type" class="full-input" filterable allow-create default-first-option clearable :placeholder="ct('selectOrInputCustomerType')">
                    <el-option v-for="item in companyCustomerTypeOptions" :key="item.value" :label="ct(item.labelKey)" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('domain.auto.5c42c048ea')">
                  <el-select v-model="form.payload.grade" class="full-input" filterable allow-create default-first-option clearable :placeholder="ct('selectOrInputGrade')">
                    <el-option v-for="item in companyGradeOptions" :key="item.value" :label="ct(item.labelKey)" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('domain.auto.e0361480e3')" class="wide-field"><el-input v-model="form.payload.note" type="textarea" :rows="3" /></el-form-item>
              </div>
            </el-tab-pane>
            <el-tab-pane :label="t('domain.auto.29eeec6537')" name="profile">
              <div v-if="companyFormTab === 'profile'" class="company-form-grid">
                <el-form-item :label="t('domain.auto.0b04e7e574')"><el-input v-model="form.payload.headquarters" /></el-form-item>
                <el-form-item :label="t('domain.auto.fd48211383')"><el-input v-model="form.payload.founded_year" /></el-form-item>
                <el-form-item :label="t('domain.auto.518beb072b')">
                  <div class="company-revenue-row">
                    <el-select v-model="form.payload.revenue_currency" class="company-revenue-currency" filterable>
                      <el-option v-for="item in currencyOptions" :key="item.value" :label="`${currencySymbol(item.value)} ${item.label}`" :value="item.value" />
                    </el-select>
                    <el-input v-model="form.payload.revenue_range" :placeholder="ct('revenueRangePlaceholder')" />
                  </div>
                </el-form-item>
                <el-form-item :label="t('domain.auto.6bc2823971')" class="wide-field"><el-input v-model="form.payload.products_services_text" :placeholder="t('domain.auto.fccfa9ffb1')" /></el-form-item>
                <el-form-item :label="t('domain.auto.8af107bd8f')" class="wide-field"><el-input v-model="form.payload.target_customers_text" :placeholder="t('domain.auto.fccfa9ffb1')" /></el-form-item>
                <el-form-item :label="t('domain.auto.0dae4ae6dc')"><el-input v-model="form.payload.business_model" /></el-form-item>
                <el-form-item :label="t('domain.auto.ab576ae358')"><el-input v-model="form.payload.markets_text" :placeholder="t('domain.auto.61288a6364')" /></el-form-item>
                <el-form-item :label="t('domain.auto.02e3e230cb')" class="wide-field"><el-input v-model="form.payload.key_people_text" :placeholder="t('domain.auto.eab17747fe')" /></el-form-item>
              </div>
            </el-tab-pane>
            <el-tab-pane :label="t('domain.auto.4f15b35bca')" name="insight">
              <div v-if="companyFormTab === 'insight'" class="company-form-grid">
                <el-form-item :label="t('domain.auto.fa12045bfe')" class="wide-field"><el-input v-model="form.payload.buying_signals_text" :placeholder="t('domain.auto.fccfa9ffb1')" /></el-form-item>
                <el-form-item :label="t('domain.auto.a69053af14')" class="wide-field"><el-input v-model="form.payload.risk_signals_text" :placeholder="t('domain.auto.fccfa9ffb1')" /></el-form-item>
                <el-form-item :label="t('domain.auto.fe3901e5d5')" class="wide-field"><el-input v-model="form.payload.social_links_text" :placeholder="t('domain.auto.2db3b84f76')" /></el-form-item>
                <el-form-item :label="t('domain.auto.a2c6f15e72')" class="wide-field"><el-input v-model="form.payload.tech_stack_text" :placeholder="t('domain.auto.fccfa9ffb1')" /></el-form-item>
              </div>
            </el-tab-pane>
            <el-tab-pane :label="t('domain.auto.7aa2dc547c')" name="detail">
              <el-form-item v-if="companyFormTab === 'detail'" :label="t('domain.auto.cdf61ecdf5')" class="wide-field">
                <div class="rich-editor">
                  <div class="rich-editor-toolbar">
                    <el-button size="small" @click="wrapSelectedDetailText('**', '**')">{{ t('domain.auto.f670b2adc0') }}</el-button>
                    <el-button size="small" @click="insertDetailTemplate">{{ t('domain.auto.f96b00842d') }}</el-button>
                  </div>
                  <el-input ref="companyDetailEditorRef" v-model="form.payload.description" type="textarea" :rows="12" :placeholder="t('domain.auto.2c8ea2174c')" />
                </div>
              </el-form-item>
            </el-tab-pane>
          </el-tabs>
        </template>
        <el-form-item v-if="activeType === 'contact'" :label="t('domain.auto.974d383f36')">
          <el-select v-model="form.owner_id" class="full-input" clearable filterable :placeholder="t('domain.auto.8608d655e6')">
            <el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="activeType === 'contact'" :label="t('domain.auto.e0361480e3')" class="wide-field"><el-input v-model="form.payload.note" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('domain.auto.09cbc97ae2') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="researchDialog" :title="t('domain.auto.3cfcd6c434')" width="760px" destroy-on-close>
      <el-form :model="researchForm" label-width="96px" class="lead-dialog-form">
        <el-form-item :label="t('domain.auto.94c4852ffb')"><el-input v-model="researchForm.company_name" /></el-form-item>
        <el-form-item :label="t('domain.auto.7c67482ff1')"><el-input v-model="researchForm.website" /></el-form-item>
      </el-form>
      <div class="detail-actions">
        <el-button type="primary" :loading="researchLoading" @click="submitResearchJob">{{ t('domain.auto.e1777b14cd') }}</el-button>
        <el-tag v-if="researchJob?.status" :type="researchStatusType(researchJob.status)">{{ researchStatusLabel(researchJob.status) }}</el-tag>
        <span v-if="researchJob?.usage?.total_tokens" class="research-usage">{{ researchJob.usage.total_tokens }} tokens · {{ researchJob.usage.cost_amount }} {{ researchJob.usage.currency }}</span>
      </div>
      <el-alert v-if="researchJob?.error_message" type="error" show-icon :closable="false" :title="researchJob.error_message" />
      <template v-if="researchFormFillKeys.length">
        <el-divider>{{ t('domain.auto.6718bfd817') }}</el-divider>
        <div class="research-profile-grid">
          <div v-for="item in researchProfileCards" :key="item.key" class="research-profile-card">
            <span>{{ item.label }}</span>
            <strong>{{ formatResearchValue(item.value) || '-' }}</strong>
          </div>
        </div>
        <div v-if="researchStrategyItems.length" class="research-list-block">
          <strong>{{ t('domain.auto.c79ada01a2') }}</strong>
          <ul>
            <li v-for="item in researchStrategyItems" :key="item">{{ item }}</li>
          </ul>
        </div>
        <el-divider>{{ t('domain.auto.0313f49772') }}</el-divider>
        <el-table :data="researchSuggestionRows" class="system-table" max-height="320">
          <el-table-column width="48">
            <template #default="{ row }"><el-checkbox :model-value="researchSelectedFields.includes(row.key)" @change="toggleResearchField(row.key, $event)" /></template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.77a49f2c38')" prop="label" width="120" />
          <el-table-column :label="t('domain.auto.8b0db67871')" min-width="180"><template #default="{ row }">{{ formatResearchValue(row.current) || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.8d736aa35d')" min-width="220"><template #default="{ row }">{{ formatResearchValue(row.value) || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.b78c2dc2e2')" width="110"><template #default="{ row }">{{ Math.round(Number(row.confidence || 0) * 100) }}%</template></el-table-column>
          <el-table-column :label="t('domain.auto.d984d98f05')" width="110"><template #default="{ row }">{{ researchBasisLabel(row.basis) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.68d4cdfc09')" min-width="220"><template #default="{ row }">{{ row.evidence || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.c63f79e636')" min-width="180"><template #default="{ row }">{{ (row.source_urls || row.sources || []).slice(0, 2).join('、') || '-' }}</template></el-table-column>
        </el-table>
        <el-divider>{{ t('domain.auto.5bae9cb37d') }}</el-divider>
        <div class="research-list-block">
          <p>{{ researchJob.insights?.summary || '-' }}</p>
          <strong>{{ t('domain.auto.947f51393f') }}</strong>
          <ul>
            <li v-for="item in researchJob.insights?.recommended_next_actions || []" :key="item">{{ item }}</li>
          </ul>
        </div>
        <el-divider>{{ t('domain.auto.a70e10299e') }}</el-divider>
        <div class="related-list research-source-list">
          <div v-for="item in researchJob.sources || []" :key="item.url || item.title" class="related-item">
            <strong>{{ item.title || item.url || t('domain.auto.c63f79e636') }}</strong>
            <span>{{ item.url || '-' }} · {{ item.status || 'ok' }} · {{ formatTime(item.fetched_at) || '-' }}</span>
          </div>
        </div>
      </template>
      <template #footer>
        <el-button @click="researchDialog = false">{{ t('domain.auto.6c14bd7f6f') }}</el-button>
        <el-button type="primary" :loading="researchApplyLoading" :disabled="!researchSelectedFields.length || researchJob?.status !== 'succeeded'" @click="applyResearchResult">{{ t('domain.auto.593380b558') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="interactionDialog" :title="t('domain.auto.f19c3343bd')" width="560px" destroy-on-close>
      <el-form :model="interactionForm" label-width="86px">
        <el-form-item :label="t('domain.auto.0c60bea87c')">
          <el-select v-model="interactionForm.type">
            <el-option :label="t('domain.auto.f39c6b629f')" value="call" />
            <el-option :label="t('domain.auto.1c8e464184')" value="email" />
            <el-option :label="t('domain.auto.33e9651895')" value="meeting" />
            <el-option :label="t('domain.auto.1f3cb01e46')" value="social" />
            <el-option :label="t('domain.auto.e0361480e3')" value="note" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.46d4c1b4e4')" required><el-input v-model="interactionForm.summary" type="textarea" :rows="4" /></el-form-item>
        <el-form-item :label="t('domain.auto.ea0ef2ae72')"><el-input v-model="interactionForm.next_action" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="interactionDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitInteraction">{{ t('domain.auto.09cbc97ae2') }}</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { i18nState, t } from '../../i18n'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'
import { canManageDomain } from '../../permissions'
import { currencyOptions, currencySymbols, formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { timelineOperatorText } from '../../utils/timeline'
import MfgCustomer360Panel from '../industry/components/MfgCustomer360Panel.vue'
import { apiErrorMessage } from '../../utils/api-error'

const route = useRoute()
const router = useRouter()
const activeType = ref(route.meta.objectType || 'contact')
const rows = ref([])
const users = ref([])
const loading = ref(false)
const detailDrawer = ref(false)
const detailActiveTab = ref('overview')
const formDialog = ref(false)
const interactionDialog = ref(false)
const researchDialog = ref(false)
const researchLoading = ref(false)
const researchApplyLoading = ref(false)
const companyDetailEditorRef = ref(null)
const editingId = ref(null)
const detailRecord = ref(null)
const interactionTarget = ref(null)
const duplicateCandidates = ref([])
const companyOptions = ref([])
const companyOptionsLoading = ref(false)
const contactOptions = ref([])
const contactOptionsLoading = ref(false)
const filters = reactive({ keyword: '', status: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const related = reactive({ contacts: [], companies: [], campaigns: [], leads: [], opportunities: [], quotes: [], deals: [], orders: [], projects: [], tickets: [], interactions: [], service: {}, deal_summary: {}, summary: {}, recent_timeline: [] })
const form = reactive(defaultForm())
const interactionForm = reactive({ type: 'call', summary: '', next_action: '' })
const researchForm = reactive({ company_id: 0, company_name: '', website: '', region: '', industry_hint: '', note: '', mode: 'deep', source: 'form' })
const researchJob = ref(null)
const researchHistory = ref([])
const researchSelectedFields = ref([])
const researchPollTimer = ref(null)
const companyFormTab = ref('base')
const industryManifest = ref({ packages: [], pageTabs: {} })
const companyIndustryOptions = [
  { value: 'software_saas', labelKey: 'contactsCompany.industry.softwareSaas' },
  { value: 'artificial_intelligence', labelKey: 'contactsCompany.industry.ai' },
  { value: 'cloud_data', labelKey: 'contactsCompany.industry.cloudData' },
  { value: 'cybersecurity', labelKey: 'contactsCompany.industry.cybersecurity' },
  { value: 'ecommerce', labelKey: 'contactsCompany.industry.ecommerce' },
  { value: 'cross_border_ecommerce', labelKey: 'contactsCompany.industry.crossBorderEcommerce' },
  { value: 'foreign_trade_services', labelKey: 'contactsCompany.industry.foreignTradeServices' },
  { value: 'manufacturing', labelKey: 'contactsCompany.industry.manufacturing' },
  { value: 'industrial_equipment', labelKey: 'contactsCompany.industry.industrialEquipment' },
  { value: 'electronics_semiconductor', labelKey: 'contactsCompany.industry.electronicsSemiconductor' },
  { value: 'automotive_parts', labelKey: 'contactsCompany.industry.automotiveParts' },
  { value: 'new_energy', labelKey: 'contactsCompany.industry.newEnergy' },
  { value: 'energy_chemical', labelKey: 'contactsCompany.industry.energyChemical' },
  { value: 'construction_engineering', labelKey: 'contactsCompany.industry.constructionEngineering' },
  { value: 'real_estate', labelKey: 'contactsCompany.industry.realEstate' },
  { value: 'logistics_supply_chain', labelKey: 'contactsCompany.industry.logisticsSupplyChain' },
  { value: 'transportation', labelKey: 'contactsCompany.industry.transportation' },
  { value: 'wholesale_retail', labelKey: 'contactsCompany.industry.wholesaleRetail' },
  { value: 'consumer_goods', labelKey: 'contactsCompany.industry.consumerGoods' },
  { value: 'food_beverage', labelKey: 'contactsCompany.industry.foodBeverage' },
  { value: 'apparel_textile', labelKey: 'contactsCompany.industry.apparelTextile' },
  { value: 'financial_services', labelKey: 'contactsCompany.industry.financialServices' },
  { value: 'insurance', labelKey: 'contactsCompany.industry.insurance' },
  { value: 'business_services', labelKey: 'contactsCompany.industry.businessServices' },
  { value: 'professional_services', labelKey: 'contactsCompany.industry.professionalServices' },
  { value: 'consulting', labelKey: 'contactsCompany.industry.consulting' },
  { value: 'legal_tax', labelKey: 'contactsCompany.industry.legalTax' },
  { value: 'advertising_media', labelKey: 'contactsCompany.industry.advertisingMedia' },
  { value: 'culture_entertainment', labelKey: 'contactsCompany.industry.cultureEntertainment' },
  { value: 'education_training', labelKey: 'contactsCompany.industry.educationTraining' },
  { value: 'healthcare', labelKey: 'contactsCompany.industry.healthcare' },
  { value: 'biopharma', labelKey: 'contactsCompany.industry.biopharma' },
  { value: 'agriculture', labelKey: 'contactsCompany.industry.agriculture' },
  { value: 'travel_hospitality', labelKey: 'contactsCompany.industry.travelHospitality' },
  { value: 'public_utilities', labelKey: 'contactsCompany.industry.publicUtilities' },
  { value: 'government_nonprofit', labelKey: 'contactsCompany.industry.governmentNonprofit' },
  { value: 'other', labelKey: 'contactsCompany.industry.other' }
]
const companySizeOptions = [
  { value: '1-10', labelKey: 'contactsCompany.size.1_10' },
  { value: '11-50', labelKey: 'contactsCompany.size.11_50' },
  { value: '51-100', labelKey: 'contactsCompany.size.51_100' },
  { value: '101-500', labelKey: 'contactsCompany.size.101_500' },
  { value: '501-1000', labelKey: 'contactsCompany.size.501_1000' },
  { value: '1001-5000', labelKey: 'contactsCompany.size.1001_5000' },
  { value: '5000+', labelKey: 'contactsCompany.size.5000_plus' }
]
const companyGradeOptions = [
  { value: 'vip', labelKey: 'contactsCompany.grade.vip' },
  { value: 'important', labelKey: 'contactsCompany.grade.important' },
  { value: 'normal', labelKey: 'contactsCompany.grade.normal' },
  { value: 'potential', labelKey: 'contactsCompany.grade.potential' },
  { value: 'low_priority', labelKey: 'contactsCompany.grade.lowPriority' }
]
const companyCustomerTypeOptions = [
  { value: 'prospect', labelKey: 'contactsCompany.customerType.prospect' },
  { value: 'trial_customer', labelKey: 'contactsCompany.customerType.trialCustomer' },
  { value: 'paying_customer', labelKey: 'contactsCompany.customerType.payingCustomer' },
  { value: 'key_account', labelKey: 'contactsCompany.customerType.keyAccount' },
  { value: 'strategic_account', labelKey: 'contactsCompany.customerType.strategicAccount' },
  { value: 'channel_partner', labelKey: 'contactsCompany.customerType.channelPartner' },
  { value: 'dealer_agent', labelKey: 'contactsCompany.customerType.dealerAgent' },
  { value: 'supplier', labelKey: 'contactsCompany.customerType.supplier' },
  { value: 'business_partner', labelKey: 'contactsCompany.customerType.businessPartner' },
  { value: 'competitor', labelKey: 'contactsCompany.customerType.competitor' },
  { value: 'other', labelKey: 'contactsCompany.customerType.other' }
]
const contactsCompanyI18n = {
  'zh-CN': {
    companyPhone: t('domain.auto.f66b1b3e74'),
    companyPhonePlaceholder: t('domain.auto.c9c7370bd6'),
    selectOrInputIndustry: t('domain.auto.5c5ec709d4'),
    selectOrInputSize: t('domain.auto.6c666d2792'),
    selectOrInputGrade: t('domain.auto.e5fd987c11'),
    selectOrInputCustomerType: '选择或输入客户类型',
    selectOrInputContact: '选择或输入联系人',
    revenueRangePlaceholder: '例如：100万-500万',
    'contactsCompany.industry.softwareSaas': t('domain.auto.53f8c77e4d'),
    'contactsCompany.industry.ai': t('domain.auto.63fe855a00'),
    'contactsCompany.industry.cloudData': t('domain.auto.768868fdcc'),
    'contactsCompany.industry.cybersecurity': t('domain.auto.38c84c8015'),
    'contactsCompany.industry.ecommerce': t('domain.auto.ef738f8b74'),
    'contactsCompany.industry.crossBorderEcommerce': t('domain.auto.fb172be33c'),
    'contactsCompany.industry.foreignTradeServices': t('domain.auto.41a15b0d6e'),
    'contactsCompany.industry.manufacturing': t('domain.auto.8362b8c309'),
    'contactsCompany.industry.industrialEquipment': t('domain.auto.21145170b8'),
    'contactsCompany.industry.electronicsSemiconductor': t('domain.auto.d001c330fe'),
    'contactsCompany.industry.automotiveParts': t('domain.auto.a4964f0f97'),
    'contactsCompany.industry.newEnergy': t('domain.auto.f95e2fc9d2'),
    'contactsCompany.industry.energyChemical': t('domain.auto.5b867b928b'),
    'contactsCompany.industry.constructionEngineering': t('domain.auto.a1e1387a0e'),
    'contactsCompany.industry.realEstate': t('domain.auto.75a5e1bc51'),
    'contactsCompany.industry.logisticsSupplyChain': t('domain.auto.c10473007a'),
    'contactsCompany.industry.transportation': t('domain.auto.9dff9ecef1'),
    'contactsCompany.industry.wholesaleRetail': t('domain.auto.997d925340'),
    'contactsCompany.industry.consumerGoods': t('domain.auto.f3c9a86040'),
    'contactsCompany.industry.foodBeverage': t('domain.auto.765d72d1e4'),
    'contactsCompany.industry.apparelTextile': t('domain.auto.ebd09d4e34'),
    'contactsCompany.industry.financialServices': t('domain.auto.104748401b'),
    'contactsCompany.industry.insurance': t('domain.auto.be85a8a123'),
    'contactsCompany.industry.businessServices': t('domain.auto.b8e19f721e'),
    'contactsCompany.industry.professionalServices': t('domain.auto.a024155f2a'),
    'contactsCompany.industry.consulting': t('domain.auto.8419f77a65'),
    'contactsCompany.industry.legalTax': t('domain.auto.5210a24baf'),
    'contactsCompany.industry.advertisingMedia': t('domain.auto.f564c9fa71'),
    'contactsCompany.industry.cultureEntertainment': t('domain.auto.0536282b6d'),
    'contactsCompany.industry.educationTraining': t('domain.auto.6a2ee4312e'),
    'contactsCompany.industry.healthcare': t('domain.auto.6bab23385c'),
    'contactsCompany.industry.biopharma': t('domain.auto.12fa59e952'),
    'contactsCompany.industry.agriculture': t('domain.auto.6f9019a1eb'),
    'contactsCompany.industry.travelHospitality': t('domain.auto.f781114cad'),
    'contactsCompany.industry.publicUtilities': t('domain.auto.c8fc23f687'),
    'contactsCompany.industry.governmentNonprofit': t('domain.auto.71b6d3b344'),
    'contactsCompany.industry.other': t('domain.auto.1a26edf94a'),
    'contactsCompany.size.1_10': t('domain.auto.3810297702'),
    'contactsCompany.size.11_50': t('domain.auto.1a291e5e18'),
    'contactsCompany.size.51_100': t('domain.auto.758124b853'),
    'contactsCompany.size.101_500': t('domain.auto.42843ed29c'),
    'contactsCompany.size.501_1000': t('domain.auto.eb1e694d77'),
    'contactsCompany.size.1001_5000': t('domain.auto.c5d8351256'),
    'contactsCompany.size.5000_plus': t('domain.auto.01f6102d62'),
    'contactsCompany.grade.vip': 'VIP',
    'contactsCompany.grade.important': t('domain.auto.b7f4670752'),
    'contactsCompany.grade.normal': t('domain.auto.7cda072d45'),
    'contactsCompany.grade.potential': t('domain.auto.8ed3458bb1'),
    'contactsCompany.grade.lowPriority': t('domain.auto.2217b22f64'),
    'contactsCompany.customerType.prospect': '潜在客户',
    'contactsCompany.customerType.trialCustomer': '试用客户',
    'contactsCompany.customerType.payingCustomer': '成交客户',
    'contactsCompany.customerType.keyAccount': '重点客户',
    'contactsCompany.customerType.strategicAccount': '战略客户',
    'contactsCompany.customerType.channelPartner': '渠道伙伴',
    'contactsCompany.customerType.dealerAgent': '经销商/代理商',
    'contactsCompany.customerType.supplier': '供应商',
    'contactsCompany.customerType.businessPartner': '合作伙伴',
    'contactsCompany.customerType.competitor': '竞争对手',
    'contactsCompany.customerType.other': '其他'
  },
  'zh-TW': {
    companyPhone: t('domain.auto.e404d66200'),
    companyPhonePlaceholder: t('domain.auto.2d73285d4f'),
    selectOrInputIndustry: t('domain.auto.29c5136cd3'),
    selectOrInputSize: t('domain.auto.2d64c64611'),
    selectOrInputGrade: t('domain.auto.b5ecddbfac'),
    selectOrInputCustomerType: '選擇或輸入客戶類型',
    selectOrInputContact: '選擇或輸入聯絡人',
    revenueRangePlaceholder: '例如：100萬-500萬',
    'contactsCompany.industry.softwareSaas': t('domain.auto.da917b0b89'),
    'contactsCompany.industry.ai': t('domain.auto.76abc8344f'),
    'contactsCompany.industry.cloudData': t('domain.auto.556aab2b43'),
    'contactsCompany.industry.cybersecurity': t('domain.auto.b42648a6e5'),
    'contactsCompany.industry.ecommerce': t('domain.auto.39dd39b118'),
    'contactsCompany.industry.crossBorderEcommerce': t('domain.auto.10d179b412'),
    'contactsCompany.industry.foreignTradeServices': t('domain.auto.893e7321ea'),
    'contactsCompany.industry.manufacturing': t('domain.auto.95593bf47f'),
    'contactsCompany.industry.industrialEquipment': t('domain.auto.52da77407d'),
    'contactsCompany.industry.electronicsSemiconductor': t('domain.auto.fac3347250'),
    'contactsCompany.industry.automotiveParts': t('domain.auto.8e75918ea4'),
    'contactsCompany.industry.newEnergy': t('domain.auto.f95e2fc9d2'),
    'contactsCompany.industry.energyChemical': t('domain.auto.5b867b928b'),
    'contactsCompany.industry.constructionEngineering': t('domain.auto.4ba2ffce24'),
    'contactsCompany.industry.realEstate': t('domain.auto.81e39c1731'),
    'contactsCompany.industry.logisticsSupplyChain': t('domain.auto.75e21c49c6'),
    'contactsCompany.industry.transportation': t('domain.auto.0cc84dcd9a'),
    'contactsCompany.industry.wholesaleRetail': t('domain.auto.81f2df17b8'),
    'contactsCompany.industry.consumerGoods': t('domain.auto.61029d2051'),
    'contactsCompany.industry.foodBeverage': t('domain.auto.393c316131'),
    'contactsCompany.industry.apparelTextile': t('domain.auto.78bbde5093'),
    'contactsCompany.industry.financialServices': t('domain.auto.cd169932b8'),
    'contactsCompany.industry.insurance': t('domain.auto.808d27c87d'),
    'contactsCompany.industry.businessServices': t('domain.auto.d6f1e9cd12'),
    'contactsCompany.industry.professionalServices': t('domain.auto.4bbd674092'),
    'contactsCompany.industry.consulting': t('domain.auto.2a162820c6'),
    'contactsCompany.industry.legalTax': t('domain.auto.fd888d7053'),
    'contactsCompany.industry.advertisingMedia': t('domain.auto.c355e9c767'),
    'contactsCompany.industry.cultureEntertainment': t('domain.auto.5e0eb4e123'),
    'contactsCompany.industry.educationTraining': t('domain.auto.6ef6cd46c6'),
    'contactsCompany.industry.healthcare': t('domain.auto.0ab1c8405b'),
    'contactsCompany.industry.biopharma': t('domain.auto.4b1e0fdbe8'),
    'contactsCompany.industry.agriculture': t('domain.auto.c430220bb7'),
    'contactsCompany.industry.travelHospitality': t('domain.auto.f0b20b6a3b'),
    'contactsCompany.industry.publicUtilities': t('domain.auto.875a13dd17'),
    'contactsCompany.industry.governmentNonprofit': t('domain.auto.af4e070841'),
    'contactsCompany.industry.other': t('domain.auto.1a26edf94a'),
    'contactsCompany.size.1_10': t('domain.auto.3810297702'),
    'contactsCompany.size.11_50': t('domain.auto.1a291e5e18'),
    'contactsCompany.size.51_100': t('domain.auto.758124b853'),
    'contactsCompany.size.101_500': t('domain.auto.42843ed29c'),
    'contactsCompany.size.501_1000': t('domain.auto.eb1e694d77'),
    'contactsCompany.size.1001_5000': t('domain.auto.c5d8351256'),
    'contactsCompany.size.5000_plus': t('domain.auto.01f6102d62'),
    'contactsCompany.grade.vip': 'VIP',
    'contactsCompany.grade.important': t('domain.auto.b7f4670752'),
    'contactsCompany.grade.normal': t('domain.auto.7cda072d45'),
    'contactsCompany.grade.potential': t('domain.auto.313072f90d'),
    'contactsCompany.grade.lowPriority': t('domain.auto.fb0611c9c2'),
    'contactsCompany.customerType.prospect': '潛在客戶',
    'contactsCompany.customerType.trialCustomer': '試用客戶',
    'contactsCompany.customerType.payingCustomer': '成交客戶',
    'contactsCompany.customerType.keyAccount': '重點客戶',
    'contactsCompany.customerType.strategicAccount': '戰略客戶',
    'contactsCompany.customerType.channelPartner': '渠道夥伴',
    'contactsCompany.customerType.dealerAgent': '經銷商/代理商',
    'contactsCompany.customerType.supplier': '供應商',
    'contactsCompany.customerType.businessPartner': '合作夥伴',
    'contactsCompany.customerType.competitor': '競爭對手',
    'contactsCompany.customerType.other': '其他'
  },
  'en-US': {
    companyPhone: 'Company Phone',
    companyPhonePlaceholder: 'Landline, switchboard, or company phone',
    selectOrInputIndustry: 'Select or enter industry',
    selectOrInputSize: 'Select or enter company size',
    selectOrInputGrade: 'Select or enter customer grade',
    selectOrInputCustomerType: 'Select or enter customer type',
    selectOrInputContact: 'Select or enter contact',
    revenueRangePlaceholder: 'e.g. 1M-5M',
    'contactsCompany.industry.softwareSaas': 'Software/SaaS',
    'contactsCompany.industry.ai': 'Artificial Intelligence',
    'contactsCompany.industry.cloudData': 'Cloud/Data Services',
    'contactsCompany.industry.cybersecurity': 'Cybersecurity',
    'contactsCompany.industry.ecommerce': 'E-commerce',
    'contactsCompany.industry.crossBorderEcommerce': 'Cross-border E-commerce',
    'contactsCompany.industry.foreignTradeServices': 'Foreign Trade Services',
    'contactsCompany.industry.manufacturing': 'Manufacturing',
    'contactsCompany.industry.industrialEquipment': 'Industrial Equipment',
    'contactsCompany.industry.electronicsSemiconductor': 'Electronics/Semiconductor',
    'contactsCompany.industry.automotiveParts': 'Automotive/Parts',
    'contactsCompany.industry.newEnergy': 'New Energy',
    'contactsCompany.industry.energyChemical': 'Energy/Chemicals',
    'contactsCompany.industry.constructionEngineering': 'Construction/Engineering',
    'contactsCompany.industry.realEstate': 'Real Estate',
    'contactsCompany.industry.logisticsSupplyChain': 'Logistics/Supply Chain',
    'contactsCompany.industry.transportation': 'Transportation',
    'contactsCompany.industry.wholesaleRetail': 'Wholesale/Retail',
    'contactsCompany.industry.consumerGoods': 'Consumer Goods',
    'contactsCompany.industry.foodBeverage': 'Food & Beverage',
    'contactsCompany.industry.apparelTextile': 'Apparel/Textile',
    'contactsCompany.industry.financialServices': 'Financial Services',
    'contactsCompany.industry.insurance': 'Insurance',
    'contactsCompany.industry.businessServices': 'Business Services',
    'contactsCompany.industry.professionalServices': 'Professional Services',
    'contactsCompany.industry.consulting': 'Consulting',
    'contactsCompany.industry.legalTax': 'Legal/Tax',
    'contactsCompany.industry.advertisingMedia': 'Advertising/Media',
    'contactsCompany.industry.cultureEntertainment': 'Culture/Entertainment',
    'contactsCompany.industry.educationTraining': 'Education/Training',
    'contactsCompany.industry.healthcare': 'Healthcare',
    'contactsCompany.industry.biopharma': 'Biopharma',
    'contactsCompany.industry.agriculture': 'Agriculture',
    'contactsCompany.industry.travelHospitality': 'Travel/Hospitality',
    'contactsCompany.industry.publicUtilities': 'Public Utilities',
    'contactsCompany.industry.governmentNonprofit': 'Government/Nonprofit',
    'contactsCompany.industry.other': 'Other',
    'contactsCompany.size.1_10': '1-10 people',
    'contactsCompany.size.11_50': '11-50 people',
    'contactsCompany.size.51_100': '51-100 people',
    'contactsCompany.size.101_500': '101-500 people',
    'contactsCompany.size.501_1000': '501-1000 people',
    'contactsCompany.size.1001_5000': '1001-5000 people',
    'contactsCompany.size.5000_plus': '5000+ people',
    'contactsCompany.grade.vip': 'VIP',
    'contactsCompany.grade.important': 'Important',
    'contactsCompany.grade.normal': 'Normal',
    'contactsCompany.grade.potential': 'Potential',
    'contactsCompany.grade.lowPriority': 'Low Priority',
    'contactsCompany.customerType.prospect': 'Prospect',
    'contactsCompany.customerType.trialCustomer': 'Trial Customer',
    'contactsCompany.customerType.payingCustomer': 'Paying Customer',
    'contactsCompany.customerType.keyAccount': 'Key Account',
    'contactsCompany.customerType.strategicAccount': 'Strategic Account',
    'contactsCompany.customerType.channelPartner': 'Channel Partner',
    'contactsCompany.customerType.dealerAgent': 'Dealer/Agent',
    'contactsCompany.customerType.supplier': 'Supplier',
    'contactsCompany.customerType.businessPartner': 'Business Partner',
    'contactsCompany.customerType.competitor': 'Competitor',
    'contactsCompany.customerType.other': 'Other'
  }
}
for (const locale of ['es-ES', 'fr-FR', 'de-DE', 'ja-JP', 'ko-KR', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT']) {
  contactsCompanyI18n[locale] = { ...contactsCompanyI18n['en-US'] }
}
Object.assign(contactsCompanyI18n['es-ES'], {
  companyPhone: 'Teléfono de la empresa',
  companyPhonePlaceholder: 'Línea fija, centralita o teléfono de empresa',
  selectOrInputIndustry: 'Seleccionar o escribir sector',
  selectOrInputSize: 'Seleccionar o escribir tamaño',
  selectOrInputGrade: 'Seleccionar o escribir nivel de cliente',
  selectOrInputCustomerType: 'Seleccionar o escribir tipo de cliente',
  selectOrInputContact: 'Seleccionar o escribir contacto'
})
Object.assign(contactsCompanyI18n['fr-FR'], {
  companyPhone: 'Téléphone de l’entreprise',
  companyPhonePlaceholder: 'Fixe, standard ou téléphone de l’entreprise',
  selectOrInputIndustry: 'Sélectionner ou saisir un secteur',
  selectOrInputSize: 'Sélectionner ou saisir la taille',
  selectOrInputGrade: 'Sélectionner ou saisir le niveau client',
  selectOrInputCustomerType: 'Sélectionner ou saisir le type de client',
  selectOrInputContact: 'Sélectionner ou saisir un contact'
})
Object.assign(contactsCompanyI18n['de-DE'], {
  companyPhone: 'Firmentelefon',
  companyPhonePlaceholder: 'Festnetz, Zentrale oder Firmentelefon',
  selectOrInputIndustry: 'Branche auswählen oder eingeben',
  selectOrInputSize: 'Unternehmensgröße auswählen oder eingeben',
  selectOrInputGrade: 'Kundenstufe auswählen oder eingeben',
  selectOrInputCustomerType: 'Kundentyp auswählen oder eingeben',
  selectOrInputContact: 'Kontakt auswählen oder eingeben'
})
Object.assign(contactsCompanyI18n['ja-JP'], {
  companyPhone: t('domain.auto.06e92083cb'),
  companyPhonePlaceholder: t('domain.auto.9e674947bf'),
  selectOrInputIndustry: t('domain.auto.8ae808b8b9'),
  selectOrInputSize: t('domain.auto.6b07229c12'),
  selectOrInputGrade: t('domain.auto.a5146682e6'),
  selectOrInputCustomerType: '顧客タイプを選択または入力',
  selectOrInputContact: '連絡先を選択または入力'
})
Object.assign(contactsCompanyI18n['ko-KR'], {
  companyPhone: '회사 전화',
  companyPhonePlaceholder: '대표번호, 교환대 또는 회사 연락처',
  selectOrInputIndustry: '업종 선택 또는 입력',
  selectOrInputSize: '회사 규모 선택 또는 입력',
  selectOrInputGrade: '고객 등급 선택 또는 입력',
  selectOrInputCustomerType: '고객 유형 선택 또는 입력',
  selectOrInputContact: '연락처 선택 또는 입력'
})
Object.assign(contactsCompanyI18n['pt-PT'], {
  companyPhone: 'Telefone da empresa',
  companyPhonePlaceholder: 'Telefone fixo, central ou contacto da empresa',
  selectOrInputIndustry: 'Selecionar ou inserir setor',
  selectOrInputSize: 'Selecionar ou inserir dimensão',
  selectOrInputGrade: 'Selecionar ou inserir classificação',
  selectOrInputCustomerType: 'Selecionar ou inserir tipo de cliente',
  selectOrInputContact: 'Selecionar ou inserir contacto'
})
Object.assign(contactsCompanyI18n['ar-SA'], {
  companyPhone: 'هاتف الشركة',
  companyPhonePlaceholder: 'الهاتف الأرضي أو السنترال أو رقم الشركة',
  selectOrInputIndustry: 'اختر أو أدخل القطاع',
  selectOrInputSize: 'اختر أو أدخل حجم الشركة',
  selectOrInputGrade: 'اختر أو أدخل درجة العميل',
  selectOrInputCustomerType: 'اختر أو أدخل نوع العميل',
  selectOrInputContact: 'اختر أو أدخل جهة اتصال'
})
Object.assign(contactsCompanyI18n['ru-RU'], {
  companyPhone: 'Телефон компании',
  companyPhonePlaceholder: 'Городской, общий или контактный телефон компании',
  selectOrInputIndustry: 'Выберите или введите отрасль',
  selectOrInputSize: 'Выберите или введите размер компании',
  selectOrInputGrade: 'Выберите или введите уровень клиента',
  selectOrInputCustomerType: 'Выберите или введите тип клиента',
  selectOrInputContact: 'Выберите или введите контакт'
})
Object.assign(contactsCompanyI18n['it-IT'], {
  companyPhone: 'Telefono aziendale',
  companyPhonePlaceholder: 'Fisso, centralino o telefono aziendale',
  selectOrInputIndustry: 'Seleziona o inserisci settore',
  selectOrInputSize: 'Seleziona o inserisci dimensione',
  selectOrInputGrade: 'Seleziona o inserisci livello cliente',
  selectOrInputCustomerType: 'Seleziona o inserisci tipo cliente',
  selectOrInputContact: 'Seleziona o inserisci contatto'
})
const detailPayload = computed(() => payload(detailRecord.value))
const detailDrawerSize = computed(() => window.innerWidth <= 768 ? '100%' : '72%')
const isCompanyDetail = computed(() => detailRecord.value?.object_type === 'company')
const showManufacturingDetailTab = computed(() => {
  if (!isCompanyDetail.value) return false
  const packages = Array.isArray(industryManifest.value?.packages) ? industryManifest.value.packages : []
  if (!packages.some((item) => item?.code === 'manufacturing')) return false
  const tabs = industryManifest.value?.pageTabs?.['customer.detail'] || []
  return Array.isArray(tabs) && tabs.some((item) => String(item?.key || '').startsWith('mfg_'))
})
const serviceSummary = computed(() => detailRecord.value ? (related.service || {}) : {})
const dealSummary = computed(() => detailRecord.value ? (related.deal_summary || {}) : {})
const contactSummary = computed(() => detailRecord.value ? (related.summary || {}) : {})
const detailCompanyName = computed(() => {
  if (!detailRecord.value) return '-'
  const p = detailPayload.value || {}
  return p.company || p.company_name || related.companies?.[0]?.name || (detailRecord.value.company_id ? `${t('domain.auto.54175dfac9')}${detailRecord.value.company_id}` : '-')
})
const kpiCards = computed(() => [
  { key: 'leads', label: t('domain.auto.3d0ce81995'), value: Number(contactSummary.value.lead_count || related.leads.length), hint: `${Number(contactSummary.value.campaign_count || related.campaigns.length)}${t('domain.auto.17b138d5e0')}` },
  { key: 'opportunities', label: t('domain.auto.260617751a'), value: Number(contactSummary.value.active_opportunities || 0), hint: formatAmount(contactSummary.value.active_pipeline_amount || 0) },
  { key: 'revenue', label: t('domain.auto.5ac60d781c'), value: formatAmount(contactSummary.value.closed_revenue || 0), hint: `${Number(contactSummary.value.closed_deals || 0)}${t('domain.auto.6f3b8cfbc5')}` },
  { key: 'receivable', label: t('domain.auto.a770bf6a96'), value: formatAmount(contactSummary.value.open_receivable || 0), hint: `${Number(contactSummary.value.open_orders || 0)}${t('domain.auto.ca4f974fbd')}` },
  { key: 'orders', label: t('domain.auto.084389cc97'), value: Number(contactSummary.value.order_count || related.orders.length), hint: `${Number(contactSummary.value.project_count || related.projects.length)}${t('domain.auto.d3c450dd3c')}` },
  { key: 'tickets', label: t('domain.auto.125432691e'), value: Number(contactSummary.value.ticket_processing || 0), hint: `${t('domain.auto.e80ecbfeb6')}${formatCSAT(contactSummary.value.avg_csat)}` },
  { key: 'last', label: t('domain.auto.78acf9e064'), value: formatShortTime(contactSummary.value.last_interaction_at), hint: `${Number(contactSummary.value.interaction_count || related.interactions.length)}${t('domain.auto.8603768df0')}` }
])
const latestResearchInsights = computed(() => {
  const p = detailPayload.value || {}
  const latest = researchHistory.value.find((item) => item.status === 'succeeded') || {}
  return latest.insights || p.ai_research?.insights || {}
})
const latestCompanyProfile = computed(() => {
  const p = detailPayload.value || {}
  const latest = researchHistory.value.find((item) => item.status === 'succeeded') || {}
  return latest.company_profile || latest.profile || p.ai_research?.company_profile || p.ai_research?.profile || {}
})
const latestResearchSources = computed(() => {
  const p = detailPayload.value || {}
  const latest = researchHistory.value.find((item) => item.status === 'succeeded') || {}
  return latest.sources || p.ai_research?.sources || []
})
const companyProfileFields = computed(() => {
  const p = detailPayload.value || {}
  const profile = latestCompanyProfile.value || {}
  const identity = profile.identity || {}
  const business = profile.business || {}
  const customers = profile.customers || {}
  return [
    { key: 'description', label: t('domain.auto.7c0a72b0b1'), value: business.description || p.description || p.note },
    { key: 'products_services', label: t('domain.auto.31f1419000'), value: business.products_services || p.products_services },
    { key: 'target_customers', label: t('domain.auto.8af107bd8f'), value: customers.target_customers || p.target_customers },
    { key: 'markets', label: t('domain.auto.ab576ae358'), value: business.markets || p.markets },
    { key: 'headquarters', label: t('domain.auto.0b04e7e574'), value: identity.headquarters || p.headquarters },
    { key: 'founded_year', label: t('domain.auto.fd48211383'), value: identity.founded_year || p.founded_year },
    { key: 'employee_count_range', label: t('domain.auto.82846b736b'), value: identity.employee_count_range || p.employee_count_range || p.size },
    { key: 'key_people', label: t('domain.auto.02e3e230cb'), value: p.key_people }
  ]
})
const researchCompleteness = computed(() => {
  const required = [
    ['industry', t('domain.auto.b313f81655')],
    ['website', t('domain.auto.7c67482ff1')],
    ['description', t('domain.auto.7c0a72b0b1')],
    ['products_services', t('domain.auto.6bc2823971')],
    ['target_customers', t('domain.auto.8af107bd8f')],
    ['headquarters', t('domain.auto.0b04e7e574')],
    ['employee_count_range', t('domain.auto.82846b736b')],
    ['buying_signals', t('domain.auto.fa12045bfe')]
  ]
  const p = detailPayload.value || {}
  const missing = required.filter(([key]) => isResearchEmpty(p[key])).map(([, label]) => label)
  return { percentage: Math.round(((required.length - missing.length) / required.length) * 100), missing }
})
const researchFormFill = computed(() => researchJob.value?.form_fill || researchJob.value?.suggestions || {})
const researchFormFillKeys = computed(() => Object.keys(researchFormFill.value || {}))
const researchFieldDefs = [
  { key: 'name', label: t('domain.auto.e1272432a4'), current: () => form.name || detailRecord.value?.name },
  { key: 'website', label: t('domain.auto.7c67482ff1'), current: () => form.payload.website || detailPayload.value.website },
  { key: 'industry', label: t('domain.auto.b313f81655'), current: () => form.payload.industry || detailPayload.value.industry },
  { key: 'size', label: t('domain.auto.675a254004'), current: () => form.payload.size || detailPayload.value.size },
  { key: 'customer_type', label: t('domain.auto.c3785b2f81'), current: () => form.payload.customer_type || detailPayload.value.customer_type },
  { key: 'grade', label: t('domain.auto.5c42c048ea'), current: () => form.payload.grade || detailPayload.value.grade },
  { key: 'description', label: t('domain.auto.7c0a72b0b1'), current: () => form.payload.description || detailPayload.value.description },
  { key: 'note', label: t('domain.auto.e0361480e3'), current: () => form.payload.note || detailPayload.value.note },
  { key: 'headquarters', label: t('domain.auto.0b04e7e574'), current: () => form.payload.headquarters || detailPayload.value.headquarters },
  { key: 'employee_count_range', label: t('domain.auto.82846b736b'), current: () => form.payload.employee_count_range || detailPayload.value.employee_count_range },
  { key: 'products_services', label: t('domain.auto.6bc2823971'), current: () => form.payload.products_services || detailPayload.value.products_services },
  { key: 'target_customers', label: t('domain.auto.8af107bd8f'), current: () => form.payload.target_customers || detailPayload.value.target_customers },
  { key: 'business_model', label: t('domain.auto.0dae4ae6dc'), current: () => form.payload.business_model || detailPayload.value.business_model },
  { key: 'markets', label: t('domain.auto.ab576ae358'), current: () => form.payload.markets || detailPayload.value.markets },
  { key: 'buying_signals', label: t('domain.auto.fa12045bfe'), current: () => form.payload.buying_signals || detailPayload.value.buying_signals },
  { key: 'risk_signals', label: t('domain.auto.a69053af14'), current: () => form.payload.risk_signals || detailPayload.value.risk_signals },
  { key: 'tech_stack', label: t('domain.auto.a2c6f15e72'), current: () => form.payload.tech_stack || detailPayload.value.tech_stack },
  { key: 'tags', label: t('domain.auto.ae0a7afece'), current: () => form.tags || splitTags(detailRecord.value?.tags) }
]
const researchSuggestionRows = computed(() => {
  const suggestions = researchFormFill.value || {}
  return researchFieldDefs
    .filter((field) => suggestions[field.key])
    .map((field) => {
      const suggestion = suggestions[field.key] || {}
      return {
        ...field,
        value: suggestion.value,
        confidence: suggestion.confidence,
        sources: suggestion.sources || [],
        source_urls: suggestion.source_urls || suggestion.sources || [],
        evidence: suggestion.evidence || '',
        basis: suggestion.basis || '',
        current: field.current(),
        selected: researchSelectedFields.value.includes(field.key)
      }
    })
})
const researchProfileCards = computed(() => {
  const profile = researchJob.value?.company_profile || researchJob.value?.profile || {}
  const identity = profile.identity || {}
  const classification = profile.classification || {}
  const business = profile.business || {}
  const customers = profile.customers || {}
  const signals = profile.signals || {}
  const risks = profile.risks || {}
  return [
    { key: 'name', label: t('domain.auto.38931612ea'), value: identity.name },
    { key: 'industry', label: t('domain.auto.4efe3aeb99'), value: [classification.industry, classification.sub_industry].filter(Boolean).join(' / ') },
    { key: 'headquarters', label: t('domain.auto.a63823d892'), value: [identity.headquarters, identity.employee_count_range].filter(Boolean).join(' / ') },
    { key: 'business_model', label: t('domain.auto.0dae4ae6dc'), value: business.business_model },
    { key: 'products', label: t('domain.auto.31f1419000'), value: business.products_services },
    { key: 'customers', label: t('domain.auto.8af107bd8f'), value: customers.target_customers },
    { key: 'markets', label: t('domain.auto.ab576ae358'), value: business.markets },
    { key: 'signals', label: t('domain.auto.7558d95176'), value: signals.buying_signals },
    { key: 'risks', label: t('domain.auto.5b44e0c958'), value: risks.risk_signals || risks.unknowns }
  ].filter((item) => !isResearchEmpty(item.value))
})
const researchStrategyItems = computed(() => {
  const profile = researchJob.value?.company_profile || researchJob.value?.profile || {}
  const strategy = profile.sales_strategy || {}
  return [
    ...(strategy.potential_needs || []).map((item) => `${t('domain.auto.c98c638430')}${item}`),
    ...(strategy.recommended_next_actions || []),
    ...(strategy.opening_talk_tracks || []).map((item) => `${t('domain.auto.13ef93676d')}${item}`)
  ].filter(Boolean).slice(0, 8)
})
const activeTypeLabel = computed(() => activeType.value === 'company' ? t('domain.auto.5d80806d7f') : t('domain.auto.2425bd4bc1'))
const contactFormTitle = computed(() => {
  if (!editingId.value) return activeType.value === 'company' ? t('domain.auto.dfc213f3cd') : t('domain.auto.ae11d6a9cf')
  return activeType.value === 'company' ? `${t('domain.auto.a7f814c0a4')} ${t('domain.auto.5d80806d7f')}` : `${t('domain.auto.a7f814c0a4')} ${t('domain.auto.2425bd4bc1')}`
})
const searchPlaceholder = computed(() => activeType.value === 'company' ? t('domain.auto.307181d0bd') : t('domain.auto.7a94f57db8'))
const canManage = computed(() => canManageDomain('contacts'))

watch(() => route.meta.objectType, (value) => {
  const nextType = value || 'contact'
  if (nextType !== activeType.value) {
    activeType.value = nextType
  }
})

watch(activeType, () => {
  pagination.page = 1
  detailDrawer.value = false
  formDialog.value = false
  researchHistory.value = []
  loadPage()
})
watch(researchDialog, (visible) => {
  if (!visible) stopResearchPolling()
})
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))

onMounted(async () => {
  await Promise.all([loadUsers(), loadPage(), loadIndustryManifest()])
})

onBeforeUnmount(() => {
  stopResearchPolling()
})

async function loadUsers() {
  try {
    const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } })
    users.value = res.data?.items || []
  } catch {
    users.value = []
  }
}

async function loadIndustryManifest() {
  try {
    const res = await api.get('/api/v1/prd/extensions/manifest')
    industryManifest.value = res.data || { packages: [], pageTabs: {} }
  } catch {
    industryManifest.value = { packages: [], pageTabs: {} }
  }
}

async function loadCompanyOptions(keyword = '') {
  companyOptionsLoading.value = true
  try {
    const res = await api.get('/api/v1/prd/contacts/objects', {
      params: cleanParams({
        object_type: 'company',
        keyword,
        page: 1,
        size: 50
      })
    })
    companyOptions.value = res.data?.items || []
  } catch {
    companyOptions.value = []
  } finally {
    companyOptionsLoading.value = false
  }
}

async function loadContactOptions(keyword = '') {
  contactOptionsLoading.value = true
  try {
    const res = await api.get('/api/v1/prd/contacts/objects', {
      params: cleanParams({
        object_type: 'contact',
        keyword,
        status: 'active',
        page: 1,
        size: 20
      })
    })
    contactOptions.value = res.data?.items || []
  } catch {
    contactOptions.value = []
  } finally {
    contactOptionsLoading.value = false
  }
}

async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/contacts/objects', { params: cleanParams({ object_type: activeType.value, keyword: filters.keyword, status: filters.status, page: pagination.page, size: pagination.size }) })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.f6b7a41e56')))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  duplicateCandidates.value = []
  Object.assign(form, defaultForm())
  form.object_type = activeType.value
  companyFormTab.value = 'base'
  normalizeCompanyPayloadForForm(form.payload)
  formDialog.value = true
  if (activeType.value === 'contact') loadCompanyOptions()
  if (activeType.value === 'company') loadContactOptions()
}

async function openEdit(row) {
  editingId.value = row.id
  Object.assign(form, { object_type: row.object_type, name: row.name, code: row.code || '', status: row.status || 'active', owner_id: row.owner_id || 0, amount: row.amount || 0, currency: row.currency || defaultBusinessCurrency(), score: row.score || 0, tags: splitTags(row.tags), payload: { ...payload(row) } })
  companyFormTab.value = 'base'
  normalizeCompanyPayloadForForm(form.payload)
  formDialog.value = true
  if (form.object_type === 'contact') loadCompanyOptions(form.payload.company || '')
  if (form.object_type === 'company') loadContactOptions(form.payload.contact_name || '')
  await checkDuplicates()
}

async function submitForm() {
  if (!form.name) {
    ElMessage.error(t('domain.auto.ca898456b2'))
    return
  }
  try {
    await checkDuplicates()
    const submitPayload = normalizedSubmitForm()
    const editingRecordID = editingId.value
    const res = editingRecordID
      ? await api.put(`/api/v1/prd/contacts/objects/${editingRecordID}`, submitPayload)
      : await api.post('/api/v1/prd/contacts/objects', submitPayload)
    const savedRecord = res.data || res
    if (savedRecord?.object_type === 'company') await linkSelectedContactToCompany(savedRecord)
    ElMessage.success(t('domain.auto.3c5be5f2a1'))
    formDialog.value = false
    await loadPage()
    if (detailDrawer.value && detailRecord.value?.id && savedRecord?.id && Number(detailRecord.value.id) === Number(savedRecord.id)) {
      detailRecord.value = savedRecord
      await loadRelated(savedRecord)
      if (savedRecord.object_type === 'company') await loadResearchHistory(savedRecord.id)
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.f6b7a41e56')))
  }
}

async function openDetail(row) {
  detailRecord.value = row
  detailActiveTab.value = 'overview'
  detailDrawer.value = true
  await loadRelated(row)
  if (row.object_type === 'company') await loadResearchHistory(row.id)
}

async function loadRelated(row) {
  const res = await api.get(`/api/v1/prd/flows/contacts/${row.id}/360`)
  related.contacts = res.data?.contacts || []
  related.companies = res.data?.companies || []
  related.campaigns = res.data?.campaigns || []
  related.leads = res.data?.leads || []
  related.opportunities = res.data?.opportunities || []
  related.quotes = res.data?.quotes || []
  related.deals = res.data?.deals || []
  related.orders = res.data?.orders || []
  related.projects = res.data?.projects || []
  related.tickets = res.data?.tickets || []
  related.interactions = res.data?.interactions || []
  related.service = res.data?.service || {}
  related.deal_summary = res.data?.deal_summary || {}
  related.summary = res.data?.summary || {}
  related.recent_timeline = res.data?.recent_timeline || []
}

function timelineOperator(item) { return timelineOperatorText(item, users.value, formatTime) }

async function loadResearchHistory(companyID) {
  if (!companyID) {
    researchHistory.value = []
    return
  }
  try {
    const res = await api.get('/api/v1/prd/contacts/company-research/jobs', { params: { company_id: companyID, page: 1, size: 10 } })
    researchHistory.value = res.data?.items || []
  } catch {
    researchHistory.value = []
  }
}

function openResearchFromForm() {
  if (activeType.value !== 'company') return
  Object.assign(researchForm, {
    company_id: editingId.value || 0,
    company_name: form.name,
    website: form.payload.website || '',
    region: form.payload.headquarters || '',
    industry_hint: form.payload.industry || '',
    note: form.payload.note || form.payload.description || '',
    mode: 'deep',
    source: 'form'
  })
  researchJob.value = null
  researchSelectedFields.value = []
  researchDialog.value = true
}

function openResearchFromDetail(row) {
  const p = payload(row)
  Object.assign(researchForm, {
    company_id: row.id,
    company_name: row.name,
    website: p.website || '',
    region: p.headquarters || '',
    industry_hint: p.industry || '',
    note: p.note || p.description || '',
    mode: 'deep',
    source: 'detail'
  })
  researchJob.value = null
  researchSelectedFields.value = []
  researchDialog.value = true
}

function openResearchSources() {
  if (!latestResearchSources.value.length) {
    ElMessage.info(t('domain.auto.f74070d4fb'))
    return
  }
  const latest = researchHistory.value.find((item) => item.status === 'succeeded')
  if (latest) showResearchJob(latest)
  else researchDialog.value = true
}

async function submitResearchJob() {
  if (!researchForm.company_name && !researchForm.website) {
    ElMessage.error(t('domain.auto.6c71a4a16e'))
    return
  }
  stopResearchPolling()
  researchLoading.value = true
  try {
    const res = await api.post('/api/v1/prd/contacts/company-research/jobs', {
      company_id: Number(researchForm.company_id || 0),
      company_name: researchForm.company_name,
      website: researchForm.website,
      region: researchForm.region,
      industry_hint: researchForm.industry_hint,
      note: researchForm.note,
      mode: researchForm.mode || 'deep'
    })
    researchJob.value = res.data
    pollResearchJob(res.data?.id)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.5711e726e5')))
  } finally {
    researchLoading.value = false
  }
}

function pollResearchJob(jobID) {
  if (!jobID) return
  researchPollTimer.value = window.setInterval(async () => {
    try {
      const res = await api.get(`/api/v1/prd/contacts/company-research/jobs/${jobID}`)
      researchJob.value = res.data
      if (['succeeded', 'failed', 'blocked'].includes(res.data?.status)) {
        stopResearchPolling()
        buildDefaultResearchSelection()
        if (res.data?.status === 'succeeded') {
          ElMessage.success(t('domain.auto.9ad9a31e2f'))
          if (detailRecord.value?.id) await loadResearchHistory(detailRecord.value.id)
        }
      }
    } catch (error) {
      stopResearchPolling()
      ElMessage.error(apiErrorMessage(error, t('domain.auto.e8e675b522')))
    }
  }, 1500)
}

function stopResearchPolling() {
  if (researchPollTimer.value) {
    window.clearInterval(researchPollTimer.value)
    researchPollTimer.value = null
  }
}

function buildDefaultResearchSelection() {
  const suggestions = researchFormFill.value || {}
  researchSelectedFields.value = researchFieldDefs
    .filter((field) => {
      const suggestion = suggestions[field.key]
      if (!suggestion || isResearchEmpty(suggestion.value)) return false
      if (!(suggestion.source_urls || suggestion.sources || []).length || !suggestion.evidence) return false
      const current = field.current()
      return isResearchEmpty(current) && (suggestion.apply_default === true || Number(suggestion.confidence || 0) >= 0.75)
    })
    .map((field) => field.key)
}

function toggleResearchField(key, checked) {
  const set = new Set(researchSelectedFields.value)
  if (checked) set.add(key)
  else set.delete(key)
  researchSelectedFields.value = [...set]
}

async function applyResearchResult() {
  if (!researchSelectedFields.value.length || !researchJob.value) return
  researchApplyLoading.value = true
  try {
    if (researchForm.source === 'form' && !researchForm.company_id) {
      applyResearchToForm()
      ElMessage.success(t('domain.auto.884fc2f29e'))
      researchDialog.value = false
      return
    }
    const res = await api.post(`/api/v1/prd/contacts/company-research/jobs/${researchJob.value.id}/apply`, { fields: researchSelectedFields.value, overwrite_existing: false })
    ElMessage.success(t('domain.auto.63c28bfbeb'))
    researchDialog.value = false
    if (res.data?.company) {
      detailRecord.value = res.data.company
    }
    await loadPage()
    if (detailRecord.value?.id) {
      await loadRelated(detailRecord.value)
      await loadResearchHistory(detailRecord.value.id)
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.fba9d9dc4b')))
  } finally {
    researchApplyLoading.value = false
  }
}

function applyResearchToForm() {
  const suggestions = researchFormFill.value || {}
  const listFields = new Set(['products_services', 'target_customers', 'markets', 'key_people', 'social_links', 'buying_signals', 'risk_signals', 'tech_stack'])
  for (const key of researchSelectedFields.value) {
    const value = suggestions[key]?.value
    if (isResearchEmpty(value)) continue
    if (key === 'name') form.name = String(value)
    else if (key === 'tags') form.tags = Array.isArray(value) ? value : splitTags(value)
    else {
      form.payload[key] = value
      if (listFields.has(key)) {
        form.payload[`${key}_text`] = Array.isArray(value) ? value.join(',') : String(value || '')
      }
    }
  }
  normalizeCompanyPayloadForForm(form.payload)
}

function showResearchJob(item) {
  researchJob.value = item
  buildDefaultResearchSelection()
  researchDialog.value = true
}

function insertDetailTemplate() {
  const template = [
    t('domain.auto.9ceb565de2'),
    '',
    t('domain.auto.8050055673'),
    '',
    t('domain.auto.c34a9a0d48'),
    '',
    t('domain.auto.f1714dee1e'),
    '',
    t('domain.auto.fd7ddcd320'),
    ''
  ].join('\n')
  form.payload.description = form.payload.description ? `${form.payload.description}\n\n${template}` : template
}

function wrapSelectedDetailText(prefix, suffix) {
  const textarea = companyDetailEditorRef.value?.textarea
  if (!textarea) {
    form.payload.description = `${prefix}${form.payload.description || ''}${suffix}`
    return
  }
  const start = textarea.selectionStart || 0
  const end = textarea.selectionEnd || 0
  const text = form.payload.description || ''
  const selected = text.slice(start, end) || t('domain.auto.162177e7a6')
  form.payload.description = `${text.slice(0, start)}${prefix}${selected}${suffix}${text.slice(end)}`
}

function openInteraction(row) {
  interactionTarget.value = row
  Object.assign(interactionForm, { type: 'call', summary: '', next_action: '' })
  interactionDialog.value = true
}

async function submitInteraction() {
  if (!interactionTarget.value) return
  if (!interactionForm.summary) {
    ElMessage.error(t('domain.auto.9931cd9365'))
    return
  }
  const target = interactionTarget.value
  const p = payload(target)
  const relatedType = target.object_type || activeType.value
  const companyID = relatedType === 'company' ? target.id : Number(target.company_id || p.company_id || 0)
  const contactID = relatedType === 'contact' ? target.id : 0
  const res = await api.post('/api/v1/prd/contacts/objects', {
    object_type: 'interaction',
    name: `${target.name}-${interactionTypeLabel(interactionForm.type)}`,
    status: 'active',
    owner_id: target.owner_id || 0,
    company_id: companyID,
    contact_id: contactID,
    currency: target.currency || defaultBusinessCurrency(),
    tags: ['interaction', relatedType],
    payload: {
      type: interactionForm.type,
      interaction_type: interactionForm.type,
      summary: interactionForm.summary,
      next_action: interactionForm.next_action,
      related_type: relatedType,
      related_id: target.id,
      related_object_id: target.id,
      source_module: 'contacts',
      happened_at: new Date().toISOString(),
      company: relatedType === 'company' ? target.name : p.company,
      company_id: companyID || undefined,
      contact: relatedType === 'contact' ? target.name : '',
      contact_id: contactID || undefined
    }
  })
  if (res.data?.id) {
    try {
      await api.post('/api/v1/prd/relations', {
        source_type: 'contacts',
        source_id: target.id,
        target_type: 'contacts',
        target_id: res.data.id,
        relation_type: 'timeline',
        payload: { source: 'contact_detail_interaction' }
      })
    } catch {
      // 后端对象创建已自动补关系；这里失败不影响互动展示。
    }
  }
  ElMessage.success(t('domain.auto.39bd019991'))
  interactionDialog.value = false
  if (detailRecord.value?.id === target.id) await loadRelated(target)
}

async function inactiveOne(row) {
  if (row.status === 'inactive') return
  await ElMessageBox.confirm(`${t('domain.auto.e2ac985eb0')}${row.name}${t('domain.auto.e60178e849')}`, t('domain.auto.f3a1e6101f'), { type: 'warning' })
  await api.put(`/api/v1/prd/contacts/objects/${row.id}`, { ...row, status: 'inactive', tags: splitTags(row.tags), payload: payload(row) })
  ElMessage.success(t('domain.auto.6c7dcbb73a'))
  await loadPage()
  if (detailRecord.value?.id === row.id) {
    detailRecord.value = { ...detailRecord.value, status: 'inactive' }
  }
}

async function checkDuplicates() {
  const p = form.payload || {}
  try {
    const res = await api.post('/api/v1/prd/flows/contacts/dedupe', {
      object_type: form.object_type || activeType.value,
      object_id: editingId.value || 0,
      name: form.name,
      phone: p.phone || '',
      email: p.email || '',
      company: p.company || form.name,
      website: p.website || ''
    })
    duplicateCandidates.value = res.data?.items || []
  } catch {
    duplicateCandidates.value = []
  }
}

async function restoreOne(row) {
  await api.post(`/api/v1/prd/flows/contacts/${row.id}/restore`, { reason: t('domain.auto.987e86e1e9') })
  ElMessage.success(t('domain.auto.6aa9ad7241'))
  await loadPage()
  if (detailRecord.value?.id === row.id) {
    detailRecord.value = { ...detailRecord.value, status: 'active' }
    await loadRelated(detailRecord.value)
  }
}

async function unlinkCompany(company) {
  if (!detailRecord.value || activeType.value !== 'contact') return
  await ElMessageBox.confirm(`${t('domain.auto.45a4615400')}${company.name}${t('domain.auto.fc99be90fd')}`, t('domain.auto.25a139280f'), { type: 'warning' })
  await api.post('/api/v1/prd/flows/contacts/unlink-company', {
    contact_id: detailRecord.value.id,
    company_id: company.id,
    reason: t('domain.auto.f18dfbf84f')
  })
  ElMessage.success(t('domain.auto.dc2ea0d6d4'))
  await Promise.all([loadPage(), loadRelated(detailRecord.value)])
}

function resetFilters() {
  filters.keyword = ''
  filters.status = ''
  pagination.page = 1
  loadPage()
}

function defaultForm() {
  return { object_type: activeType.value, name: '', code: '', status: 'active', owner_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { source: 'manual', grade: 'normal' } }
}
function normalizedSubmitForm() {
  const next = { ...form, tags: [...(form.tags || [])], payload: { ...(form.payload || {}) } }
  if (next.object_type === 'company') normalizeCompanyPayloadForSubmit(next.payload)
  if (next.object_type === 'contact') normalizeContactCompanyPayloadForSubmit(next)
  return next
}
function handleContactCompanyChange(value) {
  const name = String(value || '').trim()
  const selected = companyOptions.value.find((item) => item.name === name)
  form.company_id = selected?.id || 0
  form.payload.company = name
  form.payload.company_name = name
  form.payload.company_id = selected?.id || 0
}
function handleCompanyContactChange(value) {
  const name = String(value || '').trim()
  const selected = contactOptions.value.find((item) => item.name === name)
  form.contact_id = selected?.id || 0
  if (selected) {
    form.payload.contact = selected.name
    form.payload.contact_name = selected.name
    form.payload.contact_id = selected.id
    return
  }
  delete form.payload.contact
  delete form.payload.contact_name
  delete form.payload.contact_id
}
async function linkSelectedContactToCompany(company) {
  const contactID = Number(form.contact_id || form.payload.contact_id || 0)
  if (!contactID || !company?.id) return
  await api.post('/api/v1/prd/flows/contacts/link-company', {
    company_id: company.id,
    contact_id: contactID,
    primary: true,
    reason: 'company_form'
  })
}
function normalizeContactCompanyPayloadForSubmit(next) {
  const name = String(next.payload.company || next.payload.company_name || '').trim()
  const selected = companyOptions.value.find((item) => item.name === name)
  next.company_id = selected?.id || Number(next.payload.company_id || 0) || 0
  if (name) {
    next.payload.company = name
    next.payload.company_name = name
  }
  if (next.company_id) next.payload.company_id = next.company_id
  else delete next.payload.company_id
}
function normalizeCompanyPayloadForForm(payloadData) {
  if (!payloadData || form.object_type !== 'company') return
  payloadData.revenue_currency = payloadData.revenue_currency || form.currency || defaultBusinessCurrency()
  form.contact_id = Number(payloadData.contact_id || 0) || 0
  payloadData.contact_name = payloadData.contact_name || payloadData.contact || ''
  for (const key of ['products_services', 'target_customers', 'markets', 'key_people', 'social_links', 'buying_signals', 'risk_signals', 'tech_stack']) {
    const textKey = `${key}_text`
    if (payloadData[textKey] === undefined) payloadData[textKey] = Array.isArray(payloadData[key]) ? payloadData[key].join(',') : String(payloadData[key] || '')
  }
}
function normalizeCompanyPayloadForSubmit(payloadData) {
  payloadData.revenue_currency = payloadData.revenue_currency || form.currency || defaultBusinessCurrency()
  const selected = contactOptions.value.find((item) => Number(item.id) === Number(form.contact_id || payloadData.contact_id || 0))
  form.contact_id = selected?.id || Number(form.contact_id || payloadData.contact_id || 0) || 0
  if (selected) {
    payloadData.contact = selected.name
    payloadData.contact_name = selected.name
  }
  if (form.contact_id) payloadData.contact_id = form.contact_id
  else {
    delete payloadData.contact
    delete payloadData.contact_name
    delete payloadData.contact_id
  }
  for (const key of ['products_services', 'target_customers', 'markets', 'key_people', 'social_links', 'buying_signals', 'risk_signals', 'tech_stack']) {
    const textKey = `${key}_text`
    if (payloadData[textKey] !== undefined) {
      payloadData[key] = splitTags(payloadData[textKey])
      delete payloadData[textKey]
    }
  }
}
function payload(row) {
  try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} }
}
function isResearchEmpty(value) {
  if (Array.isArray(value)) return value.length === 0
  if (value && typeof value === 'object') return Object.keys(value).length === 0
  const text = String(value ?? '').trim()
  return !text || text === '<nil>'
}
function formatResearchValue(value) {
  if (Array.isArray(value)) return value.filter(Boolean).join('、')
  if (value && typeof value === 'object') return Object.values(value).filter(Boolean).join('、')
  return String(value ?? '').trim()
}
function researchStatusLabel(value) {
  return ({ pending: t('domain.auto.6d57212411'), running: t('domain.auto.c7e9624387'), succeeded: t('domain.auto.e99b48a29b'), failed: t('domain.auto.3e3c8068bb'), blocked: t('domain.auto.202a7aabc1') }[value] || value || '-')
}
function researchStatusType(value) {
  return ({ pending: 'info', running: 'warning', succeeded: 'success', failed: 'danger', blocked: 'danger' }[value] || 'info')
}
function researchBasisLabel(value) {
  return ({ source: t('domain.auto.c63f79e636'), model_knowledge: t('domain.auto.cba7c13689'), inference: t('domain.auto.4ab7348381'), user_input: t('domain.auto.a10aaeeadd') }[value] || value || '-')
}
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function statusLabel(value) { return ({ active: t('domain.auto.d4e9ca3dd4'), inactive: t('domain.auto.d989e55188'), merged: t('domain.auto.9c8acd3d30'), new: t('domain.auto.0cda8d1c71'), assigned: t('domain.auto.732b2f5de1'), processing: t('domain.auto.fcb979ef0b'), waiting_customer: t('domain.auto.69b352c10e'), resolved: t('domain.auto.5e045cba10'), closed: t('domain.auto.f628761bf5'), reopened: t('domain.auto.c753c38b3b'), draft: t('domain.auto.0f436818c0'), pending_review: t('domain.auto.f53b68002d'), confirmed: t('domain.auto.1b9b73eb06'), partial_shipped: t('domain.auto.9727328c5a'), shipped: t('domain.auto.4340959624'), completed: t('domain.auto.e99b48a29b'), canceled: t('domain.auto.a5ffdc95ee'), initial_contact: t('domain.auto.45b80954f8'), need_confirmed: t('domain.auto.eb5a8bd262'), proposal: t('domain.auto.df33ce33bb'), negotiation: t('domain.auto.83b022f008'), won: t('domain.auto.e65d23ddf1'), lost: t('domain.auto.5375e3d681'), on_hold: t('domain.auto.f5f7939ed7'), pending_assign: t('domain.auto.cb0b7f6d57'), following: t('domain.auto.a5b46b1a52'), mql: 'MQL', sql: 'SQL', converted: t('domain.auto.45d2a4da13'), invalid: t('domain.auto.93736531ce') }[value] || value || '-') }
function ownerName(ownerID) {
  const id = Number(ownerID || 0)
  if (!id) return t('domain.auto.3bb92e6efc')
  const user = users.value.find((item) => Number(item.id) === id)
  return user ? userDisplayName(user) : `${t('domain.auto.9ba763ea34')}${id}`
}
function userDisplayName(user) {
  const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`
  return user.department_name ? `${label} · ${user.department_name}` : label
}
function contactCompanyName(row) {
  const p = payload(row)
  return p.company || p.company_name || (row.company_id ? `${t('domain.auto.54175dfac9')}${row.company_id}` : '-')
}
function optionLabel(value, options) {
  const text = String(value || '').trim()
  if (!text) return ''
  const option = options.find((item) => item.value === text)
  return option ? ct(option.labelKey) : text
}
function contactRoleGradeLabel(row) {
  const p = payload(row)
  if (activeType.value === 'company') return optionLabel(p.grade, companyGradeOptions) || '-'
  const role = String(p.decision_role || '').trim()
  if (role) {
    const roleLabels = {
      decision_maker: t('domain.auto.9bed2d370d'),
      influencer: t('domain.auto.ff5691b83d'),
      user: t('domain.auto.4a7f444ac5')
    }
    return roleLabels[role] || role
  }
  return optionLabel(p.grade, companyGradeOptions) || '-'
}
function ct(key) {
  const locale = i18nState.locale || 'zh-CN'
  const bundle = contactsCompanyI18n[locale] || contactsCompanyI18n['en-US'] || contactsCompanyI18n['zh-CN']
  return bundle[key] || contactsCompanyI18n['en-US']?.[key] || contactsCompanyI18n['zh-CN']?.[key] || key
}
function limited(items, size = 8) {
  return Array.isArray(items) ? items.slice(0, size) : []
}
function goDomain(domain) {
  const routes = {
    campaigns: '/campaigns',
    leads: '/leads',
    opportunities: '/opportunities',
    deals: '/deals',
    orders: '/orders',
    projects: '/projects',
    tickets: '/tickets',
    contacts: '/contacts',
    companies: '/companies'
  }
  router.push(routes[domain] || '/contacts')
}
function timelineDomainLabel(domain, objectType) {
  if (objectType === 'quote') return t('domain.auto.78208f311a')
  return ({
    campaigns: t('domain.auto.5fa95b1b73'),
    leads: t('domain.auto.8de0b8fb1d'),
    opportunities: t('domain.auto.c4beee8569'),
    deals: t('domain.auto.4483df6e28'),
    orders: t('domain.auto.81b3798beb'),
    projects: t('domain.auto.22336e6b89'),
    tickets: t('domain.auto.85c230ed15'),
    contacts: objectType === 'interaction' ? t('domain.auto.477f5f46af') : t('domain.auto.2425bd4bc1')
  }[domain] || objectType || domain || t('domain.auto.753ccc8e2e'))
}
function companyOptionLabel(item) {
  const p = payload(item)
  const info = p.industry || p.website || ''
  return info ? `${item.name} · ${info}` : item.name
}
function contactOptionLabel(item) {
  const p = payload(item)
  const info = p.title || p.phone || p.email || ''
  return info ? `${item.name} · ${info}` : item.name
}
function currencySymbol(currency) {
  return currencySymbols[currency] || `${currency} `
}
function interactionTypeLabel(value) { return ({ call: t('domain.auto.f39c6b629f'), email: t('domain.auto.1c8e464184'), meeting: t('domain.auto.33e9651895'), social: t('domain.auto.1f3cb01e46'), note: t('domain.auto.e0361480e3') }[value] || value || t('domain.auto.477f5f46af')) }
function formatAmount(amount, currency = defaultBusinessCurrency()) {
  return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency()), i18nState.locale)
}
function formatCSAT(value) {
  const score = Number(value || 0)
  if (!score) return '-'
  return `${score.toFixed(2)} / 5`
}
function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value).replace('T', ' ').slice(0, 19)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
function formatShortTime(value) {
  if (!value) return '-'
  const full = formatTime(value)
  return full === '-' ? '-' : full.slice(5)
}
async function promptText(title, message) {
  try {
    const { value } = await ElMessageBox.prompt(message, title, { confirmButtonText: t('domain.auto.f526c89937'), cancelButtonText: t('domain.auto.4d0b4688c7') })
    return value
  } catch {
    return ''
  }
}
</script>

<style scoped>
.company-profile-grid,
.company-insight-panel {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
}

.company-profile-item,
.company-insight-panel > div {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.company-profile-item span,
.company-insight-panel span {
  display: block;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.company-profile-item strong,
.company-insight-panel strong {
  display: block;
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
}

.research-completeness {
  display: grid;
  gap: 10px;
}

.research-missing {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  color: var(--el-text-color-secondary);
}

.research-list-block {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 12px;
  background: var(--el-fill-color-blank);
}

.research-profile-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.research-profile-card {
  min-height: 78px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.research-profile-card span {
  display: block;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.research-profile-card strong {
  display: block;
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
}

.research-list-block ul {
  margin: 8px 0 0;
  padding-left: 18px;
}

.research-list-block li {
  margin: 4px 0;
}

.research-source-list .related-item span {
  word-break: break-all;
}

.research-usage {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.company-form-tabs {
  width: 100%;
  max-width: 860px;
  margin: 0 auto;
}

.company-form-tabs :deep(.el-tabs__content) {
  padding-top: 14px;
}

.company-form-tabs :deep(.el-form-item) {
  margin-bottom: 16px;
}

.company-form-tabs :deep(.el-form-item__label) {
  min-width: 124px;
  height: auto;
  min-height: 32px;
  padding-right: 12px;
  text-align: right;
  white-space: normal;
  overflow-wrap: anywhere;
  line-height: 1.35;
}

.company-form-tabs :deep(.el-form-item__content) {
  min-width: 0;
}

.company-form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 16px 28px;
  align-items: start;
}

.company-dialog-form {
  display: block;
}

.company-form-grid .wide-field {
  grid-column: 1 / -1;
}

.company-form-grid :deep(.el-input),
.company-form-grid :deep(.el-select),
.company-form-grid :deep(.el-textarea) {
  width: 100%;
}

.company-website-row {
  display: flex;
  width: 100%;
  gap: 10px;
}

.company-website-row .el-input {
  flex: 1;
  min-width: 0;
}

.company-revenue-row {
  display: flex;
  width: 100%;
  gap: 10px;
}

.company-revenue-currency {
  flex: 0 0 112px;
}

.company-revenue-row .el-input {
  flex: 1;
  min-width: 0;
}

.company-option {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.company-option strong,
.company-option span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.company-option span {
  max-width: 220px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.rich-editor {
  width: 100%;
}

.rich-editor-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.contact-overview-drawer {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.contact-identity-card,
.contact-overview-section,
.contact-kpi-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.contact-identity-card {
  padding: 16px;
}

.contact-identity-main {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  min-width: 0;
}

.contact-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.contact-title-row h2 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 22px;
  line-height: 1.25;
  word-break: break-word;
}

.contact-identity-main p {
  margin: 8px 0 0;
  color: var(--el-text-color-secondary);
}

.contact-actions {
  flex: 0 0 auto;
  justify-content: flex-end;
  margin: 0;
}

.contact-identity-grid,
.summary-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.contact-identity-grid > div,
.summary-metrics > div {
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.contact-identity-grid span,
.summary-metrics span,
.contact-kpi-card span {
  display: block;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.contact-identity-grid strong,
.summary-metrics strong,
.contact-kpi-card strong {
  display: block;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.45;
  word-break: break-word;
}

.contact-kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(145px, 1fr));
  gap: 10px;
}

.contact-kpi-card {
  min-height: 94px;
  padding: 12px;
}

.contact-kpi-card strong {
  font-size: 20px;
}

.contact-kpi-card small {
  display: block;
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  line-height: 1.35;
  word-break: break-word;
}

.contact-overview-tabs :deep(.el-tabs__content) {
  padding-top: 4px;
}

.contact-overview-grid,
.contact-tab-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 14px;
}

.contact-overview-section {
  min-width: 0;
  padding: 14px;
}

.contact-overview-section.wide {
  grid-column: 1 / -1;
}

.contact-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.contact-section-head h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 15px;
  line-height: 1.35;
}

.overview-list,
.timeline-list {
  display: grid;
  gap: 8px;
}

.overview-item,
.timeline-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.overview-item > div,
.timeline-row {
  min-width: 0;
}

.overview-item strong,
.overview-item span,
.timeline-row strong,
.timeline-row small,
.timeline-row span {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-item strong,
.timeline-row strong {
  color: var(--el-text-color-primary);
  font-size: 13px;
}

.overview-item span,
.timeline-row small,
.timeline-row span {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.timeline-list.compact .timeline-row {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  align-items: start;
}

.timeline-list.compact .timeline-row small {
  grid-column: 2;
}

@media (max-width: 768px) {
  .contact-identity-main,
  .overview-item {
    align-items: stretch;
    flex-direction: column;
  }

  .contact-actions {
    justify-content: flex-start;
  }

  .contact-overview-grid,
  .contact-tab-grid,
  .contact-identity-grid,
  .summary-metrics {
    grid-template-columns: 1fr;
  }

  .timeline-list.compact .timeline-row {
    grid-template-columns: 1fr;
  }

  .timeline-list.compact .timeline-row small {
    grid-column: auto;
  }
}
</style>
