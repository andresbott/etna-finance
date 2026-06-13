<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import MultiSelect from 'primevue/multiselect'
import InputText from 'primevue/inputtext'

defineProps<{
    typeOptions: string[]
    exchangeOptions: string[]
}>()

const search = defineModel<string>('search', { default: '' })
const types = defineModel<string[]>('types', { default: () => [] })
const exchanges = defineModel<string[]>('exchanges', { default: () => [] })
const expanded = defineModel<boolean>('expanded', { default: false })

const hasActiveFilters = computed(
    () => search.value !== '' || types.value.length > 0 || exchanges.value.length > 0
)

defineExpose({ hasActiveFilters })

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
    types.value = []
    exchanges.value = []
    searchInput.value = ''
    search.value = ''
}

watch(expanded, (val) => {
    if (!val) clearFilters()
})
</script>

<template>
    <div v-if="expanded" class="filter-panel mb-3">
        <div class="filter-row">
            <InputText
                v-model="searchInput"
                placeholder="Search symbol or name..."
                aria-label="Search instruments by symbol or name"
                class="filter-input-search"
                @keydown.enter="submitSearch"
            />
            <MultiSelect
                v-model="types"
                :options="typeOptions"
                placeholder="Type"
                aria-label="Filter by type"
                scrollHeight="20rem"
                class="filter-input"
                :showToggleAll="false"
            />
            <MultiSelect
                v-model="exchanges"
                :options="exchangeOptions"
                placeholder="Exchange"
                aria-label="Filter by exchange"
                scrollHeight="20rem"
                class="filter-input"
                :showToggleAll="false"
            />
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
</style>
