<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import MultiSelect from 'primevue/multiselect'
import Checkbox from 'primevue/checkbox'
import InputText from 'primevue/inputtext'
import { useCategories } from '@/composables/useCategories'

const categoryIds = defineModel<number[]>('categoryIds', { default: () => [] })
const types = defineModel<string[]>('types', { default: () => [] })
const hasAttachment = defineModel<boolean>('hasAttachment', { default: false })
const search = defineModel<string>('search', { default: '' })
const expanded = defineModel<boolean>('expanded', { default: false })

const hasActiveFilters = computed(() =>
    categoryIds.value.length > 0 ||
    types.value.length > 0 ||
    hasAttachment.value ||
    search.value !== ''
)

defineExpose({ hasActiveFilters })

// Type options
const typeOptions = [
    { label: 'Income', value: 'income' },
    { label: 'Expense', value: 'expense' },
    { label: 'Transfer', value: 'transfer' },
    { label: 'Investment', value: 'investment' },
    { label: 'Balance Status', value: 'balancestatus' }
]

// Category options — flat list
const { incomeCategories, expenseCategories } = useCategories()

const categoryOptions = computed(() => {
    const options: { label: string; value: number }[] = []
    const income = incomeCategories.data.value
    if (income) {
        for (const cat of income) {
            options.push({ label: cat.name, value: Number(cat.id) })
        }
    }
    const expense = expenseCategories.data.value
    if (expense) {
        for (const cat of expense) {
            options.push({ label: cat.name, value: Number(cat.id) })
        }
    }
    return options
})

const searchInput = ref(search.value)

watch(search, (val) => {
    if (val !== searchInput.value) {
        searchInput.value = val
    }
})

function submitSearch() {
    search.value = searchInput.value
}

function clearFilters() {
    categoryIds.value = []
    types.value = []
    hasAttachment.value = false
    searchInput.value = ''
    search.value = ''
}

watch(expanded, (val) => {
    if (!val) clearFilters()
})
</script>

<template>
    <div v-if="expanded" class="filter-panel">
        <div class="filter-row">
            <InputText
                v-model="searchInput"
                placeholder="Search description/notes..."
                class="filter-input-search"
                @keydown.enter="submitSearch"
            />
            <MultiSelect
                v-model="types"
                :options="typeOptions"
                optionLabel="label"
                optionValue="value"
                placeholder="Type"
                class="filter-input"
                :showToggleAll="false"
                scrollHeight="20rem"
            />
            <MultiSelect
                v-model="categoryIds"
                :options="categoryOptions"
                optionLabel="label"
                optionValue="value"
                placeholder="Category"
                class="filter-input"
                filter
            />
            <div class="filter-checkbox">
                <Checkbox v-model="hasAttachment" :binary="true" inputId="hasAttachment" />
                <label for="hasAttachment">Has attachment</label>
            </div>
            <Button
                v-if="hasActiveFilters"
                label="Clear"
                severity="secondary"
                text
                size="small"
                @click="clearFilters"
            />
        </div>
    </div>
</template>

<style scoped>
.filter-panel {
    flex: 1;
}

.filter-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
}

.filter-input {
    min-width: 10rem;
    max-width: 15rem;
}

.filter-input-search {
    min-width: 14rem;
    max-width: 22rem;
    flex: 1;
}

.filter-checkbox {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    white-space: nowrap;
}
</style>
