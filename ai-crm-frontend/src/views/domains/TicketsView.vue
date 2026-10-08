<template>
  <div class="system-page crm-flow-page">
    <section class="crm-help-strip"><span>{{ t('domain.auto.d3395a94f9') }}</span></section>

    <section class="system-card">
      <div class="list-card-search">
        <el-form :model="filters" label-width="72px" class="system-search-form ticket-search-form">
          <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.9e88f360bc')" clearable /></el-form-item>
          <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="filters.status" clearable :placeholder="t('domain.auto.1a4c26d92d')"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="filters.owner_id" clearable filterable :placeholder="t('domain.auto.8b134548e7')"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
          <el-form-item :label="t('domain.auto.cd99b225a0')">
            <el-select v-model="filters.language" clearable :placeholder="t('domain.auto.5924113b5c')">
              <el-option v-for="item in languageOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <div class="system-search-actions">
            <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
            <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
          </div>
        </el-form>
      </div>
      <div class="crm-stage-board">
        <button class="crm-stage-card" :class="{ active: !filters.status }" @click="quickStatus('')">
          <strong>{{ t('domain.auto.3e7562a2b5') }}</strong>
          <span>{{ t('domain.auto.ad91043311') }}</span>
          <em>{{ pagination.total }} {{ t('domain.auto.f7b2a6ee68') }}</em>
        </button>
        <button v-for="item in statusOptions" :key="item.value" class="crm-stage-card" :class="{ active: filters.status === item.value }" @click="quickStatus(item.value)">
          <strong>{{ item.label }}</strong>
          <span>{{ item.desc }}</span>
          <em>{{ statusCount(item.value) }} {{ t('domain.auto.f7b2a6ee68') }}</em>
        </button>
      </div>
      <div class="system-toolbar">
        <div class="toolbar-left">
          <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.4fc04ec282') }}</el-button>
          <el-button v-if="canFollowRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('note')">{{ t('domain.auto.71a58cc706') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('status')">{{ t('domain.auto.f92064cffd') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('assign')">{{ t('domain.auto.6a93715283') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('transfer')">{{ t('domain.auto.e18b64d93a') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="autoAssignTicket">{{ t('domain.auto.8871d101f4') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('upsell')">{{ t('domain.auto.8335753549') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('bug')">{{ t('domain.auto.ab266e40e3') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord || !selectedPayload.bug_task_id" @click="openAction('bug_status')">{{ t('domain.auto.c348cd66c6') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="createKnowledge">{{ t('domain.auto.ee13c2f5b3') }}</el-button>
          <el-button v-if="canManage" @click="openTransferBoard">{{ t('domain.auto.bdb6144a3e') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="success" plain @click="openAction('close')">{{ t('domain.auto.3b810106e7') }}</el-button>
        </div>
        <div class="toolbar-icons"><el-tooltip :content="t('domain.auto.38108eaa1d')"><button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage"><Icon icon="ri:refresh-line" /></button></el-tooltip></div>
      </div>

      <el-table :data="rows" class="system-table" row-key="id" v-loading="loading" highlight-current-row :current-row-key="selectedRecord?.id" :row-class-name="ticketRowClass">
        <el-table-column prop="name" :label="t('domain.auto.85c230ed15')" min-width="220"><template #default="{ row }"><div class="record-title"><strong>{{ row.name }}</strong><span>{{ payload(row).category || t('domain.auto.b28f13ea3e') }} · {{ payload(row).company || `${t('domain.auto.f206870601')}#${row.company_id || '-'}` }}</span></div></template></el-table-column>
        <el-table-column :label="t('domain.auto.62e951a692')" width="130"><template #default="{ row }"><el-tag>{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('domain.auto.62a7c63929')" width="110"><template #default="{ row }"><el-tag :type="priorityType(payload(row).priority)">{{ priorityLabel(payload(row).priority) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('ticket.serviceRole.label')" width="120"><template #default="{ row }"><el-tag effect="light">{{ serviceRoleLabel(payload(row).service_role || payload(row).support_queue) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('domain.auto.974d383f36')" width="140"><template #default="{ row }">{{ ownerName(row.owner_id) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.cd99b225a0')" width="130"><template #default="{ row }">{{ languageLabel(payload(row).language || payload(row).locale) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.6102c5c9e6')" width="170"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.f3ea6d345e')" width="190" fixed="right"><template #default="{ row }"><div class="row-actions three" @click.stop><button class="action-btn view" @click="openDetail(row)"><Icon icon="ri:eye-line" /></button><button v-if="canEditRecord(row)" class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button><button v-if="canFollowRecord(row)" class="action-btn success" @click="openAction('note', row)"><Icon icon="ri:chat-check-line" /></button></div></template></el-table-column>
      </el-table>
      <div class="system-pagination"><span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span><el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" /></div>
    </section>

    <el-drawer v-model="detailDrawer" size="48%" :title="selectedRecord?.name || t('domain.auto.90e24acc91')" destroy-on-close>
      <div v-if="selectedRecord" class="lead-detail">
        <div class="lead-summary">
          <div><span>{{ t('domain.auto.62e951a692') }}</span><strong>{{ statusLabel(selectedRecord.status) }}</strong></div>
          <div><span>{{ t('domain.auto.62a7c63929') }}</span><strong>{{ priorityLabel(selectedPayload.priority) }}</strong></div>
          <div><span>{{ t('ticket.serviceRole.label') }}</span><strong>{{ serviceRoleLabel(selectedPayload.service_role || selectedPayload.support_queue) }}</strong></div>
          <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(selectedRecord.owner_id) }}</strong></div>
          <div><span>{{ t('domain.auto.f2992377d6') }}</span><strong>{{ selectedPayload.customer_grade || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.969d961c63') }}</span><strong>{{ selectedPayload.order_id || selectedRecord.deal_id || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.cbf170ed3b') }}</span><strong>{{ selectedPayload.language || selectedPayload.locale || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.49af5ada33') }}</span><strong>{{ selectedPayload.bug_task_id || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.bda8cda9ff') }}</span><strong>{{ bugStatusLabel(selectedPayload.bug_status) }}</strong></div>
        </div>
        <div class="detail-actions">
          <el-button v-if="canEditRecord(selectedRecord)" @click="openEdit(selectedRecord)">{{ t('domain.auto.a70e2987f8') }}</el-button>
          <el-button v-if="canFollowRecord(selectedRecord)" @click="openAction('note')">{{ t('domain.auto.71a58cc706') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('status')">{{ t('domain.auto.f92064cffd') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('transfer')">{{ t('domain.auto.e18b64d93a') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="autoAssignTicket">{{ t('domain.auto.8871d101f4') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('upsell')">{{ t('domain.auto.8335753549') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('bug')">{{ t('domain.auto.ab266e40e3') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord) && selectedPayload.bug_task_id" @click="openAction('bug_status')">{{ t('domain.auto.c348cd66c6') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="createKnowledge">{{ t('domain.auto.ee13c2f5b3') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" type="success" plain @click="openAction('close')">{{ t('domain.auto.3b810106e7') }}</el-button>
        </div>
        <el-divider>{{ t('domain.auto.1a687c311d') }}</el-divider>
        <p class="crm-inline-help">{{ t('domain.auto.b011b1310f') }}</p>
        <div class="crm-line-list">
          <div v-for="item in knowledgeSuggestions" :key="item.id" class="crm-line-item">
            <strong>{{ item.name }}</strong>
            <span>{{ payload(item).question || payload(item).answer || t('domain.auto.ca2cd83b63') }}</span>
          </div>
        </div>
        <el-empty v-if="!knowledgeSuggestions.length" :description="t('domain.auto.1b064a035a')" />
        <el-divider>{{ t('domain.auto.cced177f01') }}</el-divider>
        <el-timeline><el-timeline-item v-for="item in timeline" :key="item.id" :timestamp="formatTime(item.created_at)"><strong>{{ payload(item).summary || item.name }}</strong><small class="timeline-operator">{{ timelineOperator(item) }}</small><p>{{ payload(item).reason || payload(item).content || payload(item).summary || '-' }}</p></el-timeline-item></el-timeline>
        <el-empty v-if="!timeline.length" :description="t('domain.auto.6246113cc2')" />
        <el-divider>{{ t('domain.auto.1dabf50650') }}</el-divider>
        <p class="crm-inline-help">{{ t('domain.auto.127c029ce9') }}</p>
        <div class="detail-actions"><el-button v-if="canFollowRecord(selectedRecord)" type="primary" @click="openTicketFileDialog">{{ t('domain.auto.6a58757e1b') }}</el-button></div>
        <el-table :data="ticketFiles" class="system-table" row-key="id">
          <el-table-column prop="name" :label="t('domain.auto.49deaf7da2')" min-width="220" />
          <el-table-column :label="t('domain.auto.435c5259e4')" width="120"><template #default="{ row }">{{ ticketFileCategoryLabel(payload(row).category) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.61890b022c')" width="110"><template #default="{ row }">{{ ticketFileVisibilityLabel(payload(row).visibility) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.fd20702c73')" width="100"><template #default="{ row }">{{ formatSize(payload(row).file_size) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="140"><template #default="{ row }"><el-button link type="primary" @click="downloadTicketFile(row)">{{ t('domain.auto.df5766f734') }}</el-button><el-button v-if="canEditRecord(selectedRecord)" link type="danger" @click="deleteTicketFile(row)">{{ t('domain.auto.3755f56f2f') }}</el-button></template></el-table-column>
        </el-table>
        <el-empty v-if="!ticketFiles.length" :description="t('domain.auto.b883e12cdf')" />
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.a70e2987f8') : t('domain.auto.4fc04ec282')" width="min(980px, 96vw)" destroy-on-close>
      <el-form :model="form" label-width="96px" class="lead-dialog-form">
        <el-form-item :label="t('domain.auto.a5cf6a19bd')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('domain.auto.e941d410f4')"><el-input v-model="form.payload.company" /></el-form-item>
        <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="form.status"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.b7c588c460')"><el-input v-model="form.payload.category" :placeholder="t('domain.auto.5eed50a7f3')" /></el-form-item>
        <el-form-item :label="t('domain.auto.644ee1c5ee')"><el-input v-model="form.payload.product_line" :placeholder="t('domain.auto.cf2867d8f1')" /></el-form-item>
        <el-form-item :label="t('domain.auto.cbf170ed3b')">
          <el-select v-model="form.payload.language" clearable :placeholder="t('domain.auto.5f5f2ea6ad')">
            <el-option v-for="item in languageOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('ticket.serviceRole.field')" class="wide-field">
          <div class="ticket-role-field">
            <el-select v-model="form.payload.service_role" class="full-input">
              <el-option v-for="item in serviceRoleOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <small>{{ t('ticket.serviceRole.help') }}</small>
          </div>
        </el-form-item>
        <el-form-item :label="t('domain.auto.62a7c63929')"><el-select v-model="form.payload.priority"><el-option :label="t('domain.auto.b096b3f5ac')" value="high" /><el-option :label="t('domain.auto.0869071c92')" value="normal" /><el-option :label="t('domain.auto.b9ee259b7f')" value="low" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.f2992377d6')"><el-select v-model="form.payload.customer_grade"><el-option label="VIP" value="VIP" /><el-option :label="t('domain.auto.b7f4670752')" value="important" /><el-option :label="t('domain.auto.7cda072d45')" value="normal" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="form.owner_id" clearable filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.e8835da11a')" class="wide-field"><el-input v-model="form.payload.description" type="textarea" :rows="4" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="formDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitForm">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="actionDialog" :title="actionTitle" width="640px" destroy-on-close>
      <el-form :model="actionForm" label-width="110px">
        <el-form-item v-if="actionType === 'status'" :label="t('domain.auto.62e951a692')"><el-select v-model="actionForm.status" class="full-input"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item v-if="actionType === 'assign'" :label="t('domain.auto.3e132a2a8e')"><el-select v-model="actionForm.owner_id" filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item v-if="actionType === 'transfer'" :label="t('domain.auto.32653c0221')"><el-select v-model="actionForm.owner_id" filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item v-if="actionType === 'assign' || actionType === 'transfer'" :label="t('ticket.serviceRole.field')"><el-select v-model="actionForm.service_role" class="full-input"><el-option v-for="item in serviceRoleOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item v-if="actionType === 'note'" :label="t('domain.auto.e4e46c7235')"><el-select v-model="actionForm.visibility" class="full-input"><el-option :label="t('domain.auto.c1c558ba57')" value="internal" /><el-option :label="t('domain.auto.aedfcdfe67')" value="public" /></el-select></el-form-item>
        <el-form-item v-if="actionType === 'upsell'" :label="t('domain.auto.84b592481d')"><el-input v-model="actionForm.opportunity_name" /></el-form-item>
        <el-form-item v-if="actionType === 'upsell'" :label="t('domain.auto.ee24edaa10')"><el-input-number v-model="actionForm.amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item v-if="actionType === 'bug'" :label="t('domain.auto.6b6375e99b')"><el-input v-model="actionForm.bug_name" /></el-form-item>
        <el-form-item v-if="actionType === 'bug'" :label="t('domain.auto.d918e4778a')">
          <el-select v-model="actionForm.severity" class="full-input">
            <el-option :label="t('domain.auto.6d715a9fa1')" value="critical" />
            <el-option :label="t('domain.auto.b096b3f5ac')" value="high" />
            <el-option :label="t('domain.auto.0869071c92')" value="medium" />
            <el-option :label="t('domain.auto.b9ee259b7f')" value="low" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'bug'" :label="t('domain.auto.27713f242f')"><el-input v-model="actionForm.module" :placeholder="t('domain.auto.0b64c07e5d')" /></el-form-item>
        <el-form-item v-if="actionType === 'bug'" :label="t('domain.auto.5ab8ecbd78')"><el-select v-model="actionForm.owner_id" clearable filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item v-if="actionType === 'bug_status'" :label="t('domain.auto.bda8cda9ff')">
          <el-select v-model="actionForm.bug_status" class="full-input">
            <el-option :label="t('domain.auto.59a9eb4e65')" value="open" />
            <el-option :label="t('domain.auto.fcb979ef0b')" value="in_progress" />
            <el-option :label="t('domain.auto.50138c31ac')" value="fixed" />
            <el-option :label="t('domain.auto.f628761bf5')" value="closed" />
            <el-option :label="t('domain.auto.c753c38b3b')" value="reopened" />
            <el-option :label="t('domain.auto.11d1528f68')" value="wontfix" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'bug'" :label="t('domain.auto.21264a13eb')"><el-input v-model="actionForm.bug_description" type="textarea" :rows="3" :placeholder="t('domain.auto.3c85b051f8')" /></el-form-item>
        <el-alert v-if="actionType === 'close'" type="info" :closable="false" :title="t('ticket.closeEvaluation.help')" />
        <el-form-item v-if="actionType === 'close'" :label="t('ticket.closeEvaluation.csat')">
          <div class="ticket-score-field"><el-input-number v-model="actionForm.csat" :min="1" :max="5" class="full-input" /><small>{{ t('ticket.closeEvaluation.csatHelp') }}</small></div>
        </el-form-item>
        <el-form-item v-if="actionType === 'close'" label="NPS">
          <div class="ticket-score-field"><el-input-number v-model="actionForm.nps" :min="0" :max="10" class="full-input" /><small>{{ t('ticket.closeEvaluation.npsHelp') }}</small></div>
        </el-form-item>
        <el-form-item :label="t('domain.auto.cd612544e1')" required><el-input v-model="actionForm.reason" type="textarea" :rows="4" :placeholder="t('domain.auto.e52e05145d')" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="actionDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitAction">{{ t('domain.auto.f526c89937') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="ticketFileDialog" :title="t('domain.auto.7d8aadbe92')" width="620px" destroy-on-close>
      <el-form :model="ticketFileForm" label-width="100px">
        <el-form-item :label="t('domain.auto.21a6f5a8d9')" required><input type="file" @change="onTicketFileChange" /></el-form-item>
        <el-form-item :label="t('domain.auto.435c5259e4')">
          <el-select v-model="ticketFileForm.category" class="full-input">
            <el-option :label="t('domain.auto.49830b4f74')" value="screenshot" />
            <el-option :label="t('domain.auto.4de50894b8')" value="log" />
            <el-option :label="t('domain.auto.6ce563711b')" value="feedback" />
            <el-option :label="t('domain.auto.ca3c9e8022')" value="resolution" />
            <el-option :label="t('domain.auto.1a26edf94a')" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.61890b022c')">
          <el-select v-model="ticketFileForm.visibility" class="full-input">
            <el-option :label="t('domain.auto.d162da4eaf')" value="internal" />
            <el-option :label="t('domain.auto.b8692e7870')" value="customer_visible" />
            <el-option :label="t('domain.auto.c931219c30')" value="private" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.989d1affa0')"><el-input v-model="ticketFileForm.version" /></el-form-item>
        <el-form-item :label="t('domain.auto.e0361480e3')"><el-input v-model="ticketFileForm.remark" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="ticketFileDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" :loading="ticketFileUploading" @click="submitTicketFile">{{ t('domain.auto.a977650db1') }}</el-button></template>
    </el-dialog>

    <el-drawer v-model="transferDrawer" :title="t('domain.auto.bdb6144a3e')" size="560px">
      <p class="crm-inline-help">{{ t('ticket.transferBoard.help') }}</p>
      <div class="crm-stage-board">
        <button class="crm-stage-card"><strong>{{ serviceRoleLabel('l1_support') }}</strong><em>{{ transferStats.queue_counts?.l1_support || 0 }} {{ t('domain.auto.f7b2a6ee68') }}</em></button>
        <button class="crm-stage-card"><strong>{{ serviceRoleLabel('l2_support') }}</strong><em>{{ transferStats.queue_counts?.l2_support || 0 }} {{ t('domain.auto.f7b2a6ee68') }}</em></button>
        <button class="crm-stage-card"><strong>{{ serviceRoleLabel('product_manager') }}</strong><em>{{ transferStats.queue_counts?.product_manager || 0 }} {{ t('domain.auto.f7b2a6ee68') }}</em></button>
      </div>
      <div class="crm-line-list">
        <div v-for="item in transferQueue" :key="item.ticket.id" class="crm-line-item quote-line">
          <div>
            <strong>{{ item.ticket.name }}</strong>
            <span>{{ serviceRoleLabel(item.queue) }} · {{ ownerName(item.ticket.owner_id) }} · {{ item.is_overdue ? t('domain.auto.411ebfdad7') : t('domain.auto.6f1972e48e') }} · {{ t('ticket.transferBoard.holdMinutes', { minutes: item.hold_minutes }) }}</span>
          </div>
          <el-button v-if="canEditRecord(item.ticket)" size="small" type="primary" plain @click="openAction('transfer', item.ticket)">{{ t('domain.auto.caee4d659d') }}</el-button>
        </div>
      </div>
      <el-empty v-if="!transferQueue.length" :description="t('domain.auto.2eceaab71c')" />
      <el-divider>{{ t('domain.auto.822edb18cc') }}</el-divider>
      <div class="lead-summary">
        <div><span>{{ t('domain.auto.4eae2676e4') }}</span><strong>{{ transferStats.transfer_count || 0 }}</strong></div>
        <div><span>{{ t('domain.auto.b2fd34c0fe') }}</span><strong>{{ transferStats.reopen_after_transfer || 0 }}</strong></div>
        <div><span>{{ t('domain.auto.8a5d3b355c') }}</span><strong>{{ transferStats.overdue_after_transfer || 0 }}</strong></div>
        <div><span>{{ t('domain.auto.7cd236c607') }}</span><strong>{{ Number(transferStats.avg_handle_hours_transfer || 0).toFixed(1) }}h</strong></div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canEditRecord, canFollowRecord, canManageDomain } from '../../permissions'
import { confirmDangerAction } from '../../utils/confirm'
import { downloadProtectedFile, unwrapFileUploadResponse } from '../../utils/download'
import { timelineOperatorText } from '../../utils/timeline'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { apiErrorMessage } from '../../utils/api-error'

const rows = ref([])
const users = ref([])
const loading = ref(false)
const selectedRecord = ref(null)
const timeline = ref([])
const ticketFiles = ref([])
const knowledgeSuggestions = ref([])
const escalationQueue = ref([])
const transferQueue = ref([])
const transferStats = ref({})
const detailDrawer = ref(false)
const formDialog = ref(false)
const actionDialog = ref(false)
const ticketFileDialog = ref(false)
const transferDrawer = ref(false)
const ticketFileUploading = ref(false)
const selectedTicketUploadFile = ref(null)
const editingId = ref(null)
const actionType = ref('note')
const filters = reactive({ keyword: '', status: '', owner_id: undefined, language: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
const actionForm = reactive({
  status: 'processing',
  owner_id: 0,
  service_role: 'l1_support',
  visibility: 'internal',
  reason: '',
  opportunity_name: '',
  amount: 0,
  csat: 5,
  nps: 9,
  bug_name: '',
  severity: 'medium',
  module: '',
  bug_description: '',
  bug_status: 'in_progress'
})
const ticketFileForm = reactive({ category: 'other', visibility: 'internal', version: 'v1', remark: '' })
const statusOptions = [
  { value: 'new', label: t('domain.auto.0cda8d1c71'), desc: t('domain.auto.17e0f64701') },
  { value: 'assigned', label: t('domain.auto.732b2f5de1'), desc: t('domain.auto.d3c74618d4') },
  { value: 'processing', label: t('domain.auto.fcb979ef0b'), desc: t('domain.auto.9e26542c8f') },
  { value: 'waiting_customer', label: t('domain.auto.69b352c10e'), desc: t('domain.auto.65a7e6702b') },
  { value: 'resolved', label: t('domain.auto.5e045cba10'), desc: t('domain.auto.25a45621ed') },
  { value: 'closed', label: t('domain.auto.f628761bf5'), desc: t('domain.auto.59377733d8') },
  { value: 'reopened', label: t('domain.auto.c753c38b3b'), desc: t('domain.auto.8a55d8e30f') }
]
const languageOptions = [
  { label: '中文（简体）', value: 'zh-CN' },
  { label: '中文（繁体）', value: 'zh-TW' },
  { label: 'English', value: 'en-US' },
  { label: '日本語', value: 'ja-JP' },
  { label: '한국어', value: 'ko-KR' },
  { label: 'Deutsch', value: 'de-DE' },
  { label: 'Français', value: 'fr-FR' },
  { label: 'Español', value: 'es-ES' },
  { label: 'Português', value: 'pt-PT' },
  { label: 'Русский', value: 'ru-RU' },
  { label: 'العربية', value: 'ar-SA' },
  { label: 'ไทย', value: 'th-TH' }
]
const serviceRoleOptions = [
  { label: t('ticket.serviceRole.l1'), value: 'l1_support' },
  { label: t('ticket.serviceRole.l2'), value: 'l2_support' },
  { label: t('ticket.serviceRole.productManager'), value: 'product_manager' }
]
const selectedPayload = computed(() => payload(selectedRecord.value))
const canManage = computed(() => canManageDomain('tickets'))
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))
const actionTitle = computed(() => ({ note: t('domain.auto.71a58cc706'), status: t('domain.auto.471b84d6e0'), assign: t('domain.auto.d119f14855'), transfer: t('domain.auto.e18b64d93a'), upsell: t('domain.auto.8335753549'), bug: t('domain.auto.ab266e40e3'), bug_status: t('domain.auto.c348cd66c6'), close: t('domain.auto.d372c6fb31') }[actionType.value] || t('domain.auto.29fa84ab96')))

onMounted(async () => { await Promise.all([loadUsers(), loadPage()]) })
async function loadUsers() { try { const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } }); users.value = res.data?.items || [] } catch { users.value = [] } }
async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/tickets/objects', { params: cleanParams({ object_type: 'ticket', ...filters, page: pagination.page, size: pagination.size }) })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
    if (selectedRecord.value) selectedRecord.value = rows.value.find((item) => item.id === selectedRecord.value.id) || selectedRecord.value
  } catch (error) { ElMessage.error(apiErrorMessage(error, t('domain.auto.5e69732380'))) } finally { loading.value = false }
}
function openCreate() { editingId.value = null; Object.assign(form, defaultForm()); formDialog.value = true }
function openEdit(row) { editingId.value = row.id; const p = payload(row); const serviceRole = serviceRoleValue(p.service_role || p.support_queue); Object.assign(form, { object_type: 'ticket', name: row.name, code: row.code || '', status: row.status || 'new', owner_id: row.owner_id || 0, amount: row.amount || 0, currency: row.currency || defaultBusinessCurrency(), score: row.score || 0, tags: splitTags(row.tags), payload: { priority: 'normal', customer_grade: 'normal', product_line: '', language: '', ...p, service_role: serviceRole, support_queue: serviceRole } }); formDialog.value = true }
async function submitForm() { if (!form.name) return ElMessage.error(t('domain.auto.f062b639ec')); try { if (editingId.value) await api.put(`/api/v1/prd/tickets/objects/${editingId.value}`, form); else await api.post('/api/v1/prd/tickets/objects', form); ElMessage.success(t('domain.auto.1392649ca8')); formDialog.value = false; await loadPage() } catch (error) { ElMessage.error(apiErrorMessage(error, t('domain.auto.b412eb86ab'))) } }
async function openDetail(row) { selectedRecord.value = row; detailDrawer.value = true; await Promise.all([loadTimeline(row), loadKnowledgeSuggestions(row), loadTicketFiles(row)]) }
async function loadTimeline(row) { const rel = await api.get('/api/v1/prd/relations', { params: { object_type: 'tickets', object_id: row.id, relation_type: 'timeline' } }); const ids = (rel.data?.items || []).map((item) => item.target_type === 'contacts' ? item.target_id : item.source_id); const items = await Promise.all(ids.map((id) => api.get(`/api/v1/prd/contacts/objects/${id}`).catch(() => null))); timeline.value = items.map((res) => res?.data).filter(Boolean) }
async function loadKnowledgeSuggestions(row) { try { const res = await api.get(`/api/v1/prd/flows/tickets/${row.id}/knowledge-suggestions`); knowledgeSuggestions.value = res.data?.items || [] } catch { knowledgeSuggestions.value = [] } }
async function loadTicketFiles(row) { try { const res = await api.get(`/api/v1/prd/flows/tickets/${row.id}/files`); ticketFiles.value = res.data?.items || [] } catch { ticketFiles.value = [] } }
function openAction(type, row = selectedRecord.value) {
  if (!row) return
  selectedRecord.value = row
  actionType.value = type
  const rowPayload = payload(row)
  Object.assign(actionForm, {
    status: nextStatus(row.status),
    owner_id: row.owner_id || 0,
    service_role: serviceRoleValue(rowPayload.service_role || rowPayload.support_queue),
    visibility: 'internal',
    reason: '',
    opportunity_name: `${t('domain.auto.8d446933b4')}${row.name}`,
    amount: 0,
    csat: 5,
    nps: 9,
    bug_name: `${t('domain.auto.ca2332f67a')}${row.name}`,
    severity: rowPayload.bug_severity || (rowPayload.priority === 'urgent' || rowPayload.priority === 'high' ? 'high' : 'medium'),
    module: rowPayload.bug_module || rowPayload.category || '',
    bug_description: rowPayload.description || '',
    bug_status: rowPayload.bug_status || 'in_progress'
  })
  actionDialog.value = true
}
async function submitAction() {
  if (!actionForm.reason) return ElMessage.error(t('domain.auto.bd59f8c970'))
  const row = selectedRecord.value
  try {
    if (actionType.value === 'note') await api.post(`/api/v1/prd/flows/tickets/${row.id}/note`, { visibility: actionForm.visibility, content: actionForm.reason })
    else if (actionType.value === 'status') await api.post(`/api/v1/prd/flows/tickets/${row.id}/status`, { status: actionForm.status, reason: actionForm.reason })
    else if (actionType.value === 'assign') await api.post(`/api/v1/prd/flows/tickets/${row.id}/assign`, { owner_id: actionForm.owner_id, service_role: actionForm.service_role, reason: actionForm.reason })
    else if (actionType.value === 'transfer') await api.post(`/api/v1/prd/flows/tickets/${row.id}/transfer`, { owner_id: actionForm.owner_id, service_role: actionForm.service_role, reason: actionForm.reason })
    else if (actionType.value === 'upsell') { const res = await api.post(`/api/v1/prd/flows/tickets/${row.id}/opportunity`, { name: actionForm.opportunity_name, amount: actionForm.amount, reason: actionForm.reason }); ElMessage.success(`${t('domain.auto.f1d9682643')}${res.data?.opportunity?.name || actionForm.opportunity_name}`) }
    else if (actionType.value === 'bug') {
      const res = await api.post(`/api/v1/prd/flows/tickets/${row.id}/bug`, { name: actionForm.bug_name, severity: actionForm.severity, module: actionForm.module, description: actionForm.bug_description, reason: actionForm.reason, owner_id: actionForm.owner_id || 0 })
      ElMessage.success(`${t('domain.auto.6ebe282201')}${res.data?.bug_task?.name || actionForm.bug_name}`)
    } else if (actionType.value === 'bug_status') await api.post(`/api/v1/prd/flows/tickets/${row.id}/bug-status`, { status: actionForm.bug_status, reason: actionForm.reason, sync_bug: true })
    else if (actionType.value === 'close') await api.post(`/api/v1/prd/flows/tickets/${row.id}/close`, { csat: actionForm.csat, nps: actionForm.nps, comment: actionForm.reason })
    actionDialog.value = false
    await loadPage()
    selectedRecord.value = rows.value.find((item) => item.id === row.id) || row
    await Promise.all([loadTimeline(selectedRecord.value), loadKnowledgeSuggestions(selectedRecord.value), loadTicketFiles(selectedRecord.value)])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca37aec661')))
  }
}
async function autoAssignTicket() {
  if (!selectedRecord.value) return
  const p = payload(selectedRecord.value)
  try {
    await api.post(`/api/v1/prd/flows/tickets/${selectedRecord.value.id}/auto-assign`, { service_role: serviceRoleValue(p.service_role || p.support_queue), reason: t('domain.auto.c33d7a9e4e') })
    ElMessage.success(t('domain.auto.19ad4b98b1'))
    await loadPage()
    selectedRecord.value = rows.value.find((item) => item.id === selectedRecord.value.id) || selectedRecord.value
    await loadTimeline(selectedRecord.value)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca37aec661')))
  }
}
function openTicketFileDialog() {
  selectedTicketUploadFile.value = null
  Object.assign(ticketFileForm, { category: 'other', visibility: 'internal', version: 'v1', remark: '' })
  ticketFileDialog.value = true
}
function onTicketFileChange(event) { selectedTicketUploadFile.value = event.target.files?.[0] || null }
async function submitTicketFile() {
  if (!selectedRecord.value) return
  if (!selectedTicketUploadFile.value) return ElMessage.error(t('domain.auto.eba557edfa'))
  ticketFileUploading.value = true
  try {
    const data = new FormData()
    data.append('file', selectedTicketUploadFile.value)
    const upload = await api.post('/api/v1/files/upload', data)
    const info = unwrapFileUploadResponse(upload)
    await api.post(`/api/v1/prd/flows/tickets/${selectedRecord.value.id}/files`, { ...ticketFileForm, file_id: info.file_id, file_name: info.original_name || selectedTicketUploadFile.value.name, file_size: info.size || selectedTicketUploadFile.value.size, file_type: selectedTicketUploadFile.value.type, storage_node_id: info.storage_node_id, download_url: info.download_url })
    ElMessage.success(t('domain.auto.9bb2df781a'))
    ticketFileDialog.value = false
    await Promise.all([loadTicketFiles(selectedRecord.value), loadTimeline(selectedRecord.value)])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.85b15ff585')))
  } finally {
    ticketFileUploading.value = false
  }
}
async function downloadTicketFile(row) {
  const filePayload = payload(row)
  const fileID = filePayload.file_id
  const url = filePayload.download_url || (fileID ? `/api/v1/files/${fileID}/download` : '')
  try {
    await downloadProtectedFile(url, filePayload.file_name || row.name)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.7d58c3b853')))
  }
}
async function deleteTicketFile(row) {
  if (!selectedRecord.value) return
  try {
    await confirmDangerAction(`${t('domain.auto.3a61f7a246')}${row.name || payload(row).file_name || t('domain.auto.a5e5e5853e')}${t('domain.auto.30baaa0f0c')}`, t('domain.auto.8c8daaea68'))
    await api.delete(`/api/v1/prd/flows/tickets/${selectedRecord.value.id}/files/${row.id}`, { skipDeleteConfirm: true })
    ElMessage.success(t('domain.auto.67f35d5091'))
    await Promise.all([loadTicketFiles(selectedRecord.value), loadTimeline(selectedRecord.value)])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.e8e675b522')))
  }
}
async function createKnowledge() {
  if (!selectedRecord.value) return
  try {
    await api.post(`/api/v1/prd/flows/tickets/${selectedRecord.value.id}/knowledge`)
    ElMessage.success(t('domain.auto.30012f37ec'))
    await loadTimeline(selectedRecord.value)
    await loadKnowledgeSuggestions(selectedRecord.value)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ca37aec661')))
  }
}
async function openTransferBoard() {
  try {
    const [queueRes, statsRes] = await Promise.all([
      api.get('/api/v1/prd/flows/tickets/transfer-queue', { params: { limit: 80, group_by: 'queue' } }),
      api.get('/api/v1/prd/flows/tickets/transfer-stats', { params: { days: 30 } })
    ])
    transferQueue.value = queueRes.data?.items || []
    transferStats.value = statsRes.data || {}
    transferDrawer.value = true
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.f6b7a41e56')))
  }
}
function quickStatus(status) { filters.status = filters.status === status ? '' : status; loadPage() }
function resetFilters() { Object.assign(filters, { keyword: '', status: '', owner_id: undefined, language: '' }); pagination.page = 1; loadPage() }
function defaultForm() { return { object_type: 'ticket', name: '', code: '', status: 'new', owner_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { company: '', category: '', product_line: '', language: '', service_role: 'l1_support', support_queue: 'l1_support', priority: 'normal', customer_grade: 'normal', description: '' } } }
function payload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function statusLabel(value) { return statusOptions.find((item) => item.value === value)?.label || value || '-' }
function nextStatus(status) { return ({ new: 'assigned', assigned: 'processing', processing: 'waiting_customer', waiting_customer: 'resolved', resolved: 'closed', closed: 'reopened', reopened: 'processing' }[status] || 'processing') }
function statusCount(status) { return rows.value.filter((item) => item.status === status).length }
function priorityLabel(value) { return ({ high: t('domain.auto.b096b3f5ac'), urgent: t('domain.auto.b096b3f5ac'), normal: t('domain.auto.0869071c92'), low: t('domain.auto.b9ee259b7f') }[value] || t('domain.auto.0869071c92')) }
function priorityType(value) { return value === 'high' || value === 'urgent' ? 'danger' : value === 'low' ? 'info' : 'warning' }
function bugStatusLabel(value) { return ({ open: t('domain.auto.59a9eb4e65'), in_progress: t('domain.auto.fcb979ef0b'), fixed: t('domain.auto.50138c31ac'), closed: t('domain.auto.f628761bf5'), reopened: t('domain.auto.c753c38b3b'), wontfix: t('domain.auto.11d1528f68') }[value] || '-') }
function queueLabel(value) {
  const text = String(value || '').trim()
  if (!text || text === '<nil>' || text === 'nil') return t('ticket.owner.unassigned')
  return ({ l1_support: t('domain.auto.9cca37c454'), l2_support: t('domain.auto.d055eb7f23'), product_manager: t('domain.auto.775f14fc80') }[text] || text)
}
function serviceRoleValue(value) {
  const text = String(value || '').trim()
  return serviceRoleOptions.some((item) => item.value === text) ? text : 'l1_support'
}
function serviceRoleLabel(value) {
  return serviceRoleOptions.find((item) => item.value === serviceRoleValue(value))?.label || t('ticket.serviceRole.l1')
}
function languageLabel(value) {
  const text = String(value || '').trim()
  if (!text || text === '<nil>' || text === 'nil') return '-'
  return languageOptions.find((item) => item.value === text)?.label || text
}
function slaText(row) {
  const due = payload(row).sla_due_at
  if (!due) return t('domain.auto.55a04b58cd')
  if (isOverdue(row)) return `${t('domain.auto.411ebfdad7')}${formatTime(due)}`
  if (isNearDue(row)) return `${t('domain.auto.ef8d36f143')}${formatTime(due)}`
  return `${t('domain.auto.7f090d0a3e')}${formatTime(due)}`
}
function isOverdue(row) { const due = new Date(payload(row).sla_due_at || ''); return !Number.isNaN(due.getTime()) && Date.now() > due.getTime() && !['closed', 'resolved'].includes(row.status) }
function isNearDue(row) {
  const due = new Date(payload(row).sla_due_at || '')
  if (Number.isNaN(due.getTime()) || ['closed', 'resolved'].includes(row.status)) return false
  const delta = due.getTime() - Date.now()
  return delta > 0 && delta <= 15 * 60 * 1000
}
function ticketRowClass({ row }) {
  if (selectedRecord.value && Number(selectedRecord.value.id) === Number(row.id)) return 'is-selected-row'
  return isOverdue(row) ? 'row-danger' : isNearDue(row) ? 'row-warning' : ''
}
function ownerName(ownerID) { const id = Number(ownerID || 0); if (!id) return t('domain.auto.3bb92e6efc'); const user = users.value.find((item) => Number(item.id) === id); return user ? userDisplayName(user) : `${t('domain.auto.9ba763ea34')}${id}` }
function userDisplayName(user) { const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`; return user.department_name ? `${label} · ${user.department_name}` : label }
function timelineOperator(item) { return timelineOperatorText(item, users.value, formatTime) }
function ticketFileCategoryLabel(value) { return ({ screenshot: t('domain.auto.49830b4f74'), log: t('domain.auto.4de50894b8'), feedback: t('domain.auto.6ce563711b'), resolution: t('domain.auto.ca3c9e8022'), other: t('domain.auto.1a26edf94a') }[value] || t('domain.auto.1a26edf94a')) }
function ticketFileVisibilityLabel(value) { return ({ internal: t('domain.auto.d162da4eaf'), customer_visible: t('domain.auto.b8692e7870'), private: t('domain.auto.c931219c30') }[value] || t('domain.auto.d162da4eaf')) }
function formatSize(value) { const n = Number(value || 0); if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`; if (n >= 1024) return `${(n / 1024).toFixed(1)} KB`; return `${n} B` }
function formatTime(value) { if (!value) return '-'; const date = new Date(value); return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}` }
</script>

<style scoped>
.timeline-operator {
  display: block;
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  line-height: 1.4;
}

.sla-warning {
  color: #d97706;
  font-weight: 600;
}

.ticket-role-field,
.ticket-score-field {
  width: 100%;
  display: grid;
  gap: 6px;
}

.ticket-role-field small,
.ticket-score-field small {
  color: #667085;
  font-size: 12px;
  line-height: 1.6;
}

.ticket-search-form {
  grid-template-columns: minmax(260px, 1.1fr) minmax(220px, 1fr) minmax(240px, 1fr) minmax(180px, 0.75fr) auto;
}

.ticket-search-form .system-search-actions {
  align-self: center;
}

@media (max-width: 1280px) {
  .ticket-search-form {
    grid-template-columns: repeat(2, minmax(240px, 1fr));
  }

  .ticket-search-form .system-search-actions {
    justify-content: flex-start;
  }
}
</style>
