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
import {
    useMarketInstruments,
    useMarketDataMutations,
    toLocalDateString,
    formatPrice,
    formatPct,
    getChangeSeverity
} from '@/composables/useMarketData'
const { formatDate, pickerDateFormat } = useDateFormat()

const router = useRouter()
const { instruments, isLoading, isError, error, refetch } = useMarketInstruments()
const leftSidebarCollapsed = ref(true)

const addDialogInstrument = ref(null)
const addDialogVisible = ref(false)
const addDialogForm = ref({ date: '', price: 0 })
const addDialogSymbol = computed(() => addDialogInstrument.value?.symbol ?? '')
const { createPrice: createPriceMutation, isCreating: isCreatingPrice } = useMarketDataMutations(addDialogSymbol)

function openAddDialog(inst) {
    addDialogInstrument.value = inst
    addDialogForm.value = { date: toLocalDateString(new Date()), price: 0 }
    addDialogVisible.value = true
}

async function saveAddDialog() {
    if (!addDialogInstrument.value) return
    const { date, price } = addDialogForm.value
    if (!date) return
    const time = date.includes('T') ? toLocalDateString(new Date(date)) : date
    try {
        await createPriceMutation({ time, price })
        addDialogVisible.value = false
        refetch()
    } catch (_) {}
}

const onRowClick = (event) => {
    router.push({ name: 'stock-detail', params: { id: event.data.id, tab: 'overview' } })
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
                            Stock Market
                        </h1>
                        <p class="text-color-secondary mt-0 mb-3">
                            Investment instruments overview with market data
                            <i
                                class="ti ti-help-circle"
                                v-tooltip.bottom="'Use Tasks to update and schedule market data ingestion'"
                                style="cursor: help"
                            />
                        </p>
                    </div>
                </div>

                <Message v-if="isError" severity="error" :closable="false" class="mb-3">
                    <div class="flex align-items-center gap-2 flex-wrap">
                        <i class="ti ti-alert-triangle"></i>
                        <span>{{ error?.message ?? 'Failed to load market data.' }}</span>
                        <Button label="Retry" icon="ti ti-refresh" text size="small" @click="refetch" />
                    </div>
                </Message>

                <Card v-if="!instruments.length && !isLoading">
                    <template #content>
                        <div class="empty-message">
                            No instruments configured. Add instruments in
                            <router-link to="/instruments">Investment Products</router-link>
                            to see market data here.
                        </div>
                    </template>
                </Card>

                <Card v-else>
                    <template #content>
                        <DataTable
                            :value="instruments"
                            :loading="isLoading"
                            dataKey="id"
                            stripedRows
                            :paginator="instruments.length > 15"
                            :rows="15"
                            class="p-datatable-sm clickable-rows stock-table"
                            selectionMode="single"
                            @rowClick="onRowClick"
                        >
                            <Column field="symbol" header="Symbol">
                                <template #body="{ data }">
                                    <span class="font-bold">{{ data.symbol }}</span>
                                </template>
                            </Column>
                            <Column field="name" header="Name" />
                            <Column field="lastPrice" header="Price">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ formatPrice(data.lastPrice) }}</span>
                                    <span class="text-color-secondary text-sm ml-1">{{ data.currency }}</span>
                                </template>
                            </Column>
                            <Column field="changePct" header="Change">
                                <template #body="{ data }">
                                    <Tag
                                        :value="formatPct(data.changePct)"
                                        :severity="getChangeSeverity(data.changePct)"
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
                                            v-tooltip.bottom="'Add price'"
                                            :loading="isCreatingPrice && addDialogInstrument?.id === data.id"
                                            @click.stop="openAddDialog(data)"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                        <p class="text-color-secondary text-sm mt-2 mb-0">
                            Click a row to open details, chart and edit data. Use + to add a price.
                        </p>
                    </template>
                </Card>

                <Dialog
                    v-model:visible="addDialogVisible"
                    header="Add market data"
                    modal
                    class="entry-dialog"
                    @hide="addDialogVisible = false"
                >
                    <div v-if="addDialogInstrument" class="flex flex-column gap-3 py-2">
                        <p class="text-color-secondary mt-0 mb-0">
                            {{ addDialogInstrument.symbol }} – {{ addDialogInstrument.name }}
                        </p>
                        <div class="field">
                            <label for="add-data-date">Date</label>
                            <DatePicker
                                id="add-data-date"
                                :modelValue="addDialogForm.date ? new Date(addDialogForm.date + 'T12:00:00') : null"
                                @update:modelValue="(d) => { addDialogForm.date = d ? toLocalDateString(d) : '' }"
                                :dateFormat="pickerDateFormat"
                                showIcon
                                class="w-full"
                            />
                        </div>
                        <div class="field">
                            <label for="add-data-price">Price</label>
                            <InputNumber
                                id="add-data-price"
                                v-model="addDialogForm.price"
                                mode="decimal"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                                :min="0"
                                class="w-full"
                            />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Save" icon="ti ti-check" :loading="isCreatingPrice" @click="saveAddDialog" />
                        <Button label="Cancel" text severity="secondary" @click="addDialogVisible = false" />
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

:deep(.stock-table .p-datatable-thead th.last-update-cell .p-datatable-column-header-content) {
    justify-content: flex-end;
}

.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
}
</style>
