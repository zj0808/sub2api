<template>
  <div class="mb-4 flex items-center justify-between p-3 bg-primary-50 dark:bg-dark-700 rounded-lg">
    <div class="flex items-center gap-4">
      <div class="flex gap-2">
        <button @click="$emit('select-all')" class="btn btn-outline btn-sm">{{ t('admin.accounts.bulkActions.selectAll') }}</button>
        <button @click="$emit('invert-selection')" class="btn btn-outline btn-sm">{{ t('admin.accounts.bulkActions.invertSelection') }}</button>
        <button v-if="selectedIds.length > 0" @click="$emit('clear')" class="btn btn-ghost btn-sm">{{ t('admin.accounts.bulkActions.clearSelection') }}</button>
      </div>
      <span v-if="selectedIds.length > 0" class="text-sm font-medium text-primary-600 dark:text-primary-400">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
    </div>
    <div v-if="selectedIds.length > 0" class="flex gap-2">
      <button @click="$emit('delete')" class="btn btn-danger btn-sm">{{ t('admin.accounts.bulkActions.delete') }}</button>
      <button @click="$emit('edit')" class="btn btn-primary btn-sm">{{ t('admin.accounts.bulkActions.edit') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
defineProps(['selectedIds'])
defineEmits(['delete', 'edit', 'clear', 'select-page', 'select-all', 'invert-selection'])
const { t } = useI18n()
</script>