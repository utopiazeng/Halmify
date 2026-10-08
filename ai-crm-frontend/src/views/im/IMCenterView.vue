<template>
  <div class="im-page" :class="{ 'platform-support-page': isPlatformAdmin }">
    <aside class="im-shell im-system">
      <div class="im-panel-head strong-head">
        <div>
          <strong>{{ isPlatformAdmin ? t('platform.messages.demoRequests') : t('domain.auto.f581d83fe2') }}</strong>
          <small>{{ systemUnread }} {{ t('domain.auto.c20364e35e') }}</small>
        </div>
        <el-button size="small" text @click="loadSystemNotifications">{{ t('domain.auto.38108eaa1d') }}</el-button>
      </div>
      <div class="system-toolbar">
        <el-segmented v-model="systemStatusFilter" size="small" :options="systemStatusOptions" @change="loadSystemNotifications" />
        <div class="system-toolbar-actions">
          <el-button v-if="isTenantAdmin" size="small" type="primary" plain @click="openBroadcastDialog">{{ t('domain.auto.6dd8258409') }}</el-button>
          <el-button size="small" text @click="markAllSystemRead">{{ t('domain.auto.ac210d6e7d') }}</el-button>
          <el-dropdown trigger="click">
            <el-button size="small" text>
              {{ t('domain.auto.907c3945d8') }}<Icon icon="ri:arrow-down-s-line" />
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="archiveReadSystemNotifications">{{ t('domain.auto.c479c83e93') }}</el-dropdown-item>
                <el-dropdown-item @click="cleanupOldSystemNotifications">{{ t('domain.auto.5649426d71') }}</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <div class="im-list im-system-list">
        <button
          v-for="item in systemNotifications"
          :key="item.id"
          class="system-message-row"
          :class="{ unread: isSystemUnread(item), archived: item.status === 'archived' }"
          @click="readSystemNotification(item)"
        >
          <span class="system-message-icon"><Icon icon="ri:mail-line" /></span>
          <span class="system-message-main">
            <strong>{{ systemTitle(item) }}</strong>
            <small>{{ systemSender(item) }} · {{ formatTs(item.created_at, true) }}</small>
            <em>{{ systemMessageContent(item) }}</em>
          </span>
          <el-tag v-if="isSystemUnread(item)" size="small" type="danger">{{ t('domain.auto.1e230aa201') }}</el-tag>
          <el-tag v-else-if="item.status === 'archived'" size="small" type="info">{{ t('domain.auto.5cfbea2b76') }}</el-tag>
        </button>
        <div v-if="!systemNotifications.length && !loading.system" class="im-empty-mini">
          {{ t('domain.auto.cbdfd3ba78') }}
        </div>
      </div>
    </aside>

    <aside v-if="!isPlatformAdmin" class="im-shell im-contacts">
      <div class="im-panel-head strong-head">
        <div>
          <strong>{{ t('im.contacts') }}</strong>
          <small>{{ onlineCount }} {{ t('im.online') }}</small>
        </div>
      </div>
      <div v-if="isTenantAdmin" class="support-entry">
        <button class="support-entry-btn" @click="openSupportDialog">
          <span><Icon icon="ri:customer-service-2-line" /></span>
          <strong>{{ t('domain.auto.ce3720fba2') }}</strong>
          <small>{{ t('domain.auto.9b740885e8') }}</small>
        </button>
      </div>
      <div class="im-filters">
        <el-input
          v-model="contactQuery"
          :placeholder="t('im.searchContacts')"
          clearable
          @input="debouncedLoadContacts"
          @clear="loadContacts"
        >
          <template #prefix>
            <Icon icon="ri:search-line" />
          </template>
        </el-input>
      </div>
      <div class="im-list im-contact-list">
        <button
          v-for="item in contacts"
          :key="item.user_id"
          class="im-person-row"
          :class="{ active: activePeerId === item.user_id }"
          @click="openByPeer(item.user_id)"
        >
          <span class="avatar-wrap">
            <img v-if="item.avatar_display_url" :src="item.avatar_display_url" :alt="displayName(item)" />
            <span v-else>{{ initials(item) }}</span>
            <i class="presence-dot" :class="`presence-${item.status || 'offline'}`"></i>
          </span>
          <span class="person-main">
            <strong>{{ displayName(item) }}</strong>
            <small>{{ subtitle(item) }}</small>
          </span>
        </button>
        <div v-if="!contacts.length && !loading.contacts" class="im-empty-mini">
          {{ t('im.noContacts') }}
        </div>
      </div>
    </aside>

    <aside class="im-shell im-conversations">
      <div class="im-panel-head strong-head">
        <div>
          <strong>{{ isPlatformAdmin ? t('platform.messages.supportConversations') : t('im.conversations') }}</strong>
          <small>{{ conversations.length }} {{ isPlatformAdmin ? t('platform.messages.pendingConversations') : t('im.conversationCount') }}</small>
        </div>
        <el-badge v-if="totalUnread" :value="totalUnread" />
      </div>
      <div v-if="isPlatformAdmin" class="support-filters">
        <el-select v-model="supportFilters.category" size="small" clearable :placeholder="t('domain.auto.fa6968d9f0')">
          <el-option v-for="item in supportCategories" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="supportFilters.priority" size="small" clearable :placeholder="t('domain.auto.00a6823667')">
          <el-option v-for="item in supportPriorities" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-input v-model="supportFilters.tenant_id" size="small" clearable :placeholder="t('domain.auto.05676bf71f')" />
        <el-checkbox v-model="supportFilters.unread_only">{{ t('domain.auto.e5038d192e') }}</el-checkbox>
      </div>
      <div class="im-list im-conv-list">
        <button
          v-for="item in conversations"
          :key="item.conversation_id"
          class="im-conv-row"
          :class="{ active: activeConversationId === item.conversation_id }"
          @click="openConversation(item)"
        >
          <span class="avatar-wrap">
            <img v-if="item.peer_avatar_display_url" :src="item.peer_avatar_display_url" :alt="peerName(item)" />
            <span v-else>{{ peerInitials(item) }}</span>
            <i class="presence-dot" :class="`presence-${item.peer_status || 'offline'}`"></i>
          </span>
          <span class="conv-main">
            <span class="conv-top">
              <strong>{{ item.conversation_type === 'support' ? (item.tenant_name || t('domain.auto.c7b2356e93')) : peerName(item) }}</strong>
              <small>{{ formatTs(item.last_message_at, true) }}</small>
            </span>
            <span class="conv-bottom">
              <small>{{ item.conversation_type === 'support' ? supportConversationPreview(item) : previewText(item) }}</small>
              <el-badge v-if="item.unread_count" :value="item.unread_count" />
              <Icon v-else-if="item.pinned" class="pin-icon" icon="ri:pushpin-2-fill" />
            </span>
          </span>
        </button>
        <div v-if="!conversations.length && !loading.conversations" class="im-empty-mini">
          {{ t('im.noConversations') }}
        </div>
      </div>
    </aside>

    <main class="im-shell im-chat">
      <template v-if="activeConversationId">
        <div class="im-chat-head">
          <div class="chat-peer">
            <span class="avatar-wrap big">
              <img v-if="activePeer?.avatar" :src="activePeer.avatar" :alt="activePeer.name" />
              <span v-else>{{ activePeerInitial }}</span>
              <i class="presence-dot" :class="`presence-${activePeer?.status || 'offline'}`"></i>
            </span>
            <div>
              <strong>{{ activeTitle }}</strong>
              <small>{{ activePeerMeta }}</small>
            </div>
          </div>
          <div class="retention-chip">
            <Icon icon="ri:time-line" />
            <span>{{ t('im.retentionNotice') }}</span>
          </div>
        </div>

        <div ref="msgListRef" class="im-msg-list" @scroll.passive="handleMessageScroll">
          <el-button
            v-if="hasMoreMessages"
            class="load-more"
            text
            :loading="loading.messages"
            @click="loadMoreMessages"
          >
            {{ t('im.loadMore') }}
          </el-button>
          <div
            v-for="(msg, index) in messages"
            :key="messageKey(msg)"
            class="msg-block"
            :class="{ self: msg.sender_id === currentUserId }"
          >
            <div v-if="shouldShowTime(index)" class="msg-time-sep">{{ formatTs(msg.created_at) }}</div>
            <div class="msg-row">
              <span v-if="msg.sender_id !== currentUserId" class="avatar-wrap small">
                <img v-if="activePeer?.avatar" :src="activePeer.avatar" :alt="activePeer.name" />
                <span v-else>{{ activePeerInitial }}</span>
              </span>
              <div class="bubble" :class="{ emoji: msg.type === 'emoji' }">{{ messageContent(msg) }}</div>
              <span v-if="msg.sender_id === currentUserId" class="avatar-wrap small self-avatar">
                <img v-if="currentUserAvatar" :src="currentUserAvatar" :alt="currentUserName" />
                <span v-else>{{ currentUserInitial }}</span>
              </span>
            </div>
          </div>
        </div>

        <div class="im-input">
          <div class="composer-tools">
            <el-popover trigger="click" placement="top-start" :width="320">
              <template #reference>
                <el-button text class="tool-btn">
                  <Icon icon="ri:emotion-line" />
                </el-button>
              </template>
              <div class="emoji-grid">
                <button
                  v-for="emoji in emojiList"
                  :key="emoji"
                  class="emoji-btn"
                  @click="sendEmoji(emoji)"
                >
                  {{ emoji }}
                </button>
              </div>
            </el-popover>
            <span class="composer-hint">{{ t('im.inputHint') }}</span>
          </div>
          <el-input
            ref="inputRef"
            v-model="draft"
            type="textarea"
            resize="none"
            :autosize="{ minRows: 1, maxRows: 6 }"
            :maxlength="5000"
            show-word-limit
            :placeholder="t('im.inputPlaceholder')"
            @keydown.enter.exact.prevent="sendText"
            @keydown.enter.shift.exact.stop
          />
          <div class="composer-footer">
            <span :class="['ws-state', wsConnected ? 'on' : 'off']">
              {{ wsConnected ? t('im.connected') : t('im.reconnecting') }}
            </span>
            <el-button type="primary" :disabled="!draft.trim()" :loading="sending" @click="sendText">
              {{ t('im.send') }}
            </el-button>
          </div>
        </div>
      </template>
      <div v-else class="im-empty">
        <Icon icon="ri:chat-smile-3-line" />
        <strong>{{ isPlatformAdmin ? t('platform.messages.selectConversation') : t('im.selectConversation') }}</strong>
        <p>{{ isPlatformAdmin ? t('platform.messages.selectConversationTip') : t('im.selectConversationTip') }}</p>
      </div>
    </main>
  </div>

  <el-dialog v-model="systemDetailVisible" :title="t('domain.auto.994d426014')" width="620px" destroy-on-close>
    <div v-if="activeSystemNotification" class="system-detail">
      <h3>{{ systemTitle(activeSystemNotification) }}</h3>
      <div class="detail-meta">
        <span>{{ t('domain.auto.31debc738f') }}{{ activeSystemSender }}</span>
        <span>{{ t('domain.auto.32d77333df') }}{{ formatTs(activeSystemNotification.created_at) }}</span>
        <span>{{ t('domain.auto.bf0ac89783') }}{{ systemStatusLabel(activeSystemNotification.status) }}</span>
      </div>
      <p>{{ activeSystemContent }}</p>
      <button v-if="hasSystemRelatedObject" class="related-object-box" type="button" @click="goSystemRelatedObject">
        <Icon icon="ri:links-line" />
        <span>{{ t('domain.auto.577d82e2de') }}{{ activeSystemRelatedLabel }}</span>
      </button>
    </div>
    <template #footer>
      <el-button @click="systemDetailVisible = false">{{ t('domain.auto.6c14bd7f6f') }}</el-button>
      <el-button v-if="isSystemUnread(activeSystemNotification)" @click="markActiveSystemRead">{{ t('domain.auto.504ecf732d') }}</el-button>
      <el-button v-if="activeSystemNotification?.status !== 'archived'" type="warning" plain @click="archiveSystemNotification(activeSystemNotification)">{{ t('domain.auto.ddfde75bec') }}</el-button>
      <el-button v-if="hasSystemRelatedObject" type="primary" @click="goSystemRelatedObject">{{ t('domain.auto.7a1726220d') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="broadcastDialogVisible" :title="t('domain.auto.d05e150d50')" width="560px" destroy-on-close>
    <el-alert
      class="broadcast-tip"
      type="info"
      show-icon
      :closable="false"
      :title="t('domain.auto.3093c01ced')"
    />
    <el-form label-position="top" class="support-form">
      <el-form-item :label="t('domain.auto.1cf9ae8f64')" required>
        <el-select v-model="broadcastForm.category" :placeholder="t('domain.auto.a66c882c58')">
          <el-option v-for="item in broadcastCategories" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('domain.auto.936ade7379')" required>
        <el-segmented v-model="broadcastForm.priority" :options="broadcastPriorityOptions" />
      </el-form-item>
      <el-form-item :label="t('domain.auto.748d7dc7e3')" required>
        <el-input v-model="broadcastForm.title" maxlength="120" show-word-limit :placeholder="t('domain.auto.fc47274cec')" />
      </el-form-item>
      <el-form-item :label="t('domain.auto.163aec9194')" required>
        <el-input v-model="broadcastForm.content" type="textarea" :rows="6" maxlength="2000" show-word-limit :placeholder="t('domain.auto.48d8b840e8')" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="broadcastDialogVisible = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
      <el-button type="primary" :loading="broadcastSubmitting" @click="submitBroadcastNotification">{{ t('domain.auto.221f71a458') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="supportDialogVisible" :title="t('domain.auto.ce3720fba2')" width="560px" destroy-on-close>
    <el-form label-position="top" class="support-form">
      <el-form-item :label="t('domain.auto.b7c588c460')" required>
        <el-select v-model="supportForm.category" :placeholder="t('domain.auto.5453135b06')">
          <el-option v-for="item in supportCategories" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('domain.auto.62a7c63929')" required>
        <el-segmented v-model="supportForm.priority" :options="supportPriorityOptions" />
      </el-form-item>
      <el-form-item :label="t('domain.auto.748d7dc7e3')" required>
        <el-input v-model="supportForm.title" maxlength="120" show-word-limit :placeholder="t('domain.auto.39f750e2ae')" />
      </el-form-item>
      <el-form-item :label="t('domain.auto.163aec9194')" required>
        <el-input v-model="supportForm.content" type="textarea" :rows="6" maxlength="2000" show-word-limit :placeholder="t('domain.auto.d250aeb54e')" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="supportDialogVisible = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
      <el-button type="primary" :loading="supportSubmitting" @click="submitSupportConversation">{{ t('domain.auto.4c0d5232e6') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'
import { currentUser, isPlatformAdminUser, isTenantAdminUser } from '../../permissions'
import { resolveProtectedImageUrl, revokeObjectUrl } from '../../utils/protectedImage'

const contacts = ref([])
const conversations = ref([])
const messages = ref([])
const systemNotifications = ref([])
const systemUnread = ref(0)
const systemStatusFilter = ref('active')
const contactQuery = ref('')
const activeConversationId = ref(0)
const activePeerId = ref(0)
const draft = ref('')
const reconnectTimer = ref(null)
const contactTimer = ref(null)
const wsConnected = ref(false)
const sending = ref(false)
const hasMoreMessages = ref(true)
const msgListRef = ref(null)
const inputRef = ref(null)
const loading = reactive({ contacts: false, conversations: false, messages: false, system: false })
const route = useRoute()
const router = useRouter()
const currentUserInfo = ref(currentUser())
const currentUserAvatarDisplay = ref('')
const currentUserId = computed(() => Number(currentUserInfo.value.id || 0))
const currentUserName = computed(() => currentUserInfo.value.name || currentUserInfo.value.username || 'Me')
const currentUserAvatar = computed(() => currentUserAvatarDisplay.value || currentUserInfo.value.avatar_url || '')
const currentUserInitial = computed(() => String(currentUserName.value || 'M').slice(0, 1).toUpperCase())
const totalUnread = ref(0)
const isPlatformAdmin = computed(() => isPlatformAdminUser())
const isTenantAdmin = computed(() => isTenantAdminUser() && !isPlatformAdmin.value)
const supportDialogVisible = ref(false)
const systemDetailVisible = ref(false)
const broadcastDialogVisible = ref(false)
const activeSystemNotification = ref(null)
const supportFilters = reactive({ category: '', priority: '', tenant_id: '', unread_only: false })
const supportForm = reactive({ category: 'feedback', title: '', content: '', priority: 'normal' })
const supportSubmitting = ref(false)
const broadcastForm = reactive({ category: 'announcement', title: '', content: '', priority: 'normal' })
const broadcastSubmitting = ref(false)
const relatedObjectCache = ref({})
const supportCategories = [
  { value: 'feedback', label: t('domain.auto.3ea03fba3e') },
  { value: 'feature_suggestion', label: t('domain.auto.a310658fef') },
  { value: 'bug', label: t('domain.auto.9659fd91a1') },
  { value: 'billing_ai', label: t('domain.auto.f93ed893a0') },
  { value: 'account_permission', label: t('domain.auto.11b83f7d0f') },
  { value: 'data_permission', label: t('domain.auto.d7a0156b2b') },
  { value: 'other', label: t('domain.auto.1a26edf94a') }
]
const supportPriorities = [
  { value: 'normal', label: t('domain.auto.7cda072d45') },
  { value: 'urgent', label: t('domain.auto.38bccd3d4d') },
  { value: 'critical', label: t('domain.auto.27429f465b') }
]
const supportPriorityOptions = supportPriorities.map((item) => ({ label: item.label, value: item.value }))
const broadcastCategories = [
  { value: 'announcement', label: t('domain.auto.c767b502cc') },
  { value: 'maintenance', label: t('domain.auto.8f7aed7395') },
  { value: 'policy', label: t('domain.auto.079f154090') },
  { value: 'training', label: t('domain.auto.26d3ae584b') },
  { value: 'business', label: t('domain.auto.208f221d23') },
  { value: 'other', label: t('domain.auto.1a26edf94a') }
]
const broadcastPriorityOptions = [
  { label: t('domain.auto.7cda072d45'), value: 'normal' },
  { label: t('domain.auto.b7f4670752'), value: 'important' },
  { label: t('domain.auto.27429f465b'), value: 'urgent' }
]
const systemStatusOptions = [
  { label: t('domain.auto.778fc8f994'), value: 'active' },
  { label: t('domain.auto.1e230aa201'), value: 'unread' },
  { label: t('domain.auto.642ec8b596'), value: 'read' },
  { label: t('domain.auto.5cfbea2b76'), value: 'archived' }
]
const emojiList = ['😀', '😄', '😁', '😂', '😊', '😉', '😍', '🤝', '👍', '👏', '🎉', '🙏', '💪', '❤️', '🔥', '🚀', '✅', '📌', '📣', '💡', '☕', '🌟', '😎', '🙌']

const activeConversation = computed(() => conversations.value.find((it) => Number(it.conversation_id) === Number(activeConversationId.value)))
const activeContact = computed(() => contacts.value.find((it) => Number(it.user_id) === Number(activePeerId.value)))
const activePeer = computed(() => {
  const conv = activeConversation.value
  const contact = activeContact.value
  if (conv) {
    return {
      name: peerName(conv),
      avatar: conv.peer_avatar_display_url || contact?.avatar_display_url || '',
      status: conv.peer_status || contact?.status || 'offline',
      department: conv.peer_department || contact?.department || '',
      post: conv.peer_post_name || contact?.post_name || ''
    }
  }
  if (contact) {
    return {
      name: displayName(contact),
      avatar: contact.avatar_display_url || '',
      status: contact.status || 'offline',
      department: contact.department || '',
      post: contact.post_name || ''
    }
  }
  return null
})
const activeTitle = computed(() => {
  const conv = activeConversation.value
  if (conv?.conversation_type === 'support') return supportConversationTitle(conv)
  return activePeer.value?.name || t('im.noConversation')
})
const activePeerInitial = computed(() => String(activePeer.value?.name || 'U').slice(0, 1).toUpperCase())
const activePeerMeta = computed(() => {
  const conv = activeConversation.value
  if (conv?.conversation_type === 'support') {
    return [conv.tenant_name, supportCategoryLabel(conv.category, conv.category_label), supportPriorityLabel(conv.priority, conv.priority_label), peerName(conv)].filter(Boolean).join(' · ')
  }
  const peer = activePeer.value || {}
  const parts = [presenceLabel(peer.status), peer.department, peer.post].filter(Boolean)
  return parts.join(' · ') || t('im.departmentEmpty')
})
const onlineCount = computed(() => contacts.value.filter((item) => item.status === 'online').length)

onMounted(async () => {
  await refreshCurrentUserAvatar()
  if (isPlatformAdmin.value) {
    await Promise.all([loadConversations(), loadUnreadSummary(), loadSystemNotifications()])
  } else {
    await Promise.all([loadContacts(), loadConversations(), loadUnreadSummary(), loadSystemNotifications()])
  }
  await openConversationFromRoute()
  window.addEventListener('im-ws-event', handleSharedWSEvent)
  window.addEventListener('im-ws-state', handleSharedWSState)
  window.addEventListener('current-user-updated', handleCurrentUserUpdated)
  wsConnected.value = Boolean(window.__AI_CRM_IM_WS_CONNECTED)
})

watch(() => route.query.tab, (value) => {
  if (value === 'system') loadSystemNotifications()
})

watch(() => route.query.conversation_id, () => {
  openConversationFromRoute()
})

watch(() => ({ ...supportFilters }), () => {
  if (isPlatformAdmin.value) loadConversations()
}, { deep: true })

onUnmounted(() => {
  if (reconnectTimer.value) clearTimeout(reconnectTimer.value)
  if (contactTimer.value) clearTimeout(contactTimer.value)
  window.removeEventListener('im-ws-event', handleSharedWSEvent)
  window.removeEventListener('im-ws-state', handleSharedWSState)
  window.removeEventListener('current-user-updated', handleCurrentUserUpdated)
  cleanupAvatarObjectUrls(contacts.value, 'avatar_display_url')
  cleanupAvatarObjectUrls(conversations.value, 'peer_avatar_display_url')
  revokeObjectUrl(currentUserAvatarDisplay.value)
})

async function loadContacts() {
  if (isPlatformAdmin.value) {
    contacts.value = []
    return
  }
  loading.contacts = true
  try {
    const res = await api.get('/api/v1/im/contacts', {
      params: { q: contactQuery.value, page: 1, size: 200 }
    })
    cleanupAvatarObjectUrls(contacts.value, 'avatar_display_url')
    contacts.value = await withAvatarDisplayUrls(res.data?.items || [], 'avatar_url', 'avatar_display_url')
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
  } finally {
    loading.contacts = false
  }
}

function debouncedLoadContacts() {
  if (contactTimer.value) clearTimeout(contactTimer.value)
  contactTimer.value = setTimeout(loadContacts, 240)
}

async function loadConversations() {
  loading.conversations = true
  try {
    const endpoint = isPlatformAdmin.value ? '/api/v1/im/support/conversations' : '/api/v1/im/conversations'
    const params = isPlatformAdmin.value ? {
      category: supportFilters.category || undefined,
      priority: supportFilters.priority || undefined,
      unread_only: supportFilters.unread_only ? '1' : undefined,
      tenant_id: supportFilters.tenant_id || undefined
    } : undefined
    const res = await api.get(endpoint, { params })
    cleanupAvatarObjectUrls(conversations.value, 'peer_avatar_display_url')
    conversations.value = await withAvatarDisplayUrls(res.data?.items || [], 'peer_avatar_url', 'peer_avatar_display_url')
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
  } finally {
    loading.conversations = false
  }
}

async function openConversationFromRoute() {
  const convID = Number(route.query.conversation_id || 0)
  if (!convID) return
  if (!conversations.value.length) {
    await loadConversations()
  }
  const conv = conversations.value.find((item) => Number(item.conversation_id) === convID)
  if (conv) {
    await openConversation(conv)
    return
  }
  activeConversationId.value = convID
  activePeerId.value = 0
  await loadMessages()
  focusInput()
}

async function loadUnreadSummary() {
  try {
    const res = await api.get('/api/v1/im/unread/summary')
    totalUnread.value = Number(res.data?.total_unread || 0)
  } catch {
    totalUnread.value = 0
  }
}

async function handleCurrentUserUpdated() {
  currentUserInfo.value = currentUser()
  await refreshCurrentUserAvatar()
}

async function refreshCurrentUserAvatar() {
  const source = String(currentUserInfo.value?.avatar_url || '').trim()
  const previous = currentUserAvatarDisplay.value
  if (!source) {
    currentUserAvatarDisplay.value = ''
    revokeObjectUrl(previous)
    return
  }
  try {
    const next = await resolveProtectedImageUrl(source)
    currentUserAvatarDisplay.value = next
    if (previous && previous !== next) revokeObjectUrl(previous)
  } catch {
    currentUserAvatarDisplay.value = ''
    revokeObjectUrl(previous)
  }
}

async function withAvatarDisplayUrls(items, sourceKey, targetKey) {
  return Promise.all((items || []).map(async (item) => {
    const source = String(item?.[sourceKey] || '').trim()
    if (!source) return { ...item, [targetKey]: '' }
    try {
      return { ...item, [targetKey]: await resolveProtectedImageUrl(source) }
    } catch {
      return { ...item, [targetKey]: '' }
    }
  }))
}

function cleanupAvatarObjectUrls(items, key) {
  ;(items || []).forEach((item) => revokeObjectUrl(item?.[key]))
}

async function loadSystemNotifications() {
  loading.system = true
  try {
    const res = await api.get('/api/v1/im/system-notifications', { params: { status: systemStatusFilter.value, page: 1, size: 50 } })
    systemNotifications.value = res.data?.items || []
    systemUnread.value = Number(res.data?.unread || 0)
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
  } finally {
    loading.system = false
  }
}

async function readSystemNotification(item) {
  if (!item?.id) return
  let nextItem = item
  if (isSystemUnread(item)) {
    try {
      const res = await api.post(`/api/v1/im/system-notifications/${item.id}/read`)
      nextItem = res.data?.notification || { ...item, status: 'read' }
      await loadSystemNotifications()
      ElMessage.success(t('domain.auto.b1437640bc'))
    } catch (error) {
      ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
      return
    }
  }
  activeSystemNotification.value = nextItem
  await preloadSystemRelatedObject(nextItem)
  systemDetailVisible.value = true
}

async function markActiveSystemRead() {
  if (!activeSystemNotification.value?.id) return
  await readSystemNotification(activeSystemNotification.value)
}

async function markAllSystemRead() {
  try {
    await ElMessageBox.confirm(t('domain.auto.268a010b09'), t('domain.auto.ac210d6e7d'), { type: 'warning' })
    const res = await api.post('/api/v1/im/system-notifications/read-all')
    systemUnread.value = Number(res.data?.unread || 0)
    await loadSystemNotifications()
    ElMessage.success(`${t('domain.auto.d1e3d7cbb2')}${Number(res.data?.updated || 0)}${t('domain.auto.f444f71fc3')}`)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || t('domain.auto.c4be092b0b'))
  }
}

async function archiveSystemNotification(item) {
  if (!item?.id) return
  try {
    await ElMessageBox.confirm(t('domain.auto.9a2c7f5815'), t('domain.auto.4517f65d18'), { type: 'warning' })
    const res = await api.post('/api/v1/im/system-notifications/archive', { ids: [item.id] })
    systemUnread.value = Number(res.data?.unread || systemUnread.value || 0)
    systemDetailVisible.value = false
    activeSystemNotification.value = null
    await loadSystemNotifications()
    ElMessage.success(t('domain.auto.291310b46f'))
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || t('domain.auto.29a3cfdc73'))
  }
}

async function archiveReadSystemNotifications() {
  try {
    await ElMessageBox.confirm(t('domain.auto.bd8a978924'), t('domain.auto.9eaba1feeb'), { type: 'warning' })
    const res = await api.post('/api/v1/im/system-notifications/archive', { status: 'read' })
    systemUnread.value = Number(res.data?.unread || 0)
    await loadSystemNotifications()
    ElMessage.success(`${t('domain.auto.5cfbea2b76')}${Number(res.data?.archived || 0)}${t('domain.auto.6635731451')}`)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || t('domain.auto.6f2390db7a'))
  }
}

async function cleanupOldSystemNotifications() {
  try {
    await ElMessageBox.confirm(t('domain.auto.ecf3034b95'), t('domain.auto.05b5226bcd'), { type: 'warning' })
    const res = await api.post('/api/v1/im/system-notifications/cleanup', { before_days: 30 })
    systemUnread.value = Number(res.data?.unread || 0)
    await loadSystemNotifications()
    ElMessage.success(`${t('domain.auto.5cfbea2b76')}${Number(res.data?.archived || 0)}${t('domain.auto.3b88e0eb99')}`)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || t('domain.auto.6f2390db7a'))
  }
}

function openBroadcastDialog() {
  broadcastForm.category = 'announcement'
  broadcastForm.priority = 'normal'
  broadcastForm.title = ''
  broadcastForm.content = ''
  broadcastDialogVisible.value = true
}

async function submitBroadcastNotification() {
  if (!broadcastForm.category || !broadcastForm.title.trim() || !broadcastForm.content.trim()) {
    ElMessage.warning(t('domain.auto.a991959376'))
    return
  }
  try {
    await ElMessageBox.confirm(t('domain.auto.eacbeab139'), t('domain.auto.15f9d038e0'), { type: 'warning' })
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    throw error
  }
  broadcastSubmitting.value = true
  try {
    const res = await api.post('/api/v1/im/system-notifications/broadcast', { ...broadcastForm })
    broadcastDialogVisible.value = false
    await loadSystemNotifications()
    ElMessage.success(`${t('domain.auto.e85f367e31')}${Number(res.data?.recipient_count || 0)}${t('domain.auto.8422e30c5e')}`)
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('domain.auto.4336fb0756'))
  } finally {
    broadcastSubmitting.value = false
  }
}

async function openByPeer(peerId) {
  try {
    const res = await api.post('/api/v1/im/conversations', { peer_id: peerId })
    const convID = Number(res.data?.conversation_id || 0)
    if (!convID) return
    activePeerId.value = Number(peerId)
    await loadConversations()
    const conv = conversations.value.find((item) => Number(item.conversation_id) === convID)
    activeConversationId.value = convID
    if (conv) activePeerId.value = Number(conv.peer_id)
    await loadMessages()
    focusInput()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
  }
}

function openSupportDialog() {
  supportForm.category = 'feedback'
  supportForm.title = ''
  supportForm.content = ''
  supportForm.priority = 'normal'
  supportDialogVisible.value = true
}

async function submitSupportConversation() {
  if (!supportForm.category || !supportForm.title.trim() || !supportForm.content.trim()) {
    ElMessage.warning(t('domain.auto.34e0ad0645'))
    return
  }
  supportSubmitting.value = true
  try {
    const res = await api.post('/api/v1/im/support/conversation', { ...supportForm })
    const convID = Number(res.data?.conversation_id || 0)
    supportDialogVisible.value = false
    ElMessage.success(t('domain.auto.e0a7ba608e'))
    await loadConversations()
    const conv = conversations.value.find((item) => Number(item.conversation_id) === convID)
    if (conv) {
      await openConversation(conv)
    } else if (convID) {
      activeConversationId.value = convID
      await loadMessages()
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('domain.auto.e767d34c78'))
  } finally {
    supportSubmitting.value = false
  }
}

async function openConversation(item) {
  activeConversationId.value = Number(item.conversation_id)
  activePeerId.value = Number(item.peer_id)
  await loadMessages()
  focusInput()
}

async function loadMessages(beforeID = 0, appendTop = false) {
  if (!activeConversationId.value || loading.messages) return
  loading.messages = true
  const oldScrollHeight = msgListRef.value?.scrollHeight || 0
  try {
    const res = await api.get(`/api/v1/im/conversations/${activeConversationId.value}/messages`, {
      params: { before_id: beforeID, limit: 20 }
    })
    const items = res.data?.items || []
    hasMoreMessages.value = items.length >= 20
    if (appendTop) {
      messages.value = mergeMessages([...items, ...messages.value])
      await nextTick(() => restoreScrollAfterPrepend(oldScrollHeight))
    } else {
      messages.value = items
      await nextTick(scrollToBottom)
    }
    await markActiveRead()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
  } finally {
    loading.messages = false
  }
}

function loadMoreMessages() {
  const first = messages.value[0]
  if (!first) return
  loadMessages(first.id, true)
}

async function markActiveRead() {
  if (!activeConversationId.value) return
  try {
    await api.post(`/api/v1/im/conversations/${activeConversationId.value}/read`)
    conversations.value = conversations.value.map((item) => Number(item.conversation_id) === Number(activeConversationId.value) ? { ...item, unread_count: 0 } : item)
    await loadUnreadSummary()
  } catch {
    // read state should not block chat usage
  }
}

function handleMessageScroll() {
  const el = msgListRef.value
  if (!el || el.scrollTop > 4 || !hasMoreMessages.value || loading.messages) return
  loadMoreMessages()
}

function scrollToBottom() {
  const el = msgListRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

function restoreScrollAfterPrepend(oldScrollHeight) {
  const el = msgListRef.value
  if (!el) return
  el.scrollTop = Math.max(el.scrollHeight - oldScrollHeight, 0)
}

async function sendText() {
  const content = String(draft.value || '').trim()
  if (!content || !activeConversationId.value || sending.value) return
  await sendMessage('text', content)
  draft.value = ''
}

async function sendEmoji(emoji) {
  if (!activeConversationId.value || sending.value) return
  await sendMessage('emoji', emoji)
}

async function sendMessage(type, content) {
  sending.value = true
  try {
    const res = await api.post(`/api/v1/im/conversations/${activeConversationId.value}/messages`, {
      type,
      content,
      client_msg_id: `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
    })
    if (res.data?.id) appendMessage(res.data)
    await loadConversations()
    await nextTick(scrollToBottom)
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || t('im.loadFailed'))
  } finally {
    sending.value = false
  }
}

async function handleSharedWSEvent(evt) {
  try {
    const payload = evt.detail || {}
    const event = payload?.event
    const data = normalizeEventData(payload?.data)
    if (event === 'im.message.new') {
      await handleIncomingMessage(data)
    } else if (event === 'im.message.ack') {
      await loadConversations()
    } else if (event === 'im.conversation.unread') {
      totalUnread.value = Number(data.total_unread || totalUnread.value || 0)
      await loadConversations()
    } else if (event === 'im.presence.changed') {
      updatePresence(data)
    } else if (event === 'im.system.unread') {
      systemUnread.value = Number(data.unread || systemUnread.value || 0)
    }
  } catch {
    // ignore malformed websocket payload
  }
}

function handleSharedWSState(evt) {
  wsConnected.value = Boolean(evt.detail?.connected)
}

async function handleIncomingMessage(msg) {
  if (!msg?.conversation_id) {
    await loadConversations()
    return
  }
  if (Number(msg.conversation_id) === Number(activeConversationId.value)) {
    appendMessage(msg)
    await nextTick(scrollToBottom)
    await markActiveRead()
  } else {
    await loadUnreadSummary()
  }
  await loadConversations()
}

function appendMessage(msg) {
  messages.value = mergeMessages([...messages.value, msg])
}

function mergeMessages(items) {
  const seen = new Set()
  return items.filter((item) => {
    const key = messageKey(item)
    if (seen.has(key)) return false
    seen.add(key)
    return true
  }).sort((a, b) => Number(a.id || 0) - Number(b.id || 0))
}

function updatePresence(data) {
  const userID = Number(data?.user_id || 0)
  const status = data?.status || 'offline'
  if (!userID) return
  contacts.value = contacts.value.map((item) => Number(item.user_id) === userID ? { ...item, status } : item)
  conversations.value = conversations.value.map((item) => Number(item.peer_id) === userID ? { ...item, peer_status: status } : item)
}

function normalizeEventData(data) {
  if (!data) return {}
  if (typeof data === 'string') {
    try { return JSON.parse(data) } catch { return {} }
  }
  return data
}

function focusInput() {
  nextTick(() => inputRef.value?.focus?.())
}

function displayName(item) {
  return item?.name || item?.username || '-'
}

function peerName(item) {
  return item?.peer_name || item?.peer_username || '-'
}

function initials(item) {
  return String(displayName(item) || 'U').slice(0, 1).toUpperCase()
}

function peerInitials(item) {
  return String(peerName(item) || 'U').slice(0, 1).toUpperCase()
}

function subtitle(item) {
  return [item.department, item.post_name].filter(Boolean).join(' · ') || t('im.departmentEmpty')
}

function presenceLabel(status) {
  return ({ online: t('im.online'), away: t('im.away'), offline: t('im.offline') }[status] || t('im.offline'))
}

function previewText(item) {
  const payloadPreview = localizedPayloadText(parsePayloadJSON(item?.last_message_payload_json), 'content')
  const preview = payloadPreview || item.last_message_preview
  if (!preview) return t('im.noMessages')
  return item.last_message_type === 'emoji' ? `${t('im.emojiMessage')} ${String(preview).replace(/^\[emoji\]\s*/, '')}` : preview
}

function parsePayloadJSON(value) {
  try {
    return typeof value === 'string' ? JSON.parse(value || '{}') : (value || {})
  } catch {
    return {}
  }
}

function payloadParams(info, scope = '') {
  const params = scope ? (info?.[`${scope}_i18n_params`] || info?.i18n_params) : info?.i18n_params
  if (!params || typeof params !== 'object') return {}
  const normalized = { ...params }
  for (const [key, value] of Object.entries(params)) {
    const paramKey = `${key}_i18n_key`
    if (params[paramKey]) normalized[key] = t(params[paramKey])
    else if (key === 'upload_status') normalized[key] = localizedLegacyValue(value)
  }
  return normalized
}

function localizedPayloadText(info, scope = 'content') {
  const key = scope === 'title' ? (info?.title_i18n_key || info?.name_i18n_key) : info?.i18n_key
  if (key) return t(key, payloadParams(info, scope === 'title' ? 'title' : ''))
  const fallback = scope === 'title'
    ? (info?.fallback_title || info?.title)
    : (info?.fallback_content || info?.content || info?.summary)
  return localizeLegacyGeneratedText(fallback || '')
}

function messagePayload(msg) {
  return parsePayloadJSON(msg?.payload_json)
}

function messageContent(msg) {
  return localizedPayloadText(messagePayload(msg), 'content') || localizeLegacyGeneratedText(msg?.content || '')
}

function localizedLegacyValue(value) {
  const text = String(value || '').trim()
  return ({
    '可上传': t('storage.uploadStatus.available'),
    '存储增购已到期，请续费后继续上传': t('storage.uploadStatus.locked')
  }[text] || value)
}

function localizeLegacyGeneratedText(value) {
  let text = String(value || '')
  if (!text) return ''
  const replacements = [
    ['上传状态：可上传', `上传状态：${t('storage.uploadStatus.available')}`],
    ['Upload status: 可上传', `Upload status: ${t('storage.uploadStatus.available')}`],
    ['上传状态：存储增购已到期，请续费后继续上传', `上传状态：${t('storage.uploadStatus.locked')}`],
    ['Upload status: 存储增购已到期，请续费后继续上传', `Upload status: ${t('storage.uploadStatus.locked')}`],
    ['项目计划完成提醒', t('notification.project.due.titleShort')],
    ['项目计划完成时间临近或已逾期，请检查交付风险。', t('notification.project.due.content')],
    ['项目跟进临近提醒', t('notification.project.nextFollow.titleShort')],
    ['项目下次跟进时间临近，请及时处理。', t('notification.project.nextFollow.content')],
    ['项目长期未跟进提醒', t('notification.project.idle.titleShort')],
    ['项目超过7天没有跟进记录，请确认进度。', t('notification.project.idle.content')]
  ]
  for (const [from, to] of replacements) {
    text = text.replaceAll(from, to)
  }
  return text
}

function supportCategoryLabel(value, fallback = '') {
  return ({
    feedback: t('im.support.category.feedback'),
    feature_suggestion: t('im.support.category.featureSuggestion'),
    bug: t('im.support.category.bug'),
    billing_ai: t('im.support.category.billingAi'),
    account_permission: t('im.support.category.accountPermission'),
    data_permission: t('im.support.category.dataPermission'),
    storage_purchase: t('im.support.category.storagePurchase'),
    other: t('im.support.category.other')
  }[value] || fallback || t('im.support.category.other'))
}

function supportPriorityLabel(value, fallback = '') {
  return ({
    normal: t('im.support.priority.normal'),
    urgent: t('im.support.priority.urgent'),
    critical: t('im.support.priority.critical')
  }[value] || fallback || t('im.support.priority.normal'))
}

function supportConversationTitle(item) {
  const info = parsePayloadJSON(item?.payload_json)
  return localizedPayloadText(info, 'title') || item?.title || t('platform.messages.supportConversations')
}

function supportConversationPreview(item) {
  return [supportCategoryLabel(item?.category, item?.category_label), supportPriorityLabel(item?.priority, item?.priority_label), supportConversationTitle(item) || previewText(item)].filter(Boolean).join(' · ')
}

function systemPayload(item) {
  return parsePayloadJSON(item?.payload_json)
}

function systemTitle(item) {
  const info = systemPayload(item)
  return localizedPayloadText(info, 'title') || item?.name || item?.code || '-'
}

function systemSender(item) {
  const info = systemPayload(item)
  if (info.sender_i18n_key) return t(info.sender_i18n_key, payloadParams(info, 'sender'))
  return info.sender_name || t('domain.auto.1a1f6dff78')
}

function systemMessageContent(item) {
  const info = systemPayload(item)
  return localizedPayloadText(info, 'content') || item?.code || '-'
}

function isSystemUnread(item) {
  return item?.status === 'open' || item?.status === 'new'
}

function systemStatusLabel(status) {
  return ({
    open: t('domain.auto.1e230aa201'),
    new: t('domain.auto.1e230aa201'),
    read: t('domain.auto.642ec8b596'),
    archived: t('domain.auto.5cfbea2b76')
  }[status] || status || '-')
}

function activeSystemPayload() {
  return systemPayload(activeSystemNotification.value)
}

const activeSystemContent = computed(() => systemMessageContent(activeSystemNotification.value))
const activeSystemSender = computed(() => systemSender(activeSystemNotification.value))
const activeSystemRelated = computed(() => {
  const info = activeSystemPayload()
  const domain = info.domain || info.object_domain || info.module || ''
  const objectID = info.object_id || info.related_object_id || info.id || ''
  return { domain, objectID }
})
const hasSystemRelatedObject = computed(() => Boolean(activeSystemRelated.value.domain && activeSystemRelated.value.objectID))
const activeSystemRelatedLabel = computed(() => {
  const { domain, objectID } = activeSystemRelated.value
  if (!domain || !objectID) return '-'
  const info = activeSystemPayload()
  const inlineName = info.object_name || info.related_object_name || info.name
  const cachedName = relatedObjectCache.value[relatedObjectKey(domain, objectID)]?.name
  return [systemDomainLabel(domain), cachedName || inlineName || `#${objectID}`].filter(Boolean).join(' ')
})

watch(() => activeSystemRelated.value, (value) => {
  if (systemDetailVisible.value && value.domain && value.objectID) preloadSystemRelatedObject(activeSystemNotification.value)
}, { deep: true })

function systemDomainLabel(domain) {
  return ({
    leads: t('domain.auto.8de0b8fb1d'),
    opportunities: t('domain.auto.2559c4e3c0'),
    deals: t('domain.auto.6a30a471cf'),
    closed_deal: t('domain.auto.6a30a471cf'),
    orders: t('domain.auto.81b3798beb'),
    contacts: t('domain.auto.2425bd4bc1'),
    companies: t('domain.auto.5d80806d7f'),
    tickets: t('domain.auto.85c230ed15'),
    projects: t('domain.auto.22336e6b89'),
    campaigns: t('domain.auto.5fa95b1b73'),
    workspace: t('domain.auto.a5361c67df')
  }[domain] || domain)
}

function relatedObjectKey(domain, objectID) {
  return `${normalizeRelatedDomain(domain)}:${objectID}`
}

function normalizeRelatedDomain(domain) {
  return ({ closed_deal: 'deals' }[domain] || domain || '').trim()
}

function relatedObjectName(row) {
  if (!row) return ''
  const payload = (() => {
    try {
      return typeof row.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row.payload_json || {})
    } catch {
      return {}
    }
  })()
  return row.name || payload.name || payload.project_name || payload.title || payload.subject || ''
}

async function preloadSystemRelatedObject(item) {
  const info = systemPayload(item)
  const domain = info.domain || info.object_domain || info.module || ''
  const objectID = info.object_id || info.related_object_id || info.id || ''
  const apiDomain = normalizeRelatedDomain(domain)
  if (!apiDomain || !objectID || relatedObjectCache.value[relatedObjectKey(domain, objectID)]) return
  try {
    const res = await api.get(`/api/v1/prd/${apiDomain}/objects/${objectID}`)
    relatedObjectCache.value = {
      ...relatedObjectCache.value,
      [relatedObjectKey(domain, objectID)]: {
        id: objectID,
        domain: apiDomain,
        name: relatedObjectName(res.data)
      }
    }
  } catch {
    relatedObjectCache.value = {
      ...relatedObjectCache.value,
      [relatedObjectKey(domain, objectID)]: { id: objectID, domain: apiDomain, name: '' }
    }
  }
}

function goSystemRelatedObject() {
  const { domain, objectID } = activeSystemRelated.value
  if (!domain || !objectID) return
  const routeMap = {
    leads: '/leads',
    opportunities: '/opportunities',
    deals: '/deals',
    closed_deal: '/deals',
    orders: '/orders',
    contacts: '/contacts',
    companies: '/contacts',
    tickets: '/tickets',
    projects: '/projects',
    campaigns: '/campaigns',
    workspace: '/workspace'
  }
  router.push({ path: routeMap[domain] || `/${domain}`, query: { id: objectID } })
  systemDetailVisible.value = false
}

function messageKey(msg) {
  return msg.id ? `id-${msg.id}` : `${msg.client_msg_id || msg.created_at}-${msg.sender_id}`
}

function shouldShowTime(index) {
  if (index === 0) return true
  const prev = new Date(messages.value[index - 1]?.created_at).getTime()
  const current = new Date(messages.value[index]?.created_at).getTime()
  if (Number.isNaN(prev) || Number.isNaN(current)) return false
  return current - prev > 5 * 60 * 1000
}

function formatTs(value, short = false) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  if (short && diff >= 0 && diff < 60 * 1000) return t('im.justNow')
  const sameDay = date.toDateString() === now.toDateString()
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  if (sameDay) return `${hh}:${mm}`
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${m}-${d} ${hh}:${mm}`
}
</script>

<style scoped>
.im-page {
  display: grid;
  grid-template-columns: minmax(220px, 260px) minmax(200px, 230px) minmax(220px, 260px) minmax(300px, 1fr);
  gap: 14px;
  height: calc(100vh - 150px);
  min-height: 680px;
  min-width: 0;
  max-width: 100%;
}


.platform-support-page {
  grid-template-columns: minmax(260px, 320px) minmax(300px, 360px) minmax(360px, 1fr);
}

.support-entry {
  padding: 14px 16px 0;
}

.support-entry-btn {
  width: 100%;
  border: 1px solid #bfdbfe;
  background: linear-gradient(135deg, #eff6ff, #ffffff);
  border-radius: 14px;
  padding: 12px;
  display: grid;
  grid-template-columns: 38px 1fr;
  gap: 3px 10px;
  text-align: left;
  cursor: pointer;
  color: #0f3f88;
}

.support-entry-btn span {
  grid-row: span 2;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  display: inline-grid;
  place-items: center;
  background: #2563eb;
  color: #fff;
  font-size: 20px;
}

.support-entry-btn strong {
  color: #0f172a;
  font-size: 14px;
}

.support-entry-btn small {
  color: #64748b;
  font-size: 12px;
}

.support-filters {
  display: grid;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid #eef2f7;
}

.system-toolbar {
  display: grid;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid #eef2f7;
}

.system-toolbar :deep(.el-segmented) {
  width: 100%;
}

.system-toolbar-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.system-toolbar-actions :deep(.el-button) {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.system-detail {
  display: grid;
  gap: 14px;
}

.system-detail h3 {
  margin: 0;
  color: #0f172a;
  font-size: 18px;
}

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.detail-meta span,
.related-object-box {
  border-radius: 999px;
  padding: 6px 10px;
  background: #f1f5f9;
  color: #475569;
  font-size: 12px;
}

.system-detail p {
  margin: 0;
  padding: 14px;
  border-radius: 12px;
  background: #f8fafc;
  color: #1f2937;
  line-height: 1.7;
  white-space: pre-wrap;
}

.related-object-box {
  width: fit-content;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  background: #eff6ff;
  color: #1d4ed8;
  cursor: pointer;
}

.support-form :deep(.el-select) {
  width: 100%;
}

.broadcast-tip {
  margin-bottom: 14px;
}

.im-shell {
  min-width: 0;
  max-width: 100%;
  min-height: 0;
  overflow: hidden;
  background: linear-gradient(180deg, #ffffff 0%, #fbfcff 100%);
  border: 1px solid #e5e9f1;
  border-radius: 18px;
  box-shadow: 0 18px 50px rgba(24, 39, 75, 0.06);
  display: flex;
  flex-direction: column;
}

.im-panel-head,
.im-chat-head {
  padding: 16px 18px;
  border-bottom: 1px solid #eef2f7;
}

.strong-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.strong-head div,
.chat-peer div {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.strong-head strong,
.chat-peer strong {
  font-size: 16px;
  color: #111827;
}

.strong-head small,
.chat-peer small {
  color: #64748b;
  font-size: 12px;
}

.im-filters {
  padding: 14px 16px;
  border-bottom: 1px solid #eef2f7;
}

.im-list,
.im-msg-list {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.im-person-row,
.im-conv-row,
.system-message-row {
  width: calc(100% - 16px);
  margin: 5px 8px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  padding: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  text-align: left;
  transition: background .18s ease, transform .18s ease, box-shadow .18s ease;
}

.im-person-row:hover,
.im-conv-row:hover,
.system-message-row:hover,
.im-person-row.active,
.im-conv-row.active {
  background: #eef6ff;
  box-shadow: inset 3px 0 0 #2f7df6;
}

.im-person-row:active,
.im-conv-row:active,
.system-message-row:active {
  transform: scale(.99);
}

.system-message-row {
  align-items: flex-start;
}

.system-message-row.unread {
  background: #fff7ed;
  box-shadow: inset 3px 0 0 #f59e0b;
}

.system-message-row.archived {
  opacity: .72;
}

.system-message-icon {
  width: 34px;
  height: 34px;
  flex: none;
  border-radius: 12px;
  display: inline-grid;
  place-items: center;
  color: #1761d8;
  background: #eef6ff;
}

.system-message-main {
  min-width: 0;
  flex: 1;
  display: grid;
  gap: 4px;
}

.system-message-main strong,
.system-message-main small,
.system-message-main em {
  overflow: hidden;
  text-overflow: ellipsis;
}

.system-message-main strong {
  color: #111827;
  font-size: 14px;
  white-space: nowrap;
}

.system-message-main small {
  color: #64748b;
  font-size: 12px;
  white-space: nowrap;
}

.system-message-main em {
  display: -webkit-box;
  color: #475569;
  font-size: 12px;
  font-style: normal;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.avatar-wrap {
  position: relative;
  width: 42px;
  height: 42px;
  border-radius: 15px;
  overflow: visible;
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #2455a6;
  font-weight: 800;
  background: linear-gradient(135deg, #dbeafe, #f0f7ff);
  box-shadow: inset 0 0 0 1px rgba(47, 125, 246, .12);
}

.avatar-wrap img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: inherit;
}

.avatar-wrap.big {
  width: 52px;
  height: 52px;
  border-radius: 18px;
}

.avatar-wrap.small {
  width: 32px;
  height: 32px;
  border-radius: 12px;
  font-size: 12px;
}

.presence-dot {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid #fff;
  background: #94a3b8;
}

.presence-online { background: #16a34a; }
.presence-away { background: #f59e0b; }
.presence-offline { background: #94a3b8; }

.person-main,
.conv-main {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.person-main strong,
.conv-top strong {
  color: #111827;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.person-main small,
.conv-bottom small,
.conv-top small {
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conv-top,
.conv-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.conv-bottom small {
  flex: 1;
}

.pin-icon {
  color: #f59e0b;
  flex: none;
}

.im-chat-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.chat-peer {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.retention-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 999px;
  color: #9a5b00;
  background: #fff7ed;
  border: 1px solid #fed7aa;
  font-size: 12px;
  white-space: nowrap;
}

.im-msg-list {
  padding: 18px 22px;
  background:
    radial-gradient(circle at 18% 0%, rgba(47, 125, 246, .07), transparent 28%),
    linear-gradient(180deg, #f8fbff 0%, #f6f7fb 100%);
}

.load-more {
  display: block;
  margin: 0 auto 12px;
}

.msg-block {
  margin-bottom: 10px;
}

.msg-time-sep {
  width: fit-content;
  max-width: 80%;
  margin: 14px auto;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(100, 116, 139, .12);
  color: #64748b;
  font-size: 11px;
}

.msg-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.msg-block.self .msg-row {
  justify-content: flex-end;
}

.bubble {
  max-width: min(68%, 720px);
  padding: 10px 12px;
  border-radius: 16px 16px 16px 6px;
  background: #fff;
  border: 1px solid #e5e9f1;
  color: #1f2937;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
  box-shadow: 0 8px 24px rgba(15, 23, 42, .06);
}

.msg-block.self .bubble {
  border-radius: 16px 16px 6px 16px;
  background: linear-gradient(135deg, #2f7df6, #1761d8);
  border-color: transparent;
  color: #fff;
}

.bubble.emoji {
  font-size: 28px;
  line-height: 1.2;
}

.im-input {
  border-top: 1px solid #eef2f7;
  padding: 12px 14px 14px;
  background: #fff;
}

.composer-tools,
.composer-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.composer-tools {
  margin-bottom: 8px;
}

.composer-footer {
  margin-top: 10px;
}

.tool-btn {
  font-size: 19px;
}

.composer-hint,
.ws-state {
  color: #94a3b8;
  font-size: 12px;
}

.ws-state.on { color: #16a34a; }
.ws-state.off { color: #f59e0b; }

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 7px;
}

.emoji-btn {
  border: 0;
  background: #f8fafc;
  padding: 9px 0;
  border-radius: 10px;
  cursor: pointer;
  font-size: 19px;
}

.emoji-btn:hover {
  background: #eaf3ff;
}

.im-empty,
.im-empty-mini {
  color: #64748b;
  text-align: center;
}

.im-empty-mini {
  padding: 32px 12px;
  font-size: 13px;
}

.im-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
}

.im-empty svg {
  font-size: 52px;
  color: #2f7df6;
}

.im-empty strong {
  color: #111827;
  font-size: 18px;
}

.im-empty p {
  margin: 0;
  font-size: 13px;
}

@media (max-width: 1500px) {
  .im-page {
    grid-template-columns: minmax(220px, 260px) minmax(220px, 260px) minmax(320px, 1fr);
  }

  .platform-support-page {
    grid-template-columns: 280px 320px minmax(360px, 1fr);
  }

  .im-system {
    grid-column: 1 / -1;
    min-height: 220px;
  }
}

@media (max-width: 1180px) {
  .im-page {
    grid-template-columns: minmax(240px, 320px) minmax(340px, 1fr);
    height: auto;
  }

  .im-system,
  .im-contacts {
    grid-column: auto;
  }
}

@media (max-width: 900px) {
  .im-page {
    grid-template-columns: 1fr;
    height: auto;
  }

  .im-shell {
    min-height: 260px;
  }
}
</style>
