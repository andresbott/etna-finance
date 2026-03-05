<script setup>
import { ref, computed } from 'vue'
import Select from 'primevue/select'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import BuySellInstrumentDialog from '@/views/entries/dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from '@/views/entries/dialogs/GrantDialog.vue'
import TransferInstrumentDialog from '@/views/entries/dialogs/TransferInstrumentDialog.vue'
import TransferDialog from '@/views/entries/dialogs/TransferDialog.vue'
import { getAllowedOperations } from '@/types/account'
import { useSettingsStore } from '@/store/settingsStore.js'

const settings = useSettingsStore()

/* Props for pre-populating account fields */
const props = defineProps({
    // Pre-populate account for income/expense
    defaultAccountId: {
        type: Number,
        default: null
    },
    // Pre-populate origin account for transfers
    defaultOriginAccountId: {
        type: Number,
        default: null
    },
    // Account type for filtering allowed operations
    // When null/undefined, all operations are shown (e.g., "All Transactions" view)
    accountType: {
        type: String,
        default: null
    }
})

/* Internal state for dialog visibility */
const dialogs = ref({
    expense: false,
    income: false,
    transfer: false,
    buyStock: false,
    sellStock: false,
    grantStock: false,
    transferInstrument: false
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

/* All available dropdown options */
const allDropdownOptions = [
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
        label: 'Buy instrument',
        value: 'buyStock',
        icon: 'pi pi-arrow-down-left'
    },
    {
        label: 'Sell instrument',
        value: 'sellStock',
        icon: 'pi pi-arrow-up-right'
    },
    {
        label: 'Grant instrument',
        value: 'grantStock',
        icon: 'pi pi-gift'
    },
    {
        label: 'Transfer instrument',
        value: 'transferInstrument',
        icon: 'pi pi-arrow-right-arrow-left'
    }
]

const instrumentOperations = ['buyStock', 'sellStock', 'grantStock', 'transferInstrument']

/* Filtered dropdown options based on account type and instrument settings */
const dropdownOptions = computed(() => {
    const allowedOps = getAllowedOperations(props.accountType)
    return allDropdownOptions.filter(option => {
        if (!allowedOps.includes(option.value)) return false
        if (!settings.instruments && instrumentOperations.includes(option.value)) return false
        return true
    })
})

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
            scrollHeight="22rem"
            @change="handleSelection"
            class="add-entry-select button-style"
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
            :account-id="defaultAccountId"
            @update:visible="dialogs.expense = $event"
        />

        <!-- Income Dialog -->
        <IncomeExpenseDialog
            v-model:visible="dialogs.income"
            :isEdit="false"
            entryType="income"
            :account-id="defaultAccountId"
            @update:visible="dialogs.income = $event"
        />

        <!-- Transfer Dialog -->
        <TransferDialog
            v-model:visible="dialogs.transfer"
            :isEdit="false"
            :origin-account-id="defaultAccountId || defaultOriginAccountId"
            @update:visible="dialogs.transfer = $event"
        />

        <!-- Buy instrument Dialog -->
        <BuySellInstrumentDialog
            v-model:visible="dialogs.buyStock"
            :isEdit="false"
            operation-type="buy"
            :investment-account-id="defaultAccountId"
            @update:visible="dialogs.buyStock = $event"
        />

        <!-- Sell instrument Dialog -->
        <BuySellInstrumentDialog
            v-model:visible="dialogs.sellStock"
            :isEdit="false"
            operation-type="sell"
            :investment-account-id="defaultAccountId"
            @update:visible="dialogs.sellStock = $event"
        />

        <!-- Grant instrument Dialog -->
        <GrantDialog
            v-model:visible="dialogs.grantStock"
            :isEdit="false"
            :account-id="defaultAccountId"
            @update:visible="dialogs.grantStock = $event"
        />

        <!-- Transfer instrument Dialog -->
        <TransferInstrumentDialog
            v-model:visible="dialogs.transferInstrument"
            :isEdit="false"
            :origin-account-id="defaultOriginAccountId ?? defaultAccountId"
            @update:visible="dialogs.transferInstrument = $event"
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
    font-weight: 500;
}

/* Button-style select */
:deep(.button-style) {
    background: var(--c-primary-500);
    border: 1px solid var(--c-primary-600);
    color: var(--c-primary-50);
    font-weight: 500;
    transition: background-color 0.2s, color 0.2s, border-color 0.2s, box-shadow 0.2s;
}

:deep(.button-style:not(.p-disabled):hover) {
    background: var(--c-primary-600);
    border-color: var(--c-primary-700);
}

:deep(.button-style:not(.p-disabled):active) {
    background: var(--c-primary-700);
    border-color: var(--c-primary-800);
}

:deep(.button-style:focus-visible) {
    outline: var(--p-focus-ring-width) var(--p-focus-ring-style) var(--p-focus-ring-color);
    outline-offset: var(--p-focus-ring-offset);
}

:deep(.button-style .p-select-label),
:deep(.button-style .p-select-dropdown),
:deep(.button-style .p-icon) {
    color: var(--c-primary-50);
}
</style>
