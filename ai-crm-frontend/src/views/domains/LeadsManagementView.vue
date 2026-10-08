<template>
  <div class="system-page lead-page">
    <section class="lead-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="lead-analytics-grid">
      <div class="lead-chart-panel">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.32560779c8') }}</strong>
          <el-tag size="small">{{ t('domain.auto.bd1ab15a74') }}</el-tag>
        </div>
        <div class="lead-funnel-chart">
          <div v-for="item in dashboard.funnel" :key="item.status" class="lead-chart-row">
            <span>{{ statusLabel(item.status) }}</span>
            <div><i :style="{ width: chartWidth(item.count, maxFunnelCount) }"></i></div>
            <strong>{{ item.count }}</strong>
          </div>
        </div>
      </div>
      <div class="lead-chart-panel">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.a2858d06bc') }}</strong>
          <el-button v-if="canLeadUpdate" size="small" plain @click="runSLARecycle">{{ t('domain.auto.5d088f8c01') }}</el-button>
        </div>
        <div class="lead-source-chart">
          <div v-for="item in dashboard.sources" :key="item.source" class="lead-source-row">
            <div>
              <strong>{{ sourceLabel(item.source) }}</strong>
              <span>MQL {{ item.mql || 0 }} {{ t('domain.auto.d76bf84364') }} {{ item.converted || 0 }}</span>
            </div>
            <el-progress :percentage="Math.round(Number(item.conversion_rate || 0) * 100)" />
          </div>
        </div>
      </div>
      <div class="lead-chart-panel sdr-panel">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.3fa61eff2e') }}</strong>
          <el-button size="small" plain @click="openSDRWorkbench()">{{ t('domain.auto.ab4913f4bd') }}</el-button>
        </div>
        <div class="sdr-list">
          <button v-for="item in sdrRows" :key="item.id" class="sdr-item" @click="openSDRWorkbench(item)">
            <strong>{{ item.name }}</strong>
            <span>{{ payload(item).phone || payload(item).email || t('domain.auto.42968c046b') }}</span>
          </button>
        </div>
        <el-empty v-if="!sdrRows.length" :description="t('domain.auto.f02c471863')" />
      </div>
      <div class="lead-chart-panel supervisor-panel">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.6242baae6d') }}</strong>
          <el-button v-if="canLeadUpdate" size="small" plain @click="openAutomationDrawer">{{ t('domain.auto.a860d47b3e') }}</el-button>
        </div>
        <div class="lead-alert-grid">
          <div>
            <span>{{ t('domain.auto.b0b013b4bf') }}</span>
            <strong>{{ dashboard.alerts.mql_rejected || 0 }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.cfb12737ca') }}</span>
            <strong>{{ dashboard.alerts.overdue || 0 }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.88592f25a6') }}</span>
            <strong>{{ dashboard.kpi.converted || 0 }}</strong>
          </div>
        </div>
        <div class="lead-workload-list">
          <div v-for="item in dashboard.workload" :key="item.owner_id">
            <span>{{ item.name || ownerName(item.owner_id) }}</span>
            <el-progress :percentage="workloadPercent(item.open_leads)" />
          </div>
        </div>
      </div>
    </section>

    <section ref="tableCardRef" class="system-card" :class="{ 'system-card-fullscreen': isFullscreen }">
      <div class="list-card-search">
        <el-form :model="filters" label-width="96px" class="system-search-form lead-search-form">
          <el-form-item :label="t('domain.auto.e1272432a4')"><el-input v-model="filters.company_name" :placeholder="t('domain.auto.0792364719')" clearable /></el-form-item>
          <el-form-item :label="t('domain.auto.2425bd4bc1')"><el-input v-model="filters.contact_name" :placeholder="t('domain.auto.9feabe27d2')" clearable /></el-form-item>
          <el-form-item :label="t('domain.auto.8694f91905')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.165a5e3442')" clearable /></el-form-item>
          <template v-if="searchExpanded">
            <el-form-item :label="t('domain.auto.c63f79e636')">
              <el-select v-model="filters.source_channel" :placeholder="t('domain.auto.b187dd1433')" clearable>
                <el-option :label="t('domain.auto.0cec5c3a05')" value="manual" />
                <el-option :label="t('domain.auto.a2af045711')" value="website" />
                <el-option :label="t('domain.auto.5fa95b1b73')" value="campaign" />
                <el-option :label="t('domain.auto.3cb249b95e')" value="ads" />
                <el-option :label="t('domain.auto.29136ed9a0')" value="social" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')">
              <el-select v-model="filters.status" :placeholder="t('domain.auto.dba277df58')" clearable>
                <el-option v-for="item in leadStatuses" :key="item.value" :label="statusLabel(item.value)" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.974d383f36')">
              <el-select v-model="filters.owner_id" class="full-input" clearable filterable :placeholder="t('domain.auto.8608d655e6')">
                <el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" />
              </el-select>
            </el-form-item>
          </template>
          <div class="system-search-actions">
            <el-button @click="showAllLeads">{{ t('domain.auto.3d81345303') }}</el-button>
            <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
            <el-button link type="primary" @click="searchExpanded = !searchExpanded">{{ searchExpanded ? t('domain.auto.5d5815647c') : t('domain.auto.b0e24833f7') }}</el-button>
          </div>
        </el-form>
      </div>
      <div class="lead-view-strip lead-view-strip-in-card">
        <div class="lead-view-group">
          <el-tooltip :content="t('domain.auto.c620afaee0')">
            <button :class="{ active: !activeView && !hasActiveLeadFilters }" @click="showAllLeads">
              {{ t('domain.auto.f6631984e4') }}
            </button>
          </el-tooltip>
          <el-tooltip v-for="view in leadViews" :key="view.key" :content="leadViewHelp(view)" placement="top">
            <button :class="{ active: activeView === view.key }" @click="applyView(view)">
              {{ leadViewName(view) }}
            </button>
          </el-tooltip>
        </div>
        <div class="lead-view-group archive">
          <span>{{ t('domain.auto.0b82ceff7b') }}</span>
          <el-tooltip :content="t('domain.auto.60eb4e665b')">
            <button :class="{ active: activeView === 'converted' }" @click="applyConvertedView">
              {{ t('domain.auto.24aef3c29e') }}
            </button>
          </el-tooltip>
          <el-tooltip :content="t('domain.auto.8683382646')">
            <button :class="{ active: activeView === 'invalid' }" @click="applyInvalidView">
              {{ t('domain.auto.93736531ce') }}
            </button>
          </el-tooltip>
        </div>
      </div>
      <div class="system-toolbar">
        <div class="toolbar-left">
          <el-button v-if="canLeadCreate" type="primary" plain @click="openCreate">{{ t('domain.auto.47a1b14404') }}</el-button>
          <el-button v-if="canLeadCreate" @click="openImport">{{ t('domain.auto.14b1962ec3') }}</el-button>
          <el-button v-if="canLeadExport" @click="exportLeads">{{ t('domain.auto.188896795f') }}</el-button>
          <el-button v-if="canLeadUpdate" @click="openNurtureTemplate">{{ t('domain.auto.bdf1d9f6e7') }}</el-button>
          <el-button v-if="canLeadUpdate" @click="openAutomationDrawer">{{ t('domain.auto.a860d47b3e') }}</el-button>
          <el-button v-if="canLeadUpdate" @click="openABMDrawer">{{ t('domain.auto.ddad7664f9') }}</el-button>
          <el-button @click="openSDRWorkbench()">{{ t('domain.auto.4bd9f7445c') }}</el-button>
          <el-button @click="applyConvertedView">{{ t('domain.auto.24aef3c29e') }}</el-button>
          <el-button @click="termsDrawer = true">{{ t('domain.auto.e029dc83f8') }}</el-button>
          <el-segmented v-model="viewMode" :options="viewModeOptions" size="small" />
        </div>
        <div class="toolbar-icons">
          <el-tooltip :content="t('domain.auto.38108eaa1d')"><button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="refreshAll"><Icon icon="ri:refresh-line" /></button></el-tooltip>
          <el-tooltip :content="t('domain.auto.93c44f6b1b')"><button class="system-icon-btn" :aria-label="t('domain.auto.93c44f6b1b')" @click="isFullscreen = !isFullscreen"><Icon :icon="isFullscreen ? 'ri:fullscreen-exit-line' : 'ri:fullscreen-line'" /></button></el-tooltip>
          <el-tooltip :content="t('domain.auto.56292a48ef')"><button class="system-icon-btn" :aria-label="t('domain.auto.56292a48ef')" @click="columnDrawer = true"><Icon icon="ri:settings-3-line" /></button></el-tooltip>
        </div>
      </div>

      <el-table v-if="viewMode === 'table'" :data="rows" class="system-table" row-key="id" v-loading="loading" @selection-change="selectedRows = $event">
        <el-table-column type="selection" width="42" />
        <el-table-column v-if="visibleColumn('name')" prop="name" :label="t('domain.auto.733eabcea7')" min-width="180">
          <template #default="{ row }">
            <div class="record-title">
              <strong>{{ row.name }} <el-tag v-if="isABMLead(row)" size="small" type="danger">ABM</el-tag> <el-tag v-if="objectVisibility(row) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
              <span>{{ payload(row).company_name || payload(row).company || row.code || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column v-if="visibleColumn('contact')" :label="t('domain.auto.2425bd4bc1')" min-width="150"><template #default="{ row }">{{ payload(row).contact_name || payload(row).contact || '-' }}</template></el-table-column>
        <el-table-column v-if="visibleColumn('phone')" :label="t('domain.auto.8694f91905')" min-width="180"><template #default="{ row }">{{ payload(row).phone || payload(row).email || '-' }}</template></el-table-column>
        <el-table-column v-if="visibleColumn('source')" :label="t('domain.auto.c63f79e636')" width="120"><template #default="{ row }">{{ sourceLabel(payload(row).source_channel || payload(row).utm_source) }}</template></el-table-column>
        <el-table-column v-if="visibleColumn('campaign')" :label="t('domain.auto.bb002ebde7')" min-width="150"><template #default="{ row }">{{ payload(row).campaign_name || '-' }}</template></el-table-column>
        <el-table-column v-if="visibleColumn('status')" :label="t('domain.auto.62e951a692')" width="130"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="light">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column v-if="visibleColumn('temperature')" :label="t('domain.auto.296adf9512')" width="120"><template #default="{ row }"><span :class="['temperature-dot', temperature(row)]"></span>{{ temperatureLabel(temperature(row)) }}</template></el-table-column>
        <el-table-column v-if="visibleColumn('score')" :label="t('domain.auto.2f58566ce0')" width="130"><template #default="{ row }"><el-progress :percentage="Number(row.score || payload(row).ai_score || 0)" :show-text="false" /><span class="score-text">{{ row.score || payload(row).ai_score || 0 }}</span></template></el-table-column>
        <el-table-column v-if="visibleColumn('owner')" :label="t('domain.auto.974d383f36')" width="120"><template #default="{ row }">{{ ownerName(row.owner_id) }}</template></el-table-column>
        <el-table-column v-if="visibleColumn('process')" :label="t('domain.auto.2612f9775a')" min-width="180">
          <template #default="{ row }">
            <div class="lead-process-cell">
              <el-tag :type="processStatus(row).type" effect="light">{{ processStatus(row).label }}</el-tag>
              <span>{{ processStatus(row).desc }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column v-if="visibleColumn('updated_at')" :label="t('domain.auto.093dea88c9')" min-width="150"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.f3ea6d345e')" width="238" fixed="right">
          <template #default="{ row }">
            <div class="row-actions lead-row-actions" @click.stop>
              <el-tooltip :content="t('domain.auto.f7acefd2d4')"><button class="action-btn view" @click="openDetail(row)"><Icon icon="ri:eye-line" /></button></el-tooltip>
              <template v-if="row.status === 'converted'">
                <el-tooltip :content="t('domain.auto.c0db6ce03d')"><button class="action-btn add" @click="goOpportunity(row)"><Icon icon="ri:briefcase-line" /></button></el-tooltip>
                <el-tooltip :content="t('domain.auto.b9d1f52699')"><button class="action-btn" disabled><Icon icon="ri:lock-line" /></button></el-tooltip>
              </template>
              <template v-else>
              <el-tooltip v-if="canLeadUpdate" :content="t('domain.auto.a7f814c0a4')"><button class="action-btn edit" :disabled="isTerminal(row)" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button></el-tooltip>
              <el-tooltip v-if="canLeadAssign" :content="t('domain.auto.bdddf08727')"><button class="action-btn add" :disabled="isTerminal(row)" @click="openAssign(row)"><Icon icon="ri:user-add-line" /></button></el-tooltip>
              <el-tooltip v-if="canLeadUpdate" :content="t('domain.auto.2f58566ce0')"><button class="action-btn add" :disabled="isTerminal(row)" @click="scoreLead(row)"><Icon icon="ri:sparkling-line" /></button></el-tooltip>
              <el-tooltip v-if="canLeadUpdate" :content="t('domain.auto.b6c8bdd841')"><button class="action-btn add" :disabled="isTerminal(row)" @click="openConvert(row)"><Icon icon="ri:exchange-dollar-line" /></button></el-tooltip>
              <el-tooltip v-if="canLeadUpdate" :content="t('domain.auto.feeb19ac37')"><button class="action-btn danger" :disabled="isTerminal(row)" @click="openStatus(row, 'invalid')"><Icon icon="ri:close-circle-line" /></button></el-tooltip>
              <el-tooltip v-if="canLeadDelete" :content="t('domain.auto.119cc6fd6a')"><button class="action-btn danger" :disabled="row.status !== 'invalid'" @click="deleteInvalidLead(row)"><Icon icon="ri:delete-bin-line" /></button></el-tooltip>
              </template>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-else-if="viewMode === 'kanban'" class="lead-kanban-board" v-loading="loading">
        <div v-for="column in kanbanColumns" :key="column.status" :class="['lead-kanban-column', `lead-kanban-${column.status}`]">
          <div class="kanban-head">
            <strong>{{ statusLabel(column.status) }}</strong>
            <el-tag size="small">{{ column.items.length }}</el-tag>
          </div>
          <div class="lead-kanban-list">
            <div v-for="item in column.items" :key="item.id" class="kanban-card lead-kanban-card-static">
              <strong>{{ item.name }} <el-tag v-if="objectVisibility(item) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
              <span>{{ payload(item).company_name || payload(item).company || '-' }}</span>
              <span>{{ payload(item).phone || payload(item).email || t('domain.auto.42968c046b') }}</span>
              <el-progress :percentage="Number(item.score || 0)" :show-text="false" />
              <div class="kanban-card-actions">
                <el-tooltip :content="t('domain.auto.f7acefd2d4')"><button class="action-btn view" @click="openDetail(item)"><Icon icon="ri:eye-line" /></button></el-tooltip>
              </div>
            </div>
            <el-empty v-if="!column.items.length" :description="t('domain.auto.add07537e2')" />
          </div>
        </div>
      </div>

      <div v-else class="lead-customer-space" v-loading="loading">
        <div v-for="item in customerSpaceRows" :key="item.id" :class="['space-card', `space-card-${item.status}`]">
          <div class="space-card-head">
            <span class="space-card-avatar">{{ leadAvatarText(item) }}</span>
            <div>
              <strong>{{ payload(item).company_name || payload(item).company || item.name }}</strong>
              <span>{{ item.name }}</span>
            </div>
            <div class="space-card-tags">
              <el-tag v-if="objectVisibility(item) === 'private'" type="warning" size="small">{{ t('domain.auto.c931219c30') }}</el-tag>
              <el-tag :type="statusType(item.status)" size="small">{{ statusLabel(item.status) }}</el-tag>
            </div>
          </div>
          <div class="space-meta">
            <div><span>{{ t('domain.auto.2425bd4bc1') }}</span><strong>{{ payload(item).contact_name || payload(item).contact || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.8694f91905') }}</span><strong>{{ payload(item).phone || payload(item).email || t('domain.auto.42968c046b') }}</strong></div>
            <div><span>{{ t('domain.auto.c63f79e636') }}</span><strong>{{ sourceLabel(payload(item).source_channel || payload(item).utm_source) }}</strong></div>
            <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(item.owner_id) }}</strong></div>
          </div>
          <p>{{ t('domain.auto.66f769ac84') }}{{ leadIntentProductText(payload(item)) }}</p>
          <div class="space-actions">
            <template v-if="item.status === 'converted'">
              <el-button size="small" type="primary" plain @click="goOpportunity(item)">{{ t('domain.auto.b4f5c14081') }}</el-button>
            </template>
            <template v-else>
              <el-button v-if="canLeadUpdate" size="small" @click="openFollowUp(item)">{{ t('domain.auto.0a96a4a257') }}</el-button>
              <el-button v-if="canLeadAssign" size="small" @click="openAssign(item)">{{ t('domain.auto.bdddf08727') }}</el-button>
              <el-button v-if="canLeadUpdate" size="small" type="primary" @click="openConvert(item)">{{ t('domain.auto.b6c8bdd841') }}</el-button>
            </template>
          </div>
        </div>
        <el-empty v-if="!customerSpaceRows.length" :description="t('domain.auto.ff7a2588d2')" />
      </div>

      <div class="system-pagination">
        <span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span>
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
      </div>
    </section>

    <el-drawer v-model="detailDrawer" size="46%" :title="detailRecord?.name || t('domain.auto.d8e33fc628')" destroy-on-close>
      <div v-if="detailRecord" class="lead-detail">
        <el-alert
          v-if="isConvertedDetail"
          type="success"
          show-icon
          :closable="false"
          :title="t('domain.auto.cac76f4510')"
        />
        <el-alert
          v-else-if="isInvalidDetail"
          type="error"
          show-icon
          :closable="false"
          :title="t('domain.auto.e57aa59721')"
        />
        <el-alert v-if="dedupePossibleDuplicates.length" type="warning" show-icon :closable="false" :title="t('domain.auto.377dc5027b')" />
        <el-alert v-if="dedupe.contacts.length" type="info" show-icon :closable="false" :title="t('domain.auto.fdbee2f2ef')" />
        <el-alert v-if="dedupe.companies.length" type="info" show-icon :closable="false" :title="t('domain.auto.2f822ac3ea')" />
        <div v-if="dedupeHistoryRows.length" class="dedupe-history-panel">
          <div class="dedupe-history-head">
            <strong>{{ t('domain.auto.9a26caa85c') }}</strong>
            <span>{{ t('domain.auto.f2b62fe6ab') }}</span>
          </div>
          <div class="dedupe-history-list">
            <div v-for="item in dedupeHistoryRows.slice(0, 6)" :key="item.id" class="dedupe-history-item">
              <strong>{{ item.name }}</strong>
              <span>{{ statusLabel(item.status) }} · {{ sourceLabel(payload(item).source_channel || payload(item).utm_source) }} · {{ formatTime(item.created_at) }}</span>
            </div>
          </div>
        </div>
        <div class="lead-summary">
          <div>
            <span>{{ t('domain.auto.5d80806d7f') }}</span>
            <strong>{{ detailPayload.company_name || detailPayload.company || '-' }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.2425bd4bc1') }}</span>
            <strong>{{ detailPayload.contact_name || detailPayload.contact || '-' }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.8694f91905') }}</span>
            <strong>{{ detailPayload.phone || detailPayload.email || '-' }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.bcb5539330') }}</span>
            <strong>{{ leadIntentProductText(detailPayload) }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.296adf9512') }}</span>
            <strong><span :class="['temperature-dot', temperature(detailRecord)]"></span>{{ temperatureLabel(temperature(detailRecord)) }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.974d383f36') }}</span>
            <strong>{{ ownerName(detailRecord.owner_id) }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.8af32a9138') }}</span>
            <strong>{{ isABMLead(detailRecord) ? (payload(detailRecord).abm_target_account_name || payload(detailRecord).company_name || payload(detailRecord).company) : t('domain.auto.8bf5c10ad9') }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.4b411289ae') }}</span>
            <strong>{{ mqlRejectedText(detailRecord) }}</strong>
          </div>
          <div>
            <span>{{ t('domain.auto.f5324df3aa') }}</span>
            <strong>{{ nurtureSourceText(detailRecord) }}</strong>
          </div>
          <div v-if="isConvertedDetail">
            <span>{{ t('domain.auto.0b77721e7a') }}</span>
            <strong>{{ convertedOpportunityID ? `#${convertedOpportunityID}` : t('domain.auto.8103ef10a3') }}</strong>
          </div>
        </div>
        <ObjectAccessPanel
          domain="leads"
          :record="detailRecord"
          :users="users"
          :can-manage="canLeadUpdate"
          :object-label="t('domain.auto.8de0b8fb1d')"
          @updated="refreshDetailRealtime"
        />
        <div v-if="isConvertedDetail" class="detail-actions archived-actions">
          <el-button type="primary" @click="goOpportunity(detailRecord)">{{ t('domain.auto.b4f5c14081') }}</el-button>
          <el-button @click="goContactObject('company')">{{ t('domain.auto.c30b387770') }}</el-button>
          <el-button @click="goContactObject('contact')">{{ t('domain.auto.de469d9b95') }}</el-button>
          <el-button @click="scrollDealChain">{{ t('domain.auto.5440bacc4a') }}</el-button>
          <el-button v-if="canLeadUpdate" plain @click="openFollowUp(detailRecord)">{{ t('domain.auto.b82157bf5f') }}</el-button>
        </div>
        <div v-else-if="isInvalidDetail" class="detail-actions archived-actions">
          <el-button v-if="canLeadUpdate" plain @click="openFollowUp(detailRecord)">{{ t('domain.auto.b82157bf5f') }}</el-button>
          <el-button v-if="canLeadDelete" type="danger" @click="deleteInvalidLead(detailRecord)">{{ t('domain.auto.119cc6fd6a') }}</el-button>
        </div>
        <div v-else class="detail-actions detail-action-groups">
          <div class="detail-action-group">
            <span>{{ t('domain.auto.b013d6a4cc') }}</span>
            <el-button v-if="canLeadUpdate" @click="openFollowUp(detailRecord)">{{ t('domain.auto.a68cc3cf80') }}</el-button>
            <el-button v-if="canLeadUpdate" @click="openCommunicationAction(detailRecord, 'whatsapp')">WhatsApp</el-button>
            <el-button v-if="canLeadUpdate" @click="openCommunicationAction(detailRecord, 'line')">Line</el-button>
          </div>
          <div class="detail-action-group">
            <span>{{ t('domain.auto.4153383eb9') }}</span>
            <el-button v-if="canLeadAssign" @click="openAssign(detailRecord)">{{ t('domain.auto.bdddf08727') }}</el-button>
            <el-button v-if="canLeadUpdate" @click="releaseLead(detailRecord)">{{ t('domain.auto.9e72d69574') }}</el-button>
            <el-button v-if="canLeadUpdate" @click="openNurtureAction(detailRecord)">{{ t('domain.auto.5f40216260') }}</el-button>
            <el-button v-if="canLeadUpdate" type="primary" @click="openConvert(detailRecord)">{{ t('domain.auto.b6c8bdd841') }}</el-button>
          </div>
          <div class="detail-action-group danger">
            <span>{{ t('domain.auto.bf79cedb71') }}</span>
            <el-button v-if="canLeadUpdate" type="danger" plain @click="openStatus(detailRecord, 'invalid')">{{ t('domain.auto.feeb19ac37') }}</el-button>
            <el-button v-if="canLeadUpdate" plain @click="mergeLead(detailRecord)">{{ t('domain.auto.e5badf90e7') }}</el-button>
          </div>
        </div>
        <DealChainPanel
          v-if="isConvertedDetail"
          id="lead-deal-chain"
          object-type="leads"
          :object-id="detailRecord.id"
        />
        <el-divider>{{ t('domain.auto.4a3aec00b1') }}</el-divider>
        <el-timeline>
          <el-timeline-item v-for="item in timeline" :key="item.id" :timestamp="formatTime(item.created_at)">
            <strong>{{ timelineTitle(item) }}</strong>
            <small class="timeline-operator">{{ timelineOperator(item) }}</small>
            <p>{{ timelineSummary(item) }}</p>
            <p v-if="timelineDetail(item)" class="timeline-detail">{{ timelineDetail(item) }}</p>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-if="!timeline.length" :description="t('domain.auto.96b919608c')" />
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.a7e4341418') : t('domain.auto.47a1b14404')" width="min(920px, 96vw)" destroy-on-close>
      <el-form ref="leadFormRef" :model="form" :rules="leadFormRules" label-width="124px" class="lead-dialog-form lead-create-form">
        <el-form-item :label="t('domain.auto.e1272432a4')" required>
          <el-select
            v-model="form.payload.company_name"
            class="full-input"
            allow-create
            clearable
            filterable
            :placeholder="t('domain.auto.42379aae38')"
            @change="handleLeadCompanyChange"
            @blur="checkDedupeForm"
          >
            <el-option v-for="item in companyOptions" :key="item.id" :label="companyOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.2425bd4bc1')" required>
          <el-select
            v-model="form.payload.contact_name"
            class="full-input"
            allow-create
            clearable
            filterable
            :placeholder="t('domain.auto.56a1a3ddad')"
            @change="handleLeadContactChange"
            @blur="checkDedupeForm"
          >
            <el-option v-for="item in contactOptions" :key="item.id" :label="contactOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.9c01ad0911')" required><el-input v-model="form.payload.phone" :placeholder="t('domain.auto.11d96957da')" @blur="checkDedupeForm" /></el-form-item>
        <el-form-item :label="t('domain.auto.9ed627bcf6')" required><el-input v-model="form.payload.email" :placeholder="t('domain.auto.11d96957da')" @blur="checkDedupeForm" /></el-form-item>
        <el-form-item :label="t('domain.auto.c63f79e636')" required><el-select v-model="form.payload.source_channel" @change="handleLeadSourceChange"><el-option :label="t('domain.auto.0cec5c3a05')" value="manual" /><el-option :label="t('domain.auto.a2af045711')" value="website" /><el-option :label="t('domain.auto.5fa95b1b73')" value="campaign" /><el-option :label="t('domain.auto.3cb249b95e')" value="ads" /><el-option :label="t('domain.auto.29136ed9a0')" value="social" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.bb002ebde7')">
          <el-select v-model="form.payload.campaign_id" class="full-input" filterable clearable :placeholder="t('domain.auto.4e46f3837a')" @change="handleLeadCampaignChange">
            <el-option v-for="item in campaignOptions" :key="item.id" :label="campaignOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="form.status"><el-option v-for="item in leadStatuses" :key="item.value" :label="statusLabel(item.value)" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.bcb5539330')" required>
          <el-select
            v-model="form.payload.intent_product_ids"
            class="full-input"
            clearable
            filterable
            multiple
            collapse-tags
            :placeholder="t('domain.auto.f7c628f8e6')"
            @change="handleLeadProductChange"
          >
            <el-option v-for="item in productOptions" :key="item.id" :label="productOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.0ae5a50777')">
          <el-select
            v-model="form.payload.utm_source"
            class="full-input"
            allow-create
            clearable
            default-first-option
            filterable
            :placeholder="t('domain.auto.dc88e46f48')"
          >
            <el-option-group v-for="group in utmOptionGroups" :key="group.label" :label="group.label">
              <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.974d383f36')" required>
          <el-select v-model="form.owner_id" class="full-input" clearable filterable :placeholder="t('domain.auto.8608d655e6')">
            <el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.241d8ef92f')"><ObjectAccessField v-model="form.payload.visibility" :public-label="t('domain.auto.665c246641')" :private-label="t('domain.auto.4bc865ce1c')" /></el-form-item>
        <el-form-item :label="t('domain.auto.e0361480e3')" class="wide-field"><el-input v-model="form.payload.note" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <div v-if="formDedupeState.hasAny" class="lead-dedupe-preview inline-action">
        <el-alert v-if="formDedupeState.possibleCount" :title="formDedupeState.warningText" type="warning" show-icon :closable="false" />
        <el-alert v-if="formDedupeState.contactCount" :title="formDedupeState.contactText" type="info" show-icon :closable="false" />
        <el-alert v-if="formDedupeState.companyCount" :title="formDedupeState.companyText" type="info" show-icon :closable="false" />
        <div v-if="formDedupeState.historyCount" class="dedupe-history-note">
          {{ t('domain.auto.9a26caa85c') }} {{ formDedupeState.historyCount }} {{ t('domain.auto.34fe8ff079') }}
        </div>
      </div>
      <template #footer>
        <el-button @click="formDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ formDedupeState.possibleCount ? t('domain.auto.475a227c38') : t('domain.auto.09cbc97ae2') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="actionDialog" :title="actionTitle" width="520px" destroy-on-close>
      <el-form label-width="92px">
        <template v-if="actionMode === 'assign'">
          <el-form-item :label="t('domain.auto.974d383f36')">
            <el-select v-model="actionForm.owner_id" class="full-input" filterable :placeholder="t('domain.auto.8608d655e6')">
              <el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('domain.auto.1ff9c3d001')"><el-input v-model="actionForm.reason" /></el-form-item>
        </template>
        <template v-else-if="actionMode === 'convert'">
          <el-form-item :label="t('domain.auto.e1272432a4')"><span class="form-static-text">{{ actionForm.company_name || '-' }}</span></el-form-item>
          <el-form-item :label="t('domain.auto.2425bd4bc1')"><span class="form-static-text">{{ actionForm.contact_name || '-' }}</span></el-form-item>
          <el-form-item :label="t('domain.auto.84b592481d')"><el-input v-model="actionForm.deal_name" /></el-form-item>
          <el-form-item :label="t('domain.auto.08c9f9e611')">
            <div class="amount-with-currency">
              <el-input-number v-model="actionForm.amount" :min="0" class="full-input" />
              <el-select v-model="actionForm.currency" class="currency-select" filterable :placeholder="t('domain.auto.a81ab5e100')">
                <el-option v-for="item in currencyOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item :label="t('domain.auto.ca3c9e8022')"><el-input v-model="actionForm.reason" type="textarea" :rows="2" :placeholder="t('domain.auto.ed4849d228')" /></el-form-item>
        </template>
        <template v-else-if="actionMode === 'follow'">
          <el-form-item :label="t('domain.auto.e4e46c7235')"><el-select v-model="actionForm.type"><el-option :label="t('domain.auto.1c8e464184')" value="email" /><el-option :label="t('domain.auto.1f3cb01e46')" value="social" /><el-option :label="t('domain.auto.e0361480e3')" value="note" /></el-select></el-form-item>
          <el-form-item v-if="actionForm.channel_label" :label="t('domain.auto.c152be9f50')"><span class="form-static-text">{{ actionForm.channel_label }}</span></el-form-item>
          <el-form-item :label="t('domain.auto.163aec9194')"><el-input v-model="actionForm.content" type="textarea" :rows="4" /></el-form-item>
          <el-form-item :label="t('domain.auto.ea0ef2ae72')"><el-input v-model="actionForm.next_action" /></el-form-item>
        </template>
        <template v-else-if="actionMode === 'nurture'">
          <el-form-item :label="t('domain.auto.a51208be47')">
            <el-select v-model="actionForm.template_key" class="full-input">
              <el-option v-for="item in nurtureTemplates" :key="item.key" :label="item.name" :value="item.key" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('domain.auto.ca3c9e8022')"><el-input v-model="actionForm.reason" type="textarea" :rows="3" :placeholder="t('domain.auto.0466d5b0e7')" /></el-form-item>
        </template>
        <template v-else>
          <el-form-item :label="t('domain.auto.1ff9c3d001')"><el-input v-model="actionForm.reason" type="textarea" :rows="3" /></el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="actionDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitAction">{{ t('domain.auto.f526c89937') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="scoreDialog" :title="t('domain.auto.140f0454e5')" width="460px" destroy-on-close>
      <el-form label-width="92px">
        <el-form-item :label="t('domain.auto.da335e204f')">
          <el-slider v-model="scoreForm.score" :min="0" :max="100" show-input />
        </el-form-item>
        <el-form-item :label="t('domain.auto.296adf9512')">
          <el-tag :type="scoreForm.score >= 75 ? 'danger' : scoreForm.score >= 45 ? 'warning' : 'success'">
            {{ scoreToTemperatureLabel(scoreForm.score) }}
          </el-tag>
        </el-form-item>
        <el-form-item :label="t('domain.auto.26670dda42')">
          <el-input v-model="scoreForm.reason" type="textarea" :rows="3" :placeholder="t('domain.auto.1171646247')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="scoreDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitScore">{{ t('domain.auto.dcf369e537') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="columnDrawer" :title="t('domain.auto.56292a48ef')" size="320px">
      <div class="column-settings">
        <div class="column-settings-head">
          <el-button link type="primary" @click="columnSelections = columnOptions.map((item) => item.key)">{{ t('domain.auto.3e44b2a933') }}</el-button>
          <el-button link type="primary" @click="resetColumns">{{ t('domain.auto.a191935bc6') }}</el-button>
        </div>
        <el-checkbox-group v-model="columnSelections" class="column-checkboxes" @change="saveColumns">
          <el-checkbox v-for="item in columnOptions" :key="item.key" :label="item.key">{{ item.label }}</el-checkbox>
        </el-checkbox-group>
      </div>
    </el-drawer>

    <el-dialog v-model="importDialog" :title="t('domain.auto.1ed941ae49')" width="980px">
      <el-alert :title="t('domain.auto.6cc7e57b4d')" type="info" show-icon :closable="false" class="inline-action" />
      <div class="import-file-bar">
        <input ref="importFileInput" type="file" accept=".csv,.tsv,.txt,.json,.xls,.xlsx,text/csv,text/tab-separated-values,application/json" @change="handleImportFile" />
        <span>{{ importFileName || t('domain.auto.83a223f6b7') }}</span>
      </div>
      <div class="import-mapping-grid">
        <el-form :model="importMapping" label-width="84px" class="lead-dialog-form">
          <el-form-item v-for="field in importFieldOptions" :key="field.key" :label="field.label">
            <el-select v-model="importMapping[field.key]" clearable filterable :placeholder="t('domain.auto.d668ebb9fe')">
              <el-option v-for="header in importHeaders" :key="header" :label="header" :value="header" />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="import-preview-summary">
          <div><span>{{ t('domain.auto.81e30fb20a') }}</span><strong>{{ importPreviewRows.length }}</strong></div>
          <div><span>{{ t('domain.auto.ad385d382a') }}</span><strong>{{ importValidRows.length }}</strong></div>
          <div><span>{{ t('domain.auto.5caf279339') }}</span><strong>{{ importInvalidRows.length }}</strong></div>
        </div>
      </div>
      <el-table v-if="importPreviewRows.length" :data="importPreviewRows" class="system-table import-preview-table" height="220">
        <el-table-column :label="t('domain.auto.2a2b777ae4')" prop="line" width="62" />
        <el-table-column :label="t('domain.auto.5d80806d7f')" prop="company_name" min-width="130" />
        <el-table-column :label="t('domain.auto.2425bd4bc1')" prop="contact_name" min-width="120" />
        <el-table-column :label="t('domain.auto.8694f91905')" min-width="170"><template #default="{ row }">{{ row.phone || row.email || '-' }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.62e951a692')" width="100"><template #default="{ row }"><el-tag :type="row.valid ? 'success' : 'danger'">{{ row.valid ? t('domain.auto.414db0b4c1') : t('domain.auto.5caf279339') }}</el-tag></template></el-table-column>
        <el-table-column :label="t('domain.auto.26670dda42')" min-width="180"><template #default="{ row }">{{ row.errors.join('、') || t('domain.auto.7035f86dd0') }}</template></el-table-column>
      </el-table>
      <el-input v-model="importText" type="textarea" :rows="12" />
      <template #footer>
        <el-button @click="importDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button @click="parseImportPreview">{{ t('domain.auto.8652f5287b') }}</el-button>
        <el-button type="primary" @click="importLeads">{{ t('domain.auto.47a59ad3f7') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="sdrDrawer" :title="t('domain.auto.4bd9f7445c')" size="72%" destroy-on-close>
      <div class="feature-guide">
        <strong>{{ t('domain.auto.ab1c927714') }}</strong>
        <p>{{ t('domain.auto.9f565f4d0a') }}</p>
      </div>
      <div class="sdr-workbench" tabindex="0" @keydown="handleSDRKeydown">
        <div class="sdr-queue">
          <button v-for="item in sdrRows" :key="item.id" :class="{ active: sdrActiveLead?.id === item.id }" @click="selectSDRLead(item)">
            <strong>{{ item.name }}</strong>
            <span>{{ statusLabel(item.status) }} · {{ payload(item).phone || payload(item).email || t('domain.auto.42968c046b') }}</span>
          </button>
          <el-empty v-if="!sdrRows.length" :description="t('domain.auto.985015bbe1')" />
        </div>
        <div v-if="sdrActiveLead" class="sdr-detail-panel">
          <div class="lead-summary">
            <div><span>{{ t('domain.auto.5d80806d7f') }}</span><strong>{{ payload(sdrActiveLead).company_name || payload(sdrActiveLead).company || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.2425bd4bc1') }}</span><strong>{{ payload(sdrActiveLead).contact_name || payload(sdrActiveLead).contact || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.9c01ad0911') }}</span><strong>{{ payload(sdrActiveLead).phone || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.9ed627bcf6') }}</span><strong>{{ payload(sdrActiveLead).email || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.296adf9512') }}</span><strong><span :class="['temperature-dot', temperature(sdrActiveLead)]"></span>{{ temperatureLabel(temperature(sdrActiveLead)) }}</strong></div>
            <div><span>ABM</span><strong>{{ isABMLead(sdrActiveLead) ? t('domain.auto.7a130d7fde') : t('domain.auto.786d17cf29') }}</strong></div>
          </div>
          <div class="detail-actions">
            <el-button v-if="canLeadUpdate" @click="logLeadCommunication(sdrActiveLead, 'whatsapp')">WhatsApp</el-button>
            <el-button v-if="canLeadUpdate" @click="logLeadCommunication(sdrActiveLead, 'line')">Line</el-button>
            <el-button v-if="canLeadUpdate" @click="sdrMarkSQL">{{ t('domain.auto.a2f3c3c599') }}</el-button>
            <el-button v-if="canLeadUpdate" type="warning" plain @click="sdrRejectToNurture">{{ t('domain.auto.097528ded6') }}</el-button>
            <el-button v-if="canLeadUpdate" type="primary" plain @click="openConvert(sdrActiveLead)">{{ t('domain.auto.b6c8bdd841') }}</el-button>
          </div>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="abmDrawer" :title="t('domain.auto.f01f9a010a')" size="420px">
      <div class="abm-panel">
        <div class="feature-guide">
          <strong>{{ t('domain.auto.a5fd053ffd') }}</strong>
          <p>{{ t('domain.auto.07f56a7811') }}</p>
        </div>
        <el-input v-model="abmText" type="textarea" :rows="10" :placeholder="t('domain.auto.365671107e')" />
        <div class="automation-actions">
          <el-button @click="matchABMRows">{{ t('domain.auto.68e535c941') }}</el-button>
          <el-button type="primary" @click="saveABMTargets">{{ t('domain.auto.4de3377edd') }}</el-button>
        </div>
        <div class="abm-match-list">
          <div v-for="item in abmMatchedRows" :key="item.id">
            <strong>{{ item.name }}</strong>
            <span>{{ payload(item).company_name || payload(item).company || '-' }}</span>
          </div>
        </div>
        <el-empty v-if="!abmMatchedRows.length" :description="t('domain.auto.0cd3cb71a5')" />
      </div>
    </el-drawer>

    <el-dialog v-model="nurtureDialog" :title="t('domain.auto.bdf1d9f6e7')" width="920px">
      <div class="feature-guide">
        <strong>{{ t('domain.auto.fa3d7f4b37') }}</strong>
        <p>{{ t('domain.auto.c67fa0e638') }}</p>
      </div>
      <div class="feature-guide-grid">
        <div class="feature-guide-card">
          <strong>{{ t('domain.auto.37ce088f8e') }}</strong>
          <p>{{ t('domain.auto.b5dfeee335') }}</p>
        </div>
        <div class="feature-guide-card">
          <strong>{{ t('domain.auto.9ce80bc32c') }}</strong>
          <p><code>notice</code> {{ t('domain.auto.abe592b90e') }}<code>task</code> {{ t('domain.auto.0ce684fea6') }}<code>status</code> {{ t('domain.auto.ef48b60166') }}<code>delay_days</code> {{ t('domain.auto.3b67c64eb6') }}</p>
        </div>
      </div>
      <div class="nurture-template-list">
        <div v-for="item in nurtureTemplates" :key="item.key" class="nurture-template-item">
          <div class="template-item-main">
            <strong>{{ item.name }}</strong>
            <span>{{ item.description || t('domain.auto.2406b46e19') }}</span>
          </div>
          <div class="template-item-actions">
            <el-tag size="small" :type="isPresetTemplate(item) ? 'info' : 'success'">{{ isPresetTemplate(item) ? t('domain.auto.fb40fcad41') : t('domain.auto.c493338e8c') }}</el-tag>
            <el-button size="small" plain @click="editNurtureTemplate(item)">{{ t('domain.auto.a7f814c0a4') }}</el-button>
            <el-tooltip v-if="isPresetTemplate(item)" :content="t('domain.auto.2b5e97a7c4')">
              <el-button size="small" plain type="danger" disabled>{{ t('domain.auto.3755f56f2f') }}</el-button>
            </el-tooltip>
            <el-popconfirm
              v-else
              :title="t('domain.auto.2225f4dfea')"
              :confirm-button-text="t('domain.auto.3755f56f2f')"
              :cancel-button-text="t('domain.auto.4d0b4688c7')"
              width="260"
              @confirm="deleteNurtureTemplate(item)"
            >
              <template #reference>
                <el-button size="small" plain type="danger">{{ t('domain.auto.3755f56f2f') }}</el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>
      <el-form :model="nurtureForm" label-width="92px" class="lead-dialog-form inline-action">
        <el-form-item :label="t('domain.auto.bbc511d06d')" required><el-input v-model="nurtureForm.name" :placeholder="t('domain.auto.86bd4450ee')" /></el-form-item>
        <el-form-item :label="t('domain.auto.26670dda42')"><el-input v-model="nurtureForm.description" :placeholder="t('domain.auto.3b8bff4d1c')" /></el-form-item>
        <el-form-item :label="t('domain.auto.76967fab18')" class="wide-field">
          <div class="nurture-node-builder">
            <div class="node-builder-head">
              <span>{{ t('domain.auto.93f5b8ce8b') }}</span>
              <el-button size="small" plain @click="addNurtureNode">{{ t('domain.auto.926319d436') }}</el-button>
            </div>
            <div v-for="(node, index) in nurtureNodes" :key="node.uid" class="nurture-node-row">
              <div class="node-row-main">
                <div class="node-order">{{ index + 1 }}</div>
                <el-select v-model="node.type" class="node-type-select" @change="normalizeNurtureNode(node)">
                  <el-option :label="t('domain.auto.22058b2396')" value="notice" />
                  <el-option :label="t('domain.auto.15e9b9b981')" value="task" />
                  <el-option :label="t('domain.auto.69d14924ff')" value="status" />
                </el-select>
                <el-input v-model="node.title" class="node-title-input" :placeholder="nurtureNodeTitlePlaceholder(node.type)" />
                <div class="node-delay-group">
                  <el-input-number v-model="node.delay_days" :min="0" :max="365" controls-position="right" class="node-delay-input" />
                  <span>{{ t('domain.auto.6f2e559c9e') }}</span>
                </div>
                <el-select v-if="node.type === 'status'" v-model="node.status" class="node-status-select">
                  <el-option v-for="item in leadStatuses.filter((status) => !['converted', 'invalid'].includes(status.value))" :key="item.value" :label="statusLabel(item.value)" :value="item.value" />
                </el-select>
              </div>
              <div class="node-row-bottom">
                <p class="node-row-help">{{ nurtureNodeHelp(node) }}</p>
                <div class="node-row-actions">
                  <el-button size="small" text :disabled="index === 0" @click="moveNurtureNode(index, -1)">{{ t('domain.auto.8a0c839791') }}</el-button>
                  <el-button size="small" text :disabled="index === nurtureNodes.length - 1" @click="moveNurtureNode(index, 1)">{{ t('domain.auto.05c46fa3b7') }}</el-button>
                  <el-button size="small" text type="danger" :disabled="nurtureNodes.length <= 1" @click="removeNurtureNode(index)">{{ t('domain.auto.3755f56f2f') }}</el-button>
                </div>
              </div>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button v-if="editingNurtureTemplateKey" @click="resetNurtureTemplateForm">{{ t('domain.auto.a38a511b8c') }}</el-button>
        <el-button @click="nurtureDialog = false">{{ t('domain.auto.6c14bd7f6f') }}</el-button>
        <el-button type="primary" @click="saveNurtureTemplate">{{ editingNurtureTemplateKey ? t('domain.auto.60b4ae9082') : t('domain.auto.fbfc58b501') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="automationDrawer" :title="t('domain.auto.190762da1f')" size="760px" destroy-on-close>
      <div class="automation-rule-panel">
        <div class="feature-guide">
          <strong>{{ t('domain.auto.8c882bf368') }}</strong>
          <p>{{ t('domain.auto.045ddc12c2') }}</p>
        </div>
        <el-form :model="automationRules" label-width="96px" class="automation-form">
          <section class="feature-section">
            <div class="feature-section-head">
              <strong>{{ t('domain.auto.5d088f8c01') }}</strong>
              <span>{{ t('domain.auto.5f3818f855') }}</span>
            </div>
            <div class="automation-field-grid">
              <el-form-item :label="t('domain.auto.d4e9ca3dd4')">
                <el-switch v-model="automationRules.sla.enabled" />
              </el-form-item>
              <p class="field-help">{{ t('domain.auto.f57a46036f') }}</p>
              <el-form-item :label="t('domain.auto.aaa64ad3c2')">
                <el-input-number v-model="automationRules.sla.recycle_hours" :min="1" :max="720" class="full-input" />
              </el-form-item>
              <p class="field-help">{{ t('domain.auto.22c0f617b7') }}</p>
              <el-form-item :label="t('domain.auto.4a9e2ebe75')">
                <el-input-number v-model="automationRules.sla.limit" :min="1" :max="200" class="full-input" />
              </el-form-item>
              <p class="field-help">{{ t('domain.auto.e66f1c084e') }}</p>
              <el-form-item :label="t('domain.auto.2cf6c4f195')">
                <el-select v-model="automationRules.sla.statuses" multiple class="full-input"><el-option v-for="item in leadStatuses.filter((status) => !['converted', 'invalid', 'pending_assign', 'nurturing'].includes(status.value))" :key="item.value" :label="statusLabel(item.value)" :value="item.value" /></el-select>
              </el-form-item>
              <p class="field-help">{{ t('domain.auto.619b8b68b7') }}</p>
              <el-form-item :label="t('domain.auto.d9b124b146')">
                <el-input v-model="automationRules.sla.reason" />
              </el-form-item>
              <p class="field-help">{{ t('domain.auto.023d875b5c') }}</p>
            </div>
          </section>

          <section class="feature-section">
            <div class="feature-section-head">
              <strong>{{ t('domain.auto.7d52312f9e') }}</strong>
              <span>{{ t('domain.auto.77a9c24eb4') }}</span>
            </div>
            <div class="automation-field-grid">
              <el-form-item :label="t('domain.auto.d4e9ca3dd4')"><el-switch v-model="automationRules.nurture.enabled" /></el-form-item>
              <p class="field-help">{{ t('domain.auto.2a0a7b3e2e') }}</p>
              <el-form-item :label="t('domain.auto.a51208be47')">
                <div v-if="automationNurtureTemplates.length <= 1" class="template-static-card">
                  <strong>{{ selectedAutomationNurtureTemplate.name }}</strong>
                  <span>{{ nurtureTemplateUsage(selectedAutomationNurtureTemplate) }}</span>
                </div>
                <el-select v-else v-model="automationRules.nurture.template_key" class="full-input">
                  <el-option v-for="item in automationNurtureTemplates" :key="item.key" :label="nurtureTemplateLabel(item)" :value="item.key">
                    <div class="template-option">
                      <strong>{{ item.name }}</strong>
                      <span>{{ nurtureTemplateUsage(item) }}</span>
                    </div>
                  </el-option>
                </el-select>
              </el-form-item>
              <p class="field-help">{{ t('domain.auto.3c851529ea') }}</p>
              <el-form-item :label="t('domain.auto.60e7618974')"><el-select v-model="automationRules.nurture.statuses" multiple class="full-input"><el-option v-for="item in leadStatuses.filter((status) => ['new', 'mql', 'sdr_following', 'following'].includes(status.value))" :key="item.value" :label="statusLabel(item.value)" :value="item.value" /></el-select></el-form-item>
              <p class="field-help">{{ t('domain.auto.d2a2599024') }}</p>
              <el-form-item :label="t('domain.auto.0e459d7304')"><el-input-number v-model="automationRules.nurture.score_lt" :min="0" :max="100" class="full-input" /></el-form-item>
              <p class="field-help">{{ t('domain.auto.6e402758e0') }}</p>
              <el-form-item :label="t('domain.auto.2e17ae7126')"><el-input-number v-model="automationRules.nurture.cooldown_hours" :min="0" :max="720" class="full-input" /></el-form-item>
              <p class="field-help">{{ t('domain.auto.1f35b66a6b') }}</p>
              <el-form-item :label="t('domain.auto.0dd15e791b')"><el-switch v-model="automationRules.nurture.only_without_template" /></el-form-item>
              <p class="field-help">{{ t('domain.auto.02ad1d8e53') }}</p>
            </div>
          </section>

          <section class="feature-section">
            <div class="feature-section-head">
              <strong>{{ t('domain.auto.36019dc2f4') }}</strong>
              <span>{{ t('domain.auto.76304605d0') }}</span>
            </div>
            <div class="automation-field-grid">
              <el-form-item :label="t('domain.auto.d4e9ca3dd4')"><el-switch v-model="automationRules.schedule.enabled" /></el-form-item>
              <p class="field-help">{{ t('domain.auto.22bf7ff946') }}</p>
              <el-form-item :label="t('domain.auto.075daec18d')"><el-input-number v-model="automationRules.schedule.interval_minutes" :min="5" :max="1440" class="full-input" /></el-form-item>
              <p class="field-help">{{ t('domain.auto.ff3ffb85ba') }}</p>
              <el-form-item :label="t('domain.auto.434c6bf57e')"><span class="form-static-text">{{ formatTime(automationRules.schedule.last_run_at) }}</span></el-form-item>
              <p class="field-help">{{ t('domain.auto.cca4663eb8') }}</p>
              <el-form-item :label="t('domain.auto.373074f4ee')"><span class="form-static-text">{{ formatTime(automationRules.schedule.next_run_at) }}</span></el-form-item>
              <p class="field-help">{{ t('domain.auto.4e34d63a6a') }}</p>
            </div>
          </section>
        </el-form>
        <div class="automation-actions">
          <el-button @click="dryRunAutomation">{{ t('domain.auto.03021abde0') }}</el-button>
          <el-button @click="runAutomation">{{ t('domain.auto.9a42843d10') }}</el-button>
          <el-button type="primary" @click="saveAutomationRules">{{ t('domain.auto.f5d4126103') }}</el-button>
        </div>
        <div v-if="automationResult" class="automation-result">
          <div><span>{{ t('domain.auto.998c936377') }}</span><strong>{{ automationResult.scanned || 0 }}</strong></div>
          <div><span>{{ t('domain.auto.6eda6254dd') }}</span><strong>{{ (automationResult.recycled || []).length }}</strong></div>
          <div><span>{{ t('domain.auto.8e1d856e2c') }}</span><strong>{{ (automationResult.nurtured || []).length }}</strong></div>
          <div><span>{{ t('domain.auto.3de431b126') }}</span><strong>{{ (automationResult.created || []).length }}</strong></div>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="termsDrawer" :title="t('domain.auto.fac3582ee1')" size="560px" custom-class="glossary-drawer">
      <div class="glossary-hero">
        <span><Icon icon="ri:book-open-line" /></span>
        <div>
          <strong>{{ t('domain.auto.4fb72d785f') }}</strong>
          <p>{{ t('domain.auto.dab73df046') }}</p>
        </div>
      </div>
      <div class="glossary-tools">
        <el-input v-model="glossaryKeyword" clearable :placeholder="t('domain.auto.3e6ff77ec9')" />
        <el-segmented v-model="glossaryCategory" :options="glossaryCategoryOptions" />
      </div>
      <div class="lead-terms-list">
        <div v-for="term in filteredGlossaryTerms" :key="term.key" class="lead-term-item">
          <span class="term-icon"><Icon :icon="term.icon" /></span>
          <div class="term-main">
            <div class="term-title">
              <strong>{{ term.key }}</strong>
              <el-tag size="small" effect="plain">{{ term.name }}</el-tag>
            </div>
            <small>{{ term.fullName }}</small>
            <p>{{ term.text }}</p>
            <em>{{ term.action }}</em>
          </div>
        </div>
        <el-empty v-if="!filteredGlossaryTerms.length" :description="t('domain.auto.3168738872')" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { timelineOperatorText } from '../../utils/timeline'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'
import DealChainPanel from '../../components/DealChainPanel.vue'
import ObjectAccessPanel from '../../components/ObjectAccessPanel.vue'
import ObjectAccessField from '../../components/ObjectAccessField.vue'
import MetricCard from '../../components/MetricCard.vue'
import { playCelebration } from '../../utils/celebration'
import { currencyOptions } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { apiErrorMessage } from '../../utils/api-error'

const leadStatuses = [
  { value: 'new' },
  { value: 'pending_assign' },
  { value: 'following' },
  { value: 'mql' },
  { value: 'sdr_following' },
  { value: 'sql' },
  { value: 'nurturing' },
  { value: 'converted' },
  { value: 'invalid' }
]
const DAILY_EXCLUDE_STATUSES = 'converted,invalid'
const columnOptions = [
  { key: 'name', label: t('domain.auto.733eabcea7') },
  { key: 'contact', label: t('domain.auto.2425bd4bc1') },
  { key: 'phone', label: t('domain.auto.8694f91905') },
  { key: 'source', label: t('domain.auto.c63f79e636') },
  { key: 'campaign', label: t('domain.auto.bb002ebde7') },
  { key: 'status', label: t('domain.auto.62e951a692') },
  { key: 'temperature', label: t('domain.auto.296adf9512') },
  { key: 'score', label: t('domain.auto.2f58566ce0') },
  { key: 'owner', label: t('domain.auto.974d383f36') },
  { key: 'process', label: t('domain.auto.2612f9775a') },
  { key: 'updated_at', label: t('domain.auto.093dea88c9') }
]
const viewModeOptions = [
  { label: t('domain.auto.d46f82fdf0'), value: 'table' },
  { label: t('domain.auto.9dc1a19f2c'), value: 'kanban' },
  { label: t('domain.auto.5a6c64382a'), value: 'space' }
]
const utmOptionGroups = [
  {
    label: t('domain.auto.975b40f704'),
    options: [
      { value: 'google_ads', label: 'Google Ads' },
      { value: 'bing_ads', label: 'Bing Ads' },
      { value: 'meta_ads', label: 'Meta/Facebook Ads' },
      { value: 'linkedin_ads', label: 'LinkedIn Ads' },
      { value: 'tiktok_ads', label: 'TikTok Ads' }
    ]
  },
  {
    label: t('domain.auto.cb77f9f63b'),
    options: [
      { value: 'website_form', label: t('domain.auto.a2af045711') },
      { value: 'landing_page', label: t('domain.auto.1ed8160dcb') },
      { value: 'webinar', label: t('domain.auto.454170ab01') },
      { value: 'whitepaper', label: t('domain.auto.785e4caa19') },
      { value: 'expo_event', label: t('domain.auto.2fce6b3fa6') }
    ]
  },
  {
    label: t('domain.auto.8ba30a6c1f'),
    options: [
      { value: 'linkedin', label: 'LinkedIn' },
      { value: 'facebook', label: 'Facebook' },
      { value: 'whatsapp', label: 'WhatsApp' },
      { value: 'line', label: 'Line' },
      { value: 'referral', label: t('domain.auto.b5ea49f77d') }
    ]
  },
  {
    label: t('domain.auto.25927bb889'),
    options: [
      { value: 'email_campaign', label: t('domain.auto.b773308aff') },
      { value: 'newsletter', label: 'Newsletter' },
      { value: 'remarketing', label: t('domain.auto.358005f41b') },
      { value: 'direct', label: t('domain.auto.3400e6edb1') },
      { value: 'organic_search', label: t('domain.auto.8803fa49b4') }
    ]
  }
]

const rows = ref([])
const users = ref([])
const products = ref([])
const campaigns = ref([])
const companies = ref([])
const contacts = ref([])
const leadViews = ref([])
const selectedRows = ref([])
const timeline = ref([])
const loading = ref(false)
const searchExpanded = ref(false)
const isFullscreen = ref(false)
const columnDrawer = ref(false)
const formDialog = ref(false)
const actionDialog = ref(false)
const scoreDialog = ref(false)
const detailDrawer = ref(false)
const importDialog = ref(false)
const importFileInput = ref(null)
const leadFormRef = ref(null)
const nurtureDialog = ref(false)
const automationDrawer = ref(false)
const sdrDrawer = ref(false)
const abmDrawer = ref(false)
const termsDrawer = ref(false)
const glossaryKeyword = ref('')
const glossaryCategory = ref('all')
const activeView = ref('')
const editingId = ref(null)
const activeRecord = ref(null)
const detailRecord = ref(null)
const sdrActiveLead = ref(null)
const actionMode = ref('')
const editingNurtureTemplateKey = ref('')
const importText = ref('{"items":[]}')
const importFileName = ref('')
const importHeaders = ref([])
const importRawRows = ref([])
const importPreviewRows = ref([])
const viewMode = ref('table')
const abmText = ref('')
const abmTargets = ref(loadABMTargets())
const nurtureTemplates = ref([])
const nurtureNodes = ref(defaultNurtureNodes())
const formDedupeText = ref('')
const columnSelections = ref(loadColumns())
const filters = reactive({ company_name: '', contact_name: '', keyword: '', source_channel: '', status: '', owner_id: undefined })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const dashboard = reactive({ kpi: {}, funnel: [], sources: [], temperature: {}, workload: [], alerts: {} })
const dedupe = reactive({ leads: [], possible_duplicates: [], same_contact_leads: [], same_company_leads: [], companies: [], contacts: [], match_summary: {} })
const formDedupe = reactive({ leads: [], possible_duplicates: [], same_contact_leads: [], same_company_leads: [], companies: [], contacts: [], match_summary: {} })
const form = reactive(defaultForm())
const leadFormRules = {
  'payload.company_name': [{ validator: validateLeadCompanyName, trigger: ['blur', 'change'] }],
  'payload.contact_name': [{ validator: validateLeadContactName, trigger: ['blur', 'change'] }],
  'payload.phone': [{ validator: validateLeadContactMethod, trigger: ['blur', 'change'] }],
  'payload.email': [{ validator: validateLeadContactMethod, trigger: ['blur', 'change'] }],
  'payload.source_channel': [{ validator: validateLeadSource, trigger: ['blur', 'change'] }],
  'payload.campaign_id': [{ validator: validateLeadCampaign, trigger: ['blur', 'change'] }],
  'payload.intent_product_ids': [{ validator: validateLeadIntentProduct, trigger: ['blur', 'change'] }],
  owner_id: [{ validator: validateLeadOwner, trigger: ['blur', 'change'] }]
}
const actionForm = reactive({})
const scoreForm = reactive({ score: 0, reason: '' })
const nurtureForm = reactive({ name: t('domain.auto.54c2287c74'), description: t('domain.auto.30c4361c5d') })
const automationRules = reactive(defaultAutomationRules())
const automationResult = ref(null)
const importMapping = reactive(defaultImportMapping())
const importFieldOptions = [
  { key: 'company_name', label: t('domain.auto.e1272432a4') },
  { key: 'contact_name', label: t('domain.auto.2425bd4bc1') },
  { key: 'phone', label: t('domain.auto.9c01ad0911') },
  { key: 'email', label: t('domain.auto.9ed627bcf6') },
  { key: 'source_channel', label: t('domain.auto.c63f79e636') },
  { key: 'intent_product', label: t('domain.auto.bcb5539330') },
  { key: 'utm_source', label: 'UTM' },
  { key: 'status', label: t('domain.auto.62e951a692') },
  { key: 'owner_id', label: t('domain.auto.974d383f36') },
  { key: 'amount', label: t('domain.auto.34943c40c9') },
  { key: 'tags', label: t('domain.auto.ae0a7afece') }
]
const glossaryCategoryOptions = [
  { label: t('domain.auto.778fc8f994'), value: 'all' },
  { label: t('domain.auto.aa59e051af'), value: 'marketing' },
  { label: t('domain.auto.a5fd0e515f'), value: 'sales' },
  { label: t('domain.auto.4b81846f62'), value: 'service' }
]
const glossaryTerms = [
  { key: 'ABM', name: t('domain.auto.c2bf01ab8a'), fullName: 'Account-Based Marketing', category: 'marketing', icon: 'ri:building-4-line', text: t('domain.auto.c2b29f65fd'), action: t('domain.auto.824a81ee0b') },
  { key: 'SDR', name: t('domain.auto.5775570394'), fullName: 'Sales Development Representative', category: 'sales', icon: 'ri:customer-service-2-line', text: t('domain.auto.6351f84cc4'), action: t('domain.auto.667864ed28') },
  { key: 'UTM', name: t('domain.auto.105611e075'), fullName: 'Urchin Tracking Module', category: 'marketing', icon: 'ri:route-line', text: t('domain.auto.d09819a6c4'), action: t('domain.auto.ac57980acc') },
  { key: 'MQL', name: t('domain.auto.5ccfefa3a5'), fullName: 'Marketing Qualified Lead', category: 'marketing', icon: 'ri:user-star-line', text: t('domain.auto.b0458c0c90'), action: t('domain.auto.c8ba3d0d4e') },
  { key: 'SQL', name: t('domain.auto.4d297cc4d2'), fullName: 'Sales Qualified Lead', category: 'sales', icon: 'ri:exchange-dollar-line', text: t('domain.auto.b3e4b37ed8'), action: t('domain.auto.1ac2562dde') },
  { key: 'SLA', name: t('domain.auto.176e12801f'), fullName: 'Service Level Agreement', category: 'service', icon: 'ri:timer-flash-line', text: t('domain.auto.b167bdb3e7'), action: t('domain.auto.b6eca9edf3') }
]
const filteredGlossaryTerms = computed(() => {
  const keyword = glossaryKeyword.value.trim().toLowerCase()
  return glossaryTerms.filter((term) => {
    const matchCategory = glossaryCategory.value === 'all' || term.category === glossaryCategory.value
    const text = `${term.key} ${term.name} ${term.fullName} ${term.text} ${term.action}`.toLowerCase()
    return matchCategory && (!keyword || text.includes(keyword))
  })
})
const hasActiveLeadFilters = computed(() => Object.values(filters).some((value) => value !== '' && value !== undefined && value !== null))

const router = useRouter()
const route = useRoute()
const detailPayload = computed(() => payload(detailRecord.value))
const isConvertedDetail = computed(() => detailRecord.value?.status === 'converted')
const isInvalidDetail = computed(() => detailRecord.value?.status === 'invalid')
const convertedOpportunityID = computed(() => normalizeRecordID(detailPayload.value.opportunity_id || detailPayload.value.converted_opportunity_id))
const convertedCompanyID = computed(() => normalizeRecordID(detailRecord.value?.company_id || detailPayload.value.company_id))
const convertedContactID = computed(() => normalizeRecordID(detailRecord.value?.contact_id || detailPayload.value.contact_id))
const dedupePossibleDuplicates = computed(() => dedupe.possible_duplicates?.length ? dedupe.possible_duplicates : dedupe.leads || [])
const dedupeHistoryRows = computed(() => uniqueLeadRows([...(dedupe.same_contact_leads || []), ...(dedupe.same_company_leads || [])], detailRecord.value?.id))
const formDedupeState = computed(() => {
  const possible = formDedupe.possible_duplicates?.length ? formDedupe.possible_duplicates : formDedupe.leads || []
  const history = uniqueLeadRows([...(formDedupe.same_contact_leads || []), ...(formDedupe.same_company_leads || [])])
  const possibleCount = possible.length
  const contactCount = formDedupe.contacts?.length || 0
  const companyCount = formDedupe.companies?.length || 0
  const historyCount = history.length
  return {
    possibleCount,
    contactCount,
    companyCount,
    historyCount,
    hasAny: Boolean(possibleCount || contactCount || companyCount || historyCount),
    warningText: `${t('domain.auto.523e40a074')}${possibleCount}${t('domain.auto.56ddb5a8ec')}`,
    contactText: `${t('domain.auto.9a9c4d18d9')}${contactCount}${t('domain.auto.17682f2cc3')}`,
    companyText: `${t('domain.auto.9a9c4d18d9')}${companyCount}${t('domain.auto.c6bd810683')}`
  }
})
const kpiCards = computed(() => [
  { key: 'total', title: t('domain.auto.f3ce444ad8'), value: dashboard.kpi.total || 0, icon: 'ri:contacts-line', tone: 'blue', help: t('domain.auto.156ba4a59f') },
  { key: 'today', title: t('domain.auto.a2e8544489'), value: dashboard.kpi.today || 0, icon: 'ri:add-circle-line', tone: 'green', help: t('domain.auto.3b17d7e43c') },
  { key: 'high_value', title: t('domain.auto.2ca8e356ce'), value: dashboard.kpi.high_value || 0, icon: 'ri:vip-crown-line', tone: 'purple', help: t('domain.auto.543dbb684a') },
  { key: 'overdue', title: t('domain.auto.ef8d36f143'), value: dashboard.kpi.overdue || 0, icon: 'ri:alarm-warning-line', tone: 'orange', help: t('domain.auto.44cdb39741') }
])
const maxFunnelCount = computed(() => Math.max(1, ...((dashboard.funnel || []).map((item) => Number(item.count || 0)))))
const sdrRows = computed(() => rows.value.filter((item) => ['mql', 'sdr_following', 'following'].includes(item.status)).slice(0, 6))
const kanbanColumns = computed(() => ['new', 'pending_assign', 'following', 'mql', 'sdr_following', 'sql', 'nurturing'].map((status) => ({ status, items: rows.value.filter((item) => item.status === status) })))
const customerSpaceRows = computed(() => rows.value.filter((item) => !isTerminal(item)).slice(0, 12))
const actionTitle = computed(() => ({ assign: t('domain.auto.562cdcd951'), convert: t('domain.auto.8005c2448b'), follow: t('domain.auto.a68cc3cf80'), status: t('domain.auto.12bfbb0aac'), release: t('domain.auto.9e72d69574'), nurture: t('domain.auto.5f40216260') }[actionMode.value] || t('domain.auto.3b7c56d6b3')))
const importValidRows = computed(() => importPreviewRows.value.filter((item) => item.valid))
const importInvalidRows = computed(() => importPreviewRows.value.filter((item) => !item.valid))
const abmMatchedRows = computed(() => rows.value.filter((item) => isABMLead(item)))
const selectedInvalidRows = computed(() => selectedRows.value.filter((item) => item.status === 'invalid'))
const automationNurtureTemplates = computed(() => {
  const filtered = nurtureTemplates.value.filter((item) => item.key !== 'mql_rejected_recycle')
  return filtered.length ? filtered : nurtureTemplates.value
})
const selectedAutomationNurtureTemplate = computed(() => {
  return automationNurtureTemplates.value.find((item) => item.key === automationRules.nurture.template_key) ||
    automationNurtureTemplates.value[0] ||
    { key: automationRules.nurture.template_key || 'standard_education', name: t('domain.auto.4ee753df0b'), description: t('domain.auto.cd189477bb') }
})
const ownerOptions = computed(() => {
  const fromUsers = users.value.map((user) => ({ id: user.id, label: userDisplayName(user) })).filter((item) => item.id)
  return fromUsers
})
const productOptions = computed(() => products.value.filter((item) => item.status !== 'stopped'))
const companyOptions = computed(() => companies.value.filter((item) => item.status !== 'inactive' && item.status !== 'merged'))
const contactOptions = computed(() => {
  const companyID = normalizeRecordID(form.payload.company_id)
  const active = contacts.value.filter((item) => item.status !== 'inactive' && item.status !== 'merged')
  if (!companyID) return active
  const matched = active.filter((item) => Number(item.company_id || payload(item).company_id || 0) === companyID)
  return matched.length ? matched : active
})
const campaignOptions = computed(() => campaigns.value.filter((item) => item.object_type === 'campaign'))
const canLeadCreate = computed(() => hasLeadPermission('create'))
const canLeadUpdate = computed(() => hasLeadPermission('update'))
const canLeadDelete = computed(() => hasLeadPermission('delete'))
const canLeadExport = computed(() => hasLeadPermission('export'))
const canLeadAssign = computed(() => hasLeadPermission('assign') || hasLeadPermission('update'))

onMounted(async () => {
  await refreshAll()
  await applyCampaignQueryCreate()
  applyCreateQuery()
})

async function refreshAll() {
  await Promise.all([loadUsers(), loadProducts(), loadCampaigns(), loadContactRecords(), loadViews(), loadDashboard(), loadNurtureTemplates(), loadAutomationRules(), loadABMSettings(), loadPage()])
}

async function loadUsers() {
  try {
    const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } })
    users.value = res.data?.items || []
    if (!users.value.length) {
      const fallback = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200 } })
      users.value = fallback.data?.items || []
    }
  } catch {
    const me = getCurrentUser()
    users.value = me?.id ? [me] : []
  }
  const me = getCurrentUser()
  if (me?.id && !users.value.some((item) => Number(item.id) === Number(me.id))) {
    users.value = [{ ...me, name: me.name || me.username }, ...users.value]
  }
}

async function loadPage() {
  loading.value = true
  try {
    const params = cleanParams({ ...filters, page: pagination.page, size: pagination.size, object_type: 'lead', exclude_status: defaultExcludeStatus() })
    const res = await api.get('/api/v1/prd/leads/objects', { params })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.leads.message.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function loadProducts() {
  try {
    const res = await api.get('/api/v1/prd/products/objects', { params: { object_type: 'product', page: 1, size: 200 } })
    products.value = res.data?.items || []
  } catch {
    products.value = []
  }
}

async function loadCampaigns() {
  try {
    const res = await api.get('/api/v1/prd/campaigns/objects', { params: { object_type: 'campaign', page: 1, size: 300 } })
    campaigns.value = res.data?.items || []
  } catch {
    campaigns.value = []
  }
}

async function loadContactRecords() {
  try {
    const [companyRes, contactRes] = await Promise.all([
      api.get('/api/v1/prd/contacts/objects', { params: { object_type: 'company', page: 1, size: 300 } }),
      api.get('/api/v1/prd/contacts/objects', { params: { object_type: 'contact', page: 1, size: 300 } })
    ])
    companies.value = companyRes.data?.items || []
    contacts.value = contactRes.data?.items || []
  } catch {
    companies.value = []
    contacts.value = []
  }
}

async function loadViews() {
  const res = await api.get('/api/v1/prd/leads/views')
  leadViews.value = res.data?.items || []
}

async function loadDashboard() {
  const res = await api.get('/api/v1/prd/leads/dashboard')
  Object.assign(dashboard, res.data || {})
}

async function loadNurtureTemplates() {
  const res = await api.get('/api/v1/prd/leads/nurture/templates')
  nurtureTemplates.value = res.data?.items || []
}

async function loadAutomationRules() {
  const res = await api.get('/api/v1/prd/leads/automation/rules')
  Object.assign(automationRules, normalizeAutomationRules(res.data?.rules || {}))
}

function applyView(view) {
  activeView.value = view.key
  resetFilters(false)
  Object.assign(filters, normalizeViewFilters(view.filters || {}))
  loadPage()
}

function applyInvalidView() {
  activeView.value = 'invalid'
  resetFilters(false)
  filters.status = 'invalid'
  loadPage()
}

function applyConvertedView() {
  activeView.value = 'converted'
  resetFilters(false)
  filters.status = 'converted'
  loadPage()
}

function showAllLeads() {
  resetFilters(true)
  loadPage()
}

function openCreate() {
  editingId.value = null
  Object.assign(form, defaultForm())
  formDialog.value = true
  formDedupeText.value = ''
  resetDedupeState(formDedupe)
  if (!products.value.length) loadProducts()
  if (!campaigns.value.length) loadCampaigns()
  if (!companies.value.length || !contacts.value.length) loadContactRecords()
}

function openEdit(row) {
  if (isTerminal(row)) return
  editingId.value = row.id
  const p = payload(row)
  Object.assign(form, {
    object_type: 'lead',
    name: row.name,
    code: row.code || '',
    status: row.status,
    owner_id: row.owner_id || 0,
    amount: row.amount || 0,
    currency: row.currency || defaultBusinessCurrency(),
    score: row.score || 0,
    tags: splitTags(row.tags),
    company_id: row.company_id || normalizeRecordID(p.company_id),
    contact_id: row.contact_id || normalizeRecordID(p.contact_id),
    payload: { ...p, company_id: row.company_id || normalizeRecordID(p.company_id), contact_id: row.contact_id || normalizeRecordID(p.contact_id), intent_product_ids: normalizeProductIDs(p.intent_product_ids || p.product_ids || p.intent_product_id || p.product_id), intent_product_id: normalizeProductID(p.intent_product_id || p.product_id) }
  })
  formDialog.value = true
  formDedupeText.value = ''
  resetDedupeState(formDedupe)
  if (!products.value.length) loadProducts()
  if (!companies.value.length || !contacts.value.length) loadContactRecords()
}

async function goOpportunity(row) {
  const id = await resolveConvertedOpportunityID(row)
  if (!id) {
    ElMessage.warning(t('domain.auto.1bb4f70923'))
    return
  }
  router.push({ path: '/opportunities', query: { id } })
}

async function goContactObject(type) {
  const id = type === 'company' ? convertedCompanyID.value : convertedContactID.value
  if (!id) {
    ElMessage.warning(type === 'company' ? t('domain.auto.cfe3a143cb') : t('domain.auto.8e43baa810'))
    return
  }
  router.push({ path: type === 'company' ? '/companies' : '/contacts', query: { id } })
}

function scrollDealChain() {
  document.getElementById('lead-deal-chain')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function resolveConvertedOpportunityID(row) {
  const p = payload(row)
  const fromPayload = normalizeRecordID(p.opportunity_id || p.converted_opportunity_id)
  if (fromPayload) return fromPayload
  try {
    const res = await api.get('/api/v1/prd/relations', { params: { object_type: 'leads', object_id: row.id } })
    const relation = (res.data?.items || []).find((item) => item.target_type === 'opportunities' && item.relation_type === 'converted_to')
    return normalizeRecordID(relation?.target_id)
  } catch {
    return 0
  }
}

async function submitForm() {
  const validation = validateLeadFormRequired()
  if (!validation.valid) {
    ElMessage.error(`${t('domain.auto.8148ff0a4a')}${validation.missing.join('、')}`)
    await leadFormRef.value?.validate?.().catch(() => false)
    return
  }
  normalizeSelectedLeadCompanyContact()
  normalizeSelectedLeadProduct()
  form.name = `${form.payload.company_name}-${form.payload.contact_name}`
  const payload = { ...form, tags: form.tags || [] }
  try {
    if (editingId.value) await api.put(`/api/v1/prd/leads/objects/${editingId.value}`, payload)
    else await api.post('/api/v1/prd/leads/objects', payload)
    ElMessage.success(t('common.message.opSuccess'))
    formDialog.value = false
    await refreshAll()
  } catch (error) {
    ElMessage.error(actionFailureMessage(editingId.value ? 'updateLead' : 'createLead', error))
  }
}

async function checkDedupeForm() {
  if (!form.payload.company_name && !form.payload.contact_name && !form.payload.phone && !form.payload.email) {
    resetDedupeState(formDedupe)
    formDedupeText.value = ''
    return
  }
  const res = await api.post('/api/v1/prd/flows/leads/dedupe', leadDedupePayload({ ...form.payload, lead_id: editingId.value }))
  Object.assign(formDedupe, normalizeDedupeResult(res.data || {}))
  formDedupeText.value = formDedupe.possible_duplicates.length ? formDedupeState.value.warningText : ''
}

async function openDetail(row) {
  detailRecord.value = row
  detailDrawer.value = true
  await Promise.all([loadDedupe(row), loadTimeline(row)])
}

async function loadDedupe(row) {
  const p = payload(row)
  const res = await api.post('/api/v1/prd/flows/leads/dedupe', leadDedupePayload({ ...p, lead_id: row.id, company_id: row.company_id || p.company_id, contact_id: row.contact_id || p.contact_id }))
  Object.assign(dedupe, normalizeDedupeResult(res.data || {}))
}

async function loadTimeline(row) {
  const res = await api.get('/api/v1/prd/relations', { params: { object_type: 'leads', object_id: row.id } })
  const relations = res.data?.items || []
  const ids = relations
    .filter((item) => item.target_type === 'contacts' && item.relation_type === 'timeline')
    .map((item) => Number(item.target_id))
    .filter((id) => id > 0)
  if (!ids.length) {
    timeline.value = []
    return
  }
  const fetched = await Promise.all(ids.map((id) => api.get(`/api/v1/prd/contacts/objects/${id}`)))
  timeline.value = fetched
    .map((res) => res.data)
    .filter((item) => item && item.object_type === 'interaction')
    .sort((a, b) => new Date(b.created_at || 0).getTime() - new Date(a.created_at || 0).getTime())
}

async function refreshDetailRealtime(targetID) {
  if (!detailDrawer.value || !detailRecord.value) return
  if (targetID && Number(detailRecord.value.id) !== Number(targetID)) return
  const latest = rows.value.find((item) => Number(item.id) === Number(detailRecord.value.id))
  if (latest) detailRecord.value = latest
  await Promise.all([loadDedupe(detailRecord.value), loadTimeline(detailRecord.value)])
}

function openAssign(row) {
  activeRecord.value = row
  actionMode.value = 'assign'
  Object.assign(actionForm, { owner_id: row.owner_id || 1, reason: t('domain.auto.05e327834f') })
  actionDialog.value = true
}

function openConvert(row) {
  const p = payload(row)
  const companyName = p.company_name || p.company || ''
  const contactName = p.contact_name || p.contact || ''
  if (!String(companyName || '').trim() || !String(contactName || '').trim()) {
    ElMessage.warning(t('domain.leads.message.convertMissingCompanyContact'))
    openEdit(row)
    return
  }
  activeRecord.value = row
  actionMode.value = 'convert'
  Object.assign(actionForm, { company_name: companyName, contact_name: contactName, deal_name: `${row.name}${t('domain.auto.c4beee8569')}`, amount: row.amount || 100000, currency: row.currency || p.currency || defaultBusinessCurrency() })
  actionDialog.value = true
}

function openFollowUp(row) {
  activeRecord.value = row
  actionMode.value = 'follow'
  Object.assign(actionForm, { type: 'note', content: '', next_action: payload(row).next_action || '', channel_label: '' })
  actionDialog.value = true
}

function openCommunicationAction(row, provider) {
  const labels = { whatsapp: t('domain.auto.a44fdcc712'), line: t('domain.auto.f22c86aa43') }
  activeRecord.value = row
  actionMode.value = 'follow'
  const p = payload(row)
  Object.assign(actionForm, {
    type: 'social',
    channel_label: labels[provider] || t('domain.auto.1f3cb01e46'),
    content: `${labels[provider] || t('domain.auto.1f3cb01e46')}：`,
    next_action: p.next_action || t('domain.auto.52e4a62ffa')
  })
  actionDialog.value = true
}

function logLeadCommunication(row, provider) {
  openCommunicationAction(row, provider)
}

function openStatus(row, status) {
  activeRecord.value = row
  actionMode.value = 'status'
  Object.assign(actionForm, { status, reason: status === 'invalid' ? t('domain.auto.31d21dd06e') : t('domain.auto.d314665f35') })
  actionDialog.value = true
}

async function submitAction() {
  const row = activeRecord.value
  if (!row) return
  const mode = actionMode.value
  try {
    if (mode === 'assign') {
      await api.post(`/api/v1/prd/flows/leads/${row.id}/assign`, { owner_id: actionForm.owner_id, reason: actionForm.reason || t('domain.auto.05e327834f') })
    } else if (mode === 'convert') {
      if (!String(actionForm.reason || '').trim()) {
        ElMessage.warning(t('domain.leads.message.convertReasonRequired'))
        return
      }
      await api.post(`/api/v1/prd/flows/leads/${row.id}/convert`, {
        deal_name: actionForm.deal_name,
        amount: actionForm.amount,
        currency: actionForm.currency || row.currency || defaultBusinessCurrency(),
        reason: actionForm.reason
      })
      playCelebration('milestone')
    } else if (mode === 'follow') {
      if (!String(actionForm.content || '').trim()) {
        ElMessage.warning(t('domain.leads.message.followupRequired'))
        return
      }
      await api.post(`/api/v1/prd/flows/leads/${row.id}/follow-up`, actionForm)
    } else if (mode === 'nurture') {
      if (!String(actionForm.reason || '').trim()) {
        ElMessage.warning(t('domain.leads.message.nurtureReasonRequired'))
        return
      }
      await startNurtureWithReason(row, actionForm.template_key || 'standard_education', actionForm.reason)
      playCelebration('milestone')
    } else if (mode === 'status') {
      if (!String(actionForm.reason || '').trim()) {
        ElMessage.warning(t('domain.leads.message.statusReasonRequired'))
        return
      }
      await api.post(`/api/v1/prd/flows/leads/${row.id}/status`, actionForm)
    }
    ElMessage.success(actionSuccessMessage(mode))
    actionDialog.value = false
    await refreshAll()
    if (shouldCollapseLeadFlowDrawer(row, mode)) {
      detailDrawer.value = false
    } else if (mode !== 'convert') {
      await refreshDetailRealtime(row.id)
    }
  } catch (error) {
    ElMessage.error(actionFailureMessage(mode, error))
  }
}

function scoreLead(row) {
  activeRecord.value = row
  Object.assign(scoreForm, {
    score: Number(row.score || payload(row).ai_score || 0),
    reason: payload(row).score_reason || ''
  })
  scoreDialog.value = true
}

async function submitScore() {
  const row = activeRecord.value
  if (!row) return
  const p = payload(row)
  const score = Number(scoreForm.score || 0)
  try {
    await api.put(`/api/v1/prd/leads/objects/${row.id}`, {
      object_type: 'lead',
      name: row.name,
      code: row.code || '',
      status: row.status,
      owner_id: row.owner_id || 0,
      company_id: row.company_id || p.company_id || 0,
      contact_id: row.contact_id || p.contact_id || 0,
      amount: row.amount || 0,
      currency: row.currency || defaultBusinessCurrency(),
      score,
      tags: splitTags(row.tags),
      payload: {
        ...p,
        ai_score: score,
        temperature: scoreToTemperature(score),
        score_reason: scoreForm.reason || t('domain.auto.75528d7ad5'),
        scored_at: new Date().toISOString()
      }
    })
    scoreDialog.value = false
    ElMessage.success(t('domain.leads.message.scoreSaved'))
    await refreshAll()
    await refreshDetailRealtime(row.id)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.leads.message.scoreSaveFailed')))
  }
}

async function releaseLead(row) {
  activeRecord.value = row
  actionMode.value = 'release'
  Object.assign(actionForm, { reason: t('domain.auto.4bbeaff4cf') })
  await submitRelease()
}

async function submitRelease() {
  const row = activeRecord.value
  if (!row) return
  try {
    await api.post(`/api/v1/prd/flows/leads/${row.id}/release`, { reason: actionForm.reason || t('domain.auto.c663363b13') })
    ElMessage.success(t('domain.leads.message.released'))
    await refreshAll()
    if (shouldCollapseLeadFlowDrawer(row, 'release')) {
      detailDrawer.value = false
    } else {
      await refreshDetailRealtime(row.id)
    }
  } catch (error) {
    ElMessage.error(actionFailureMessage('release', error))
  }
}

async function runSLARecycle() {
  if (!ensureLeadManagePermission()) return
  await api.post('/api/v1/prd/flows/leads/sla-recycle', { hours: 0, limit: 100, reason: t('domain.auto.511064e2c7'), force: false })
  ElMessage.success(t('domain.leads.message.slaRecycled'))
  await refreshAll()
}

function openNurtureAction(row) {
  activeRecord.value = row
  actionMode.value = 'nurture'
  const template = nurtureTemplates.value[0] || { key: 'standard_education' }
  Object.assign(actionForm, { template_key: template.key, reason: '' })
  actionDialog.value = true
}

async function startNurtureWithReason(row, templateKey, reason) {
  await api.post(`/api/v1/prd/flows/leads/${row.id}/nurture`, { template_key: templateKey, reason })
  await refreshDetailRealtime(row.id)
}

async function applySDRView() {
  resetFilters(false)
  filters.status = 'sdr_following'
  activeView.value = 'sdr'
  await loadPage()
}

function openSDRWorkbench(row) {
  sdrActiveLead.value = row || sdrRows.value[0] || null
  sdrDrawer.value = true
}

function selectSDRLead(row) {
  sdrActiveLead.value = row
}

function handleSDRKeydown(event) {
  if (!sdrActiveLead.value) return
  if (event.key === '1') {
    event.preventDefault()
    sdrMarkSQL()
  } else if (event.key === '2') {
    event.preventDefault()
    sdrRejectToNurture()
  }
}

async function sdrMarkSQL() {
  if (!sdrActiveLead.value) return
  await api.post(`/api/v1/prd/flows/leads/${sdrActiveLead.value.id}/status`, { status: 'sql', reason: t('domain.auto.397f53b9a8') })
  ElMessage.success(t('domain.leads.message.markedSql'))
  sdrDrawer.value = false
  await refreshAll()
  await refreshDetailRealtime(sdrActiveLead.value.id)
}

async function sdrRejectToNurture() {
  if (!sdrActiveLead.value) return
  const reason = t('domain.auto.f182335159')
  await api.post(`/api/v1/prd/flows/leads/${sdrActiveLead.value.id}/status`, { status: 'nurturing', reason })
  const template = nurtureTemplates.value.find((item) => item.key === 'mql_rejected_recycle') || nurtureTemplates.value[0] || { key: 'mql_rejected_recycle' }
  await startNurtureWithReason(sdrActiveLead.value, template.key, reason)
  ElMessage.success(t('domain.leads.message.sentNurture'))
  sdrDrawer.value = false
  await refreshAll()
  await refreshDetailRealtime(sdrActiveLead.value.id)
}

async function openABMDrawer() {
  if (!ensureLeadManagePermission()) return
  await loadABMSettings()
  abmText.value = abmTargets.value.join('\n')
  matchABMRows()
  abmDrawer.value = true
}

async function saveABMTargets() {
  if (!ensureLeadManagePermission()) return
  abmTargets.value = String(abmText.value || '').split(/\r?\n/).map((item) => item.trim()).filter(Boolean)
  localStorage.setItem('lead_abm_targets', JSON.stringify(abmTargets.value))
  await api.post('/api/v1/prd/settings', { scope: 'lead_abm', key: 'targets', value: { accounts: abmTargets.value } })
  await matchABMRows(true)
  ElMessage.success(t('domain.leads.message.abmSaved'))
}

async function loadABMSettings() {
  try {
    const res = await api.get('/api/v1/prd/settings', { params: { scope: 'lead_abm', key: 'targets' } })
    const setting = res.data?.items?.[0]
    if (!setting) return
    const value = typeof setting.value_json === 'string' ? JSON.parse(setting.value_json || '{}') : (setting.value_json || {})
    const accounts = Array.isArray(value.accounts) ? value.accounts : []
    if (accounts.length) {
      abmTargets.value = accounts
      localStorage.setItem('lead_abm_targets', JSON.stringify(accounts))
    }
  } catch {
    abmTargets.value = loadABMTargets()
  }
}

async function matchABMRows(persist = false) {
  const targets = String(abmText.value || '').split(/\r?\n/).map((item) => item.trim()).filter(Boolean)
  abmTargets.value = targets
  if (!persist) return
  for (const row of rows.value) {
    const matched = matchedABMTarget(row)
    const p = payload(row)
    if (matched && !p.abm_target_account) {
      await api.put(`/api/v1/prd/leads/objects/${row.id}`, {
        ...row,
        tags: splitTags(row.tags).includes('abm') ? splitTags(row.tags) : [...splitTags(row.tags), 'abm'],
        payload: { ...p, abm_target_account: true, abm_target_account_name: matched, abm_matched_at: new Date().toISOString() }
      })
    }
  }
  await refreshAll()
}

async function openNurtureTemplate() {
  if (!ensureLeadManagePermission()) return
  await loadNurtureTemplates()
  resetNurtureTemplateForm()
  nurtureDialog.value = true
}

async function openAutomationDrawer() {
  if (!ensureLeadManagePermission()) return
  automationResult.value = null
  automationDrawer.value = true
  try {
    await Promise.all([loadNurtureTemplates(), loadAutomationRules()])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.leads.message.automationLoadFailed')))
  }
}

async function saveNurtureTemplate() {
  if (!ensureLeadManagePermission()) return
  const nodes = serializeNurtureNodes()
  if (!nodes.length) {
    ElMessage.warning(t('domain.leads.message.nurtureNodeRequired'))
    return
  }
  if (editingNurtureTemplateKey.value) {
    await api.put(`/api/v1/prd/leads/nurture/templates/${encodeURIComponent(editingNurtureTemplateKey.value)}`, { ...nurtureForm, nodes })
    ElMessage.success(t('domain.leads.message.nurtureTemplateUpdated'))
  } else {
    await api.post('/api/v1/prd/leads/nurture/templates', { ...nurtureForm, nodes })
    ElMessage.success(t('domain.leads.message.nurtureTemplateSaved'))
  }
  resetNurtureTemplateForm()
  await loadNurtureTemplates()
}

function editNurtureTemplate(template) {
  Object.assign(nurtureForm, {
    name: template.name || '',
    description: template.description || ''
  })
  nurtureNodes.value = templateNodesForEditor(template.nodes || [])
  editingNurtureTemplateKey.value = template.key || ''
  if (isPresetTemplate(template)) {
    ElMessage.info(t('domain.leads.message.nurtureTemplatePresetInfo'))
    editingNurtureTemplateKey.value = ''
    nurtureForm.name = `${template.name || t('domain.auto.fd7f5e1104')}${t('domain.auto.71d6eea429')}`
  }
}

async function deleteNurtureTemplate(template) {
  if (isPresetTemplate(template)) {
    ElMessage.warning(t('domain.leads.message.nurtureTemplatePresetNotDelete'))
    return
  }
  await api.delete(`/api/v1/prd/leads/nurture/templates/${encodeURIComponent(template.key)}`)
  if (editingNurtureTemplateKey.value === template.key) resetNurtureTemplateForm()
  ElMessage.success(t('domain.leads.message.nurtureTemplateDeleted'))
  await loadNurtureTemplates()
}

async function saveAutomationRules() {
  if (!ensureLeadManagePermission()) return
  const res = await api.post('/api/v1/prd/leads/automation/rules', normalizeAutomationRules(automationRules))
  Object.assign(automationRules, normalizeAutomationRules(res.data?.rules || automationRules))
  ElMessage.success(t('domain.leads.message.automationSaved'))
}

async function dryRunAutomation() {
  if (!ensureLeadManagePermission()) return
  const res = await api.post('/api/v1/prd/flows/leads/automation/run', { dry_run: true, limit: 100, reason: t('domain.auto.f912f96797') })
  automationResult.value = res.data || {}
  ElMessage.success(t('domain.leads.message.automationDryRunDone'))
}

async function runAutomation() {
  if (!ensureLeadManagePermission()) return
  const res = await api.post('/api/v1/prd/flows/leads/automation/run', { dry_run: false, force: true, limit: 100, reason: t('domain.auto.a6973b5f8d') })
  automationResult.value = res.data || {}
  ElMessage.success(t('domain.leads.message.automationRunDone'))
  await refreshAll()
}

async function mergeLead(row) {
  await loadDedupe(row)
  const candidates = dedupe.possible_duplicates?.length ? dedupe.possible_duplicates : dedupe.leads
  const ids = candidates.map((item) => item.id).filter((id) => id !== row.id)
  if (!ids.length) {
    ElMessage.warning(t('domain.leads.message.noMergeCandidates'))
    return
  }
  await api.post(`/api/v1/prd/flows/leads/${row.id}/merge`, { duplicate_ids: ids.slice(0, 3), reason: t('domain.auto.cf047a96b2') })
  ElMessage.success(t('domain.leads.message.merged'))
  await refreshAll()
}

async function batchAssign() {
  if (!ensureLeadManagePermission()) return
  const owner = Number(await promptText(t('domain.auto.4cfbd6e408'), t('domain.auto.4caf0c0f20')))
  if (!owner) return
  for (const row of selectedRows.value.filter((item) => !isTerminal(item))) {
    await api.post(`/api/v1/prd/flows/leads/${row.id}/assign`, { owner_id: owner, reason: t('domain.auto.4cfbd6e408') })
  }
  ElMessage.success(t('domain.leads.message.batchAssignDone'))
  await refreshAll()
}

async function batchInvalidate() {
  if (!ensureLeadManagePermission()) return
  await ElMessageBox.confirm(t('domain.leads.message.confirmBatchInvalid'), t('domain.leads.message.batchInvalidTitle'), { type: 'warning' })
  for (const row of selectedRows.value.filter((item) => !isTerminal(item))) {
    await api.post(`/api/v1/prd/flows/leads/${row.id}/status`, { status: 'invalid', reason: t('domain.auto.76ebdb3f2c') })
  }
  ElMessage.success(t('domain.leads.message.batchInvalidDone'))
  await refreshAll()
}

async function deleteInvalidLead(row) {
  if (row.status !== 'invalid') {
    ElMessage.warning(t('domain.leads.message.onlyDeleteInvalid'))
    return
  }
  await ElMessageBox.confirm(t('domain.leads.message.confirmDeleteInvalidOne').replace('{name}', row.name), t('domain.leads.message.deleteInvalidTitle'), { type: 'warning', confirmButtonText: t('common.action.delete'), cancelButtonText: t('common.action.cancel') })
  await api.delete(`/api/v1/prd/leads/objects/${row.id}`)
  ElMessage.success(t('domain.leads.message.deletedInvalid'))
  detailDrawer.value = false
  await refreshAll()
}

async function batchDeleteInvalid() {
  if (!ensureLeadManagePermission()) return
  if (!selectedInvalidRows.value.length) return
  await ElMessageBox.confirm(t('domain.leads.message.confirmBatchDeleteInvalid').replace('{count}', String(selectedInvalidRows.value.length)), t('domain.leads.message.batchDeleteInvalidTitle'), { type: 'warning', confirmButtonText: t('common.action.delete'), cancelButtonText: t('common.action.cancel') })
  for (const row of selectedInvalidRows.value) {
    await api.delete(`/api/v1/prd/leads/objects/${row.id}`)
  }
  ElMessage.success(t('domain.leads.message.batchDeleteInvalidDone'))
  selectedRows.value = []
  await refreshAll()
}

function openImport() {
  importText.value = JSON.stringify({ items: [] }, null, 2)
  importFileName.value = ''
  importHeaders.value = []
  importRawRows.value = []
  importPreviewRows.value = []
  Object.assign(importMapping, defaultImportMapping())
  if (importFileInput.value) importFileInput.value.value = ''
  importDialog.value = true
}

async function importLeads() {
  if (!importPreviewRows.value.length) parseImportPreview()
  if (!importValidRows.value.length) {
    ElMessage.error(t('domain.leads.message.noValidImportRows'))
    return
  }
  await api.post('/api/v1/prd/leads/objects/import', { items: importValidRows.value.map((row) => row.item) })
  ElMessage.success(t('domain.leads.message.importDone').replace('{valid}', String(importValidRows.value.length)).replace('{invalid}', String(importInvalidRows.value.length)))
  importDialog.value = false
  await refreshAll()
}

async function exportLeads() {
  try {
    const params = cleanParams({ ...filters, page: 1, size: 1000, object_type: 'lead', exclude_status: defaultExcludeStatus() })
    const res = await api.get('/api/v1/prd/leads/objects', { params })
    const items = res.data?.items || []
    downloadExcelTable(leadExportTable(items), `leads-${dateStamp()}.xls`)
    ElMessage.success(`${t('domain.auto.35f670828a')}${items.length}${t('domain.auto.bce2ef6151')}`)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.6753381261')))
  }
}

async function handleImportFile(event) {
  const file = event.target.files?.[0]
  if (!file) return
  importFileName.value = file.name
  importText.value = await file.text()
  parseImportPreview()
  ElMessage.success(t('domain.leads.message.fileLoaded'))
}

function resetFilters(clearView = true) {
  Object.assign(filters, { company_name: '', contact_name: '', keyword: '', source_channel: '', status: '', owner_id: undefined })
  pagination.page = 1
  if (clearView) activeView.value = ''
}

function defaultExcludeStatus() {
  return filters.status ? '' : DAILY_EXCLUDE_STATUSES
}


function defaultForm() {
  return { object_type: 'lead', name: '', code: '', status: 'new', owner_id: 0, company_id: 0, contact_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { company_id: undefined, company_name: '', contact_id: undefined, contact_name: '', phone: '', email: '', source_channel: 'manual', campaign_id: undefined, campaign_name: '', utm_source: '', utm_medium: '', utm_campaign: '', intent_product_id: undefined, intent_product_ids: [], intent_product: '', intent_products: [], temperature: 'cold', visibility: 'public' } }
}
function defaultAutomationRules() {
  return {
    sla: { enabled: true, recycle_hours: 24, limit: 100, statuses: ['new', 'following', 'mql', 'sdr_following'], reason: t('domain.auto.39377a688e') },
    nurture: { enabled: true, template_key: 'standard_education', statuses: ['new', 'mql', 'sdr_following'], score_lt: 80, limit: 100, cooldown_hours: 24, only_without_template: true, reason: t('domain.auto.7ba6067aa6') },
    schedule: { enabled: false, interval_minutes: 30 }
  }
}
function defaultImportMapping() {
  return {
    company_name: 'company_name',
    contact_name: 'contact_name',
    phone: 'phone',
    email: 'email',
    source_channel: 'source_channel',
    intent_product: 'intent_product',
    utm_source: 'utm_source',
    status: 'status',
    owner_id: 'owner_id',
    amount: 'amount',
    tags: 'tags'
  }
}
function normalizeAutomationRules(value) {
  const base = defaultAutomationRules()
  return {
    sla: { ...base.sla, ...(value.sla || {}) },
    nurture: { ...base.nurture, ...(value.nurture || {}) },
    schedule: { ...base.schedule, ...(value.schedule || {}) }
  }
}

function defaultNurtureNodes() {
  return [
    createNurtureNode({ type: 'notice', title: t('domain.auto.21da16ef76'), delay_days: 0 }),
    createNurtureNode({ type: 'task', title: t('domain.auto.5205facb2c'), delay_days: 2 })
  ]
}
function resetNurtureTemplateForm() {
  editingNurtureTemplateKey.value = ''
  Object.assign(nurtureForm, { name: t('domain.auto.54c2287c74'), description: t('domain.auto.30c4361c5d') })
  nurtureNodes.value = defaultNurtureNodes()
}
function createNurtureNode(value = {}) {
  return {
    uid: `${Date.now()}_${Math.random().toString(36).slice(2)}`,
    type: value.type || 'task',
    title: value.title || '',
    delay_days: Number(value.delay_days || 0),
    status: value.status || 'nurturing'
  }
}
function templateNodesForEditor(nodes) {
  const list = Array.isArray(nodes) ? nodes : []
  const mapped = list.map((node) => createNurtureNode(node))
  return mapped.length ? mapped : defaultNurtureNodes()
}
function isPresetTemplate(template) {
  return template?.preset === true || ['standard_education', 'mql_rejected_recycle'].includes(template?.key)
}
function addNurtureNode() {
  nurtureNodes.value.push(createNurtureNode({ type: 'task', title: t('domain.auto.13808b3fc9'), delay_days: 2 }))
}
function removeNurtureNode(index) {
  if (nurtureNodes.value.length <= 1) return
  nurtureNodes.value.splice(index, 1)
}
function moveNurtureNode(index, direction) {
  const target = index + direction
  if (target < 0 || target >= nurtureNodes.value.length) return
  const nodes = [...nurtureNodes.value]
  const [item] = nodes.splice(index, 1)
  nodes.splice(target, 0, item)
  nurtureNodes.value = nodes
}
function normalizeNurtureNode(node) {
  if (node.type === 'notice' && !node.title) node.title = t('domain.auto.9eee9b126b')
  if (node.type === 'task' && !node.title) node.title = t('domain.auto.13808b3fc9')
  if (node.type === 'status') {
    if (!node.title) node.title = t('domain.auto.70aaba128c')
    if (!node.status) node.status = 'nurturing'
  }
}
function nurtureNodeTitlePlaceholder(type) {
  if (type === 'notice') return t('domain.auto.af5c716fe1')
  if (type === 'status') return t('domain.auto.45f0ae3301')
  return t('domain.auto.5121d10c54')
}
function nurtureNodeHelp(node) {
  const days = Number(node.delay_days || 0)
  const when = days === 0 ? t('domain.auto.4a7a8bebcf') : `${days}${t('domain.auto.6f2e559c9e')}`
  if (node.type === 'notice') return `${when}${t('domain.auto.7fd49e27e3')}`
  if (node.type === 'status') return `${when}${t('domain.auto.7015afb8c0')}${statusLabel(node.status)}」。`
  return `${when}${t('domain.auto.257b366d39')}`
}
function serializeNurtureNodes() {
  return nurtureNodes.value
    .map((node) => {
      const type = node.type || 'task'
      const item = {
        type,
        title: String(node.title || '').trim() || nurtureNodeTitlePlaceholder(type).replace(t('domain.auto.e75661aaca'), ''),
        delay_days: Number(node.delay_days || 0)
      }
      if (type === 'status') item.status = node.status || 'nurturing'
      return item
    })
    .filter((node) => node.title)
}

function nurtureTemplateLabel(item) {
  return `${item.name} - ${nurtureTemplateUsage(item)}`
}
function nurtureTemplateUsage(item) {
  if (item?.key === 'standard_education') return t('domain.auto.cd189477bb')
  if (item?.key === 'mql_rejected_recycle') return t('domain.auto.a98b019631')
  return item?.description || t('domain.auto.93e76e05e5')
}

function payload(row) {
  try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} }
}
function objectVisibility(row) { const p = payload(row); return p.visibility === 'private' || p.access_visibility === 'private' || p.private === true || p.private === 'true' ? 'private' : 'public' }
function normalizeDedupeResult(data = {}) {
  const possible = data.possible_duplicates?.length ? data.possible_duplicates : data.leads || []
  return {
    leads: possible,
    possible_duplicates: possible,
    same_contact_leads: data.same_contact_leads || [],
    same_company_leads: data.same_company_leads || [],
    companies: data.companies || [],
    contacts: data.contacts || [],
    match_summary: data.match_summary || {}
  }
}
function resetDedupeState(target) {
  Object.assign(target, normalizeDedupeResult({}))
}
function uniqueLeadRows(items, excludeID = 0) {
  const seen = new Set()
  return (items || []).filter((item) => {
    const id = Number(item?.id || 0)
    if (!id || id === Number(excludeID || 0) || seen.has(id)) return false
    seen.add(id)
    return true
  })
}
function leadDedupePayload(source = {}) {
  return {
    lead_id: source.lead_id,
    company_id: normalizeRecordID(source.company_id),
    contact_id: normalizeRecordID(source.contact_id),
    company_name: source.company_name || source.company,
    contact_name: source.contact_name || source.contact,
    phone: source.phone || source.mobile,
    email: source.email,
    intent_product_id: normalizeProductID(source.intent_product_id || source.product_id),
    intent_product_ids: normalizeProductIDs(source.intent_product_ids || source.product_ids || source.intent_product_id || source.product_id),
    intent_product: source.intent_product || source.product_name,
    source_channel: source.source_channel || source.source,
    utm_source: source.utm_source,
    campaign_id: normalizeRecordID(source.campaign_id),
    external_submission_id: source.external_submission_id || source.submission_id || source.form_submission_id,
    import_batch_id: source.import_batch_id || source.batch_id,
    note: source.note || source.requirement || source.summary
  }
}
function normalizeProductID(value) {
  const id = Number(value || 0)
  return id > 0 ? id : undefined
}
function normalizeProductIDs(value) {
  const raw = Array.isArray(value) ? value : (value ? [value] : [])
  return Array.from(new Set(raw.map((item) => normalizeProductID(item)).filter(Boolean)))
}
function normalizeRecordID(value) {
  const id = Number(value || 0)
  return id > 0 ? id : undefined
}
function companyOptionLabel(item) {
  const p = payload(item)
  return `${item.name}${p.industry ? ` · ${p.industry}` : ''}`
}
function contactOptionLabel(item) {
  const p = payload(item)
  const company = p.company ? ` · ${p.company}` : ''
  const contact = p.phone || p.email ? ` · ${p.phone || p.email}` : ''
  return `${item.name}${company}${contact}`
}
function handleLeadCompanyChange(value) {
  const selectedID = normalizeRecordID(value)
  const company = selectedID
    ? companyOptions.value.find((item) => Number(item.id) === Number(selectedID))
    : null
  if (!company) {
    form.company_id = 0
    delete form.payload.company_id
    form.payload.company_name = String(value || '').trim()
    return
  }
  form.company_id = company.id
  form.payload.company_id = company.id
  form.payload.company_name = company.name
  const selectedContact = contacts.value.find((item) => Number(item.id) === Number(form.payload.contact_id || 0))
  if (selectedContact) {
    const p = payload(selectedContact)
    const contactCompanyID = Number(selectedContact.company_id || p.company_id || 0)
    if (contactCompanyID > 0 && contactCompanyID !== Number(company.id)) {
      form.contact_id = 0
      delete form.payload.contact_id
      form.payload.contact_name = ''
    }
  }
}
function handleLeadContactChange(value) {
  const selectedID = normalizeRecordID(value)
  const contact = selectedID
    ? contactOptions.value.find((item) => Number(item.id) === Number(selectedID))
    : null
  if (!contact) {
    form.contact_id = 0
    delete form.payload.contact_id
    form.payload.contact_name = String(value || '').trim()
    return
  }
  const p = payload(contact)
  form.contact_id = contact.id
  form.payload.contact_id = contact.id
  form.payload.contact_name = contact.name
  if (!form.payload.phone && p.phone) form.payload.phone = p.phone
  if (!form.payload.email && p.email) form.payload.email = p.email
  const contactCompanyID = normalizeRecordID(contact.company_id || p.company_id)
  if (contactCompanyID && !normalizeRecordID(form.payload.company_id)) {
    const company = companies.value.find((item) => Number(item.id) === Number(contactCompanyID))
    if (company) {
      form.company_id = company.id
      form.payload.company_id = company.id
      form.payload.company_name = company.name
    }
  }
  leadFormRef.value?.validateField?.(['payload.phone', 'payload.email'])
}

function handleLeadSourceChange(value) {
  if (value === 'campaign') {
    if (!campaigns.value.length) loadCampaigns()
    return
  }
}

function handleLeadCampaignChange(campaignID) {
  const campaign = campaignOptions.value.find((item) => Number(item.id) === Number(campaignID))
  if (!campaign) {
    delete form.payload.campaign_id
    form.payload.campaign_name = ''
    form.payload.utm_medium = ''
    form.payload.utm_campaign = ''
    return
  }
  applyCampaignToLeadForm(campaign)
}

function applyCampaignToLeadForm(campaign, fallback = {}) {
  if (!campaign?.id) return
  const p = payload(campaign)
  form.payload.source_channel = 'campaign'
  form.payload.campaign_id = campaign.id
  form.payload.campaign_name = campaign.name
  form.payload.utm_source = fallback.utm_source || p.utm_source || p.source_channel || form.payload.utm_source || 'campaign'
  form.payload.utm_medium = fallback.utm_medium || p.utm_medium || form.payload.utm_medium || ''
  form.payload.utm_campaign = fallback.utm_campaign || p.utm_campaign || form.payload.utm_campaign || slugify(campaign.name || `campaign_${campaign.id}`)
  const companyID = normalizeRecordID(fallback.company_id || p.company_id || campaign.company_id)
  const contactID = normalizeRecordID(fallback.contact_id || p.contact_id || campaign.contact_id)
  if (companyID) {
    form.company_id = companyID
    form.payload.company_id = companyID
    const company = companyOptions.value.find((item) => Number(item.id) === Number(companyID))
    if (company) form.payload.company_name = company.name
  }
  if (contactID) {
    form.contact_id = contactID
    form.payload.contact_id = contactID
    const contact = contactOptions.value.find((item) => Number(item.id) === Number(contactID))
    if (contact) {
      form.payload.contact_name = contact.name
      const contactPayload = payload(contact)
      if (!form.payload.phone && contactPayload.phone) form.payload.phone = contactPayload.phone
      if (!form.payload.email && contactPayload.email) form.payload.email = contactPayload.email
    }
  }
}

function campaignOptionLabel(item) {
  const p = payload(item)
  const source = sourceLabel(p.source_channel || p.utm_source)
  return `${item.name}${source && source !== '-' ? ` · ${source}` : ''}`
}

async function applyCampaignQueryCreate() {
  const campaignID = normalizeRecordID(route.query.campaign_id)
  if (!campaignID || editingId.value || formDialog.value) return
  if (!canLeadCreate.value) return
  if (!campaigns.value.length) await loadCampaigns()
  openCreate()
  const fallback = {
    utm_source: String(route.query.utm_source || ''),
    utm_medium: String(route.query.utm_medium || ''),
    utm_campaign: String(route.query.utm_campaign || '')
  }
  let campaign = campaignOptions.value.find((item) => Number(item.id) === Number(campaignID))
  if (!campaign) {
    campaign = {
      id: campaignID,
      name: String(route.query.campaign_name || `${t('domain.auto.aa0ac1530e')}${campaignID}`),
      object_type: 'campaign',
      payload_json: {
        source_channel: route.query.source_channel || 'campaign',
        ...fallback
      }
    }
    campaigns.value = [campaign, ...campaigns.value]
  }
  applyCampaignToLeadForm(campaign, fallback)
  ElMessage.success(t('domain.auto.d487c2d5c1'))
}

function applyCreateQuery() {
  if (route.query.create !== '1' || editingId.value || formDialog.value || !canLeadCreate.value) return
  openCreate()
  if (route.query.source_channel) form.payload.source_channel = String(route.query.source_channel)
}

function productOptionLabel(item) {
  const p = payload(item)
  const sku = p.sku || item.code || ''
  const category = p.category ? ` · ${p.category}` : ''
  return `${item.name}${sku ? `（${sku}）` : ''}${category}`
}
function handleLeadProductChange(productID) {
  const ids = normalizeProductIDs(productID)
  const selectedProducts = ids
    .map((id) => products.value.find((item) => Number(item.id) === Number(id)))
    .filter(Boolean)
  if (!selectedProducts.length) {
    form.payload.intent_product_ids = []
    form.payload.intent_products = []
    form.payload.intent_product = ''
    delete form.payload.intent_product_id
    delete form.payload.product_id
    delete form.payload.product_name
    delete form.payload.product_ids
    delete form.payload.product_names
    return
  }
  form.payload.intent_product_ids = selectedProducts.map((item) => item.id)
  form.payload.intent_products = selectedProducts.map((item) => item.name)
  form.payload.intent_product_id = selectedProducts[0].id
  form.payload.intent_product = selectedProducts.map((item) => item.name).join('、')
  form.payload.product_id = selectedProducts[0].id
  form.payload.product_name = selectedProducts[0].name
  form.payload.product_ids = form.payload.intent_product_ids
  form.payload.product_names = form.payload.intent_products
  leadFormRef.value?.validateField?.('payload.intent_product_ids')
}
function normalizeSelectedLeadProduct() {
  const ids = normalizeProductIDs(form.payload.intent_product_ids || form.payload.product_ids || form.payload.intent_product_id || form.payload.product_id)
  if (ids.length) {
    handleLeadProductChange(ids)
  }
}
function normalizeSelectedLeadCompanyContact() {
  if (normalizeRecordID(form.payload.company_id)) {
    handleLeadCompanyChange(form.payload.company_id)
  } else {
    form.company_id = 0
    delete form.payload.company_id
    form.payload.company_name = String(form.payload.company_name || '').trim()
  }
  if (normalizeRecordID(form.payload.contact_id)) {
    handleLeadContactChange(form.payload.contact_id)
  } else {
    form.contact_id = 0
    delete form.payload.contact_id
    form.payload.contact_name = String(form.payload.contact_name || '').trim()
  }
  form.company_id = normalizeRecordID(form.payload.company_id) || 0
  form.contact_id = normalizeRecordID(form.payload.contact_id) || 0
}
function validateLeadFormRequired() {
  const missing = []
  if (!String(form.payload.company_name || '').trim()) missing.push(t('domain.auto.5d80806d7f'))
  if (!String(form.payload.contact_name || '').trim()) missing.push(t('domain.auto.2425bd4bc1'))
  if (!String(form.payload.phone || '').trim() && !String(form.payload.email || '').trim()) missing.push(t('domain.auto.c86abb1ea4'))
  if (!String(form.payload.source_channel || '').trim()) missing.push(t('domain.auto.c63f79e636'))
  if (form.payload.source_channel === 'campaign' && !normalizeRecordID(form.payload.campaign_id)) missing.push(t('domain.auto.bb002ebde7'))
  if (!normalizeProductIDs(form.payload.intent_product_ids || form.payload.product_ids || form.payload.intent_product_id || form.payload.product_id).length && !String(form.payload.intent_product || '').trim()) missing.push(t('domain.auto.bcb5539330'))
  if (!Number(form.owner_id || 0)) missing.push(t('domain.auto.974d383f36'))
  return { valid: missing.length === 0, missing }
}
function validateLeadCompanyName(rule, value, callback) {
  if (String(form.payload.company_name || '').trim()) callback()
  else callback(new Error(t('domain.auto.d835073d02')))
}
function validateLeadContactName(rule, value, callback) {
  if (String(form.payload.contact_name || '').trim()) callback()
  else callback(new Error(t('domain.auto.918dd5dfd2')))
}
function validateLeadContactMethod(rule, value, callback) {
  if (String(form.payload.phone || '').trim() || String(form.payload.email || '').trim()) callback()
  else callback(new Error(t('domain.auto.11d96957da')))
}
function validateLeadSource(rule, value, callback) {
  if (String(form.payload.source_channel || '').trim()) callback()
  else callback(new Error(t('domain.auto.b187dd1433')))
}
function validateLeadCampaign(rule, value, callback) {
  if (form.payload.source_channel !== 'campaign' || normalizeRecordID(form.payload.campaign_id)) callback()
  else callback(new Error(t('domain.auto.0696d015d6')))
}
function validateLeadIntentProduct(rule, value, callback) {
  if (normalizeProductIDs(form.payload.intent_product_ids || form.payload.product_ids || form.payload.intent_product_id || form.payload.product_id).length || String(form.payload.intent_product || '').trim()) callback()
  else callback(new Error(t('domain.auto.c6e9e511bf')))
}
function validateLeadOwner(rule, value, callback) {
  if (Number(form.owner_id || 0)) callback()
  else callback(new Error(t('domain.auto.8608d655e6')))
}
function actionSuccessMessage(mode) {
  const messages = {
    assign: t('domain.auto.0605156238'),
    convert: t('domain.auto.58c3fda8d3'),
    follow: t('domain.auto.7417999509'),
    nurture: t('domain.auto.62f563c692'),
    status: t('domain.auto.136a9a23ed'),
    release: t('domain.auto.d87e17d3e7')
  }
  return messages[mode] || t('common.message.opSuccess')
}
function actionFailureMessage(mode, error) {
  const messages = {
    assign: t('domain.auto.62628fbd28'),
    convert: t('domain.auto.46d39f8ede'),
    follow: t('domain.auto.173bb97212'),
    nurture: t('domain.auto.581e4a5dfa'),
    status: t('domain.auto.34c5b1b174'),
    release: t('domain.auto.41cea49b24'),
    createLead: t('domain.auto.3196cd2db3'),
    updateLead: t('domain.auto.2b8de3c833')
  }
  const detail = apiErrorMessage(error, '')
  return detail ? `${messages[mode] || t('domain.auto.09e424b5e8')}：${detail}` : (messages[mode] || t('domain.auto.09e424b5e8'))
}
function shouldCollapseLeadFlowDrawer(row, mode) {
  return detailDrawer.value &&
    detailRecord.value &&
    Number(detailRecord.value.id) === Number(row?.id || 0) &&
    ['assign', 'release', 'nurture', 'convert'].includes(mode)
}
function cleanParams(params) {
  return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null))
}
function leadExportTable(items) {
  const headers = [
    t('domain.auto.733eabcea7'),
    t('domain.auto.e1272432a4'),
    t('domain.auto.2425bd4bc1'),
    t('domain.auto.9c01ad0911'),
    t('domain.auto.9ed627bcf6'),
    t('domain.auto.c63f79e636'),
    t('domain.auto.bb002ebde7'),
    t('domain.auto.62e951a692'),
    t('domain.auto.296adf9512'),
    t('domain.auto.2f58566ce0'),
    t('domain.auto.974d383f36'),
    t('domain.auto.bcb5539330'),
    t('domain.auto.093dea88c9')
  ]
  const rows = items.map((row) => {
    const p = payload(row)
    return [
      row.name || '',
      p.company_name || p.company || '',
      p.contact_name || p.contact || '',
      p.phone || '',
      p.email || '',
      sourceLabel(p.source_channel || p.utm_source),
      p.campaign_name || '',
      statusLabel(row.status),
      temperatureLabel(temperature(row)),
      row.score || p.ai_score || 0,
      ownerName(row.owner_id),
      leadIntentProductText(p),
      formatTime(row.updated_at)
    ]
  })
  return { headers, rows }
}
function downloadExcelTable(table, filename) {
  const headerCells = table.headers.map((value) => `<th>${escapeExcelHTML(value)}</th>`).join('')
  const bodyRows = table.rows.map((row) => `<tr>${row.map((value) => `<td>${escapeExcelHTML(value)}</td>`).join('')}</tr>`).join('')
  const html = `<!doctype html><html><head><meta charset="UTF-8"></head><body><table><thead><tr>${headerCells}</tr></thead><tbody>${bodyRows}</tbody></table></body></html>`
  const blob = new Blob([html], { type: 'application/vnd.ms-excel;charset=utf-8' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = filename
  document.body.appendChild(link)
  link.click()
  URL.revokeObjectURL(link.href)
  link.remove()
}
function escapeExcelHTML(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
function dateStamp() {
  const date = new Date()
  return `${date.getFullYear()}${String(date.getMonth() + 1).padStart(2, '0')}${String(date.getDate()).padStart(2, '0')}-${String(date.getHours()).padStart(2, '0')}${String(date.getMinutes()).padStart(2, '0')}${String(date.getSeconds()).padStart(2, '0')}`
}
function normalizeViewFilters(value) {
  const next = { ...value }
  if (Array.isArray(next.status)) next.status = next.status[0]
  if (Array.isArray(next.exclude_status)) next.exclude_status = next.exclude_status.join(',')
  delete next.exclude_status
  delete next.overdue_hours
  return next
}
function visibleColumn(key) { return columnSelections.value.includes(key) }
function ownerName(ownerID) {
  const id = Number(ownerID || 0)
  if (!id) return t('domain.auto.3bb92e6efc')
  const user = users.value.find((item) => Number(item.id) === id)
  return user ? userDisplayName(user) : fallbackOwnerName(id)
}
function userDisplayName(user) {
  const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`
  return user.department_name ? `${label} · ${user.department_name}` : label
}
function fallbackOwnerName(ownerID) {
  const currentUser = getCurrentUser()
  if (Number(currentUser.id || 0) === Number(ownerID || 0)) {
    return currentUser.name || currentUser.username || t('domain.auto.16ce12de7b')
  }
  return t('domain.auto.a3d93a3c7c')
}
function leadViewHelp(view) {
  if (view.key === 'high_value') return t('domain.auto.d420c182d2')
  if (view.key === 'overdue') return t('domain.auto.20273157f1')
  if (view.key === 'pool') return t('domain.auto.927a56f666')
  return leadViewName(view)
}
function leadViewName(view) {
  const key = view?.key
  if (view?.preset === true) {
    if (key === 'today') return t('lead.view.today')
    if (key === 'high_value') return t('lead.view.highValue')
    if (key === 'overdue') return t('lead.view.overdue')
    if (key === 'pool') return t('lead.view.pool')
    if (key === 'sdr') return t('lead.view.sdrFollowing')
  }
  return view?.name || t('domain.auto.26f974dca1')
}
function getCurrentUser() {
  try { return JSON.parse(localStorage.getItem('current_user') || '{}') } catch { return {} }
}
function hasLeadManagePermission() {
  return canLeadUpdate.value || canLeadAssign.value
}
function hasLeadPermission(action) {
  const user = getCurrentUser()
  const permissions = user.permissions || []
  if (user.account_type === 'super_admin' || user.role === 'super_admin' || user.account_type === 'tenant_admin') return true
  if (permissions.includes('*')) return true
  if (permissions.includes(`leads:${action}`)) return true
  if (action !== 'view' && action !== 'manage' && permissions.includes('leads:manage')) return true
  if (action === 'view' && permissions.includes('leads:manage')) return true
  return false
}
function ensureLeadManagePermission() {
  if (canLeadUpdate.value || canLeadAssign.value) return true
  ElMessage.warning(t('domain.leads.message.noManagePermission'))
  return false
}
function loadColumns() {
  const defaults = columnOptions.map((item) => item.key)
  try {
    const stored = JSON.parse(localStorage.getItem('lead_columns') || '[]')
    const allowed = new Set(defaults)
    const normalized = stored.filter((item) => allowed.has(item))
    if (normalized.length && !normalized.includes('campaign')) normalized.splice(Math.min(normalized.indexOf('source') + 1 || normalized.length, normalized.length), 0, 'campaign')
    return normalized.length ? normalized : defaults
  } catch {
    return defaults
  }
}
function saveColumns() { localStorage.setItem('lead_columns', JSON.stringify(columnSelections.value)) }
function resetColumns() { columnSelections.value = columnOptions.map((item) => item.key); saveColumns() }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function isTerminal(row) { return ['converted', 'invalid'].includes(row?.status) }
function statusLabel(value) {
  const map = {
    new: 'domain.leads.status.new',
    pending_assign: 'domain.leads.status.pendingAssign',
    following: 'domain.leads.status.following',
    mql: 'domain.leads.status.mql',
    sdr_following: 'domain.leads.status.sdrFollowing',
    sql: 'domain.leads.status.sql',
    nurturing: 'domain.leads.status.nurturing',
    converted: 'domain.leads.status.converted',
    invalid: 'domain.leads.status.invalid'
  }
  return map[value] ? t(map[value]) : (value || '-')
}
function statusType(value) { return ({ converted: 'success', sql: 'success', mql: 'primary', following: 'primary', nurturing: 'warning', invalid: 'danger' }[value] || 'info') }
function processStatus(row) {
  const p = payload(row)
  const nextAction = p.next_action ? `${t('domain.auto.9c9db830ea')}${p.next_action}` : ''
  if (row.status === 'invalid') return { label: t('domain.auto.93736531ce'), type: 'danger', desc: p.invalid_reason || p.status_reason || t('domain.auto.069987eb1d') }
  if (row.status === 'converted') return { label: t('domain.auto.88592f25a6'), type: 'success', desc: p.converted_at ? `${t('domain.auto.6e8c46ea5a')}${formatTime(p.converted_at)}` : t('domain.auto.a7d9511809') }
  if (isMQLRejectedLead(row)) return { label: t('domain.auto.4b411289ae'), type: 'warning', desc: p.mql_rejected_reason || p.nurture_reason || t('domain.auto.57c3300c85') }
  if (row.status === 'pending_assign' || !Number(row.owner_id || 0)) return { label: t('domain.auto.cb0b7f6d57'), type: 'info', desc: t('domain.auto.2626de10cc') }
  if (p.last_follow_up_at) return { label: t('domain.auto.366e6f2617'), type: 'success', desc: `${formatTime(p.last_follow_up_at)} ${nextAction}`.trim() }
  if (row.status === 'nurturing') return { label: t('domain.auto.129f5a3f5c'), type: 'warning', desc: nextAction || t('domain.auto.b655523669') }
  if (row.status === 'sql') return { label: t('domain.auto.d9fea67ad2'), type: 'success', desc: nextAction || t('domain.auto.37117821ec') }
  if (['following', 'mql', 'sdr_following'].includes(row.status)) return { label: t('domain.auto.21e9c7a50a'), type: 'warning', desc: nextAction || t('domain.auto.c51f255659') }
  return { label: t('domain.auto.453d58eaeb'), type: 'info', desc: nextAction || t('domain.auto.5b50357bf6') }
}
function timelinePayload(item) {
  return payload(item)
}
function timelineTitle(item) {
  const p = timelinePayload(item)
  const event = String(p.event_type || '').trim()
  if (event === 'status_change') return t('domain.auto.12bfbb0aac')
  if (event === 'follow_up') return t('domain.auto.4d7216f529')
  if (event === 'assign') return t('domain.auto.3bfec5e2fc')
  if (event === 'release') return t('domain.auto.9e72d69574')
  if (event === 'nurture_start') return t('domain.auto.5f40216260')
  if (event === 'nurture_exit') return t('domain.auto.d235e8107b')
  if (event === 'convert') return t('domain.auto.b6c8bdd841')
  if (event === 'merge') return t('domain.auto.380e60ddd1')
  return item.name || t('domain.auto.b2554c4d85')
}
function timelineSummary(item) {
  const p = timelinePayload(item)
  return p.summary || p.event_type || '-'
}
function timelineOperator(item) { return timelineOperatorText(item, users.value, formatTime) }
function timelineDetail(item) {
  const p = timelinePayload(item)
  if (p.reason) return `${t('domain.auto.718d786a36')}${p.reason}`
  if (p.content) return `${t('domain.auto.718d786a36')}${p.content}`
  if (p.next_action) return `${t('domain.auto.9c9db830ea')}${p.next_action}`
  return ''
}
function isMQLRejectedLead(row) {
  const p = payload(row)
  return p.mql_rejected === true || p.mql_rejected === 'true' || String(p.nurture_template_key || '').includes('mql_rejected')
}
function mqlRejectedText(row) {
  if (!row || !isMQLRejectedLead(row)) return t('domain.auto.8bf5c10ad9')
  const p = payload(row)
  const at = p.mql_rejected_at ? ` · ${formatTime(p.mql_rejected_at)}` : ''
  return `${p.mql_rejected_reason || p.nurture_reason || t('domain.auto.1db73fbdce')}${at}`
}
function nurtureSourceText(row) {
  if (!row) return '-'
  const p = payload(row)
  if (!p.nurture_template_name && row.status !== 'nurturing') return '-'
  return `${p.nurture_template_name || t('domain.auto.a51208be47')}${p.nurture_reason ? ` · ${p.nurture_reason}` : ''}`
}
function sourceLabel(value) {
  const map = {
    manual: 'domain.leads.source.manual',
    website: 'domain.leads.source.website',
    campaign: 'domain.leads.source.campaign',
    ads: 'domain.leads.source.ads',
    social: 'domain.leads.source.social',
    webinar: 'domain.leads.source.webinar',
    webinar_mfg: 'domain.leads.source.webinarMfg',
    whitepaper: 'domain.leads.source.whitepaper',
    form_submit: 'domain.leads.source.formSubmit'
  }
  const preset = utmOptionGroups.flatMap((group) => group.options).find((item) => item.value === value)
  if (preset) return preset.label
  return map[value] ? t(map[value]) : (value || '-')
}
function slugify(value) {
  const text = String(value || '').trim().toLowerCase()
  const slug = text.replace(/[^a-z0-9\u4e00-\u9fa5]+/gi, '_').replace(/^_+|_+$/g, '')
  return slug || 'campaign'
}
function leadIntentProductText(p = {}) {
  if (Array.isArray(p.intent_products) && p.intent_products.length) return p.intent_products.join('、')
  if (Array.isArray(p.product_names) && p.product_names.length) return p.product_names.join('、')
  return p.intent_product || p.product_name || '-'
}
function leadAvatarText(item) {
  const p = payload(item)
  const name = p.company_name || p.company || p.contact_name || p.contact || item?.name || ''
  return String(name).trim().slice(0, 1).toUpperCase() || 'L'
}
function temperature(row) { return payload(row).temperature || (Number(row.score || 0) >= 75 ? 'hot' : Number(row.score || 0) >= 45 ? 'warm' : 'cold') }
function temperatureLabel(value) {
  const map = { hot: 'domain.leads.temperature.hot', warm: 'domain.leads.temperature.warm', cold: 'domain.leads.temperature.cold' }
  return map[value] ? t(map[value]) : (value || '-')
}
function scoreToTemperature(score) { return Number(score || 0) >= 75 ? 'hot' : Number(score || 0) >= 45 ? 'warm' : 'cold' }
function scoreToTemperatureLabel(score) { return temperatureLabel(scoreToTemperature(score)) }
const chartWidth = (value, max) => `${Math.max(8, Math.round((Number(value || 0) / Number(max || 1)) * 100))}%`
const workloadPercent = (value) => Math.min(100, Math.round((Number(value || 0) / 30) * 100))
function parseImportPayload(text) {
  const raw = String(text || '').trim()
  if (!raw) return { items: [] }
  if (raw.startsWith('{') || raw.startsWith('[')) {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? { items: parsed } : parsed
  }
  const lines = raw.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
  if (lines.length < 2) return { items: [] }
  const headers = splitCSVLine(lines[0])
  const items = lines.slice(1).map((line, index) => {
    const values = splitCSVLine(line)
    const payload = {}
    headers.forEach((header, headerIndex) => { payload[header] = values[headerIndex] || '' })
    const company = payload.company_name || payload.company || `${t('domain.auto.93eaf579c2')}${index + 1}`
    const contact = payload.contact_name || payload.contact || `${t('domain.auto.baad2738dd')}${index + 1}`
    return {
      object_type: 'lead',
      name: `${company}-${contact}`,
      status: payload.status || 'new',
      owner_id: Number(payload.owner_id || 0),
      amount: Number(payload.amount || 0),
      currency: payload.currency || defaultBusinessCurrency(),
      score: Number(payload.score || 0),
      tags: String(payload.tags || '').split('|').filter(Boolean),
      payload
    }
  })
  return { items }
}
function parseImportPreview() {
  const parsed = parseImportSource(importText.value)
  importHeaders.value = parsed.headers
  importRawRows.value = parsed.rows
  autoMapImportHeaders()
  importPreviewRows.value = parsed.rows.map((row, index) => buildImportPreviewRow(row, index))
  ElMessage.success(t('domain.leads.message.importParsed').replace('{valid}', String(importValidRows.value.length)).replace('{invalid}', String(importInvalidRows.value.length)))
}
function parseImportSource(text) {
  const raw = String(text || '').trim()
  if (!raw) return { headers: [], rows: [] }
  if (raw.startsWith('{') || raw.startsWith('[')) {
    const payload = parseImportPayload(raw)
    const rows = (payload.items || []).map((item) => {
      const source = { ...(item.payload || {}) }
      Object.assign(source, {
        company_name: source.company_name || source.company || item.company_name,
        contact_name: source.contact_name || source.contact || item.contact_name,
        phone: source.phone || item.phone,
        email: source.email || item.email,
        source_channel: source.source_channel || item.source_channel,
        intent_product: source.intent_product || item.intent_product,
        utm_source: source.utm_source || item.utm_source,
        status: item.status || source.status,
        owner_id: item.owner_id || source.owner_id,
        amount: item.amount || source.amount,
        tags: Array.isArray(item.tags) ? item.tags.join('|') : item.tags || source.tags
      })
      return source
    })
    return { headers: uniqueHeaders(rows), rows }
  }
  const lines = raw.split(/\r?\n/).filter((line) => line.trim())
  if (lines.length < 2) return { headers: [], rows: [] }
  const delimiter = lines[0].includes('\t') ? '\t' : ','
  const headers = splitDelimitedLine(lines[0], delimiter)
  const rows = lines.slice(1).map((line) => {
    const values = splitDelimitedLine(line, delimiter)
    const row = {}
    headers.forEach((header, headerIndex) => { row[header] = values[headerIndex] || '' })
    return row
  })
  return { headers, rows }
}
function autoMapImportHeaders() {
  const aliases = {
    company_name: ['company_name', 'company', t('domain.auto.e1272432a4'), t('domain.auto.5d80806d7f'), t('domain.auto.3484d8d283')],
    contact_name: ['contact_name', 'contact', 'name', t('domain.auto.2425bd4bc1'), t('domain.auto.be4c2616b1')],
    phone: ['phone', 'mobile', 'tel', t('domain.auto.9c01ad0911'), t('domain.auto.f39c6b629f')],
    email: ['email', 'mail', t('domain.auto.9ed627bcf6')],
    source_channel: ['source_channel', 'source', t('domain.auto.c63f79e636'), t('domain.auto.c152be9f50')],
    intent_product: ['intent_product', 'product', t('domain.auto.bcb5539330')],
    utm_source: ['utm_source', 'utm', t('domain.auto.c138d2d832')],
    status: ['status', t('domain.auto.62e951a692')],
    owner_id: ['owner_id', t('domain.auto.974d383f36'), 'owner'],
    amount: ['amount', t('domain.auto.34943c40c9'), t('domain.auto.0dcf0e012a')],
    tags: ['tags', t('domain.auto.ae0a7afece')]
  }
  for (const field of importFieldOptions) {
    const current = importMapping[field.key]
    if (current && importHeaders.value.includes(current)) continue
    importMapping[field.key] = importHeaders.value.find((header) => aliases[field.key]?.includes(String(header).trim())) || ''
  }
}
function buildImportPreviewRow(sourceRow, index) {
  const mapped = {}
  for (const field of importFieldOptions) {
    const sourceKey = importMapping[field.key]
    mapped[field.key] = sourceKey ? sourceRow[sourceKey] : ''
  }
  const company = String(mapped.company_name || '').trim()
  const contact = String(mapped.contact_name || '').trim()
  const phone = String(mapped.phone || '').trim()
  const email = String(mapped.email || '').trim()
  const errors = []
  if (!company) errors.push(t('domain.auto.7310ad0d4a'))
  if (!contact) errors.push(t('domain.auto.f0286e24fe'))
  if (!phone && !email) errors.push(t('domain.auto.ff7ac8eeb1'))
  if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errors.push(t('domain.auto.dfe47da05a'))
  const matched = matchedABMName(company)
  if (matched) {
    mapped.abm_target_account = true
    mapped.abm_target_account_name = matched
    mapped.abm_matched_at = new Date().toISOString()
  }
  const tags = String(mapped.tags || '').split(/[|,，]/).map((item) => item.trim()).filter(Boolean)
  if (matched && !tags.includes('abm')) tags.push('abm')
  const item = {
    object_type: 'lead',
    name: `${company || `${t('domain.auto.93eaf579c2')}${index + 1}`}-${contact || `${t('domain.auto.baad2738dd')}${index + 1}`}`,
    status: mapped.status || 'new',
    owner_id: Number(mapped.owner_id || 0),
    amount: Number(mapped.amount || 0),
    currency: defaultBusinessCurrency(),
    score: 0,
    tags,
    payload: { ...mapped, source_channel: mapped.source_channel || 'manual' }
  }
  return { line: index + 2, ...mapped, errors, valid: errors.length === 0, item }
}
function uniqueHeaders(rows) {
  return Array.from(rows.reduce((set, row) => {
    Object.keys(row || {}).forEach((key) => set.add(key))
    return set
  }, new Set()))
}
function splitCSVLine(line) {
  return splitDelimitedLine(line, ',')
}
function splitDelimitedLine(line, delimiter) {
  const result = []
  let current = ''
  let quoted = false
  for (const char of line) {
    if (char === '"') {
      quoted = !quoted
    } else if (char === delimiter && !quoted) {
      result.push(current.trim())
      current = ''
    } else {
      current += char
    }
  }
  result.push(current.trim())
  return result
}
function loadABMTargets() {
  try {
    return JSON.parse(localStorage.getItem('lead_abm_targets') || '[]')
  } catch {
    return []
  }
}
function matchedABMName(companyName) {
  const company = String(companyName || '').trim().toLowerCase()
  if (!company) return ''
  return abmTargets.value.find((target) => {
    const normalized = String(target || '').trim().toLowerCase()
    return normalized && (company.includes(normalized) || normalized.includes(company))
  }) || ''
}
function matchedABMTarget(row) {
  const p = payload(row)
  return matchedABMName(p.company_name || p.company || row.name)
}
function isABMLead(row) {
  const p = payload(row)
  return Boolean(p.abm_target_account || splitTags(row?.tags).includes('abm') || matchedABMTarget(row))
}
function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value).replace('T', ' ').slice(0, 19)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
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
:deep(.lead-create-form .el-form-item__label) {
  min-height: 20px;
  height: auto;
  line-height: 1.35;
  white-space: normal;
  overflow-wrap: anywhere;
}
</style>
