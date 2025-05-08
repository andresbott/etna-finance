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

// Create expanded keys object to keep tree expanded
const expandedKeys = computed(() => {
    if (!accounts.value) return {}

    // Create an object with all provider keys set to true
    return accounts.value.reduce((acc, provider) => {
        acc[`provider-${provider.id}`] = true
        return acc
    }, {})
})

// Transform accounts data into tree structure
const accountsTree = computed(() => {
    if (!accounts.value) return []

    return accounts.value.map((provider) => ({
        key: `provider-${provider.id}`,
        label: provider.name,
        selectable: false,
        children: provider.accounts.map((account) => ({
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
const unwrapNode = (val) => (Array.isArray(val) ? val[0] : val)
</script>

<template>
    <div class="account-select">
        <TreeSelect
            v-model="value"
            :options="accountsTree"
            :expandedKeys="expandedKeys"
            :disabled="disabled || isLoading"
            :placeholder="placeholder"
            :name="name"
            selectionMode="single"
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
                    {{ unwrapNode(slotProps.value)?.label || placeholder }}
                </div>
                <span v-else>{{ slotProps.placeholder }}</span>
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
