<script setup>
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import BuySellInstrumentDialog from './dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from './dialogs/GrantDialog.vue'
import TransferInstrumentDialog from './dialogs/TransferInstrumentDialog.vue'
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
import DeleteDialog from '@/components/common/ConfirmDialog.vue'

defineProps({
    selectedEntry: { type: Object, default: null },
    isEditMode: { type: Boolean, default: false },
    isDuplicateMode: { type: Boolean, default: false },
    dialogs: { type: Object, required: true },
    deleteDialogVisible: { type: Boolean, default: false },
    entryToDelete: { type: Object, default: null },
    transformDeleteId: { type: Number, default: null }
})

const emit = defineEmits(['update:deleteDialogVisible', 'confirmDelete', 'transformToTransfer'])
</script>

<template>
    <IncomeExpenseDialog
        v-model:visible="dialogs.incomeExpense.value"
        :is-edit="isEditMode"
        :entry-type="selectedEntry?.type"
        :description="selectedEntry?.description"
        :amount="selectedEntry?.Amount"
        :account-id="selectedEntry?.accountId"
        :stock-amount="selectedEntry?.targetStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :entry-id="selectedEntry?.id"
        :category-id="selectedEntry?.categoryId"
        :autofocus-amount="isDuplicateMode"
        :attachment-id="isDuplicateMode ? undefined : selectedEntry?.attachmentId"
        :notes="selectedEntry?.notes ?? ''"
        @transform-to-transfer="emit('transformToTransfer', $event)"
    />

    <TransferDialog
        v-model:visible="dialogs.transfer.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :target-amount="selectedEntry?.targetAmount"
        :origin-amount="selectedEntry?.originAmount"
        :target-stock-amount="selectedEntry?.targetStockAmount"
        :origin-stock-amount="selectedEntry?.originStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :target-account-id="selectedEntry?.targetAccountId"
        :origin-account-id="selectedEntry?.originAccountId"
        :autofocus-amount="isDuplicateMode"
        :attachment-id="isDuplicateMode ? undefined : selectedEntry?.attachmentId"
        :notes="selectedEntry?.notes ?? ''"
        :delete-after-create-id="transformDeleteId"
    />

    <BuySellInstrumentDialog
        v-model:visible="dialogs.buyStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        operation-type="buy"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :price-per-share="selectedEntry?.StockAmount && selectedEntry?.quantity ? selectedEntry.StockAmount / selectedEntry.quantity : undefined"
        :cash-amount="selectedEntry?.totalAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :investment-account-id="selectedEntry?.investmentAccountId"
        :cash-account-id="selectedEntry?.cashAccountId"
        :notes="selectedEntry?.notes ?? ''"
        @update:visible="dialogs.buyStock.value = $event"
    />
    <BuySellInstrumentDialog
        v-model:visible="dialogs.sellStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        operation-type="sell"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :price-per-share="(selectedEntry?.quantity && (selectedEntry?.costBasis != null || selectedEntry?.StockAmount != null)) ? ((selectedEntry?.costBasis ?? selectedEntry?.StockAmount) / selectedEntry.quantity) : undefined"
        :cash-amount="(selectedEntry?.totalAmount ?? 0) - (selectedEntry?.fees ?? 0)"
        :fees="selectedEntry?.fees ?? 0"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :investment-account-id="selectedEntry?.investmentAccountId"
        :cash-account-id="selectedEntry?.cashAccountId"
        :notes="selectedEntry?.notes ?? ''"
        :initial-lot-allocations="isEditMode ? (selectedEntry?.lotAllocations ?? []) : []"
        @update:visible="dialogs.sellStock.value = $event"
    />

    <GrantDialog
        v-model:visible="dialogs.grantStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :fair-market-value="selectedEntry?.fairMarketValue ?? 0"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :account-id="selectedEntry?.accountId"
        :notes="selectedEntry?.notes ?? ''"
        @update:visible="dialogs.grantStock.value = $event"
    />
    <TransferInstrumentDialog
        v-model:visible="dialogs.transferInstrument.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :origin-account-id="selectedEntry?.originAccountId"
        :target-account-id="selectedEntry?.targetAccountId"
        :notes="selectedEntry?.notes ?? ''"
        @update:visible="dialogs.transferInstrument.value = $event"
    />

    <BalanceStatusDialog
        v-model:visible="dialogs.balanceStatus.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :amount="selectedEntry?.Amount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :account-id="selectedEntry?.accountId"
        :attachment-id="isDuplicateMode ? undefined : selectedEntry?.attachmentId"
        :notes="selectedEntry?.notes ?? ''"
    />

    <DeleteDialog
        :visible="deleteDialogVisible"
        :name="entryToDelete?.description"
        message="Are you sure you want to delete this entry?"
        @update:visible="emit('update:deleteDialogVisible', $event)"
        @confirm="emit('confirmDelete')"
    />
</template>
