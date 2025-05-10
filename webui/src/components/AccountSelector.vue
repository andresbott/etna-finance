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
        type: [Number, Object, null],
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
    if (!node) return props.placeholder
    
    // Handle both direct selection and object with provider/label
    if (node.provider && node.label) {
        return `${node.provider}/${node.label}`
    }
    
    return props.placeholder
}

// Function to convert between the numeric ID and the {id: true} format
const convertToFormFormat = (accountId) => {
    if (accountId === null || accountId === undefined) return null;
    return { [accountId]: true };
}

const extractIdFromFormFormat = (formValue) => {
    if (!formValue) return null;
    
    // Handle numeric ID
    if (typeof formValue === 'number') return formValue;
    
    // Handle {id: true} format
    if (typeof formValue === 'object') {
        const keys = Object.keys(formValue);
        if (keys.length > 0) {
            return parseInt(keys[0], 10);
        }
    }
    
    return null;
}

// -----------------------------------------------------------------------------
// Watchers & Event Handlers
// -----------------------------------------------------------------------------
// Update selected node when modelValue changes
watch(
    () => props.modelValue,
    (newValue) => {
        if (!accounts.value || newValue == null) {
            selectedTreeNode.value = null;
            return;
        }

        // Extract the account ID from the model value (could be number or {id: true})
        const accountId = extractIdFromFormFormat(newValue);
        
        if (accountId === null) {
            selectedTreeNode.value = null;
            return;
        }

        // Find the account that matches the ID and set it as selected
        for (const provider of accounts.value) {
            const account = provider.accounts.find((acc) => acc.id === accountId);
            if (account) {
                selectedTreeNode.value = {
                    key: account.id,
                    label: `${account.name} (${account.currency})`,
                    provider: provider.name,
                    data: account
                };
                return;
            }
        }

        // Reset if account not found
        selectedTreeNode.value = null;
    },
    { immediate: true }
)

// Handle selection change from TreeSelect component
const handleSelectionChange = (val) => {
    // Check if clearing the selection
    if (!val) {
        selectedTreeNode.value = null;
        emit('update:modelValue', null);
        return;
    }
    
    // Handle both direct node selection and object selection
    let accountId = null;
    
    if (val.key) {
        // Standard node selection containing a key property
        selectedTreeNode.value = val;
        accountId = val.key;
    } else if (typeof val === 'object') {
        // We're getting a direct {id: true} object from TreeSelect
        const keys = Object.keys(val);
        if (keys.length > 0) {
            accountId = parseInt(keys[0], 10);
            
            // Find the corresponding node to update selectedTreeNode
            if (accounts.value) {
                for (const provider of accounts.value) {
                    const account = provider.accounts.find(acc => acc.id === accountId);
                    if (account) {
                        selectedTreeNode.value = {
                            key: account.id,
                            label: `${account.name} (${account.currency})`,
                            provider: provider.name,
                            data: account
                        };
                        break;
                    }
                }
            }
        }
    }
    
    // Convert to {id: true} format for form validation
    const formValue = convertToFormFormat(accountId);
    
    // Emit to parent
    emit('update:modelValue', formValue);
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
            <template #value>
                <div v-if="selectedTreeNode">
                    {{ selectedTreeNode.provider }}/{{ selectedTreeNode.label }}
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
