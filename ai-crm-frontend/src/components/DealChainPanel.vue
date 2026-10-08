<template>
  <section class="deal-chain-panel">
    <div class="deal-chain-head">
      <strong>{{ title }}</strong>
      <span v-if="summaryText">{{ summaryText }}</span>
    </div>
    <div v-if="nodes.length && !showGuidedEmpty" class="deal-chain-steps">
      <button
        v-for="node in nodes"
        :key="node.key || `${node.domain}-${node.id}`"
        class="deal-chain-node"
        :class="{ active: isCurrentNode(node), muted: isVirtualNode(node) }"
        type="button"
        @click="goNode(node)"
      >
        <span>{{ stepLabel(node.step) }}</span>
        <strong>{{ node.name || '-' }}</strong>
        <em>{{ node.status || '-' }} · {{ formatAmount(node.amount, node.currency) }}</em>
        <small>{{ nodeOperatorText(node) }}</small>
      </button>
    </div>
    <div v-if="showGuidedEmpty" class="deal-chain-empty">
      <strong>{{ emptyTitle }}</strong>
      <span>{{ chainError || emptyDescription }}</span>
      <div v-if="showOpportunityActions" class="deal-chain-empty-actions">
        <el-button size="small" type="primary" plain @click="$emit('createQuote')">{{ t('domain.auto.7e37e9b4fb') }}</el-button>
        <el-button size="small" type="success" plain @click="$emit('winOrder')">{{ t('opportunity.action.winToDeal') }}</el-button>
      </div>
    </div>
    <el-empty v-else-if="!nodes.length" :description="emptyTitle" />
  </section>
</template>

<script setup>
import { i18nState, t } from '../i18n'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { timelineOperatorText } from '../utils/timeline'
import { formatCurrencyAmount, normalizeCurrencyCode } from '../utils/currency'
import { defaultBusinessCurrency } from '../utils/company-format'
import { apiErrorMessage } from '../utils/api-error'

const props = defineProps({
  objectType: { type: String, default: '' },
  objectId: { type: [Number, String], default: 0 },
  autoLoad: { type: Boolean, default: true },
  title: { type: String, default: t('domain.auto.91238956f1') },
  currentStep: { type: String, default: '' },
  users: { type: Array, default: () => [] },
  emptyTitle: { type: String, default: t('domain.auto.cfb4d42789') },
  emptyDescription: { type: String, default: t('domain.auto.2c375ff8a3') },
  showOpportunityActions: { type: Boolean, default: false }
})
defineEmits(['createQuote', 'winOrder'])

const router = useRouter()
const loading = ref(false)
const nodes = ref([])
const summary = ref({})
const chainError = ref('')
const currentKey = computed(() => `${normalizeDomain(props.objectType)}:${Number(props.objectId || 0)}`)
const summaryText = computed(() => {
  const data = summary.value || {}
  const receivable = Number(data.open_receivable || 0)
  const orders = Number(data.open_orders || 0)
  if (!receivable && !orders) return ''
  return `${t('domain.auto.404c64e4c1')}${orders}${t('domain.auto.0a3e0fabd4')}${formatAmount(receivable)}`
})
const showGuidedEmpty = computed(() => {
  if (!nodes.value.length) return true
  if (!props.currentStep) return false
  return nodes.value.length === 1 && stepKey(nodes.value[0]) === props.currentStep
})

onMounted(loadChain)
watch(() => [props.objectType, props.objectId], loadChain)

async function loadChain() {
  if (!props.autoLoad || !props.objectType || !Number(props.objectId || 0)) {
    nodes.value = []
    summary.value = {}
    return
  }
  loading.value = true
  chainError.value = ''
  try {
    const res = await api.get('/api/v1/prd/flows/deal-chain', {
      params: { object_type: normalizeDomain(props.objectType), object_id: props.objectId }
    })
    nodes.value = res.data?.nodes || []
    summary.value = res.data?.summary || {}
  } catch (error) {
    nodes.value = []
    summary.value = {}
    chainError.value = apiErrorMessage(error, t('deal.chain.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

function normalizeDomain(value) {
  if (['lead', 'leads'].includes(value)) return 'leads'
  if (['opportunity', 'quote', 'opportunities'].includes(value)) return 'opportunities'
  if (['deal', 'closed_deal', 'deals'].includes(value)) return 'deals'
  if (['order', 'invoice', 'orders'].includes(value)) return 'orders'
  if (['ticket', 'tickets'].includes(value)) return 'tickets'
  return value || ''
}
function isCurrentNode(node) { return `${node.domain}:${Number(node.id || 0)}` === currentKey.value }
function isVirtualNode(node) { return ['payments', 'shipments', 'invoices'].includes(node.domain) }
function stepKey(node) { return node?.step || normalizeDomain(node?.domain || '').replace(/s$/, '') }
function stepLabel(step) {
  return {
    lead: t('domain.auto.8de0b8fb1d'),
    opportunity: t('domain.auto.c4beee8569'),
    quote: t('domain.auto.78208f311a'),
    deal: t('domain.auto.4483df6e28'),
    order: t('domain.auto.81b3798beb'),
    payment: t('domain.auto.50b07f3dfa'),
    shipment: t('domain.auto.b06463cd7b'),
    invoice: t('domain.auto.cb819e1641'),
    ticket: t('domain.auto.7f7a2fa0ea')
  }[step] || step || t('domain.auto.e840cd6f1e')
}
function nodeOperatorText(node) {
  return timelineOperatorText(node, props.users, formatTime)
}
function goNode(node) {
  if (isVirtualNode(node)) return
  const path = { leads: '/leads', opportunities: '/opportunities', deals: '/deals', orders: '/orders', tickets: '/tickets' }[node.domain]
  if (path) router.push(path)
}
function formatAmount(amount, currency = defaultBusinessCurrency()) {
  return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency()))
}
function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString(i18nState.locale, { hour12: false })
}
</script>
