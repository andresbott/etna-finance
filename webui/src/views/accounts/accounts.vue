<script setup>
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import TopBar from '@/views/topbar.vue'
import { useAccounts } from '@/composable/useAccounts.js'
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
                        <div>
                            <div class="flex justify-content-end mb-3">
                                <Button
                                    label=""
                                    severity="secondary"
                                    variant="text"
                                    icon="pi pi-plus"
                                    @click="openNewAccountDialog"
                                />
                            </div>
                            <DataTable
                                :value="accounts"
                                responsiveLayout="scroll"
                                sortField="id"
                                :sortOrder="1"
                            >
                                <Column field="id" header="ID" :sortable></Column>
                                <Column field="name" header="Name"></Column>
                                <Column field="currency" header="Currency"></Column>
                                <Column field="type" header="Type"></Column>
                                <Column header="Actions">
                                    <template #body="slotProps">
                                        <Button
                                            icon="pi pi-pencil"
                                            class="p-button-rounded p-button-warning p-mr-2"
                                            @click="editAccount(slotProps.data)"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            class="p-button-rounded p-button-danger"
                                            @click="showDeleteDialog(slotProps.data)"
                                        />
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
