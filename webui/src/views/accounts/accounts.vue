<script setup>
import { ref, computed, watch } from 'vue'
import Column from 'primevue/column'
import Button from 'primevue/button'
import TreeTable from 'primevue/treetable'
import Card from 'primevue/card'

import AccountDialog from '@/views/accounts/AccountDialog.vue'
import DeleteDialog from '@/components/common/ConfirmDialog.vue'
import AccountProviderDialog from './AccountProviderDialog.vue'

import { useAccounts } from '@/composables/useAccounts'
import { getAccountTypeLabel, ACCOUNT_TYPES } from '@/types/account'
import { useSettingsStore } from '@/store/settingsStore.js'

// Documentation URL for the accounts section (open in new tab)
const ACCOUNTS_DOCS_URL = 'https://github.com/andresbott/etna-finance#readme'

// Composables
const { accounts, isLoading, deleteAccount, deleteAccountProvider } = useAccounts()
const settings = useSettingsStore()
const instrumentAccountTypes = [ACCOUNT_TYPES.INVESTMENT, ACCOUNT_TYPES.UNVESTED]

// Reactive State
const expandedKeys = ref({})
const selectedItem = ref(null)

const accountDialogVisible = ref(false)
const isEdit = ref(false)
const selectedAccount = ref(null)

const providerDialogVisible = ref(false)
const isEditProvider = ref(false)
const selectedProvider = ref(null)

const deleteAccountDialogVisible = ref(false)
const deleteProviderDialogVisible = ref(false)

// Computed TreeTable Data
const treeTableData = computed(() => {
    if (!accounts.value) return []

    return accounts.value.map((provider) => {
        const allChildren = provider.accounts?.map((account) => ({
            key: account.id,
            data: {
                id: account.id,
                name: account.name,
                type: account.type,
                currency: account.currency,
                icon: account.icon || 'pi-wallet'
            }
        })) || []

        const children = settings.instruments
            ? allChildren
            : allChildren.filter(child => !instrumentAccountTypes.includes(child.data.type))

        return {
            key: provider.id,
            data: {
                id: provider.id,
                name: provider.name,
                description: provider.description,
                icon: provider.icon || 'pi-building'
            },
            children
        }
    })
})

// Auto-expand all provider nodes when data changes
watch(treeTableData, (data) => {
    expandedKeys.value = data.reduce((acc, node) => {
        acc[node.key] = true
        return acc
    }, {})
}, { immediate: true })

// Handlers
const openNewProviderDialog = () => {
    selectedProvider.value = null
    isEditProvider.value = false
    providerDialogVisible.value = true
}

const editProvider = (provider) => {
    selectedProvider.value = provider
    isEditProvider.value = true
    providerDialogVisible.value = true
}

const addAccountToProvider = (provider) => {
    selectedAccount.value = {
        providerId: provider.data.id,
        icon: 'pi-wallet'
    }
    isEdit.value = false
    accountDialogVisible.value = true
}

const editAccount = (account) => {
    selectedAccount.value = account
    isEdit.value = true
    accountDialogVisible.value = true
}

const showDeleteAccountDialog = (account) => {
    selectedItem.value = account
    deleteAccountDialogVisible.value = true
}

const showDeleteProviderDialog = (provider) => {
    selectedItem.value = provider
    deleteProviderDialogVisible.value = true
}

const handleDeleteAccount = async () => {
    if (selectedItem.value) {
        await deleteAccount(selectedItem.value.id)
        deleteAccountDialogVisible.value = false
    }
}

const handleDeleteProvider = async () => {
    if (selectedItem.value) {
        await deleteAccountProvider(selectedItem.value.id)
        deleteProviderDialogVisible.value = false
    }
}
</script>

<template>
    <div>
        <div class="mb-4">
            <div class="flex align-items-center gap-2 mb-2">
                <h1 class="text-2xl font-bold m-0 text-color">Account Setup</h1>
                <a
                    :href="ACCOUNTS_DOCS_URL"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex link-unstyled"
                    aria-label="About accounts"
                    v-tooltip.top="'About accounts'"
                >
                    <Button icon="pi pi-question-circle" text rounded severity="secondary" class="p-button-sm" />
                </a>
            </div>
            <p class="text-color-secondary m-0 mb-3 text-base">
                Manage account providers and accounts used to track your finances
            </p>
            <div class="flex justify-content-end">
                <Button
                    label="Add Account Provider"
                    icon="pi pi-plus"
                    @click="openNewProviderDialog"
                />
            </div>
        </div>

            <Card v-if="!accounts || accounts.length === 0">
                <template #content>
                    <div class="info-message">
                        No account providers available. Please add one to get started.
                    </div>
                </template>
            </Card>

            <Card v-else>
                <template #content>
                    <TreeTable
                        :value="treeTableData"
                        :loading="isLoading"
                        :expandedKeys="expandedKeys"
                        class="p-treetable-sm"
                    >
                        <Column field="name" header="Name" expander>
                            <template #body="{ node }">
                                <div class="flex align-items-center gap-2">
                                    <i :class="['pi', node.data.icon]" class="text-color-secondary"></i>
                                    <span>{{ node.data.name }}</span>
                                </div>
                            </template>
                        </Column>

                        <Column field="description" header="Description">
                            <template #body="{ node }">
                                <div v-if="node.children">
                                    <span>{{ node.data.description }}</span>
                                </div>
                                <div v-else>
                                    <i>{{ getAccountTypeLabel(node.data.type) }}</i>
                                </div>
                            </template>
                        </Column>

                        <Column field="currency" header="Currency">
                            <template #body="{ node }">
                                <span v-if="!node.children">{{ node.data.currency || '—' }}</span>
                            </template>
                        </Column>

                        <Column>
                            <template #body="{ node }">
                                <div
                                    class="flex gap-2 justify-content-end w-full"
                                    :class="{ 'actions-row--indent': !node.children }"
                                >
                                    <Button
                                        icon="pi pi-plus"
                                        v-if="node.children"
                                        text
                                        rounded
                                        class="p-1"
                                        @click="addAccountToProvider(node)"
                                    />
                                    <Button
                                        icon="pi pi-pencil"
                                        text
                                        rounded
                                        class="p-1"
                                        @click="
                                            node.children ? editProvider(node.data) : editAccount(node.data)
                                        "
                                    />
                                    <Button
                                        icon="pi pi-trash"
                                        text
                                        rounded
                                        severity="danger"
                                        class="p-1"
                                        :disabled="node.children && node.children.length > 0"
                                        @click="
                                            node.children
                                                ? showDeleteProviderDialog(node.data)
                                                : showDeleteAccountDialog(node.data)
                                        "
                                        tooltip="Delete"
                                        tooltipOptions="{ position: 'top' }"
                                    />
                                </div>
                            </template>
                        </Column>
                    </TreeTable>
                </template>
        </Card>
    </div>

    <AccountDialog
        v-if="selectedAccount"
        v-model:visible="accountDialogVisible"
        :is-edit="isEdit"
        :account-id="selectedAccount?.id"
        :provider-id="selectedAccount?.providerId"
        :name="selectedAccount?.name"
        :currency="selectedAccount?.currency"
        :type="selectedAccount?.type"
        :icon="selectedAccount?.icon"
        :import-profile-id="selectedAccount?.importProfileId"
    />

    <DeleteDialog
        v-if="selectedItem && !deleteProviderDialogVisible"
        v-model:visible="deleteAccountDialogVisible"
        :name="selectedItem.name"
        title="Delete Account"
        message="Are you sure you want to delete this account?"
        @confirm="handleDeleteAccount"
    />

    <DeleteDialog
        v-if="selectedItem && !deleteAccountDialogVisible"
        v-model:visible="deleteProviderDialogVisible"
        :name="selectedItem.name"
        title="Delete Account Provider"
        message="Are you sure you want to delete this account provider?"
        @confirm="handleDeleteProvider"
    />

    <AccountProviderDialog
        v-model:visible="providerDialogVisible"
        :is-edit="isEditProvider"
        :provider-id="selectedProvider?.id"
        :name="selectedProvider?.name"
        :description="selectedProvider?.description"
        :icon="selectedProvider?.icon"
    />
</template>
