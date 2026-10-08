import { computed, ref } from 'vue'
import api from '../api'

export function payload(row) {
  try {
    return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {})
  } catch {
    return {}
  }
}

export function normalizeRecordID(value) {
  const id = Number(value || 0)
  return Number.isFinite(id) && id > 0 ? id : 0
}

export function useCustomerChainOptions(form) {
  const companies = ref([])
  const contacts = ref([])
  const companyOptions = computed(() => companies.value.filter((item) => item.status !== 'inactive' && item.status !== 'merged'))
  const contactOptions = computed(() => {
    const companyID = normalizeRecordID(form.company_id || form.payload?.company_id)
    const active = contacts.value.filter((item) => item.status !== 'inactive' && item.status !== 'merged')
    if (!companyID) return active
    const matched = active.filter((item) => normalizeRecordID(item.company_id || payload(item).company_id) === companyID)
    return matched.length ? matched : active
  })

  async function loadCustomerOptions() {
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

  function companyOptionLabel(item) {
    const p = payload(item)
    return `${item.name}${p.industry ? ` · ${p.industry}` : ''}`
  }

  function contactOptionLabel(item) {
    const p = payload(item)
    const company = p.company || p.company_name ? ` · ${p.company || p.company_name}` : ''
    const contact = p.phone || p.email ? ` · ${p.phone || p.email}` : ''
    return `${item.name}${company}${contact}`
  }

  function applyCompany(companyID) {
    const id = normalizeRecordID(companyID)
    const company = companyOptions.value.find((item) => normalizeRecordID(item.id) === id)
    if (!company) {
      form.company_id = 0
      if (form.payload) {
        delete form.payload.company_id
        form.payload.company = ''
        form.payload.company_name = ''
      }
      return
    }
    form.company_id = company.id
    form.payload.company_id = company.id
    form.payload.company = company.name
    form.payload.company_name = company.name
    const selectedContact = contacts.value.find((item) => normalizeRecordID(item.id) === normalizeRecordID(form.contact_id || form.payload.contact_id))
    if (selectedContact) {
      const contactCompanyID = normalizeRecordID(selectedContact.company_id || payload(selectedContact).company_id)
      if (contactCompanyID && contactCompanyID !== normalizeRecordID(company.id)) {
        form.contact_id = 0
        delete form.payload.contact_id
        form.payload.contact = ''
        form.payload.contact_name = ''
      }
    }
  }

  function applyContact(contactID) {
    const id = normalizeRecordID(contactID)
    const contact = contactOptions.value.find((item) => normalizeRecordID(item.id) === id) || contacts.value.find((item) => normalizeRecordID(item.id) === id)
    if (!contact) {
      form.contact_id = 0
      if (form.payload) {
        delete form.payload.contact_id
        form.payload.contact = ''
        form.payload.contact_name = ''
      }
      return
    }
    const p = payload(contact)
    form.contact_id = contact.id
    form.payload.contact_id = contact.id
    form.payload.contact = contact.name
    form.payload.contact_name = contact.name
    if (!form.payload.phone && p.phone) form.payload.phone = p.phone
    if (!form.payload.email && p.email) form.payload.email = p.email
    const companyID = normalizeRecordID(contact.company_id || p.company_id)
    if (companyID) applyCompany(companyID)
  }

  function syncCustomerChainFromRecord(row) {
    const p = payload(row)
    form.company_id = normalizeRecordID(row?.company_id || p.company_id)
    form.contact_id = normalizeRecordID(row?.contact_id || p.contact_id)
    if (form.payload) {
      form.payload.company_id = form.company_id || undefined
      form.payload.contact_id = form.contact_id || undefined
      form.payload.company = p.company || p.company_name || form.payload.company || ''
      form.payload.company_name = p.company_name || p.company || form.payload.company_name || ''
      form.payload.contact = p.contact || p.contact_name || form.payload.contact || ''
      form.payload.contact_name = p.contact_name || p.contact || form.payload.contact_name || ''
    }
  }

  function normalizeCustomerChainForSubmit() {
    const companyID = normalizeRecordID(form.company_id || form.payload?.company_id)
    const contactID = normalizeRecordID(form.contact_id || form.payload?.contact_id)
    form.company_id = companyID
    form.contact_id = contactID
    if (companyID) applyCompany(companyID)
    if (contactID) applyContact(contactID)
  }

  return {
    companies,
    contacts,
    companyOptions,
    contactOptions,
    loadCustomerOptions,
    companyOptionLabel,
    contactOptionLabel,
    applyCompany,
    applyContact,
    syncCustomerChainFromRecord,
    normalizeCustomerChainForSubmit
  }
}
