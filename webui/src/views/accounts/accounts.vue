<script setup>
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import TopBar from '@/views/topbar.vue'
import { useAccounts } from '@/composables/useAccounts.js'
import { useUserStore } from '@/lib/user/userstore.js'
import {  ref, computed } from 'vue'

import Column from 'primevue/column'
import Button from 'primevue/button'
import AccountDialog from '@/views/accounts/AccountDialog.vue'
import DeleteDialog from '@/components/deleteDialog.vue'
import TreeTable from 'primevue/treetable'
import AccountProviderDialog from './AccountProviderDialog.vue'

const { accounts, isLoading, deleteAccount, deleteAccountProvider } = useAccounts()
const userStore = useUserStore()

// Modify the computed property to also set expanded keys
const treeTableData = computed(() => {
  if (!accounts.value) return []

  const data = accounts.value.map(provider => ({
    key: provider.id,
    data: {
      id: provider.id,
      name: provider.name,
      description: provider.description
    },
    children: provider.accounts?.map(account => ({
      key: account.id,
      data: {
        id: account.id,
        name: account.name,
        type: account.type,
        currency: account.currency
      }
    })) || []
  }))

  // Set all keys as expanded
  expandedKeys.value = data.reduce((acc, node) => {
    acc[node.key] = true
    return acc
  }, {})

  return data
})


const deleteDialogVisible = ref(false)
const accountDialogVisible = ref(false)
const selectedAccount = ref(null)
const isEdit = ref(false)
const providerDialogVisible = ref(false)
const selectedProvider = ref(null)
const isEditProvider = ref(false)
const deleteProviderDialogVisible = ref(false)
const deleteAccountDialogVisible = ref(false)
const selectedItem = ref(null)

// userStore.registerLogoutAction(() => {
//     resetAccounts()
// })

// Add expandedKeys ref and computed property
const expandedKeys = ref({})

const editAccount = (account) => {
    selectedAccount.value = account
    isEdit.value = true
    accountDialogVisible.value = true
}

// Add function to handle adding account to provider
const addAccountToProvider = (provider) => {
    selectedAccount.value = {
        providerId: provider.data.id  // Set the provider ID for the new account
    }
    isEdit.value = false
    accountDialogVisible.value = true
}

// Update openNewAccountDialog to handle provider creation
const openNewProviderDialog = () => {
    selectedProvider.value = null
    isEditProvider.value = false
    providerDialogVisible.value = true
}

// Add edit provider function
const editProvider = (provider) => {
    selectedProvider.value = provider
    isEditProvider.value = true
    providerDialogVisible.value = true
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

const showDeleteAccountDialog = (account) => {
    selectedItem.value = account
    deleteAccountDialogVisible.value = true
}

const showDeleteProviderDialog = (provider) => {
    selectedItem.value = provider
    deleteProviderDialogVisible.value = true
}

</script>
<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <TopBar />
        </template>
        <template #default>
            <HorizontalLayout
                :fullHeight="true"
                :centerContent="true"
                :verticalCenterContent="false"
            >
                <template #default>
                    <Placeholder :width="'960px'" :height="'auto'">
                        <div class="accounts-view">
                            <div class="header">
                                <h1>Accounts</h1>
                                <Button
                                    label="Add Account Provider"
                                    icon="pi pi-plus"
                                    @click="openNewProviderDialog"
                                />
                            </div>
                            <TreeTable 
                                :value="treeTableData" 
                                :loading="isLoading" 
                                :expandedKeys="expandedKeys"
                                class="p-treetable-sm"
                            >
                                <Column field="name" expander />

                                <Column field="description"  >
                                  <template #body="{ node }">

                                    <div  v-if="node.children" >
                                    <span>{{ node.data.description }}</span>
                                    </div>
                                    <div v-else>
                                      <i>{{ node.data.type }}</i>
                                    </div>
                                  </template>
                                </Column>
                              <Column field="currency"  />
                                <Column  >
                                    <template #body="{ node }">
                                      <div  v-if="node.children" class="actions">
                                        <Button
                                                      icon="pi pi-plus"
                                            text
                                            rounded
                                            class="action-button"
                                            @click="addAccountToProvider(node)"
                                        />
                                        <Button
                                            icon="pi pi-pencil"
                                            text
                                            rounded
                                            class="action-button"
                                            @click="editProvider(node.data)"
                                        />
                                        <Button
                                                                     icon="pi pi-trash"
                                            text
                                            rounded
                                            severity="danger"
                                            class="action-button"
                                            @click="showDeleteProviderDialog(node.data)"
                                            :disabled="node.children.length > 0"
                                            tooltip="Delete Account Provider"
                                            tooltipOptions="{ position: 'top' }"
                                        />

                                      </div>
                                      <div v-else class="actions" style="margin-left:38px">
                                        <Button
                                            icon="pi pi-pencil"
                                            text
                                            rounded
                                            class="action-button"
                                            @click="editAccount(node.data)"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            text
                                            rounded
                                            severity="danger"
                                            class="action-button"
                                            @click="showDeleteAccountDialog(node.data)"
                                            tooltip="Delete Account"
                                            tooltipOptions="{ position: 'top' }"
                                        />
                                      </div>

                                    </template>
                                </Column>
                            </TreeTable>
                        </div>
                    </Placeholder>
                </template>
            </HorizontalLayout>
        </template>
        <template #footer>
            <Placeholder :width="'100%'" :height="30" :color="12">Footer</Placeholder>
        </template>
    </VerticalLayout>

    <AccountDialog
        v-model:visible="accountDialogVisible"
        :is-edit="isEdit"
        :account-id="selectedAccount?.id"
        :provider-id="selectedAccount?.providerId"
        :name="selectedAccount?.name"
        :currency="selectedAccount?.currency"
        :type="selectedAccount?.type"
    />

    <DeleteDialog
        v-model:visible="deleteAccountDialogVisible"
        v-if="selectedItem && !deleteProviderDialogVisible"
        :name="selectedItem.name"
        title="Delete Account"
        message="Are you sure you want to delete this account?"
        :onConfirm="handleDeleteAccount"
    />

    <DeleteDialog
        v-model:visible="deleteProviderDialogVisible"
        v-if="selectedItem && !deleteAccountDialogVisible"
        :name="selectedItem.name"
        title="Delete Account Provider"
        message="Are you sure you want to delete this account provider?"
        :onConfirm="handleDeleteProvider"
    />

    <AccountProviderDialog
        v-model:visible="providerDialogVisible"
        :is-edit="isEditProvider"
        :provider-id="selectedProvider?.id"
        :name="selectedProvider?.name"
        :description="selectedProvider?.description"
    />
</template>

<style scoped>
.accounts-view {
    padding: 2rem;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
}

.actions {
    display: flex;

    justify-content: flex-start;
}

.action-button {
    padding: 0.25rem;
}
</style>
