<template>
  <div class="activate-page">
    <el-card class="activate-card" shadow="never">
      <h1>{{ t('domain.auto.504ade81d9') }}</h1>
      <p class="activate-subtitle">{{ t('domain.auto.4a686d4b30') }}</p>

      <el-alert
        v-if="tokenMissing"
        type="error"
        show-icon
        :closable="false"
        :title="t('domain.auto.d0bc43fd7f')"
      />

      <el-form v-else ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="submit">
        <el-form-item :label="t('domain.auto.d22c9c0085')" prop="new_password">
          <el-input
            v-model="form.new_password"
            type="password"
            show-password
            :placeholder="t('domain.auto.081aeed927')"
            autocomplete="new-password"
          />
        </el-form-item>

        <el-form-item :label="t('domain.auto.d4477adb6f')" prop="confirm_password">
          <el-input
            v-model="form.confirm_password"
            type="password"
            show-password
            :placeholder="t('domain.auto.eb82e7f0a9')"
            autocomplete="new-password"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="submitting" @click="submit">{{ t('domain.auto.7e422146dd') }}</el-button>
          <el-button @click="goLogin">{{ t('domain.auto.f2fe4ecc0f') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { t } from '../i18n'
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../api'
import { apiErrorMessage } from '../utils/api-error'

const route = useRoute()
const router = useRouter()
const formRef = ref(null)
const submitting = ref(false)
const token = String(route.query.token || '').trim()
const tokenMissing = !token

const form = reactive({
  new_password: '',
  confirm_password: ''
})

const rules = {
  new_password: [
    { required: true, message: t('domain.auto.23ba228494'), trigger: 'blur' },
    { min: 8, message: t('domain.auto.25b4d28ac9'), trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: t('domain.auto.c2d891d87f'), trigger: 'blur' },
    {
      validator: (_, value, callback) => {
        if (value !== form.new_password) {
          callback(new Error(t('domain.auto.299f76cac6')))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

async function submit() {
  if (tokenMissing) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    await api.post('/api/v1/auth/activate/password', {
      token,
      new_password: form.new_password
    })
    ElMessage.success(t('domain.auto.c9f75fa0a3'))
    router.push('/')
  } catch (e) {
    ElMessage.error(apiErrorMessage(e, t('domain.auto.d60e07d17e')))
  } finally {
    submitting.value = false
  }
}

function goLogin() {
  router.push('/')
}
</script>

<style scoped>
.activate-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(145deg, #f4f7ff 0%, #f9fbff 50%, #eef3ff 100%);
}

.activate-card {
  width: 100%;
  max-width: 460px;
}

.activate-card h1 {
  margin: 0;
  font-size: 24px;
  line-height: 1.3;
  color: #1f2f5d;
}

.activate-subtitle {
  margin: 8px 0 20px;
  color: #6c7ea5;
  font-size: 14px;
}
</style>
