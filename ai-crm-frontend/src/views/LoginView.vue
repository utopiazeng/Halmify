<template>
  <div class="art-login-page">
    <section class="login-visual">
      <div class="visual-head">
        <img class="visual-logo-image" src="/halmify-logo-horizontal.svg" :alt="BRAND_NAME" />
      </div>
      <div class="visual-illustration" aria-hidden="true">
        <ArtCrmHero />
      </div>
      <div class="visual-foot">
        <h2>{{ t('login.visualTitle') }}</h2>
        <p>{{ t('login.visualDesc') }}</p>
      </div>
    </section>

    <section class="login-panel-wrap">
      <div class="login-locale">
        <el-dropdown trigger="click" @command="changeLocale">
          <button type="button" class="locale-trigger" :aria-label="t('language')">
            <Icon icon="ri:global-line" />
            <span class="locale-code">{{ currentLocaleShortLabel }}</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="item in localeOptions"
                :key="item.value"
                :command="item.value"
                :class="{ 'is-active-locale': item.value === i18nState.locale }"
              >
                {{ item.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <div
        v-if="showMaintenanceNotice"
        class="crm-maintenance-notice login-maintenance-notice"
        role="status"
        aria-live="polite"
        :dir="maintenanceNotice.locale === 'ar-SA' ? 'rtl' : 'ltr'"
      >
        <div class="crm-maintenance-icon" aria-hidden="true">!</div>
        <div class="crm-maintenance-body">
          <span>{{ maintenanceNotice.label }}</span>
          <strong>{{ maintenanceNotice.title }}</strong>
          <p>{{ maintenanceNotice.text }}</p>
        </div>
      </div>

      <el-card class="login-panel" shadow="never">
        <div class="panel-top-gap"></div>
        <div class="panel-head">
          <h1>{{ t('login.welcome') }}</h1>
          <p>{{ t('login.subtitle') }}</p>
        </div>

        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="login">
          <el-form-item :label="t('login.username')" prop="username">
            <el-input v-model="form.username" :placeholder="t('login.usernamePlaceholder')" autocomplete="username" />
          </el-form-item>

          <el-form-item :label="t('login.password')" prop="password">
            <el-input v-model="form.password" type="password" show-password :placeholder="t('login.passwordPlaceholder')" autocomplete="current-password" />
          </el-form-item>

          <el-form-item v-if="mfaRequired" :label="t('login.mfa')" prop="mfa_code">
            <el-input v-model="form.mfa_code" :placeholder="t('login.mfaPlaceholder')" />
          </el-form-item>

          <div ref="sliderTrackRef" class="slider-verify" :class="sliderClass">
            <button
              ref="sliderHandleRef"
              type="button"
              class="slider-handle"
              :style="{ left: `${sliderState.handleLeft}px` }"
              @pointerdown="startSlide"
              :disabled="sliderState.verified || sliderState.verifying"
            >
              <span>»</span>
            </button>
            <div class="slider-track-fill" :style="{ width: `${sliderState.handleLeft + sliderState.handleWidth}px` }"></div>
            <span class="slider-text">{{ sliderText }}</span>
          </div>

          <el-form-item class="login-submit">
            <el-button type="primary" native-type="submit" :loading="loading" :disabled="!sliderState.verified" :aria-label="t('login.submit')">{{ t('login.submit') }}</el-button>
          </el-form-item>
        </el-form>

        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
      </el-card>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus/es/components/message/index'
import api from '../api'
import ArtCrmHero from '../components/art/ArtCrmHero.vue'
import { Icon } from '@iconify/vue'
import { BRAND_NAME } from '../brand'
import { i18nState, localeOptions, setLocale, t } from '../i18n'
import { apiErrorMessage, normalizeApiErrorResponse } from '../utils/api-error'

const DEMO_CREDENTIALS = Object.freeze({
  username: 'testaccount',
  password: 'HalmifyDemo2026!'
})

const router = useRouter()
const authApi = axios.create({
  baseURL: import.meta.env.BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})
authApi.interceptors.request.use((config) => {
  config.headers = config.headers || {}
  config.headers['Accept-Language'] = i18nState.locale || 'zh-CN'
  return config
})
authApi.interceptors.response.use(
  (response) => response,
  (error) => {
    normalizeApiErrorResponse(error)
    return Promise.reject(error)
  }
)
const loading = ref(false)
const error = ref('')
const mfaRequired = ref(false)
const formRef = ref(null)
const sliderTrackRef = ref(null)
const sliderHandleRef = ref(null)
const maintenanceNotice = ref(null)
const form = reactive({
  username: DEMO_CREDENTIALS.username,
  password: DEMO_CREDENTIALS.password,
  mfa_code: '',
  captcha_token: ''
})
const rules = {
  username: [{ required: true, message: t('login.inputUsername'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.inputPassword'), trigger: 'blur' }],
  mfa_code: [{ required: true, message: t('login.inputMfa'), trigger: 'blur' }]
}

const sliderState = reactive({
  challengeId: '',
  trackLength: 320,
  handleWidth: 44,
  handleLeft: 0,
  dragging: false,
  dragStartX: 0,
  baseLeft: 0,
  trace: [],
  startedAt: 0,
  verified: false,
  verifying: false,
  error: ''
})

const sliderText = computed(() => {
  if (sliderState.verifying) return t('login.sliderVerifying')
  if (sliderState.verified) return t('login.sliderSuccess')
  if (sliderState.error) return sliderState.error
  return t('login.sliderHint')
})

const sliderClass = computed(() => ({
  verified: sliderState.verified,
  error: Boolean(sliderState.error)
}))
const showMaintenanceNotice = computed(() => Boolean(maintenanceNotice.value?.visible && maintenanceNotice.value?.text))
const currentLocaleShortLabel = computed(() => {
  const option = localeOptions.find((item) => item.value === i18nState.locale)
  return option?.value?.split('-')?.[0]?.toUpperCase() || 'LANG'
})

const maxLeft = computed(() => Math.max(0, sliderState.trackLength - sliderState.handleWidth))

function syncSliderGeometry() {
  const trackWidth = Number(sliderTrackRef.value?.getBoundingClientRect?.().width || 0)
  const handleWidth = Number(sliderHandleRef.value?.getBoundingClientRect?.().width || 0)
  if (trackWidth > 0) sliderState.trackLength = Math.round(trackWidth)
  if (handleWidth > 0) sliderState.handleWidth = Math.round(handleWidth)
  if (sliderState.handleLeft > maxLeft.value) {
    sliderState.handleLeft = maxLeft.value
  }
}

function resetSlider(keepChallenge = false) {
  sliderState.handleLeft = 0
  sliderState.dragging = false
  sliderState.dragStartX = 0
  sliderState.baseLeft = 0
  sliderState.trace = []
  sliderState.startedAt = 0
  sliderState.verified = false
  sliderState.verifying = false
  sliderState.error = ''
  form.captcha_token = ''
  if (!keepChallenge) sliderState.challengeId = ''
}

async function initChallenge() {
  resetSlider(true)
  try {
    const res = await api.get('/api/v1/auth/captcha/challenge')
    sliderState.challengeId = res?.data?.challenge_id || ''
    // Keep challenge hint length as fallback, then sync to rendered width.
    sliderState.trackLength = Number(res?.data?.track_length || 320)
    if (!sliderState.challengeId) {
      sliderState.error = t('login.sliderInitFailed')
    }
    await nextTick()
    syncSliderGeometry()
  } catch {
    sliderState.error = t('login.sliderInitFailed')
  }
}

function startSlide(event) {
  if (sliderState.verified || sliderState.verifying || !sliderState.challengeId) return
  sliderState.dragging = true
  sliderState.dragStartX = event.clientX
  sliderState.baseLeft = sliderState.handleLeft
  sliderState.trace = [Math.round((sliderState.handleLeft / Math.max(maxLeft.value, 1)) * 100)]
  sliderState.startedAt = Date.now()
  sliderState.error = ''
  event.currentTarget?.setPointerCapture?.(event.pointerId)
}

async function endSlide() {
  if (!sliderState.dragging) return
  sliderState.dragging = false
  if (sliderState.trace.length < 2) {
    sliderState.error = t('login.sliderNeedRight')
    sliderState.handleLeft = 0
    return
  }
  if (sliderState.handleLeft < maxLeft.value * 0.98) {
    sliderState.error = t('login.sliderNeedRight')
    sliderState.handleLeft = 0
    return
  }
  await verifySlider()
}

function onPointerMove(event) {
  if (!sliderState.dragging) return
  const delta = event.clientX - sliderState.dragStartX
  const next = Math.max(0, Math.min(maxLeft.value, sliderState.baseLeft + delta))
  sliderState.handleLeft = next
  sliderState.trace.push(Math.round((next / Math.max(maxLeft.value, 1)) * 100))
}

function onPointerUp() {
  if (!sliderState.dragging) return
  endSlide()
}

async function verifySlider() {
  sliderState.verifying = true
  sliderState.error = ''
  try {
    const elapsed = Math.max(1, Date.now() - sliderState.startedAt)
    const finalPercent = Math.round((sliderState.handleLeft / Math.max(maxLeft.value, 1)) * 10000) / 100
    const res = await api.post('/api/v1/auth/captcha/verify', {
      challenge_id: sliderState.challengeId,
      elapsed_ms: elapsed,
      track: sliderState.trace,
      final_percent: finalPercent
    })
    const token = res?.data?.captcha_token
    if (!token) {
      sliderState.error = t('login.sliderFailed')
      sliderState.handleLeft = 0
      return
    }
    form.captcha_token = token
    sliderState.verified = true
    sliderState.handleLeft = maxLeft.value
  } catch (e) {
    sliderState.error = apiErrorMessage(e, t('login.sliderFailed'))
    await initChallenge()
  } finally {
    sliderState.verifying = false
  }
}

const login = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  if (!sliderState.verified || !form.captcha_token) {
    error.value = t('login.captchaRequired')
    ElMessage.error(error.value)
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = (await authApi.post('/api/v1/auth/login', form)).data
    if (res?.code === 200 && res?.data?.mfa_required) {
      mfaRequired.value = true
      error.value = t('login.mfaRequired')
      ElMessage.warning(error.value)
      await initChallenge()
      return
    }
    if (res?.code === 200 && res?.data?.token) {
      localStorage.setItem('token', res.data.token)
      localStorage.setItem('current_user', JSON.stringify(res.data.user || {}))
      localStorage.removeItem('industry_packages')
      localStorage.removeItem('industry_packages_loaded')
      sessionStorage.setItem('login_success_notice', '1')
      if (res.data.user?.language_pref) {
        localStorage.setItem('locale', res.data.user.language_pref)
      }
      router.push('/workspace')
      return
    }
    error.value = localizedLoginError({ response: { data: res } })
    ElMessage.error(error.value)
    await initChallenge()
  } catch (e) {
    error.value = localizedLoginError(e)
    ElMessage.error(error.value)
    await initChallenge()
  } finally {
    loading.value = false
  }
}

function localizedLoginError(errorLike) {
  const body = errorLike?.response?.data || errorLike || {}
  const code = body?.data?.error_code || body?.error_code || body?.message_key || body?.message
  if (code === 'TENANT_EXPIRED') return t('login.error.tenantExpired')
  if (code === 'TENANT_DISABLED') return t('login.error.tenantDisabled')
  return apiErrorMessage(errorLike, t('login.failedNoToken'))
}

async function loadMaintenanceNotice() {
  try {
    const res = await authApi.get('/api/public/maintenance-notice', {
      params: { locale: i18nState.locale }
    })
    maintenanceNotice.value = res?.data?.data || null
  } catch {
    maintenanceNotice.value = null
  }
}

async function changeLocale(locale) {
  await setLocale(locale, { syncServer: false })
  await loadMaintenanceNotice()
}

onMounted(async () => {
  await loadMaintenanceNotice()
  await initChallenge()
  syncSliderGeometry()
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('resize', syncSliderGeometry)
})

onUnmounted(() => {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('resize', syncSliderGeometry)
})
</script>
