<script setup>
import { ref, computed, watch } from 'vue'
import TreeSelect from 'primevue/treeselect'
import { useAccounts } from '@/composables/useAccounts.js'

/**
 * AccountSelector Component
 * 
 * Provides a hierarchical account selector that groups accounts by provider.
 * 
 * IMPORTANT: The component passes {id:true} to form validation, not a number!
 */

// -----------------------------------------------------------------------------
// Props & Emits
// -----------------------------------------------------------------------------
const props = defineProps({
    modelValue: {
        type: [Number, null],
        default: null
    },
    name: {
        type: String,
        required: true
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

// -----------------------------------------------------------------------------
// Data & Computed Properties
// -----------------------------------------------------------------------------
// Get accounts data from the composable
const { accounts, isLoading, isError } = useAccounts()

// Currently selected node
const selectedTreeNode = ref(null)

// Transform accounts data into tree structure for TreeSelect
const accountsTree = computed(() => {
    if (!accounts.value) return []

    return accounts.value.map((provider) => ({
        key: `provider-${provider.id}`,
        label: provider.name,
        selectable: false,
        children: provider.accounts.map((account) => ({
            key: account.id,
            label: `${account.name} (${account.currency})`,
            provider: provider.name,
            data: account
        }))
    }))
})

// Keep all provider nodes expanded by default
const expandedKeys = computed(() => {
    if (!accounts.value) return {}

    return accounts.value.reduce((acc, provider) => {
        acc[`provider-${provider.id}`] = true
        return acc
    }, {})
})

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------
// Extract first item from array if needed
const unwrapNode = (val) => (Array.isArray(val) ? val[0] : val)

// Format display label for selected account
const formatSelectedLabel = (val) => {
    if (!val || (Array.isArray(val) && val.length === 0)) {
        return props.placeholder
    }
    const node = unwrapNode(val)
    return node?.provider && node?.label ? `${node.provider}/${node.label}` : props.placeholder
}

// -----------------------------------------------------------------------------
// Watchers & Event Handlers
// -----------------------------------------------------------------------------
// Update selected node when modelValue (account ID) changes
watch(
    () => props.modelValue,
    (newAccountId) => {
        if (!accounts.value || newAccountId == null) {
            selectedTreeNode.value = null
            return
        }

        // Find the account that matches the ID and set it as selected
        for (const provider of accounts.value) {
            const account = provider.accounts.find((acc) => acc.id === newAccountId)
            if (account) {
                selectedTreeNode.value = {
                    key: account.id,
                    label: `${account.name} (${account.currency})`,
                    provider: provider.name,
                    data: account
                }
                return
            }
        }

        // Reset if account not found
        selectedTreeNode.value = null
    },
    { immediate: true }
)

// Handle selection change from TreeSelect component
const handleSelectionChange = (val) => {
    const selected = unwrapNode(val)
    selectedTreeNode.value = selected
    
    // Extract the ID (or null) and emit to parent
    const accountId = selected?.key ?? null
    emit('update:modelValue', accountId)
}
</script>

<template>
    <div class="account-select">
        <TreeSelect
            :modelValue="selectedTreeNode"
            @update:modelValue="handleSelectionChange"
            :name="name"
            :options="accountsTree"
            :expandedKeys="expandedKeys"
            :disabled="disabled || isLoading"
            :placeholder="placeholder"
            :showClear="true"
            selectionMode="single"
            :required="required"
            class="w-full"
            :loading="isLoading"
            fluid
        >
            <!-- Empty state templates -->
            <template #empty>
                <div class="p-2" v-if="isLoading">Loading accounts...</div>
                <div class="p-2 text-red-500" v-else-if="isError">Failed to load accounts</div>
                <div class="p-2" v-else>No accounts found</div>
            </template>
            
            <!-- Selected value display template -->
            <template #value="slotProps">
                <div v-if="slotProps.value">
                    {{ formatSelectedLabel(slotProps.value) }}
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
