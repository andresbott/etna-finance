<script setup>
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import TopBar from '@/views/topbar.vue'
import { useAccounts } from '@/composables/useAccounts.js'
import { useUserStore } from '@/lib/user/userstore.js'
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import AccountDialog from '@/views/accounts/AccountDialog.vue'
import DeleteDialog from '@/components/deleteDialog.vue'

const { accounts, isLoading, deleteAccount } = useAccounts()
const userStore = useUserStore()

const deleteDialogVisible = ref(false)
const accountDialogVisible = ref(false)
const selectedAccount = ref(null)
const isEdit = ref(false)

// userStore.registerLogoutAction(() => {
//     resetAccounts()
// })

const editAccount = (account) => {
    selectedAccount.value = account
    isEdit.value = true
    accountDialogVisible.value = true
}

const handleDeleteAccount = async () => {
    if (selectedAccount.value) {
        await deleteAccount(selectedAccount.value.id)
    }
}

const showDeleteDialog = (account) => {
    selectedAccount.value = account
    deleteDialogVisible.value = true
}

const openNewAccountDialog = () => {
    selectedAccount.value = null
    isEdit.value = false
    accountDialogVisible.value = true
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
                                    label="Add Account"
                                    icon="pi pi-plus"
                                    @click="openNewAccountDialog"
                                />
                            </div>
                            <DataTable
                                :value="accounts"
                                :loading="isLoading"
                                stripedRows
                                class="p-datatable-sm"
                            >
                                <Column field="id" header="ID" sortable />
                                <Column field="name" header="Name" sortable />
                                <Column field="currency" header="Currency" sortable />
                                <Column field="type" header="Type" sortable />
                                <Column header="Actions" style="width: 100px">
                                    <template #body="{ data }">
                                        <div class="actions">
                                            <Button
                                                icon="pi pi-pencil"
                                                text
                                                rounded
                                                class="action-button"
                                                @click="editAccount(data)"
                                            />
                                            <Button
                                                icon="pi pi-trash"
                                                text
                                                rounded
                                                severity="danger"
                                                class="action-button"
                                                @click="showDeleteDialog(data)"
                                            />
                                        </div>
                                    </template>
                                </Column>
                            </DataTable>
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
        :name="selectedAccount?.name"
        :currency="selectedAccount?.currency"
        :type="selectedAccount?.type"
    />

    <DeleteDialog
        v-model:visible="deleteDialogVisible"
        v-if="selectedAccount"
        :name="selectedAccount.name"
        title="Delete Account"
        message="Are you sure you want to delete the account"
        :onConfirm="handleDeleteAccount"
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
    gap: 0.5rem;
    justify-content: flex-start;
}

.action-button {
    padding: 0.25rem;
}
</style>
