<script setup>
import { ref, computed, onMounted } from 'vue'
import TreeSelect from 'primevue/treeselect'
import { useAccounts } from '@/composables/useAccounts.js'

const props = defineProps({
  modelValue: {
    type: [Number, null],
    default: null
  },
  name: {
    type: String,
    default: 'accountId'
  },
  placeholder: {
    type: String,
    default: 'Select Account'
  },
  required: {
    type: Boolean,
    default: false
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

// Get accounts from the composable
const { accounts, isLoading, isError } = useAccounts()

// Transform accounts data into tree structure
const accountsTree = computed(() => {
  if (!accounts.value) return []
  
  return accounts.value.map(provider => ({
    key: `provider-${provider.id}`,
    label: provider.name,
    selectable: false,
    children: provider.accounts.map(account => ({
      key: account.id,
      label: `${account.name} (${account.currency})`,
      data: account
    }))
  }))
})

// Handle value changes
const handleChange = (e) => {
  emit('update:modelValue', e.value)
}

// For form integration
const value = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>

<template>
  <div class="account-select">
    <TreeSelect
      v-model="value"
      :options="accountsTree"
      :disabled="disabled || isLoading"
      :placeholder="placeholder"
      :name="name"
      :required="required"
      class="w-full"
      :loading="isLoading"
      @change="handleChange"
      filter
    >
      <template #empty>
        <div class="p-2" v-if="isLoading">Loading accounts...</div>
        <div class="p-2 text-red-500" v-else-if="isError">Failed to load accounts</div>
        <div class="p-2" v-else>No accounts found</div>
      </template>
      <template #value="slotProps">
        <div v-if="slotProps.value">
          {{ accounts.value?.flatMap(p => p.accounts).find(a => a.id === slotProps.value)?.name || 'Select Account' }}
        </div>
        <span v-else>{{ placeholder }}</span>
      </template>
    </TreeSelect>
  </div>
</template>

<style scoped>
.account-select {
  width: 100%;
}

/* Add any additional custom styling here */
</style>
