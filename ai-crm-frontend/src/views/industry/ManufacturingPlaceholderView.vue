<template>
  <section class="domain-page manufacturing-workspace">
    <div class="manufacturing-workspace__header">
      <div>
        <p>{{ t('industry.manufacturing.customer360.eyebrow') }}</p>
        <h2>{{ currentWorkspaceTitle }}</h2>
        <span>{{ currentWorkspaceDesc }}</span>
      </div>
      <div class="manufacturing-workspace__header-actions">
        <el-button v-if="canRefreshMetricSnapshots" :loading="metricRefreshing" @click="refreshMetricSnapshots">
          {{ t('industry.manufacturing.action.refreshMetrics') }}
        </el-button>
        <el-button :loading="dashboardLoading" @click="refreshAll">
          {{ t('industry.manufacturing.action.refresh') }}
        </el-button>
      </div>
    </div>

    <div v-if="isHomeWorkflow" class="mfg-executive-dashboard" v-loading="dashboardLoading">
      <section class="mfg-executive-panel mfg-executive-panel--wide">
        <div class="mfg-executive-panel__head">
          <div>
            <h3>{{ t('industry.manufacturing.home.executiveTitle') }}</h3>
            <p>{{ t('industry.manufacturing.home.executiveDesc') }}</p>
          </div>
          <span>{{ t('industry.manufacturing.home.dataUpdatedAt') }} {{ dashboardUpdatedAtText }}</span>
        </div>
        <div class="mfg-kpi-grid">
          <button
            v-for="item in executiveKpiCards"
            :key="item.key"
            type="button"
            class="mfg-kpi-card"
            :class="[`tone-${item.tone}`, { 'is-clickable': item.metric }]"
            @click="openDashboardMetric(item)"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <small>{{ item.caption }}</small>
          </button>
        </div>
      </section>

      <section class="mfg-executive-panel">
        <div class="mfg-executive-panel__head">
          <div>
            <h3>{{ t('industry.manufacturing.home.quoteBusiness') }}</h3>
            <p>{{ t('industry.manufacturing.home.quoteBusinessDesc') }}</p>
          </div>
        </div>
        <div class="mfg-business-metrics">
          <div v-for="item in quoteBusinessMetrics" :key="item.key">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </section>

      <section class="mfg-executive-panel">
        <div class="mfg-executive-panel__head">
          <div>
            <h3>{{ t('industry.manufacturing.home.deliveryRisk') }}</h3>
            <p>{{ t('industry.manufacturing.home.deliveryRiskDesc') }}</p>
          </div>
        </div>
        <div class="mfg-delivery-risk">
          <div v-for="item in deliveryRiskMetrics" :key="item.key" :class="`risk-${item.key}`">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
        <div class="mfg-on-time-rate">
          <span>{{ t('industry.manufacturing.home.onTimeRate') }}</span>
          <strong>{{ percentText(deliveryMetrics.on_time_rate) }}</strong>
        </div>
      </section>

      <section class="mfg-executive-panel mfg-executive-panel--wide">
        <div class="mfg-executive-panel__head">
          <div>
            <h3>{{ t('industry.manufacturing.home.metricGroups') }}</h3>
            <p>{{ t('industry.manufacturing.home.metricGroupsDesc') }}</p>
          </div>
        </div>
        <div class="mfg-metric-group-grid">
          <button
            v-for="group in executiveMetricGroups"
            :key="group.key"
            type="button"
            class="mfg-metric-group-card"
            :class="{ 'is-clickable': group.metric }"
            @click="openDashboardMetric(group)"
          >
            <header>
              <strong>{{ group.label }}</strong>
              <span>{{ group.updatedAt }}</span>
            </header>
            <div class="mfg-metric-group-card__body">
              <div v-for="metric in group.metrics" :key="metric.key">
                <span>{{ metric.label }}</span>
                <strong>{{ metric.value }}</strong>
              </div>
            </div>
            <small v-if="group.refreshText">{{ group.refreshText }}</small>
          </button>
        </div>
      </section>
    </div>

    <div v-if="!isHomeWorkflow" class="manufacturing-workspace__layout" :class="{ 'is-single': !isCustomerWorkflow }">
      <aside v-if="isCustomerWorkflow" class="manufacturing-workspace__companies">
        <el-input
          v-model="companyKeyword"
          :placeholder="t('industry.manufacturing.customer360.searchCompany')"
          clearable
          @keyup.enter="loadCompanies"
        />
        <div v-loading="companiesLoading" class="manufacturing-company-list">
          <button
            v-for="company in companies"
            :key="company.id"
            type="button"
            class="manufacturing-company-list__item"
            :class="{ active: selectedCustomerID === company.id }"
            @click="selectCompany(company)"
          >
            <strong>{{ company.name || '-' }}</strong>
            <span>{{ company.code || t('industry.manufacturing.customer360.noCode') }}</span>
          </button>
          <el-empty
            v-if="!companiesLoading && !companies.length"
            :description="t('industry.manufacturing.customer360.emptyCompanies')"
          />
        </div>
      </aside>

      <main class="manufacturing-workspace__main">
        <el-tabs v-model="activeWorkspaceTab">
          <el-tab-pane v-if="isCustomerWorkflow" :label="t('industry.manufacturing.workspace.customer360')" name="customer360">
            <mfg-customer360-panel
              :customer-id="selectedCustomerID"
              :customer-name="selectedCompany?.name || ''"
            />
          </el-tab-pane>

          <el-tab-pane
            v-for="section in visibleSections"
            :key="section.key"
            :label="t(section.titleKey)"
            :name="section.key"
          >
            <div class="mfg-section">
              <div class="mfg-section__toolbar">
                <div>
                  <h3>{{ t(section.titleKey) }}</h3>
                  <p>{{ sectionDescription(section) }}</p>
                  <p v-if="section.key === 'rfqs'" class="mfg-rfq-scope-note">
                    {{ t('industry.manufacturing.rfq.scopeNote') }}
                  </p>
                  <p v-if="section.key === 'cost'" class="mfg-rfq-scope-note">
                    {{ t('industry.manufacturing.costing.flowDesc') }}
                  </p>
                </div>
                <div class="mfg-section__actions">
                  <el-input
                    v-model="section.keyword"
                    :placeholder="t('industry.manufacturing.action.search')"
                    clearable
                    @keyup.enter="loadSection(section)"
                  />
                  <el-button @click="loadSection(section)">{{ t('industry.manufacturing.action.query') }}</el-button>
                  <el-button v-if="canCreateSection(section)" type="primary" @click="openSectionCreateDialog(section)">
                    {{ createButtonText(section) }}
                  </el-button>
                </div>
              </div>

              <el-table v-if="section.key === 'quoteWorkspace'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="220">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <strong>{{ rfqDisplayName(row) }}</strong>
                      <small v-if="row.business_name && row.rfq_no">{{ row.rfq_no }}</small>
                      <span>{{ rfqProductText(row) }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="180">
                  <template #default="{ row }">{{ rfqCustomerText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.rfq.table.quantity')" width="110">
                  <template #default="{ row }">{{ row.quantity || '-' }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.rfq.table.targetPrice')" width="140">
                  <template #default="{ row }">{{ rfqTargetPriceText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.rfq.table.deliveryDate')" width="140">
                  <template #default="{ row }">{{ formatDate(row.required_delivery_date) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ mfgStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.costing.nextAction')" min-width="180">
                  <template #default="{ row }">{{ readinessActionLabel(rfqReadinessAction(row)) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="380" fixed="right">
                  <template #default="{ row }">
                    <el-button size="small" type="primary" plain @click="openRFQWorkspaceDialog(row)">
                      {{ t('industry.manufacturing.quoteWorkspace.detail') }}
                    </el-button>
                    <el-button v-if="canEditRow(rfqSection, row)" size="small" @click="openRecordDialog(rfqSection, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-tooltip
                      v-for="action in visibleRowActions(rfqSection, row)"
                      :key="action.key"
                      :content="actionHelpText(action)"
                      placement="top"
                    >
                      <el-button size="small" @click="runAction(rfqSection, row, action)">
                        {{ t(action.labelKey) }}
                      </el-button>
                    </el-tooltip>
                    <el-button v-if="canDeleteRow(rfqSection, row)" size="small" type="danger" @click="deleteRecord(rfqSection, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <el-table v-else-if="section.key === 'rfqs'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="220">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <strong>{{ rfqDisplayName(row) }}</strong>
                      <small v-if="row.business_name && row.rfq_no">{{ row.rfq_no }}</small>
                      <span>{{ rfqProductText(row) }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="180">
                  <template #default="{ row }">{{ rfqCustomerText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.rfq.table.quantity')" width="110">
                  <template #default="{ row }">{{ row.quantity || '-' }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.rfq.table.targetPrice')" width="140">
                  <template #default="{ row }">{{ rfqTargetPriceText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.rfq.table.deliveryDate')" width="140">
                  <template #default="{ row }">{{ formatDate(row.required_delivery_date) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ mfgStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="300" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canEditRow(section, row)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-tooltip
                      v-for="action in visibleRowActions(section, row)"
                      :key="action.key"
                      :content="actionHelpText(action)"
                      placement="top"
                    >
                      <el-button size="small" @click="runAction(section, row, action)">
                        {{ t(action.labelKey) }}
                      </el-button>
                    </el-tooltip>
                    <el-button v-if="canDeleteRow(section, row)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div v-else-if="section.key === 'cost'" class="mfg-cost-workbench">
                <div class="mfg-cost-workbench__guide">
                  <div>
                    <strong>{{ t('industry.manufacturing.costing.queueTitle') }}</strong>
                    <p>{{ t('industry.manufacturing.costing.queueDesc') }}</p>
                  </div>
                  <el-tag type="warning" effect="light">{{ t('industry.manufacturing.costing.ownerRole') }}</el-tag>
                </div>
                <el-table :data="section.items" border v-loading="section.loading">
                  <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="180">
                    <template #default="{ row }">
                      <div class="mfg-record-title">
                        <strong>{{ row.rfq_no || `${t('industry.manufacturing.field.rfqId')} #${row.rfq_id}` }}</strong>
                        <span>{{ costProductText(row) }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="160">
                    <template #default="{ row }">{{ costCustomerText(row) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.rfq.table.quantity')" width="100">
                    <template #default="{ row }">{{ row.quantity || '-' }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.rfq.table.targetPrice')" width="130">
                    <template #default="{ row }">{{ costTargetPriceText(row) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.rfq.table.deliveryDate')" width="120">
                    <template #default="{ row }">{{ formatDate(row.required_delivery_date) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                    <template #default="{ row }">
                      <el-tag size="small" :type="statusTagType(row.status)">
                      {{ compensationStatusText(row.status) }}
                    </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.costing.nextAction')" min-width="170">
                    <template #default="{ row }">
                      <div class="mfg-cost-next">
                        <strong>{{ costNextActionText(row) }}</strong>
                        <small v-if="isBOMMissing(row)">{{ t('industry.manufacturing.costing.engineeringRole') }}</small>
                        <span>{{ costRiskText(row) }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.action.operations')" width="280" fixed="right">
                    <template #default="{ row }">
                      <el-button v-if="canCreateWorkspaceBOM && isBOMMissing(row)" size="small" type="warning" @click="goCreateBOMSnapshotFromCost(row)">
                        {{ t('industry.manufacturing.action.goCreateBOMSnapshot') }}
                      </el-button>
                      <el-button v-else-if="canCreateWorkspaceCost" size="small" @click="openRecordDialog(section, row)">
                        {{ row.status === 'pending' ? t('industry.manufacturing.action.calculateCost') : t('industry.manufacturing.action.edit') }}
                      </el-button>
                      <el-button v-if="canCreateWorkspaceQuote && isCostingCompleted(row)" size="small" type="success" @click="goCreateQuoteFromCost(row)">
                        {{ t('industry.manufacturing.action.createQuoteVersion') }}
                      </el-button>
                      <el-button v-if="canCompleteCostSheet(row)" size="small" type="primary" @click="completeCostSheet(row)">
                        {{ t('industry.manufacturing.action.completeCosting') }}
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <el-table v-else-if="section.key === 'bom'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="220">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <strong>{{ row.snapshot_no || row.id || '-' }}</strong>
                      <span>{{ row.rfq_no ? `${t('industry.manufacturing.field.rfq')} ${row.rfq_no}` : '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="160">
                  <template #default="{ row }">{{ businessCustomerText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.productDemand')" min-width="180">
                  <template #default="{ row }">{{ businessProductText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.bomLineCount')" width="110">
                  <template #default="{ row }">{{ row.bom_line_count || 0 }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.amount')" width="130">
                  <template #default="{ row }">{{ businessAmountText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ mfgStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                  <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="180" fixed="right">
                  <template #default="{ row }">
                    <el-button size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.view') }}
                    </el-button>
                    <el-button v-if="canDeleteSection(section)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <el-table v-else-if="section.key === 'bomTemplates'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="220">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <strong>{{ row.template_no || row.id || '-' }}</strong>
                      <span>{{ t('industry.manufacturing.field.version') }} {{ row.version || '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.workspace.product')" min-width="170">
                  <template #default="{ row }">{{ productFamilyName(row.product_family_id) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.bomLineCount')" width="110">
                  <template #default="{ row }">{{ row.bom_line_count || parseBOMLines(row).length || 0 }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ mfgStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                  <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="180" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canEditSection(section)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-button v-if="canDeleteSection(section)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <el-table v-else-if="section.key === 'product'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.productFamilyName')" min-width="220">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <strong>{{ row.name || '-' }}</strong>
                      <span>{{ row.description || '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.familyCode')" width="160">
                  <template #default="{ row }">{{ row.family_code || '-' }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ mfgStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                  <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="180" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canEditSection(section)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-button v-if="canDeleteSection(section)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div v-else-if="section.key === 'production'" class="mfg-grouped-workbench">
                <div class="mfg-grouped-workbench__bar">
                  <span>{{ t('industry.manufacturing.production.groupMode') }}</span>
                  <el-segmented v-model="productionGroupMode" :options="productionGroupOptions" />
                </div>
                <template v-if="productionGroupedItems.length">
                  <section v-for="groupItem in productionGroupedItems" :key="groupItem.key" class="mfg-grouped-section">
                    <div class="mfg-grouped-section__head">
                      <strong>{{ groupItem.label }}</strong>
                      <span>{{ t('industry.manufacturing.production.groupCount', { count: groupItem.items.length }) }}</span>
                    </div>
                    <el-table :data="groupItem.items" border v-loading="section.loading">
                      <el-table-column :label="t('industry.manufacturing.field.milestone')" min-width="170">
                        <template #default="{ row }">
                          <div class="mfg-record-title">
                            <strong>{{ productionMilestoneText(row.milestone) }}</strong>
                            <span>{{ productionPrimaryContext(row) }}</span>
                          </div>
                        </template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.production.progressStatus')" width="130">
                        <template #default="{ row }">
                          <el-tag size="small" :type="productionStatusTagType(row.progress_status)">
                            {{ productionStatusText(row.progress_status) }}
                          </el-tag>
                        </template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="150">
                        <template #default="{ row }">{{ customerNameText(row) }}</template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.field.salesOwner')" min-width="140">
                        <template #default="{ row }">{{ salesOwnerNameText(row) }}</template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.field.riskLevel')" width="120">
                        <template #default="{ row }">
                          <el-tag size="small" :type="statusTagType(row.risk_level)">
                            {{ mfgStatusText(row.risk_level) }}
                          </el-tag>
                        </template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.field.plannedAt')" width="140">
                        <template #default="{ row }">{{ formatDate(row.planned_at) }}</template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.field.actualAt')" width="140">
                        <template #default="{ row }">{{ formatDate(row.actual_at) }}</template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                        <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
                      </el-table-column>
                      <el-table-column :label="t('industry.manufacturing.action.operations')" width="260" fixed="right">
                        <template #default="{ row }">
                          <el-button v-if="canEditSection(section)" size="small" @click="openRecordDialog(section, row)">
                            {{ t('industry.manufacturing.action.edit') }}
                          </el-button>
                          <el-tooltip
                            v-for="action in visibleRowActions(section, row)"
                            :key="action.key"
                            :content="actionHelpText(action)"
                            placement="top"
                          >
                            <el-button size="small" @click="runAction(section, row, action)">
                              {{ t(action.labelKey) }}
                            </el-button>
                          </el-tooltip>
                          <el-button v-if="canDeleteSection(section)" size="small" type="danger" @click="deleteRecord(section, row)">
                            {{ t('industry.manufacturing.action.delete') }}
                          </el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                  </section>
                </template>
              </div>

              <el-table v-else-if="section.key === 'ecns'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="170">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <el-button link type="primary" @click="openDeliveryChangeDetail(section, row)">
                        {{ row.ecn_no || row.id || '-' }}
                      </el-button>
                      <span>{{ row.change_reason || '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.impactLevel')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small" :type="impactLevelTagType(row.impact_level)">
                      {{ impactLevelText(row.impact_level) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.costImpact')" width="140">
                  <template #default="{ row }">{{ moneyWithCurrency(row.cost_impact, row.cost_impact_currency) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.deliveryImpactDays')" width="130">
                  <template #default="{ row }">{{ row.delivery_impact_days || 0 }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.impactScope')" min-width="180">
                  <template #default="{ row }">{{ row.impact_scope || '-' }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ ecnStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.approvalStatus')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.approval_status)">
                      {{ quoteApprovalStatusText(row.approval_status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.ecn.customerConfirmStatus')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small" :type="row.customer_confirmed ? 'success' : 'warning'">
                      {{ ecnCustomerConfirmText(row.customer_confirmed) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="360" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canEditRow(section, row)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-tooltip
                      v-for="action in visibleRowActions(section, row)"
                      :key="action.key"
                      :content="actionHelpText(action)"
                      placement="top"
                    >
                      <el-button size="small" @click="runAction(section, row, action)">
                        {{ t(action.labelKey) }}
                      </el-button>
                    </el-tooltip>
                    <el-button v-if="canDeleteRow(section, row)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <el-table v-else-if="section.key === 'compensationRecords'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.order')" min-width="150">
                  <template #default="{ row }">
                    <el-button link type="primary" @click="openDeliveryChangeDetail(section, row)">
                      {{ row.compensation_no || row.order_name || (row.order_id ? `#${row.order_id}` : '-') }}
                    </el-button>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="150">
                  <template #default="{ row }">{{ customerNameText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.salesOwner')" min-width="140">
                  <template #default="{ row }">{{ salesOwnerNameText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.compensationReason')" min-width="180">
                  <template #default="{ row }">{{ row.reason || '-' }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.compensationAmount')" width="140">
                  <template #default="{ row }">{{ moneyWithCurrency(row.amount, row.currency) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.responsibilitySplit')" min-width="180">
                  <template #default="{ row }">{{ row.responsibility_split || '-' }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status)">
                      {{ compensationStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.approvalStatus')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.approval_status)">
                      {{ quoteApprovalStatusText(row.approval_status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                  <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="260" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canEditRow(section, row)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-tooltip
                      v-for="action in visibleRowActions(section, row)"
                      :key="action.key"
                      :content="actionHelpText(action)"
                      placement="top"
                    >
                      <el-button size="small" @click="runAction(section, row, action)">
                        {{ t(action.labelKey) }}
                      </el-button>
                    </el-tooltip>
                    <el-button v-if="canDeleteRow(section, row)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <el-table v-else-if="section.key === 'quotes'" :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="220">
                  <template #default="{ row }">
                    <div class="mfg-record-title">
                      <el-button link type="primary" @click="openQuoteDetail(row)">
                        {{ row.quote_no || row.id || '-' }}
                      </el-button>
                      <span>{{ t('industry.manufacturing.field.revision') }} {{ row.revision || 1 }} · {{ row.rfq_no || '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="150">
                  <template #default="{ row }">{{ businessCustomerText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.productDemand')" min-width="160">
                  <template #default="{ row }">{{ businessProductText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.amount')" width="130">
                  <template #default="{ row }">{{ businessAmountText(row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.marginLevel')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.margin_level)">
                      {{ mfgStatusText(row.margin_level) }}
                    </el-tag>
                  </template>
                </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.approvalStatus')" width="120">
                    <template #default="{ row }">
                      <el-tag size="small" :type="statusTagType(row.approval_status)">
                      {{ quoteApprovalStatusText(row.approval_status) }}
                    </el-tag>
                    </template>
                  </el-table-column>
                <el-table-column :label="t('industry.manufacturing.quote.sendStatus')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="quoteSendStatusTagType(row.send_status)">{{ quoteSendStatusText(row.send_status) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.quote.customerStatus')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small" :type="quoteCustomerStatusTagType(row.status)">{{ quoteCustomerStatusText(row.status) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="360" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canCreateWorkspaceQuote && canEditQuote(row)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-button
                      v-for="action in visibleQuoteActions(row)"
                      :key="action.key"
                      size="small"
                      :type="action.type || 'default'"
                      @click="runQuoteAction(section, row, action.key)"
                    >
                      {{ t(action.labelKey) }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <el-table v-else :data="section.items" border v-loading="section.loading">
                <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="180">
                  <template #default="{ row }">{{ primaryText(section, row) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.status')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small" :type="statusTagType(row.status || row.stage || row.risk_level)">
                      {{ sectionStatusText(section, row) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.customer')" width="160">
                  <template #default="{ row }">{{ customerNameText(row) }}</template>
                </el-table-column>
                <el-table-column v-if="section.key !== 'samples'" :label="t('industry.manufacturing.field.amount')" width="130">
                  <template #default="{ row }">{{ money(row.amount || row.target_price || row.cost_impact) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                  <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
                </el-table-column>
                <el-table-column :label="t('industry.manufacturing.action.operations')" width="260" fixed="right">
                  <template #default="{ row }">
                    <el-button v-if="canEditSection(section)" size="small" @click="openRecordDialog(section, row)">
                      {{ t('industry.manufacturing.action.edit') }}
                    </el-button>
                    <el-tooltip
                      v-for="action in visibleRowActions(section, row)"
                      :key="action.key"
                      :content="actionHelpText(action)"
                      placement="top"
                    >
                      <el-button size="small" @click="runAction(section, row, action)">
                        {{ t(action.labelKey) }}
                      </el-button>
                    </el-tooltip>
                    <el-button v-if="canDeleteSection(section)" size="small" type="danger" @click="deleteRecord(section, row)">
                      {{ t('industry.manufacturing.action.delete') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-empty
                v-if="!section.loading && !section.items.length"
                :description="t('industry.manufacturing.empty.workflow')"
              >
                <el-button v-if="canCreateSection(section)" type="primary" @click="openSectionCreateDialog(section)">
                  {{ createButtonText(section) }}
                </el-button>
              </el-empty>
            </div>
          </el-tab-pane>
        </el-tabs>
      </main>
    </div>

    <el-dialog v-model="rfqWorkspaceDialogVisible" :title="t('industry.manufacturing.quoteWorkspace.detailTitle')" width="min(1180px, 96vw)" class="mfg-rfq-workspace-dialog" destroy-on-close @closed="closeRFQWorkspaceDialog">
      <div class="mfg-rfq-dialog" v-loading="quoteWorkspaceLoading">
        <div class="mfg-rfq-dialog__hero">
          <div>
            <span class="mfg-rfq-dialog__eyebrow">{{ t('industry.manufacturing.quoteWorkspace.collabEyebrow') }}</span>
            <h3>{{ rfqDisplayName(quoteWorkspaceRFQ) || `${t('industry.manufacturing.field.rfq')} #${selectedRFQID || '-'}` }}</h3>
            <p>{{ rfqCustomerText(quoteWorkspaceRFQ) }} · {{ rfqProductText(quoteWorkspaceRFQ) }}</p>
          </div>
          <div class="mfg-rfq-dialog__hero-meta">
            <el-tag :type="quoteWorkspaceContext.readiness?.ready ? 'success' : 'warning'" effect="light">
              {{ quoteWorkspaceContext.readiness?.ready ? t('industry.manufacturing.readiness.readyBadge') : t('industry.manufacturing.readiness.missingBadge') }}
            </el-tag>
            <span>{{ t('industry.manufacturing.costing.nextAction') }}: {{ quoteWorkspaceNextActionText }}</span>
          </div>
        </div>
        <div class="mfg-rfq-dialog__summary">
          <div v-for="item in rfqWorkspaceSummaryRows" :key="item.key" class="mfg-rfq-dialog__metric" :class="`is-${item.tone}`">
            <el-icon><component :is="item.icon" /></el-icon>
            <div>
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
        </div>
        <el-tabs v-model="activeRFQWorkspaceTab" class="mfg-rfq-dialog__tabs">
          <el-tab-pane name="rfq">
            <template #label>
              <span class="mfg-rfq-dialog__tab-label"><el-icon><Document /></el-icon>{{ t('industry.manufacturing.quoteWorkspace.tab.rfq') }}</span>
            </template>
            <div class="mfg-rfq-dialog__section-head">
              <div>
                <h4>{{ t('industry.manufacturing.quoteWorkspace.block.rfq') }}</h4>
                <p>{{ t('industry.manufacturing.quoteWorkspace.block.rfqDesc') }}</p>
              </div>
              <div class="mfg-rfq-dialog__section-actions">
                <el-button v-if="canEditRow(rfqSection, quoteWorkspaceRFQ)" size="small" @click="openRecordDialog(rfqSection, quoteWorkspaceRFQ)">{{ t('industry.manufacturing.action.edit') }}</el-button>
                <el-tooltip
                  v-for="action in visibleRowActions(rfqSection, quoteWorkspaceRFQ)"
                  :key="action.key"
                  :content="actionHelpText(action)"
                  placement="top"
                >
                  <el-button size="small" @click="runAction(rfqSection, quoteWorkspaceRFQ, action)">
                    {{ t(action.labelKey) }}
                  </el-button>
                </el-tooltip>
              </div>
            </div>
            <div class="mfg-rfq-detail-groups">
              <section v-for="group in rfqWorkspaceDetailRows(quoteWorkspaceRFQ)" :key="group.key" class="mfg-rfq-detail-group">
                <h5><span></span>{{ group.title }}</h5>
                <dl>
                  <template v-for="item in group.items" :key="item.key">
                    <dt>{{ item.label }}</dt>
                    <dd>
                      <template v-if="item.kind === 'download' && item.file_id">
                        <span>{{ item.value }}</span>
                        <el-button size="small" link type="primary" :icon="Download" @click="downloadRFQCustomerFile(item)">
                          {{ t('industry.manufacturing.rfq.downloadCustomerFile') }}
                        </el-button>
                      </template>
                      <template v-else>{{ item.value }}</template>
                    </dd>
                  </template>
                </dl>
              </section>
            </div>
            <div class="mfg-readiness-result mfg-rfq-dialog__readiness" :class="{ 'is-ready': quoteWorkspaceContext.readiness?.ready }">
              <div class="mfg-readiness-result__hero">
                <span>{{ quoteWorkspaceContext.readiness?.ready ? t('industry.manufacturing.readiness.readyBadge') : t('industry.manufacturing.readiness.missingBadge') }}</span>
                <strong>{{ quoteWorkspaceContext.readiness?.ready ? t('industry.manufacturing.readiness.readyTitle') : t('industry.manufacturing.readiness.missingTitle') }}</strong>
                <p>{{ quoteWorkspaceContext.readiness?.ready ? t('industry.manufacturing.readiness.readyDesc') : t('industry.manufacturing.readiness.missingDesc') }}</p>
              </div>
              <div v-if="!quoteWorkspaceContext.readiness?.ready" class="mfg-readiness-result__missing">
                <strong>{{ t('industry.manufacturing.readiness.missingListTitle') }}</strong>
                <ul>
                  <li v-for="item in quoteWorkspaceMissingItems" :key="item.key">
                    <span>{{ item.label }}</span>
                    <small>{{ item.action }}</small>
                  </li>
                </ul>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane name="clarifications">
            <template #label>
              <span class="mfg-rfq-dialog__tab-label"><el-icon><ChatDotRound /></el-icon>{{ t('industry.manufacturing.quoteWorkspace.block.clarification') }}</span>
            </template>
            <div class="mfg-rfq-dialog__section-head">
              <div>
                <h4>{{ t('industry.manufacturing.quoteWorkspace.block.clarification') }}</h4>
                <p>{{ t('industry.manufacturing.quoteWorkspace.block.clarificationDesc') }}</p>
              </div>
              <el-button v-if="canCreateSection(clarificationsSection)" size="small" type="primary" @click="openWorkspaceCreate('clarifications')">{{ t('industry.manufacturing.action.create') }}</el-button>
            </div>
            <el-table v-if="(quoteWorkspaceContext.clarifications || []).length" :data="quoteWorkspaceContext.clarifications || []" size="small" class="mfg-rfq-table">
              <el-table-column :label="t('industry.manufacturing.rfq.clarificationCategory')" width="150">
                <template #default="{ row }">
                  {{ clarificationSourceText(row) }}
                </template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.question')" min-width="220" prop="question" />
              <el-table-column :label="t('industry.manufacturing.field.status')" width="140">
                <template #default="{ row }">
                  <el-select v-model="row.status" size="small" class="full-input" :disabled="!canAnswerClarification()">
                    <el-option value="open" :label="mfgStatusText('open')" />
                    <el-option value="resolved" :label="mfgStatusText('resolved')" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.answer')" min-width="240">
                <template #default="{ row }">
                  <el-input v-model="row.answer" type="textarea" :autosize="{ minRows: 1, maxRows: 4 }" :disabled="!canAnswerClarification()" />
                </template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.rfq.latestAnswer')" width="180">
                <template #default="{ row }">
                  <div class="mfg-answer-meta">
                    <strong>{{ answerUserName(row.last_answered_by, row.last_answered_by_name) }}</strong>
                    <span>{{ formatTime(row.last_answered_at) }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.action.operations')" width="160" fixed="right">
                <template #default="{ row }">
                  <el-button v-if="canAnswerClarification()" size="small" type="primary" plain :loading="row._saving" @click="saveRFQClarification(row)">
                    {{ t('industry.manufacturing.action.save') }}
                  </el-button>
                  <el-button size="small" link type="primary" :loading="row._historyLoading" @click="openClarificationAnswerHistory(row)">
                    {{ t('industry.manufacturing.action.history') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else class="mfg-rfq-empty" :description="t('industry.manufacturing.quoteWorkspace.empty.clarification')" />
          </el-tab-pane>

          <el-tab-pane name="bom">
            <template #label>
              <span class="mfg-rfq-dialog__tab-label"><el-icon><Box /></el-icon>{{ t('industry.manufacturing.quoteWorkspace.block.bom') }}</span>
            </template>
            <div class="mfg-rfq-dialog__section-head">
              <div>
                <h4>{{ t('industry.manufacturing.quoteWorkspace.block.bom') }}</h4>
                <p>{{ t('industry.manufacturing.quoteWorkspace.block.bomDesc') }}</p>
              </div>
              <el-button v-if="canCreateWorkspaceBOM" size="small" type="primary" @click="openWorkspaceCreate('bom')">{{ t('industry.manufacturing.action.regenerateBOMSnapshot') }}</el-button>
            </div>
            <div v-if="currentBOMSnapshot" class="mfg-bom-current">
              <div v-for="item in currentBOMSnapshotRows" :key="item.key" class="mfg-bom-current__item">
                <span>{{ item.label }}</span>
                <el-button v-if="item.key === 'no'" type="primary" link class="mfg-link-button" @click="openBOMDetail(currentBOMSnapshot)">
                  {{ item.value }}
                </el-button>
                <strong v-else>{{ item.value }}</strong>
              </div>
            </div>
            <el-empty v-else class="mfg-rfq-empty" :description="t('industry.manufacturing.quoteWorkspace.empty.bom')" />
            <el-collapse v-if="bomSnapshotHistory.length" class="mfg-bom-history">
              <el-collapse-item :title="t('industry.manufacturing.bom.historyTitle')" name="history">
                <el-table :data="bomSnapshotHistory" size="small" class="mfg-rfq-table">
                  <el-table-column :label="t('industry.manufacturing.workspace.bom')" min-width="150">
                    <template #default="{ row }">
                      <el-button type="primary" link class="mfg-link-button" @click="openBOMDetail(row)">
                        {{ bomSnapshotDisplayName(row) }}
                      </el-button>
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.bomLineCount')" width="110" prop="bom_line_count" />
                  <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
                    <template #default="{ row }">{{ mfgStatusText(row.status) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.createdAt')" width="150">
                    <template #default="{ row }">{{ formatDate(row.created_at || row.updated_at) }}</template>
                  </el-table-column>
                </el-table>
              </el-collapse-item>
            </el-collapse>
          </el-tab-pane>

          <el-tab-pane v-if="canViewWorkspacePrice" name="cost">
            <template #label>
              <span class="mfg-rfq-dialog__tab-label"><el-icon><Money /></el-icon>{{ t('industry.manufacturing.quoteWorkspace.block.cost') }}</span>
            </template>
            <div class="mfg-rfq-dialog__section-head">
              <div>
                <h4>{{ t('industry.manufacturing.quoteWorkspace.block.cost') }}</h4>
                <p>{{ t('industry.manufacturing.quoteWorkspace.block.costDesc') }}</p>
              </div>
              <el-button v-if="canCreateWorkspaceCost" size="small" type="primary" @click="openWorkspaceCreate('cost')">{{ t('industry.manufacturing.action.calculateCost') }}</el-button>
            </div>
            <el-table v-if="currentCostSheet" :data="[currentCostSheet]" size="small" class="mfg-rfq-table mfg-rfq-table--full">
              <el-table-column :label="t('industry.manufacturing.field.quoteTotal')" min-width="160">
                <template #default="{ row }">{{ businessAmountText(row) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.unitPrice')" min-width="160">
                <template #default="{ row }">{{ unitPriceText(row) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.marginLevel')" min-width="130">
                <template #default="{ row }">{{ mfgStatusText(row.margin_level) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.status')" min-width="130">
                <template #default="{ row }">{{ mfgStatusText(row.status) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.action.operations')" width="220" align="right">
                <template #default="{ row }">
                  <el-button v-if="canCreateWorkspaceCost" size="small" @click="openRecordDialog(costSection, row)">
                    {{ t('industry.manufacturing.action.edit') }}
                  </el-button>
                  <el-button v-if="canCompleteCostSheet(row)" size="small" type="primary" @click="completeCostSheet(row)">
                    {{ t('industry.manufacturing.action.completeCosting') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else class="mfg-rfq-empty" :description="t('industry.manufacturing.quoteWorkspace.empty.cost')" />
            <el-collapse v-if="historicalCostSheets.length" class="mfg-bom-history">
              <el-collapse-item :title="t('industry.manufacturing.costing.historyTitle')" name="cost-history">
                <el-table :data="historicalCostSheets" size="small" class="mfg-rfq-table mfg-rfq-table--full">
                  <el-table-column :label="t('industry.manufacturing.field.quoteTotal')" min-width="150">
                    <template #default="{ row }">{{ businessAmountText(row) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.workspace.bom')" min-width="160">
                    <template #default="{ row }">{{ bomSnapshotLabel(row.bom_snapshot_id) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.marginLevel')" width="120">
                    <template #default="{ row }">{{ mfgStatusText(row.margin_level) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
                    <template #default="{ row }">{{ formatDate(row.updated_at || row.created_at) }}</template>
                  </el-table-column>
                </el-table>
              </el-collapse-item>
            </el-collapse>
          </el-tab-pane>

          <el-tab-pane v-if="canViewWorkspacePrice" name="quotes">
            <template #label>
              <span class="mfg-rfq-dialog__tab-label"><el-icon><Tickets /></el-icon>{{ t('industry.manufacturing.quoteWorkspace.block.quote') }}</span>
            </template>
            <div class="mfg-rfq-dialog__section-head">
              <div>
                <h4>{{ t('industry.manufacturing.quoteWorkspace.block.quote') }}</h4>
                <p>{{ t('industry.manufacturing.quoteWorkspace.block.quoteDesc') }}</p>
              </div>
              <el-button v-if="canCreateWorkspaceQuote" size="small" type="primary" @click="openWorkspaceCreate('quotes')">{{ t('industry.manufacturing.action.createQuoteVersion') }}</el-button>
            </div>
            <el-table v-if="(quoteWorkspaceContext.quotes || []).length" :data="quoteWorkspaceContext.quotes || []" size="small" class="mfg-rfq-table">
              <el-table-column :label="t('industry.manufacturing.workspace.quote')" min-width="150">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openQuoteDetail(row)">
                    {{ row.quote_no || row.id }} · {{ t('industry.manufacturing.field.revision') }} {{ row.revision || 1 }}
                  </el-button>
                </template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.quoteTotal')" width="130">
                <template #default="{ row }">{{ businessAmountText(row) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.unitPrice')" width="130">
                <template #default="{ row }">{{ unitPriceText(row) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.approvalStatus')" width="120">
                <template #default="{ row }">{{ quoteApprovalStatusText(row.approval_status) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.quote.sendStatus')" width="120">
                <template #default="{ row }">{{ quoteSendStatusText(row.send_status) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.quote.customerStatus')" width="120">
                <template #default="{ row }">{{ quoteCustomerStatusText(row.status) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.action.operations')" width="300" fixed="right">
                <template #default="{ row }">
                  <el-button
                    v-for="action in visibleQuoteActions(row)"
                    :key="action.key"
                    size="small"
                    :type="action.type || 'default'"
                    @click="runQuoteAction(quoteSection, row, action.key)"
                  >
                    {{ t(action.labelKey) }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else class="mfg-rfq-empty" :description="t('industry.manufacturing.quoteWorkspace.empty.quote')" />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>

    <el-dialog v-model="bomDetailVisible" :title="t('industry.manufacturing.bom.detailTitle')" width="min(980px, 96vw)" destroy-on-close>
      <div v-if="activeBOMDetail" class="mfg-bom-detail">
        <div class="mfg-context-card">
          <div>
            <span>{{ t('industry.manufacturing.workspace.bom') }}</span>
            <strong>{{ bomSnapshotDisplayName(activeBOMDetail) }}</strong>
          </div>
          <div>
            <span>{{ t('industry.manufacturing.field.status') }}</span>
            <strong>{{ mfgStatusText(activeBOMDetail.status) }}</strong>
          </div>
          <div>
            <span>{{ t('industry.manufacturing.bom.source') }}</span>
            <strong>{{ bomSnapshotSourceText(activeBOMDetail) }}</strong>
          </div>
          <div>
            <span>{{ t('industry.manufacturing.field.createdAt') }}</span>
            <strong>{{ formatDate(activeBOMDetail.created_at || activeBOMDetail.updated_at) }}</strong>
          </div>
        </div>
        <div class="mfg-context-card">
          <div>
            <span>{{ t('industry.manufacturing.field.rfq') }}</span>
                <strong>{{ rfqDisplayName(quoteWorkspaceRFQ) || activeBOMDetail.rfq_business_name || activeBOMDetail.rfq_no || '-' }}</strong>
          </div>
          <div>
            <span>{{ t('industry.manufacturing.field.customer') }}</span>
            <strong>{{ quoteWorkspaceRFQ.rfq_payload?.customer_name || activeBOMDetail.customer_name || '-' }}</strong>
          </div>
          <div>
            <span>{{ t('industry.manufacturing.field.productDemand') }}</span>
            <strong>{{ quoteWorkspaceRFQ.rfq_payload?.product_name || quoteWorkspaceRFQ.rfq_payload?.custom_product_name || activeBOMDetail.product_name || '-' }}</strong>
          </div>
          <div>
            <span>{{ t('industry.manufacturing.field.bomLineCount') }}</span>
            <strong>{{ activeBOMDetail.bom_line_count || bomDetailLines.length }}</strong>
          </div>
        </div>
        <el-table :data="bomDetailLines" size="small" border class="mfg-rfq-table mfg-rfq-table--full">
          <el-table-column :label="t('industry.manufacturing.form.level')" width="80" prop="level" />
          <el-table-column :label="t('industry.manufacturing.form.itemCode')" min-width="130" prop="item_code" />
          <el-table-column :label="t('industry.manufacturing.form.itemName')" min-width="150" prop="item_name" />
          <el-table-column :label="t('industry.manufacturing.form.spec')" min-width="140" prop="spec" />
          <el-table-column :label="t('industry.manufacturing.form.qtyPerUnit')" width="110" prop="qty" />
          <el-table-column :label="t('industry.manufacturing.form.unit')" width="90" prop="unit" />
          <el-table-column :label="t('industry.manufacturing.form.makeBuy')" width="110">
            <template #default="{ row }">{{ row.sourcing === 'make' ? t('industry.manufacturing.form.make') : t('industry.manufacturing.form.buy') }}</template>
          </el-table-column>
          <el-table-column :label="t('industry.manufacturing.field.description')" min-width="150" prop="remark" />
        </el-table>
      </div>
    </el-dialog>

    <el-dialog v-model="recordDialogVisible" :title="recordDialogTitle" width="min(960px, 96vw)" destroy-on-close>
      <el-form v-if="activeSection?.key === 'rfqs'" label-position="top" class="mfg-record-form mfg-rfq-form">
        <section class="mfg-form-block">
          <h4>{{ t('industry.manufacturing.rfq.section.customerInfo') }}</h4>
          <el-row :gutter="12">
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.businessName')">
                <el-input v-model="recordForm.business_name" :placeholder="t('industry.manufacturing.rfq.businessNamePlaceholder')" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.customer')" required>
                <el-select
                  v-model="recordForm.customer_id"
                  class="full-input"
                  filterable
                  remote
                  clearable
                  :remote-method="loadRFQCompanies"
                  :loading="rfqOptionsLoading.companies"
                  @change="handleRFQCompanyChange"
                >
                  <el-option v-for="item in rfqCompanyOptions" :key="item.id" :label="companyOptionLabel(item)" :value="item.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.contact')">
                <el-select
                  v-model="recordForm.contact_id"
                  class="full-input"
                  filterable
                  clearable
                  :loading="rfqOptionsLoading.contacts"
                  @change="handleRFQContactChange"
                >
                  <el-option v-for="item in rfqContactOptions" :key="item.id" :label="contactOptionLabel(item)" :value="item.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.opportunity')">
                <el-select
                  v-model="recordForm.opportunity_id"
                  class="full-input"
                  filterable
                  remote
                  clearable
                  :remote-method="loadRFQOpportunities"
                  :loading="rfqOptionsLoading.opportunities"
                  @change="handleRFQOpportunityChange"
                >
                  <el-option v-for="item in rfqOpportunityOptions" :key="item.id" :label="opportunityOptionLabel(item)" :value="item.id" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="mfg-form-block">
          <h4>{{ t('industry.manufacturing.rfq.section.productDemand') }}</h4>
          <el-form-item :label="t('industry.manufacturing.rfq.productType')">
            <el-radio-group v-model="recordForm.product_type">
              <el-radio-button label="standard">{{ t('industry.manufacturing.rfq.productType.standard') }}</el-radio-button>
              <el-radio-button label="custom">{{ t('industry.manufacturing.rfq.productType.custom') }}</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-row :gutter="12">
            <el-col v-if="recordForm.product_type === 'standard'" :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.standardProduct')">
                <el-select
                  v-model="recordForm.product_id"
                  class="full-input"
                  filterable
                  remote
                  clearable
                  :remote-method="loadRFQProducts"
                  :loading="rfqOptionsLoading.products"
                  @change="handleRFQProductChange"
                >
                  <el-option v-for="item in rfqProductOptions" :key="item.id" :label="productOptionLabel(item)" :value="item.id" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row v-if="recordForm.product_type === 'custom'" :gutter="12">
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.customProductName')">
                <el-input v-model="recordForm.custom_product_name" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.material')">
                <el-input v-model="recordForm.material" />
              </el-form-item>
            </el-col>
            <el-col :xs="24">
              <el-form-item :label="t('industry.manufacturing.rfq.specification')">
                <el-input v-model="recordForm.specification" type="textarea" :rows="3" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.keyParameters')">
                <el-input v-model="recordForm.key_parameters" type="textarea" :rows="3" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item :label="t('industry.manufacturing.rfq.process')">
                <el-input v-model="recordForm.process" type="textarea" :rows="3" />
              </el-form-item>
            </el-col>
            <el-col :xs="24">
              <el-form-item :label="t('industry.manufacturing.rfq.applicationScene')">
                <el-input v-model="recordForm.application_scene" type="textarea" :rows="2" />
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="mfg-form-block">
          <h4>{{ t('industry.manufacturing.rfq.section.drawing') }}</h4>
          <el-form-item :label="t('industry.manufacturing.rfq.drawingUpload')">
            <div class="mfg-upload-row">
              <el-upload :show-file-list="false" :before-upload="uploadRFQDrawing" :disabled="rfqDrawingUploading">
                <el-button :loading="rfqDrawingUploading">{{ t('industry.manufacturing.rfq.uploadDrawing') }}</el-button>
              </el-upload>
              <span class="mfg-upload-file">{{ recordForm.drawing_file_name || t('industry.manufacturing.rfq.noDrawingUploaded') }}</span>
            </div>
          </el-form-item>
          <el-form-item v-if="!recordForm.drawing_file_id" :label="t('industry.manufacturing.rfq.noDrawingReason')">
            <el-input v-model="recordForm.no_drawing_reason" type="textarea" :rows="2" />
          </el-form-item>
          <el-button v-if="recordForm.drawing_file_id" size="small" :icon="Download" @click="downloadRFQCustomerFile(recordForm)">
            {{ t('industry.manufacturing.rfq.downloadCustomerFile') }}
          </el-button>
        </section>

        <section class="mfg-form-block">
          <h4>{{ t('industry.manufacturing.rfq.section.quoteRequest') }}</h4>
          <el-row :gutter="12">
            <el-col :xs="24" :md="6">
              <el-form-item :label="t('industry.manufacturing.field.quantity')" required>
                <el-input-number v-model="recordForm.quantity" :min="0" class="full-input" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item :label="t('industry.manufacturing.field.targetPrice')">
                <el-input-number v-model="recordForm.target_price" :min="0" class="full-input" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item :label="t('industry.manufacturing.rfq.currency')">
                <el-select v-model="recordForm.currency" class="full-input">
                  <el-option v-for="item in rfqCurrencyOptions" :key="item" :label="item" :value="item" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item :label="t('industry.manufacturing.rfq.requiredDeliveryDate')">
                <el-date-picker v-model="recordForm.required_delivery_date" type="date" value-format="YYYY-MM-DD" class="full-input" />
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="mfg-form-block mfg-rfq-clarification-block">
          <div class="mfg-form-block__head mfg-rfq-clarification-toolbar">
            <div>
              <h4>{{ t('industry.manufacturing.rfq.clarificationItems') }}</h4>
              <p>{{ t('industry.manufacturing.rfq.clarificationItemsDesc') }}</p>
            </div>
            <el-button size="small" type="primary" plain @click="addRFQClarificationItem">{{ t('industry.manufacturing.rfq.addClarificationItem') }}</el-button>
          </div>
          <div class="mfg-rfq-clarification-editor">
            <div v-if="recordForm.clarification_items?.length" class="mfg-rfq-clarification-table">
              <div class="mfg-rfq-clarification-table__head">
                <span>{{ t('industry.manufacturing.rfq.clarificationNo') }}</span>
                <span>{{ t('industry.manufacturing.rfq.clarificationCategory') }}</span>
                <span>{{ t('industry.manufacturing.rfq.clarificationRemark') }}</span>
                <span>{{ t('industry.manufacturing.rfq.needClarification') }}</span>
                <span>{{ t('industry.manufacturing.action.operations') }}</span>
              </div>
              <section v-for="(item, index) in recordForm.clarification_items" :key="item.key" class="mfg-rfq-clarification-row">
                <div class="mfg-rfq-clarification-cell mfg-rfq-clarification-cell--index">
                  <span>{{ t('industry.manufacturing.rfq.clarificationNo') }}</span>
                  <strong>{{ index + 1 }}</strong>
                </div>
                <div class="mfg-rfq-clarification-cell">
                  <span>{{ t('industry.manufacturing.rfq.clarificationCategory') }}</span>
                  <el-select v-model="item.category" size="small" class="full-input" @change="handleRFQClarificationCategoryChange(item)">
                    <el-option v-for="option in rfqClarificationCategories" :key="option.value" :label="t(option.labelKey)" :value="option.value" />
                  </el-select>
                </div>
                <div class="mfg-rfq-clarification-cell">
                  <span>{{ t('industry.manufacturing.rfq.clarificationRemark') }}</span>
                  <el-input v-model="item.remark" size="small" />
                </div>
                <div class="mfg-rfq-clarification-cell mfg-rfq-clarification-cell--switch">
                  <span>{{ t('industry.manufacturing.rfq.needClarification') }}</span>
                  <el-switch v-model="item.need_clarification" />
                </div>
                <div class="mfg-rfq-clarification-cell mfg-rfq-clarification-cell--actions">
                  <span>{{ t('industry.manufacturing.action.operations') }}</span>
                  <div>
                    <el-tooltip :content="t('industry.manufacturing.action.moveUp')" placement="top">
                      <el-button size="small" text circle :icon="ArrowUp" :disabled="index === 0" :aria-label="t('industry.manufacturing.action.moveUp')" @click="moveRFQClarificationItem(index, -1)" />
                    </el-tooltip>
                    <el-tooltip :content="t('industry.manufacturing.action.moveDown')" placement="top">
                      <el-button size="small" text circle :icon="ArrowDown" :disabled="index === recordForm.clarification_items.length - 1" :aria-label="t('industry.manufacturing.action.moveDown')" @click="moveRFQClarificationItem(index, 1)" />
                    </el-tooltip>
                    <el-tooltip :content="t('industry.manufacturing.action.delete')" placement="top">
                      <el-button size="small" text circle type="danger" :icon="Delete" :aria-label="t('industry.manufacturing.action.delete')" @click="removeRFQClarificationItem(index)" />
                    </el-tooltip>
                  </div>
                </div>
                <div class="mfg-rfq-clarification-cell mfg-rfq-clarification-cell--content">
                  <span>{{ t('industry.manufacturing.rfq.clarificationContent') }}</span>
                  <el-input v-model="item.content" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" />
                </div>
              </section>
            </div>
            <el-empty v-if="!recordForm.clarification_items?.length" :description="t('industry.manufacturing.rfq.noClarificationItems')" />
          </div>
        </section>
      </el-form>
      <el-form v-else label-position="top" class="mfg-record-form mfg-business-form">
        <section v-if="activeFormSpec?.descKey" class="mfg-form-intro">
          {{ t(activeFormSpec.descKey) }}
        </section>

        <template v-if="isBOMEditorSection">
          <section class="mfg-form-block">
            <h4>{{ t(activeSection?.key === 'bomTemplates' ? 'industry.manufacturing.form.templateContext' : 'industry.manufacturing.form.context') }}</h4>
            <div v-if="activeSection?.key === 'bom'" class="mfg-context-card">
              <div>
                <span>{{ t('industry.manufacturing.field.rfq') }}</span>
                <strong>{{ selectedRFQSummary.business_name || selectedRFQSummary.rfq_no || selectedRFQSummary.id || t('industry.manufacturing.form.selectRFQFirst') }}</strong>
              </div>
              <div>
                <span>{{ t('industry.manufacturing.field.customer') }}</span>
                <strong>{{ selectedRFQSummary.customer_name || '-' }}</strong>
              </div>
              <div>
                <span>{{ t('industry.manufacturing.rfq.section.productDemand') }}</span>
                <strong>{{ selectedRFQSummary.product_name || '-' }}</strong>
              </div>
              <div>
                <span>{{ t('industry.manufacturing.rfq.table.quantity') }}</span>
                <strong>{{ selectedRFQSummary.quantity || '-' }}</strong>
              </div>
            </div>
            <el-row v-if="activeSection?.key === 'bom'" :gutter="12">
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.bom.businessName')">
                  <el-input v-model="recordForm.business_name" :placeholder="t('industry.manufacturing.bom.businessNamePlaceholder')" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.rfq')">
                  <el-select v-model="recordForm.rfq_id" class="full-input" filterable clearable @change="handleBOMRFQChange">
                    <el-option v-for="item in relationOptions.rfqs" :key="item.id" :label="rfqOptionLabel(item)" :value="item.id" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.form.bomSource')">
                  <el-radio-group v-model="recordForm.bom_source" @change="handleBOMSourceChange">
                    <el-radio-button label="manual">{{ t('industry.manufacturing.form.bomSourceManual') }}</el-radio-button>
                    <el-radio-button label="template">{{ t('industry.manufacturing.form.bomSourceTemplate') }}</el-radio-button>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col v-if="recordForm.bom_source === 'template'" :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.workspace.bomTemplate')">
                  <el-select v-model="recordForm.bom_template_id" class="full-input" filterable clearable :loading="relationLoading.bomTemplates" @change="handleBOMTemplateChange">
                    <el-option v-for="item in relationOptions.bomTemplates" :key="item.id" :label="bomTemplateOptionLabel(item)" :value="item.id" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.form.snapshotVersion')">
                  <el-input v-model="recordForm.source_config_version" :placeholder="t('industry.manufacturing.form.versionPlaceholder')" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.form.saveAsTemplate')">
                  <el-switch v-model="recordForm.save_as_template" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row v-if="activeSection?.key === 'bomTemplates'" :gutter="12">
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.workspace.product')">
                  <el-select v-model="recordForm.product_family_id" class="full-input" filterable clearable :loading="relationLoading.productFamilies">
                    <el-option v-for="item in relationOptions.productFamilies" :key="item.id" :label="relationOptionLabel('productFamilies', item)" :value="item.id" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.templateNo')">
                  <el-input v-model="recordForm.template_no" :placeholder="t('industry.manufacturing.form.templateNoAuto')" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.version')">
                  <el-input v-model="recordForm.version" :placeholder="t('industry.manufacturing.form.versionPlaceholder')" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.status')">
                  <el-select v-model="recordForm.status" class="full-input">
                    <el-option v-for="option in ['draft', 'active', 'inactive']" :key="option" :label="mfgStatusText(option)" :value="option" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24">
                <el-form-item :label="t('industry.manufacturing.form.templateApplicability')">
                  <el-input v-model="recordForm.rule_text" type="textarea" :rows="2" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row v-if="activeSection?.key === 'bom' && recordForm.save_as_template" :gutter="12" class="mfg-save-template-fields">
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.workspace.product')">
                  <el-select v-model="recordForm.product_family_id" class="full-input" filterable clearable :loading="relationLoading.productFamilies">
                    <el-option v-for="item in relationOptions.productFamilies" :key="item.id" :label="relationOptionLabel('productFamilies', item)" :value="item.id" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.templateNo')">
                  <el-input v-model="recordForm.template_no" :placeholder="t('industry.manufacturing.form.templateNoAuto')" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.version')">
                  <el-input v-model="recordForm.template_version" :placeholder="t('industry.manufacturing.form.versionPlaceholder')" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.field.status')">
                  <el-select v-model="recordForm.template_status" class="full-input">
                    <el-option v-for="option in ['draft', 'active', 'inactive']" :key="option" :label="mfgStatusText(option)" :value="option" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24">
                <el-form-item :label="t('industry.manufacturing.form.templateApplicability')">
                  <el-input v-model="recordForm.template_rule_text" type="textarea" :rows="2" />
                </el-form-item>
              </el-col>
            </el-row>
          </section>

          <section class="mfg-form-block">
            <div class="mfg-form-block__head">
              <h4>{{ t('industry.manufacturing.form.bomLines') }}</h4>
              <el-button size="small" type="primary" plain @click="addBOMLine">{{ t('industry.manufacturing.action.addLine') }}</el-button>
            </div>
            <el-table :data="recordForm.bom_lines" border class="mfg-bom-editor">
              <el-table-column :label="t('industry.manufacturing.form.level')" width="90">
                <template #default="{ row }"><el-input v-model="row.level" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.itemCode')" min-width="130">
                <template #default="{ row }"><el-input v-model="row.item_code" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.itemName')" min-width="150">
                <template #default="{ row }"><el-input v-model="row.item_name" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.spec')" min-width="140">
                <template #default="{ row }"><el-input v-model="row.spec" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.rfq.material')" width="130">
                <template #default="{ row }"><el-input v-model="row.material" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.rfq.process')" width="130">
                <template #default="{ row }"><el-input v-model="row.process" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.qtyPerUnit')" width="120">
                <template #default="{ row }"><el-input-number v-model="row.qty" :min="0" :precision="3" class="full-input" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.unit')" width="100">
                <template #default="{ row }"><el-input v-model="row.unit" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.lossRate')" width="120">
                <template #default="{ row }"><el-input-number v-model="row.loss_rate" :min="0" :max="100" class="full-input" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.form.makeBuy')" width="120">
                <template #default="{ row }">
                  <el-select v-model="row.sourcing" class="full-input">
                    <el-option :label="t('industry.manufacturing.form.make')" value="make" />
                    <el-option :label="t('industry.manufacturing.form.buy')" value="buy" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.description')" min-width="150">
                <template #default="{ row }"><el-input v-model="row.remark" /></template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.action.operations')" width="90" fixed="right">
                <template #default="{ $index }">
                  <el-button size="small" type="danger" text @click="removeBOMLine($index)">{{ t('industry.manufacturing.action.delete') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </section>
        </template>

        <template v-else>
          <section v-for="group in activeBusinessGroups" :key="group.titleKey" class="mfg-form-block">
            <h4>{{ t(group.titleKey) }}</h4>
            <el-row :gutter="12">
              <el-col v-for="field in group.fields" :key="field.key" :xs="24" :md="field.wide ? 24 : 12">
                <el-form-item :label="t(field.labelKey)">
                  <div v-if="field.kind === 'readonly'" class="mfg-readonly-field">
                    {{ readonlyFieldText(field) }}
                  </div>
                  <el-select
                    v-else-if="field.kind === 'relation'"
                    v-model="recordForm[field.key]"
                    class="full-input"
                    filterable
                    clearable
                    :loading="relationLoading[field.relation]"
                    @change="handleRelationFieldChange(field, $event)"
                  >
                    <el-option v-for="item in relationOptions[field.relation] || []" :key="item.id" :label="relationOptionLabel(field.relation, item)" :value="item.id" />
                  </el-select>
                  <div v-else-if="field.kind === 'number'" class="mfg-number-field">
                    <el-input-number
                      v-model="recordForm[field.key]"
                      :min="0"
                      class="full-input"
                    />
                    <span v-if="field.suffix" class="mfg-field-suffix">{{ fieldSuffixText(field) }}</span>
                  </div>
                  <el-select
                    v-else-if="field.kind === 'user'"
                    v-model="recordForm[field.key]"
                    class="full-input"
                    filterable
                    clearable
                    :loading="usersLoading"
                  >
                    <el-option v-for="item in userOptions" :key="item.id" :label="userDisplayName(item)" :value="item.id" />
                  </el-select>
                  <el-date-picker
                    v-else-if="field.kind === 'date'"
                    v-model="recordForm[field.key]"
                    type="date"
                    value-format="YYYY-MM-DD"
                    class="full-input"
                  />
                  <el-date-picker
                    v-else-if="field.kind === 'datetime'"
                    v-model="recordForm[field.key]"
                    type="datetime"
                    value-format="YYYY-MM-DD HH:mm"
                    format="YYYY-MM-DD HH:mm"
                    class="full-input"
                  />
                  <el-select
                    v-else-if="field.kind === 'creatableSelect'"
                    v-model="recordForm[field.key]"
                    class="full-input"
                    filterable
                    allow-create
                    default-first-option
                    clearable
                  >
                    <el-option v-for="option in field.options" :key="option" :label="creatableOptionText(field, option)" :value="option" />
                  </el-select>
                  <el-select v-else-if="field.options" v-model="recordForm[field.key]" class="full-input" clearable>
                    <el-option v-for="option in field.options" :key="option" :label="fieldOptionText(field, option)" :value="option" />
                  </el-select>
                  <el-input
                    v-else-if="field.kind === 'textarea'"
                    v-model="recordForm[field.key]"
                    type="textarea"
                    :rows="3"
                  />
                  <el-input v-else v-model="recordForm[field.key]" />
                </el-form-item>
              </el-col>
            </el-row>
          </section>
          <section v-if="activeSection?.key === 'samples' && editingRecordID" class="mfg-form-block">
            <h4>{{ t('industry.manufacturing.sample.progressHistory') }}</h4>
            <el-table :data="sampleProgressHistory" border size="small" v-loading="sampleProgressHistoryLoading">
              <el-table-column :label="t('industry.manufacturing.sample.stageChange')" min-width="180">
                <template #default="{ row }">{{ sampleStageText(row.from_stage) }} -> {{ sampleStageText(row.to_stage) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.riskLevel')" width="120">
                <template #default="{ row }">{{ mfgStatusText(row.risk_level) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.customerFeedback')" min-width="180">
                <template #default="{ row }">{{ row.customer_feedback || '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.sample.operator')" width="130">
                <template #default="{ row }">{{ row.created_by_name || '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.createdAt')" width="150">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </section>
          <section v-if="activeSection?.key === 'production' && editingRecordID" class="mfg-form-block">
            <h4>{{ t('industry.manufacturing.production.progressHistory') }}</h4>
            <el-table :data="productionProgressHistory" border size="small" v-loading="productionProgressHistoryLoading">
              <el-table-column :label="t('industry.manufacturing.production.milestoneChange')" min-width="180">
                <template #default="{ row }">{{ productionMilestoneText(row.from_milestone) }} -> {{ productionMilestoneText(row.to_milestone) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.production.statusChange')" min-width="180">
                <template #default="{ row }">{{ productionStatusText(row.from_progress_status) }} -> {{ productionStatusText(row.to_progress_status) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.riskLevel')" width="120">
                <template #default="{ row }">{{ mfgStatusText(row.risk_level) }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.riskReason')" min-width="180">
                <template #default="{ row }">{{ row.risk_reason || '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.sample.operator')" width="130">
                <template #default="{ row }">{{ row.created_by_name || '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('industry.manufacturing.field.createdAt')" width="150">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </section>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="recordDialogVisible = false">{{ t('industry.manufacturing.action.cancel') }}</el-button>
        <el-button v-if="canSaveActiveRecord()" type="primary" :loading="recordSaving" @click="saveRecord">{{ t('industry.manufacturing.action.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="drilldownVisible" :title="drilldownTitle" width="820px">
      <el-table :data="drilldownItems" border v-loading="drilldownLoading">
        <el-table-column :label="t('industry.manufacturing.field.primary')" min-width="180">
          <template #default="{ row }">{{ drilldownPrimaryText(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.status')" width="140">
          <template #default="{ row }">{{ drilldownStatusText(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.customer')" min-width="160">
          <template #default="{ row }">{{ customerNameText(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.amount')" width="130">
          <template #default="{ row }">{{ money(row.amount || row.quote_amount || row.overdue_amount || 0) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
          <template #default="{ row }">{{ drilldownTimeText(row) }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="actionResultVisible" :title="actionResultTitle" width="620px">
      <div v-if="actionResultType === 'readiness'" class="mfg-readiness-result" :class="{ 'is-ready': actionResultData.ready }">
        <div class="mfg-readiness-result__hero">
          <span>{{ actionResultData.ready ? t('industry.manufacturing.readiness.readyBadge') : t('industry.manufacturing.readiness.missingBadge') }}</span>
          <strong>{{ actionResultData.ready ? t('industry.manufacturing.readiness.readyTitle') : t('industry.manufacturing.readiness.missingTitle') }}</strong>
          <p>{{ actionResultData.ready ? t('industry.manufacturing.readiness.readyDesc') : t('industry.manufacturing.readiness.missingDesc') }}</p>
        </div>
        <div v-if="!actionResultData.ready" class="mfg-readiness-result__missing">
          <strong>{{ t('industry.manufacturing.readiness.missingListTitle') }}</strong>
          <ul>
            <li v-for="item in readinessMissingItems" :key="item.key">
              <span>{{ item.label }}</span>
              <small>{{ item.action }}</small>
            </li>
          </ul>
        </div>
        <div class="mfg-readiness-result__next">
          <strong>{{ t('industry.manufacturing.readiness.nextAction') }}</strong>
          <span>{{ readinessNextActionText }}</span>
        </div>
      </div>
    </el-dialog>

    <el-drawer v-model="quoteRevisionDrawerVisible" :title="t('industry.manufacturing.quote.revisionDrawerTitle')" size="720px">
      <el-table :data="quoteRevisionItems" border v-loading="quoteRevisionLoading">
        <el-table-column :label="t('industry.manufacturing.field.revision')" width="90">
          <template #default="{ row }">
            <el-button link type="primary" @click="openQuoteDetail(row)">{{ row.revision || 1 }}</el-button>
          </template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.amount')" width="130">
          <template #default="{ row }">{{ businessAmountText(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.approvalStatus')" width="120">
          <template #default="{ row }">{{ quoteApprovalStatusText(row.approval_status) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.field.updatedAt')" width="150">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('industry.manufacturing.action.operations')" width="140" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openQuoteDiff(row)">
              {{ t('industry.manufacturing.action.versionDiff') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-drawer v-model="quoteDetailVisible" :title="t('industry.manufacturing.quote.detailTitle')" size="760px">
      <div v-if="activeQuoteDetail" class="mfg-quote-detail">
        <div class="mfg-quote-detail__header">
          <div>
            <span>{{ t('industry.manufacturing.workspace.quote') }}</span>
            <h3>{{ activeQuoteDetail.quote_no || activeQuoteDetail.id || '-' }}</h3>
            <p>{{ t('industry.manufacturing.field.revision') }} {{ activeQuoteDetail.revision || 1 }}</p>
          </div>
          <div class="mfg-quote-detail__status">
            <el-tag :type="statusTagType(activeQuoteDetail.approval_status)">{{ quoteApprovalStatusText(activeQuoteDetail.approval_status) }}</el-tag>
            <el-tag :type="quoteSendStatusTagType(activeQuoteDetail.send_status)">{{ quoteSendStatusText(activeQuoteDetail.send_status) }}</el-tag>
            <el-tag :type="quoteCustomerStatusTagType(activeQuoteDetail.status)">{{ quoteCustomerStatusText(activeQuoteDetail.status) }}</el-tag>
          </div>
        </div>
        <div class="mfg-quote-detail__metrics">
          <div v-for="item in quoteDetailMetrics" :key="item.key" class="mfg-quote-detail__metric">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
        <div class="mfg-quote-detail__section">
          <h4>{{ t('industry.manufacturing.quote.detailContext') }}</h4>
          <div class="mfg-quote-detail__grid">
            <div v-for="item in quoteDetailContextRows" :key="item.key">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
        </div>
        <div class="mfg-quote-detail__section">
          <h4>{{ t('industry.manufacturing.form.quoteTermsText') }}</h4>
          <p class="mfg-quote-detail__terms">{{ quoteTermsText(activeQuoteDetail) }}</p>
        </div>
        <div class="mfg-quote-detail__section" v-loading="quoteDetailLoading">
          <h4>{{ t('industry.manufacturing.quote.approvalHistory') }}</h4>
          <el-table v-if="quoteApprovalHistoryItems.length" :data="quoteApprovalHistoryItems" size="small" class="mfg-rfq-table">
            <el-table-column :label="t('industry.manufacturing.quote.approvalAction')" width="110">
              <template #default="{ row }">{{ quoteApprovalActionText(row.action) }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.approvalOperator')" width="130">
              <template #default="{ row }">{{ row.operator_name || (row.operator_id ? `#${row.operator_id}` : '-') }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.approvalTime')" width="150">
              <template #default="{ row }">{{ formatTime(row.operated_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.statusChange')" width="150">
              <template #default="{ row }">{{ quoteApprovalStatusText(row.old_approval_status) }} → {{ quoteApprovalStatusText(row.new_approval_status) }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.approvalRemark')" min-width="180">
              <template #default="{ row }">{{ row.remark || '-' }}</template>
            </el-table-column>
          </el-table>
          <el-empty v-else-if="!quoteDetailLoading" class="mfg-rfq-empty" :description="t('industry.manufacturing.quote.noApprovalHistory')" />
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="deliveryChangeDetailVisible" :title="deliveryChangeDetailTitle" size="760px">
      <div v-if="activeDeliveryChangeDetail" class="mfg-quote-detail" v-loading="deliveryChangeDetailLoading">
        <div class="mfg-quote-detail__header">
          <div>
            <span>{{ deliveryChangeDetailTypeText }}</span>
            <h3>{{ deliveryChangeDetailPrimary }}</h3>
            <p>{{ deliveryChangeDetailSubtitle }}</p>
          </div>
          <div class="mfg-quote-detail__status">
            <el-tag :type="statusTagType(activeDeliveryChangeDetail.status)">{{ deliveryChangeStatusText(activeDeliveryChangeDetail) }}</el-tag>
            <el-tag :type="statusTagType(activeDeliveryChangeDetail.approval_status)">{{ quoteApprovalStatusText(activeDeliveryChangeDetail.approval_status) }}</el-tag>
            <el-tag v-if="activeDeliveryChangeDetailSection === 'ecns'" :type="activeDeliveryChangeDetail.customer_confirmed ? 'success' : 'warning'">{{ ecnCustomerConfirmText(activeDeliveryChangeDetail.customer_confirmed) }}</el-tag>
          </div>
        </div>
        <div class="mfg-quote-detail__metrics">
          <div v-for="item in deliveryChangeDetailMetrics" :key="item.key" class="mfg-quote-detail__metric">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
        <div class="mfg-quote-detail__section">
          <h4>{{ t('industry.manufacturing.detail.basicInfo') }}</h4>
          <div class="mfg-quote-detail__grid">
            <div v-for="item in deliveryChangeDetailRows" :key="item.key">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
        </div>
        <div class="mfg-quote-detail__section">
          <h4>{{ t('industry.manufacturing.quote.approvalHistory') }}</h4>
          <el-table v-if="deliveryChangeApprovalHistoryItems.length" :data="deliveryChangeApprovalHistoryItems" size="small" class="mfg-rfq-table">
            <el-table-column :label="t('industry.manufacturing.quote.approvalAction')" width="110">
              <template #default="{ row }">{{ quoteApprovalActionText(row.action) }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.approvalOperator')" width="130">
              <template #default="{ row }">{{ row.operator_name || (row.operator_id ? `#${row.operator_id}` : '-') }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.approvalTime')" width="150">
              <template #default="{ row }">{{ formatTime(row.operated_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.statusChange')" width="150">
              <template #default="{ row }">{{ quoteApprovalStatusText(row.old_approval_status) }} → {{ quoteApprovalStatusText(row.new_approval_status) }}</template>
            </el-table-column>
            <el-table-column :label="t('industry.manufacturing.quote.approvalRemark')" min-width="180">
              <template #default="{ row }">{{ row.remark || '-' }}</template>
            </el-table-column>
          </el-table>
          <el-empty v-else class="mfg-rfq-empty" :description="t('industry.manufacturing.quote.noApprovalHistory')" />
        </div>
      </div>
    </el-drawer>

    <el-dialog v-model="quoteApprovalRemarkVisible" :title="quoteApprovalRemarkTitle" width="520px">
      <el-form label-position="top">
        <el-form-item :label="t('industry.manufacturing.quote.approvalRemark')" :required="quoteApprovalRemarkRequired">
          <el-input
            v-model="quoteApprovalRemark"
            type="textarea"
            :rows="4"
            :placeholder="t('industry.manufacturing.quote.approvalRemarkPlaceholder')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="quoteApprovalRemarkVisible = false">{{ t('industry.manufacturing.action.cancel') }}</el-button>
        <el-button type="primary" :loading="quoteApprovalRemarkSaving" @click="confirmQuoteApprovalRemark">{{ t('industry.manufacturing.action.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="quoteDiffVisible" :title="t('industry.manufacturing.quote.diffTitle')" width="560px">
      <div class="mfg-quote-diff">
        <div>
          <span>{{ t('industry.manufacturing.quote.amountDelta') }}</span>
          <strong>{{ money(quoteDiffData.diff?.amount_delta || 0) }}</strong>
        </div>
        <div>
          <span>{{ t('industry.manufacturing.quote.termsChanged') }}</span>
          <strong>{{ quoteDiffData.diff?.terms_changed ? t('industry.manufacturing.option.yes') : t('industry.manufacturing.option.no') }}</strong>
        </div>
        <div>
          <span>{{ t('industry.manufacturing.quote.bomChanged') }}</span>
          <strong>{{ quoteDiffData.diff?.bom_changed ? t('industry.manufacturing.option.yes') : t('industry.manufacturing.option.no') }}</strong>
        </div>
      </div>
    </el-dialog>

    <el-drawer v-model="clarificationHistoryVisible" :title="t('industry.manufacturing.rfq.answerHistoryTitle')" size="640px">
      <div class="mfg-answer-history" v-loading="clarificationHistoryLoading">
        <div class="mfg-answer-history__head">
          <span>{{ t('industry.manufacturing.rfq.clarificationCategory') }}</span>
          <strong>{{ clarificationSourceText(activeClarificationHistoryRow) }}</strong>
          <p>{{ activeClarificationHistoryRow?.question || '-' }}</p>
        </div>
        <el-table v-if="clarificationHistoryItems.length" :data="clarificationHistoryItems" size="small" class="mfg-rfq-table">
          <el-table-column :label="t('industry.manufacturing.rfq.answeredBy')" width="130">
            <template #default="{ row }">{{ answerUserName(row.answered_by, row.answered_by_name) }}</template>
          </el-table-column>
          <el-table-column :label="t('industry.manufacturing.rfq.answeredAt')" width="150">
            <template #default="{ row }">{{ formatTime(row.answered_at) }}</template>
          </el-table-column>
          <el-table-column :label="t('industry.manufacturing.rfq.answerChange')" min-width="260">
            <template #default="{ row }">
              <div class="mfg-answer-history__change">
                <span>{{ t('industry.manufacturing.rfq.oldAnswer') }}</span>
                <p>{{ row.old_answer || '-' }}</p>
                <span>{{ t('industry.manufacturing.rfq.newAnswer') }}</span>
                <p>{{ row.new_answer || '-' }}</p>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('industry.manufacturing.field.status')" width="120">
            <template #default="{ row }">{{ mfgStatusText(row.new_status || row.old_status) }}</template>
          </el-table-column>
        </el-table>
        <el-empty v-else class="mfg-rfq-empty" :description="t('industry.manufacturing.rfq.noAnswerHistory')" />
      </div>
    </el-drawer>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown, ArrowUp, Box, Calendar, ChatDotRound, Delete, Document, Download, Finished, Money, PriceTag, Tickets, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import api from '../../api'
import { t } from '../../i18n'
import { apiErrorMessage } from '../../utils/api-error'
import { payload as crmPayload } from '../../utils/customerChain'
import { downloadProtectedFile, unwrapFileUploadResponse } from '../../utils/download'
import { currentUser, hasExactPermission, hasPermission, isAdminUser } from '../../permissions'
import MfgCustomer360Panel from './components/MfgCustomer360Panel.vue'

const route = useRoute()
const router = useRouter()
const dashboardLoading = ref(false)
const metricRefreshing = ref(false)
const dashboard = ref({})
const drilldownVisible = ref(false)
const drilldownLoading = ref(false)
const drilldownItems = ref([])
const drilldownTitle = ref('')
const drilldownTarget = ref('')
const companies = ref([])
const companiesLoading = ref(false)
const companyKeyword = ref('')
const selectedCompany = ref(null)
const selectedCustomerID = ref(0)
const activeWorkspaceTab = ref('')
const productionGroupMode = ref('none')
const recordDialogVisible = ref(false)
const recordSaving = ref(false)
const activeSectionKey = ref('')
const editingRecordID = ref(0)
const activeRecordCapabilities = ref({})
const recordForm = reactive({})
const sampleProgressHistory = ref([])
const sampleProgressHistoryLoading = ref(false)
const productionProgressHistory = ref([])
const productionProgressHistoryLoading = ref(false)
const actionResultVisible = ref(false)
const actionResultType = ref('')
const actionResultData = ref({})
const quoteRevisionDrawerVisible = ref(false)
const quoteRevisionLoading = ref(false)
const quoteRevisionItems = ref([])
const quoteDetailVisible = ref(false)
const quoteDetailLoading = ref(false)
const activeQuoteDetail = ref(null)
const deliveryChangeDetailVisible = ref(false)
const deliveryChangeDetailLoading = ref(false)
const activeDeliveryChangeDetail = ref(null)
const activeDeliveryChangeDetailSection = ref('')
const quoteApprovalRemarkVisible = ref(false)
const quoteApprovalRemarkSaving = ref(false)
const quoteApprovalRemark = ref('')
const pendingQuoteApprovalAction = ref(null)
const quoteDiffVisible = ref(false)
const quoteDiffData = ref({})
const clarificationHistoryVisible = ref(false)
const clarificationHistoryLoading = ref(false)
const clarificationHistoryItems = ref([])
const activeClarificationHistoryRow = ref(null)
const bomDetailVisible = ref(false)
const activeBOMDetail = ref(null)
const rfqWorkspaceDialogVisible = ref(false)
const activeRFQWorkspaceTab = ref('rfq')
const selectedRFQID = ref(0)
const activeWorkspaceBlock = ref('rfq')
const quoteWorkspaceContext = ref({})
const quoteWorkspaceLoading = ref(false)
const rfqCompanyOptions = ref([])
const rfqContactOptions = ref([])
const rfqProductOptions = ref([])
const rfqOpportunityOptions = ref([])
const rfqDrawingUploading = ref(false)
const userOptions = ref([])
const usersLoading = ref(false)
const rfqOptionsLoading = reactive({
  companies: false,
  contacts: false,
  products: false,
  opportunities: false
})
const rfqCurrencyOptions = ['CNY', 'USD', 'EUR', 'JPY', 'HKD']
const currencyOptions = ['CNY', 'USD', 'EUR', 'JPY', 'HKD']
const rfqClarificationCategories = [
  { value: 'technical_requirements', labelKey: 'industry.manufacturing.rfq.technicalRequirements' },
  { value: 'quality_requirements', labelKey: 'industry.manufacturing.rfq.qualityRequirements' },
  { value: 'packaging_requirements', labelKey: 'industry.manufacturing.rfq.packagingRequirements' },
  { value: 'commercial_terms', labelKey: 'industry.manufacturing.rfq.commercialTerms' },
  { value: 'customer_notes', labelKey: 'industry.manufacturing.rfq.customerNotes' }
]
const sampleStageOptions = [
  'sample_need',
  'material_prepare',
  'sample_build',
  'internal_test',
  'sample_sent',
  'customer_test',
  'revision',
  'sealed_confirm',
  'mass_prepare',
  'completed',
  'paused'
]
const productionMilestoneOptions = [
  'order_confirmed',
  'scheduled',
  'material_prepare',
  'production',
  'quality_check',
  'packaging',
  'ready_to_ship',
  'shipped',
  'completed',
  'paused'
]
const productionStatusOptions = ['not_started', 'in_progress', 'paused', 'delayed', 'completed', 'canceled']
const relationOptions = reactive({
  companies: [],
  productFamilies: [],
  rfqs: [],
  bomTemplates: [],
  bom: [],
  cost: [],
  quotes: [],
  orders: [],
  users: [],
  assets: [],
  service: []
})
const relationLoading = reactive({
  companies: false,
  productFamilies: false,
  rfqs: false,
  bomTemplates: false,
  bom: false,
  cost: false,
  quotes: false,
  orders: false,
  users: false,
  assets: false,
  service: false
})

const sectionDefinitions = [
  {
    key: 'quoteWorkspace',
    endpoint: 'rfqs',
    titleKey: 'industry.manufacturing.quoteWorkspace.title',
    descKey: 'industry.manufacturing.quoteWorkspace.desc',
    primary: 'rfq_no',
    creatable: true,
    fields: [],
    actions: []
  },
  {
    key: 'agreements',
    endpoint: 'agreements',
    titleKey: 'industry.manufacturing.workspace.agreement',
    descKey: 'industry.manufacturing.workspace.agreementDesc',
    primary: 'agreement_no',
    creatable: true,
    fields: [
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('product_family_id', 'industry.manufacturing.field.productFamilyId', 'number'),
      field('year', 'industry.manufacturing.field.year', 'number'),
      field('committed_quantity', 'industry.manufacturing.field.committedQuantity', 'number'),
      field('fulfilled_quantity', 'industry.manufacturing.field.fulfilledQuantity', 'number'),
      field('target_amount', 'industry.manufacturing.field.targetAmount', 'number'),
      field('risk_level', 'industry.manufacturing.field.riskLevel', null, false, ['green', 'yellow', 'red']),
      field('terms', 'industry.manufacturing.field.terms', 'textarea', true)
    ],
    actions: []
  },
  {
    key: 'forecasts',
    endpoint: 'forecasts',
    titleKey: 'industry.manufacturing.workspace.forecast',
    descKey: 'industry.manufacturing.workspace.forecastDesc',
    primary: 'forecast_no',
    creatable: true,
    fields: [
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('product_family_id', 'industry.manufacturing.field.productFamilyId', 'number'),
      field('period', 'industry.manufacturing.field.period'),
      field('forecast_quantity', 'industry.manufacturing.field.forecastQuantity', 'number'),
      field('actual_quantity', 'industry.manufacturing.field.actualQuantity', 'number'),
      field('forecast_amount', 'industry.manufacturing.field.forecastAmount', 'number'),
      field('actual_amount', 'industry.manufacturing.field.actualAmount', 'number'),
      field('confidence_level', 'industry.manufacturing.field.confidenceLevel', null, false, ['high', 'medium', 'low']),
      field('assumption', 'industry.manufacturing.field.assumption', 'textarea', true)
    ],
    actions: []
  },
  {
    key: 'rfqs',
    endpoint: 'rfqs',
    titleKey: 'industry.manufacturing.workspace.rfq',
    descKey: 'industry.manufacturing.workspace.rfqDesc',
    primary: 'rfq_no',
    creatable: true,
    fields: [
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('product_family_id', 'industry.manufacturing.field.productFamilyId', 'number'),
      field('quantity', 'industry.manufacturing.field.quantity', 'number'),
      field('target_price', 'industry.manufacturing.field.targetPrice', 'number'),
      field('drawing_file_id', 'industry.manufacturing.field.drawingFileId'),
      field('no_drawing_reason', 'industry.manufacturing.field.noDrawingReason', 'textarea', true)
    ],
    actions: [
      action('readiness', 'industry.manufacturing.action.readinessCheck', (row) => api.post(`/api/v1/mfg/rfqs/${row.id}/readiness-check`)),
      action('submit', 'industry.manufacturing.action.submitRFQ', (row) => api.post(`/api/v1/mfg/rfqs/${row.id}/submit`)),
      action('startCosting', 'industry.manufacturing.action.startCosting', (row) => api.post(`/api/v1/mfg/rfqs/${row.id}/start-costing`))
    ]
  },
  {
    key: 'clarifications',
    endpoint: 'rfq-clarifications',
    titleKey: 'industry.manufacturing.workspace.clarification',
    descKey: 'industry.manufacturing.workspace.clarificationDesc',
    primary: 'question',
    creatable: true,
    fields: [
      field('rfq_id', 'industry.manufacturing.field.rfqId', 'number'),
      field('source_key', 'industry.manufacturing.rfq.clarificationCategory', null, false, rfqClarificationCategories.map((item) => item.value), 'clarificationCategory'),
      field('question', 'industry.manufacturing.field.question', 'textarea', true),
      field('answer', 'industry.manufacturing.field.answer', 'textarea', true),
      field('owner_id', 'industry.manufacturing.form.owner', 'user'),
      field('status', 'industry.manufacturing.field.status', null, false, ['open', 'resolved'])
    ],
    actions: []
  },
  {
    key: 'drawings',
    endpoint: 'drawings',
    titleKey: 'industry.manufacturing.workspace.drawing',
    descKey: 'industry.manufacturing.workspace.drawingDesc',
    primary: 'drawing_no',
    creatable: true,
    fields: [
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('rfq_id', 'industry.manufacturing.field.rfqId', 'number'),
      field('drawing_no', 'industry.manufacturing.field.drawingNo'),
      field('version', 'industry.manufacturing.field.version'),
      field('file_id', 'industry.manufacturing.field.fileId'),
      field('storage_node_id', 'industry.manufacturing.field.storageNodeId'),
      field('classification_level', 'industry.manufacturing.field.classificationLevel', null, false, ['L1', 'L2', 'L3', 'L4']),
      field('download_policy', 'industry.manufacturing.field.downloadPolicy', null, false, ['audit_required', 'approval_required']),
      field('approval_status', 'industry.manufacturing.field.approvalStatus', null, false, ['not_required', 'pending', 'approved']),
      field('external_send_policy', 'industry.manufacturing.field.externalSendPolicy', null, false, ['audit_required', 'approval_required'])
    ],
    actions: [
      action('viewDrawing', 'industry.manufacturing.action.viewDrawing', (row) => api.post(`/api/v1/mfg/drawings/${row.id}/view`)),
      action('downloadDrawing', 'industry.manufacturing.action.downloadDrawing', (row) => api.post(`/api/v1/mfg/drawings/${row.id}/download`)),
      action('externalSendDrawing', 'industry.manufacturing.action.externalSendDrawing', (row) => api.post(`/api/v1/mfg/drawings/${row.id}/external-send`)),
      action('approveDrawingAccess', 'industry.manufacturing.action.approveDrawingAccess', (row) => api.post(`/api/v1/mfg/drawings/${row.id}/approve-access`))
    ]
  },
  {
    key: 'product',
    endpoint: 'product-families',
    titleKey: 'industry.manufacturing.workspace.product',
    descKey: 'industry.manufacturing.workspace.productDesc',
    primary: 'name',
    creatable: true,
    fields: [
      field('family_code', 'industry.manufacturing.field.familyCode'),
      field('name', 'industry.manufacturing.field.name'),
      field('description', 'industry.manufacturing.field.description', 'textarea', true),
      field('status', 'industry.manufacturing.field.status', null, false, ['active', 'inactive'])
    ],
    actions: []
  },
  {
    key: 'configModels',
    endpoint: 'config-models',
    titleKey: 'industry.manufacturing.workspace.configModel',
    descKey: 'industry.manufacturing.workspace.configModelDesc',
    primary: 'model_no',
    creatable: true,
    fields: [
      field('product_family_id', 'industry.manufacturing.field.productFamilyId', 'number'),
      field('model_no', 'industry.manufacturing.field.modelNo'),
      field('version', 'industry.manufacturing.field.version'),
      field('parameters', 'industry.manufacturing.field.parameters', 'textarea', true),
      field('constraints', 'industry.manufacturing.field.constraints', 'textarea', true),
      field('default_values', 'industry.manufacturing.field.defaultValues', 'textarea', true),
      field('status', 'industry.manufacturing.field.status')
    ],
    actions: []
  },
  {
    key: 'bomTemplates',
    endpoint: 'bom-templates',
    titleKey: 'industry.manufacturing.workspace.bomTemplate',
    descKey: 'industry.manufacturing.workspace.bomTemplateDesc',
    primary: 'template_no',
    creatable: true,
    fields: [
      field('product_family_id', 'industry.manufacturing.field.productFamilyId', 'number'),
      field('template_no', 'industry.manufacturing.field.templateNo'),
      field('version', 'industry.manufacturing.field.version'),
      field('rule', 'industry.manufacturing.field.rule', 'textarea', true),
      field('status', 'industry.manufacturing.field.status')
    ],
    actions: []
  },
  {
    key: 'bom',
    endpoint: 'bom-snapshots',
    titleKey: 'industry.manufacturing.workspace.bom',
    descKey: 'industry.manufacturing.workspace.bomDesc',
    primary: 'snapshot_no',
    creatable: true,
    createAction: (payload) => api.post('/api/v1/mfg/bom-snapshots/from-config', normalizeBOMSnapshotPayload(payload)),
    fields: [
      field('source_template_id', 'industry.manufacturing.field.sourceTemplateId', 'number'),
      field('source_config_version', 'industry.manufacturing.field.version'),
      field('rfq_id', 'industry.manufacturing.field.rfqId', 'number'),
      field('bom_lines', 'industry.manufacturing.field.bomLines', 'textarea', true),
      field('config_values', 'industry.manufacturing.field.configValues', 'textarea', true)
    ],
    actions: []
  },
  {
    key: 'cost',
    endpoint: 'cost-sheets',
    titleKey: 'industry.manufacturing.workspace.cost',
    descKey: 'industry.manufacturing.workspace.costDesc',
    primary: 'margin_level',
    creatable: true,
    createAction: (payload) => api.post('/api/v1/mfg/cost-sheets/calculate', payload),
    fields: [
      field('rfq_id', 'industry.manufacturing.field.rfqId', 'number'),
      field('bom_snapshot_id', 'industry.manufacturing.field.bomSnapshotId', 'number'),
      field('quote_amount', 'industry.manufacturing.field.quoteAmount', 'number'),
      field('material_cost', 'industry.manufacturing.field.materialCost', 'number'),
      field('labor_cost', 'industry.manufacturing.field.laborCost', 'number'),
      field('outsourcing_cost', 'industry.manufacturing.field.outsourcingCost', 'number'),
      field('manufacturing_overhead', 'industry.manufacturing.field.manufacturingOverhead', 'number')
    ],
    actions: []
  },
  {
    key: 'quotes',
    endpoint: 'quotes',
    titleKey: 'industry.manufacturing.workspace.quote',
    descKey: 'industry.manufacturing.workspace.quoteDesc',
    primary: 'quote_no',
    creatable: true,
    createAction: (payload) => api.post('/api/v1/mfg/quotes', payload),
    fields: [
      field('rfq_id', 'industry.manufacturing.field.rfqId', 'number'),
      field('cost_sheet_id', 'industry.manufacturing.field.costSheetId', 'number'),
      field('bom_snapshot_id', 'industry.manufacturing.field.bomSnapshotId', 'number'),
      field('amount', 'industry.manufacturing.field.amount', 'number'),
      field('currency', 'industry.manufacturing.field.currency'),
      field('terms', 'industry.manufacturing.field.terms', 'textarea', true)
    ],
    actions: []
  },
  {
    key: 'samples',
    endpoint: 'sample-projects',
    titleKey: 'industry.manufacturing.workspace.sample',
    descKey: 'industry.manufacturing.workspace.sampleDesc',
    primary: 'sample_no',
    creatable: true,
    fields: [
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('rfq_id', 'industry.manufacturing.field.rfqId', 'number'),
      field('bom_snapshot_id', 'industry.manufacturing.field.bomSnapshotId', 'number'),
      field('stage', 'industry.manufacturing.field.stage'),
      field('risk_level', 'industry.manufacturing.field.riskLevel', null, false, ['green', 'yellow', 'red']),
      field('customer_feedback', 'industry.manufacturing.field.customerFeedback', 'textarea', true)
    ],
    actions: [
      action('testRecord', 'industry.manufacturing.action.testRecord', (row) => api.post(`/api/v1/mfg/sample-projects/${row.id}/test-records`, { test_item: row.sample_no, test_result: 'passed' }))
    ]
  },
  {
    key: 'production',
    endpoint: 'production-progress',
    titleKey: 'industry.manufacturing.workspace.production',
    descKey: 'industry.manufacturing.workspace.productionDesc',
    primary: 'milestone',
    creatable: true,
    fields: [
      field('order_id', 'industry.manufacturing.field.orderId', 'number'),
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('sales_owner_id', 'industry.manufacturing.field.salesOwnerId', 'number'),
      field('erp_order_no', 'industry.manufacturing.field.erpOrderNo'),
      field('mes_work_order_no', 'industry.manufacturing.field.mesWorkOrderNo'),
      field('milestone', 'industry.manufacturing.field.milestone'),
      field('progress_status', 'industry.manufacturing.production.progressStatus', null, false, productionStatusOptions, 'productionStatus'),
      field('planned_at', 'industry.manufacturing.field.plannedAt'),
      field('actual_at', 'industry.manufacturing.field.actualAt'),
      field('risk_level', 'industry.manufacturing.field.riskLevel', null, false, ['green', 'yellow', 'red']),
      field('risk_reason', 'industry.manufacturing.field.riskReason', 'textarea', true)
    ],
    actions: [
      action('raiseRisk', 'industry.manufacturing.action.raiseRisk', (row) => api.post(`/api/v1/mfg/production-progress/${row.id}/raise-risk`, { risk_level: 'red', risk_reason: row.risk_reason || 'manual' }))
    ]
  },
  {
    key: 'ecns',
    endpoint: 'ecns',
    titleKey: 'industry.manufacturing.workspace.ecn',
    descKey: 'industry.manufacturing.workspace.ecnDesc',
    primary: 'ecn_no',
    creatable: true,
    fields: [
      field('order_id', 'industry.manufacturing.field.orderId', 'number'),
      field('old_bom_snapshot_id', 'industry.manufacturing.field.oldBomSnapshotId', 'number'),
      field('new_bom_snapshot_id', 'industry.manufacturing.field.newBomSnapshotId', 'number'),
      field('change_reason', 'industry.manufacturing.field.changeReason', 'textarea', true),
      field('impact_level', 'industry.manufacturing.field.impactLevel', null, false, ['normal', 'major', 'critical'], 'mfgStatus'),
      field('cost_impact', 'industry.manufacturing.field.costImpact', 'number'),
      field('cost_impact_currency', 'industry.manufacturing.field.costImpactCurrency', null, false, currencyOptions),
      field('delivery_impact_days', 'industry.manufacturing.field.deliveryImpactDays', 'number'),
      field('impact_scope', 'industry.manufacturing.field.impactScope', 'textarea', true)
    ],
    actions: []
  },
  {
    key: 'compensationRecords',
    endpoint: 'compensation-records',
    titleKey: 'industry.manufacturing.workspace.compensationHistory',
    descKey: 'industry.manufacturing.workspace.compensationHistoryDesc',
    primary: 'compensation_no',
    creatable: true,
    fields: [
      field('order_id', 'industry.manufacturing.field.orderId', 'number'),
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('sales_owner_id', 'industry.manufacturing.field.salesOwnerId', 'number'),
      field('reason', 'industry.manufacturing.field.compensationReason', 'textarea', true),
      field('amount', 'industry.manufacturing.field.compensationAmount', 'number'),
      field('currency', 'industry.manufacturing.field.currency', null, false, currencyOptions),
      field('responsibility_split', 'industry.manufacturing.field.responsibilitySplit', 'textarea', true),
      field('improvement_actions', 'industry.manufacturing.field.improvementActions', 'textarea', true),
      field('compensation_date', 'industry.manufacturing.field.compensationDate')
    ],
    actions: []
  },
  {
    key: 'assets',
    endpoint: 'assets',
    titleKey: 'industry.manufacturing.workspace.asset',
    descKey: 'industry.manufacturing.workspace.assetDesc',
    primary: 'serial_no',
    creatable: true,
    fields: [
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('order_id', 'industry.manufacturing.field.orderId', 'number'),
      field('bom_snapshot_id', 'industry.manufacturing.field.bomSnapshotId', 'number'),
      field('serial_no', 'industry.manufacturing.field.serialNo'),
      field('batch_no', 'industry.manufacturing.field.batchNo'),
      field('install_location', 'industry.manufacturing.field.installLocation'),
      field('iot_asset_ref', 'industry.manufacturing.field.iotAssetRef'),
      field('iot_authorized', 'industry.manufacturing.field.iotAuthorized', null, false, [true, false])
    ],
    actions: [
      action('serviceHistory', 'industry.manufacturing.action.serviceHistory', (row) => api.get(`/api/v1/mfg/assets/${row.id}/service-history`)),
      action('customerView', 'industry.manufacturing.action.customerView', (row) => api.get(`/api/v1/mfg/assets/${row.id}/customer-view`)),
      action('authorizeIoT', 'industry.manufacturing.action.authorizeIoT', (row) => api.post(`/api/v1/mfg/assets/${row.id}/iot-authorize`, { iot_asset_ref: row.iot_asset_ref })),
      action('revokeIoT', 'industry.manufacturing.action.revokeIoT', (row) => api.post(`/api/v1/mfg/assets/${row.id}/iot-revoke`))
    ]
  },
  {
    key: 'service',
    endpoint: 'service-tickets',
    titleKey: 'industry.manufacturing.workspace.service',
    descKey: 'industry.manufacturing.workspace.serviceDesc',
    primary: 'fault_code',
    creatable: true,
    fields: [
      field('asset_id', 'industry.manufacturing.field.assetId', 'number'),
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('fault_code', 'industry.manufacturing.field.faultCode'),
      field('service_type', 'industry.manufacturing.field.serviceType'),
      field('service_result', 'industry.manufacturing.field.serviceResult', 'textarea', true),
      field('status', 'industry.manufacturing.field.status')
    ],
    actions: [
      action('sparePartQuote', 'industry.manufacturing.action.sparePartQuote', (row) => api.post(`/api/v1/mfg/service-tickets/${row.id}/spare-part-quote`, { asset_id: row.asset_id, amount: 0 })),
      action('createOpportunity', 'industry.manufacturing.action.createOpportunity', (row) => api.post(`/api/v1/mfg/service-tickets/${row.id}/create-opportunity`, { name: row.fault_code, amount: 0 }))
    ]
  },
  {
    key: 'knowledge',
    endpoint: 'knowledge',
    titleKey: 'industry.manufacturing.workspace.knowledge',
    descKey: 'industry.manufacturing.workspace.knowledgeDesc',
    primary: 'title',
    creatable: true,
    fields: [
      field('title', 'industry.manufacturing.field.title'),
      field('knowledge_type', 'industry.manufacturing.field.knowledgeType', null, false, ['quote', 'drawing', 'process', 'sample', 'service', 'supply']),
      field('scene', 'industry.manufacturing.field.scene'),
      field('customer_id', 'industry.manufacturing.field.customerId', 'number'),
      field('product_family_id', 'industry.manufacturing.field.productFamilyId', 'number'),
      field('problem', 'industry.manufacturing.field.problem', 'textarea', true),
      field('root_cause', 'industry.manufacturing.field.rootCause', 'textarea', true),
      field('solution', 'industry.manufacturing.field.solution', 'textarea', true),
      field('risk_tip', 'industry.manufacturing.field.riskTip', 'textarea', true),
      field('tags', 'industry.manufacturing.field.tags')
    ],
    actions: []
  },
  {
    key: 'integrationOutbox',
    endpoint: 'integration-outbox',
    titleKey: 'industry.manufacturing.workspace.integrationOutbox',
    descKey: 'industry.manufacturing.workspace.integrationOutboxDesc',
    primary: 'event_type',
    creatable: false,
    fields: [],
    actions: [
      action('retryIntegration', 'industry.manufacturing.action.retryIntegration', (row) => api.post(`/api/v1/mfg/integration-outbox/${row.id}/retry`)),
      action('deadLetterIntegration', 'industry.manufacturing.action.deadLetterIntegration', (row) => api.post(`/api/v1/mfg/integration-outbox/${row.id}/dead-letter`, { reason: 'manual_review' }))
    ]
  }
]

const sections = reactive(sectionDefinitions.map((item) => ({
  ...item,
  keyword: '',
  items: [],
  total: 0,
  loading: false
})))

const manufacturingFormRegistry = {
  agreements: formSpec('industry.manufacturing.form.agreementTitle', 'industry.manufacturing.form.agreementDesc', [
    group('industry.manufacturing.form.customerContext', [
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      relationField('product_family_id', 'industry.manufacturing.workspace.product', 'productFamilies'),
      numberField('year', 'industry.manufacturing.field.year')
    ]),
    group('industry.manufacturing.form.commitment', [
      numberField('committed_quantity', 'industry.manufacturing.field.committedQuantity'),
      numberField('fulfilled_quantity', 'industry.manufacturing.field.fulfilledQuantity'),
      numberField('target_amount', 'industry.manufacturing.field.targetAmount'),
      selectField('risk_level', 'industry.manufacturing.field.riskLevel', ['green', 'yellow', 'red']),
      textAreaField('terms', 'industry.manufacturing.form.agreementTerms')
    ])
  ]),
  forecasts: formSpec('industry.manufacturing.form.forecastTitle', 'industry.manufacturing.form.forecastDesc', [
    group('industry.manufacturing.form.customerContext', [
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      relationField('product_family_id', 'industry.manufacturing.workspace.product', 'productFamilies'),
      textField('period', 'industry.manufacturing.field.period')
    ]),
    group('industry.manufacturing.form.forecastPlan', [
      numberField('forecast_quantity', 'industry.manufacturing.field.forecastQuantity'),
      numberField('actual_quantity', 'industry.manufacturing.field.actualQuantity'),
      numberField('forecast_amount', 'industry.manufacturing.field.forecastAmount'),
      numberField('actual_amount', 'industry.manufacturing.field.actualAmount'),
      selectField('confidence_level', 'industry.manufacturing.field.confidenceLevel', ['high', 'medium', 'low']),
      textAreaField('assumption', 'industry.manufacturing.field.assumption')
    ])
  ]),
  clarifications: formSpec('industry.manufacturing.form.clarificationTitle', 'industry.manufacturing.form.clarificationDesc', [
    group('industry.manufacturing.form.context', [
      relationField('rfq_id', 'industry.manufacturing.field.rfq', 'rfqs'),
      clarificationCategoryField('source_key', 'industry.manufacturing.rfq.clarificationCategory'),
      selectField('status', 'industry.manufacturing.field.status', ['open', 'resolved'])
    ]),
    group('industry.manufacturing.form.clarificationContent', [
      textAreaField('question', 'industry.manufacturing.field.question'),
      textAreaField('answer', 'industry.manufacturing.field.answer'),
      userField('owner_id', 'industry.manufacturing.form.owner')
    ])
  ]),
  drawings: formSpec('industry.manufacturing.form.drawingTitle', 'industry.manufacturing.form.drawingDesc', [
    group('industry.manufacturing.form.context', [
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      relationField('rfq_id', 'industry.manufacturing.field.rfq', 'rfqs'),
      textField('drawing_no', 'industry.manufacturing.field.drawingNo'),
      textField('version', 'industry.manufacturing.field.version')
    ]),
    group('industry.manufacturing.form.drawingControl', [
      textField('file_name', 'industry.manufacturing.rfq.drawingUpload'),
      selectField('classification_level', 'industry.manufacturing.field.classificationLevel', ['L1', 'L2', 'L3', 'L4']),
      selectField('download_policy', 'industry.manufacturing.field.downloadPolicy', ['audit_required', 'approval_required']),
      selectField('approval_status', 'industry.manufacturing.field.approvalStatus', ['not_required', 'pending', 'approved']),
      selectField('external_send_policy', 'industry.manufacturing.field.externalSendPolicy', ['audit_required', 'approval_required'])
    ])
  ]),
  product: formSpec('industry.manufacturing.form.productFamilyTitle', 'industry.manufacturing.form.productFamilyDesc', [
    group('industry.manufacturing.form.baseInfo', [
      textField('family_code', 'industry.manufacturing.field.familyCode'),
      textField('name', 'industry.manufacturing.field.name'),
      selectField('status', 'industry.manufacturing.field.status', ['active', 'inactive']),
      textAreaField('description', 'industry.manufacturing.field.description')
    ])
  ]),
  configModels: formSpec('industry.manufacturing.form.configModelTitle', 'industry.manufacturing.form.configModelDesc', [
    group('industry.manufacturing.form.context', [
      relationField('product_family_id', 'industry.manufacturing.workspace.product', 'productFamilies'),
      textField('model_no', 'industry.manufacturing.field.modelNo'),
      textField('version', 'industry.manufacturing.field.version'),
      selectField('status', 'industry.manufacturing.field.status', ['draft', 'active', 'inactive'])
    ]),
    group('industry.manufacturing.form.configRules', [
      textAreaField('parameters', 'industry.manufacturing.form.parametersText'),
      textAreaField('constraints', 'industry.manufacturing.form.constraintsText'),
      textAreaField('default_values', 'industry.manufacturing.form.defaultValuesText')
    ])
  ]),
  bomTemplates: formSpec('industry.manufacturing.form.bomTemplateTitle', 'industry.manufacturing.form.bomTemplateDesc', [
    group('industry.manufacturing.form.context', [
      relationField('product_family_id', 'industry.manufacturing.workspace.product', 'productFamilies'),
      textField('template_no', 'industry.manufacturing.field.templateNo'),
      textField('version', 'industry.manufacturing.field.version'),
      selectField('status', 'industry.manufacturing.field.status', ['draft', 'active', 'inactive'])
    ])
  ]),
  bom: formSpec('industry.manufacturing.form.bomSnapshotTitle', 'industry.manufacturing.form.bomSnapshotDesc', []),
  cost: formSpec('industry.manufacturing.form.costTitle', 'industry.manufacturing.form.costDesc', [
    group('industry.manufacturing.form.context', [
      relationField('rfq_id', 'industry.manufacturing.field.rfq', 'rfqs'),
      readonlyField('bom_snapshot_id', 'industry.manufacturing.workspace.bom', 'bom')
    ]),
    group('industry.manufacturing.form.costBreakdown', [
      selectField('currency', 'industry.manufacturing.field.currency', currencyOptions),
      numberField('quote_amount', 'industry.manufacturing.field.quoteTotal', { suffix: 'currency' }),
      numberField('material_cost', 'industry.manufacturing.field.materialCost', { suffix: 'currency' }),
      numberField('labor_cost', 'industry.manufacturing.field.laborCost', { suffix: 'currency' }),
      numberField('outsourcing_cost', 'industry.manufacturing.field.outsourcingCost', { suffix: 'currency' }),
      numberField('manufacturing_overhead', 'industry.manufacturing.field.manufacturingOverhead', { suffix: 'currency' })
    ])
  ]),
  quotes: formSpec('industry.manufacturing.form.quoteTitle', 'industry.manufacturing.form.quoteDesc', [
    group('industry.manufacturing.form.context', [
      relationField('rfq_id', 'industry.manufacturing.field.rfq', 'rfqs'),
      relationField('cost_sheet_id', 'industry.manufacturing.workspace.cost', 'cost'),
      readonlyField('bom_snapshot_id', 'industry.manufacturing.workspace.bom', 'bom')
    ]),
    group('industry.manufacturing.form.quoteTerms', [
      numberField('amount', 'industry.manufacturing.field.quoteTotal', { suffix: 'currency' }),
      textField('currency', 'industry.manufacturing.field.currency'),
      textAreaField('terms', 'industry.manufacturing.form.quoteTermsText')
    ])
  ]),
  samples: formSpec('industry.manufacturing.form.sampleTitle', 'industry.manufacturing.form.sampleDesc', [
    group('industry.manufacturing.form.context', [
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      relationField('rfq_id', 'industry.manufacturing.field.rfq', 'rfqs'),
      relationField('bom_snapshot_id', 'industry.manufacturing.workspace.bom', 'bom')
    ]),
    group('industry.manufacturing.form.sampleProgress', [
      creatableSelectField('stage', 'industry.manufacturing.field.stage', sampleStageOptions, 'sampleStage'),
      selectField('risk_level', 'industry.manufacturing.field.riskLevel', ['green', 'yellow', 'red']),
      textAreaField('customer_feedback', 'industry.manufacturing.field.customerFeedback')
    ])
  ]),
  production: formSpec('industry.manufacturing.form.productionTitle', 'industry.manufacturing.form.productionDesc', [
    group('industry.manufacturing.form.context', [
      relationField('order_id', 'industry.manufacturing.form.order', 'orders'),
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      userField('sales_owner_id', 'industry.manufacturing.field.salesOwner'),
      textField('erp_order_no', 'industry.manufacturing.field.erpOrderNo'),
      textField('mes_work_order_no', 'industry.manufacturing.field.mesWorkOrderNo')
    ]),
    group('industry.manufacturing.form.deliveryRisk', [
      creatableSelectField('milestone', 'industry.manufacturing.field.milestone', productionMilestoneOptions, 'productionMilestone'),
      selectField('progress_status', 'industry.manufacturing.production.progressStatus', productionStatusOptions, 'productionStatus'),
      dateTimeField('planned_at', 'industry.manufacturing.field.plannedAt'),
      dateTimeField('actual_at', 'industry.manufacturing.field.actualAt'),
      selectField('risk_level', 'industry.manufacturing.field.riskLevel', ['green', 'yellow', 'red']),
      textAreaField('risk_reason', 'industry.manufacturing.field.riskReason')
    ])
  ]),
  ecns: formSpec('industry.manufacturing.form.ecnTitle', 'industry.manufacturing.form.ecnDesc', [
    group('industry.manufacturing.form.context', [
      relationField('order_id', 'industry.manufacturing.form.order', 'orders'),
      relationField('old_bom_snapshot_id', 'industry.manufacturing.form.oldBomSnapshot', 'bom'),
      relationField('new_bom_snapshot_id', 'industry.manufacturing.form.newBomSnapshot', 'bom')
    ]),
    group('industry.manufacturing.form.changeImpact', [
      textAreaField('change_reason', 'industry.manufacturing.field.changeReason'),
      selectField('impact_level', 'industry.manufacturing.field.impactLevel', ['normal', 'major', 'critical'], 'mfgStatus'),
      numberField('cost_impact', 'industry.manufacturing.field.costImpact'),
      selectField('cost_impact_currency', 'industry.manufacturing.field.costImpactCurrency', currencyOptions),
      numberField('delivery_impact_days', 'industry.manufacturing.field.deliveryImpactDays'),
      textAreaField('impact_scope', 'industry.manufacturing.field.impactScope')
    ])
  ]),
  compensationRecords: formSpec('industry.manufacturing.form.compensationTitle', 'industry.manufacturing.form.compensationDesc', [
    group('industry.manufacturing.form.context', [
      relationField('order_id', 'industry.manufacturing.form.order', 'orders'),
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      userField('sales_owner_id', 'industry.manufacturing.field.salesOwner'),
      selectField('status', 'industry.manufacturing.field.status', ['open', 'processing', 'closed'])
    ]),
    group('industry.manufacturing.form.compensationDetail', [
      textAreaField('reason', 'industry.manufacturing.field.compensationReason'),
      numberField('amount', 'industry.manufacturing.field.compensationAmount'),
      selectField('currency', 'industry.manufacturing.field.currency', currencyOptions),
      dateField('compensation_date', 'industry.manufacturing.field.compensationDate'),
      textAreaField('responsibility_split', 'industry.manufacturing.field.responsibilitySplit'),
      textAreaField('improvement_actions', 'industry.manufacturing.field.improvementActions')
    ])
  ]),
  assets: formSpec('industry.manufacturing.form.assetTitle', 'industry.manufacturing.form.assetDesc', [
    group('industry.manufacturing.form.context', [
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      relationField('order_id', 'industry.manufacturing.form.order', 'orders'),
      relationField('bom_snapshot_id', 'industry.manufacturing.workspace.bom', 'bom')
    ]),
    group('industry.manufacturing.form.assetInfo', [
      textField('serial_no', 'industry.manufacturing.field.serialNo'),
      textField('batch_no', 'industry.manufacturing.field.batchNo'),
      textField('install_location', 'industry.manufacturing.field.installLocation'),
      textField('iot_asset_ref', 'industry.manufacturing.field.iotAssetRef'),
      selectField('iot_authorized', 'industry.manufacturing.field.iotAuthorized', [true, false])
    ])
  ]),
  service: formSpec('industry.manufacturing.form.serviceTitle', 'industry.manufacturing.form.serviceDesc', [
    group('industry.manufacturing.form.context', [
      relationField('asset_id', 'industry.manufacturing.workspace.asset', 'assets'),
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies')
    ]),
    group('industry.manufacturing.form.serviceInfo', [
      textField('fault_code', 'industry.manufacturing.field.faultCode'),
      textField('service_type', 'industry.manufacturing.field.serviceType'),
      selectField('status', 'industry.manufacturing.field.status', ['open', 'processing', 'resolved', 'closed']),
      textAreaField('service_result', 'industry.manufacturing.field.serviceResult')
    ])
  ]),
  knowledge: formSpec('industry.manufacturing.form.knowledgeTitle', 'industry.manufacturing.form.knowledgeDesc', [
    group('industry.manufacturing.form.context', [
      relationField('customer_id', 'industry.manufacturing.field.customer', 'companies'),
      relationField('product_family_id', 'industry.manufacturing.workspace.product', 'productFamilies'),
      selectField('knowledge_type', 'industry.manufacturing.field.knowledgeType', ['quote', 'drawing', 'process', 'sample', 'service', 'supply'])
    ]),
    group('industry.manufacturing.form.knowledgeContent', [
      textField('title', 'industry.manufacturing.field.title'),
      textField('scene', 'industry.manufacturing.field.scene'),
      textAreaField('problem', 'industry.manufacturing.field.problem'),
      textAreaField('root_cause', 'industry.manufacturing.field.rootCause'),
      textAreaField('solution', 'industry.manufacturing.field.solution'),
      textAreaField('risk_tip', 'industry.manufacturing.field.riskTip'),
      textField('tags', 'industry.manufacturing.field.tags')
    ])
  ])
}

const workflowDefinitions = {
  home: {
    titleKey: 'industry.manufacturing.workflow.home',
    descKey: 'industry.manufacturing.workflow.homeDesc',
    sections: [],
    steps: []
  },
  'quote-to-order': {
    titleKey: 'industry.manufacturing.workflow.quoteToOrder',
    descKey: 'industry.manufacturing.workflow.quoteToOrderDesc',
    sections: ['quoteWorkspace'],
    steps: ['rfq', 'clarification', 'bom', 'cost', 'quote', 'order']
  },
  'sample-to-mass': {
    titleKey: 'industry.manufacturing.workflow.sampleToMass',
    descKey: 'industry.manufacturing.workflow.sampleToMassDesc',
    sections: ['samples', 'production'],
    steps: ['sampleNeed', 'sampleBuild', 'test', 'seal', 'mass']
  },
  'delivery-change': {
    titleKey: 'industry.manufacturing.workflow.deliveryChange',
    descKey: 'industry.manufacturing.workflow.deliveryChangeDesc',
    sections: ['ecns', 'compensationRecords'],
    steps: ['orderRisk', 'customerImpact', 'ecn', 'approval', 'followUp']
  },
  'asset-service': {
    titleKey: 'industry.manufacturing.workflow.assetService',
    descKey: 'industry.manufacturing.workflow.assetServiceDesc',
    sections: ['assets', 'service', 'knowledge'],
    steps: ['asset', 'warranty', 'ticket', 'spare', 'upsell']
  },
  analytics: {
    titleKey: 'industry.manufacturing.workflow.analytics',
    descKey: 'industry.manufacturing.workflow.analyticsDesc',
    sections: ['agreements', 'forecasts', 'rfqs', 'quotes'],
    steps: ['account', 'agreement', 'forecast', 'margin', 'review']
  },
  settings: {
    titleKey: 'industry.manufacturing.workflow.settings',
    descKey: 'industry.manufacturing.workflow.settingsDesc',
    sections: ['product', 'bomTemplates'],
    steps: ['productFamily', 'template']
  }
}

const moduleSectionMap = {
  customers: { workflow: 'home', tab: 'customer360' },
  dashboard: { workflow: 'home', tab: '' },
  analytics: { workflow: 'home', tab: '' },
  'asset-service': { workflow: 'home', tab: '' },
  rfqs: { workflow: 'quote-to-order', tab: 'quoteWorkspace', block: 'rfq' },
  clarifications: { workflow: 'quote-to-order', tab: 'quoteWorkspace', block: 'clarifications' },
  drawings: { workflow: 'quote-to-order', tab: 'quoteWorkspace', block: 'rfq' },
  'product-families': { workflow: 'settings', tab: 'product' },
  'bom-templates': { workflow: 'settings', tab: 'bomTemplates' },
  'bom-snapshots': { workflow: 'quote-to-order', tab: 'quoteWorkspace', block: 'bom' },
  'cost-sheets': { workflow: 'quote-to-order', tab: 'quoteWorkspace', block: 'cost' },
  quotes: { workflow: 'quote-to-order', tab: 'quoteWorkspace', block: 'quotes' },
  samples: { workflow: 'sample-to-mass', tab: 'samples' },
  production: { workflow: 'sample-to-mass', tab: 'production' },
  ecns: { workflow: 'delivery-change', tab: 'ecns' },
  'compensation-records': { workflow: 'delivery-change', tab: 'compensationRecords' }
}

const canUsePriceQuickAccess = computed(() => (
  hasExactPermission('mfg:cost:view_sensitive') ||
  hasExactPermission('mfg:cost:manage') ||
  hasExactPermission('mfg:quote:manage') ||
  hasExactPermission('mfg:quote:approve')
))
const canRefreshMetricSnapshots = computed(() => hasPermission('mfg:metrics:manage'))

function sectionDescription(section) {
  if (section?.key === 'quoteWorkspace' && !canUsePriceQuickAccess.value) {
    return t('industry.manufacturing.quoteWorkspace.readOnlyDesc')
  }
  return t(section?.descKey || '')
}

const currentModule = computed(() => String(route.params.module || 'home'))
const resolvedWorkflow = computed(() => moduleSectionMap[currentModule.value]?.workflow || currentModule.value)
const currentWorkflow = computed(() => workflowDefinitions[resolvedWorkflow.value] || workflowDefinitions.home)
const isHomeWorkflow = computed(() => resolvedWorkflow.value === 'home')
const isCustomerWorkflow = computed(() => false)
const visibleSections = computed(() => {
  const keys = currentWorkflow.value.sections || []
  return sections.filter((item) => keys.includes(item.key))
})
const priceWorkspaceBlocks = ['cost', 'quotes']
const currentSection = computed(() => visibleSections.value[0])
const currentWorkspaceTitle = computed(() => {
  return t(currentWorkflow.value.titleKey || 'industry.manufacturing.menu.title')
})
const currentWorkspaceDesc = computed(() => {
  return t(currentWorkflow.value.descKey || 'industry.manufacturing.menu.desc')
})
const activeSection = computed(() => sections.find((item) => item.key === activeSectionKey.value))
const activeSectionFields = computed(() => activeSection.value?.fields || [])
const activeFormSpec = computed(() => manufacturingFormRegistry[activeSection.value?.key] || null)
const activeBusinessGroups = computed(() => activeFormSpec.value?.groups || [])
const productionSection = computed(() => sections.find((item) => item.key === 'production'))
const productionGroupOptions = computed(() => [
  { label: t('industry.manufacturing.production.groupNone'), value: 'none' },
  { label: t('industry.manufacturing.production.groupByCustomer'), value: 'customer' },
  { label: t('industry.manufacturing.production.groupBySalesOwner'), value: 'sales_owner' }
])
const productionGroupedItems = computed(() => {
  const items = productionSection.value?.items || []
  if (productionGroupMode.value === 'customer') return groupRowsBy(items, customerNameText, 'customer')
  if (productionGroupMode.value === 'sales_owner') return groupRowsBy(items, salesOwnerNameText, 'sales_owner')
  return [{ key: 'all', label: t('industry.manufacturing.production.groupAll'), items }]
})
const rfqSection = computed(() => sections.find((item) => item.key === 'rfqs'))
const clarificationsSection = computed(() => sections.find((item) => item.key === 'clarifications'))
const bomSection = computed(() => sections.find((item) => item.key === 'bom'))
const costSection = computed(() => sections.find((item) => item.key === 'cost'))
const quoteSection = computed(() => sections.find((item) => item.key === 'quotes'))
const currentQuoteWorkspaceSection = computed(() => sections.find((item) => item.key === 'quoteWorkspace'))
const quoteWorkspaceRFQ = computed(() => quoteWorkspaceContext.value?.rfq || currentQuoteWorkspaceSection.value?.items?.find((item) => Number(item.id) === Number(selectedRFQID.value)) || {})
const quoteWorkspacePermissions = computed(() => quoteWorkspaceContext.value?.permissions || {})
const canCreateWorkspaceBOM = computed(() => Boolean(quoteWorkspacePermissions.value.can_create_bom ?? quoteWorkspacePermissions.value.can_manage_bom ?? quoteWorkspaceRFQ.value?.can_create_bom))
const canCreateWorkspaceCost = computed(() => Boolean(quoteWorkspacePermissions.value.can_create_cost ?? quoteWorkspacePermissions.value.can_manage_cost ?? quoteWorkspaceRFQ.value?.can_create_cost))
const canCreateWorkspaceQuote = computed(() => Boolean(quoteWorkspacePermissions.value.can_create_quote ?? quoteWorkspacePermissions.value.can_manage_quote ?? quoteWorkspaceRFQ.value?.can_create_quote))
const quoteWorkspaceNextActionText = computed(() => readinessActionLabel(quoteWorkspaceContext.value?.next_action || quoteWorkspaceContext.value?.readiness?.next_action || 'start_costing'))
const canViewWorkspacePrice = computed(() => quoteWorkspacePermissions.value.can_view_price !== false && !quoteWorkspacePermissions.value.price_masked)
const rfqWorkspaceSummaryRows = computed(() => {
  const rfq = quoteWorkspaceRFQ.value || {}
  return [
    { key: 'rfq', label: t('industry.manufacturing.field.rfq'), value: rfqDisplayName(rfq) || selectedRFQID.value || '-', icon: Document, tone: 'blue' },
    { key: 'customer', label: t('industry.manufacturing.field.customer'), value: rfqCustomerText(rfq), icon: User, tone: 'cyan' },
    { key: 'target', label: t('industry.manufacturing.rfq.table.targetPrice'), value: rfqTargetPriceText(rfq), icon: PriceTag, tone: 'amber' },
    { key: 'delivery', label: t('industry.manufacturing.rfq.table.deliveryDate'), value: formatDate(rfq.required_delivery_date), icon: Calendar, tone: 'green' },
    { key: 'status', label: t('industry.manufacturing.field.status'), value: mfgStatusText(rfq.status), icon: Finished, tone: 'blue' },
    { key: 'next', label: t('industry.manufacturing.costing.nextAction'), value: quoteWorkspaceNextActionText.value, icon: Tickets, tone: 'cyan' }
  ]
})
const quoteWorkspaceMissingItems = computed(() => {
  const missing = Array.isArray(quoteWorkspaceContext.value?.readiness?.missing) ? quoteWorkspaceContext.value.readiness.missing : []
  return missing.map((key) => ({
    key,
    label: readinessMissingLabel(key),
    action: readinessMissingAction(key)
  }))
})
watch(canViewWorkspacePrice, (allowed) => {
  if (!allowed && priceWorkspaceBlocks.includes(activeRFQWorkspaceTab.value)) activeRFQWorkspaceTab.value = 'rfq'
})
const currentBOMSnapshot = computed(() => {
  const current = quoteWorkspaceContext.value?.current_bom_snapshot
  if (current && typeof current === 'object' && Object.keys(current).length) return current
  return (quoteWorkspaceContext.value?.bom_snapshots || []).find((item) => String(item.status || '').toLowerCase() === 'active') || null
})
const bomSnapshotHistory = computed(() => {
  const history = quoteWorkspaceContext.value?.bom_snapshot_history
  if (Array.isArray(history)) return history
  const currentID = Number(currentBOMSnapshot.value?.id || 0)
  return (quoteWorkspaceContext.value?.bom_snapshots || []).filter((item) => Number(item.id || 0) !== currentID)
})
const currentCostSheet = computed(() => {
  if (!canViewWorkspacePrice.value) return null
  const currentBOMID = Number(currentBOMSnapshot.value?.id || 0)
  if (!currentBOMID) return null
  return (quoteWorkspaceContext.value?.cost_sheets || []).find((item) => Number(item.bom_snapshot_id || 0) === currentBOMID) || null
})
const historicalCostSheets = computed(() => {
  if (!canViewWorkspacePrice.value) return []
  const currentID = Number(currentCostSheet.value?.id || 0)
  return (quoteWorkspaceContext.value?.cost_sheets || []).filter((item) => Number(item.id || 0) !== currentID)
})
const currentBOMSnapshotRows = computed(() => {
  const row = currentBOMSnapshot.value || {}
  return [
    { key: 'no', label: t('industry.manufacturing.workspace.bom'), value: bomSnapshotDisplayName(row) },
    { key: 'snapshot_no', label: t('industry.manufacturing.field.snapshotNo'), value: row.snapshot_no || '-' },
    { key: 'lines', label: t('industry.manufacturing.field.bomLineCount'), value: row.bom_line_count || 0 },
    { key: 'status', label: t('industry.manufacturing.field.status'), value: mfgStatusText(row.status) },
    { key: 'created', label: t('industry.manufacturing.field.createdAt'), value: formatDate(row.created_at || row.updated_at) },
    { key: 'source', label: t('industry.manufacturing.bom.source'), value: bomSnapshotSourceText(row) }
  ]
})
const bomDetailLines = computed(() => parseBOMLines(activeBOMDetail.value).filter((line) => line.item_code || line.item_name || line.spec || line.qty))
const selectedRFQSummary = computed(() => {
  const rfqID = Number(recordForm.rfq_id || 0)
  const rfq = relationOptions.rfqs.find((item) => Number(item.id) === rfqID)
  return buildRFQSummary(rfq)
})
const isBOMEditorSection = computed(() => ['bom', 'bomTemplates'].includes(activeSection.value?.key))
const actionResultTitle = computed(() => actionResultType.value === 'readiness'
  ? t('industry.manufacturing.readiness.title')
  : t('industry.manufacturing.action.actionResult'))
const readinessMissingItems = computed(() => {
  const missing = Array.isArray(actionResultData.value?.missing) ? actionResultData.value.missing : []
  return missing.map((key) => ({
    key,
    label: readinessMissingLabel(key),
    action: readinessMissingAction(key)
  }))
})
const readinessNextActionText = computed(() => readinessActionLabel(actionResultData.value?.next_action))
const recordDialogTitle = computed(() => editingRecordID.value
  ? t(activeFormSpec.value?.editTitleKey || 'industry.manufacturing.action.edit')
  : t(activeSection.value?.key === 'rfqs' ? 'industry.manufacturing.rfq.dialogTitle' : (activeFormSpec.value?.titleKey || 'industry.manufacturing.action.create')))
const dashboardUpdatedAtText = computed(() => {
  const values = executiveMetricGroups.value.map((group) => group.rawUpdatedAt).filter(Boolean)
  if (!values.length) return '-'
  const latest = values
    .map((value) => new Date(value))
    .filter((value) => !Number.isNaN(value.getTime()))
    .sort((a, b) => b.getTime() - a.getTime())[0]
  return latest ? formatTime(latest.toISOString()) : '-'
})
const executiveKpiCards = computed(() => [
  { key: 'rfq_count', metric: 'rfqs', tone: 'blue', label: t('industry.manufacturing.dashboard.metric.rfqCount'), value: numberText(dashboard.value.rfq_count), caption: t('industry.manufacturing.dashboard.metric.totalScale') },
  { key: 'open_rfq_count', metric: 'rfqs', tone: 'yellow', label: t('industry.manufacturing.dashboard.metric.openRfqCount'), value: numberText(dashboard.value.open_rfq_count), caption: t('industry.manufacturing.dashboard.metric.needProcessing') },
  { key: 'quote_amount', metric: 'quotes', tone: 'blue', label: t('industry.manufacturing.dashboard.metric.quoteAmount'), value: money(dashboard.value.quote_amount || 0), caption: t('industry.manufacturing.dashboard.metric.revenuePipeline') },
  { key: 'low_margin_quote_count', metric: 'profit', tone: 'red', label: t('industry.manufacturing.dashboard.metric.lowMarginQuoteCount'), value: numberText(dashboard.value.low_margin_quote_count), caption: t('industry.manufacturing.dashboard.metric.marginRisk') },
  { key: 'red_delivery_count', metric: 'delivery', tone: 'red', label: t('industry.manufacturing.dashboard.metric.redDeliveryCount'), value: numberText(dashboard.value.red_delivery_count), caption: t('industry.manufacturing.dashboard.metric.deliveryRisk') },
  { key: 'ecn_pending_count', metric: 'ecns', tone: 'yellow', label: t('industry.manufacturing.dashboard.metric.ecnPendingCount'), value: numberText(dashboard.value.ecn_pending_count), caption: t('industry.manufacturing.dashboard.metric.approvalPending') },
  { key: 'compensation_record_count', metric: 'compensation-records', tone: 'gray', label: t('industry.manufacturing.dashboard.metric.compensationRecordCount'), value: numberText(dashboard.value.compensation_record_count), caption: t('industry.manufacturing.dashboard.metric.afterSalesCost') },
  { key: 'asset_service_count', metric: 'lifecycle', tone: 'green', label: t('industry.manufacturing.dashboard.metric.assetServiceCount'), value: numberText(Number(dashboard.value.asset_count || 0) + Number(dashboard.value.service_ticket_count || 0)), caption: t('industry.manufacturing.dashboard.metric.lifecycleScale') }
])
const quoteMetrics = computed(() => groupMetrics('efficiency'))
const quoteBusinessMetrics = computed(() => [
  { key: 'quote_amount', label: t('industry.manufacturing.dashboard.metric.quoteAmount'), value: money(dashboard.value.quote_amount || 0) },
  { key: 'low_margin_quote_count', label: t('industry.manufacturing.dashboard.metric.lowMarginQuoteCount'), value: numberText(dashboard.value.low_margin_quote_count) },
  { key: 'avg_quote_cycle_hours', label: t('industry.manufacturing.dashboard.metric.avgQuoteCycleHours'), value: metricValueText('avg_quote_cycle_hours', quoteMetrics.value.avg_quote_cycle_hours) },
  { key: 'costing_cycle_hours', label: t('industry.manufacturing.dashboard.metric.costingCycleHours'), value: metricValueText('costing_cycle_hours', quoteMetrics.value.costing_cycle_hours) },
  { key: 'approval_cycle_hours', label: t('industry.manufacturing.dashboard.metric.approvalCycleHours'), value: metricValueText('approval_cycle_hours', quoteMetrics.value.approval_cycle_hours) }
])
const deliveryMetrics = computed(() => ({
  green_count: Number(dashboard.value.snapshots?.delivery?.green_count ?? groupMetrics('delivery').green_count ?? 0),
  yellow_count: Number(dashboard.value.snapshots?.delivery?.yellow_count ?? groupMetrics('delivery').yellow_count ?? 0),
  red_count: Number(dashboard.value.snapshots?.delivery?.red_count ?? groupMetrics('delivery').red_count ?? dashboard.value.red_delivery_count ?? 0),
  on_time_rate: Number(dashboard.value.snapshots?.delivery?.on_time_rate ?? groupMetrics('delivery').on_time_rate ?? 0)
}))
const deliveryRiskMetrics = computed(() => [
  { key: 'green', label: t('industry.manufacturing.dashboard.metric.greenCount'), value: numberText(deliveryMetrics.value.green_count) },
  { key: 'yellow', label: t('industry.manufacturing.dashboard.metric.yellowCount'), value: numberText(deliveryMetrics.value.yellow_count) },
  { key: 'red', label: t('industry.manufacturing.dashboard.metric.redCount'), value: numberText(deliveryMetrics.value.red_count) }
])
const executiveMetricGroups = computed(() => {
  const groups = Array.isArray(dashboard.value.metric_groups) ? dashboard.value.metric_groups : []
  return groups.map((group) => {
    const metrics = group.metrics || {}
    return {
      key: group.key,
      metric: group.drilldown,
      label: t(`industry.manufacturing.dashboard.group.${group.key}`),
      rawUpdatedAt: group.updated_at,
      updatedAt: formatTime(group.updated_at),
      refreshText: refreshStatusText(group.refresh_status, group.refresh_error),
      metrics: Object.keys(metrics).map((key) => ({
        key,
        label: metricLabelText(key),
        value: metricValueText(key, metrics[key])
      }))
    }
  })
})

onMounted(async () => {
  syncWorkspaceRouteState()
  await refreshAll()
})
watch(() => route.fullPath, async () => {
  syncWorkspaceRouteState()
  await Promise.all([
    isCustomerWorkflow.value ? loadCompanies() : Promise.resolve(),
    ...visibleSections.value.map((section) => loadSection(section))
  ])
})

async function refreshAll() {
  await Promise.all([loadDashboard(), isCustomerWorkflow.value ? loadCompanies() : Promise.resolve(), ...visibleSections.value.map((section) => loadSection(section))])
}

async function loadUsers() {
  if (usersLoading.value) return
  usersLoading.value = true
  try {
    const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } })
    const data = unwrapMfgResponseData(res)
    userOptions.value = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
  } catch {
    userOptions.value = []
  } finally {
    usersLoading.value = false
  }
}

async function loadDashboard() {
  dashboardLoading.value = true
  try {
    const res = await api.get('/api/v1/mfg/dashboard')
    dashboard.value = unwrapMfgResponseData(res) || {}
  } catch (error) {
    dashboard.value = {}
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    dashboardLoading.value = false
  }
}

async function refreshMetricSnapshots() {
  if (!canRefreshMetricSnapshots.value) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  metricRefreshing.value = true
  try {
    await api.post('/api/v1/mfg/dashboard/refresh-metrics')
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    await loadDashboard()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    metricRefreshing.value = false
  }
}

async function openMetricDrilldown(card) {
  drilldownVisible.value = true
  drilldownTitle.value = card.label
  drilldownTarget.value = card.metric || ''
  drilldownLoading.value = true
  try {
    const res = await api.get('/api/v1/mfg/dashboard/drilldown', {
      params: { metric: card.metric, page: 1, size: 30 }
    })
    const data = unwrapMfgResponseData(res)
    drilldownItems.value = data?.items || []
    drilldownTarget.value = data?.target || drilldownTarget.value
  } catch (error) {
    drilldownItems.value = []
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    drilldownLoading.value = false
  }
}

function openDashboardMetric(item) {
  if (!item?.metric) return
  openMetricDrilldown(item)
}

function drilldownPrimaryText(row = {}) {
  const target = drilldownTarget.value
  if (target === 'rfqs') return compactParts([rfqDisplayName(row), customerNameText(row), rfqProductText(row)])
  if (target === 'quotes') return compactParts([row.quote_no || row.id, rfqDisplayName(row), customerNameText(row)])
  if (target === 'production-progress') return compactParts([row.order_name || row.order_code || row.erp_order_no || row.mes_work_order_no || (row.order_id ? `#${row.order_id}` : ''), productionMilestoneText(row.milestone)])
  if (target === 'assets') return compactParts([row.serial_no || row.batch_no || row.id, customerNameText(row), row.install_location])
  if (target === 'service-tickets') return compactParts([row.fault_code || row.service_type || row.id, customerNameText(row), mfgStatusText(row.service_result)])
  if (target === 'ecns') return compactParts([row.ecn_no || row.id, row.order_name || row.order_code, ecnStatusText(row.status)])
  if (target === 'compensation-records') return compactParts([row.compensation_no || row.id, row.order_name || row.order_code, customerNameText(row)])
  if (target === 'customer-metrics') return compactParts([customerNameText(row), formatDate(row.snapshot_date || row.stat_date)])
  return compactParts([row.business_name, row.name, row.rfq_no, row.quote_no, row.serial_no, row.fault_code, row.milestone, row.id])
}

function drilldownStatusText(row = {}) {
  const target = drilldownTarget.value
  if (target === 'rfqs') return mfgStatusText(row.status)
  if (target === 'quotes') return quoteApprovalStatusText(row.approval_status || row.status)
  if (target === 'production-progress') return productionStatusText(row.progress_status || row.status) || mfgStatusText(row.risk_level)
  if (target === 'ecns') return compactParts([ecnStatusText(row.status), quoteApprovalStatusText(row.approval_status)])
  if (target === 'compensation-records') return compactParts([compensationStatusText(row.status), quoteApprovalStatusText(row.approval_status)])
  if (target === 'customer-metrics') return refreshStatusText(row.refresh_status, row.refresh_error) || '-'
  if (row.stage) return sampleStageText(row.stage)
  if (row.risk_level) return mfgStatusText(row.risk_level)
  if (row.refresh_status) return refreshStatusText(row.refresh_status, row.refresh_error)
  return mfgStatusText(row.status)
}

function drilldownTimeText(row = {}) {
  return formatTime(row.updated_at || row.created_at || row.stat_date || row.snapshot_date)
}

function compactParts(parts) {
  const values = parts.map((item) => String(item || '').trim()).filter((item) => item && item !== '-')
  return values.length ? values.join(' · ') : '-'
}

function groupMetrics(key) {
  const group = Array.isArray(dashboard.value.metric_groups)
    ? dashboard.value.metric_groups.find((item) => item.key === key)
    : null
  return group?.metrics || {}
}

function metricLabelText(key) {
  return t(`industry.manufacturing.dashboard.metric.${toCamelCase(key)}`)
}

function metricValueText(key, value) {
  if (value === null || value === undefined || value === '') return '-'
  if (String(key).includes('rate')) return percentText(value)
  if (String(key).includes('hours')) return hourText(value)
  if (String(key).includes('amount')) return money(value)
  if (String(key).includes('target')) return targetText(value)
  if (Number.isFinite(Number(value))) return numberText(value)
  return String(value || '-')
}

function refreshStatusText(status, error) {
  if (error) return `${t('industry.manufacturing.dashboard.refreshError')}: ${error}`
  if (!status) return ''
  const key = `industry.manufacturing.dashboard.refreshStatus.${status}`
  const translated = t(key)
  return translated === key ? String(status) : translated
}

function targetText(value) {
  if (!value) return '-'
  const key = String(value)
  const translated = t(`industry.manufacturing.dashboard.target.${key}`)
  return translated === `industry.manufacturing.dashboard.target.${key}` ? key : translated
}

function toCamelCase(value) {
  return String(value || '').replace(/_([a-z])/g, (_, letter) => letter.toUpperCase())
}

async function loadCompanies() {
  companiesLoading.value = true
  try {
    const res = await api.get('/api/v1/prd/contacts/objects', {
      params: {
        object_type: 'company',
        keyword: companyKeyword.value,
        page: 1,
        size: 100
      }
    })
    const data = unwrapMfgResponseData(res)
    companies.value = data?.items || []
    if (!selectedCustomerID.value && companies.value.length) {
      selectCompany(companies.value[0])
    }
  } catch (error) {
    companies.value = []
    if (error?.response?.status !== 403) {
      ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
    }
  } finally {
    companiesLoading.value = false
  }
}

function selectCompany(company) {
  selectedCompany.value = company
  selectedCustomerID.value = company?.id || 0
}

function goWorkflow(workflow, tab = '') {
  const target = workflow || 'home'
  const requestedTab = tab || workflowDefinitions[target]?.sections?.[0] || 'rfqs'
  const normalizedTab = resolveWorkspaceTab(target, requestedTab)
  activeWorkspaceTab.value = normalizedTab
  const query = normalizedTab ? { tab: normalizedTab } : {}
  if (target === 'quote-to-order') {
    const block = accessibleRFQWorkspaceBlock(tabToWorkspaceBlock(requestedTab))
    activeWorkspaceBlock.value = block
    query.block = block
  }
  router.push({ path: `/manufacturing/${target}`, query })
}

function normalizeWorkspaceTab(tab) {
  if (['rfqs', 'clarifications', 'drawings', 'bom', 'cost', 'quotes', 'quoteWorkspace'].includes(tab)) return 'quoteWorkspace'
  return tab || 'quoteWorkspace'
}

function syncWorkspaceRouteState() {
  activeWorkspaceTab.value = resolveWorkspaceTab(currentModule.value, route.query.tab)
  activeWorkspaceBlock.value = accessibleRFQWorkspaceBlock(resolveWorkspaceBlock(currentModule.value, route.query.tab, route.query.block))
}

function resolveWorkspaceTab(module, queryTab = '') {
  const legacy = moduleSectionMap[module]
  const workflowKey = legacy?.workflow || module || 'home'
  const workflow = workflowDefinitions[workflowKey] || workflowDefinitions.home
  const allowedTabs = [...(workflow.sections || [])]
  if (workflowKey === 'home') return ''
  const requested = normalizeWorkspaceTab(String(queryTab || legacy?.tab || ''))
  if (requested && allowedTabs.includes(requested)) return requested
  return allowedTabs[0] || ''
}

function resolveWorkspaceBlock(module, queryTab = '', queryBlock = '') {
  const legacy = moduleSectionMap[module]
  if ((legacy?.workflow || module) !== 'quote-to-order') return 'rfq'
  return String(queryBlock || legacy?.block || tabToWorkspaceBlock(String(queryTab || legacy?.tab || 'rfqs')) || 'rfq')
}

function accessibleRFQWorkspaceBlock(block) {
  const value = String(block || 'rfq')
  if (priceWorkspaceBlocks.includes(value) && !canViewWorkspacePrice.value) return 'rfq'
  return value
}

function createButtonText(section) {
  if (['rfqs', 'quoteWorkspace'].includes(section.key)) return t('industry.manufacturing.rfq.createCustomerRfq')
  return t(manufacturingFormRegistry[section.key]?.titleKey || 'industry.manufacturing.action.create')
}

function sectionManagePermission(section) {
  const key = section?.key || ''
  if (['quoteWorkspace', 'rfqs', 'clarifications', 'drawings'].includes(key)) return 'mfg:rfq:manage'
  if (['agreements', 'forecasts'].includes(key)) return 'mfg:agreement:manage'
  if (['product', 'configModels', 'bomTemplates', 'bom'].includes(key)) return 'mfg:product:manage'
  if (key === 'cost') return 'mfg:cost:manage'
  if (key === 'quotes') return 'mfg:quote:manage'
  if (key === 'samples') return 'mfg:sample:manage'
  if (key === 'production') return 'mfg:production:manage'
  if (key === 'ecns') return 'mfg:ecn:manage'
  if (key === 'compensationRecords') return 'mfg:compensation:manage'
  if (key === 'assets') return 'mfg:asset:manage'
  if (key === 'service') return 'mfg:service:manage'
  if (key === 'knowledge') return 'mfg:knowledge:manage'
  return ''
}

function canManageSection(section) {
  const permission = sectionManagePermission(section)
  return Boolean(permission && hasPermission(permission))
}

function canCreateSection(section) {
  if (['quoteWorkspace', 'rfqs'].includes(section?.key || '')) return Boolean(section?.creatable && canCreateRFQ())
  if (section?.key === 'bom' && selectedRFQID.value) return Boolean(section?.creatable && canCreateWorkspaceBOM.value)
  if (section?.key === 'cost' && selectedRFQID.value) return Boolean(section?.creatable && canCreateWorkspaceCost.value)
  if (section?.key === 'quotes' && selectedRFQID.value) return Boolean(section?.creatable && canCreateWorkspaceQuote.value)
  return Boolean(section?.creatable && canManageSection(section))
}

function canEditSection(section) {
  return canManageSection(section)
}

function canAnswerClarification() {
  return Boolean(selectedRFQID.value || quoteWorkspaceContext.value?.rfq?.id)
}

function canDeleteSection(section) {
  return canManageSection(section)
}

function canCreateRFQ() {
  return isAdminUser() || isCurrentUserSales() || hasExactPermission('mfg:rfq:manage') || hasExactPermission('mfg:quote:approve')
}

function isCurrentUserSales() {
  const user = currentUser()
  const values = [
    user.role,
    user.role_name,
    ...(Array.isArray(user.role_names) ? user.role_names : []),
    ...(Array.isArray(user.role_codes) ? user.role_codes : []),
    user.role_code
  ].map((item) => String(item || '').trim()).filter(Boolean)
  return values.some((value) => {
    const normalized = value.toLowerCase().replace(/-/g, '_')
    return value === '销售员' ||
      value === '销售代表' ||
      value.includes('销售员') ||
      value.includes('销售代表') ||
      normalized.includes('sales') ||
      ['sales', 'salesperson', 'sales_rep', 'r_sales', 'r_salesperson', 'r_sales_rep'].includes(normalized)
  })
}

function canEditRow(section, row) {
  if (['quoteWorkspace', 'rfqs'].includes(section?.key || '')) return Boolean(row?.can_edit)
  if (section?.key === 'ecns') return canManageSection(section) && canEditECN(row)
  if (section?.key === 'compensationRecords') return canManageSection(section) && canEditCompensation(row)
  return canEditSection(section)
}

function canDeleteRow(section, row) {
  if (['quoteWorkspace', 'rfqs'].includes(section?.key || '')) return Boolean(row?.can_delete)
  if (section?.key === 'ecns') return canManageSection(section) && canDeleteECN(row)
  if (section?.key === 'compensationRecords') return canManageSection(section) && canDeleteCompensation(row)
  return canDeleteSection(section)
}

function isReadOnlyAction(actionItem) {
  return ['readiness', 'viewDrawing', 'downloadDrawing', 'serviceHistory', 'customerView', 'revisions', 'diff'].includes(actionItem?.key)
}

function canRunSectionAction(section, actionItem) {
  if (isReadOnlyAction(actionItem)) return true
  return canManageSection(section)
}

function canRunRowAction(section, row, actionItem) {
  if (isReadOnlyAction(actionItem)) return true
  if (['quoteWorkspace', 'rfqs'].includes(section?.key || '')) {
    if (actionItem?.key === 'submit') return Boolean(row?.can_submit)
    if (actionItem?.key === 'startCosting') return Boolean(row?.can_start_costing)
    return Boolean(row?.can_edit)
  }
  if (section?.key === 'ecns') {
    if (['approve', 'reject'].includes(actionItem?.key)) return hasPermission('mfg:ecn:approve')
    return hasPermission('mfg:ecn:manage')
  }
  if (section?.key === 'compensationRecords') {
    if (['approve', 'reject'].includes(actionItem?.key)) return hasPermission('mfg:compensation:approve')
    return hasPermission('mfg:compensation:manage')
  }
  return canRunSectionAction(section, actionItem)
}

function canSaveActiveRecord() {
  const section = activeSection.value
  if (!section) return false
  if (section.key === 'rfqs') {
    if (!editingRecordID.value) return canCreateRFQ()
    return Boolean(activeRecordCapabilities.value.can_edit || recordForm.can_edit || canManageSection(section))
  }
  if (section.key === 'bom' && selectedRFQID.value) return canCreateWorkspaceBOM.value
  if (section.key === 'cost' && selectedRFQID.value) return canCreateWorkspaceCost.value
  if (section.key === 'quotes' && selectedRFQID.value) return canCreateWorkspaceQuote.value
  return canManageSection(section)
}

async function openSectionCreateDialog(section) {
  if (section?.key === 'quoteWorkspace') {
    await openRecordDialog(rfqSection.value)
    return
  }
  await openRecordDialog(section)
}

async function goCreateBOMSnapshotFromCost(row) {
  if (!canCreateWorkspaceBOM.value && !canCreateSection(bomSection.value)) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  const rfqID = Number(row?.rfq_id || 0)
  activeWorkspaceTab.value = 'quoteWorkspace'
  activeWorkspaceBlock.value = 'bom'
  selectedRFQID.value = rfqID || selectedRFQID.value
  await router.push({
    path: '/manufacturing/quote-to-order',
    query: {
      tab: 'quoteWorkspace',
      block: 'bom',
      from: 'costing',
      ...(rfqID ? { rfq_id: String(rfqID) } : {})
    }
  })
  const bomSection = sections.find((item) => item.key === 'bom')
  if (bomSection) {
    await loadSection(bomSection)
    await openRecordDialog(bomSection)
  }
}

async function goCreateQuoteFromCost(row) {
  if (!canViewWorkspacePrice.value || !canCreateWorkspaceQuote.value) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  activeWorkspaceTab.value = 'quoteWorkspace'
  activeWorkspaceBlock.value = 'quotes'
  selectedRFQID.value = Number(row.rfq_id || selectedRFQID.value || 0)
  await router.push({
    path: '/manufacturing/quote-to-order',
    query: {
      tab: 'quoteWorkspace',
      block: 'quotes',
      from: 'costing',
      cost_sheet_id: String(row.id || ''),
      rfq_id: String(row.rfq_id || ''),
      bom_snapshot_id: String(effectiveBOMSnapshotID(row) || ''),
      amount: String(row.amount || row.target_price || 0),
      currency: row.currency || costPayload(row).currency || 'CNY'
    }
  })
  const quoteSection = sections.find((item) => item.key === 'quotes')
  if (quoteSection) {
    await loadSection(quoteSection)
    await openRecordDialog(quoteSection)
  }
}

async function loadSection(section) {
  section.loading = true
  try {
    const res = await api.get(`/api/v1/mfg/${section.endpoint}`, {
      params: { page: 1, size: 50, keyword: section.keyword }
    })
    const data = unwrapMfgResponseData(res)
    section.items = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
    section.total = Number(data?.total || section.items.length || 0)
    if (section.key === 'quoteWorkspace') {
      await ensureQuoteWorkspaceSelection(section)
      await maybeOpenRFQWorkspaceFromRoute(section)
    }
  } catch (error) {
    section.items = []
    section.total = 0
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    section.loading = false
  }
}

function tabToWorkspaceBlock(tab) {
  return {
    rfqs: 'rfq',
    quoteWorkspace: activeWorkspaceBlock.value || String(route.query.block || 'rfq'),
    clarifications: 'clarifications',
    drawings: 'rfq',
    bom: 'bom',
    cost: 'cost',
    quotes: 'quotes'
  }[tab] || 'rfq'
}

async function ensureQuoteWorkspaceSelection(section) {
  const queryRFQID = Number(route.query.rfq_id || 0)
  const nextID = queryRFQID || (rfqWorkspaceDialogVisible.value ? selectedRFQID.value : 0)
  if (nextID > 0) {
    selectedRFQID.value = nextID
    await loadQuoteWorkspaceContext(nextID)
  } else {
    quoteWorkspaceContext.value = {}
  }
}

async function selectQuoteWorkspaceRFQ(row) {
  await openRFQWorkspaceDialog(row)
}

async function loadQuoteWorkspaceContext(rfqID = selectedRFQID.value) {
  if (!rfqID) return
  quoteWorkspaceLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/rfqs/${rfqID}/quote-workspace`)
    const data = unwrapMfgResponseData(res) || {}
    if (Array.isArray(data.clarifications)) {
      data.clarifications = data.clarifications.map((item) => ({ ...item, _original_answer: item.answer || '' }))
    }
    quoteWorkspaceContext.value = data
    syncActiveQuoteDetailFromWorkspace()
    activeWorkspaceBlock.value = accessibleRFQWorkspaceBlock(activeWorkspaceBlock.value)
    activeRFQWorkspaceTab.value = accessibleRFQWorkspaceBlock(activeRFQWorkspaceTab.value)
  } catch (error) {
    quoteWorkspaceContext.value = {}
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    quoteWorkspaceLoading.value = false
  }
}

async function refreshRFQWorkspaceContext() {
  if (!selectedRFQID.value) return
  await loadQuoteWorkspaceContext(selectedRFQID.value)
}

async function maybeOpenRFQWorkspaceFromRoute(section) {
  const rfqID = Number(route.query.rfq_id || 0)
  if (!rfqID || rfqWorkspaceDialogVisible.value) return
  const row = section.items?.find((item) => Number(item.id) === rfqID) || { id: rfqID }
  await openRFQWorkspaceDialog(row, accessibleRFQWorkspaceBlock(String(route.query.block || tabToWorkspaceBlock(String(route.query.tab || 'rfqs')))))
}

async function openRFQWorkspaceDialog(row, block = 'rfq') {
  selectedRFQID.value = Number(row?.id || 0)
  const safeBlock = accessibleRFQWorkspaceBlock(block)
  activeWorkspaceBlock.value = safeBlock
  activeRFQWorkspaceTab.value = workspaceBlockToDialogTab(safeBlock)
  rfqWorkspaceDialogVisible.value = true
  await refreshRFQWorkspaceContext()
}

function closeRFQWorkspaceDialog() {
  rfqWorkspaceDialogVisible.value = false
  selectedRFQID.value = 0
  quoteWorkspaceContext.value = {}
  activeRFQWorkspaceTab.value = 'rfq'
}

function workspaceBlockToDialogTab(block) {
  return {
    rfq: 'rfq',
    clarifications: 'clarifications',
    drawings: 'rfq',
    bom: 'bom',
    cost: 'cost',
    quotes: 'quotes'
  }[block] || 'rfq'
}

async function openWorkspaceCreate(sectionKey) {
  activeWorkspaceBlock.value = sectionKey
  const section = sections.find((item) => item.key === sectionKey)
  if (!section) return
  if (!canCreateSection(section)) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  await loadSection(section)
  const existingRow = sectionKey === 'cost' ? currentCostSheet.value : null
  await openRecordDialog(section, existingRow)
  if (selectedRFQID.value) {
    recordForm.rfq_id = selectedRFQID.value
    if (sectionKey === 'cost') {
      const snapshot = currentBOMSnapshot.value || (quoteWorkspaceContext.value.bom_snapshots || [])[0]
      recordForm.bom_snapshot_id = snapshot?.id || 0
      recordForm.quote_amount = quoteWorkspaceRFQ.value.target_price || 0
    }
    if (sectionKey === 'quotes') {
      const cost = currentCostSheet.value || (quoteWorkspaceContext.value.cost_sheets || [])[0]
      const snapshot = currentBOMSnapshot.value || (quoteWorkspaceContext.value.bom_snapshots || [])[0]
      recordForm.cost_sheet_id = cost?.id || 0
      recordForm.bom_snapshot_id = snapshot?.id || cost?.bom_snapshot_id || 0
      recordForm.amount = cost?.amount || quoteWorkspaceRFQ.value.target_price || 0
      recordForm.currency = quoteWorkspaceRFQ.value.rfq_payload?.currency || quoteWorkspaceRFQ.value.currency || 'CNY'
    }
  }
}

async function loadBusinessFormOptions(section) {
  const relations = new Set()
  const spec = manufacturingFormRegistry[section?.key]
  for (const groupItem of spec?.groups || []) {
    for (const fieldItem of groupItem.fields || []) {
      if (fieldItem.kind === 'relation') relations.add(fieldItem.relation)
      if (fieldItem.kind === 'user') relations.add('users')
    }
  }
  if (section?.key === 'bom') relations.add('rfqs')
  if (section?.key === 'bom') relations.add('bomTemplates')
  if (['bom', 'bomTemplates'].includes(section?.key)) relations.add('productFamilies')
  await Promise.all([...relations].map((name) => loadRelationOptions(name)))
}

async function loadRelationOptions(name) {
  if (!name || relationLoading[name]) return
  relationLoading[name] = true
  try {
    let res = null
    if (name === 'companies') {
      res = await api.get('/api/v1/prd/contacts/objects', { params: { object_type: 'company', page: 1, size: 100 } })
    } else if (name === 'orders') {
      res = await api.get('/api/v1/prd/orders/objects', { params: { object_type: 'order', page: 1, size: 100 } })
    } else if (name === 'users') {
      await loadUsers()
      relationOptions.users = userOptions.value
      return
    } else {
      const endpoint = relationEndpoint(name)
      if (!endpoint) return
      res = await api.get(`/api/v1/mfg/${endpoint}`, { params: { page: 1, size: 100 } })
    }
    const data = unwrapMfgResponseData(res)
    relationOptions[name] = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
  } catch (error) {
    relationOptions[name] = []
    if (name !== 'users') ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    relationLoading[name] = false
  }
}

function relationEndpoint(name) {
  return {
    productFamilies: 'product-families',
    rfqs: 'rfqs',
    bomTemplates: 'bom-templates',
    bom: 'bom-snapshots',
    cost: 'cost-sheets',
    quotes: 'quotes',
    assets: 'assets',
    service: 'service-tickets'
  }[name]
}

async function openRecordDialog(section, row = null) {
  if (!section) return
  activeRecordCapabilities.value = {
    can_edit: Boolean(row?.can_edit),
    can_delete: Boolean(row?.can_delete),
    can_submit: Boolean(row?.can_submit),
    can_create_bom: Boolean(row?.can_create_bom),
    can_create_cost: Boolean(row?.can_create_cost),
    can_create_quote: Boolean(row?.can_create_quote)
  }
  if (section.key === 'rfqs') {
    const allowed = row ? canEditRow(section, row) : canCreateRFQ()
    if (!allowed) {
      ElMessage.error(t('common.message.permissionDenied'))
      return
    }
    activeSectionKey.value = section.key
    editingRecordID.value = row?.id || 0
    for (const key of Object.keys(recordForm)) delete recordForm[key]
    Object.assign(recordForm, defaultRFQForm(row))
    recordDialogVisible.value = true
    await loadRFQDialogOptions(row)
    return
  }
  if (selectedRFQID.value && ['bom', 'cost', 'quotes'].includes(section.key)) {
    const allowed = section.key === 'bom'
      ? canCreateWorkspaceBOM.value
      : section.key === 'cost'
        ? canCreateWorkspaceCost.value
        : canCreateWorkspaceQuote.value
    if (!allowed) {
      ElMessage.error(t('common.message.permissionDenied'))
      return
    }
    activeSectionKey.value = section.key
    editingRecordID.value = row?.id || 0
    for (const key of Object.keys(recordForm)) delete recordForm[key]
    await loadBusinessFormOptions(section)
    for (const item of section.fields) {
      recordForm[item.key] = row ? normalizeFormValue(row[item.key]) : defaultBusinessFieldValue(item, section)
    }
    Object.assign(recordForm, defaultBusinessForm(section, row))
    recordDialogVisible.value = true
    return
  }
  if (!canManageSection(section)) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  activeSectionKey.value = section.key
  editingRecordID.value = row?.id || 0
  sampleProgressHistory.value = []
  productionProgressHistory.value = []
  for (const key of Object.keys(recordForm)) delete recordForm[key]
  await loadBusinessFormOptions(section)
  if (section.key === 'clarifications') await loadUsers()
  for (const item of section.fields) {
    recordForm[item.key] = row ? normalizeFormValue(row[item.key]) : defaultBusinessFieldValue(item, section)
  }
  if (!row && section.key === 'bom') {
    const rfqID = Number(route.query.rfq_id || 0)
    if (rfqID > 0) recordForm.rfq_id = rfqID
  }
  Object.assign(recordForm, defaultBusinessForm(section, row))
  recordDialogVisible.value = true
  if (section.key === 'samples' && row?.id) await loadSampleProgressHistory(row.id)
  if (section.key === 'production' && row?.id) await loadProductionProgressHistory(row.id)
}

function buildRecordPayload(section) {
  if (section.key === 'rfqs') return buildRFQPayload()
  if (section.key === 'bom') return buildBOMSnapshotPayload()
  if (section.key === 'bomTemplates') return buildBOMTemplatePayload()
  const payload = { ...recordForm }
  for (const fieldItem of section.fields) {
    if (fieldItem.type === 'number' || fieldItem.kind === 'number' || fieldItem.kind === 'user' || ['customer_id', 'sales_owner_id', 'order_id'].includes(fieldItem.key)) payload[fieldItem.key] = normalizeID(payload[fieldItem.key])
  }
  if (section.key === 'clarifications') {
    payload.source_type = payload.source_key ? 'manual' : ''
    payload.source_label = payload.source_key ? clarificationCategoryLabel(payload.source_key) : ''
  }
  if (section.key === 'cost') {
    payload.amount = Number(payload.quote_amount || payload.amount || 0)
    payload.currency = recordForm.currency || quoteWorkspaceRFQ.value?.rfq_payload?.currency || 'CNY'
  }
  if (section.key === 'product') {
    payload.status = normalizeProductFamilyStatus(payload.status)
  }
  if (section.key === 'production') {
    payload.planned_at = normalizeDateTimeForSubmit(payload.planned_at)
    payload.actual_at = normalizeDateTimeForSubmit(payload.actual_at)
  }
  if (section.key === 'compensationRecords') {
    payload.compensation_date = normalizeDateForSubmit(payload.compensation_date)
  }
  return payload
}

function defaultBusinessForm(section, row = null) {
  if (section.key === 'samples') {
    return {
      stage: row?.stage || 'sample_need',
      risk_level: row?.risk_level || 'green'
    }
  }
  if (section.key === 'production') {
    return {
      milestone: row?.milestone || 'order_confirmed',
      progress_status: row?.progress_status || 'not_started',
      planned_at: formatDateTimeForInput(row?.planned_at),
      actual_at: formatDateTimeForInput(row?.actual_at),
      risk_level: row?.risk_level || 'green'
    }
  }
  if (section.key === 'bom') {
    return {
      business_name: row?.business_name || '',
      bom_source: row?.source_template_id ? 'template' : 'manual',
      bom_template_id: normalizeID(row?.source_template_id),
      source_config_version: row?.source_config_version || row?.version || 'v1',
      bom_lines: parseBOMLines(row),
      save_as_template: false,
      template_version: row?.source_config_version || row?.version || 'v1',
      template_status: 'draft',
      product_family_id: normalizeID(row?.product_family_id || selectedRFQSummary.value.product_family_id)
    }
  }
  if (section.key === 'bomTemplates') {
    return {
      product_family_id: normalizeID(row?.product_family_id),
      template_no: row?.template_no || '',
      version: row?.version || 'v1',
      status: row?.status || 'draft',
      rule_text: ruleTextFromValue(row?.rule || row?.rule_json),
      bom_lines: parseBOMLines(row)
    }
  }
  if (section.key === 'cost') {
    return {
      quote_amount: Number(row?.quote_amount || row?.amount || row?.target_price || 0),
      currency: row?.currency || quoteWorkspaceRFQ.value?.rfq_payload?.currency || quoteWorkspaceRFQ.value?.currency || 'CNY'
    }
  }
  if (section.key === 'quotes' && !row) {
    return {
      rfq_id: normalizeID(route.query.rfq_id),
      cost_sheet_id: normalizeID(route.query.cost_sheet_id),
      bom_snapshot_id: normalizeID(route.query.bom_snapshot_id),
      amount: Number(route.query.amount || 0),
      currency: String(route.query.currency || 'CNY'),
      terms: ''
    }
  }
  if (section.key === 'clarifications' && !row) {
    return {
      source_key: rfqClarificationCategories[0]?.value || '',
      status: 'open'
    }
  }
  if (section.key === 'production' && !row) {
    return {
      risk_level: 'green'
    }
  }
  if (section.key === 'ecns' && !row) {
    return {
      impact_level: 'normal',
      cost_impact_currency: 'CNY',
      status: 'draft'
    }
  }
  if (section.key === 'compensationRecords' && !row) {
    return {
      currency: 'CNY',
      status: 'open'
    }
  }
  return {}
}

function defaultBusinessFieldValue(item, section = null) {
  if (item.key === 'currency') return 'CNY'
  if (item.key === 'cost_impact_currency') return 'CNY'
  if (item.key === 'impact_level') return 'normal'
  if (section?.key === 'product' && item.key === 'status') return 'active'
  if (item.key === 'status') return 'open'
  if (item.type === 'number') return 0
  if (item.options?.length) return item.options[0]
  return ''
}

function buildBOMSnapshotPayload() {
  const source = recordForm.bom_source === 'template' ? 'template' : 'manual'
  const version = recordForm.source_config_version || selectedBOMTemplate()?.version || 'v1'
  return {
    business_name: String(recordForm.business_name || '').trim(),
    rfq_id: normalizeID(recordForm.rfq_id),
    product_family_id: normalizeID(recordForm.product_family_id),
    source_config_version: version,
    bom_template_id: source === 'template' ? normalizeID(recordForm.bom_template_id) : 0,
    save_as_template: Boolean(recordForm.save_as_template),
    template_no: recordForm.template_no || '',
    template_version: recordForm.template_version || version,
    template_status: recordForm.template_status || 'draft',
    template_rule: templateRuleFromText(recordForm.template_rule_text),
    config_values: {
      source,
      rfq_id: normalizeID(recordForm.rfq_id),
      bom_template_id: source === 'template' ? normalizeID(recordForm.bom_template_id) : 0,
      source_config_version: version
    },
    bom_lines: normalizedBOMLines(recordForm.bom_lines)
  }
}

function buildBOMTemplatePayload() {
  return {
    product_family_id: normalizeID(recordForm.product_family_id),
    template_no: recordForm.template_no || '',
    version: recordForm.version || 'v1',
    status: recordForm.status || 'draft',
    rule: templateRuleFromText(recordForm.rule_text),
    bom_lines: normalizedBOMLines(recordForm.bom_lines)
  }
}

function normalizedBOMLines(lines) {
  return (lines || []).filter((line) => line.item_name || line.item_code).map((line, index) => ({
      line_no: index + 1,
      level: line.level || '1',
      item_code: line.item_code || '',
      item_name: line.item_name || '',
      spec: line.spec || '',
      material: line.material || '',
      process: line.process || '',
      qty: Number(line.qty || 0),
      unit: line.unit || '',
      loss_rate: Number(line.loss_rate || 0),
      sourcing: line.sourcing || 'buy',
      remark: line.remark || ''
    }))
}

function addBOMLine() {
  if (!Array.isArray(recordForm.bom_lines)) recordForm.bom_lines = []
  recordForm.bom_lines.push(defaultBOMLine())
}

function removeBOMLine(index) {
  recordForm.bom_lines.splice(index, 1)
  if (!recordForm.bom_lines.length) addBOMLine()
}

function defaultBOMLine() {
  return {
    level: '1',
    item_code: '',
    item_name: '',
    spec: '',
    material: '',
    process: '',
    qty: 1,
    unit: 'pcs',
    loss_rate: 0,
    sourcing: 'buy',
    remark: ''
  }
}

function parseBOMLines(row) {
  const lines = parseArrayField(row?.bom_lines_json || row?.bom_lines || recordForm.bom_lines)
  if (Array.isArray(lines) && lines.length) return lines.map((line) => ({ ...defaultBOMLine(), ...line }))
  return [defaultBOMLine()]
}

function handleBOMRFQChange() {
  if (!recordForm.source_config_version) recordForm.source_config_version = 'v1'
  const rfq = relationOptions.rfqs.find((item) => Number(item.id) === Number(recordForm.rfq_id || 0))
  const summary = buildRFQSummary(rfq)
  if (summary.product_family_id) recordForm.product_family_id = summary.product_family_id
}

function handleBOMSourceChange() {
  if (recordForm.bom_source === 'manual') {
    recordForm.bom_template_id = 0
    return
  }
  handleBOMTemplateChange(recordForm.bom_template_id)
}

function handleBOMTemplateChange(value) {
  const template = relationOptions.bomTemplates.find((item) => Number(item.id) === Number(value || 0))
  if (!template) return
  recordForm.bom_template_id = template.id
  recordForm.source_config_version = template.version || recordForm.source_config_version || 'v1'
  recordForm.template_version = recordForm.source_config_version
  recordForm.product_family_id = normalizeID(template.product_family_id) || normalizeID(recordForm.product_family_id)
  recordForm.template_rule_text = ruleTextFromValue(template.rule || template.rule_json)
  recordForm.bom_lines = parseBOMLines(template)
}

function handleRelationFieldChange(fieldItem, value) {
  if (fieldItem.key === 'rfq_id' && activeSection.value?.key === 'bom') {
    handleBOMRFQChange(value)
  }
  if (fieldItem.key === 'order_id' && ['production', 'compensationRecords'].includes(activeSection.value?.key)) {
    applyOrderCRMContext(value)
  }
}

function applyOrderCRMContext(orderID) {
  const order = relationOptions.orders.find((item) => Number(item.id) === Number(orderID || 0))
  if (!order) return
  const info = crmPayload(order)
  const customerID = normalizeID(order.company_id || info.company_id || info.customer_id)
  const ownerID = normalizeID(order.owner_id || info.owner_id || info.sales_owner_id)
  if (customerID) recordForm.customer_id = customerID
  if (ownerID) recordForm.sales_owner_id = ownerID
  if (!recordForm.erp_order_no) recordForm.erp_order_no = order.code || info.erp_order_no || ''
}

function groupRowsBy(items, labelFn, prefix) {
  const groups = new Map()
  for (const item of items) {
    const label = labelFn(item) || t('industry.manufacturing.production.groupUnknown')
    if (!groups.has(label)) groups.set(label, [])
    groups.get(label).push(item)
  }
  return [...groups.entries()].map(([label, rows], index) => ({
    key: `${prefix}-${index}-${label}`,
    label,
    items: rows
  }))
}

function selectedBOMTemplate() {
  return relationOptions.bomTemplates.find((item) => Number(item.id) === Number(recordForm.bom_template_id || 0))
}

function templateRuleFromText(value) {
  const text = String(value || '').trim()
  return text ? { remark: text } : {}
}

function ruleTextFromValue(value) {
  if (!value) return ''
  if (typeof value === 'string') {
    const parsed = parseObjectField(value)
    return parsed.remark || value
  }
  return value.remark || JSON.stringify(value)
}

function bomTemplateOptionLabel(item) {
  return `${item.template_no || item.id}${item.version ? ` · ${item.version}` : ''}${item.bom_line_count ? ` · ${item.bom_line_count} ${t('industry.manufacturing.field.bomLineCount')}` : ''}`
}

function productFamilyName(id) {
  const productFamilyID = normalizeID(id)
  const item = relationOptions.productFamilies.find((entry) => Number(entry.id) === productFamilyID)
  return item ? relationOptionLabel('productFamilies', item) : (productFamilyID ? `#${productFamilyID}` : '-')
}

async function saveRecord() {
  const section = activeSection.value
  if (!section) return
  if (!canSaveActiveRecord()) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  recordSaving.value = true
  try {
    const payload = buildRecordPayload(section)
    let res = null
    if (editingRecordID.value && section.key === 'cost') {
      res = await api.post(`/api/v1/mfg/cost-sheets/${editingRecordID.value}/complete`, payload)
    } else if (editingRecordID.value) {
      res = await api.put(`/api/v1/mfg/${section.endpoint}/${editingRecordID.value}`, payload)
    } else if (section.createAction) {
      res = await section.createAction(payload)
    } else {
      res = await api.post(`/api/v1/mfg/${section.endpoint}`, payload)
    }
    const savedItem = unwrapMfgResponseData(res)?.item
    mergeSavedSectionItem(section, savedItem)
    ElMessage.success(reusedRecordMessage(section, savedItem))
    recordDialogVisible.value = false
    const refreshJobs = [loadSection(section), loadDashboard()]
    if (section.key === 'bom' && route.query.from === 'costing') {
      const costSection = sections.find((item) => item.key === 'cost')
      if (costSection) refreshJobs.push(loadSection(costSection))
      activeWorkspaceTab.value = 'quoteWorkspace'
      activeWorkspaceBlock.value = 'cost'
      router.replace({ path: '/manufacturing/quote-to-order', query: { tab: 'quoteWorkspace', block: 'cost', ...(selectedRFQID.value ? { rfq_id: String(selectedRFQID.value) } : {}) } })
    }
    const quoteWorkspaceSection = sections.find((item) => item.key === 'quoteWorkspace')
    if (quoteWorkspaceSection && section.key !== 'quoteWorkspace') refreshJobs.push(loadSection(quoteWorkspaceSection))
    if (selectedRFQID.value && ['rfqs', 'clarifications', 'bom', 'cost', 'quotes'].includes(section.key)) {
      refreshJobs.push(loadQuoteWorkspaceContext())
    }
    await Promise.all(refreshJobs)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    recordSaving.value = false
  }
}

async function loadSampleProgressHistory(id = editingRecordID.value) {
  if (!id) {
    sampleProgressHistory.value = []
    return
  }
  sampleProgressHistoryLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/sample-projects/${id}/progress-history`)
    sampleProgressHistory.value = res.data?.items || res.data?.data?.items || []
  } catch (error) {
    sampleProgressHistory.value = []
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    sampleProgressHistoryLoading.value = false
  }
}

async function loadProductionProgressHistory(id = editingRecordID.value) {
  if (!id) {
    productionProgressHistory.value = []
    return
  }
  productionProgressHistoryLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/production-progress/${id}/progress-history`)
    productionProgressHistory.value = res.data?.items || res.data?.data?.items || []
  } catch (error) {
    productionProgressHistory.value = []
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    productionProgressHistoryLoading.value = false
  }
}

function reusedRecordMessage(section, item) {
  if (!item?.reused) return t('industry.manufacturing.customer360.saved')
  if (section?.key === 'production') return t('industry.manufacturing.production.reusedExisting')
  if (section?.key === 'samples') return t('industry.manufacturing.sample.reusedExisting')
  return t('industry.manufacturing.customer360.saved')
}

async function deleteRecord(section, row) {
  if (!row?.id) return
  if (!canDeleteSection(section)) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  try {
    await api.delete(`/api/v1/mfg/${section.endpoint}/${row.id}`)
    ElMessage.success(t('industry.manufacturing.customer360.deleted'))
    const quoteWorkspaceSection = sections.find((item) => item.key === 'quoteWorkspace')
    await Promise.all([
      loadSection(section),
      quoteWorkspaceSection && section.key !== 'quoteWorkspace' ? loadSection(quoteWorkspaceSection) : Promise.resolve(),
      loadDashboard(),
      selectedRFQID.value ? loadQuoteWorkspaceContext() : Promise.resolve()
    ])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  }
}

async function runAction(section, row, actionItem) {
  if (!row?.id) return
  if (!canRunRowAction(section, row, actionItem)) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  if (['ecns', 'compensationRecords'].includes(section?.key) && ['approve', 'reject'].includes(actionItem?.key)) {
    openQuoteApprovalRemark(section, row, actionItem.key)
    return
  }
  try {
    const res = await runMfgRowAction(section, row, actionItem)
    const data = unwrapMfgResponseData(res)
    if (actionItem.key === 'readiness') {
      actionResultType.value = 'readiness'
      actionResultData.value = data || {}
      actionResultVisible.value = true
      return
    } else if (data?.item) {
      mergeSavedSectionItem(section, data.item)
    }
    if (actionItem.key === 'startCosting' && data?.cost_sheet) {
      const costSection = sections.find((item) => item.key === 'cost')
      if (costSection) mergeSavedSectionItem(costSection, data.cost_sheet)
    }
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    const quoteWorkspaceSection = sections.find((item) => item.key === 'quoteWorkspace')
    await Promise.all([
      loadSection(section),
      quoteWorkspaceSection && section.key !== 'quoteWorkspace' ? loadSection(quoteWorkspaceSection) : Promise.resolve(),
      loadDashboard(),
      selectedRFQID.value ? loadQuoteWorkspaceContext() : Promise.resolve()
    ])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  }
}

async function runMfgRowAction(section, row, actionItem, payload = {}) {
  if (section.key === 'ecns') {
    if (actionItem.key === 'impactAnalysis') return api.post(`/api/v1/mfg/ecns/${row.id}/impact-analysis`, payload)
    if (actionItem.key === 'submitApproval') return api.post(`/api/v1/mfg/ecns/${row.id}/submit-approval`, payload)
    if (actionItem.key === 'approve') return api.post(`/api/v1/mfg/ecns/${row.id}/approve`, payload)
    if (actionItem.key === 'reject') return api.post(`/api/v1/mfg/ecns/${row.id}/reject`, payload)
    if (actionItem.key === 'confirm') return api.post(`/api/v1/mfg/ecns/${row.id}/customer-confirm`, payload)
    if (actionItem.key === 'effective') return api.post(`/api/v1/mfg/ecns/${row.id}/effective`, payload)
  }
  if (section.key === 'compensationRecords') {
    if (actionItem.key === 'submitApproval') return api.post(`/api/v1/mfg/compensation-records/${row.id}/submit-approval`, payload)
    if (actionItem.key === 'approve') return api.post(`/api/v1/mfg/compensation-records/${row.id}/approve`, payload)
    if (actionItem.key === 'reject') return api.post(`/api/v1/mfg/compensation-records/${row.id}/reject`, payload)
    if (actionItem.key === 'closeCompensation') return api.post(`/api/v1/mfg/compensation-records/${row.id}/close`, payload)
  }
  return actionItem.run(row)
}

function businessCustomerText(row) {
  return row?.customer_name || row?.rfq_payload?.customer_name || '-'
}

function isMaskedPrice(value) {
  return String(value ?? '').trim() === '***'
}

function businessProductText(row) {
  return row?.product_name || row?.rfq_payload?.product_name || row?.rfq_payload?.custom_product_name || row?.rfq_payload?.sku || '-'
}

function businessAmount(row) {
  if (isMaskedPrice(row?.quote_amount) || isMaskedPrice(row?.amount) || isMaskedPrice(row?.target_price) || isMaskedPrice(row?.cost_impact)) return '***'
  return Number(row?.quote_amount || row?.amount || row?.target_price || row?.cost_impact || 0)
}

function businessAmountText(row) {
  const amount = businessAmount(row)
  if (isMaskedPrice(amount)) return '***'
  const currency = row?.currency || row?.rfq_payload?.currency || quoteWorkspaceRFQ.value?.rfq_payload?.currency || 'CNY'
  return amount > 0 ? moneyWithCurrency(amount, currency) : '-'
}

function bomSnapshotSourceText(row) {
  if (!row) return '-'
  if (row.source_template_id) return t('industry.manufacturing.bom.sourceTemplate')
  return t('industry.manufacturing.bom.sourceManual')
}

function openBOMDetail(row) {
  activeBOMDetail.value = row
  bomDetailVisible.value = true
}

function bomSnapshotByID(id) {
  const snapshotID = Number(id || 0)
  if (!snapshotID) return null
  const items = [currentBOMSnapshot.value, ...bomSnapshotHistory.value].filter(Boolean)
  return items.find((item) => Number(item.id || 0) === snapshotID) || null
}

function bomSnapshotLabel(id) {
  const snapshot = bomSnapshotByID(id)
  return snapshot ? bomSnapshotDisplayName(snapshot) : (id ? `#${id}` : '-')
}

function costSheetDetailText(row = {}) {
  return row.cost_sheet_label || row.cost_sheet_no || row.cost_sheet_status || row.margin_level || '-'
}

function quoteQuantity(row = {}) {
  return Number(row.quantity || quoteWorkspaceRFQ.value?.quantity || 0)
}

function unitPriceText(row) {
  const amount = businessAmount(row)
  if (isMaskedPrice(amount)) return '***'
  const quantity = quoteQuantity(row)
  if (!amount || !quantity) return '-'
  const currency = row?.currency || row?.rfq_payload?.currency || quoteWorkspaceRFQ.value?.rfq_payload?.currency || 'CNY'
  return `${moneyWithCurrency(amount / quantity, currency)} / ${t('industry.manufacturing.field.pieceUnit')}`
}

const quoteDetailMetrics = computed(() => {
  const row = activeQuoteDetail.value || {}
  return [
    { key: 'amount', label: t('industry.manufacturing.field.quoteTotal'), value: businessAmountText(row) },
    { key: 'unitPrice', label: t('industry.manufacturing.field.unitPrice'), value: unitPriceText(row) },
    { key: 'quantity', label: t('industry.manufacturing.field.quantity'), value: quoteQuantity(row) || '-' },
    { key: 'currency', label: t('industry.manufacturing.field.currency'), value: row.currency || row.rfq_payload?.currency || 'CNY' },
    { key: 'sendStatus', label: t('industry.manufacturing.quote.sendStatus'), value: quoteSendStatusText(row.send_status) },
    { key: 'customerStatus', label: t('industry.manufacturing.quote.customerStatus'), value: quoteCustomerStatusText(row.status) }
  ]
})

const quoteDetailContextRows = computed(() => {
  const row = activeQuoteDetail.value || {}
  return [
    { key: 'rfq', label: t('industry.manufacturing.field.rfq'), value: row.rfq_business_name || row.rfq_no || (row.rfq_id ? `${t('industry.manufacturing.field.rfq')} #${row.rfq_id}` : '-') },
    { key: 'customer', label: t('industry.manufacturing.field.customer'), value: businessCustomerText(row) },
    { key: 'product', label: t('industry.manufacturing.field.productDemand'), value: businessProductText(row) },
    { key: 'bom', label: t('industry.manufacturing.workspace.bom'), value: row.bom_business_name || row.bom_snapshot_no || bomSnapshotLabel(row.bom_snapshot_id) },
    { key: 'cost', label: t('industry.manufacturing.workspace.cost'), value: costSheetDetailText(row) },
    { key: 'created', label: t('industry.manufacturing.field.createdAt'), value: formatDate(row.created_at) },
    { key: 'updated', label: t('industry.manufacturing.field.updatedAt'), value: formatDate(row.updated_at || row.created_at) }
  ]
})

const quoteApprovalHistoryItems = computed(() => {
  const row = activeQuoteDetail.value || {}
  return Array.isArray(row.approval_histories) ? row.approval_histories : []
})

const deliveryChangeDetailTitle = computed(() => {
  return activeDeliveryChangeDetailSection.value === 'ecns'
    ? t('industry.manufacturing.detail.ecnTitle')
    : t('industry.manufacturing.detail.compensationTitle')
})

const deliveryChangeDetailTypeText = computed(() => {
  return activeDeliveryChangeDetailSection.value === 'ecns'
    ? t('industry.manufacturing.workspace.ecn')
    : t('industry.manufacturing.workspace.compensationHistory')
})

const deliveryChangeDetailPrimary = computed(() => {
  const row = activeDeliveryChangeDetail.value || {}
  if (activeDeliveryChangeDetailSection.value === 'ecns') return row.ecn_no || row.id || '-'
  return row.compensation_no || row.order_name || row.id || '-'
})

const deliveryChangeDetailSubtitle = computed(() => {
  const row = activeDeliveryChangeDetail.value || {}
  if (activeDeliveryChangeDetailSection.value === 'ecns') return row.change_reason || '-'
  return row.reason || '-'
})

const deliveryChangeDetailMetrics = computed(() => {
  const row = activeDeliveryChangeDetail.value || {}
  if (activeDeliveryChangeDetailSection.value === 'ecns') {
    return [
      { key: 'impact', label: t('industry.manufacturing.field.impactLevel'), value: impactLevelText(row.impact_level) },
      { key: 'cost', label: t('industry.manufacturing.field.costImpact'), value: moneyWithCurrency(row.cost_impact, row.cost_impact_currency) },
      { key: 'delivery', label: t('industry.manufacturing.field.deliveryImpactDays'), value: row.delivery_impact_days || 0 },
      { key: 'customerConfirm', label: t('industry.manufacturing.ecn.customerConfirmStatus'), value: ecnCustomerConfirmText(row.customer_confirmed) }
    ]
  }
  return [
    { key: 'amount', label: t('industry.manufacturing.field.compensationAmount'), value: moneyWithCurrency(row.amount, row.currency) },
    { key: 'customer', label: t('industry.manufacturing.field.customer'), value: customerNameText(row) },
    { key: 'salesOwner', label: t('industry.manufacturing.field.salesOwner'), value: salesOwnerNameText(row) },
    { key: 'date', label: t('industry.manufacturing.field.compensationDate'), value: formatDate(row.compensation_date) }
  ]
})

const deliveryChangeDetailRows = computed(() => {
  const row = activeDeliveryChangeDetail.value || {}
  if (activeDeliveryChangeDetailSection.value === 'ecns') {
    return [
      { key: 'order', label: t('industry.manufacturing.field.order'), value: row.order_name || (row.order_id ? `#${row.order_id}` : '-') },
      { key: 'oldBom', label: t('industry.manufacturing.field.oldBomSnapshotId'), value: row.old_bom_business_name || row.old_bom_snapshot_no || (row.old_bom_snapshot_id ? `#${row.old_bom_snapshot_id}` : '-') },
      { key: 'newBom', label: t('industry.manufacturing.field.newBomSnapshotId'), value: row.new_bom_business_name || row.new_bom_snapshot_no || (row.new_bom_snapshot_id ? `#${row.new_bom_snapshot_id}` : '-') },
      { key: 'scope', label: t('industry.manufacturing.field.impactScope'), value: row.impact_scope || '-' },
      { key: 'quality', label: t('industry.manufacturing.field.qualityImpact'), value: row.quality_impact || '-' },
      { key: 'status', label: t('industry.manufacturing.field.status'), value: ecnStatusText(row.status) },
      { key: 'approval', label: t('industry.manufacturing.field.approvalStatus'), value: quoteApprovalStatusText(row.approval_status) },
      { key: 'updated', label: t('industry.manufacturing.field.updatedAt'), value: formatDate(row.updated_at || row.created_at) }
    ]
  }
  return [
    { key: 'order', label: t('industry.manufacturing.field.order'), value: row.order_name || (row.order_id ? `#${row.order_id}` : '-') },
    { key: 'customer', label: t('industry.manufacturing.field.customer'), value: customerNameText(row) },
    { key: 'reason', label: t('industry.manufacturing.field.compensationReason'), value: row.reason || '-' },
    { key: 'responsibility', label: t('industry.manufacturing.field.responsibilitySplit'), value: row.responsibility_split || '-' },
    { key: 'improvement', label: t('industry.manufacturing.field.improvementActions'), value: row.improvement_actions || '-' },
    { key: 'status', label: t('industry.manufacturing.field.status'), value: compensationStatusText(row.status) },
    { key: 'approval', label: t('industry.manufacturing.field.approvalStatus'), value: quoteApprovalStatusText(row.approval_status) },
    { key: 'updated', label: t('industry.manufacturing.field.updatedAt'), value: formatDate(row.updated_at || row.created_at) }
  ]
})

const deliveryChangeApprovalHistoryItems = computed(() => {
  const row = activeDeliveryChangeDetail.value || {}
  return Array.isArray(row.approval_histories) ? row.approval_histories : []
})

function quoteTermsText(row) {
  const raw = row?.terms ?? row?.terms_json
  if (!raw) return '-'
  if (typeof raw === 'string') {
    const text = raw.trim()
    if (!text) return '-'
    try {
      const parsed = JSON.parse(text)
      return quoteTermsText({ terms: parsed })
    } catch (_) {
      return text
    }
  }
  if (Array.isArray(raw)) {
    const text = raw.map((item) => quoteTermsText({ terms: item })).filter((item) => item && item !== '-').join('\n')
    return text || '-'
  }
  if (typeof raw === 'object') {
    const text = Object.entries(raw)
      .map(([key, value]) => `${key}: ${typeof value === 'object' ? quoteTermsText({ terms: value }) : String(value ?? '')}`)
      .filter((line) => !line.endsWith(': '))
      .join('\n')
    return text || '-'
  }
  return String(raw)
}

function fieldSuffixText(field) {
  if (field.suffix === 'currency') return recordForm.currency || quoteWorkspaceRFQ.value?.rfq_payload?.currency || 'CNY'
  return field.suffix || ''
}

function readonlyFieldText(field) {
  const value = recordForm[field.key]
  if (field.relation === 'bom') return bomSnapshotLabel(value)
  if (field.relation) {
    const item = (relationOptions[field.relation] || []).find((entry) => Number(entry.id) === Number(value || 0))
    return item ? relationOptionLabel(field.relation, item) : (value ? `#${value}` : '-')
  }
  return value || '-'
}

function canEditQuote(row) {
  const status = String(row?.status || '').toLowerCase()
  const approval = String(row?.approval_status || '').toLowerCase()
  const send = String(row?.send_status || '').toLowerCase()
  return !['approved', 'sent', 'confirmed', 'closed'].includes(status) && !['approved', 'pending'].includes(approval) && send !== 'sent'
}

function visibleQuoteActions(row) {
  const status = String(row?.status || '').toLowerCase()
  const approval = String(row?.approval_status || 'draft').toLowerCase()
  const send = String(row?.send_status || 'not_sent').toLowerCase()
  const canManageQuote = canCreateWorkspaceQuote.value
  const actions = []
  if (canManageQuote && ['draft', 'rejected', ''].includes(approval) && !['approved', 'pending'].includes(approval)) {
    actions.push({ key: 'submitApproval', labelKey: 'industry.manufacturing.action.submitApproval', type: 'primary' })
  }
  if (approval === 'pending') {
    if (hasExactPermission('mfg:quote:approve')) {
      actions.push({ key: 'approve', labelKey: 'industry.manufacturing.action.approve', type: 'success' })
      actions.push({ key: 'reject', labelKey: 'industry.manufacturing.action.reject', type: 'warning' })
    }
  }
  if (canManageQuote && approval === 'approved' && send !== 'sent') {
    actions.push({ key: 'send', labelKey: 'industry.manufacturing.action.send', type: 'success' })
  }
  if (send === 'sent') {
    if (canManageQuote && status !== 'confirmed') actions.push({ key: 'customerConfirm', labelKey: 'industry.manufacturing.action.customerConfirm', type: 'success' })
  }
  if (canManageQuote && status === 'confirmed') {
    actions.push({ key: 'createOrder', labelKey: 'industry.manufacturing.action.createOrder', type: 'success' })
  }
  actions.push({ key: 'revisions', labelKey: 'industry.manufacturing.action.revisions' })
  if (Number(row?.revision || 1) > 1) actions.push({ key: 'diff', labelKey: 'industry.manufacturing.action.versionDiff' })
  return actions
}

function quoteApprovalStatusText(status) {
  const value = String(status || 'draft').toLowerCase()
  return t(`industry.manufacturing.quoteApprovalStatus.${value}`)
}

function quoteSendStatusText(status) {
  const value = String(status || 'not_sent').toLowerCase() === 'sent' ? 'sent' : 'not_sent'
  return t(`industry.manufacturing.quoteSendStatus.${value}`)
}

function quoteSendStatusTagType(status) {
  return String(status || '').toLowerCase() === 'sent' ? 'success' : 'info'
}

function quoteCustomerStatusText(status) {
  const value = String(status || '').toLowerCase() === 'confirmed' ? 'confirmed' : 'pending'
  return t(`industry.manufacturing.quoteCustomerStatus.${value}`)
}

function quoteCustomerStatusTagType(status) {
  return String(status || '').toLowerCase() === 'confirmed' ? 'success' : 'warning'
}

function quoteApprovalActionText(action) {
  const value = String(action || '').toLowerCase()
  return t(`industry.manufacturing.quoteApprovalAction.${value || 'submit'}`)
}

const quoteApprovalRemarkRequired = computed(() => {
  const action = pendingQuoteApprovalAction.value || {}
  if (String(action.key || '') === 'reject') return true
  return !['ecns', 'compensationRecords'].includes(action.section?.key) && String(action.key || '') === 'approve'
})

const quoteApprovalRemarkTitle = computed(() => {
  const key = String(pendingQuoteApprovalAction.value?.key || '')
  if (key === 'approve') return t('industry.manufacturing.quote.approveRemarkTitle')
  if (key === 'reject') return t('industry.manufacturing.quote.rejectRemarkTitle')
  return t('industry.manufacturing.quote.approvalRemarkTitle')
})

function openQuoteApprovalRemark(section, row, key) {
  pendingQuoteApprovalAction.value = { section, row, key }
  quoteApprovalRemark.value = ''
  quoteApprovalRemarkVisible.value = true
}

async function confirmQuoteApprovalRemark() {
  const action = pendingQuoteApprovalAction.value
  if (!action?.row?.id || !action.key) return
  if (quoteApprovalRemarkRequired.value && !String(quoteApprovalRemark.value || '').trim()) {
    ElMessage.error(t('industry.manufacturing.error.approvalRemarkRequired'))
    return
  }
  quoteApprovalRemarkSaving.value = true
  try {
    if (['ecns', 'compensationRecords'].includes(action.section?.key)) {
      const actionItem = { key: action.key, labelKey: `industry.manufacturing.action.${action.key}` }
      await runMfgRowAction(action.section, action.row, actionItem, { remark: String(quoteApprovalRemark.value || '').trim() })
      ElMessage.success(t('industry.manufacturing.customer360.saved'))
      await Promise.all([loadSection(action.section), loadDashboard()])
    } else {
      await executeQuoteAction(action.section, action.row, action.key, { remark: String(quoteApprovalRemark.value || '').trim() })
    }
    quoteApprovalRemarkVisible.value = false
    pendingQuoteApprovalAction.value = null
    quoteApprovalRemark.value = ''
  } finally {
    quoteApprovalRemarkSaving.value = false
  }
}

async function runQuoteAction(section, row, key) {
  if (!row?.id) return
  const writeQuoteActions = ['submitApproval', 'approve', 'reject', 'send', 'customerConfirm', 'createOrder']
  if (writeQuoteActions.includes(key) && !canCreateWorkspaceQuote.value && !['approve', 'reject'].includes(key)) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  if (['approve', 'reject'].includes(key) && !hasExactPermission('mfg:quote:approve')) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  if (['approve', 'reject'].includes(key)) {
    openQuoteApprovalRemark(section, row, key)
    return
  }
  await executeQuoteAction(section, row, key)
}

async function executeQuoteAction(section, row, key, payload = {}) {
  if (!row?.id) return
  let res = null
  try {
    if (key === 'submitApproval') res = await api.post(`/api/v1/mfg/quotes/${row.id}/submit-approval`, payload)
    if (key === 'approve') res = await api.post(`/api/v1/mfg/quotes/${row.id}/approve`, payload)
    if (key === 'reject') res = await api.post(`/api/v1/mfg/quotes/${row.id}/reject`, payload)
    if (key === 'send') res = await api.post(`/api/v1/mfg/quotes/${row.id}/send`)
    if (key === 'customerConfirm') res = await api.post(`/api/v1/mfg/quotes/${row.id}/customer-confirm`)
    if (key === 'createOrder') {
      await router.push({ path: '/orders', query: { mfg_quote_id: String(row.id) } })
      return
    }
    if (key === 'revisions') {
      await openQuoteRevisions(row)
      return
    }
    if (key === 'diff') {
      await openQuoteDiff(row)
      return
    }
    const data = unwrapMfgResponseData(res)
    if (data?.item) {
      mergeSavedSectionItem(section, data.item)
      syncActiveQuoteDetail(data.item)
    }
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    const quoteWorkspaceSection = sections.find((item) => item.key === 'quoteWorkspace')
    await Promise.all([
      loadSection(section),
      quoteWorkspaceSection ? loadSection(quoteWorkspaceSection) : Promise.resolve(),
      loadDashboard(),
      selectedRFQID.value ? loadQuoteWorkspaceContext() : Promise.resolve()
    ])
    if (['submitApproval', 'approve', 'reject'].includes(key)) {
      await refreshActiveQuoteDetail(row.id)
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  }
}

async function saveRFQClarification(row) {
  if (!row?.id) return
  if (String(row._original_answer || '').trim() && !String(row.answer || '').trim()) {
    ElMessage.error(t('industry.manufacturing.error.clarificationAnswerLocked'))
    return
  }
  row._saving = true
  try {
    const payload = {
      rfq_id: normalizeID(row.rfq_id || selectedRFQID.value),
      question: row.question || '',
      answer: row.answer || '',
      owner_id: normalizeID(row.owner_id),
      status: row.status || 'open'
    }
    const res = await api.put(`/api/v1/mfg/rfq-clarifications/${row.id}`, payload)
    const data = unwrapMfgResponseData(res)
    if (data?.item) Object.assign(row, data.item)
    row._original_answer = row.answer || ''
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    const clarificationsSection = sections.find((item) => item.key === 'clarifications')
    const quoteWorkspaceSection = sections.find((item) => item.key === 'quoteWorkspace')
    await Promise.all([
      clarificationsSection ? loadSection(clarificationsSection) : Promise.resolve(),
      quoteWorkspaceSection ? loadSection(quoteWorkspaceSection) : Promise.resolve(),
      selectedRFQID.value ? loadQuoteWorkspaceContext() : Promise.resolve()
    ])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('industry.manufacturing.error.clarificationSaveFailed')))
  } finally {
    row._saving = false
  }
}

async function openClarificationAnswerHistory(row) {
  if (!row?.id) return
  activeClarificationHistoryRow.value = row
  clarificationHistoryVisible.value = true
  row._historyLoading = true
  clarificationHistoryLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/rfq-clarifications/${row.id}/answer-histories`)
    const data = unwrapMfgResponseData(res)
    clarificationHistoryItems.value = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
  } catch (error) {
    clarificationHistoryItems.value = []
    ElMessage.error(apiErrorMessage(error, t('industry.manufacturing.rfq.answerHistoryLoadFailed')))
  } finally {
    row._historyLoading = false
    clarificationHistoryLoading.value = false
  }
}

async function openQuoteRevisions(row) {
  quoteRevisionDrawerVisible.value = true
  quoteRevisionLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/quotes/${row.id}/revisions`)
    const data = unwrapMfgResponseData(res)
    quoteRevisionItems.value = Array.isArray(data?.items) ? data.items : []
  } catch (error) {
    quoteRevisionItems.value = []
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    quoteRevisionLoading.value = false
  }
}

async function openQuoteDetail(row) {
  activeQuoteDetail.value = row || null
  quoteDetailVisible.value = Boolean(row)
  if (row?.id) await refreshActiveQuoteDetail(row.id)
}

function syncActiveQuoteDetail(item) {
  if (!item?.id || !activeQuoteDetail.value?.id) return
  if (Number(item.id) !== Number(activeQuoteDetail.value.id)) return
  activeQuoteDetail.value = { ...activeQuoteDetail.value, ...item }
}

function syncActiveQuoteDetailFromWorkspace() {
  const activeID = Number(activeQuoteDetail.value?.id || 0)
  if (!activeID) return
  const latest = (quoteWorkspaceContext.value?.quotes || []).find((item) => Number(item.id || 0) === activeID)
  if (latest) syncActiveQuoteDetail(latest)
}

async function refreshActiveQuoteDetail(id = activeQuoteDetail.value?.id) {
  const quoteID = Number(id || 0)
  if (!quoteID) return
  quoteDetailLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/quotes/${quoteID}`)
    const data = unwrapMfgResponseData(res)
    if (data?.item) syncActiveQuoteDetail(data.item)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    quoteDetailLoading.value = false
  }
}

async function openDeliveryChangeDetail(section, row) {
  activeDeliveryChangeDetailSection.value = section?.key || ''
  activeDeliveryChangeDetail.value = row || null
  deliveryChangeDetailVisible.value = Boolean(row)
  const endpoint = section?.endpoint
  if (!endpoint || !row?.id) return
  deliveryChangeDetailLoading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/${endpoint}/${row.id}`)
    const data = unwrapMfgResponseData(res)
    if (data?.item) activeDeliveryChangeDetail.value = { ...activeDeliveryChangeDetail.value, ...data.item }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  } finally {
    deliveryChangeDetailLoading.value = false
  }
}

async function openQuoteDiff(row) {
  try {
    const res = await api.get(`/api/v1/mfg/quotes/${row.id}/diff`)
    quoteDiffData.value = unwrapMfgResponseData(res) || {}
    quoteDiffVisible.value = true
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  }
}

async function completeCostSheet(row) {
  if (!row?.id) return
  if (!canCreateWorkspaceCost.value) {
    ElMessage.error(t('common.message.permissionDenied'))
    return
  }
  const payload = {
    rfq_id: row.rfq_id,
    bom_snapshot_id: effectiveBOMSnapshotID(row),
    quote_amount: row.amount || row.target_price || 0,
    material_cost: row.material_cost || 0,
    labor_cost: row.labor_cost || 0,
    outsourcing_cost: row.outsourcing_cost || 0,
    manufacturing_overhead: row.manufacturing_overhead || 0,
    sensitive_cost_json: row.sensitive_cost_json || {}
  }
  try {
    const res = await api.post(`/api/v1/mfg/cost-sheets/${row.id}/complete`, payload)
    const data = unwrapMfgResponseData(res)
    const costSection = sections.find((item) => item.key === 'cost')
    if (costSection && data?.item) mergeSavedSectionItem(costSection, data.item)
    ElMessage.success(t('industry.manufacturing.costing.completed'))
    await Promise.all([loadSection(costSection || activeSection.value), loadDashboard(), selectedRFQID.value ? loadQuoteWorkspaceContext() : Promise.resolve()])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('common.message.loadFailed')))
  }
}

function canCompleteCostSheet(row) {
  return canCreateWorkspaceCost.value && row?.status !== 'calculated' && effectiveBOMSnapshotID(row) > 0
}

function isCostingCompleted(row) {
  return ['calculated', 'costed'].includes(String(row?.status || '').toLowerCase())
}

function isBOMMissing(row) {
  return Boolean(row?.bom_missing) || effectiveBOMSnapshotID(row) <= 0
}

function effectiveBOMSnapshotID(row) {
  return Number(row?.bom_snapshot_id || row?.latest_bom_snapshot_id || 0)
}

function primaryText(section, row) {
  if (section?.key === 'samples') return samplePrimaryText(row)
  return row[section.primary] || row.name || row.id || '-'
}

function samplePrimaryText(row = {}) {
  const base = row.sample_no || row.id || '-'
  const context = [row.rfq_business_name || row.rfq_no, row.bom_business_name || row.bom_snapshot_no, row.product_name]
    .filter(Boolean)
    .join(' · ')
  return context ? `${base} · ${context}` : base
}

function readinessMissingLabel(key) {
  return t(`industry.manufacturing.readiness.missing.${normalizeReadinessKey(key)}`)
}

function readinessMissingAction(key) {
  return t(`industry.manufacturing.readiness.fix.${normalizeReadinessKey(key)}`)
}

function readinessActionLabel(key) {
  return t(`industry.manufacturing.readiness.action.${normalizeReadinessKey(key || 'start_costing')}`)
}

function visibleRowActions(section, row) {
  if (section.key === 'ecns') return visibleECNActions(row)
  if (section.key === 'compensationRecords') return visibleCompensationActions(row)
  if (section.key !== 'rfqs') return (section.actions || []).filter((item) => canRunSectionAction(section, item))
  const status = String(row?.status || '').toLowerCase()
  return (section.actions || []).filter((item) => {
    if (!canRunRowAction(section, row, item)) return false
    if (item.key === 'submit') return !['submitted', 'costing', 'costed', 'quoted', 'confirmed', 'closed'].includes(status)
    if (item.key === 'startCosting') return status === 'submitted'
    return item.key === 'readiness'
  })
}

function actionHelpText(actionItem) {
  return t(`industry.manufacturing.actionHelp.${actionItem.key}`)
}

function normalizeReadinessKey(key) {
  return String(key || '').replace(/[^a-zA-Z0-9]+(.)/g, (_, char) => String(char).toUpperCase())
}

function unwrapMfgResponseData(res) {
  if (!res || typeof res !== 'object') return res
  if (res.data && typeof res.data === 'object') return res.data
  return res
}

function mergeSavedSectionItem(section, item) {
  if (!item?.id) return
  const index = section.items.findIndex((entry) => Number(entry.id) === Number(item.id))
  if (index >= 0) {
    section.items.splice(index, 1, item)
  } else {
    section.items.unshift(item)
    section.total = Number(section.total || 0) + 1
  }
}

function defaultRFQForm(row = null) {
  const info = rfqPayload(row)
  return {
    can_edit: Boolean(row?.can_edit),
    can_delete: Boolean(row?.can_delete),
    can_submit: Boolean(row?.can_submit),
    can_create_bom: Boolean(row?.can_create_bom),
    can_create_cost: Boolean(row?.can_create_cost),
    can_create_quote: Boolean(row?.can_create_quote),
    business_name: row?.business_name || '',
    customer_id: Number(row?.customer_id || info.customer_id || 0) || undefined,
    contact_id: Number(row?.contact_id || info.contact_id || 0) || undefined,
    opportunity_id: Number(row?.opportunity_id || info.opportunity_id || 0) || undefined,
    product_family_id: Number(row?.product_family_id || info.product_family_id || 0) || undefined,
    product_type: info.product_type || (info.product_id ? 'standard' : 'custom'),
    product_id: Number(info.product_id || 0) || undefined,
    product_name: info.product_name || '',
    sku: info.sku || '',
    custom_product_name: info.custom_product_name || '',
    specification: info.specification || '',
    key_parameters: info.key_parameters || '',
    material: info.material || '',
    process: info.process || '',
    application_scene: info.application_scene || '',
    quantity: Number(row?.quantity || 0),
    target_price: Number(row?.target_price || 0),
    currency: info.currency || 'CNY',
    required_delivery_date: formatDateForInput(row?.required_delivery_date),
    drawing_file_id: row?.drawing_file_id || info.drawing_file_id || '',
    drawing_storage_node_id: row?.drawing_storage_node_id || info.drawing_storage_node_id || '',
    drawing_file_name: info.drawing_file_name || info.file_name || '',
    drawing_file_size: info.drawing_file_size || info.file_size || 0,
    drawing_download_url: info.drawing_download_url || info.download_url || '',
    no_drawing_reason: row?.no_drawing_reason || '',
    clarification_items: normalizeRFQClarificationItems(info, true)
  }
}

function normalizeRFQClarificationItems(info = {}, includeEmptyDefaults = false) {
  const rawItems = Array.isArray(info.clarification_items) ? info.clarification_items : []
  const items = rawItems.map((item, index) => normalizeRFQClarificationItem(item, index)).filter((item) => item.content || item.remark)
  if (items.length) return items
  const defaults = rfqClarificationCategories
    .map((category, index) => normalizeRFQClarificationItem({
      key: category.value,
      category: category.value,
      label: t(category.labelKey),
      content: info[category.value] || '',
      need_clarification: true,
      sort_order: index + 1
    }, index))
  return includeEmptyDefaults ? defaults : defaults.filter((item) => item.content)
}

function normalizeRFQClarificationItem(item = {}, index = 0) {
  const category = item.category || item.key || rfqClarificationCategories[0].value
  const option = rfqClarificationCategories.find((entry) => entry.value === category) || rfqClarificationCategories[0]
  return {
    key: item.key || `${category}_${Date.now()}_${index}`,
    category,
    label: item.label || t(option.labelKey),
    content: item.content || '',
    remark: item.remark || '',
    need_clarification: item.need_clarification !== false,
    sort_order: Number(item.sort_order || index + 1)
  }
}

function addRFQClarificationItem() {
  if (!Array.isArray(recordForm.clarification_items)) recordForm.clarification_items = []
  recordForm.clarification_items.push(normalizeRFQClarificationItem({
    key: `custom_${Date.now()}`,
    category: rfqClarificationCategories[0].value,
    need_clarification: true,
    sort_order: recordForm.clarification_items.length + 1
  }, recordForm.clarification_items.length))
}

function removeRFQClarificationItem(index) {
  recordForm.clarification_items.splice(index, 1)
  normalizeRFQClarificationSort()
}

function moveRFQClarificationItem(index, direction) {
  const next = index + direction
  if (next < 0 || next >= recordForm.clarification_items.length) return
  const items = recordForm.clarification_items
  const current = items[index]
  items.splice(index, 1)
  items.splice(next, 0, current)
  normalizeRFQClarificationSort()
}

function normalizeRFQClarificationSort() {
  ;(recordForm.clarification_items || []).forEach((item, index) => {
    item.sort_order = index + 1
  })
}

function handleRFQClarificationCategoryChange(item) {
  const option = rfqClarificationCategories.find((entry) => entry.value === item.category)
  item.label = option ? t(option.labelKey) : item.category
}

async function loadRFQDialogOptions(row = null) {
  await Promise.all([
    loadRFQCompanies(),
    loadRFQContacts(recordForm.customer_id),
    loadRFQProducts(),
    loadRFQOpportunities()
  ])
  const info = rfqPayload(row)
  ensureSelectedOption(rfqCompanyOptions.value, row?.customer_id, {
    id: row?.customer_id,
    name: info.customer_name || recordForm.customer_name || ''
  })
  ensureSelectedOption(rfqContactOptions.value, row?.contact_id, {
    id: row?.contact_id,
    name: info.contact_name || recordForm.contact_name || ''
  })
  ensureSelectedOption(rfqOpportunityOptions.value, row?.opportunity_id, {
    id: row?.opportunity_id,
    name: info.opportunity_name || recordForm.opportunity_name || ''
  })
  ensureSelectedOption(rfqProductOptions.value, info.product_id, {
    id: info.product_id,
    name: info.product_name || recordForm.product_name || '',
    code: info.sku || ''
  })
}

async function loadRFQCompanies(keyword = '') {
  rfqOptionsLoading.companies = true
  try {
    const res = await api.get('/api/v1/prd/contacts/objects', {
      params: { object_type: 'company', keyword, page: 1, size: 100 }
    })
    const data = unwrapMfgResponseData(res)
    rfqCompanyOptions.value = data?.items || []
  } finally {
    rfqOptionsLoading.companies = false
  }
}

async function loadRFQContacts(companyID = recordForm.customer_id) {
  rfqOptionsLoading.contacts = true
  try {
    const params = { object_type: 'contact', page: 1, size: 100 }
    if (companyID) params.company_id = companyID
    const res = await api.get('/api/v1/prd/contacts/objects', { params })
    const data = unwrapMfgResponseData(res)
    rfqContactOptions.value = data?.items || []
  } finally {
    rfqOptionsLoading.contacts = false
  }
}

async function loadRFQProducts(keyword = '') {
  rfqOptionsLoading.products = true
  try {
    const res = await api.get('/api/v1/prd/products/objects', {
      params: { object_type: 'product', keyword, page: 1, size: 100 }
    })
    const data = unwrapMfgResponseData(res)
    rfqProductOptions.value = data?.items || []
  } finally {
    rfqOptionsLoading.products = false
  }
}

async function loadRFQOpportunities(keyword = '') {
  rfqOptionsLoading.opportunities = true
  try {
    const res = await api.get('/api/v1/prd/opportunities/objects', {
      params: { object_type: 'opportunity', keyword, page: 1, size: 100 }
    })
    const data = unwrapMfgResponseData(res)
    rfqOpportunityOptions.value = data?.items || []
  } finally {
    rfqOptionsLoading.opportunities = false
  }
}

function handleRFQCompanyChange(value) {
  const company = rfqCompanyOptions.value.find((item) => Number(item.id) === Number(value))
  recordForm.customer_name = company?.name || ''
  recordForm.contact_id = undefined
  recordForm.contact_name = ''
  loadRFQContacts(value)
}

function handleRFQContactChange(value) {
  const contact = rfqContactOptions.value.find((item) => Number(item.id) === Number(value))
  recordForm.contact_name = contact?.name || ''
}

function handleRFQOpportunityChange(value) {
  const opportunity = rfqOpportunityOptions.value.find((item) => Number(item.id) === Number(value))
  recordForm.opportunity_name = opportunity?.name || ''
}

function handleRFQProductChange(value) {
  const product = rfqProductOptions.value.find((item) => Number(item.id) === Number(value))
  const info = crmPayload(product)
  recordForm.product_name = product?.name || ''
  recordForm.sku = info.sku || product?.code || ''
  if (!recordForm.product_family_id && info.product_family_id) {
    recordForm.product_family_id = Number(info.product_family_id)
  }
}

async function uploadRFQDrawing(file) {
  rfqDrawingUploading.value = true
  try {
    const data = new FormData()
    data.append('file', file)
    const upload = await api.post('/api/v1/files/upload', data)
    const info = unwrapFileUploadResponse(upload)
    recordForm.drawing_file_id = info.file_id
    recordForm.drawing_storage_node_id = info.storage_node_id || ''
    recordForm.drawing_file_name = info.original_name || info.file_name || file.name
    recordForm.drawing_file_size = info.size || file.size || 0
    recordForm.drawing_download_url = info.download_url || `/api/v1/files/${info.file_id}/download`
    recordForm.no_drawing_reason = ''
    ElMessage.success(t('industry.manufacturing.rfq.uploadSuccess'))
  } finally {
    rfqDrawingUploading.value = false
  }
  return false
}

async function downloadRFQCustomerFile(source) {
  const fileID = source?.file_id || source?.drawing_file_id || ''
  const url = source?.download_url || source?.drawing_download_url || (fileID ? `/api/v1/files/${fileID}/download` : '')
  const name = source?.file_name || source?.drawing_file_name || t('industry.manufacturing.rfq.customerFile')
  await downloadProtectedFile(url, name)
}

function buildRFQPayload() {
  const product = rfqProductOptions.value.find((item) => Number(item.id) === Number(recordForm.product_id || 0))
  const productInfo = crmPayload(product)
  const company = rfqCompanyOptions.value.find((item) => Number(item.id) === Number(recordForm.customer_id || 0))
  const contact = rfqContactOptions.value.find((item) => Number(item.id) === Number(recordForm.contact_id || 0))
  const opportunity = rfqOpportunityOptions.value.find((item) => Number(item.id) === Number(recordForm.opportunity_id || 0))
  const clarificationItems = (recordForm.clarification_items || [])
    .filter((item) => String(item.content || '').trim() || String(item.remark || '').trim())
    .map((item, index) => {
      const option = rfqClarificationCategories.find((entry) => entry.value === item.category)
      return {
        key: item.key || `${item.category || 'requirement'}_${index + 1}`,
        category: item.category || option?.value || 'technical_requirements',
        label: item.label || (option ? t(option.labelKey) : ''),
        content: String(item.content || '').trim(),
        remark: String(item.remark || '').trim(),
        need_clarification: item.need_clarification !== false,
        sort_order: index + 1
      }
    })
  const legacyByCategory = Object.fromEntries(rfqClarificationCategories.map((category) => [
    category.value,
    clarificationItems.find((item) => item.category === category.value)?.content || ''
  ]))
  const payload = {
    customer_id: normalizeID(recordForm.customer_id) || undefined,
    customer_name: company?.name || recordForm.customer_name || '',
    contact_id: normalizeID(recordForm.contact_id) || undefined,
    contact_name: contact?.name || recordForm.contact_name || '',
    opportunity_id: normalizeID(recordForm.opportunity_id) || undefined,
    opportunity_name: opportunity?.name || recordForm.opportunity_name || '',
    product_type: recordForm.product_type || 'standard',
    product_id: normalizeID(recordForm.product_id) || undefined,
    product_name: product?.name || recordForm.product_name || '',
    sku: productInfo.sku || recordForm.sku || product?.code || '',
    product_family_id: normalizeID(recordForm.product_family_id) || undefined,
    custom_product_name: recordForm.custom_product_name || '',
    specification: recordForm.specification || '',
    key_parameters: recordForm.key_parameters || '',
    material: recordForm.material || '',
    process: recordForm.process || '',
    application_scene: recordForm.application_scene || '',
    currency: recordForm.currency || 'CNY',
    drawing_file_id: recordForm.drawing_file_id || '',
    drawing_storage_node_id: recordForm.drawing_storage_node_id || '',
    drawing_file_name: recordForm.drawing_file_name || '',
    drawing_file_size: recordForm.drawing_file_size || 0,
    drawing_download_url: recordForm.drawing_download_url || '',
    clarification_items: clarificationItems,
    technical_requirements: legacyByCategory.technical_requirements || '',
    quality_requirements: legacyByCategory.quality_requirements || '',
    packaging_requirements: legacyByCategory.packaging_requirements || '',
    customer_notes: legacyByCategory.customer_notes || '',
    commercial_terms: legacyByCategory.commercial_terms || ''
  }
  return {
    business_name: String(recordForm.business_name || '').trim(),
    customer_id: normalizeID(recordForm.customer_id),
    contact_id: normalizeID(recordForm.contact_id),
    opportunity_id: normalizeID(recordForm.opportunity_id),
    product_family_id: normalizeID(recordForm.product_family_id),
    quantity: Number(recordForm.quantity || 0),
    target_price: Number(recordForm.target_price || 0),
    required_delivery_date: normalizeDateForSubmit(recordForm.required_delivery_date),
    drawing_file_id: recordForm.drawing_file_id || '',
    drawing_storage_node_id: recordForm.drawing_storage_node_id || '',
    no_drawing_reason: recordForm.drawing_file_id ? '' : (recordForm.no_drawing_reason || ''),
    payload
  }
}

function rfqPayload(row) {
  return parseJSONPayload(row?.payload_json)
}

function parseJSONPayload(value) {
  try {
    return typeof value === 'string' ? JSON.parse(value || '{}') : (value || {})
  } catch {
    return {}
  }
}

function rfqCustomerText(row) {
  const info = rfqPayload(row)
  const company = rfqCompanyOptions.value.find((item) => Number(item.id) === Number(row.customer_id || info.customer_id || 0))
  return info.customer_name || company?.name || (row.customer_id ? `${t('industry.manufacturing.field.customer')} #${row.customer_id}` : '-')
}

function rfqDisplayName(row) {
  return row?.business_name || row?.rfq_business_name || row?.rfq_no || row?.id || '-'
}

function rfqProductText(row) {
  const info = rfqPayload(row)
  if (info.product_type === 'custom' || info.custom_product_name) {
    return info.custom_product_name || t('industry.manufacturing.rfq.productType.custom')
  }
  return info.product_name || info.sku || t('industry.manufacturing.rfq.productDemandPending')
}

function rfqTargetPriceText(row) {
  const info = rfqPayload(row)
  if (isMaskedPrice(row.target_price)) return '***'
  if (!Number(row.target_price)) return '-'
  return `${info.currency || 'CNY'} ${money(row.target_price)}`
}

function rfqReadinessAction(row) {
  const status = String(row?.status || '').toLowerCase()
  if (status === 'draft') return 'start_costing'
  if (status === 'submitted') return 'start_costing'
  if (status === 'costing') return 'create_bom_snapshot'
  if (['costed', 'calculated'].includes(status)) return 'create_quote'
  return 'start_costing'
}

function rfqWorkspaceDetailRows(row) {
  const info = rfqPayload(row)
  const productType = info.product_type === 'custom' || info.custom_product_name
    ? t('industry.manufacturing.rfq.productType.custom')
    : t('industry.manufacturing.rfq.productType.standard')
  return [
    {
      key: 'customer',
      title: t('industry.manufacturing.rfq.section.customerInfo'),
      items: [
        { key: 'customer', label: t('industry.manufacturing.rfq.customer'), value: rfqCustomerText(row) },
        { key: 'contact', label: t('industry.manufacturing.rfq.contact'), value: info.contact_name || (row.contact_id ? `#${row.contact_id}` : '-') },
        { key: 'opportunity', label: t('industry.manufacturing.rfq.opportunity'), value: info.opportunity_name || (row.opportunity_id ? `#${row.opportunity_id}` : '-') },
        { key: 'status', label: t('industry.manufacturing.field.status'), value: mfgStatusText(row.status) }
      ]
    },
    {
      key: 'product',
      title: t('industry.manufacturing.rfq.section.productDemand'),
      items: [
        { key: 'type', label: t('industry.manufacturing.rfq.productType'), value: productType },
        { key: 'product', label: t('industry.manufacturing.field.productDemand'), value: rfqProductText(row) },
        { key: 'sku', label: t('industry.manufacturing.rfq.sku'), value: info.sku || '-' },
        { key: 'custom', label: t('industry.manufacturing.rfq.customProductName'), value: info.custom_product_name || '-' },
        { key: 'spec', label: t('industry.manufacturing.rfq.specification'), value: info.specification || '-' },
        { key: 'params', label: t('industry.manufacturing.rfq.keyParameters'), value: info.key_parameters || '-' },
        { key: 'material', label: t('industry.manufacturing.rfq.material'), value: info.material || '-' },
        { key: 'process', label: t('industry.manufacturing.rfq.process'), value: info.process || '-' },
        { key: 'scene', label: t('industry.manufacturing.rfq.applicationScene'), value: info.application_scene || '-' }
      ]
    },
    {
      key: 'quote',
      title: t('industry.manufacturing.rfq.section.quoteRequest'),
      items: [
        { key: 'quantity', label: t('industry.manufacturing.rfq.table.quantity'), value: row.quantity || '-' },
        { key: 'target', label: t('industry.manufacturing.rfq.table.targetPrice'), value: rfqTargetPriceText(row) },
        { key: 'currency', label: t('industry.manufacturing.rfq.currency'), value: info.currency || 'CNY' },
        { key: 'delivery', label: t('industry.manufacturing.rfq.requiredDeliveryDate'), value: formatDate(row.required_delivery_date) }
      ]
    },
    {
      key: 'drawing',
      title: t('industry.manufacturing.rfq.section.drawing'),
      items: [
        {
          key: 'file',
          label: t('industry.manufacturing.rfq.drawingUpload'),
          value: info.drawing_file_name || row.drawing_file_id || t('industry.manufacturing.rfq.noDrawingUploaded'),
          kind: 'download',
          file_id: row.drawing_file_id || info.drawing_file_id || '',
          download_url: info.drawing_download_url || info.download_url || '',
          file_name: info.drawing_file_name || ''
        },
        { key: 'reason', label: t('industry.manufacturing.rfq.noDrawingReason'), value: row.no_drawing_reason || '-' }
      ]
    },
    {
      key: 'notes',
      title: t('industry.manufacturing.rfq.clarificationItems'),
      items: rfqDetailClarificationItems(info)
    }
  ]
}

function rfqDetailClarificationItems(info) {
  const items = normalizeRFQClarificationItems(info)
  if (!items.length) {
    return [{ key: 'empty', label: t('industry.manufacturing.rfq.clarificationItems'), value: t('industry.manufacturing.rfq.noClarificationItems') }]
  }
  return items.map((item) => ({
    key: item.key,
    label: item.label,
    value: item.remark ? `${item.content} / ${item.remark}` : item.content
  }))
}

function costPayload(row) {
  return row?.rfq_payload || parseJSONPayload(row?.payload_json)
}

function costCustomerText(row) {
  const info = costPayload(row)
  return row.customer_name || info.customer_name || (row.customer_id ? `${t('industry.manufacturing.field.customer')} #${row.customer_id}` : '-')
}

function costProductText(row) {
  const info = costPayload(row)
  return row.product_name || info.product_name || info.custom_product_name || info.sku || t('industry.manufacturing.rfq.productDemandPending')
}

function costTargetPriceText(row) {
  const info = costPayload(row)
  if (isMaskedPrice(row.target_price) || isMaskedPrice(row.amount)) return '***'
  const value = Number(row.target_price || row.amount || 0)
  return value ? `${row.currency || info.currency || 'CNY'} ${money(value)}` : '-'
}

function costNextActionText(row) {
  if (isCostingCompleted(row)) return t('industry.manufacturing.costing.next.createQuote')
  return t(`industry.manufacturing.costing.next.${normalizeReadinessKey(row.next_action || (row.bom_missing ? 'create_bom_snapshot' : 'calculate_cost'))}`)
}

function costRiskText(row) {
  const flags = Array.isArray(row.risk_flags) ? row.risk_flags : []
  if (!flags.length) return t('industry.manufacturing.costing.noRisk')
  return flags.map((flag) => t(`industry.manufacturing.costing.risk.${normalizeReadinessKey(flag)}`)).join(' / ')
}

function companyOptionLabel(item) {
  const info = crmPayload(item)
  return `${item.name || '-'}${info.industry ? ` · ${info.industry}` : ''}`
}

function contactOptionLabel(item) {
  const info = crmPayload(item)
  const detail = info.phone || info.email || info.company || info.company_name || ''
  return `${item.name || '-'}${detail ? ` · ${detail}` : ''}`
}

function productOptionLabel(item) {
  const info = crmPayload(item)
  const sku = info.sku || item.code || ''
  return `${item.name || '-'}${sku ? ` · ${sku}` : ''}`
}

function opportunityOptionLabel(item) {
  return `${item.name || '-'}${item.amount ? ` · ${money(item.amount)}` : ''}`
}

function relationOptionLabel(name, item) {
  if (name === 'companies') return companyOptionLabel(item)
  if (name === 'productFamilies') return `${item.name || item.family_code || '-'}${item.family_code ? ` · ${item.family_code}` : ''}`
  if (name === 'rfqs') return rfqOptionLabel(item)
  if (name === 'bomTemplates') return bomTemplateOptionLabel(item)
  if (name === 'bom') return bomSnapshotOptionLabel(item)
  if (name === 'cost') return `${item.rfq_no || item.id}${item.margin_level ? ` · ${mfgStatusText(item.margin_level)}` : ''}`
  if (name === 'quotes') return `${item.quote_no || item.id}${item.amount ? ` · ${money(item.amount)}` : ''}`
  if (name === 'orders') return `${item.name || item.code || item.id}${item.amount ? ` · ${money(item.amount)}` : ''}`
  if (name === 'assets') return `${item.serial_no || item.id}${item.install_location ? ` · ${item.install_location}` : ''}`
  if (name === 'service') return `${item.fault_code || item.id}${item.service_type ? ` · ${item.service_type}` : ''}`
  return item.name || item.code || item.id || '-'
}

function rfqOptionLabel(item) {
  const summary = buildRFQSummary(item)
  const main = summary.business_name || summary.rfq_no || summary.id || '-'
  const no = summary.business_name && summary.rfq_no ? ` · ${summary.rfq_no}` : ''
  return `${main}${no}${summary.customer_name ? ` · ${summary.customer_name}` : ''}${summary.product_name ? ` · ${summary.product_name}` : ''}`
}

function buildRFQSummary(row) {
  if (!row) return {}
  const info = rfqPayload(row)
  return {
    id: row.id,
    rfq_no: row.rfq_no,
    business_name: row.business_name || row.rfq_business_name || '',
    customer_name: info.customer_name || row.customer_name || '',
    product_name: row.product_name || info.product_name || info.custom_product_name || info.sku || '',
    product_family_id: normalizeID(row.product_family_id || info.product_family_id),
    quantity: row.quantity || 0,
    target_price: row.target_price || 0,
    required_delivery_date: row.required_delivery_date,
    no_drawing_reason: row.no_drawing_reason || ''
  }
}

function bomSnapshotDisplayName(row) {
  return row?.business_name || row?.bom_business_name || row?.snapshot_no || row?.bom_snapshot_no || row?.id || '-'
}

function bomSnapshotOptionLabel(item) {
  const main = bomSnapshotDisplayName(item)
  const no = (item.business_name || item.bom_business_name) && (item.snapshot_no || item.bom_snapshot_no) ? ` · ${item.snapshot_no || item.bom_snapshot_no}` : ''
  const rfq = item.rfq_business_name || item.rfq_no || (item.rfq_id ? `${t('industry.manufacturing.field.rfq')} #${item.rfq_id}` : '')
  return `${main}${no}${rfq ? ` · ${rfq}` : ''}`
}

function optionLabel(option) {
  if (typeof option === 'boolean') return t(`industry.manufacturing.option.${option ? 'yes' : 'no'}`)
  if (/^[A-Z]{3}$/.test(String(option))) return String(option)
  return t(`industry.manufacturing.option.${String(option).replace(/[^a-zA-Z0-9]+(.)/g, (_, char) => String(char).toUpperCase())}`)
}

function normalizeID(value) {
  const id = Number(value || 0)
  return Number.isFinite(id) && id > 0 ? id : 0
}

function ensureSelectedOption(options, id, fallback) {
  if (!id || options.some((item) => Number(item.id) === Number(id))) return
  if (fallback?.name) options.push(fallback)
}

function formatDateForInput(value) {
  if (!value) return ''
  return String(value).slice(0, 10)
}

function formatDateTimeForInput(value) {
  if (!value) return ''
  const date = new Date(value)
  if (!Number.isNaN(date.getTime())) {
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  }
  return String(value).replace('T', ' ').slice(0, 16)
}

function normalizeDateForSubmit(value) {
  if (!value) return null
  const text = String(value)
  return text.includes('T') ? text : `${text}T00:00:00Z`
}

function normalizeDateTimeForSubmit(value) {
  if (!value) return null
  const text = String(value).trim()
  if (!text) return null
  if (text.includes('T')) return text
  const match = text.match(/^(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2})$/)
  if (!match) return text
  const [, year, month, day, hour, minute] = match
  const date = new Date(Number(year), Number(month) - 1, Number(day), Number(hour), Number(minute), 0)
  return Number.isNaN(date.getTime()) ? text : date.toISOString()
}

function statusTagType(value) {
  const text = String(value || '').toLowerCase()
  if (['red', 'failed', 'rejected', 'overdue'].includes(text)) return 'danger'
  if (['yellow', 'pending', 'open', 'draft'].includes(text)) return 'warning'
  if (['green', 'approved', 'active', 'closed', 'resolved', 'sent'].includes(text)) return 'success'
  if (['inactive', 'disabled'].includes(text)) return 'info'
  return 'info'
}

function impactLevelTagType(value) {
  const text = String(value || '').toLowerCase()
  if (text === 'critical') return 'danger'
  if (text === 'major') return 'warning'
  return 'info'
}

function impactLevelText(value) {
  return mfgStatusText(value || 'normal')
}

function ecnStatusText(value) {
  const normalized = normalizeReadinessKey(value || 'draft')
  const key = `industry.manufacturing.ecn.status.${normalized}`
  const translated = t(key)
  return isRealTranslation(key, translated) ? translated : mfgStatusText(value)
}

function compensationStatusText(value) {
  const normalized = normalizeReadinessKey(value || 'open')
  const key = `industry.manufacturing.compensation.status.${normalized}`
  const translated = t(key)
  return isRealTranslation(key, translated) ? translated : optionLabel(value || 'open')
}

function ecnCustomerConfirmText(value) {
  return t(`industry.manufacturing.ecn.customerConfirm.${value ? 'confirmed' : 'pending'}`)
}

function deliveryChangeStatusText(row = {}) {
  return activeDeliveryChangeDetailSection.value === 'ecns' ? ecnStatusText(row.status) : compensationStatusText(row.status)
}

function canEditECN(row = {}) {
  const approval = String(row.approval_status || 'draft').toLowerCase()
  const status = String(row.status || 'draft').toLowerCase()
  return status !== 'effective' && !['pending', 'approved'].includes(approval)
}

function canDeleteECN(row = {}) {
  const approval = String(row.approval_status || 'draft').toLowerCase()
  const status = String(row.status || 'draft').toLowerCase()
  return status === 'draft' && ['draft', 'rejected', ''].includes(approval)
}

function visibleECNActions(row = {}) {
  const approval = String(row.approval_status || 'draft').toLowerCase()
  const status = String(row.status || 'draft').toLowerCase()
  const canManage = hasPermission('mfg:ecn:manage')
  const canApprove = hasPermission('mfg:ecn:approve')
  const actions = []
  if (canManage && status !== 'effective' && ['draft', 'impact_analyzed'].includes(status) && ['draft', 'rejected', ''].includes(approval)) {
    actions.push({ key: 'impactAnalysis', labelKey: 'industry.manufacturing.action.impactAnalysis' })
    actions.push({ key: 'submitApproval', labelKey: 'industry.manufacturing.action.submitApproval', type: 'primary' })
  }
  if (approval === 'pending' && canApprove) {
    actions.push({ key: 'approve', labelKey: 'industry.manufacturing.action.approve', type: 'success' })
    actions.push({ key: 'reject', labelKey: 'industry.manufacturing.action.reject', type: 'warning' })
  }
  if (canManage && approval === 'approved' && status !== 'effective' && !row.customer_confirmed) {
    actions.push({ key: 'confirm', labelKey: 'industry.manufacturing.action.customerConfirm', type: 'success' })
  }
  if (canManage && approval === 'approved' && row.customer_confirmed && status !== 'effective') {
    actions.push({ key: 'effective', labelKey: 'industry.manufacturing.action.effective', type: 'success' })
  }
  return actions
}

function canEditCompensation(row = {}) {
  const approval = String(row.approval_status || 'draft').toLowerCase()
  const status = String(row.status || 'open').toLowerCase()
  return status !== 'closed' && ['draft', 'rejected', ''].includes(approval)
}

function canDeleteCompensation(row = {}) {
  const approval = String(row.approval_status || 'draft').toLowerCase()
  const status = String(row.status || 'open').toLowerCase()
  return status !== 'closed' && ['draft', 'rejected', ''].includes(approval)
}

function visibleCompensationActions(row = {}) {
  const approval = String(row.approval_status || 'draft').toLowerCase()
  const status = String(row.status || 'open').toLowerCase()
  const canManage = hasPermission('mfg:compensation:manage')
  const canApprove = hasPermission('mfg:compensation:approve')
  const actions = []
  if (canManage && status !== 'closed' && ['draft', 'rejected', ''].includes(approval)) {
    actions.push({ key: 'submitApproval', labelKey: 'industry.manufacturing.action.submitApproval', type: 'primary' })
  }
  if (approval === 'pending' && canApprove) {
    actions.push({ key: 'approve', labelKey: 'industry.manufacturing.action.approve', type: 'success' })
    actions.push({ key: 'reject', labelKey: 'industry.manufacturing.action.reject', type: 'warning' })
  }
  if (canManage && approval === 'approved' && status !== 'closed') {
    actions.push({ key: 'closeCompensation', labelKey: 'industry.manufacturing.action.close', type: 'success' })
  }
  return actions
}

function mfgStatusText(value) {
  if (!value) return '-'
  const normalized = String(value).replace(/[^a-zA-Z0-9]+(.)/g, (_, char) => String(char).toUpperCase())
  const key = `industry.manufacturing.status.${normalized}`
  const translated = t(key)
  if (isRealTranslation(key, translated)) return translated
  const optionKey = `industry.manufacturing.option.${normalized}`
  const optionTranslated = t(optionKey)
  return isRealTranslation(optionKey, optionTranslated) ? optionTranslated : value
}

function sampleStageText(value) {
  if (!value) return '-'
  const key = `industry.manufacturing.sampleStage.${normalizeReadinessKey(value)}`
  const translated = t(key)
  return isRealTranslation(key, translated) ? translated : value
}

function productionMilestoneText(value) {
  if (!value) return '-'
  const key = `industry.manufacturing.productionMilestone.${normalizeReadinessKey(value)}`
  const translated = t(key)
  return isRealTranslation(key, translated) ? translated : value
}

function productionStatusText(value) {
  if (!value) return '-'
  const key = `industry.manufacturing.productionStatus.${normalizeReadinessKey(value)}`
  const translated = t(key)
  return isRealTranslation(key, translated) ? translated : value
}

function productionStatusTagType(value) {
  const status = String(value || '').toLowerCase()
  if (status === 'completed') return 'success'
  if (status === 'delayed') return 'danger'
  if (status === 'paused' || status === 'canceled') return 'warning'
  if (status === 'in_progress') return 'primary'
  return 'info'
}

function productionPrimaryContext(row = {}) {
  return [row.order_name || row.order_code || (row.order_id ? `#${row.order_id}` : ''), row.erp_order_no, row.mes_work_order_no].filter(Boolean).join(' · ') || '-'
}

function fieldOptionText(field, option) {
  if (field.optionType === 'clarificationCategory') return clarificationCategoryLabel(option)
  if (field.optionType === 'mfgStatus') return mfgStatusText(option)
  if (field.optionType === 'productionStatus') return productionStatusText(option)
  if (field.optionType === 'productionMilestone') return productionMilestoneText(option)
  return optionLabel(option)
}

function creatableOptionText(field, option) {
  if (field.optionType === 'productionMilestone') return productionMilestoneText(option)
  if (field.optionType === 'sampleStage') return sampleStageText(option)
  return optionLabel(option)
}

function sectionStatusText(section, row) {
  const value = row.status || row.stage || row.risk_level
  if (section?.key === 'samples') return sampleStageText(value)
  if (section?.key === 'production') return productionStatusText(row.progress_status || value)
  if (section?.key === 'compensationRecords') return compensationStatusText(value)
  if (section?.key === 'product') return mfgStatusText(normalizeProductFamilyStatus(value))
  return mfgStatusText(value)
}

function normalizeProductFamilyStatus(value) {
  const text = String(value || '').trim().toLowerCase()
  if (['inactive', 'disabled', 'disable', 'off', 'closed', 'deleted'].includes(text)) return 'inactive'
  return 'active'
}

function isRealTranslation(key, translated) {
  const text = String(translated || '').trim()
  if (!text || text === key) return false
  return text !== key.replace(/[._-]+/g, ' ').trim()
}

function clarificationSourceText(row) {
  const raw = row?.source_label || row?.source_key || row?.source_type || ''
  const category = rfqClarificationCategories.find((item) => item.value === row?.source_key || item.value === row?.source_label)
  if (category) return t(category.labelKey)
  return raw || t('industry.manufacturing.rfq.uncategorizedClarification')
}

function clarificationCategoryLabel(value) {
  const category = rfqClarificationCategories.find((item) => item.value === value)
  return category ? t(category.labelKey) : (value || '-')
}

function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value).slice(0, 19)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function userName(userID) {
  const id = normalizeID(userID)
  if (!id) return '-'
  const user = userOptions.value.find((item) => Number(item.id) === id)
  return user ? userDisplayName(user) : `#${id}`
}

function answerUserName(userID, savedName = '') {
  const name = String(savedName || '').trim()
  if (name) return name
  const id = normalizeID(userID)
  if (!id) return '-'
  const user = userOptions.value.find((item) => Number(item.id) === id)
  return user ? userDisplayName(user) : '-'
}

function userDisplayName(user) {
  return user?.name || user?.nickname || user?.username || user?.email || `#${user?.id || ''}`
}

function field(key, labelKey, type = null, wide = false, options = null, optionType = '') {
  return { key, labelKey, type, wide, options, optionType, kind: type === 'user' ? 'user' : undefined }
}

function action(key, labelKey, run) {
  return { key, labelKey, run }
}

function formSpec(titleKey, descKey, groups) {
  return { titleKey, editTitleKey: 'industry.manufacturing.form.editTitle', descKey, groups }
}

function group(titleKey, fields) {
  return { titleKey, fields }
}

function textField(key, labelKey, wide = false) {
  return { key, labelKey, kind: 'text', wide }
}

function textAreaField(key, labelKey) {
  return { key, labelKey, kind: 'textarea', wide: true }
}

function numberField(key, labelKey, extra = {}) {
  return { key, labelKey, kind: 'number', ...extra }
}

function dateField(key, labelKey) {
  return { key, labelKey, kind: 'date' }
}

function dateTimeField(key, labelKey) {
  return { key, labelKey, kind: 'datetime' }
}

function userField(key, labelKey) {
  return { key, labelKey, kind: 'user' }
}

function selectField(key, labelKey, options, optionType = '') {
  return { key, labelKey, kind: 'select', options, optionType }
}

function creatableSelectField(key, labelKey, options, optionType = '') {
  return { key, labelKey, kind: 'creatableSelect', options, optionType }
}

function clarificationCategoryField(key, labelKey) {
  return { key, labelKey, kind: 'select', options: rfqClarificationCategories.map((item) => item.value), optionType: 'clarificationCategory' }
}

function relationField(key, labelKey, relation) {
  return { key, labelKey, kind: 'relation', relation }
}

function readonlyField(key, labelKey, relation = '') {
  return { key, labelKey, kind: 'readonly', relation }
}

function defaultFieldValue(item) {
  if (item.type === 'number') return 0
  return ''
}

function normalizeFormValue(value) {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return value
}

function normalizeBOMSnapshotPayload(payload) {
  return {
    ...payload,
    bom_template_id: Number(payload.source_template_id || payload.bom_template_id || 0),
    product_family_id: normalizeID(payload.product_family_id),
    config_values: parseObjectField(payload.config_values),
    template_rule: parseObjectField(payload.template_rule)
  }
}

function parseObjectField(value) {
	if (!value) return {}
	if (typeof value === 'object') return value
  try {
    return JSON.parse(value)
  } catch {
    return {}
  }
}

function parseArrayField(value) {
  if (!value) return []
  if (Array.isArray(value)) return value
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value || '[]') : value
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function formatDate(value) {
  if (!value) return '-'
  return String(value).slice(0, 10)
}

function numberText(value) {
  const number = Number(value || 0)
  if (!Number.isFinite(number)) return '0'
  return number.toLocaleString(undefined, { maximumFractionDigits: 0 })
}

function percentText(value) {
  const number = Number(value || 0)
  if (!Number.isFinite(number)) return '0%'
  return `${(number * 100).toFixed(1)}%`
}

function hourText(value) {
  const number = Number(value || 0)
  if (!Number.isFinite(number)) return '-'
  return `${number.toFixed(1)}h`
}

function money(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '-'
  return number.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function moneyWithCurrency(value, currency = 'CNY') {
  const amount = money(value)
  return amount === '-' ? '-' : `${currency || 'CNY'} ${amount}`
}

function customerNameText(row) {
  return row?.customer_name || (row?.customer_id ? `#${row.customer_id}` : '-')
}

function salesOwnerNameText(row) {
  return row?.sales_owner_name || userName(row?.sales_owner_id)
}
</script>

<style scoped>
.manufacturing-workspace {
  min-height: 100%;
}

.manufacturing-workspace__header {
  margin-bottom: 16px;
  padding: 24px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.manufacturing-workspace__header p {
  margin: 0 0 8px;
  color: var(--el-color-primary);
  font-weight: 700;
}

.manufacturing-workspace__header h2 {
  margin: 0 0 8px;
  font-size: 28px;
  line-height: 1.25;
  letter-spacing: 0;
}

.manufacturing-workspace__header span,
.mfg-section__toolbar p {
  color: var(--el-text-color-secondary);
  line-height: 1.7;
}

.manufacturing-workspace__header-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.mfg-executive-dashboard {
  display: grid;
  grid-template-columns: minmax(0, 1.18fr) minmax(360px, 0.82fr);
  gap: 16px;
}

.mfg-executive-panel {
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.mfg-executive-panel--wide {
  grid-column: 1 / -1;
}

.mfg-executive-panel__head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.mfg-executive-panel__head h3 {
  margin: 0 0 6px;
  font-size: 18px;
  letter-spacing: 0;
}

.mfg-executive-panel__head p,
.mfg-executive-panel__head span,
.mfg-kpi-card span,
.mfg-kpi-card small,
.mfg-business-metrics span,
.mfg-delivery-risk span,
.mfg-on-time-rate span,
.mfg-metric-group-card span,
.mfg-metric-group-card small {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.55;
}

.mfg-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.mfg-kpi-card,
.mfg-metric-group-card {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  text-align: left;
}

.mfg-kpi-card {
  min-height: 118px;
  padding: 14px;
}

.mfg-kpi-card.is-clickable,
.mfg-metric-group-card.is-clickable {
  cursor: pointer;
}

.mfg-kpi-card span,
.mfg-kpi-card small,
.mfg-kpi-card strong {
  display: block;
}

.mfg-kpi-card strong {
  margin: 10px 0 6px;
  font-size: 26px;
  line-height: 1.1;
}

.mfg-kpi-card.tone-blue {
  border-color: var(--el-color-primary-light-7);
  background: var(--el-color-primary-light-9);
}

.mfg-kpi-card.tone-green {
  border-color: var(--el-color-success-light-7);
  background: var(--el-color-success-light-9);
}

.mfg-kpi-card.tone-yellow {
  border-color: var(--el-color-warning-light-7);
  background: var(--el-color-warning-light-9);
}

.mfg-kpi-card.tone-red {
  border-color: var(--el-color-danger-light-7);
  background: var(--el-color-danger-light-9);
}

.mfg-business-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.mfg-business-metrics > div,
.mfg-delivery-risk > div,
.mfg-on-time-rate {
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.mfg-business-metrics strong,
.mfg-delivery-risk strong,
.mfg-on-time-rate strong {
  display: block;
  margin-top: 6px;
  font-size: 20px;
}

.mfg-delivery-risk {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.mfg-delivery-risk .risk-green {
  border-color: var(--el-color-success-light-7);
  background: var(--el-color-success-light-9);
}

.mfg-delivery-risk .risk-yellow {
  border-color: var(--el-color-warning-light-7);
  background: var(--el-color-warning-light-9);
}

.mfg-delivery-risk .risk-red {
  border-color: var(--el-color-danger-light-7);
  background: var(--el-color-danger-light-9);
}

.mfg-on-time-rate {
  margin-top: 10px;
}

.mfg-metric-group-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.mfg-metric-group-card {
  padding: 14px;
}

.mfg-metric-group-card header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.mfg-metric-group-card__body {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.mfg-metric-group-card__body > div {
  min-height: 64px;
  padding: 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.mfg-metric-group-card__body strong,
.mfg-metric-group-card__body span {
  display: block;
}

.mfg-metric-group-card__body strong {
  margin-top: 6px;
  font-size: 18px;
}

.mfg-metric-group-card small {
  display: block;
  margin-top: 10px;
}

.manufacturing-workspace__layout {
  display: grid;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  gap: 16px;
}

.manufacturing-workspace__layout.is-single {
  grid-template-columns: minmax(0, 1fr);
}

.manufacturing-workspace__companies,
.manufacturing-workspace__main {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  padding: 16px;
}


.mfg-crm-scope strong {
  display: block;
  margin-bottom: 6px;
}

.manufacturing-company-list {
  min-height: 320px;
  max-height: calc(100vh - 360px);
  margin-top: 12px;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.manufacturing-company-list__item {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  text-align: left;
  cursor: pointer;
}

.manufacturing-company-list__item.active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.manufacturing-company-list__item strong,
.manufacturing-company-list__item span {
  display: block;
}

.manufacturing-company-list__item span {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mfg-section__toolbar {
  margin-bottom: 14px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.mfg-section__toolbar h3 {
  margin: 0 0 4px;
  font-size: 20px;
  letter-spacing: 0;
}

.mfg-section__toolbar p {
  margin: 0;
}

.mfg-rfq-scope-note {
  margin-top: 8px !important;
  color: var(--el-color-primary) !important;
  font-weight: 600;
}

.mfg-rfq-workspace-dialog :deep(.el-dialog__body) {
  padding-top: 8px;
}

.mfg-rfq-dialog {
  max-height: min(76vh, 820px);
  overflow: auto;
  padding-right: 4px;
}

.mfg-rfq-dialog__hero {
  margin-bottom: 12px;
  padding: 16px;
  display: flex;
  justify-content: space-between;
  gap: 16px;
  border: 1px solid #d8e7f7;
  border-radius: 8px;
  background: linear-gradient(135deg, #f6fbff 0%, #f8fbf7 100%);
}

.mfg-rfq-dialog__hero h3 {
  margin: 4px 0 6px;
  color: #1f2f46;
  font-size: 20px;
  letter-spacing: 0;
}

.mfg-rfq-dialog__hero p {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.55;
}

.mfg-rfq-dialog__eyebrow {
  color: #2f6f9f;
  font-size: 12px;
  font-weight: 700;
}

.mfg-rfq-dialog__hero-meta {
  min-width: min(320px, 100%);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-regular);
  font-weight: 600;
  text-align: right;
}

.mfg-rfq-dialog__summary {
  margin-bottom: 14px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.mfg-rfq-dialog__metric {
  min-height: 82px;
  padding: 13px;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  box-shadow: 0 8px 20px rgb(31 47 70 / 6%);
}

.mfg-rfq-dialog__metric .el-icon {
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  border-radius: 8px;
  font-size: 16px;
}

.mfg-rfq-dialog__metric.is-blue .el-icon {
  background: #eaf3ff;
  color: #2f6fdb;
}

.mfg-rfq-dialog__metric.is-cyan .el-icon {
  background: #e8f7f8;
  color: #16878c;
}

.mfg-rfq-dialog__metric.is-amber .el-icon {
  background: #fff5df;
  color: #b57612;
}

.mfg-rfq-dialog__metric.is-green .el-icon {
  background: #ecf8ef;
  color: #2f8f51;
}

.mfg-rfq-dialog__metric span,
.mfg-rfq-dialog__metric strong {
  display: block;
}

.mfg-rfq-dialog__metric span {
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mfg-rfq-dialog__metric strong {
  color: #25364f;
  line-height: 1.45;
  word-break: break-word;
}

.mfg-rfq-dialog__tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}

.mfg-rfq-dialog__tabs :deep(.el-tabs__item) {
  height: 40px;
}

.mfg-rfq-dialog__tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.mfg-rfq-dialog__tab-label .el-icon {
  font-size: 15px;
}

.mfg-rfq-dialog__section-head {
  margin-bottom: 12px;
  padding: 14px 16px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: #fbfcfe;
}

.mfg-rfq-dialog__section-head h4 {
  margin: 0 0 4px;
  color: #1f2f46;
  font-size: 16px;
  letter-spacing: 0;
}

.mfg-rfq-dialog__section-head p {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.55;
}

.mfg-rfq-dialog__section-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.mfg-rfq-detail-groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.mfg-rfq-detail-group {
  padding: 15px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  box-shadow: 0 8px 20px rgb(31 47 70 / 5%);
}

.mfg-rfq-detail-group h5 {
  margin: 0 0 12px;
  padding-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: #1f2f46;
  font-size: 14px;
  letter-spacing: 0;
}

.mfg-rfq-detail-group h5 span {
  width: 4px;
  height: 14px;
  border-radius: 8px;
  background: #2f6fdb;
}

.mfg-rfq-detail-group dl {
  margin: 0;
  display: grid;
  grid-template-columns: minmax(96px, 0.32fr) minmax(0, 0.68fr);
  gap: 8px 12px;
}

.mfg-rfq-detail-group dt,
.mfg-rfq-detail-group dd {
  margin: 0;
  line-height: 1.55;
  word-break: break-word;
}

.mfg-rfq-detail-group dt {
  color: var(--el-text-color-secondary);
}

.mfg-rfq-detail-group dd {
  color: var(--el-text-color-regular);
  font-weight: 600;
}

.mfg-rfq-table {
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.mfg-rfq-table--full :deep(.el-table__inner-wrapper),
.mfg-rfq-table--full :deep(.el-table__body),
.mfg-rfq-table--full :deep(.el-table__header) {
  width: 100% !important;
}

.mfg-link-button {
  height: auto;
  padding: 0;
  font-weight: 700;
  white-space: normal;
}

.mfg-readonly-field {
  min-height: 32px;
  padding: 5px 11px;
  display: flex;
  align-items: center;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  line-height: 1.45;
  word-break: break-word;
}

.mfg-field-suffix {
  display: block;
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mfg-bom-detail {
  display: grid;
  gap: 12px;
}

.mfg-rfq-table :deep(.el-table__header th) {
  background: #f6f9fc;
  color: #55657a;
  font-weight: 700;
}

.mfg-rfq-table :deep(.el-table__row:hover > td) {
  background: #f8fbff !important;
}

.mfg-bom-current {
  margin-bottom: 12px;
  padding: 12px;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
  border: 1px solid #dce7f2;
  border-radius: 8px;
  background: #f8fbff;
}

.mfg-bom-current__item {
  min-height: 68px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.mfg-bom-current__item span,
.mfg-bom-current__item strong {
  display: block;
  line-height: 1.45;
  word-break: break-word;
}

.mfg-bom-current__item span {
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mfg-bom-current__item strong {
  color: #25364f;
  font-weight: 700;
}

.mfg-bom-history {
  margin-top: 10px;
}

.mfg-answer-meta {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.mfg-answer-meta strong {
  overflow: hidden;
  color: #27364a;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mfg-answer-meta span {
  color: #8492a6;
  font-size: 12px;
}

.mfg-answer-history {
  display: grid;
  gap: 12px;
}

.mfg-answer-history__head {
  padding: 12px;
  border: 1px solid #dce7f2;
  border-radius: 8px;
  background: #f8fbff;
}

.mfg-answer-history__head span {
  color: #60748c;
  font-size: 12px;
}

.mfg-answer-history__head strong {
  display: block;
  margin-top: 4px;
  color: #27364a;
  font-size: 14px;
}

.mfg-answer-history__head p,
.mfg-answer-history__change p {
  margin: 6px 0 0;
  color: #40536a;
  line-height: 1.5;
  white-space: pre-wrap;
}

.mfg-answer-history__change {
  display: grid;
  gap: 4px;
}

.mfg-answer-history__change span {
  color: #6c7f95;
  font-size: 12px;
  font-weight: 700;
}

.mfg-rfq-empty {
  min-height: 150px;
  border: 1px dashed #d7e2ee;
  border-radius: 8px;
  background: #fbfcfe;
}

.mfg-rfq-dialog__readiness {
  margin-top: 12px;
}

.mfg-section__actions {
  min-width: min(460px, 100%);
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.mfg-record-form .full-input {
  width: 100%;
}

.mfg-business-form {
  max-height: min(68vh, 720px);
  overflow: auto;
  padding-right: 4px;
}

.mfg-form-intro {
  margin-bottom: 14px;
  padding: 12px 14px;
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
  color: var(--el-text-color-regular);
  line-height: 1.6;
}

.mfg-record-title strong,
.mfg-record-title span {
  display: block;
}

.mfg-record-title span {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.mfg-grouped-workbench {
  display: grid;
  gap: 12px;
}

.mfg-grouped-workbench__bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.mfg-grouped-section {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.mfg-grouped-section__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.mfg-grouped-section__head span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mfg-rfq-form {
  max-height: min(68vh, 720px);
  overflow: auto;
  padding-right: 4px;
}

.mfg-form-block {
  margin-bottom: 16px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.mfg-form-block h4 {
  margin: 0 0 12px;
  font-size: 15px;
  letter-spacing: 0;
}

.mfg-form-block__head {
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.mfg-form-block__head h4 {
  margin: 0;
}

.mfg-context-card {
  margin-bottom: 14px;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.mfg-context-card > div {
  min-height: 72px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.mfg-context-card span,
.mfg-context-card strong {
  display: block;
}

.mfg-context-card span {
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mfg-context-card strong {
  font-size: 14px;
  line-height: 1.45;
}

.mfg-quote-detail {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.mfg-quote-detail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-extra-light);
}

.mfg-quote-detail__header span,
.mfg-quote-detail__metric span,
.mfg-quote-detail__grid span {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.mfg-quote-detail__header h3 {
  margin: 4px 0;
  font-size: 20px;
  line-height: 1.25;
  letter-spacing: 0;
  word-break: break-word;
}

.mfg-quote-detail__header p {
  margin: 0;
  color: var(--el-text-color-secondary);
}

.mfg-quote-detail__status {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.mfg-quote-detail__metrics,
.mfg-quote-detail__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.mfg-quote-detail__metric,
.mfg-quote-detail__grid > div {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.mfg-quote-detail__metric strong,
.mfg-quote-detail__grid strong {
  display: block;
  margin-top: 6px;
  font-size: 14px;
  line-height: 1.45;
  word-break: break-word;
}

.mfg-quote-detail__metric strong {
  font-size: 16px;
}

.mfg-quote-detail__section {
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.mfg-quote-detail__section h4 {
  margin: 0 0 10px;
  font-size: 15px;
  line-height: 1.4;
}

.mfg-quote-detail__terms {
  min-height: 72px;
  margin: 0;
  padding: 12px;
  border-radius: 6px;
  background: var(--el-fill-color-extra-light);
  color: var(--el-text-color-primary);
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}

.mfg-bom-editor :deep(.el-input__wrapper),
.mfg-bom-editor :deep(.el-input-number) {
  width: 100%;
}

.mfg-upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.mfg-upload-file {
  color: var(--el-text-color-secondary);
  word-break: break-word;
}

.mfg-rfq-clarification-editor {
  overflow-x: auto;
}

.mfg-rfq-clarification-toolbar p {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.mfg-rfq-clarification-table {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  overflow: hidden;
}

.mfg-rfq-clarification-table__head,
.mfg-rfq-clarification-row {
  display: grid;
  grid-template-columns: 44px minmax(150px, 0.9fr) minmax(220px, 1.2fr) 108px 112px;
  gap: 10px;
  align-items: center;
}

.mfg-rfq-clarification-table__head {
  padding: 9px 12px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.mfg-rfq-clarification-row {
  padding: 10px 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.mfg-rfq-clarification-row:hover {
  background: #f8fbff;
}

.mfg-rfq-clarification-cell {
  min-width: 0;
}

.mfg-rfq-clarification-cell > span {
  display: none;
}

.mfg-rfq-clarification-cell--index strong {
  display: inline-flex;
  width: 26px;
  height: 26px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #eef5ff;
  color: #1f2f46;
  font-size: 12px;
}

.mfg-rfq-clarification-cell--content :deep(.el-textarea__inner) {
  min-height: 34px !important;
  line-height: 1.45;
}

.mfg-rfq-clarification-cell--content {
  grid-column: 2 / -1;
}

.mfg-rfq-clarification-cell--content > span {
  display: block;
  margin-bottom: 5px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.2;
}

.mfg-rfq-clarification-cell--switch {
  display: flex;
  justify-content: center;
}

.mfg-rfq-clarification-cell--actions > div {
  display: flex;
  justify-content: flex-end;
  gap: 2px;
}

.mfg-rfq-clarification-cell--actions :deep(.el-button) {
  margin-left: 0;
}

.mfg-cost-workbench {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mfg-cost-workbench__guide {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  padding: 14px 16px;
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
}

.mfg-cost-workbench__guide strong {
  display: block;
  margin-bottom: 4px;
}

.mfg-cost-workbench__guide p,
.mfg-cost-next span {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.mfg-cost-next {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.mfg-cost-next strong {
  color: var(--el-text-color-primary);
}

.mfg-readiness-result {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.mfg-readiness-result__hero {
  padding: 18px;
  border: 1px solid var(--el-color-warning-light-6);
  border-radius: 8px;
  background: var(--el-color-warning-light-9);
}

.mfg-readiness-result.is-ready .mfg-readiness-result__hero {
  border-color: var(--el-color-success-light-6);
  background: var(--el-color-success-light-9);
}

.mfg-readiness-result__hero span {
  display: inline-flex;
  margin-bottom: 8px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-color-warning);
  font-size: 12px;
  font-weight: 700;
}

.mfg-readiness-result.is-ready .mfg-readiness-result__hero span {
  color: var(--el-color-success);
}

.mfg-readiness-result__hero strong {
  display: block;
  margin-bottom: 6px;
  font-size: 18px;
}

.mfg-readiness-result__hero p,
.mfg-readiness-result__missing small,
.mfg-readiness-result__next span {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.mfg-readiness-result__missing,
.mfg-readiness-result__next {
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.mfg-readiness-result__missing strong,
.mfg-readiness-result__next strong {
  display: block;
  margin-bottom: 8px;
}

.mfg-readiness-result__missing ul {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.mfg-readiness-result__missing li {
  padding: 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.mfg-readiness-result__missing li span,
.mfg-readiness-result__missing li small {
  display: block;
}

.mfg-readiness-result__missing li span {
  margin-bottom: 4px;
  font-weight: 700;
}

@media (max-width: 1080px) {
  .manufacturing-workspace__layout {
    grid-template-columns: 1fr;
  }

  .mfg-executive-dashboard,
  .mfg-context-card,
  .mfg-bom-current {
    grid-template-columns: 1fr;
  }

  .mfg-kpi-grid,
  .mfg-metric-group-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .mfg-rfq-dialog__hero,
  .mfg-rfq-dialog__section-head {
    display: block;
  }

  .mfg-rfq-dialog__hero-meta {
    margin-top: 12px;
    align-items: flex-start;
    text-align: left;
  }

  .mfg-rfq-dialog__summary,
  .mfg-rfq-detail-groups,
  .mfg-rfq-detail-group dl {
    grid-template-columns: 1fr;
  }

  .mfg-rfq-dialog__section-actions {
    margin-top: 12px;
    justify-content: flex-start;
  }

  .mfg-quote-detail__metrics,
  .mfg-quote-detail__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .mfg-rfq-clarification-editor {
    overflow-x: visible;
  }

  .mfg-rfq-clarification-toolbar {
    align-items: flex-start;
  }

  .mfg-rfq-clarification-table__head {
    display: none;
  }

  .mfg-rfq-clarification-row {
    grid-template-columns: 36px minmax(180px, 1fr) minmax(220px, 1fr);
    align-items: end;
  }

  .mfg-rfq-clarification-cell > span {
    display: block;
    margin-bottom: 5px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.2;
  }

  .mfg-rfq-clarification-cell--index > span {
    display: none;
  }

  .mfg-rfq-clarification-cell--content {
    grid-column: 2 / -1;
  }

  .mfg-rfq-clarification-cell--switch {
    justify-content: flex-start;
  }

  .mfg-rfq-clarification-cell--actions {
    grid-column: 2 / -1;
  }

  .mfg-rfq-clarification-cell--actions > div {
    justify-content: flex-end;
  }

  .mfg-section__toolbar,
  .mfg-section__actions {
    display: block;
  }

  .mfg-section__actions > * {
    margin-top: 8px;
    width: 100%;
  }
}

@media (max-width: 640px) {
  .mfg-quote-detail__header {
    flex-direction: column;
  }

  .mfg-quote-detail__status {
    justify-content: flex-start;
  }

  .mfg-quote-detail__metrics,
  .mfg-quote-detail__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .mfg-rfq-clarification-toolbar {
    display: block;
  }

  .mfg-rfq-clarification-toolbar .el-button {
    margin-top: 10px;
    width: 100%;
  }

  .mfg-rfq-clarification-row {
    grid-template-columns: 32px minmax(0, 1fr);
  }

  .mfg-rfq-clarification-cell:not(.mfg-rfq-clarification-cell--index) {
    grid-column: 2 / -1;
  }

  .mfg-kpi-grid,
  .mfg-business-metrics,
  .mfg-delivery-risk,
  .mfg-metric-group-grid,
  .mfg-metric-group-card__body {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
