<template>
  <div class="system-page crm-flow-page">
    <section class="crm-help-strip">
      <span>{{ t('domain.auto.cf87b04d03') }}</span>
    </section>

    <section class="system-card">
      <div class="system-toolbar">
        <strong>{{ t('domain.auto.d23768c1b1') }}</strong>
        <el-tag size="small">{{ boardRows.length }} {{ t('domain.auto.bce2ef6151') }}</el-tag>
      </div>
      <div class="opportunity-kanban-board">
        <div
          v-for="stage in boardStages"
          :key="stage.value"
          class="opportunity-kanban-column"
          :class="{ 'is-drag-over': dragOverStage === stage.value }"
          :style="stageStyle(stage.value)"
          @dragover.prevent="handleOpportunityDragOver(stage.value, $event)"
          @dragleave="handleOpportunityDragLeave(stage.value, $event)"
          @drop.prevent="dropOpportunityOnStage(stage.value)"
        >
          <div class="opportunity-kanban-head">
            <div>
              <strong>{{ stage.label }}</strong>
              <small>{{ stageOrderProgress(stage.value) }} {{ t('domain.auto.27643a1ee6') }} {{ stage.followDays }} {{ t('domain.auto.5672e440f1') }}</small>
            </div>
            <span>{{ stageRows(stage.value).length }} {{ t('domain.auto.e2a639e9c2') }} {{ formatAmount(stageAmount(stage.value)) }}</span>
          </div>
          <div class="opportunity-kanban-list">
            <button
              v-for="item in stageRows(stage.value)"
              :key="item.id"
              class="opportunity-kanban-card"
              :class="[`follow-${followStatus(item)}`, { 'is-dragging': Number(draggingOpportunityId) === Number(item.id), 'is-draggable': canManage }]"
              :draggable="canManage"
              @click="handleKanbanCardClick(item)"
              @dragstart.stop="startOpportunityDrag(item, $event)"
              @dragend="endOpportunityDrag"
            >
              <div class="kanban-card-title">
                <strong>{{ item.name }} <el-tag v-if="objectVisibility(item) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
                <el-tag size="small" :type="followStatusType(item)">{{ followStatusLabel(item) }}</el-tag>
              </div>
              <span>{{ payload(item).company || '-' }}</span>
              <div class="stage-progress-mini">
                <el-progress :percentage="Number(payload(item).probability || probability(item.status))" :show-text="false" :color="stageMeta(item.status).color" />
                <small>{{ stageOrderProgress(item.status) }} {{ t('domain.auto.8387f8da56') }} {{ stageStayDays(item) }} {{ t('domain.auto.c3304d1e49') }}</small>
              </div>
              <em>{{ formatAmount(item.amount, item.currency) }}</em>
              <small class="follow-due-text">{{ followDueText(item) }}</small>
            </button>
            <el-empty v-if="!stageRows(stage.value).length" :description="emptyStageText(stage)" />
          </div>
          <div class="opportunity-kanban-actions">
            <el-button v-if="canManage && nextStage(stage.value)" size="small" plain :disabled="!stageRows(stage.value).length" @click="quickAdvanceStage(stage.value)">
              {{ t('domain.auto.272c787ab7') }}{{ stageLabel(nextStage(stage.value)) }}
            </el-button>
          </div>
        </div>
      </div>
    </section>

    <section class="system-card">
      <div class="list-card-search">
        <el-form :model="filters" label-width="72px" class="system-search-form">
          <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.6502294e89')" clearable /></el-form-item>
          <el-form-item :label="t('domain.auto.4ca39faad0')">
            <el-select v-model="filters.status" clearable :placeholder="t('domain.auto.7be5dfee1a')">
              <el-option v-for="item in stageOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('domain.auto.974d383f36')">
            <el-select v-model="filters.owner_id" clearable filterable :placeholder="t('domain.auto.8b134548e7')">
              <el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('opportunity.reliability.label')">
            <el-select v-model="filters.reliability" clearable :placeholder="t('opportunity.reliability.all')">
              <el-option v-for="item in reliabilityOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('opportunity.importance.label')">
            <el-select v-model="filters.importance" clearable :placeholder="t('opportunity.importance.all')">
              <el-option v-for="item in importanceOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <div class="system-search-actions">
            <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
            <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
          </div>
        </el-form>
      </div>
      <div class="crm-stage-board">
        <button class="crm-stage-card opportunity-stage-card" :class="{ active: !filters.status }" @click="quickStage('')">
          <strong>{{ t('domain.auto.f6caea5c68') }}</strong>
          <span>{{ t('domain.auto.1921fef7f7') }}</span>
          <span>{{ t('domain.auto.c7ab5490b1') }}</span>
          <em>{{ rows.length }} {{ t('domain.auto.747c3b8ead') }} {{ formatAmount(rows.reduce((sum, item) => sum + Number(item.amount || 0), 0)) }}</em>
        </button>
        <button v-for="stage in stageOptions" :key="stage.value" class="crm-stage-card opportunity-stage-card" :class="{ active: filters.status === stage.value }" :style="stageStyle(stage.value)" @click="quickStage(stage.value)">
          <strong>{{ stage.label }}</strong>
          <span>{{ stage.desc }}</span>
          <span>{{ stageOrderProgress(stage.value) }} · {{ stage.probability }}%</span>
          <em>{{ stageCount(stage.value) }} {{ t('domain.auto.747c3b8ead') }} {{ formatAmount(stageAmount(stage.value)) }}</em>
        </button>
      </div>
      <div class="system-toolbar">
        <div class="toolbar-left">
          <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.dd43bf07e7') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('stage')">{{ t('domain.auto.74984d48e8') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('create_quote')">{{ t('domain.auto.7e37e9b4fb') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="success" plain @click="openAction('win')">{{ t('opportunity.action.winToDeal') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="danger" plain @click="openAction('lost')">{{ t('domain.auto.4dbed6b418') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="success" plain @click="openReview('won')">{{ t('opportunity.review.type.won') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="warning" plain @click="openReview('lost')">{{ t('opportunity.review.type.lost') }}</el-button>
        </div>
        <div class="toolbar-icons">
          <el-tooltip :content="t('domain.auto.38108eaa1d')"><button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage"><Icon icon="ri:refresh-line" /></button></el-tooltip>
        </div>
      </div>

      <el-table :data="rows" class="system-table opportunity-list-table" row-key="id" v-loading="loading" :row-class-name="opportunityRowClass">
        <el-table-column prop="name" :label="t('domain.auto.c4beee8569')" min-width="210">
          <template #default="{ row }">
            <div class="record-title">
              <strong>{{ row.name }} <el-tag v-if="objectVisibility(row) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
              <span>{{ payload(row).company || `${t('domain.auto.f206870601')}#${row.company_id || '-'}` }} · {{ payload(row).contact_name || payload(row).contact || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('domain.auto.4ca39faad0')" width="150">
          <template #default="{ row }">
            <el-tag effect="dark" :color="stageMeta(row.status).color">{{ stageLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('domain.auto.47b6ae52a1')" width="110"><template #default="{ row }"><el-progress :percentage="Number(payload(row).probability || probability(row.status))" :show-text="false" /><span class="score-text">{{ payload(row).probability || probability(row.status) }}%</span></template></el-table-column>
        <el-table-column :label="t('opportunity.reliability.label')" width="110"><template #default="{ row }"><el-tag :type="reliabilityType(payload(row).reliability)">{{ reliabilityLabel(payload(row).reliability) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('opportunity.importance.label')" width="120"><template #default="{ row }"><el-tag :type="importanceType(payload(row).importance)">{{ importanceLabel(payload(row).importance) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('domain.auto.acf014bff8')" width="120"><template #default="{ row }">{{ stageOrderProgress(row.status) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.991fe32fdd')" width="150">
          <template #default="{ row }">
            <el-tag :type="followStatusType(row)" size="small">{{ followStatusLabel(row) }}</el-tag>
            <span class="table-subtext">{{ followDueText(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('domain.auto.34943c40c9')" width="140"><template #default="{ row }">{{ formatAmount(row.amount, row.currency) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.974d383f36')" width="140"><template #default="{ row }">{{ ownerName(row.owner_id) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.1385302ac8')" width="130"><template #default="{ row }">{{ payload(row).expected_close_date || '-' }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.f3ea6d345e')" width="190" fixed="right">
          <template #default="{ row }">
            <div class="row-actions three" @click.stop>
              <button class="action-btn view" @click="openDetail(row)"><Icon icon="ri:eye-line" /></button>
              <button v-if="canEditRecord(row)" class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
              <button v-if="canEditRecord(row)" class="action-btn success" @click="openAction('stage', row)"><Icon icon="ri:arrow-right-circle-line" /></button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="system-pagination">
        <span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span>
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
      </div>
    </section>

    <el-drawer v-model="detailDrawer" size="58%" :title="selectedRecord?.name || t('domain.auto.620f4ee92d')" destroy-on-close>
      <div v-if="selectedRecord" class="lead-detail opportunity-detail">
        <section class="opportunity-profile" :style="stageStyle(selectedRecord.status)">
          <div class="opportunity-profile-main">
            <div>
              <span class="section-eyebrow">{{ t('domain.auto.353c088892') }}</span>
              <h2>{{ selectedRecord.name }}</h2>
              <div class="opportunity-stage-line">
                <el-tag effect="dark" :color="stageMeta(selectedRecord.status).color" class="opportunity-stage-badge">{{ stageLabel(selectedRecord.status) }}</el-tag>
                <span>{{ stageOrderProgress(selectedRecord.status) }} {{ t('domain.auto.e198f7c484') }} {{ stageStayDays(selectedRecord) }} {{ t('domain.auto.c3304d1e49') }}</span>
              </div>
            </div>
            <div class="opportunity-probability-card">
              <span>{{ t('domain.auto.7b3b3c954c') }}</span>
              <strong>{{ selectedProbability.value }}%</strong>
              <em>{{ selectedProbability.label }}</em>
            </div>
          </div>
          <el-progress :percentage="Number(selectedProbability.value)" :color="stageMeta(selectedRecord.status).color" />
          <div class="opportunity-profile-grid">
            <div>
              <span>{{ t('domain.auto.f206870601') }}</span>
              <strong>{{ selectedCustomer.name }}</strong>
              <small v-if="selectedCustomer.needsEdit">{{ t('domain.auto.dee6413190') }}</small>
            </div>
            <div>
              <span>{{ t('domain.auto.2425bd4bc1') }}</span>
              <strong>{{ selectedContact.name }}</strong>
              <small v-if="selectedContact.needsEdit">{{ t('domain.auto.f464f80908') }}</small>
            </div>
            <div>
              <span>{{ t('domain.auto.1385302ac8') }}</span>
              <strong>{{ expectedCloseLabel(selectedRecord) }}</strong>
              <small v-if="!selectedPayload.expected_close_date">{{ t('domain.auto.2c554f5299') }}</small>
            </div>
            <div>
              <span>{{ t('domain.auto.670248ff35') }}</span>
              <strong>{{ formatAmount(selectedRecord.amount, selectedRecord.currency) }}</strong>
            </div>
            <div>
              <span>{{ t('domain.auto.974d383f36') }}</span>
              <strong>{{ ownerName(selectedRecord.owner_id) }}</strong>
            </div>
            <div>
              <span>{{ t('domain.auto.77c07753b9') }}</span>
              <strong>{{ productSummary(selectedRecord) }}</strong>
              <small v-if="!lineItems(selectedRecord).length">{{ t('domain.auto.2f97cf19dd') }}</small>
            </div>
          </div>
        </section>
        <section v-if="canFollowRecord(selectedRecord) || canEditRecord(selectedRecord)" class="opportunity-action-board">
          <div class="opportunity-action-group">
            <span>{{ t('domain.auto.b89802ab99') }}</span>
            <el-button v-if="canEditRecord(selectedRecord)" @click="openEdit(selectedRecord)">{{ t('domain.auto.7719bdd192') }}</el-button>
          </div>
          <div class="opportunity-action-group">
            <span>{{ t('domain.auto.1529d8a618') }}</span>
            <el-button v-if="canFollowRecord(selectedRecord)" type="primary" plain @click="openAction('follow_up')">{{ t('domain.auto.a68cc3cf80') }}</el-button>
            <el-button v-if="canFollowRecord(selectedRecord)" plain @click="openAction('reminder')">{{ t('domain.auto.c07f8b771c') }}</el-button>
          </div>
          <div class="opportunity-action-group">
            <span>{{ t('domain.auto.47437b8912') }}</span>
            <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('create_quote')">{{ t('domain.auto.7e37e9b4fb') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('stage')">{{ t('domain.auto.74984d48e8') }}</el-button>
          </div>
          <div class="opportunity-action-group">
            <span>{{ t('domain.auto.26c2a55110') }}</span>
            <el-button v-if="canEditRecord(selectedRecord)" type="success" plain @click="openAction('win')">{{ t('opportunity.action.winToDeal') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" type="success" plain @click="openReview('won')">{{ t('opportunity.review.type.won') }}</el-button>
          </div>
          <div class="opportunity-action-group danger">
            <span>{{ t('domain.auto.bf79cedb71') }}</span>
            <el-button v-if="canEditRecord(selectedRecord)" type="danger" plain @click="openAction('lost')">{{ t('domain.auto.4dbed6b418') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" type="warning" plain @click="openReview('lost')">{{ t('opportunity.review.type.lost') }}</el-button>
          </div>
        </section>
        <el-divider>{{ t('opportunity.review.title') }}</el-divider>
        <section class="opportunity-review-panel">
          <div v-for="type in reviewTypes" :key="type" class="opportunity-review-card">
            <div class="opportunity-review-head">
              <div>
                <strong>{{ reviewFor(type)?.name || `${reviewTypeLabel(type)}: ${selectedRecord.name}` }}</strong>
                <span>{{ reviewFor(type) ? `${reviewTypeLabel(type)} · ${reviewStatusLabel(reviewFor(type).status)}` : t('opportunity.review.emptyHint') }}</span>
              </div>
              <div class="opportunity-review-actions">
                <el-tag v-if="reviewFor(type)" :type="reviewStatusType(reviewFor(type).status)">{{ reviewStatusLabel(reviewFor(type).status) }}</el-tag>
                <el-button v-if="canEditRecord(selectedRecord)" size="small" plain @click="openReview(type)">{{ reviewActionLabel(type) }}</el-button>
                <el-button v-if="canEditRecord(selectedRecord) && reviewFor(type)?.status === 'draft'" size="small" type="primary" plain :loading="reviewSubmitting && activeReviewType === type" @click="submitReview(type)">{{ t('opportunity.review.submit') }}</el-button>
                <el-button v-if="canEditRecord(selectedRecord) && reviewFor(type)?.status === 'pending_review'" size="small" type="success" plain @click="publishReview(type)">{{ t('opportunity.review.publish') }}</el-button>
                <el-button v-if="canEditRecord(selectedRecord) && reviewFor(type) && reviewFor(type).status !== 'archived'" size="small" type="danger" plain @click="archiveReview(type)">{{ t('opportunity.review.archive') }}</el-button>
              </div>
            </div>
            <div v-if="reviewFor(type)" class="opportunity-review-summary">
              <p>{{ reviewPayloadFor(type).summary || t('opportunity.review.noSummary') }}</p>
              <div>
                <span>{{ t('opportunity.review.coreReason') }}</span>
                <strong>{{ type === 'lost' ? (reviewPayloadFor(type).loss_reasons || '-') : (reviewPayloadFor(type).win_reasons || '-') }}</strong>
              </div>
              <div>
                <span>{{ t('opportunity.review.reusableExperience') }}</span>
                <strong>{{ reviewPayloadFor(type).reusable_playbook || '-' }}</strong>
              </div>
            </div>
          </div>
        </section>
        <el-divider>{{ t('domain.auto.60b1bf5a84') }}</el-divider>
        <div class="opportunity-follow-panel" :style="stageStyle(selectedRecord.status)">
          <div class="follow-progress-head">
            <div>
              <span>{{ t('domain.auto.0a2489f651') }}</span>
              <strong>{{ stageLabel(selectedRecord.status) }} · {{ stageOrderProgress(selectedRecord.status) }}</strong>
            </div>
            <el-tag :type="followStatusType(selectedRecord)">{{ followStatusLabel(selectedRecord) }}</el-tag>
          </div>
          <el-progress :percentage="Number(selectedPayload.probability || probability(selectedRecord.status))" :color="stageMeta(selectedRecord.status).color" />
          <div class="opportunity-follow-grid">
            <div><span>{{ t('domain.auto.f3eb26da23') }}</span><strong>{{ stageStayDays(selectedRecord) }} {{ t('domain.auto.c3304d1e49') }}</strong></div>
            <div><span>{{ t('domain.auto.bc30fdc074') }}</span><strong>{{ selectedPayload.last_follow_up || '-' }}</strong></div>
            <div><span>{{ t('domain.auto.908c160eb0') }}</span><strong>{{ formatTime(selectedPayload.last_follow_up_at || selectedPayload.last_follow_at) }}</strong></div>
            <div><span>{{ t('domain.auto.ec46366045') }}</span><strong>{{ formatTime(selectedPayload.next_follow_at) }}</strong></div>
            <div class="wide"><span>{{ t('domain.auto.ab2583981e') }}</span><strong>{{ selectedPayload.next_action || '-' }}</strong></div>
          </div>
        </div>
        <ObjectAccessPanel
          domain="opportunities"
          :record="selectedRecord"
          :users="users"
          :can-manage="canManageRecordACL(selectedRecord)"
          :object-label="t('domain.auto.c4beee8569')"
          @updated="refreshSelected"
        />
        <el-divider>{{ t('domain.auto.6c32f3a7db') }}> {{ t('domain.auto.c472bcaac1') }}> {{ t('domain.auto.27190f569d') }}> {{ t('domain.auto.a6eb6863ba') }}> {{ t('domain.auto.58784904ad') }}> {{ t('domain.auto.50b07f3dfa') }}</el-divider>
        <DealChainPanel
          object-type="opportunities"
          :object-id="selectedRecord.id"
          :users="users"
          current-step="opportunity"
          :empty-title="t('domain.auto.2ef64e7fe5')"
          :empty-description="t('domain.auto.29e0848f92')"
          show-opportunity-actions
          @create-quote="openAction('create_quote')"
          @win-order="openAction('win')"
        />
        <el-divider>{{ t('domain.auto.9d357e1b4d') }}</el-divider>
        <div class="crm-line-list opportunity-readiness-list">
          <div v-for="item in stageGateChecks(selectedRecord)" :key="item.key" class="crm-line-item quote-line readiness-item">
            <div>
              <strong>{{ item.title }}</strong>
              <span>{{ item.hint }}</span>
            </div>
            <div class="readiness-actions">
              <el-tag :type="item.pass ? 'success' : 'warning'">{{ item.pass ? t('domain.auto.f3b1db9b14') : t('domain.auto.0c426a865f') }}</el-tag>
              <el-button v-if="!item.pass && item.action" size="small" plain @click="handleGateAction(item)">{{ t('domain.auto.9bbfee41bf') }}</el-button>
            </div>
          </div>
        </div>
        <el-divider>{{ t('domain.auto.e101d2d7c9') }}</el-divider>
        <div class="crm-line-list">
          <div v-for="(item, index) in lineItems(selectedRecord)" :key="index" class="crm-line-item">
            <strong>{{ item.name || item.product_name || t('domain.auto.4b9c8904e9') }}</strong>
            <span>{{ item.quantity || 1 }} x {{ formatAmount(item.unit_price || item.price || 0, selectedRecord.currency) }} {{ t('domain.auto.26efe5f357') }} {{ item.discount || 0 }}%</span>
          </div>
        </div>
        <el-empty v-if="!lineItems(selectedRecord).length" :description="t('domain.auto.dec76ed0a2')" />
        <el-divider>{{ t('domain.auto.df466d27fa') }}</el-divider>
        <p class="crm-inline-help">{{ t('domain.auto.dd712bdeac') }}</p>
        <div class="crm-line-list">
          <div v-for="quote in quotes" :key="quote.id" class="crm-line-item quote-line">
            <div>
              <strong>{{ quote.code || quote.name }}</strong>
              <span>{{ quoteStatusLabel(quote.status) }} · {{ formatAmount(quote.amount, quote.currency || selectedRecord.currency) }} {{ t('domain.auto.aa933a7143') }} {{ payload(quote).valid_until || '-' }}</span>
            </div>
            <div class="quote-actions">
              <el-button v-if="canEditRecord(selectedRecord)" size="small" :disabled="quote.status !== 'draft'" @click="openQuoteAction('quote_send', quote)">{{ t('domain.auto.1214d633a4') }}</el-button>
              <el-button v-if="canEditRecord(selectedRecord)" size="small" :disabled="quote.status !== 'sent'" @click="openQuoteAction('quote_accept', quote)">{{ t('domain.auto.f4ee6d92ad') }}</el-button>
              <el-button v-if="canEditRecord(selectedRecord)" size="small" type="primary" plain :disabled="quote.status !== 'accepted'" @click="openQuoteAction('quote_order', quote)">{{ t('domain.auto.609f00d862') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-if="!quotes.length" :description="t('domain.auto.871da3d064')" />
        <el-divider>{{ t('domain.auto.4a3aec00b1') }}</el-divider>
        <el-timeline>
          <el-timeline-item v-for="item in timeline" :key="item.id" :timestamp="formatTime(item.created_at)">
            <strong>{{ timelineTitle(item) }}</strong>
            <small class="timeline-operator">{{ timelineOperator(item) }}</small>
            <p>{{ payload(item).reason || payload(item).summary || '-' }}</p>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-if="!timeline.length" :description="t('domain.auto.52370dd460')" />
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.d91cda1cc4') : t('domain.auto.dd43bf07e7')" width="min(980px, 96vw)" destroy-on-close>
      <el-form :model="form" label-width="96px" class="lead-dialog-form">
        <el-alert v-if="conflictWarnings.length" type="warning" show-icon :closable="false" class="contact-duplicate-alert">
          <template #title>{{ t('domain.auto.023fb33d33') }}</template>
          <div class="duplicate-list">
            <div v-for="item in conflictWarnings" :key="item.id" class="duplicate-item">
              <span>{{ item.name }} · {{ stageLabel(item.status) }} · {{ formatAmount(item.amount, item.currency) }}</span>
              <button class="action-btn view" @click="openConflict(item)"><Icon icon="ri:eye-line" /></button>
            </div>
          </div>
        </el-alert>
        <el-form-item :label="t('domain.auto.84b592481d')" required><el-input v-model="form.name" /></el-form-item>
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
        <el-form-item :label="t('domain.auto.bb002ebde7')">
          <el-select v-model="form.payload.campaign_id" class="full-input" clearable filterable :placeholder="t('domain.auto.897855e43f')" @change="handleCampaignChange">
            <el-option v-for="item in campaignOptions" :key="item.id" :label="campaignOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.4ca39faad0')"><el-select v-model="form.status"><el-option v-for="item in stageOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.34943c40c9')"><el-input-number v-model="form.amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.a81ab5e100')"><el-select v-model="form.currency" filterable><el-option v-for="item in currencyOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="form.owner_id" clearable filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('opportunity.reliability.label')"><el-select v-model="form.payload.reliability" class="full-input"><el-option v-for="item in reliabilityOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('opportunity.importance.label')"><el-select v-model="form.payload.importance" class="full-input"><el-option v-for="item in importanceOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.241d8ef92f')"><ObjectAccessField v-model="form.payload.visibility" :public-label="t('domain.auto.20e5d5252b')" :private-label="t('domain.auto.30628c2af4')" /></el-form-item>
        <el-form-item :label="t('domain.auto.1385302ac8')" :class="{ 'focus-field': editFocusSection === 'expected_close' }"><el-date-picker v-model="form.payload.expected_close_date" value-format="YYYY-MM-DD" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.927c11a37a')" class="wide-field" :class="{ 'focus-field': editFocusSection === 'line_items' }">
          <div class="visual-step-list opportunity-line-list">
            <div v-if="form.payload.line_items.length" class="opportunity-line-head">
              <span>{{ t('opportunity.line.product') }}</span>
              <span>{{ t('opportunity.line.name') }}</span>
              <span>{{ t('opportunity.line.quantity') }}</span>
              <span>{{ t('opportunity.line.unitPrice') }}</span>
              <span>{{ t('opportunity.line.stock') }}</span>
              <span>{{ t('opportunity.line.action') }}</span>
            </div>
            <div v-for="(item, index) in form.payload.line_items" :key="index" class="visual-step-row opportunity-line-editor">
              <el-select v-model="item.product_id" class="opportunity-line-control product" filterable clearable :placeholder="t('domain.auto.2c30787264')" @change="applyProductToLine(item)">
                <el-option v-for="product in products" :key="product.id" :label="product.name" :value="product.id">
                  <span>{{ product.name }}</span>
                  <small class="product-option-meta">{{ productInfo(product).sku || product.code || '-' }} {{ t('domain.auto.95e7b3fabb') }} {{ productInfo(product).stock || 0 }}</small>
                </el-option>
              </el-select>
              <el-input v-model="item.name" class="opportunity-line-control name" :placeholder="t('domain.auto.cbcc51df5c')" />
              <div class="opportunity-line-control compact" :data-label="t('opportunity.line.quantity')">
                <el-input-number v-model="item.quantity" :min="1" :aria-label="t('opportunity.line.quantity')" controls-position="right" />
              </div>
              <div class="opportunity-line-control compact" :data-label="t('opportunity.line.unitPrice')">
                <el-input-number v-model="item.unit_price" :min="0" :aria-label="t('opportunity.line.unitPrice')" controls-position="right" />
              </div>
              <span class="product-stock-note opportunity-line-stock" :data-label="t('opportunity.line.stock')">{{ item.stock_available ?? '-' }}</span>
              <el-button text type="danger" class="opportunity-line-delete" @click="removeLine(index)">{{ t('domain.auto.3755f56f2f') }}</el-button>
            </div>
            <el-button plain @click="addLine">{{ t('domain.auto.20d602435a') }}</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('domain.auto.fadf24dbc5') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="actionDialog" :title="actionTitle" width="620px" destroy-on-close>
      <el-form :model="actionForm" label-width="110px">
        <el-form-item v-if="actionType === 'stage'" :label="t('domain.auto.fc90b3e81d')">
          <el-select v-model="actionForm.stage" class="full-input">
            <el-option v-for="item in nextStageOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'lost'" :label="t('domain.auto.26d71fa296')" required>
          <el-select v-model="actionForm.loss_reason" class="full-input">
            <el-option :label="t('domain.auto.b70c2d28a1')" :value="t('domain.auto.b70c2d28a1')" />
            <el-option :label="t('domain.auto.b43eef8bec')" :value="t('domain.auto.b43eef8bec')" />
            <el-option :label="t('domain.auto.565cf99934')" :value="t('domain.auto.565cf99934')" />
            <el-option :label="t('domain.auto.48b3cd4fd1')" :value="t('domain.auto.48b3cd4fd1')" />
            <el-option :label="t('domain.auto.1a26edf94a')" :value="t('domain.auto.1a26edf94a')" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'lost'" :label="t('domain.auto.7029b02109')" required><el-input v-model="actionForm.competitor" /></el-form-item>
        <el-form-item v-if="actionType === 'create_quote'" :label="t('domain.auto.2c86fc7fb8')"><el-input-number v-model="actionForm.quote_amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item v-if="actionType === 'create_quote'" :label="t('domain.auto.222460c095')"><el-date-picker v-model="actionForm.valid_until" value-format="YYYY-MM-DD" class="full-input" /></el-form-item>
        <p v-if="actionType === 'create_quote'" class="crm-dialog-help">{{ t('domain.auto.eeef829188') }}</p>
        <el-form-item v-if="actionType === 'follow_up'" :label="t('domain.auto.ec46366045')"><el-date-picker v-model="actionForm.next_follow_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" /></el-form-item>
        <el-form-item v-if="actionType === 'follow_up'" :label="t('domain.auto.ab2583981e')"><el-input v-model="actionForm.next_action" :placeholder="t('domain.auto.712fabfa21')" /></el-form-item>
        <el-form-item v-if="actionType === 'reminder'" :label="t('domain.auto.19560fc10f')"><el-input v-model="actionForm.title" /></el-form-item>
        <el-form-item v-if="actionType === 'reminder'" :label="t('domain.auto.4f36c3b75a')"><el-date-picker v-model="actionForm.remind_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" /></el-form-item>
        <el-form-item v-if="actionType === 'reminder'" :label="t('domain.auto.d07f9b994c')"><el-select v-model="actionForm.owner_ids" multiple filterable clearable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.cd612544e1')" required><el-input v-model="actionForm.reason" type="textarea" :rows="4" :placeholder="actionPlaceholder" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitAction">{{ t('domain.auto.f526c89937') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reviewDialog" :title="reviewTypeLabel(reviewForm.review_type)" width="920px" destroy-on-close>
      <el-form :model="reviewForm" label-width="110px" class="lead-dialog-form review-dialog-form">
        <el-form-item :label="t('opportunity.review.field.title')" required><el-input v-model="reviewForm.title" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.type')"><el-select v-model="reviewForm.review_type" class="full-input" disabled><el-option :label="t('opportunity.review.type.won')" value="won" /><el-option :label="t('opportunity.review.type.lost')" value="lost" /></el-select></el-form-item>
        <el-form-item :label="t('opportunity.review.field.industry')"><el-input v-model="reviewForm.industry" :disabled="reviewReadonly" :placeholder="t('opportunity.review.placeholder.industry')" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.competitor')"><el-input v-model="reviewForm.competitor" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.summary')" class="wide-field"><el-input v-model="reviewForm.summary" type="textarea" :rows="3" :disabled="reviewReadonly" :placeholder="t('opportunity.review.placeholder.summary')" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.background')" class="wide-field"><el-input v-model="reviewForm.background" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.customerNeeds')" class="wide-field"><el-input v-model="reviewForm.customer_needs" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.decisionProcess')" class="wide-field"><el-input v-model="reviewForm.decision_process" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.keyActions')" class="wide-field"><el-input v-model="reviewForm.key_actions" type="textarea" :rows="3" :disabled="reviewReadonly" :placeholder="t('opportunity.review.placeholder.keyActions')" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.obstacles')" class="wide-field"><el-input v-model="reviewForm.obstacles" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item v-if="reviewForm.review_type === 'won'" :label="t('opportunity.review.field.winReasons')" class="wide-field"><el-input v-model="reviewForm.win_reasons" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item v-if="reviewForm.review_type === 'lost'" :label="t('opportunity.review.field.lossReasons')" class="wide-field"><el-input v-model="reviewForm.loss_reasons" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.lessons')" class="wide-field"><el-input v-model="reviewForm.lessons" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.reusablePlaybook')" class="wide-field"><el-input v-model="reviewForm.reusable_playbook" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.avoidNextTime')" class="wide-field"><el-input v-model="reviewForm.avoid_next_time" type="textarea" :rows="3" :disabled="reviewReadonly" /></el-form-item>
        <el-form-item :label="t('opportunity.review.field.tags')"><el-select v-model="reviewForm.tags" multiple filterable allow-create default-first-option class="full-input" :disabled="reviewReadonly" :placeholder="t('opportunity.review.placeholder.tags')"><el-option v-for="item in reviewTagOptions" :key="item" :label="item" :value="item" /></el-select></el-form-item>
        <el-form-item :label="t('opportunity.review.field.keywords')"><el-select v-model="reviewForm.keywords" multiple filterable allow-create default-first-option class="full-input" :disabled="reviewReadonly" :placeholder="t('opportunity.review.placeholder.keywords')"><el-option v-for="item in reviewKeywordOptions" :key="item" :label="item" :value="item" /></el-select></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button v-if="!reviewReadonly" type="primary" :loading="reviewSaving" :disabled="reviewSubmitting" @click="saveReview()">{{ t('opportunity.review.saveDraft') }}</el-button>
        <el-button v-if="!reviewReadonly" type="success" plain :loading="reviewSubmitting" :disabled="reviewSaving" @click="submitReview()">{{ t('opportunity.review.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canEditRecord, canFollowRecord, canManageDomain, canManageRecordACL } from '../../permissions'
import DealChainPanel from '../../components/DealChainPanel.vue'
import ObjectAccessPanel from '../../components/ObjectAccessPanel.vue'
import ObjectAccessField from '../../components/ObjectAccessField.vue'
import { playCelebration } from '../../utils/celebration'
import { timelineOperatorText } from '../../utils/timeline'
import { useCustomerChainOptions } from '../../utils/customerChain'
import { currencyOptions, formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { apiErrorMessage } from '../../utils/api-error'

const route = useRoute()
const rows = ref([])
const users = ref([])
const products = ref([])
const campaigns = ref([])
const loading = ref(false)
const selectedRecord = ref(null)
const timeline = ref([])
const quotes = ref([])
const opportunityReviews = reactive({ won: null, lost: null })
const activeReviewType = ref('won')
const reviewSaving = ref(false)
const reviewSubmitting = ref(false)
const detailDrawer = ref(false)
const formDialog = ref(false)
const actionDialog = ref(false)
const reviewDialog = ref(false)
const editingId = ref(null)
const actionType = ref('stage')
const editFocusSection = ref('')
const draggingOpportunityId = ref(null)
const dragSourceStage = ref('')
const dragOverStage = ref('')
const dragClickSuppressed = ref(false)
const dragUpdating = ref(false)
const filters = reactive({ keyword: '', status: '', owner_id: undefined, reliability: '', importance: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
const actionForm = reactive({ stage: 'need_confirmed', reason: '', loss_reason: '', competitor: '', quote_amount: 0, valid_until: '', quote_id: 0, next_follow_at: '', next_action: '', title: '', remind_at: '', owner_ids: [] })
const reviewForm = reactive(defaultReviewForm())
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
const conflictWarnings = ref([])
const stageOptions = [
  { value: 'initial_contact', label: t('domain.auto.45b80954f8'), desc: t('domain.auto.f58c6f2e68'), probability: 10, color: '#2563eb', order: 1, followDays: 1 },
  { value: 'need_confirmed', label: t('domain.auto.eb5a8bd262'), desc: t('domain.auto.fea1f68ca8'), probability: 30, color: '#0891b2', order: 2, followDays: 2 },
  { value: 'proposal', label: t('domain.auto.df33ce33bb'), desc: t('domain.auto.a18affad06'), probability: 50, color: '#7c3aed', order: 3, followDays: 2 },
  { value: 'negotiation', label: t('domain.auto.83b022f008'), desc: t('domain.auto.1ca7753775'), probability: 70, color: '#ea580c', order: 4, followDays: 1 },
  { value: 'won', label: t('domain.auto.e65d23ddf1'), desc: t('domain.auto.64a107c096'), probability: 100, color: '#16a34a', order: 5, followDays: 0 },
  { value: 'lost', label: t('domain.auto.5375e3d681'), desc: t('domain.auto.38f53861cd'), probability: 0, color: '#dc2626', order: 5, followDays: 0 },
  { value: 'on_hold', label: t('domain.auto.f5f7939ed7'), desc: t('domain.auto.7d861e3b16'), probability: 0, color: '#64748b', order: 0, followDays: 7 }
]
const reliabilityOptions = [
  { label: t('opportunity.reliability.high'), value: 'high' },
  { label: t('opportunity.reliability.medium'), value: 'medium' },
  { label: t('opportunity.reliability.low'), value: 'low' },
  { label: t('opportunity.reliability.unknown'), value: 'unknown' }
]
const importanceOptions = [
  { label: t('opportunity.importance.critical'), value: 'critical' },
  { label: t('opportunity.importance.high'), value: 'high' },
  { label: t('opportunity.importance.normal'), value: 'normal' },
  { label: t('opportunity.importance.low'), value: 'low' }
]
const selectedPayload = computed(() => payload(selectedRecord.value))
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))
const campaignOptions = computed(() => campaigns.value.filter((item) => item.object_type === 'campaign'))
const canManage = computed(() => canManageDomain('opportunities'))
const actionTitle = computed(() => ({ stage: t('domain.auto.69add93d92'), follow_up: t('domain.auto.a68cc3cf80'), reminder: t('domain.auto.c07f8b771c'), create_quote: t('domain.auto.7e37e9b4fb'), quote_send: t('domain.auto.2dd8416d0e'), quote_accept: t('domain.auto.2f99833bf9'), quote_order: t('domain.auto.915099f209'), win: t('opportunity.action.winToDeal'), lost: t('domain.auto.4dbed6b418') }[actionType.value] || t('domain.auto.23f1b1b477')))
const actionPlaceholder = computed(() => actionType.value === 'follow_up' ? t('domain.auto.2be5944b2a') : actionType.value === 'reminder' ? t('domain.auto.42e0cd73d2') : t('domain.auto.dc1432359d'))
const nextStageOptions = computed(() => stageOptions.filter((item) => !['won', 'lost'].includes(item.value)))
const boardStages = computed(() => stageOptions.filter((item) => !['won', 'lost'].includes(item.value)))
const boardRows = computed(() => {
  if (filters.keyword || filters.owner_id) return rows.value
  return rows.value
})
const selectedProbability = computed(() => probabilityInfo(selectedRecord.value))
const selectedCustomer = computed(() => customerDisplay(selectedRecord.value))
const selectedContact = computed(() => contactDisplay(selectedRecord.value))
const activeOpportunityReview = computed(() => opportunityReviews[activeReviewType.value] || null)
const reviewReadonly = computed(() => Boolean(activeOpportunityReview.value && activeOpportunityReview.value.status !== 'draft'))
const reviewTypes = ['won', 'lost']
const reviewTagOptions = computed(() => ['won', 'lost', 'price', 'competitor', 'productFit', 'customerRelationship', 'purchaseTiming', 'repeatablePlaybook'].map((key) => t(`opportunity.review.tag.${key}`)))
const reviewKeywordOptions = computed(() => ['decisionChain', 'budget', 'competitor', 'quote', 'pilot', 'procurement', 'delivery', 'risk'].map((key) => t(`opportunity.review.keyword.${key}`)))

onMounted(async () => {
  await Promise.all([loadUsers(), loadProducts(), loadCampaigns(), loadCustomerOptions(), loadPage()])
  await openRouteRecord()
})

async function loadUsers() {
  try {
    const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } })
    users.value = res.data?.items || []
  } catch {
    users.value = []
  }
}
async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/opportunities/objects', { params: cleanParams({ object_type: 'opportunity', ...filters, page: pagination.page, size: pagination.size }) })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
    if (selectedRecord.value) {
      const fresh = rows.value.find((item) => item.id === selectedRecord.value.id)
      if (fresh) selectedRecord.value = fresh
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.3f7d482e96')))
  } finally {
    loading.value = false
  }
}

async function openRouteRecord() {
  const id = Number(route.query.id || 0)
  if (!id) return
  let record = rows.value.find((item) => Number(item.id) === id)
  if (!record) {
    try {
      const res = await api.get(`/api/v1/prd/opportunities/objects/${id}`)
      record = res.data
    } catch {
      record = null
    }
  }
  if (record) await openDetail(record)
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
    const res = await api.get('/api/v1/prd/campaigns/objects', { params: { object_type: 'campaign', page: 1, size: 200 } })
    campaigns.value = res.data?.items || []
  } catch {
    campaigns.value = []
  }
}
function openCreate() {
  editingId.value = null
  editFocusSection.value = ''
  Object.assign(form, defaultForm())
  conflictWarnings.value = []
  formDialog.value = true
  if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions()
  checkConflicts()
}
function openEdit(row, focusSection = '') {
  editingId.value = row.id
  editFocusSection.value = focusSection
  Object.assign(form, { object_type: 'opportunity', name: row.name, code: row.code || '', status: row.status || 'initial_contact', owner_id: row.owner_id || 0, company_id: row.company_id || 0, contact_id: row.contact_id || 0, amount: row.amount || 0, currency: row.currency || defaultBusinessCurrency(), score: row.score || 0, tags: splitTags(row.tags), payload: { line_items: [], reliability: 'medium', importance: 'normal', ...payload(row) } })
  syncCustomerChainFromRecord(row)
  if (!Array.isArray(form.payload.line_items)) form.payload.line_items = []
  conflictWarnings.value = []
  formDialog.value = true
  if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions()
  checkConflicts()
}
async function submitForm() {
  if (!form.name) return ElMessage.error(t('domain.auto.3339b9f8d2'))
  try {
    normalizeCustomerChainForSubmit()
    form.payload.probability = probability(form.status)
    form.payload.stage = form.status
    normalizeCampaignPayload(form.payload)
    if (editingId.value) await api.put(`/api/v1/prd/opportunities/objects/${editingId.value}`, form)
    else await api.post('/api/v1/prd/opportunities/objects', form)
    ElMessage.success(t('domain.auto.9e156a5115'))
    formDialog.value = false
    await loadPage()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca37aec661')))
  }
}
async function checkConflicts() {
  const company = String(form.payload?.company || '').trim()
  if (!company) {
    conflictWarnings.value = []
    return
  }
  try {
    const res = await api.get('/api/v1/prd/flows/opportunities/conflicts', {
      params: {
        company,
        name: form.name || '',
        exclude_id: editingId.value || 0,
        limit: 6
      }
    })
    conflictWarnings.value = (res.data?.items || []).filter((item) => Number(item.id) !== Number(editingId.value || 0))
  } catch {
    conflictWarnings.value = []
  }
}
function openConflict(item) {
  formDialog.value = false
  const row = rows.value.find((entry) => Number(entry.id) === Number(item.id))
  if (row) openDetail(row)
}
async function openDetail(row) {
  selectedRecord.value = row
  detailDrawer.value = true
  await Promise.all([loadTimeline(row), loadQuotes(row), loadOpportunityReview(row)])
}
async function loadTimeline(row) {
  const rel = await api.get('/api/v1/prd/relations', { params: { object_type: 'opportunities', object_id: row.id, relation_type: 'timeline' } })
  const ids = (rel.data?.items || []).map((item) => item.target_type === 'contacts' ? item.target_id : item.source_id)
  const items = await Promise.all(ids.map((id) => api.get(`/api/v1/prd/contacts/objects/${id}`).catch(() => null)))
  timeline.value = items.map((res) => res?.data).filter(Boolean)
}
async function loadQuotes(row) {
  const rel = await api.get('/api/v1/prd/relations', { params: { object_type: 'opportunities', object_id: row.id, relation_type: 'quote' } })
  const ids = (rel.data?.items || []).filter((item) => item.target_type === 'opportunities').map((item) => item.target_id)
  const items = await Promise.all(ids.map((id) => api.get(`/api/v1/prd/opportunities/objects/${id}`).catch(() => null)))
  quotes.value = items.map((res) => res?.data).filter((item) => item?.object_type === 'quote')
}
async function loadOpportunityReview(row) {
  if (!row?.id) {
    clearOpportunityReviews()
    return
  }
  const [won, lost] = await Promise.all(reviewTypes.map((type) => api.get(`/api/v1/prd/flows/opportunities/${row.id}/review`, { params: { review_type: type } }).catch(() => null)))
  opportunityReviews.won = won?.data?.review || null
  opportunityReviews.lost = lost?.data?.review || null
}
function openAction(type, row = selectedRecord.value) {
  if (!row) return
  selectedRecord.value = row
  actionType.value = type
  Object.assign(actionForm, {
    stage: nextStage(row.status),
    reason: '',
    loss_reason: '',
    competitor: '',
    quote_amount: row.amount || 0,
    valid_until: '',
    quote_id: 0,
    next_follow_at: '',
    next_action: payload(row).next_action || '',
    title: `${t('domain.auto.5dbd2e4bd0')}${row.name}`,
    remind_at: '',
    owner_ids: row.owner_id ? [row.owner_id] : []
  })
  actionDialog.value = true
}
function openReview(type, row = selectedRecord.value) {
  if (!row) return
  const normalizedType = normalizeReviewType(type) || (row.status === 'lost' ? 'lost' : 'won')
  selectedRecord.value = row
  activeReviewType.value = normalizedType
  const info = payload(row)
  Object.assign(reviewForm, defaultReviewForm())
  const existingReview = opportunityReviews[normalizedType] || null
  const existing = payload(existingReview)
  Object.assign(reviewForm, {
    review_type: normalizedType,
    title: existingReview?.name || `${reviewTypeLabel(normalizedType)}: ${row.name}`,
    summary: existing.summary || '',
    industry: existing.industry || info.industry || '',
    competitor: existing.competitor || (normalizedType === 'lost' ? info.competitor || '' : ''),
    background: existing.background || '',
    customer_needs: existing.customer_needs || '',
    decision_process: existing.decision_process || '',
    key_actions: existing.key_actions || '',
    obstacles: existing.obstacles || '',
    win_reasons: existing.win_reasons || (normalizedType === 'won' ? info.won_reason || '' : ''),
    loss_reasons: existing.loss_reasons || (normalizedType === 'lost' ? info.loss_reason || '' : ''),
    lessons: existing.lessons || '',
    reusable_playbook: existing.reusable_playbook || '',
    avoid_next_time: existing.avoid_next_time || '',
    tags: Array.isArray(existing.tags) ? existing.tags : splitTags(existingReview?.tags),
    keywords: Array.isArray(existing.keywords) ? existing.keywords : []
  })
  reviewDialog.value = true
}
async function saveReview(options = {}) {
  if (!selectedRecord.value) return
  if (reviewReadonly.value) {
    ElMessage.error(t('opportunity.review.message.readonly'))
    return null
  }
  if (reviewSaving.value || reviewSubmitting.value) return activeOpportunityReview.value
  if (!reviewForm.title.trim()) return ElMessage.error(t('opportunity.review.message.titleRequired'))
  if (!reviewForm.summary.trim()) return ElMessage.error(t('opportunity.review.message.summaryRequired'))
  const targetType = normalizeReviewType(activeReviewType.value) || 'won'
  reviewSaving.value = true
  try {
    const res = await api.post(`/api/v1/prd/flows/opportunities/${selectedRecord.value.id}/review`, { ...reviewForm, review_type: targetType })
    opportunityReviews[targetType] = res.data?.review || null
    ElMessage.success(t('opportunity.review.message.draftSaved'))
    if (!options.keepOpen) reviewDialog.value = false
    await loadTimeline(selectedRecord.value)
    return opportunityReviews[targetType]
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opportunity.review.message.saveFailed')))
    return null
  } finally {
    reviewSaving.value = false
  }
}
async function submitReview(type = activeReviewType.value) {
  const targetType = normalizeReviewType(type) || 'won'
  activeReviewType.value = targetType
  if (reviewSubmitting.value) return
  const targetReview = opportunityReviews[targetType]
  if (isReviewReadonly(targetReview)) {
    ElMessage.error(t('opportunity.review.message.readonly'))
    return
  }
  if (targetReview?.id && targetReview.status !== 'draft') {
    ElMessage.error(t('opportunity.review.message.readonly'))
    return
  }
  reviewSubmitting.value = true
  try {
    if (!opportunityReviews[targetType]?.id) {
      reviewSubmitting.value = false
      const saved = await saveReview({ keepOpen: true })
      reviewSubmitting.value = true
      if (!saved?.id) return
    }
    const res = await api.post(`/api/v1/prd/flows/opportunities/reviews/${opportunityReviews[targetType].id}/submit`)
    opportunityReviews[targetType] = res.data?.review || opportunityReviews[targetType]
    reviewDialog.value = false
    ElMessage.success(t('opportunity.review.message.submitted'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opportunity.review.message.submitFailed')))
  } finally {
    reviewSubmitting.value = false
  }
}
async function publishReview(type = activeReviewType.value) {
  const targetType = normalizeReviewType(type) || 'won'
  const review = opportunityReviews[targetType]
  if (!review?.id) return
  try {
    const res = await api.post(`/api/v1/prd/flows/opportunities/reviews/${review.id}/publish`)
    opportunityReviews[targetType] = res.data?.review || review
    ElMessage.success(t('opportunity.review.message.published'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opportunity.review.message.publishFailed')))
  }
}
async function archiveReview(type = activeReviewType.value) {
  const targetType = normalizeReviewType(type) || 'won'
  const review = opportunityReviews[targetType]
  if (!review?.id) return
  try {
    const res = await api.post(`/api/v1/prd/flows/opportunities/reviews/${review.id}/archive`)
    opportunityReviews[targetType] = res.data?.review || review
    ElMessage.success(t('opportunity.review.message.archived'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opportunity.review.message.archiveFailed')))
  }
}
function openQuoteAction(type, quote) {
  actionType.value = type
  Object.assign(actionForm, { stage: '', reason: '', loss_reason: '', competitor: '', quote_amount: quote.amount || 0, valid_until: payload(quote).valid_until || '', quote_id: quote.id, next_follow_at: '', next_action: '', title: '', remind_at: '', owner_ids: [] })
  actionDialog.value = true
}
async function submitAction() {
  if (!actionForm.reason) return ElMessage.error(t('domain.auto.bd59f8c970'))
  const row = selectedRecord.value
  const type = actionType.value
  try {
    if (type === 'stage') {
      await api.post(`/api/v1/prd/flows/opportunities/${row.id}/stage`, { stage: actionForm.stage, comment: actionForm.reason, payload_patch: { last_follow_up: actionForm.reason } })
      ElMessage.success(t('domain.auto.35abf77553'))
    } else if (type === 'follow_up') {
      await api.post(`/api/v1/prd/flows/opportunities/${row.id}/follow-up`, { content: actionForm.reason, next_follow_at: actionForm.next_follow_at, next_action: actionForm.next_action })
      ElMessage.success(t('domain.auto.a55a9b8029'))
    } else if (type === 'reminder') {
      await api.post(`/api/v1/prd/flows/opportunities/${row.id}/reminder`, { title: actionForm.title, content: actionForm.reason, remind_at: actionForm.remind_at, owner_ids: actionForm.owner_ids })
      ElMessage.success(t('domain.auto.e28c55f4fa'))
    } else if (type === 'create_quote') {
      await api.post(`/api/v1/prd/flows/opportunities/${row.id}/quote`, { reason: actionForm.reason, quote_amount: actionForm.quote_amount || row.amount || 0, valid_until: actionForm.valid_until })
      ElMessage.success(t('domain.auto.bc34e9d806'))
    } else if (type === 'quote_send') {
      await api.post(`/api/v1/prd/flows/quotes/${actionForm.quote_id}/send`, { reason: actionForm.reason })
      ElMessage.success(t('domain.auto.f18aa237db'))
    } else if (type === 'quote_accept') {
      await api.post(`/api/v1/prd/flows/quotes/${actionForm.quote_id}/accept`, { reason: actionForm.reason })
      ElMessage.success(t('domain.auto.d30184af5e'))
    } else if (type === 'quote_order') {
      await api.post(`/api/v1/prd/flows/quotes/${actionForm.quote_id}/order`, { reason: actionForm.reason })
      ElMessage.success(t('domain.auto.26cac728f3'))
    } else if (type === 'win') {
      await api.post(`/api/v1/prd/flows/opportunities/${row.id}/win`, { reason: actionForm.reason })
      ElMessage.success(t('opportunity.message.winCreatedDeal'))
    } else if (type === 'lost') {
      if (!actionForm.loss_reason || !actionForm.competitor) return ElMessage.error(t('domain.auto.e5d3ceb3ad'))
      await api.post(`/api/v1/prd/flows/opportunities/${row.id}/stage`, { stage: 'lost', comment: actionForm.reason, payload_patch: { loss_reason: actionForm.loss_reason, competitor: actionForm.competitor } })
      ElMessage.success(t('domain.auto.77c1b276cd'))
    }
    playOpportunityCelebration(type, row, actionForm.stage)
    actionDialog.value = false
    if (shouldCollapseDetailAfterAction(type)) detailDrawer.value = false
    await loadPage()
    const fresh = rows.value.find((item) => item.id === row.id) || row
    selectedRecord.value = fresh
    if (detailDrawer.value) await Promise.all([loadTimeline(fresh), loadQuotes(fresh), loadOpportunityReview(fresh)])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca37aec661')))
  }
}
function quickStage(stage) {
  if (!stage) {
    filters.status = ''
    loadPage()
    return
  }
  filters.status = filters.status === stage ? '' : stage
  loadPage()
}
function opportunityRowClass({ row }) {
  return selectedRecord.value && Number(selectedRecord.value.id) === Number(row.id) ? 'is-selected-row' : ''
}
function stageRows(stage) { return boardRows.value.filter((item) => item.status === stage).slice(0, 24) }
function handleKanbanCardClick(item) {
  if (dragClickSuppressed.value) {
    dragClickSuppressed.value = false
    return
  }
  openDetail(item)
}
function startOpportunityDrag(item, event) {
  if (!canManage.value || dragUpdating.value) {
    event?.preventDefault?.()
    return
  }
  draggingOpportunityId.value = item.id
  dragSourceStage.value = item.status || ''
  dragOverStage.value = ''
  dragClickSuppressed.value = true
  if (event?.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(item.id))
  }
}
function handleOpportunityDragOver(stage, event) {
  if (!canManage.value || !draggingOpportunityId.value || !isBoardStage(stage)) return
  dragOverStage.value = stage
  if (event?.dataTransfer) event.dataTransfer.dropEffect = stage === dragSourceStage.value ? 'none' : 'move'
}
function handleOpportunityDragLeave(stage, event) {
  if (dragOverStage.value !== stage) return
  const next = event?.relatedTarget
  if (next && event?.currentTarget?.contains?.(next)) return
  dragOverStage.value = ''
}
async function dropOpportunityOnStage(targetStage) {
  const id = draggingOpportunityId.value
  const sourceStage = dragSourceStage.value
  dragOverStage.value = ''
  if (!canManage.value || !id || !isBoardStage(targetStage)) return endOpportunityDrag()
  if (targetStage === sourceStage) return endOpportunityDrag()
  const row = rows.value.find((item) => Number(item.id) === Number(id))
  if (!row) return endOpportunityDrag()

  const previousStatus = row.status
  const previousPayloadJson = row.payload_json
  const previousSelected = selectedRecord.value && Number(selectedRecord.value.id) === Number(row.id)
    ? { status: selectedRecord.value.status, payload_json: selectedRecord.value.payload_json }
    : null
  const nextPayload = { ...payload(row), stage: targetStage, probability: probability(targetStage) }

  dragUpdating.value = true
  row.status = targetStage
  row.payload_json = nextPayload
  if (previousSelected) {
    selectedRecord.value.status = targetStage
    selectedRecord.value.payload_json = nextPayload
  }

  try {
    await api.post(`/api/v1/prd/flows/opportunities/${row.id}/stage`, { stage: targetStage, comment: t('opportunity.kanban.dragComment'), payload_patch: { from_board_drag: true } })
    ElMessage.success(t('opportunity.message.movedToStage', { stage: stageLabel(targetStage) }))
    if (stageMeta(targetStage).order > stageMeta(previousStatus).order && targetStage !== 'on_hold') {
      playCelebration(stageMeta(targetStage).order >= 4 ? 'milestone' : 'light')
    }
    await loadPage()
    const fresh = rows.value.find((item) => Number(item.id) === Number(row.id)) || row
    if (selectedRecord.value && Number(selectedRecord.value.id) === Number(row.id)) {
      selectedRecord.value = fresh
      if (detailDrawer.value) await Promise.all([loadTimeline(fresh), loadQuotes(fresh), loadOpportunityReview(fresh)])
    }
  } catch (error) {
    row.status = previousStatus
    row.payload_json = previousPayloadJson
    if (previousSelected) {
      selectedRecord.value.status = previousSelected.status
      selectedRecord.value.payload_json = previousSelected.payload_json
    }
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca14c10f14')))
  } finally {
    dragUpdating.value = false
    endOpportunityDrag()
  }
}
function endOpportunityDrag() {
  draggingOpportunityId.value = null
  dragSourceStage.value = ''
  dragOverStage.value = ''
  window.setTimeout(() => {
    dragClickSuppressed.value = false
  }, 0)
}
function emptyStageText(stage) {
  return t('domain.opportunities.emptyStage', { stage: stage.label })
}
async function quickAdvanceStage(stage) {
  if (!canManage.value) return
  const fromRows = stageRows(stage)
  const toStage = nextStage(stage)
  if (!fromRows.length || !toStage) return
  const picked = fromRows.slice(0, 5)
  try {
    await Promise.all(picked.map((item) => api.post(`/api/v1/prd/flows/opportunities/${item.id}/stage`, { stage: toStage, comment: t('domain.auto.d0836ffe04'), payload_patch: { from_board: true } })))
    ElMessage.success(`${t('domain.auto.7365c38d6d')}${picked.length}${t('domain.auto.4868f44172')}${stageLabel(toStage)}`)
    playCelebration(stageMeta(toStage).order >= 4 ? 'milestone' : 'light')
    await loadPage()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca14c10f14')))
  }
}
function playOpportunityCelebration(type, row, targetStage) {
  if (type === 'lost') return
  if (['follow_up', 'reminder', 'quote_send'].includes(type)) return playCelebration('light')
  if (type === 'create_quote') return playCelebration('milestone')
  if (['quote_accept', 'quote_order', 'win'].includes(type)) return playCelebration('win')
  if (type === 'stage') {
    if (!targetStage || ['lost', 'on_hold'].includes(targetStage)) return
    if (stageMeta(targetStage).order > stageMeta(row?.status).order) {
      playCelebration(stageMeta(targetStage).order >= 4 ? 'milestone' : 'light')
    }
  }
}
function shouldCollapseDetailAfterAction(type) {
  return ['stage', 'create_quote', 'quote_order', 'win', 'lost'].includes(type)
}
function resetFilters() { Object.assign(filters, { keyword: '', status: '', owner_id: undefined, reliability: '', importance: '' }); pagination.page = 1; loadPage() }
async function refreshSelected(id) { await loadPage(); const next = rows.value.find((item) => Number(item.id) === Number(id)); if (next) { selectedRecord.value = next; await Promise.all([loadTimeline(next), loadQuotes(next), loadOpportunityReview(next)]) } }
function addLine() { form.payload.line_items.push({ product_id: undefined, sku: '', name: '', quantity: 1, unit_price: 0, discount: 0, tax_rate: 0, stock_available: undefined }) }
function removeLine(index) { form.payload.line_items.splice(index, 1) }
function defaultForm() { return { object_type: 'opportunity', name: '', code: '', status: 'initial_contact', owner_id: 0, company_id: 0, contact_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { company_id: undefined, company: '', company_name: '', contact_id: undefined, contact: '', contact_name: '', stage: 'initial_contact', probability: 10, campaign_id: undefined, campaign_name: '', source_channel: '', utm_source: '', utm_medium: '', utm_campaign: '', reliability: 'medium', importance: 'normal', quote_status: 'draft', visibility: 'public', line_items: [] } } }
function defaultReviewForm() { return { review_type: 'won', title: '', summary: '', industry: '', competitor: '', background: '', customer_needs: '', decision_process: '', key_actions: '', obstacles: '', win_reasons: '', loss_reasons: '', lessons: '', reusable_playbook: '', avoid_next_time: '', tags: [], keywords: [], payload: {} } }
function payload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function normalizeReviewType(value) { return value === 'lost' ? 'lost' : value === 'won' ? 'won' : '' }
function clearOpportunityReviews() { opportunityReviews.won = null; opportunityReviews.lost = null }
function reviewFor(type) { return opportunityReviews[normalizeReviewType(type)] || null }
function reviewPayloadFor(type) { return payload(reviewFor(type)) }
function isReviewReadonly(review) { return Boolean(review && review.status !== 'draft') }
function handleCampaignChange(value) {
  const campaign = campaignOptions.value.find((item) => Number(item.id) === Number(value))
  if (!campaign) {
    form.payload.campaign_name = ''
    form.payload.source_channel = ''
    form.payload.utm_source = ''
    form.payload.utm_medium = ''
    form.payload.utm_campaign = ''
    return
  }
  const info = payload(campaign)
  form.payload.campaign_id = campaign.id
  form.payload.campaign_name = campaign.name
  form.payload.source_channel = info.source_channel || 'campaign'
  form.payload.utm_source = info.utm_source || info.source_channel || 'campaign'
  form.payload.utm_medium = info.utm_medium || ''
  form.payload.utm_campaign = info.utm_campaign || campaign.name
}
function normalizeCampaignPayload(target) {
  const campaignID = Number(target.campaign_id || 0)
  if (!campaignID) {
    delete target.campaign_id
    target.campaign_name = ''
    return
  }
  const campaign = campaignOptions.value.find((item) => Number(item.id) === campaignID)
  if (!campaign) return
  const info = payload(campaign)
  target.campaign_id = campaign.id
  target.campaign_name = target.campaign_name || campaign.name
  target.source_channel = target.source_channel || info.source_channel || 'campaign'
  target.utm_source = target.utm_source || info.utm_source || info.source_channel || 'campaign'
  target.utm_medium = target.utm_medium || info.utm_medium || ''
  target.utm_campaign = target.utm_campaign || info.utm_campaign || campaign.name
}
function campaignOptionLabel(item) {
  const info = payload(item)
  return `${item.name}${info.source_channel ? ` · ${info.source_channel}` : ''}`
}
function objectVisibility(row) { const p = payload(row); return p.visibility === 'private' || p.access_visibility === 'private' || p.private === true || p.private === 'true' ? 'private' : 'public' }
function lineItems(row) { return Array.isArray(payload(row).line_items) ? payload(row).line_items : [] }
function customerDisplay(row) {
  const info = payload(row)
  const name = firstText(info.company, info.company_name, info.customer_name, info.account_name)
  if (name) return { name, needsEdit: false }
  if (row?.company_id) return { name: `${t('domain.auto.a4f7751de8')}${row.company_id}）`, needsEdit: true }
  return { name: t('domain.auto.1b234a3233'), needsEdit: true }
}
function contactDisplay(row) {
  const info = payload(row)
  const name = firstText(info.contact_name, info.contact, info.contact_person, info.buyer_name)
  if (name) return { name, needsEdit: false }
  if (row?.contact_id) return { name: `${t('domain.auto.2db385fc6f')}${row.contact_id}）`, needsEdit: true }
  return { name: t('domain.auto.53e48d62d5'), needsEdit: true }
}
function firstText(...values) {
  return values.map((value) => String(value || '').trim()).find(Boolean) || ''
}
function probabilityInfo(row) {
  if (!row) return { value: 0, label: t('domain.auto.9bcd8b85a3') }
  const info = payload(row)
  if (info.probability !== undefined && info.probability !== null && String(info.probability).trim() !== '') {
    const value = Number(info.probability || 0)
    const defaultValue = probability(row.status)
    return { value, label: value === defaultValue ? t('domain.auto.9bcd8b85a3') : t('domain.auto.c14d1a91d8') }
  }
  return { value: probability(row.status), label: t('domain.auto.9bcd8b85a3') }
}
function expectedCloseLabel(row) {
  return payload(row).expected_close_date || t('domain.auto.0c426a865f')
}
function productSummary(row) {
  const lines = lineItems(row)
  if (!lines.length) return t('domain.auto.7a48b4ca88')
  const names = lines.map((item) => item.name || item.product_name || item.sku || t('domain.auto.4b9c8904e9')).filter(Boolean)
  if (names.length <= 2) return names.join('、')
  return `${names.slice(0, 2).join('、')}${t('domain.auto.3d3e17a162')}${names.length}${t('domain.auto.64728a7727')}`
}
function stageGateChecks(row) {
  if (!row) return []
  const info = payload(row)
  const lines = lineItems(row)
  const hasLineItems = lines.length > 0
  const hasExpectedClose = Boolean(String(info.expected_close_date || '').trim())
  const hasCompetitor = Boolean(String(info.competitor || '').trim())
  const hasLossReason = Boolean(String(info.loss_reason || '').trim())
  const currentStage = row.status
  const checks = []
  if (['initial_contact', 'need_confirmed', 'proposal', 'negotiation'].includes(currentStage)) {
    checks.push({ key: 'need_line_items', title: t('domain.auto.eb7ce203cd'), hint: t('domain.auto.54669cc2e0'), pass: hasLineItems, action: 'line_items' })
  }
  if (['need_confirmed', 'proposal', 'negotiation'].includes(currentStage)) {
    checks.push({ key: 'need_expected_close', title: t('domain.auto.a432d87a19'), hint: t('domain.auto.46d8eacdf3'), pass: hasExpectedClose, action: 'expected_close' })
  }
  if (['proposal', 'negotiation'].includes(currentStage)) {
    checks.push({ key: 'need_quote_or_note', title: t('domain.auto.d4da93585d'), hint: hasLineItems ? t('domain.auto.3d2f837f69') : t('domain.auto.abe93df4c0'), pass: quotes.value.length > 0, action: hasLineItems ? 'create_quote' : 'line_items' })
  }
  if (currentStage === 'lost') {
    checks.push({ key: 'need_loss_fields', title: t('domain.auto.021d6b5590'), hint: t('domain.auto.c2e2a43550'), pass: hasLossReason && hasCompetitor, action: 'lost' })
  }
  if (!checks.length) {
    checks.push({ key: 'ready', title: t('domain.auto.abed8c12d1'), hint: t('domain.auto.2bfaee05b9'), pass: true })
  }
  return checks
}
function handleGateAction(item) {
  if (!selectedRecord.value || !item?.action) return
  if (item.action === 'line_items') return openEdit(selectedRecord.value, 'line_items')
  if (item.action === 'expected_close') return openEdit(selectedRecord.value, 'expected_close')
  if (item.action === 'create_quote') return openAction('create_quote')
  if (item.action === 'lost') return openAction('lost')
}
function productInfo(row) { return payload(row) }
function applyProductToLine(item) {
  const product = products.value.find((entry) => Number(entry.id) === Number(item.product_id))
  if (!product) return
  const info = productInfo(product)
  item.product_id = product.id
  item.sku = info.sku || product.code || ''
  item.name = product.name
  item.product_name = product.name
  item.unit_price = Number(product.amount || 0)
  item.price = Number(product.amount || 0)
  item.tax_rate = Number(info.tax_rate || 0)
  item.stock_available = Number(info.stock || 0)
}
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function stageLabel(value) { return stageOptions.find((item) => item.value === value)?.label || value || '-' }
function stageMeta(value) { return stageOptions.find((item) => item.value === value) || { value, label: value || '-', probability: 0, color: '#64748b', order: 0, followDays: 2 } }
function isBoardStage(value) { return boardStages.value.some((item) => item.value === value) }
function stageStyle(value) { return { '--stage-color': stageMeta(value).color } }
function stageOrderProgress(value) {
  const meta = stageMeta(value)
  if (value === 'on_hold') return t('domain.auto.90c88f8abc')
  if (value === 'lost') return t('domain.auto.f628761bf5')
  const maxOrder = 5
  return `${t('domain.auto.dae828fe4f')}${meta.order || 1}/${maxOrder}${t('domain.auto.4ca39faad0')}`
}
function quoteStatusLabel(value) { return ({ draft: t('domain.auto.0f436818c0'), sent: t('domain.auto.afb62932a7'), accepted: t('domain.auto.94e52fbc6e'), rejected: t('domain.auto.4c7c52c706'), expired: t('domain.auto.1354374f76') }[value] || value || '-') }
function probability(stage) { return stageOptions.find((item) => item.value === stage)?.probability || 0 }
function nextStage(stage) { return ({ initial_contact: 'need_confirmed', need_confirmed: 'proposal', proposal: 'negotiation', negotiation: 'won', on_hold: 'need_confirmed' }[stage] || 'need_confirmed') }
function stageCount(stage) { return rows.value.filter((item) => item.status === stage).length }
function stageAmount(stage) { return rows.value.filter((item) => item.status === stage).reduce((sum, item) => sum + Number(item.amount || 0), 0) }
function ownerName(ownerID) { const id = Number(ownerID || 0); if (!id) return t('domain.auto.3bb92e6efc'); const user = users.value.find((item) => Number(item.id) === id); return user ? userDisplayName(user) : `${t('domain.auto.9ba763ea34')}${id}` }
function userDisplayName(user) { const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`; return user.department_name ? `${label} · ${user.department_name}` : label }
function reliabilityLabel(value) { return reliabilityOptions.find((item) => item.value === value)?.label || t('opportunity.reliability.medium') }
function reliabilityType(value) { return value === 'high' ? 'success' : value === 'low' ? 'danger' : value === 'unknown' ? 'info' : 'warning' }
function importanceLabel(value) { return importanceOptions.find((item) => item.value === value)?.label || t('opportunity.importance.normal') }
function importanceType(value) { return value === 'critical' || value === 'high' ? 'danger' : value === 'low' ? 'info' : 'warning' }
function timelineTitle(item) { return payload(item).summary || item.name || t('domain.auto.6fcdba1f71') }
function timelineOperator(item) { return timelineOperatorText(item, users.value, formatTime) }
function reviewTypeLabel(value) { return ({ won: t('opportunity.review.type.won'), lost: t('opportunity.review.type.lost') }[value] || t('opportunity.review.title')) }
function reviewStatusLabel(value) { return ({ draft: t('opportunity.review.status.draft'), pending_review: t('opportunity.review.status.pendingReview'), published: t('opportunity.review.status.published'), archived: t('opportunity.review.status.archived') }[value] || value || '-') }
function reviewActionLabel(type) {
  const review = reviewFor(type)
  if (!review) return t('opportunity.review.create')
  return review.status === 'draft' ? t('opportunity.review.edit') : t('opportunity.review.view')
}
function reviewStatusType(value) { return ({ draft: 'info', pending_review: 'warning', published: 'success', archived: 'danger' }[value] || 'info') }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
function formatTime(value) { if (!value) return '-'; const date = new Date(value); return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}` }
function daysBetween(value) {
  if (!value) return 0
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 0
  return Math.max(0, Math.floor((Date.now() - date.getTime()) / 86400000))
}
function stageStayDays(row) {
  const info = payload(row)
  return daysBetween(info.stage_entered_at || info.stage_changed_at || row?.updated_at || row?.created_at)
}
function followStatus(row) {
  const info = payload(row)
  if (['won', 'lost'].includes(row?.status)) return 'normal'
  if (info.follow_up_status) return info.follow_up_status
  const nextAt = info.next_follow_at ? new Date(info.next_follow_at) : null
  if (nextAt && !Number.isNaN(nextAt.getTime())) {
    if (nextAt.getTime() < Date.now()) return 'overdue'
    if (nextAt.getTime() - Date.now() <= 86400000) return 'due_soon'
  }
  const followDays = stageMeta(row?.status).followDays || 2
  return stageStayDays(row) > followDays ? 'stale' : 'normal'
}
function followStatusLabel(row) {
  return ({ normal: t('domain.auto.f78d037abc'), due_soon: t('domain.auto.810ab25a9c'), overdue: t('domain.auto.bc78284ee9'), stale: t('domain.auto.46a37731a1') }[followStatus(row)] || t('domain.auto.f78d037abc'))
}
function followStatusType(row) {
  return ({ normal: 'success', due_soon: 'warning', overdue: 'danger', stale: 'warning' }[followStatus(row)] || 'info')
}
function followDueText(row) {
  const info = payload(row)
  if (info.next_follow_at) return `${t('domain.auto.1a348d2155')}${formatTime(info.next_follow_at)}`
  if (info.last_follow_up_at || info.last_follow_at) return `${t('domain.auto.8c73d90eca')}${formatTime(info.last_follow_up_at || info.last_follow_at)}`
  return `${t('domain.auto.bd9d31728c')}${stageStayDays(row)}${t('domain.auto.c3304d1e49')}`
}

let conflictCheckTimer = null
watch(() => [form.name, form.payload?.company, formDialog.value], () => {
  if (!formDialog.value) return
  if (conflictCheckTimer) clearTimeout(conflictCheckTimer)
  conflictCheckTimer = setTimeout(() => {
    checkConflicts()
  }, 250)
})
</script>
