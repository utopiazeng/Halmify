<template>
  <div class="system-page crm-flow-page orders-enhanced-page">
    <section class="crm-help-strip"><span>{{ t('domain.auto.2e5fad71fd') }}</span></section>

    <section class="system-card">
      <div class="list-card-search">
        <el-form :model="filters" label-width="72px" class="system-search-form order-search-form">
          <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.0a3b88ea03')" clearable /></el-form-item>
          <el-form-item :label="t('orders.company')"><el-input v-model="filters.company_name" :placeholder="t('orders.searchCompany')" clearable /></el-form-item>
          <el-form-item :label="t('orders.contact')"><el-input v-model="filters.contact_name" :placeholder="t('orders.searchContact')" clearable /></el-form-item>
          <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="filters.status" clearable :placeholder="t('domain.auto.1a4c26d92d')"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="filters.owner_id" clearable filterable :placeholder="t('domain.auto.8b134548e7')"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
          <el-form-item :label="t('orders.priority')"><el-select v-model="filters.priority" clearable :placeholder="t('orders.allPriority')"><el-option v-for="item in orderPriorityOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item :label="t('orders.amountRange')" class="amount-range-field"><div class="amount-range-inputs"><el-input-number v-model="filters.min_amount" :min="0" controls-position="right" :placeholder="t('orders.minAmount')" /><span>-</span><el-input-number v-model="filters.max_amount" :min="0" controls-position="right" :placeholder="t('orders.maxAmount')" /></div></el-form-item>
          <div class="system-search-actions">
            <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
            <el-button type="primary" @click="searchPage">{{ t('domain.auto.711363c424') }}</el-button>
          </div>
        </el-form>
      </div>
      <div class="crm-stage-board">
        <button class="crm-stage-card" :class="{ active: !filters.status }" @click="quickStatus('')">
          <strong>{{ t('domain.auto.1a4c26d92d') }}</strong>
          <span>{{ t('domain.auto.416a46f1d6') }}</span>
          <em>{{ pagination.total }} {{ t('domain.auto.ef533cc386') }}</em>
        </button>
        <button v-for="item in statusOptions" :key="item.value" class="crm-stage-card" :class="{ active: filters.status === item.value }" @click="quickStatus(item.value)">
          <strong>{{ item.label }}</strong>
          <span>{{ item.desc }}</span>
          <em>{{ statusCount(item.value) }} {{ t('domain.auto.ef533cc386') }} {{ formatAmount(statusAmount(item.value)) }}</em>
        </button>
      </div>
      <div class="system-toolbar">
        <div class="toolbar-left">
          <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.9d5f0710a5') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('status')">{{ t('domain.auto.f92064cffd') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('pay')">{{ t('domain.auto.6453d96518') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('payment_plan')">{{ t('domain.auto.3863f9a8f8') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('invoice')">{{ t('domain.auto.423e945f20') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openAction('ticket')">{{ t('domain.auto.6aee271b65') }}</el-button>
          <el-button v-if="canCancelOrder && canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="danger" plain @click="openAction('cancel')">{{ t('domain.auto.3eedd32d80') }}</el-button>
          <el-button v-if="canManage" plain @click="openFinanceDialog">{{ t('domain.auto.58a40c91af') }}</el-button>
        </div>
        <div class="toolbar-icons">
          <el-tooltip :content="t('domain.auto.38108eaa1d')">
            <button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage">
              <Icon icon="ri:refresh-line" />
            </button>
          </el-tooltip>
        </div>
      </div>

      <el-table
        :data="rows"
        class="system-table order-list-table"
        row-key="id"
        v-loading="loading"
        highlight-current-row
        :current-row-key="selectedRecord?.id"
        :row-class-name="orderRowClass"
        @row-click="selectRecord"
      >
        <el-table-column prop="name" :label="t('domain.auto.81b3798beb')" min-width="220">
          <template #default="{ row }">
            <div class="record-title">
              <strong>{{ row.name }} <el-tag v-if="objectVisibility(row) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
              <span>{{ orderCompany(row) }} · {{ row.code || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('orders.company')" min-width="160"><template #default="{ row }">{{ orderCompany(row) }}</template></el-table-column>
        <el-table-column :label="t('orders.contact')" min-width="140"><template #default="{ row }">{{ orderContact(row) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.62e951a692')" width="140"><template #default="{ row }"><el-tag>{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('orders.importance')" width="120"><template #default="{ row }"><el-tag :type="priorityType(payload(row).priority)">{{ priorityLabel(payload(row).priority) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('domain.auto.34943c40c9')" width="140"><template #default="{ row }">{{ formatAmount(row.amount, row.currency) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.2723c655b8')" width="130"><template #default="{ row }">{{ formatAmount(payload(row).paid_amount || 0, row.currency) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.4f6ce26224')" width="130"><template #default="{ row }">{{ formatAmount(receivable(row), row.currency) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.fa5c1fcbac')" width="130"><template #default="{ row }"><span :class="{ 'sla-overdue': Number(payload(row).overdue_amount || 0) > 0 }">{{ formatAmount(payload(row).overdue_amount || 0, row.currency) }}</span></template></el-table-column>
        <el-table-column :label="t('domain.auto.2ec852451a')" width="140"><template #default="{ row }">{{ formatAmount(payload(row).base_amount || row.amount, payload(row).base_currency || financeForm.base_currency) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.974d383f36')" width="140"><template #default="{ row }">{{ ownerName(row.owner_id) }}</template></el-table-column>
        <el-table-column :label="t('domain.auto.f3ea6d345e')" width="190" fixed="right">
          <template #default="{ row }">
            <div class="row-actions three" @click.stop>
              <button class="action-btn view" @click="openDetail(row)"><Icon icon="ri:eye-line" /></button>
              <button v-if="canEditRecord(row)" class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
              <button v-if="canEditRecord(row)" class="action-btn success" @click="openAction('pay', row)"><Icon icon="ri:coins-line" /></button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="system-pagination">
        <span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span>
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
      </div>
    </section>

    <el-drawer v-model="detailDrawer" size="52%" :title="selectedRecord?.name || t('domain.auto.dea71f6fbe')" destroy-on-close>
      <div v-if="selectedRecord" class="lead-detail">
        <div class="lead-summary">
          <div><span>{{ t('domain.auto.62e951a692') }}</span><strong>{{ statusLabel(selectedRecord.status) }}</strong></div>
          <div><span>{{ t('orders.company') }}</span><strong>{{ orderCompany(selectedRecord) }}</strong></div>
          <div><span>{{ t('orders.contact') }}</span><strong>{{ orderContact(selectedRecord) }}</strong></div>
          <div><span>{{ t('domain.auto.d98167d36d') }}</span><strong>{{ formatAmount(selectedRecord.amount, selectedRecord.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.2723c655b8') }}</span><strong>{{ formatAmount(selectedPayload.paid_amount || 0, selectedRecord.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.4f6ce26224') }}</span><strong>{{ formatAmount(receivable(selectedRecord), selectedRecord.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.d1accbcba9') }}</span><strong>{{ formatAmount(selectedPayload.tax_amount || 0, selectedRecord.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.2ec852451a') }}</span><strong>{{ formatAmount(selectedPayload.base_amount || selectedRecord.amount, selectedPayload.base_currency || financeForm.base_currency) }}</strong></div>
          <div><span>{{ t('domain.auto.c2ece89cff') }}</span><strong>{{ latestShipment(selectedRecord).shipment_no || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.0dabf6ac1b') }}</span><strong>{{ selectedPayload.invoice_no || selectedPayload.invoice_exemption_reason || '-' }}</strong></div>
        </div>
        <div v-if="completionWarnings.length" class="order-gate-alert">
          <Icon icon="ri:error-warning-line" />
          <span>{{ completionWarnings.join('，') }}</span>
        </div>
        <div v-if="canEditRecord(selectedRecord)" class="order-gate-checklist">
          <strong>{{ t('orders.shipmentRequirementsTitle') }}</strong>
          <div>
            <span v-for="item in shipmentRequirements(selectedRecord)" :key="item.label" :class="{ passed: item.passed }">
              <Icon :icon="item.passed ? 'ri:checkbox-circle-line' : 'ri:error-warning-line'" />
              {{ item.label }}
            </span>
          </div>
        </div>
        <div class="detail-actions">
          <el-button v-if="canEditRecord(selectedRecord)" @click="openEdit(selectedRecord)">{{ t('domain.auto.c62b1813d0') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('pay')">{{ t('domain.auto.6453d96518') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('payment_plan')">{{ t('domain.auto.3863f9a8f8') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" type="primary" :disabled="!canRegisterShipment(selectedRecord)" @click="openAction('ship')">{{ t('domain.auto.622bf9ab9e') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('invoice')">{{ t('domain.auto.fbbfacbc7e') }}</el-button>
          <el-button v-if="canEditRecord(selectedRecord)" @click="openAction('ticket')">{{ t('domain.auto.6aee271b65') }}</el-button>
          <el-button @click="downloadOrderBackup">{{ t('domain.auto.ae33ceb9f1') }}</el-button>
          <el-button v-if="canCancelOrder && canEditRecord(selectedRecord)" type="danger" plain @click="openAction('cancel')">{{ t('domain.auto.3eedd32d80') }}</el-button>
        </div>
        <ObjectAccessPanel
          domain="orders"
          :record="selectedRecord"
          :users="users"
          :can-manage="canManageRecordACL(selectedRecord)"
          :object-label="t('domain.auto.81b3798beb')"
          @updated="refreshSelected"
        />

        <el-divider>{{ t('domain.auto.91238956f1') }}</el-divider>
        <DealChainPanel object-type="orders" :object-id="selectedRecord.id" :users="users" />

        <el-divider>{{ t('domain.auto.21f15df3d4') }}</el-divider>
        <div class="crm-line-list">
          <div v-for="(item, index) in lineItems(selectedRecord)" :key="index" class="crm-line-item">
            <strong>{{ item.name || item.product_name || t('domain.auto.4b9c8904e9') }}</strong>
            <span>{{ item.quantity || 1 }} x {{ formatAmount(item.unit_price || item.price || 0, selectedRecord.currency) }} · {{ item.tax_code || t('domain.auto.d2dea18905') }} {{ item.tax_rate || 0 }}%</span>
          </div>
        </div>
        <el-empty v-if="!lineItems(selectedRecord).length" :description="t('domain.auto.1fb4350141')" />

        <el-divider>{{ t('domain.auto.3863f9a8f8') }}</el-divider>
        <p class="crm-inline-help">{{ t('domain.auto.f87be05a73') }}</p>
        <div class="crm-line-list">
          <div v-for="(item, index) in paymentPlan(selectedRecord)" :key="index" class="crm-line-item">
            <strong>{{ paymentPlanName(item, index) }}</strong>
            <span>{{ cleanPaymentPlanText(item.due_date) || '-' }} {{ t('domain.auto.d01b69cb3f') }} {{ formatAmount(item.amount || 0, item.currency || selectedRecord.currency) }}</span>
          </div>
        </div>
        <el-empty v-if="!paymentPlan(selectedRecord).length" :description="t('domain.auto.0854031a4f')" />

        <el-divider>{{ t('orders.shipmentRecords') }}</el-divider>
        <div class="crm-line-list">
          <div v-for="(item, index) in shipments(selectedRecord)" :key="`${item.shipment_no || index}-${item.shipped_at || index}`" class="crm-line-item shipment-line-item">
            <strong>{{ shipmentScopeLabel(item.shipment_scope) }} · {{ item.shipment_no || '-' }}</strong>
            <span>{{ item.carrier || '-' }} · {{ formatTime(item.shipped_at) }}<template v-if="item.delivery_note"> · {{ item.delivery_note }}</template></span>
          </div>
        </div>
        <el-empty v-if="!shipments(selectedRecord).length" :description="t('orders.emptyShipmentRecords')" />

        <el-divider>{{ t('domain.auto.0859f214e3') }}</el-divider>
        <div class="order-file-toolbar">
          <el-upload v-if="canFollowRecord(selectedRecord)" :show-file-list="false" :http-request="uploadOrderFile" :disabled="fileUploading">
            <el-button :loading="fileUploading">{{ t('domain.auto.da905dfb4f') }}</el-button>
          </el-upload>
          <el-button @click="loadOrderFiles(selectedRecord)" :loading="filesLoading">{{ t('domain.auto.f8ffa75cb3') }}</el-button>
        </div>
        <div class="crm-line-list">
          <div v-for="file in orderFiles" :key="file.id" class="crm-line-item order-file-item">
            <div>
              <strong>{{ file.name }}</strong>
              <span>{{ fileMeta(file) }}</span>
            </div>
            <div class="order-file-actions">
              <el-button text @click="downloadOrderFile(file)">{{ t('domain.auto.2b9d013177') }}</el-button>
              <el-button v-if="canEditRecord(selectedRecord)" text type="danger" @click="deleteOrderFile(file)">{{ t('domain.auto.3755f56f2f') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-if="!orderFiles.length" :description="t('domain.auto.e86b359d73')" />

        <el-divider>{{ t('domain.auto.7e2a4c2f48') }}</el-divider>
        <div class="order-timeline-toolbar">
          <el-button v-if="canFollowRecord(selectedRecord)" type="primary" plain @click="openTimelineDialog">{{ t('orders.addTimeline') }}</el-button>
          <el-button @click="loadTimeline(selectedRecord)">{{ t('orders.refreshTimeline') }}</el-button>
        </div>
        <el-timeline>
          <el-timeline-item v-for="item in timeline" :key="item.id" :timestamp="formatTime(item.created_at)">
            <strong>{{ timelineTitle(item) }}</strong>
            <small class="timeline-operator">{{ timelineOperator(item) }}</small>
            <p>{{ timelineSummary(item) }}</p>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-if="!timeline.length" :description="t('domain.auto.cf55876f35')" />
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.c62b1813d0') : t('domain.auto.9d5f0710a5')" width="min(1180px, 96vw)" destroy-on-close>
      <el-form :model="form" label-width="96px" class="lead-dialog-form order-dialog-form">
        <el-form-item :label="t('domain.auto.2905932150')" required><el-input v-model="form.name" /></el-form-item>
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
        <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="form.status"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.34943c40c9')"><el-input-number v-model="form.amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.a81ab5e100')">
          <el-select v-model="form.currency" filterable>
            <el-option v-for="item in orderCurrencyOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="form.owner_id" clearable filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('orders.importance')"><el-select v-model="form.payload.priority" class="full-input"><el-option v-for="item in orderPriorityOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.241d8ef92f')"><ObjectAccessField v-model="form.payload.visibility" :public-label="t('domain.auto.c4f1427223')" :private-label="t('domain.auto.6d9e3cdf28')" /></el-form-item>
        <el-form-item :label="t('domain.auto.1104c48130')">
          <el-select v-model="form.payload.payment_terms" filterable allow-create default-first-option class="full-input" :placeholder="t('orders.paymentTermsPlaceholder')">
            <el-option v-for="item in paymentTermOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.e301933ef5')" class="wide-field"><el-input v-model="form.payload.shipping_address" type="textarea" :rows="2" :placeholder="t('orders.shippingAddressPlaceholder')" /></el-form-item>
        <el-form-item :label="t('domain.auto.21f15df3d4')" class="wide-field">
          <div class="visual-step-list order-line-list">
            <div v-for="(item, index) in form.payload.line_items" :key="index" class="order-line-card">
              <div class="line-field product-field">
                <span>{{ t('orders.product') }}</span>
                <el-select v-model="item.product_id" filterable clearable :placeholder="t('domain.auto.2c30787264')" @change="applyProductToLine(item)">
                  <el-option v-for="product in products" :key="product.id" :label="product.name" :value="product.id">
                    <span>{{ product.name }}</span>
                    <small class="product-option-meta">{{ productInfo(product).sku || product.code || '-' }} {{ t('domain.auto.95e7b3fabb') }} {{ productInfo(product).stock || 0 }}</small>
                  </el-option>
                </el-select>
              </div>
              <div class="line-field name-field">
                <span>{{ t('orders.lineName') }}</span>
                <el-input v-model="item.name" :placeholder="t('domain.auto.cbcc51df5c')" />
              </div>
              <div class="line-field qty-field">
                <span>{{ t('orders.quantity') }}</span>
                <el-input-number v-model="item.quantity" :min="1" controls-position="right" class="full-input" />
              </div>
              <div class="line-field price-field">
                <span>{{ t('orders.unitPrice') }}</span>
                <el-input-number v-model="item.unit_price" :min="0" controls-position="right" class="full-input" />
              </div>
              <div class="line-field tax-code-field">
                <span>{{ t('orders.taxCode') }}</span>
                <el-select v-model="item.tax_code" clearable :placeholder="t('domain.auto.240a79fb8f')" @change="applyTaxCode(item)">
                  <el-option v-for="tax in financeForm.tax_codes" :key="tax.code" :label="tax.name" :value="tax.code" />
                </el-select>
              </div>
              <div class="line-field tax-rate-field">
                <span>{{ t('orders.taxRatePercent') }}</span>
                <el-input-number v-model="item.tax_rate" :min="0" :max="100" controls-position="right" class="full-input" />
              </div>
              <div class="line-card-footer">
                <span class="product-stock-note">{{ t('domain.auto.dfd6b7aab1') }} {{ item.stock_available ?? '-' }}</span>
                <el-button text type="danger" @click="removeLine(index)">{{ t('domain.auto.3755f56f2f') }}</el-button>
              </div>
            </div>
            <el-button plain @click="addLine">{{ t('domain.auto.20d602435a') }}</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="formDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitForm">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="actionDialog" :title="actionTitle" :width="actionDialogWidth" destroy-on-close>
      <el-form :model="actionForm" label-width="120px">
        <el-form-item v-if="actionType === 'status'" :label="t('domain.auto.62e951a692')"><el-select v-model="actionForm.status" class="full-input"><el-option v-for="item in availableStatusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-alert v-if="actionType === 'status' && actionForm.status === 'completed' && completionWarnings.length" type="warning" :closable="false" class="order-action-alert" :title="`${t('domain.auto.03e8781755')}${completionWarnings.join('，')}`" />
        <el-form-item v-if="actionType === 'pay'" :label="t('domain.auto.c60d7ff570')"><el-input-number v-model="actionForm.paid_amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item v-if="actionType === 'pay'" :label="t('domain.auto.71b3c1c9c8')"><el-input v-model="actionForm.payment_no" /></el-form-item>
        <el-form-item v-if="actionType === 'payment_plan'" :label="t('domain.auto.3863f9a8f8')" class="wide-field">
          <div class="visual-step-list">
            <div v-for="(item, index) in actionForm.payment_plan" :key="index" class="payment-plan-card">
              <el-input v-model="item.name" :placeholder="t('domain.auto.7d0cfb1797')" />
              <div class="payment-plan-amount">
                <el-input-number v-model="item.amount" :min="0" controls-position="right" class="full-input" />
                <el-select v-model="item.currency" class="payment-plan-currency" filterable :placeholder="normalizeCurrencyCode(selectedRecord?.currency)">
                  <el-option v-for="currency in orderCurrencyOptions" :key="currency.value" :label="currency.label" :value="currency.value" />
                </el-select>
              </div>
              <el-date-picker v-model="item.due_date" value-format="YYYY-MM-DD" class="full-input payment-plan-date" :placeholder="t('orders.dueDate')" />
              <el-button text type="danger" @click="removePlanLine(index)">{{ t('domain.auto.3755f56f2f') }}</el-button>
            </div>
            <el-button plain @click="addPlanLine">{{ t('domain.auto.53e22bd93c') }}</el-button>
          </div>
        </el-form-item>
        <p v-if="actionType === 'payment_plan'" class="crm-dialog-help">{{ t('domain.auto.6126fb6008') }}</p>
        <el-form-item v-if="actionType === 'ship'" :label="t('orders.shipmentType')">
          <el-segmented v-model="actionForm.shipment_scope" :options="shipmentScopeOptions" />
        </el-form-item>
        <el-form-item v-if="actionType === 'ship'" :label="t('domain.auto.c2ece89cff')"><el-input v-model="actionForm.shipment_no" /></el-form-item>
        <el-form-item v-if="actionType === 'ship'" :label="t('domain.auto.700285645a')"><el-input v-model="actionForm.carrier" /></el-form-item>
        <el-form-item v-if="actionType === 'ship'" :label="t('orders.shipmentRemark')"><el-input v-model="actionForm.delivery_note" type="textarea" :rows="3" :placeholder="t('orders.shipmentRemarkPlaceholder')" /></el-form-item>
        <el-form-item v-if="actionType === 'invoice'" :label="t('domain.auto.ada9a85eab')"><el-input v-model="actionForm.invoice_no" :placeholder="t('domain.auto.9cb71fc462')" /></el-form-item>
        <el-form-item v-if="actionType === 'invoice'" :label="t('domain.auto.3d0afdf373')">
          <el-select v-model="actionForm.invoice_type" filterable class="full-input">
            <el-option v-for="item in invoiceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'invoice'" :label="t('domain.auto.eaa8688894')"><el-input v-model="actionForm.invoice_exemption_reason" :placeholder="t('domain.auto.83a3dcd43f')" /></el-form-item>
        <el-alert v-if="actionType === 'ticket'" type="info" :closable="false" :title="t('orders.ticketCreateHint')" />
        <el-form-item v-if="actionType !== 'ticket'" :label="actionType === 'payment_plan' ? t('orders.remark') : t('domain.auto.cd612544e1')" :required="reasonRequired"><el-input v-model="actionForm.reason" type="textarea" :rows="4" :placeholder="actionType === 'payment_plan' ? t('orders.paymentPlanRemarkPlaceholder') : t('domain.auto.86b304f294')" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="actionDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" :loading="actionSubmitting" @click="submitAction">{{ t('domain.auto.f526c89937') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="timelineDialog" :title="t('orders.addOrderTimeline')" width="560px" destroy-on-close>
      <el-form :model="timelineForm" label-width="86px">
        <el-form-item :label="t('orders.timelineType')">
          <el-select v-model="timelineForm.type" class="full-input">
            <el-option :label="t('orders.timelineNote')" value="note" />
            <el-option :label="t('orders.timelineCall')" value="call" />
            <el-option :label="t('orders.timelineEmail')" value="email" />
            <el-option :label="t('orders.timelineMeeting')" value="meeting" />
            <el-option :label="t('orders.timelineFeedback')" value="feedback" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('orders.title')" required><el-input v-model="timelineForm.summary" :placeholder="t('orders.timelineTitlePlaceholder')" /></el-form-item>
        <el-form-item :label="t('orders.content')" required><el-input v-model="timelineForm.content" type="textarea" :rows="4" :placeholder="t('orders.timelineContentPlaceholder')" /></el-form-item>
        <el-form-item :label="t('orders.nextStep')"><el-input v-model="timelineForm.next_action" :placeholder="t('orders.nextStepPlaceholder')" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="timelineDialog = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="timelineSaving" @click="submitTimeline">{{ t('orders.saveTimeline') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="financeDialog" :title="t('domain.auto.58a40c91af')" width="min(1080px, 96vw)" destroy-on-close>
      <p class="crm-dialog-help">{{ t('domain.auto.97665af9eb') }}</p>
      <el-form :model="financeForm" label-width="100px" class="lead-dialog-form order-finance-form">
        <el-form-item :label="t('domain.auto.cae8c50ceb')">
          <el-select v-model="financeForm.base_currency" filterable>
            <el-option v-for="item in orderCurrencyOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.5ac4a4bf2f')" class="wide-field">
          <div class="currency-rate-grid">
            <div v-for="currency in currencyCodes" :key="currency" class="exchange-rate-card">
              <strong>{{ currency }}</strong>
              <el-input-number v-model="financeForm.exchange_rates[currency]" :min="0" :precision="4" controls-position="right" />
              <span>1 {{ currency }} {{ t('domain.auto.076cff140e') }} {{ financeForm.exchange_rates[currency] || 1 }} {{ financeForm.base_currency }}</span>
            </div>
          </div>
        </el-form-item>
        <el-form-item :label="t('orders.autoExchangeRate')" class="wide-field">
          <div class="finance-sync-row">
            <el-button plain :loading="exchangeSyncing" @click="syncExchangeRates">{{ t('orders.syncExchangeRate') }}</el-button>
            <span>{{ t('orders.exchangeRateHint') }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="t('domain.auto.240a79fb8f')" class="wide-field">
          <div class="visual-step-list">
            <div v-for="(tax, index) in financeForm.tax_codes" :key="index" class="tax-code-card">
              <el-input v-model="tax.code" :placeholder="t('domain.auto.fca6d55505')" />
              <el-input v-model="tax.name" :placeholder="t('domain.auto.75ae6a8a7d')" />
              <el-input-number v-model="tax.rate" :min="0" :max="100" controls-position="right" />
              <el-button text type="danger" @click="removeTaxCode(index)">{{ t('domain.auto.3755f56f2f') }}</el-button>
            </div>
            <el-button plain @click="addTaxCode">{{ t('domain.auto.6877fedab8') }}</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="financeDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="saveFinanceSetting">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../api'
import DealChainPanel from '../components/DealChainPanel.vue'
import ObjectAccessPanel from '../components/ObjectAccessPanel.vue'
import ObjectAccessField from '../components/ObjectAccessField.vue'
import { t } from '../i18n'
import { canEditRecord, canFollowRecord, canManageDomain, canManageRecordACL, hasExactPermission } from '../permissions'
import { confirmDangerAction } from '../utils/confirm'
import { currencyCodes, currencyOptions as orderCurrencyOptions, defaultExchangeRates, formatCurrencyAmount, normalizeCurrencyCode } from '../utils/currency'
import { defaultBusinessCurrency } from '../utils/company-format'
import { playCelebration } from '../utils/celebration'
import { downloadProtectedFile, unwrapFileUploadResponse } from '../utils/download'
import { timelineOperatorText } from '../utils/timeline'
import { useCustomerChainOptions } from '../utils/customerChain'
import { apiErrorMessage } from '../utils/api-error'

const route = useRoute()
const router = useRouter()
const rows = ref([])
const users = ref([])
const products = ref([])
const orderFiles = ref([])
const loading = ref(false)
const filesLoading = ref(false)
const fileUploading = ref(false)
const selectedRecord = ref(null)
const timeline = ref([])
const detailDrawer = ref(false)
const formDialog = ref(false)
const actionDialog = ref(false)
const financeDialog = ref(false)
const actionSubmitting = ref(false)
const exchangeSyncing = ref(false)
const editingId = ref(null)
const actionType = ref('status')
const filters = reactive({ keyword: '', company_name: '', contact_name: '', status: '', owner_id: undefined, priority: '', min_amount: undefined, max_amount: undefined })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
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
const actionForm = reactive({ status: 'confirmed', reason: '', paid_amount: 0, payment_no: '', shipment_no: '', carrier: '', shipment_scope: 'full', delivery_note: '', invoice_no: '', invoice_type: 'commercial_invoice', invoice_exemption_reason: '', payment_plan: [] })
const financeForm = reactive(defaultFinanceSetting())
const statusOptions = [
  { value: 'draft', label: t('domain.auto.0f436818c0'), desc: t('domain.auto.c764fb6205') },
  { value: 'pending_review', label: t('domain.auto.f53b68002d'), desc: t('domain.auto.359990a82f') },
  { value: 'confirmed', label: t('domain.auto.1b9b73eb06'), desc: t('domain.auto.6694ef4754') },
  { value: 'partial_shipped', label: t('domain.auto.9727328c5a'), desc: t('domain.auto.405e017a40') },
  { value: 'shipped', label: t('domain.auto.4340959624'), desc: t('domain.auto.dcef7cf2cd') },
  { value: 'completed', label: t('domain.auto.e99b48a29b'), desc: t('domain.auto.8d133edbaa') },
  { value: 'canceled', label: t('domain.auto.a5ffdc95ee'), desc: t('domain.auto.62b6ab3931') }
]
const shipmentScopeOptions = computed(() => [{ label: t('orders.fullShipment'), value: 'full' }, { label: t('orders.partialShipment'), value: 'partial' }])
const paymentTermOptions = computed(() => [
  { label: t('orders.paymentTermAdvance'), value: 'advance_payment' },
  { label: t('orders.paymentTermCod'), value: 'cash_on_delivery' },
  { label: t('orders.paymentTermBeforeShipment'), value: 'pay_before_shipment' },
  { label: t('orders.paymentTermNet7'), value: 'Net 7' },
  { label: t('orders.paymentTermNet15'), value: 'Net 15' },
  { label: t('orders.paymentTermNet30'), value: 'Net 30' },
  { label: t('orders.paymentTermNet60'), value: 'Net 60' },
  { label: t('orders.paymentTermInstallment'), value: 'installment' }
])
const invoiceTypeOptions = computed(() => [
  { label: t('orders.invoiceCommercial'), value: 'commercial_invoice' },
  { label: t('orders.invoiceSpecialVat'), value: 'special_vat' },
  { label: t('orders.invoiceVatGeneral'), value: 'vat_general' },
  { label: t('orders.invoiceElectronic'), value: 'electronic_invoice' },
  { label: t('orders.invoiceProforma'), value: 'proforma_invoice' },
  { label: t('orders.invoiceReceipt'), value: 'receipt' }
])
const orderPriorityOptions = computed(() => [
  { label: t('orders.priorityUrgent'), value: 'urgent' },
  { label: t('orders.priorityHigh'), value: 'high' },
  { label: t('orders.priorityNormal'), value: 'normal' },
  { label: t('orders.priorityLow'), value: 'low' }
])

const selectedPayload = computed(() => payload(selectedRecord.value))
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))
const canManage = computed(() => canManageDomain('orders'))
const canCancelOrder = computed(() => hasExactPermission('orders:manage'))
const actionTitle = computed(() => ({ status: t('domain.auto.139d02a181'), pay: t('domain.auto.6453d96518'), payment_plan: t('domain.auto.8b198ba444'), ship: t('domain.auto.622bf9ab9e'), invoice: t('domain.auto.c858222388'), ticket: t('domain.auto.6aee271b65'), cancel: t('domain.auto.3eedd32d80') }[actionType.value] || t('domain.auto.e127aad189')))
const actionDialogWidth = computed(() => actionType.value === 'payment_plan' ? 'min(860px, 94vw)' : 'min(680px, 94vw)')
const completionWarnings = computed(() => selectedRecord.value ? missingCompletionRequirements(selectedRecord.value) : [])
const reasonRequired = computed(() => !['payment_plan', 'ticket'].includes(actionType.value))
const availableStatusOptions = computed(() => statusTransitionOptions(selectedRecord.value))

onMounted(async () => {
  await Promise.all([loadUsers(), loadProducts(), loadCustomerOptions(), loadFinanceSetting(), loadPage()])
  await maybeOpenMfgQuotePrefill()
})

async function loadUsers() {
  try {
    const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } })
    users.value = res.data?.items || []
  } catch {
    users.value = []
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

async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/orders/objects', { params: cleanParams({ object_type: 'order', ...filters, page: pagination.page, size: pagination.size }) })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
    if (selectedRecord.value) selectedRecord.value = rows.value.find((item) => item.id === selectedRecord.value.id) || selectedRecord.value
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.ee1007d862')))
  } finally {
    loading.value = false
  }
}

function searchPage() {
  pagination.page = 1
  loadPage()
}

function openCreate() {
  editingId.value = null
  Object.assign(form, defaultForm())
  formDialog.value = true
  if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions()
}

async function maybeOpenMfgQuotePrefill() {
  const quoteID = Number(route.query.mfg_quote_id || 0)
  if (!quoteID) return
  try {
    const res = await api.get(`/api/v1/mfg/quotes/${quoteID}/order-prefill`)
    const data = res.data?.data || res.data || {}
    const nextForm = data.form || data
    editingId.value = null
    Object.assign(form, defaultForm(), nextForm)
    form.payload = { ...defaultForm().payload, ...(nextForm.payload || {}) }
    if (!Array.isArray(form.payload.line_items)) form.payload.line_items = []
    formDialog.value = true
    ElMessage.success(t('orders.mfgQuotePrefilled'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('orders.mfgQuotePrefillFailed')))
  } finally {
    const query = { ...route.query }
    delete query.mfg_quote_id
    router.replace({ path: route.path, query }).catch(() => {})
  }
}

function openEdit(row) {
  editingId.value = row.id
  Object.assign(form, {
    object_type: 'order',
    name: row.name,
    code: row.code || '',
    status: normalizedOrderStatus(row.status || 'draft'),
    owner_id: row.owner_id || 0,
    company_id: row.company_id || 0,
    contact_id: row.contact_id || 0,
    amount: row.amount || 0,
    currency: normalizeCurrencyCode(row.currency, defaultBusinessCurrency()),
    score: row.score || 0,
    tags: splitTags(row.tags),
    payload: { line_items: [], ...payload(row) }
  })
  syncCustomerChainFromRecord(row)
  if (!Array.isArray(form.payload.line_items)) form.payload.line_items = []
  formDialog.value = true
  if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions()
}

async function submitForm() {
  if (!form.name) return ElMessage.error(t('domain.auto.78d997e536'))
  try {
    normalizeCustomerChainForSubmit()
    const isCreate = !editingId.value
    if (editingId.value) await api.put(`/api/v1/prd/orders/objects/${editingId.value}`, form)
    else await api.post('/api/v1/prd/orders/objects', form)
    if (isCreate) playCelebration('milestone')
    ElMessage.success(t('domain.auto.376588a074'))
    formDialog.value = false
    await loadPage()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('orders.operationFailed')))
  }
}

async function openDetail(row) {
  selectedRecord.value = row
  detailDrawer.value = true
  await Promise.all([loadTimeline(row), loadOrderFiles(row)])
}

function selectRecord(row) {
  if (!row?.id) return
  selectedRecord.value = row
}

function orderRowClass({ row }) {
  return selectedRecord.value && Number(selectedRecord.value.id) === Number(row.id) ? 'is-selected-row' : ''
}

async function loadTimeline(row) {
  const rel = await api.get('/api/v1/prd/relations', { params: { object_type: 'orders', object_id: row.id, relation_type: 'timeline' } })
  const ids = (rel.data?.items || []).map((item) => item.target_type === 'contacts' ? item.target_id : item.source_id)
  const items = await Promise.all(ids.map((id) => api.get(`/api/v1/prd/contacts/objects/${id}`).catch(() => null)))
  timeline.value = items.map((res) => res?.data).filter(Boolean)
}

async function loadOrderFiles(row = selectedRecord.value) {
  if (!row?.id) return
  filesLoading.value = true
  try {
    const res = await api.get(`/api/v1/prd/flows/orders/${row.id}/files`)
    orderFiles.value = res.data?.items || []
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.8aa4a5fc14')))
  } finally {
    filesLoading.value = false
  }
}

function openAction(type, row = selectedRecord.value) {
  if (!row) return
  if (type === 'cancel' && !canCancelOrder.value) return ElMessage.error(t('orders.cancelPermissionDenied'))
  const statusTargets = type === 'status' ? statusTransitionOptions(row) : []
  if (type === 'status' && statusTargets.length === 0) return ElMessage.warning(t('orders.noStatusTransitionAvailable'))
  selectedRecord.value = row
  actionType.value = type
  const rowPayload = payload(row)
  Object.assign(actionForm, {
    status: type === 'cancel' ? 'canceled' : (statusTargets[0]?.value || nextStatus(row.status)),
    reason: '',
    paid_amount: receivable(row) || row.amount || 0,
    payment_no: `PAY-${Date.now()}`,
    shipment_no: `SHP-${Date.now()}`,
    carrier: rowPayload.carrier || '',
    shipment_scope: 'full',
    delivery_note: '',
    invoice_no: rowPayload.invoice_no || `INV-${Date.now()}`,
    invoice_type: rowPayload.invoice_type || 'commercial_invoice',
    invoice_exemption_reason: rowPayload.invoice_exemption_reason || '',
    payment_plan: editablePaymentPlan(row)
  })
  actionDialog.value = true
}

async function submitAction() {
  if (actionSubmitting.value) return
  const row = selectedRecord.value
  const type = actionType.value
  if (!row?.id) return ElMessage.error(t('orders.selectOrderFirst'))
  if (reasonRequired.value && !actionForm.reason) return ElMessage.error(t('domain.auto.bd59f8c970'))
  const paymentPlanPayload = type === 'payment_plan' ? normalizePlanForSubmit(row) : []
  if (type === 'payment_plan' && paymentPlanPayload.length === 0) return ElMessage.error(t('orders.keepOneValidPaymentPlan'))
  const invalidPlanIndex = type === 'payment_plan' ? paymentPlanPayload.findIndex((item) => !item.due_date || Number(item.amount || 0) <= 0) : -1
  if (invalidPlanIndex >= 0) return ElMessage.error(t('orders.paymentPlanLineInvalid', { index: invalidPlanIndex + 1 }))
  if (type === 'status') {
    const transitionTargets = statusTransitionOptions(row).map((item) => item.value)
    if (!transitionTargets.includes(actionForm.status)) return ElMessage.error(t('orders.statusTransitionNotAllowed', { from: statusLabel(row.status), to: statusLabel(actionForm.status) }))
    const missing = missingStatusRequirements(row, actionForm.status)
    if (missing.length) return ElMessage.error(t('orders.statusRequirementsMissing', { items: missing.join('，') }))
  }
  if (type === 'status' && actionForm.status === 'completed') {
    const missing = missingCompletionRequirements(row)
    if (missing.length) return ElMessage.error(`${t('domain.auto.aba5633d39')}${missing.join('，')}`)
  }
  if (type === 'ship' && !canRegisterShipment(row)) {
    return ElMessage.error(t('orders.shipmentNotAllowed'))
  }
  if (type === 'invoice' && !String(actionForm.invoice_no || '').trim() && !String(actionForm.invoice_exemption_reason || '').trim()) {
    return ElMessage.error(t('domain.auto.f4120987d7'))
  }
  actionSubmitting.value = true
  try {
    let result
    if (type === 'pay') result = await api.post(`/api/v1/prd/flows/orders/${row.id}/pay`, { payment_no: actionForm.payment_no, payment_method: 'manual', paid_amount: actionForm.paid_amount, reason: actionForm.reason })
    else if (type === 'payment_plan') result = await api.post(`/api/v1/prd/flows/orders/${row.id}/payment-plan`, { payment_plan: paymentPlanPayload, reason: actionForm.reason })
    else if (type === 'ship') result = await api.post(`/api/v1/prd/flows/orders/${row.id}/ship`, { shipment_no: actionForm.shipment_no, carrier: actionForm.carrier, shipment_scope: actionForm.shipment_scope, delivery_note: actionForm.delivery_note })
    else if (type === 'invoice') result = await api.post(`/api/v1/prd/flows/orders/${row.id}/invoice`, { invoice_no: actionForm.invoice_no, invoice_type: actionForm.invoice_type, invoice_exemption_reason: actionForm.invoice_exemption_reason, sent_to: payload(row).email || '' })
    else if (type === 'ticket') { result = await api.post(`/api/v1/prd/flows/orders/${row.id}/ticket`); ElMessage.success(t('domain.auto.e0c15b03d1')) }
    else result = await api.post(`/api/v1/prd/flows/orders/${row.id}/status`, { status: actionForm.status, reason: actionForm.reason })
    const updatedOrder = result?.data?.order || result?.order
    playOrderCelebration(type, actionForm.status)
    actionDialog.value = false
    await loadPage()
    selectedRecord.value = type === 'payment_plan'
      ? await loadOrderDetail(row.id)
      : (updatedOrder || rows.value.find((item) => Number(item.id) === Number(row.id)) || row)
    if (type === 'payment_plan' && !paymentPlan(selectedRecord.value).length) throw new Error(t('orders.paymentPlanReadbackFailed'))
    if (type === 'payment_plan') ElMessage.success(t('orders.paymentPlanSaved'))
    await Promise.all([loadTimeline(selectedRecord.value), loadOrderFiles(selectedRecord.value)])
  } catch (error) {
    ElMessage.error(orderActionErrorMessage(error))
  } finally {
    actionSubmitting.value = false
  }
}

async function loadOrderDetail(id) {
  const res = await api.get(`/api/v1/prd/orders/objects/${id}`)
  return res?.data || res
}

function playOrderCelebration(type, status) {
  if (type === 'pay') return playCelebration('revenue')
  if (type === 'ship') return playCelebration('fulfillment')
  if (type === 'invoice') return playCelebration('light')
  if (type === 'status' && status === 'completed') return playCelebration('win')
}

async function uploadOrderFile(options) {
  if (!selectedRecord.value?.id) return
  fileUploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', options.file)
    const uploadRes = await api.post('/api/v1/files/upload', formData)
    const info = unwrapFileUploadResponse(uploadRes)
    await api.post(`/api/v1/prd/flows/orders/${selectedRecord.value.id}/files`, {
      file_id: info.file_id,
      file_name: info.original_name || info.file_name || options.file.name,
      file_size: Number(info.file_size || options.file.size || 0),
      file_type: options.file.type || info.file_type || '',
      storage_node_id: info.storage_node_id || '',
      download_url: info.download_url || `/api/v1/files/${info.file_id}/download`,
      category: 'contract',
      visibility: 'internal'
    })
    ElMessage.success(t('domain.auto.0e854aeff9'))
    await loadOrderFiles()
    options.onSuccess?.(info)
  } catch (error) {
    options.onError?.(error)
    ElMessage.error(apiErrorMessage(error, t('domain.auto.b777dd3e91')))
  } finally {
    fileUploading.value = false
  }
}

async function downloadOrderFile(file) {
  const p = payload(file)
  const fileID = p.file_id
  try {
    await downloadProtectedFile(p.download_url || `/api/v1/files/${fileID}/download`, p.file_name || file.name || t('domain.auto.0859f214e3'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.7d58c3b853')))
  }
}

async function deleteOrderFile(file) {
  try {
    await confirmDangerAction(`${t('domain.auto.1c349b225b')}${file.name}${t('domain.auto.30baaa0f0c')}`, t('domain.auto.3c06abe116'))
    await api.delete(`/api/v1/prd/flows/orders/${selectedRecord.value.id}/files/${file.id}`, { skipDeleteConfirm: true })
    ElMessage.success(t('domain.auto.718272fdfc'))
    await loadOrderFiles()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.e8e675b522')))
  }
}

async function downloadOrderBackup() {
  if (!selectedRecord.value?.id) return
  try {
    await downloadProtectedFile(`/api/v1/prd/flows/orders/${selectedRecord.value.id}/backup`, `order-${selectedRecord.value.id}-excel-backup.zip`)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.7d58c3b853')))
  }
}

function canRegisterShipment(row) {
  return shipmentRequirements(row).every((item) => item.passed)
}

function shipmentRequirements(row) {
  const p = payload(row)
  return [
    { label: t('orders.shipmentRequirementStatus', { status: statusLabel(row?.status) }), passed: ['confirmed', 'partial_shipped'].includes(row?.status) },
    { label: t('orders.shipmentRequirementLineItems'), passed: lineItems(row).length > 0 },
    { label: t('orders.shipmentRequirementPaymentTerms'), passed: Boolean(String(p.payment_terms || '').trim()) },
    { label: t('orders.shipmentRequirementDeliveryInfo'), passed: Boolean(String(p.shipping_address || p.delivery_address || p.delivery_note || '').trim()) },
    { label: t('orders.shipmentRequirementAmount'), passed: Number(row?.amount || 0) > 0 }
  ]
}

function missingCompletionRequirements(row) {
  const p = payload(row)
  const missing = []
  if (!['shipped', 'partial_shipped'].includes(row?.status)) missing.push(t('domain.auto.f70fb2bfe4'))
  if (receivable(row) > 0) missing.push(t('domain.auto.2d855ef84e'))
  if (!String(p.invoice_no || '').trim() && !String(p.invoice_exemption_reason || '').trim()) missing.push(t('domain.auto.2533f81c3a'))
  return missing
}

function orderReadyRequirements(row) {
  const p = payload(row)
  return [
    { label: t('orders.shipmentRequirementLineItems'), passed: lineItems(row).length > 0 },
    { label: t('orders.shipmentRequirementPaymentTerms'), passed: Boolean(String(p.payment_terms || '').trim()) },
    { label: t('orders.shipmentRequirementDeliveryInfo'), passed: Boolean(String(p.shipping_address || p.delivery_address || p.delivery_note || '').trim()) },
    { label: t('orders.shipmentRequirementAmount'), passed: Number(row?.amount || 0) > 0 }
  ]
}

function missingOrderReadyRequirements(row) {
  return orderReadyRequirements(row).filter((item) => !item.passed).map((item) => item.label)
}

function missingStatusRequirements(row, next) {
  const status = normalizedOrderStatus(next)
  if (['confirmed', 'partial_shipped', 'shipped'].includes(status)) return missingOrderReadyRequirements(row)
  if (status === 'completed') return missingCompletionRequirements(row)
  return []
}

function statusTransitionOptions(row) {
  const targets = allowedStatusTransitions(row?.status)
  return statusOptions.filter((item) => targets.includes(item.value))
}

function allowedStatusTransitions(status) {
  return ({
    draft: ['pending_review'],
    pending_review: ['confirmed', 'draft'],
    confirmed: ['partial_shipped', 'shipped'],
    partial_shipped: ['shipped', 'completed'],
    shipped: ['completed']
  }[normalizedOrderStatus(status)] || [])
}

function quickStatus(status) { filters.status = filters.status === status ? '' : status; searchPage() }
function resetFilters() { Object.assign(filters, { keyword: '', company_name: '', contact_name: '', status: '', owner_id: undefined, priority: '', min_amount: undefined, max_amount: undefined }); pagination.page = 1; loadPage() }
function addLine() { form.payload.line_items.push({ product_id: undefined, sku: '', name: '', quantity: 1, unit_price: 0, tax_rate: 0, stock_available: undefined }) }
function removeLine(index) { form.payload.line_items.splice(index, 1) }
function addPlanLine() { actionForm.payment_plan.push({ name: `${t('domain.auto.dae828fe4f')}${actionForm.payment_plan.length + 1}${t('domain.auto.fc73601f20')}`, amount: 0, currency: normalizeCurrencyCode(selectedRecord.value?.currency, defaultBusinessCurrency()), due_date: '' }) }
function removePlanLine(index) { actionForm.payment_plan.splice(index, 1) }
function editablePaymentPlan(row) {
  const currency = normalizeCurrencyCode(row?.currency)
  const existingPlan = paymentPlan(row)
    .filter((item) => Number(item?.amount || 0) > 0 || cleanPaymentPlanText(item?.due_date))
    .map((item, index) => ({ ...item, name: paymentPlanName(item, index), amount: Number(item.amount || 0), due_date: cleanPaymentPlanText(item.due_date), currency: normalizeCurrencyCode(item.currency || currency) }))
  return existingPlan.length ? existingPlan : [{ name: t('domain.auto.23839ee5d5'), amount: row?.amount || 0, currency, due_date: '' }]
}
function normalizePlanForSubmit(row) {
  const currency = normalizeCurrencyCode(row?.currency || selectedRecord.value?.currency)
  return actionForm.payment_plan
    .map((item, index) => ({ ...item, name: paymentPlanName(item, index), amount: Number(item.amount || 0), due_date: cleanPaymentPlanText(item.due_date), currency: normalizeCurrencyCode(item.currency || currency) }))
    .filter((item) => item.due_date || item.amount > 0)
}
async function refreshSelected(id) {
  await loadPage()
  const next = rows.value.find((item) => Number(item.id) === Number(id))
  if (next) {
    selectedRecord.value = next
    await Promise.all([loadTimeline(next), loadOrderFiles(next)])
  }
}

function defaultForm() {
  return { object_type: 'order', name: '', code: '', status: 'draft', owner_id: 0, company_id: 0, contact_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { company_id: undefined, company: '', company_name: '', contact_id: undefined, contact: '', contact_name: '', payment_status: 'pending', payment_terms: 'Net 30', shipping_address: '', priority: 'normal', visibility: 'public', line_items: [] } }
}

function defaultFinanceSetting() {
  return { base_currency: defaultBusinessCurrency(), exchange_rates: { ...defaultExchangeRates }, tax_codes: [{ code: 'VAT13', name: t('domain.auto.e9f58b8125'), rate: 13 }, { code: 'VAT6', name: t('domain.auto.f4c55f16cc'), rate: 6 }, { code: 'ZERO', name: t('domain.auto.a3eef9ef31'), rate: 0 }] }
}

async function loadFinanceSetting() {
  try {
    const res = await api.get('/api/v1/prd/settings', { params: { scope: 'order_finance', key: 'default' } })
    const setting = res.data?.items?.[0]
    if (!setting) return
    const value = typeof setting.value_json === 'string' ? JSON.parse(setting.value_json || '{}') : (setting.value_json || {})
    Object.assign(financeForm, { ...defaultFinanceSetting(), ...value, exchange_rates: { ...defaultExchangeRates, ...(value.exchange_rates || {}) }, tax_codes: Array.isArray(value.tax_codes) && value.tax_codes.length ? value.tax_codes : defaultFinanceSetting().tax_codes })
  } catch {
    Object.assign(financeForm, defaultFinanceSetting())
  }
}

function openFinanceDialog() { financeDialog.value = true }
async function syncExchangeRates() {
  exchangeSyncing.value = true
  try {
    const base = normalizeCurrencyCode(financeForm.base_currency)
    const res = await api.post('/api/v1/prd/settings/exchange-rates/sync', { base_currency: base, currencies: currencyCodes })
    const nextRates = res.data?.exchange_rates || {}
    if (!Object.keys(nextRates).length) throw new Error(t('orders.noExchangeRateData'))
    financeForm.exchange_rates = { ...financeForm.exchange_rates, ...nextRates }
    ElMessage.success(t('orders.exchangeRateSynced'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('orders.exchangeRateSyncFailed')))
  } finally {
    exchangeSyncing.value = false
  }
}
async function saveFinanceSetting() {
  await api.post('/api/v1/prd/settings', { scope: 'order_finance', key: 'default', value: financeForm })
  ElMessage.success(t('domain.auto.a644822460'))
  financeDialog.value = false
}

function addTaxCode() { financeForm.tax_codes.push({ code: '', name: '', rate: 0 }) }
function removeTaxCode(index) { financeForm.tax_codes.splice(index, 1) }
function applyTaxCode(item) {
  const tax = financeForm.tax_codes.find((entry) => entry.code === item.tax_code)
  if (tax) item.tax_rate = Number(tax.rate || 0)
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
function payload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function orderCompany(row) { const p = payload(row); return p.company || p.company_name || '-' }
function orderContact(row) { const p = payload(row); return p.contact_name || p.contact || p.name_contact || '-' }
function objectVisibility(row) { const p = payload(row); return p.visibility === 'private' || p.access_visibility === 'private' || p.private === true || p.private === 'true' ? 'private' : 'public' }
function fileMeta(file) {
  const p = payload(file)
  const size = Number(p.file_size || 0)
  const sizeText = size > 0 ? `${(size / 1024).toFixed(size > 1024 * 1024 ? 0 : 1)} KB` : '-'
  return `${p.category || 'other'} · ${sizeText} · ${p.version || 'v1'}`
}
function lineItems(row) { return Array.isArray(payload(row).line_items) ? payload(row).line_items : [] }
function paymentPlan(row) {
  const items = Array.isArray(payload(row).payment_plan) ? payload(row).payment_plan : []
  return items.filter((item) => Number(item?.amount || 0) > 0 || cleanPaymentPlanText(item?.due_date))
}
function cleanPaymentPlanText(value) {
  const text = String(value ?? '').trim()
  return ['', '<nil>', 'null', 'undefined'].includes(text.toLowerCase()) ? '' : text
}
function paymentPlanName(item, index) {
  return cleanPaymentPlanText(item?.name) || `${t('domain.auto.dae828fe4f')}${index + 1}${t('domain.auto.fc73601f20')}`
}
function shipments(row) {
  const p = payload(row)
  if (Array.isArray(p.shipments)) return p.shipments
  if (p.shipment_no || p.carrier || p.shipped_at) return [{ shipment_no: p.shipment_no, carrier: p.carrier, shipment_scope: p.shipment_scope || 'full', delivery_note: p.delivery_note, shipped_at: p.shipped_at }]
  return []
}
function latestShipment(row) {
  const items = shipments(row)
  return items.length ? items[items.length - 1] : {}
}
function shipmentScopeLabel(value) { return value === 'partial' ? t('orders.partialShipment') : t('orders.fullShipment') }
function invoiceTypeLabel(value) { return invoiceTypeOptions.value.find((item) => item.value === value)?.label || value || '-' }
function receivable(row) { const p = payload(row); return Number(p.receivable_balance ?? (Number(row?.amount || 0) - Number(p.paid_amount || 0))) }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function normalizedOrderStatus(value) { return value === 'active' ? 'draft' : value }
function statusLabel(value) { return statusOptions.find((item) => item.value === normalizedOrderStatus(value))?.label || value || '-' }
function priorityLabel(value) { return orderPriorityOptions.value.find((item) => item.value === value)?.label || t('orders.priorityNormal') }
function priorityType(value) { return value === 'urgent' || value === 'high' ? 'danger' : value === 'low' ? 'info' : 'warning' }
function nextStatus(status) { return ({ draft: 'pending_review', pending_review: 'confirmed', confirmed: 'shipped', partial_shipped: 'shipped', shipped: 'completed' }[normalizedOrderStatus(status)] || 'confirmed') }
function orderActionErrorMessage(error) {
  return apiErrorMessage(error, t('orders.operationFailed'))
}
function isEffectiveOrderStatus(status) { return ['confirmed', 'partial_shipped', 'shipped', 'completed'].includes(normalizedOrderStatus(status)) }
function timelineTitle(item) {
  const p = payload(item)
  if (p.event_type === 'order_status' && (p.from || p.to)) return t('orders.statusChanged', { from: statusLabel(p.from), to: statusLabel(p.to) })
  return translateTimelineText(p.summary || item.name || '-')
}
function timelineOperator(item) { return timelineOperatorText(item, users.value, formatTime) }
function timelineSummary(item) {
  const p = payload(item)
  const parts = []
  if (p.reason) parts.push(translateTimelineText(p.reason))
  if (p.content) parts.push(translateTimelineText(p.content))
  if (p.payment_no) parts.push(t('orders.timelinePaymentNo', { value: p.payment_no }))
  if (p.shipment_no) parts.push(t('orders.timelineShipmentNo', { value: p.shipment_no }))
  if (p.carrier) parts.push(t('orders.timelineCarrier', { value: p.carrier }))
  if (p.shipment_scope) parts.push(t('orders.timelineShipmentType', { value: shipmentScopeLabel(p.shipment_scope) }))
  if (p.invoice_no) parts.push(t('orders.timelineInvoiceNo', { value: p.invoice_no }))
  if (p.invoice_type) parts.push(t('orders.timelineInvoiceType', { value: invoiceTypeLabel(p.invoice_type) }))
  if (p.invoice_exemption_reason) parts.push(t('orders.timelineInvoiceExemptionReason', { value: p.invoice_exemption_reason }))
  return parts.length ? parts.join(' · ') : '-'
}
function translateTimelineText(value) {
  const text = String(value || '').trim()
  if (!text) return ''
  return text.replace(/\b(draft|pending_review|confirmed|partial_shipped|shipped|completed|canceled)\b/g, (status) => statusLabel(status))
}
function statusCount(status) { return rows.value.filter((item) => normalizedOrderStatus(item.status) === status).length }
function statusAmount(status) { return rows.value.filter((item) => normalizedOrderStatus(item.status) === status && isEffectiveOrderStatus(item.status)).reduce((sum, item) => sum + Number(item.amount || 0), 0) }
function ownerName(ownerID) { const id = Number(ownerID || 0); if (!id) return t('domain.auto.3bb92e6efc'); const user = users.value.find((item) => Number(item.id) === id); return user ? userDisplayName(user) : `${t('domain.auto.9ba763ea34')}${id}` }
function userDisplayName(user) { const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`; return user.department_name ? `${label} · ${user.department_name}` : label }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
</script>

<style scoped>
.timeline-operator {
  display: block;
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  line-height: 1.4;
}

.order-dialog-form,
.order-finance-form {
  align-items: start;
}

.order-dialog-form {
  column-gap: 40px;
}

.order-search-form {
  grid-template-columns: repeat(4, minmax(220px, 1fr)) auto;
  align-items: start;
}

.amount-range-field {
  min-width: 320px;
}

.amount-range-inputs {
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  gap: 8px;
  align-items: center;
}

.amount-range-inputs :deep(.el-input-number) {
  width: 100%;
}

.finance-sync-row {
  width: 100%;
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.finance-sync-row span {
  color: #667085;
  font-size: 12px;
}

.order-line-list {
  min-width: 0;
}

.order-line-card,
.payment-plan-card,
.tax-code-card,
.exchange-rate-card {
  width: 100%;
  min-width: 0;
  display: grid;
  gap: 8px;
  align-items: center;
}

.order-line-card {
  grid-template-columns: minmax(200px, 1.25fr) minmax(180px, 1fr) minmax(112px, 0.55fr) minmax(132px, 0.65fr) minmax(150px, 0.75fr) minmax(112px, 0.55fr);
  padding: 12px;
  border: 1px solid #e6eaf0;
  border-radius: 8px;
  background: #fbfcfe;
}

.line-field {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.line-field > span {
  color: #667085;
  font-size: 12px;
  line-height: 1.2;
}

.line-field :deep(.el-select),
.line-field :deep(.el-input),
.line-field :deep(.el-input-number) {
  width: 100%;
}

.line-field :deep(.el-input__wrapper) {
  min-height: 34px;
}

.line-card-footer {
  grid-column: 1 / -1;
  min-height: 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.payment-plan-card {
  grid-template-columns: minmax(150px, 1fr) minmax(220px, 1fr) minmax(160px, 0.75fr) auto;
  padding: 8px 10px;
  border: 1px solid #e6eaf0;
  border-radius: 8px;
  background: #fbfcfe;
}

.payment-plan-amount {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 96px;
  gap: 6px;
  align-items: center;
}

.payment-plan-card :deep(.el-input-number),
.payment-plan-card :deep(.el-select),
.payment-plan-date {
  width: 100%;
}

.payment-plan-currency :deep(.el-input__wrapper),
.payment-plan-date :deep(.el-input__wrapper) {
  min-height: 32px;
}

.payment-plan-date {
  min-width: 0;
}

.tax-code-card {
  grid-template-columns: minmax(120px, 0.45fr) minmax(180px, 1fr) minmax(110px, 0.4fr) auto;
}

.currency-rate-grid {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.exchange-rate-card {
  grid-template-columns: 54px minmax(118px, 0.8fr) minmax(140px, 1fr);
  padding: 8px 10px;
  border: 1px solid #e6eaf0;
  border-radius: 8px;
  background: #fbfcfe;
}

.exchange-rate-card span,
.product-stock-note {
  color: #667085;
  font-size: 12px;
}

.order-file-toolbar,
.order-file-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.order-file-toolbar {
  margin-bottom: 10px;
}

.order-file-item {
  align-items: center;
}

.order-gate-alert {
  min-height: 36px;
  padding: 8px 10px;
  display: flex;
  gap: 8px;
  align-items: center;
  color: #92400e;
  border: 1px solid #fde68a;
  border-radius: 8px;
  background: #fffbeb;
}

.order-gate-checklist {
  padding: 10px 12px;
  display: grid;
  gap: 8px;
  border: 1px solid #e6eaf0;
  border-radius: 8px;
  background: #fbfcfe;
}

.order-gate-checklist strong {
  color: #172033;
  font-size: 13px;
}

.order-gate-checklist > div {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.order-gate-checklist span {
  min-height: 28px;
  padding: 4px 8px;
  display: inline-flex;
  gap: 5px;
  align-items: center;
  color: #92400e;
  font-size: 12px;
  border: 1px solid #fde68a;
  border-radius: 999px;
  background: #fffbeb;
}

.order-gate-checklist span.passed {
  color: #047857;
  border-color: #bbf7d0;
  background: #f0fdf4;
}

.order-action-alert {
  margin-bottom: 12px;
}

@media (max-width: 1180px) {
  .order-search-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .order-line-card {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .currency-rate-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 900px) {
  .payment-plan-card {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .order-search-form {
    grid-template-columns: minmax(0, 1fr);
  }

  .amount-range-field {
    min-width: 0;
  }

  .order-line-card,
  .payment-plan-card,
  .tax-code-card,
  .exchange-rate-card {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
