<script setup>
import { ref, watch, computed } from 'vue'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'

const props = defineProps({
    startDate: {
        type: Date,
        required: true
    },
    endDate: {
        type: Date,
        required: true
    },
    startLabel: {
        type: String,
        default: 'From:'
    },
    endLabel: {
        type: String,
        default: 'To:'
    },
    dateFormat: {
        type: String,
        default: 'dd/mm/y'
    },
    startPlaceholder: {
        type: String,
        default: 'Start date'
    },
    endPlaceholder: {
        type: String,
        default: 'End date'
    },
    showIcon: {
        type: Boolean,
        default: true
    },
    showButtonBar: {
        type: Boolean,
        default: true
    }
})

const emit = defineEmits(['update:startDate', 'update:endDate', 'change'])

const localStartDate = ref(props.startDate)
const localEndDate = ref(props.endDate)

watch(() => props.startDate, (newVal) => {
    localStartDate.value = newVal
})

watch(() => props.endDate, (newVal) => {
    localEndDate.value = newVal
})

watch(localStartDate, (newVal) => {
    emit('update:startDate', newVal)
    emit('change', { startDate: newVal, endDate: localEndDate.value })
})

watch(localEndDate, (newVal) => {
    emit('update:endDate', newVal)
    emit('change', { startDate: localStartDate.value, endDate: newVal })
})

const setCurrentYear = () => {
    const currentYear = new Date().getFullYear()
    localStartDate.value = new Date(currentYear, 0, 1) // January 1st
    localEndDate.value = new Date(currentYear, 11, 31) // December 31st
}

const setCurrentMonth = () => {
    const now = new Date()
    const currentYear = now.getFullYear()
    const currentMonth = now.getMonth()
    localStartDate.value = new Date(currentYear, currentMonth, 1) // First day of current month
    localEndDate.value = new Date(currentYear, currentMonth + 1, 0) // Last day of current month
}

const setPreviousMonth = () => {
    const now = new Date()
    const currentYear = now.getFullYear()
    const currentMonth = now.getMonth()
    // Get the first day of previous month
    localStartDate.value = new Date(currentYear, currentMonth - 1, 1)
    // Get the last day of previous month (day 0 of current month)
    localEndDate.value = new Date(currentYear, currentMonth, 0)
}

const setPreviousYear = () => {
    const previousYear = new Date().getFullYear() - 1
    localStartDate.value = new Date(previousYear, 0, 1) // January 1st
    localEndDate.value = new Date(previousYear, 11, 31) // December 31st
}

const now = new Date()
const currentYear = now.getFullYear()
const currentMonthName = now.toLocaleString('en-US', { month: 'long' })

const previousMonthDate = new Date(now.getFullYear(), now.getMonth() - 1, 1)
const previousMonthName = previousMonthDate.toLocaleString('en-US', { month: 'long' })

const quickSelectOptions = ref([
    { label: `Previous Month (${previousMonthName})`, value: 'previous-month' },
    { label: `Current Month (${currentMonthName})`, value: 'current-month' },
    { label: `Previous Year (${currentYear - 1})`, value: 'previous-year' },
    { label: `Current Year (${currentYear})`, value: 'current-year' }
])

const selectedQuickOption = ref(null)

watch(selectedQuickOption, (newValue) => {
    if (!newValue) return
    
    if (newValue === 'current-year') {
        setCurrentYear()
    } else if (newValue === 'current-month') {
        setCurrentMonth()
    } else if (newValue === 'previous-month') {
        setPreviousMonth()
    } else if (newValue === 'previous-year') {
        setPreviousYear()
    }
    
    // Reset selection after applying
    setTimeout(() => {
        selectedQuickOption.value = null
    }, 100)
})
</script>

<template>
    <div class="date-range-picker">
        <div class="date-field">
            <label>{{ startLabel }}</label>
            <DatePicker
                v-model="localStartDate"
                :showIcon="showIcon"
                :showButtonBar="showButtonBar"
                :dateFormat="dateFormat"
                :placeholder="startPlaceholder"
            />
        </div>
        <div class="date-field">
            <label>{{ endLabel }}</label>
            <DatePicker
                v-model="localEndDate"
                :showIcon="showIcon"
                :showButtonBar="showButtonBar"
                :dateFormat="dateFormat"
                :placeholder="endPlaceholder"
            />
        </div>
        <Select
            v-model="selectedQuickOption"
            :options="quickSelectOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Quick Select"
            class="quick-select"
        />
    </div>
</template>

<style scoped>
.date-range-picker {
    display: flex;
    gap: 1rem;
    align-items: center;
    justify-content: center;
}

.date-field {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 0.5rem;
}

.date-field label {
    font-weight: 500;
    white-space: nowrap;
}

.quick-select {
    min-width: 150px;
}
</style>

