<script setup>
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'

import DateRangePicker from '@/components/common/DateRangePicker.vue'
import EntryFilters from '@/components/entries/EntryFilters.vue'
import EntriesTable from './EntriesTable.vue'
import EntryDialogs from './EntryDialogs.vue'

import { useEntries } from '@/composables/useEntries.ts'
import { useEntryDialogs } from '@/composables/useEntryDialogs'
import { useRouteState } from '@/composables/useRouteState'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'

const props = defineProps({
    /** When true, show only financial types: transfer, stockbuy, stocksell, stockgrant, stocktransfer (excludes income/expense). */
    financialOnly: { type: Boolean, default: false }
})

const FINANCIAL_ENTRY_TYPES = ['transfer', 'stockbuy', 'stocksell', 'stockgrant', 'stocktransfer']

/* --- Reactive State (synced with URL query params) --- */
const today = new Date()
const { startDate, endDate, page, limit, categoryIds, types, hasAttachment, search } = useRouteState({
    startDate: new Date(today.getFullYear(), today.getMonth(), today.getDate() - 35),
    endDate: new Date(),
    limit: props.financialOnly ? 500 : 25
})
const first = computed(() => (page.value - 1) * limit.value)

const { entries, totalRecords, isLoading, isFetching, deleteEntry, isDeleting, refetch } = useEntries({
    startDate,
    endDate,
    page,
    limit,
    categoryIds,
    types,
    hasAttachment,
    search
})

const filteredEntries = computed(() => {
    const list = entries.value ?? []
    if (!props.financialOnly) return list
    return list.filter((e) => FINANCIAL_ENTRY_TYPES.includes(e.type))
})

const displayedEntries = computed(() => {
    if (!props.financialOnly) return filteredEntries.value
    const start = first.value
    const rows = limit.value
    return filteredEntries.value.slice(start, start + rows)
})

/* --- Computed pagination values for template --- */
const paginationRows = computed(() => limit.value)
const paginationFirst = computed(() => first.value)
const paginationTotal = computed(() =>
    props.financialOnly ? filteredEntries.value.length : (totalRecords.value || 0)
)

/* --- Pagination Handler --- */
const handlePage = (event) => {
    limit.value = event.rows
    page.value = event.page + 1 // PrimeVue uses 0-based page, we use 1-based
}

/* --- Reset pagination when date range changes --- */
watch([startDate, endDate, categoryIds, types, hasAttachment, search], () => {
    page.value = 1
})

watch(
    () => props.financialOnly,
    (financialOnly) => {
        limit.value = financialOnly ? 500 : 25
        page.value = 1
    },
    { immediate: true }
)

const {
    selectedEntry, isEditMode, isDuplicateMode, dialogs,
    deleteDialogVisible, entryToDelete, deleteError,
    openEditEntryDialog, openDuplicateEntryDialog, openDeleteDialog, handleDeleteEntry,
    openTransformToTransfer, transformDeleteId
} = useEntryDialogs(deleteEntry)

const filtersExpanded = ref(
    categoryIds.value.length > 0 ||
    types.value.length > 0 ||
    hasAttachment.value ||
    search.value !== ''
)
</script>

<template>
    <div class="main-app-content">
        <div class="entries-content">
            <div class="toolbar">
                <div class="date-filters">
                    <DateRangePicker
                        v-model:startDate="startDate"
                        v-model:endDate="endDate"
                        @change="refetch"
                    />
                    <Button
                        :icon="filtersExpanded ? 'ti ti-filter-off' : 'ti ti-filter'"
                        :severity="filtersExpanded ? 'primary' : 'secondary'"
                        :outlined="!filtersExpanded"
                        class="filter-btn"
                        @click="filtersExpanded = !filtersExpanded"
                        v-tooltip.bottom="'Filters'"
                    />
                </div>
                <div class="add-entry-menu">
                    <AddEntryMenu />
                </div>
            </div>

            <div class="filter-bar">
                <EntryFilters
                    v-model:categoryIds="categoryIds"
                    v-model:types="types"
                    v-model:hasAttachment="hasAttachment"
                    v-model:search="search"
                    v-model:expanded="filtersExpanded"
                />
            </div>

            <div class="entries-view">
                <EntriesTable
                    :entries="displayedEntries"
                    :isLoading="isLoading || isFetching"
                    :isDeleting="isDeleting"
                    :totalRecords="paginationTotal"
                    :rows="paginationRows"
                    :first="paginationFirst"
                    :financial-columns="financialOnly"
                    @edit="openEditEntryDialog"
                    @duplicate="openDuplicateEntryDialog"
                    @delete="openDeleteDialog"
                    @page="handlePage"
                />
            </div>
        </div>
    </div>

    <EntryDialogs
        :selected-entry="selectedEntry"
        :is-edit-mode="isEditMode"
        :is-duplicate-mode="isDuplicateMode"
        :dialogs="dialogs"
        :delete-dialog-visible="deleteDialogVisible"
        :entry-to-delete="entryToDelete"
        :delete-error="deleteError"
        :transform-delete-id="transformDeleteId"
        @update:delete-dialog-visible="deleteDialogVisible = $event"
        @confirm-delete="handleDeleteEntry"
        @transform-to-transfer="openTransformToTransfer"
    />
</template>

<style scoped>
.main-app-content {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.entries-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
}

.toolbar {
    display: flex;
    align-items: center;
    padding: 1rem;
    gap: 1rem;
    background-color: var(--surface-ground);
    border-bottom: 1px solid var(--surface-border);
}

.date-filters {
    display: flex;
    gap: 1rem;
    align-items: center;
}

.add-entry-menu {
    display: flex;
    align-items: center;
    margin-left: auto;
}

.filter-bar {
    padding: 0 1rem;
    background-color: var(--surface-ground);
    border-bottom: 1px solid var(--surface-border);
}

.entries-view {
    flex: 1;
    overflow: auto;
    padding: 1rem;
}

.filter-btn {
    align-self: stretch;
}
</style>
