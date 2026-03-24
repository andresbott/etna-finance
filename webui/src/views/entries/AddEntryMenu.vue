<script setup>
import { ref, computed } from 'vue'
import Select from 'primevue/select'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import BuySellInstrumentDialog from '@/views/entries/dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from '@/views/entries/dialogs/GrantDialog.vue'
import TransferInstrumentDialog from '@/views/entries/dialogs/TransferInstrumentDialog.vue'
import VestingDialog from '@/views/entries/dialogs/VestingDialog.vue'
import ForfeitDialog from '@/views/entries/dialogs/ForfeitDialog.vue'
import TransferDialog from '@/views/entries/dialogs/TransferDialog.vue'
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
import RevaluationDialog from '@/views/entries/dialogs/RevaluationDialog.vue'
import CsvUploadDialog from '@/views/csvimport/CsvUploadDialog.vue'
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
    },
    // Whether the account has an associated CSV import profile
    hasImportProfile: {
        type: Boolean,
        default: false
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
    transferInstrument: false,
    vestStock: false,
    forfeitStock: false,
    balanceStatus: false,
    revaluation: false,
    importCsv: false
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
        icon: 'ti ti-minus'
    },
    {
        label: 'Add Income',
        value: 'income',
        icon: 'ti ti-plus'
    },
    {
        label: 'Add Transfer',
        value: 'transfer',
        icon: 'ti ti-arrows-left-right'
    },
    {
        label: 'Buy instrument',
        value: 'buyStock',
        icon: 'ti ti-arrow-down-left'
    },
    {
        label: 'Sell instrument',
        value: 'sellStock',
        icon: 'ti ti-arrow-up-right'
    },
    {
        label: 'Grant instrument',
        value: 'grantStock',
        icon: 'ti ti-gift'
    },
    {
        label: 'Transfer instrument',
        value: 'transferInstrument',
        icon: 'ti ti-arrows-left-right'
    },
    {
        label: 'Vest instrument',
        value: 'vestStock',
        icon: 'ti ti-certificate'
    },
    {
        label: 'Forfeit instrument',
        value: 'forfeitStock',
        icon: 'ti ti-circle-x'
    },
    {
        label: 'Balance Status',
        value: 'balanceStatus',
        icon: 'ti ti-calculator'
    },
    {
        label: 'Revaluation',
        value: 'revaluation',
        icon: 'ti ti-adjustments'
    },
    {
        label: 'Import CSV',
        value: 'importCsv',
        icon: 'ti ti-upload'
    }
]

const instrumentOperations = ['buyStock', 'sellStock', 'grantStock', 'transferInstrument', 'vestStock', 'forfeitStock']
const rsuOperations = ['vestStock', 'forfeitStock']

/* Filtered dropdown options based on account type and instrument settings */
const dropdownOptions = computed(() => {
    const allowedOps = getAllowedOperations(props.accountType)
    return allDropdownOptions
        .filter(option => {
            if (!allowedOps.includes(option.value)) return false
            if (!settings.instruments && instrumentOperations.includes(option.value)) return false
            if (!settings.rsu && rsuOperations.includes(option.value)) return false
            if (option.value === 'importCsv' && !props.defaultAccountId) return false
            return true
        })
        .map(option => ({
            ...option,
            disabled: option.value === 'importCsv' && !props.hasImportProfile
        }))
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
            optionDisabled="disabled"
            placeholder="Add Entry"
            scrollHeight="22rem"
            @change="handleSelection"
            class="add-entry-select button-style"
        >
            <template #value="slotProps">
                <span v-if="slotProps.value">{{ slotProps.value }}</span>
                <span v-else class="placeholder-text">
                    <i class="ti ti-plus" style="margin-right: 0.5rem;"></i>
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

        <!-- Balance Status Dialog -->
        <BalanceStatusDialog
            v-model:visible="dialogs.balanceStatus"
            :isEdit="false"
            :account-id="defaultAccountId"
            @update:visible="dialogs.balanceStatus = $event"
        />

        <!-- Revaluation Dialog -->
        <RevaluationDialog
            v-model:visible="dialogs.revaluation"
            :isEdit="false"
            :account-id="defaultAccountId"
            @update:visible="dialogs.revaluation = $event"
        />

        <!-- Transfer instrument Dialog -->
        <TransferInstrumentDialog
            v-model:visible="dialogs.transferInstrument"
            :isEdit="false"
            :origin-account-id="defaultOriginAccountId ?? defaultAccountId"
            @update:visible="dialogs.transferInstrument = $event"
        />

        <!-- Vest instrument Dialog -->
        <VestingDialog
            v-model:visible="dialogs.vestStock"
            :isEdit="false"
            :origin-account-id="defaultAccountId"
            @update:visible="dialogs.vestStock = $event"
        />

        <!-- Forfeit instrument Dialog -->
        <ForfeitDialog
            v-model:visible="dialogs.forfeitStock"
            :isEdit="false"
            :account-id="defaultAccountId"
            @update:visible="dialogs.forfeitStock = $event"
        />

        <!-- CSV Upload Dialog -->
        <CsvUploadDialog
            v-model:visible="dialogs.importCsv"
            :account-id="defaultAccountId"
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
