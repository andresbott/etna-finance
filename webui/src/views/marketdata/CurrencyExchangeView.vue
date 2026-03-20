<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import InputNumber from 'primevue/inputnumber'
import { useDateFormat } from '@/composables/useDateFormat'
import { toLocalDateString } from '@/composables/useMarketData'
import {
    useFXOverview,
    useFXMutations,
    formatPct,
    getChangeSeverity
} from '@/composables/useCurrencyRates'
const { formatDate, pickerDateFormat } = useDateFormat()
const router = useRouter()
const { mainCurrency, currencyRows, isLoading, isError, error, refetch } = useFXOverview()
const leftSidebarCollapsed = ref(true)

const addDialogCurrency = ref('')
const addDialogVisible = ref(false)
const addDialogForm = ref({ date: '', rate: 0 })
const { createRate: createRateMutation, isCreating } = useFXMutations(mainCurrency, computed(() => addDialogCurrency.value || ''))

function openAddDialog(row) {
    addDialogCurrency.value = row.currency
    addDialogForm.value = { date: toLocalDateString(new Date()), rate: 0 }
    addDialogVisible.value = true
}

async function saveAddDialog() {
    if (!addDialogCurrency.value) return
    const { date, rate } = addDialogForm.value
    if (!date) return
    const time = date.includes('T') ? toLocalDateString(new Date(date)) : date
    try {
        await createRateMutation({ time, rate })
        addDialogVisible.value = false
        refetch()
    } catch (_) {}
}

function onRowClick(event) {
    router.push({ name: 'currency-detail', params: { currency: event.data.currency, tab: 'overview' } })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="mb-2 flex align-items-start justify-content-between gap-2">
                    <div>
                        <h1 class="flex align-items-center gap-3 m-0 mb-2">
                            <i class="ti ti-chart-line text-primary"></i>
                            Currency Exchange
                        </h1>
                        <p class="text-color-secondary mt-0 mb-3">
                            Exchange rates and trends
                            <i
                                class="ti ti-help-circle"
                                v-tooltip.bottom="'Use Tasks to update and schedule currency rate ingestion'"
                                style="cursor: help"
                            />
                        </p>
                    </div>
                </div>

                <Message v-if="isError" severity="error" :closable="false" class="mb-3">
                    <div class="flex align-items-center gap-2 flex-wrap">
                        <i class="ti ti-alert-triangle"></i>
                        <span>{{ error?.message ?? 'Failed to load currency data.' }}</span>
                        <Button label="Retry" icon="ti ti-refresh" text size="small" @click="refetch" />
                    </div>
                </Message>

                <Card v-if="!currencyRows.length && !isLoading">
                    <template #content>
                        <div class="empty-message">
                            No currencies configured. Set main currency and other currencies in
                            <router-link to="/settings">Settings</router-link>
                            to see exchange rates here.
                        </div>
                    </template>
                </Card>

                <Card v-else>
                    <template #content>
                        <DataTable
                            :value="currencyRows"
                            :loading="isLoading"
                            dataKey="currency"
                            stripedRows
                            :paginator="currencyRows.length > 15"
                            :rows="15"
                            class="p-datatable-sm clickable-rows currency-table"
                            selectionMode="single"
                            @rowClick="onRowClick"
                        >
                            <Column field="currency" header="Currency">
                                <template #body="{ data }">
                                    <span class="font-bold">{{ data.currency }}</span>
                                </template>
                            </Column>
                            <Column field="pair" header="Pair" />
                            <Column :header="`1 ${mainCurrency} =`" class="rate-direction-column">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ data.rate != null && data.rate !== 0 ? `${data.rate.toFixed(4)} ${data.currency}` : '-' }}</span>
                                </template>
                            </Column>
                            <Column :header="`1 → ${mainCurrency}`" class="rate-direction-column">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ data.rate != null && data.rate !== 0 ? `${(1 / data.rate).toFixed(4)} ${mainCurrency}` : '-' }}</span>
                                </template>
                            </Column>
                            <Column field="change" header="Change">
                                <template #body="{ data }">
                                    <Tag
                                        :value="formatPct(data.change)"
                                        :severity="getChangeSeverity(data.change)"
                                    />
                                </template>
                            </Column>
                            <Column field="lastUpdate" header="Last update" bodyClass="last-update-cell" headerClass="last-update-cell">
                                <template #body="{ data }">
                                    {{ data.lastUpdate ? formatDate(data.lastUpdate) : '-' }}
                                </template>
                            </Column>
                            <Column header="Actions" class="actions-column" style="width: 4rem; min-width: 4rem">
                                <template #body="{ data }">
                                    <div class="flex gap-2 justify-content-end">
                                        <Button
                                            icon="ti ti-plus"
                                            text
                                            rounded
                                            class="p-1"
                                            v-tooltip.bottom="'Add rate'"
                                            :loading="isCreating && addDialogCurrency === data.currency"
                                            @click.stop="openAddDialog(data)"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                        <p class="text-color-secondary text-sm mt-2 mb-0">
                            Click a row to open details. Use + to add a rate.
                        </p>
                    </template>
                </Card>

                <Dialog
                    v-model:visible="addDialogVisible"
                    header="Add exchange rate"
                    modal
                    class="entry-dialog"
                    @hide="addDialogVisible = false"
                >
                    <div v-if="addDialogCurrency" class="flex flex-column gap-3 py-2">
                        <p class="text-color-secondary mt-0 mb-0">
                            {{ mainCurrency }}/{{ addDialogCurrency }}
                        </p>
                        <div class="field">
                            <label for="add-fx-date">Date</label>
                            <DatePicker
                                id="add-fx-date"
                                :modelValue="addDialogForm.date ? new Date(addDialogForm.date + 'T12:00:00') : null"
                                @update:modelValue="(d) => { addDialogForm.date = d ? toLocalDateString(d) : '' }"
                                :dateFormat="pickerDateFormat"
                                showIcon
                                class="w-full"
                            />
                        </div>
                        <div class="field">
                            <label for="add-fx-rate">Rate</label>
                            <InputNumber
                                id="add-fx-rate"
                                v-model="addDialogForm.rate"
                                mode="decimal"
                                :minFractionDigits="4"
                                :maxFractionDigits="6"
                                :min="0"
                                class="w-full"
                            />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Cancel" text severity="secondary" @click="addDialogVisible = false" />
                        <Button label="Save" icon="ti ti-check" :loading="isCreating" @click="saveAddDialog" />
                    </template>
                </Dialog>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.empty-message {
    padding: 1rem;
    text-align: center;
    color: var(--p-text-muted-color);
}

.empty-message a {
    color: var(--p-primary-color);
    text-decoration: none;
    font-weight: 600;
}

.empty-message a:hover {
    text-decoration: underline;
}

:deep(.clickable-rows .p-datatable-tbody > tr) {
    cursor: pointer;
}

:deep(.last-update-cell) {
    text-align: right;
}

:deep(.currency-table .p-datatable-thead th.last-update-cell .p-datatable-column-header-content) {
    justify-content: flex-end;
}

.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
}
</style>
