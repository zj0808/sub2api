<template>
  <div class="min-h-screen bg-gray-50 px-4 py-10 dark:bg-dark-900">
    <div class="mx-auto max-w-2xl">
      <div class="card p-6">
        <!-- 弹窗模式：授权成功提示 -->
        <template v-if="isPopup && messageSent">
          <div class="text-center">
            <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
              <svg class="h-8 w-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('auth.oauth.authSuccess') }}</h1>
            <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
              {{ t('auth.oauth.autoCloseHint') }}
            </p>
            <p class="mt-4 text-xs text-gray-500 dark:text-gray-500">
              {{ t('auth.oauth.manualCloseHint') }}
            </p>
          </div>
        </template>

        <!-- 普通模式：显示code等信息 -->
        <template v-else>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">OAuth Callback</h1>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Copy the <code>code</code> (and <code>state</code> if needed) back to the admin
            authorization flow.
          </p>

          <div class="mt-6 space-y-4">
            <div>
              <label class="input-label">{{ t('auth.oauth.code') }}</label>
              <div class="flex gap-2">
                <input class="input flex-1 font-mono text-sm" :value="code" readonly />
                <button class="btn btn-secondary" type="button" :disabled="!code" @click="copy(code)">
                  Copy
                </button>
              </div>
            </div>

            <div>
              <label class="input-label">{{ t('auth.oauth.state') }}</label>
              <div class="flex gap-2">
                <input class="input flex-1 font-mono text-sm" :value="state" readonly />
                <button
                  class="btn btn-secondary"
                  type="button"
                  :disabled="!state"
                  @click="copy(state)"
                >
                  Copy
                </button>
              </div>
            </div>

            <div>
              <label class="input-label">{{ t('auth.oauth.fullUrl') }}</label>
              <div class="flex gap-2">
                <input class="input flex-1 font-mono text-xs" :value="fullUrl" readonly />
                <button
                  class="btn btn-secondary"
                  type="button"
                  :disabled="!fullUrl"
                  @click="copy(fullUrl)"
                >
                  Copy
                </button>
              </div>
            </div>

            <div
              v-if="error"
              class="rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-700 dark:bg-red-900/30"
            >
              <p class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useClipboard } from '@/composables/useClipboard'

const route = useRoute()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const code = computed(() => (route.query.code as string) || '')
const state = computed(() => (route.query.state as string) || '')
const error = computed(
  () => (route.query.error as string) || (route.query.error_description as string) || ''
)

const fullUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  return window.location.href
})

// 弹窗模式检测和自动发送
const isPopup = ref(false)
const messageSent = ref(false)

onMounted(() => {
  // 检测是否为弹窗模式（由父窗口打开）
  if (window.opener && !window.opener.closed) {
    isPopup.value = true

    // 如果有code，自动发送给父窗口
    if (code.value) {
      try {
        window.opener.postMessage(
          {
            type: 'oauth_callback',
            code: code.value,
            state: state.value,
            fullUrl: fullUrl.value
          },
          window.location.origin
        )
        messageSent.value = true

        // 3秒后自动关闭弹窗
        setTimeout(() => {
          window.close()
        }, 2000)
      } catch (e) {
        console.error('Failed to send message to parent window:', e)
      }
    }
  }
})

const copy = (value: string) => {
  if (!value) return
  copyToClipboard(value, 'Copied')
}
</script>
