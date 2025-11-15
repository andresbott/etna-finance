<script setup>
import { ref } from 'vue'
import Select from 'primevue/select'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import StockDialog from './dialogs/StockDialog.vue'
import TransferDialog from '@/views/entries/dialogs/TransferDialog.vue'

/* Internal state for dialog visibility */
const dialogs = ref({
    expense: false,
    income: false,
    transfer: false,
    stock: false
})

/* Selected value for dropdown */
const selectedOption = ref(null)

/* Open the respective dialog when a dropdown item is selected */
const handleSelection = (event) => {
    if (event.value) {
        dialogs.value[event.value] = true
        // Reset selection after opening dialog
        setTimeout(() => {
            selectedOption.value = null
        }, 100)
    }
}

/* Setup dropdown options */
const dropdownOptions = ref([
    {
        label: 'Add Expense',
        value: 'expense',
        icon: 'pi pi-minus'
    },
    {
        label: 'Add Income',
        value: 'income',
        icon: 'pi pi-plus'
    },
    {
        label: 'Add Transfer',
        value: 'transfer',
        icon: 'pi pi-arrow-right-arrow-left'
    },
    {
        label: 'Stock Operation',
        value: 'stock',
        icon: 'pi pi-chart-line'
    }
])

// Define selectedEntry for the stock dialog
const selectedEntry = ref(null)
</script>

<template>
    <div class="add-entry-menu">
        <!-- Add Entry Dropdown -->
        <Select
            v-model="selectedOption"
            :options="dropdownOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Add Entry"
            @change="handleSelection"
            class="add-entry-select"
        >
            <template #value="slotProps">
                <span v-if="slotProps.value">{{ slotProps.value }}</span>
                <span v-else class="placeholder-text">
                    <i class="pi pi-plus" style="margin-right: 0.5rem;"></i>
                    Add Entry
                </span>
            </template>
            <template #option="slotProps">
                <div class="flex align-items-center">
                    <i :class="slotProps.option.icon" style="margin-right: 0.5rem;"></i>
                    <span>{{ slotProps.option.label }}</span>
                </div>
            </template>
        </Select>

        <!-- Expense Dialog -->
        <IncomeExpenseDialog
            v-model:visible="dialogs.expense"
            :isEdit="false"
            entryType="expense"
            @update:visible="dialogs.expense = $event"
        />

        <!-- Income Dialog -->
        <IncomeExpenseDialog
            v-model:visible="dialogs.income"
            :isEdit="false"
            entryType="income"
            @update:visible="dialogs.income = $event"
        />

        <TransferDialog
            v-model:visible="dialogs.transfer"
            :isEdit="false"
            @update:visible="dialogs.transfer = $event"
        />

        <!-- Stock Dialog -->
        <StockDialog
            v-model:visible="dialogs.stock"
            :isEdit="false"
            :entryId="selectedEntry?.id"
            :description="selectedEntry?.description"
            :amount="selectedEntry?.amount"
            :stockAmount="selectedEntry?.stockAmount"
            :date="selectedEntry?.date"
            :type="selectedEntry?.type"
            :targetAccountId="selectedEntry?.targetAccountId"
            :originAccountId="selectedEntry?.originAccountId"
            :categoryId="selectedEntry?.categoryId"
            @update:visible="dialogs.stock = $event"
        />
    </div>
</template>

<style scoped>
.add-entry-menu {
    display: flex;
    justify-content: center;
    align-items: center;
}

.add-entry-select {
    min-width: 180px;
}

.placeholder-text {
    display: flex;
    align-items: center;
    color: var(--primary-color);
    font-weight: 500;
}
</style>
