<template>
  <BaseDialog :show="show" :title="t('admin.accounts.openaiRegister.title')" size="lg" @close="$emit('close')">
    <div class="space-y-6">
      <!-- Mode Selection -->
      <div class="flex gap-4">
        <button
          v-for="m in modes"
          :key="m.value"
          @click="mode = m.value"
          :class="[
            'flex-1 rounded-lg border-2 p-4 text-left transition-all',
            mode === m.value
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-dark-600 dark:hover:border-dark-500'
          ]"
        >
          <div class="font-medium text-gray-900 dark:text-white">{{ m.label }}</div>
          <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ m.desc }}</div>
        </button>
      </div>

      <!-- Auto Register Form -->
      <div v-if="mode === 'auto'" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.openaiRegister.email') }}</label>
          <div class="flex gap-2">
            <Input v-model="form.email" type="email" :placeholder="t('admin.accounts.openaiRegister.emailPlaceholder')" class="flex-1" />
            <button @click="generateTempEmail" :disabled="generatingEmail" class="rounded-lg bg-gray-600 px-3 py-2 text-sm text-white hover:bg-gray-700 disabled:opacity-50">
              {{ generatingEmail ? '...' : '生成邮箱' }}
            </button>
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.openaiRegister.password') }}</label>
          <Input v-model="form.password" type="password" :placeholder="t('admin.accounts.openaiRegister.passwordPlaceholder')" />
        </div>
        <!-- 自动化流程提示 -->
        <div class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
          <p class="text-sm text-blue-700 dark:text-blue-300">
            🚀 一键自动完成：注册 → 获取验证码 → 验证邮箱 → 登录获取 RT
          </p>
        </div>
        <p v-if="codeError" class="text-sm text-red-500">{{ codeError }}</p>
      </div>

      <!-- Session to RT Form -->
      <div v-else class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.openaiRegister.sessionToken') }}</label>
          <TextArea v-model="form.sessionToken" :rows="3" :placeholder="t('admin.accounts.openaiRegister.sessionTokenPlaceholder')" />
        </div>
      </div>

      <!-- Common Options -->
      <div class="space-y-4 border-t border-gray-200 pt-4 dark:border-dark-600">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.proxy') }}</label>
          <ProxySelector v-model="form.proxyId" :proxies="proxies" />
        </div>
        <div class="flex items-center gap-2">
          <input type="checkbox" v-model="form.createAccount" id="createAccount" class="rounded border-gray-300 text-primary-600" />
          <label for="createAccount" class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.openaiRegister.createAccount') }}</label>
        </div>
        <div v-if="form.createAccount" class="space-y-4 pl-6">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.name') }}</label>
            <Input v-model="form.name" :placeholder="t('admin.accounts.openaiRegister.namePlaceholder')" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.groups') }}</label>
            <GroupSelector v-model="form.groupIds" :groups="groups" multiple />
          </div>
        </div>
      </div>

      <!-- Result -->
      <div v-if="result" :class="['rounded-lg p-4', result.success ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20']">
        <div :class="['font-medium', result.success ? 'text-green-800 dark:text-green-300' : 'text-red-800 dark:text-red-300']">
          {{ result.success ? t('admin.accounts.openaiRegister.success') : t('admin.accounts.openaiRegister.failed') }}
        </div>
        <div v-if="result.refresh_token" class="mt-2">
          <label class="text-sm text-gray-600 dark:text-gray-400">Refresh Token:</label>
          <code class="mt-1 block break-all rounded bg-gray-100 p-2 text-xs dark:bg-dark-700">{{ result.refresh_token }}</code>
        </div>
        <div v-if="result.error" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ result.error }}</div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="rounded-lg border border-gray-300 px-4 py-2 text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700">
          {{ t('common.cancel') }}
        </button>
        <button @click="submit" :disabled="loading || !isValid" class="rounded-lg bg-primary-600 px-4 py-2 text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50">
          <span v-if="loading" class="flex items-center gap-2">
            <LoadingSpinner size="sm" />
            {{ t('common.processing') }}
          </span>
          <span v-else>{{ t('common.submit') }}</span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BaseDialog, LoadingSpinner } from '@/components/common'
import Input from '@/components/common/Input.vue'
import TextArea from '@/components/common/TextArea.vue'
import ProxySelector from '@/components/common/ProxySelector.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import { openaiRegisterAPI } from '@/api/admin'
import type { Proxy, Group } from '@/types'

defineProps<{
  show: boolean
  proxies: Proxy[]
  groups: Group[]
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const { t } = useI18n()

const mode = ref<'auto' | 'session'>('session')
const loading = ref(false)
const result = ref<{ success: boolean; refresh_token?: string; account_id?: number; error?: string } | null>(null)

// 邮局相关状态
const generatingEmail = ref(false)
const codeError = ref('')

// 邮局配置（硬编码）
const MAIL_CONFIG = {
  baseUrl: 'https://cloud-mail.enrun.ggff.net',
  adminEmail: 'admin@enrun.ggff.net',
  adminPassword: 'QAZplm123@mail',
  domains: ['enrun.ggff.net', 'panrun.me', 'runoy.me', '11sa.site', 'augmentss.icu', 'busaug.shop']
}

const form = ref({
  email: '',
  password: '',
  sessionToken: '',
  proxyId: null as number | null,
  name: '',
  groupIds: [] as number[],
  createAccount: false
})

const modes = [
  { value: 'auto' as const, label: t('admin.accounts.openaiRegister.autoMode'), desc: t('admin.accounts.openaiRegister.autoModeDesc') },
  { value: 'session' as const, label: t('admin.accounts.openaiRegister.sessionMode'), desc: t('admin.accounts.openaiRegister.sessionModeDesc') }
]

const isValid = computed(() => {
  if (mode.value === 'auto') {
    return form.value.email && form.value.password
  }
  return form.value.sessionToken
})

// 生成临时邮箱
async function generateTempEmail() {
  generatingEmail.value = true
  try {
    const randomStr = Math.random().toString(36).substring(2, 10)
    const domain = MAIL_CONFIG.domains[Math.floor(Math.random() * MAIL_CONFIG.domains.length)]
    const email = `openai_${randomStr}`

    const resp = await openaiRegisterAPI.createMailUser({
      email,
      domain,
      admin_email: MAIL_CONFIG.adminEmail,
      admin_password: MAIL_CONFIG.adminPassword,
      base_url: MAIL_CONFIG.baseUrl
    })

    form.value.email = resp.email
    // 生成默认密码
    if (!form.value.password) {
      form.value.password = 'OpenAI@' + Math.random().toString(36).substring(2, 10)
    }
  } catch (e: any) {
    codeError.value = e.message || '生成邮箱失败'
  } finally {
    generatingEmail.value = false
  }
}

async function submit() {
  loading.value = true
  result.value = null
  codeError.value = ''
  try {
    if (mode.value === 'auto') {
      // 一键自动完成：注册 → 获取验证码 → 验证邮箱 → 登录获取RT
      result.value = await openaiRegisterAPI.autoRegister({
        email: form.value.email,
        password: form.value.password,
        proxy_id: form.value.proxyId ?? undefined,
        name: form.value.name || undefined,
        group_ids: form.value.groupIds.length ? form.value.groupIds : undefined,
        create_account: form.value.createAccount,
        // 传递邮局配置，后端自动轮询获取验证码
        mail_base_url: MAIL_CONFIG.baseUrl,
        mail_admin_email: MAIL_CONFIG.adminEmail,
        mail_admin_password: MAIL_CONFIG.adminPassword
      })
    } else {
      result.value = await openaiRegisterAPI.sessionToRT({
        session_token: form.value.sessionToken,
        proxy_id: form.value.proxyId ?? undefined,
        name: form.value.name || undefined,
        group_ids: form.value.groupIds.length ? form.value.groupIds : undefined,
        create_account: form.value.createAccount
      })
    }
    if (result.value.success && result.value.account_id) {
      emit('created')
    }
  } catch (e: any) {
    result.value = { success: false, error: e.message || 'Unknown error' }
  } finally {
    loading.value = false
  }
}
</script>

