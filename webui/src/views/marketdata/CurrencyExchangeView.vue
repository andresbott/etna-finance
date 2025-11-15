<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import Chart from 'primevue/chart'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Message from 'primevue/message'

const leftSidebarCollapsed = ref(true)

// Date range
const startDate = ref(new Date(new Date().setMonth(new Date().getMonth() - 3)))
const endDate = ref(new Date())

// Mock exchange rate data
const generateMockData = () => {
    const data = []
    const currencies = ['USD', 'EUR', 'GBP', 'JPY']
    const baseRates = { USD: 0.92, EUR: 1.08, GBP: 1.26, JPY: 0.0061 }
    
    const start = new Date(startDate.value)
    const end = new Date(endDate.value)
    
    for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
        currencies.forEach(currency => {
            const variance = (Math.random() - 0.5) * 0.02
            const rate = baseRates[currency] * (1 + variance)
            data.push({
                date: new Date(d).toISOString().split('T')[0],
                currency,
                rate: rate.toFixed(4),
                change: (variance * 100).toFixed(2)
            })
        })
    }
    
    return data
}

const mockData = ref(generateMockData())

// Chart data
const chartData = computed(() => {
    const currencies = ['USD', 'EUR', 'GBP', 'JPY']
    const colors = {
        USD: '#22c55e',
        EUR: '#3b82f6',
        GBP: '#eab308',
        JPY: '#ef4444'
    }
    
    const datasets = currencies.map(currency => {
        const currencyData = mockData.value
            .filter(d => d.currency === currency)
            .sort((a, b) => new Date(a.date) - new Date(b.date))
        
        return {
            label: `CHF/${currency}`,
            data: currencyData.map(d => parseFloat(d.rate)),
            borderColor: colors[currency],
            backgroundColor: colors[currency] + '20',
            tension: 0.4,
            fill: false
        }
    })
    
    const labels = [...new Set(mockData.value.map(d => d.date))].sort()
    
    return { labels, datasets }
})

const chartOptions = computed(() => ({
    maintainAspectRatio: false,
    plugins: {
        legend: {
            labels: {
                color: getTextColor()
            }
        },
        tooltip: {
            callbacks: {
                label: function(context) {
                    return `${context.dataset.label}: ${context.parsed.y.toFixed(4)}`
                }
            }
        }
    },
    scales: {
        x: {
            ticks: {
                color: getTextColor(),
                maxRotation: 45,
                minRotation: 45
            },
            grid: {
                color: getSurfaceColor()
            }
        },
        y: {
            ticks: {
                color: getTextColor()
            },
            grid: {
                color: getSurfaceColor()
            }
        }
    }
}))

const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--text-color') || '#495057'
}

const getSurfaceColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--surface-border') || '#dfe7ef'
}

// Data by currency for tables
const usdData = computed(() => 
    mockData.value.filter(d => d.currency === 'USD').sort((a, b) => new Date(b.date) - new Date(a.date))
)

const eurData = computed(() => 
    mockData.value.filter(d => d.currency === 'EUR').sort((a, b) => new Date(b.date) - new Date(a.date))
)

const gbpData = computed(() => 
    mockData.value.filter(d => d.currency === 'GBP').sort((a, b) => new Date(b.date) - new Date(a.date))
)

const jpyData = computed(() => 
    mockData.value.filter(d => d.currency === 'JPY').sort((a, b) => new Date(b.date) - new Date(a.date))
)

const formatChange = (value) => {
    const num = parseFloat(value)
    return num >= 0 ? `+${value}%` : `${value}%`
}

const getChangeClass = (value) => {
    return parseFloat(value) >= 0 ? 'text-green-600' : 'text-red-600'
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <!-- Mock UI Warning -->
                <Message severity="error" :closable="false" class="warning-message">
                    <div class="warning-content">
                        <i class="pi pi-exclamation-triangle"></i>
                        <strong>This is a mock UI only.</strong> Exchange rate data is simulated and not real market data.
                    </div>
                </Message>

                <!-- Date Range Picker -->
                <div class="mb-4 flex justify-content-center">
                    <DateRangePicker
                        v-model:startDate="startDate"
                        v-model:endDate="endDate"
                        startLabel="From:"
                        endLabel="To:"
                    />
                </div>

                <div class="grid">
                    <!-- Exchange Rate Chart -->
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-chart-line"></i>
                                    <span>Exchange Rate Trends</span>
                                </div>
                            </template>
                            <template #content>
                                <Chart
                                    type="line"
                                    :data="chartData"
                                    :options="chartOptions"
                                    style="height: 400px"
                                />
                            </template>
                        </Card>
                    </div>

                    <!-- Exchange Rate Tables -->
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-table"></i>
                                    <span>Exchange Rate Details</span>
                                </div>
                            </template>
                            <template #content>
                                <TabView>
                                    <TabPanel header="USD">
                                        <DataTable :value="usdData" stripedRows :paginator="true" :rows="10">
                                            <Column field="date" header="Date" sortable style="min-width: 150px"></Column>
                                            <Column field="rate" header="CHF/USD Rate" sortable style="min-width: 150px"></Column>
                                            <Column field="change" header="Change %" sortable style="min-width: 120px">
                                                <template #body="slotProps">
                                                    <span :class="getChangeClass(slotProps.data.change)">
                                                        {{ formatChange(slotProps.data.change) }}
                                                    </span>
                                                </template>
                                            </Column>
                                        </DataTable>
                                    </TabPanel>
                                    
                                    <TabPanel header="EUR">
                                        <DataTable :value="eurData" stripedRows :paginator="true" :rows="10">
                                            <Column field="date" header="Date" sortable style="min-width: 150px"></Column>
                                            <Column field="rate" header="CHF/EUR Rate" sortable style="min-width: 150px"></Column>
                                            <Column field="change" header="Change %" sortable style="min-width: 120px">
                                                <template #body="slotProps">
                                                    <span :class="getChangeClass(slotProps.data.change)">
                                                        {{ formatChange(slotProps.data.change) }}
                                                    </span>
                                                </template>
                                            </Column>
                                        </DataTable>
                                    </TabPanel>
                                    
                                    <TabPanel header="GBP">
                                        <DataTable :value="gbpData" stripedRows :paginator="true" :rows="10">
                                            <Column field="date" header="Date" sortable style="min-width: 150px"></Column>
                                            <Column field="rate" header="CHF/GBP Rate" sortable style="min-width: 150px"></Column>
                                            <Column field="change" header="Change %" sortable style="min-width: 120px">
                                                <template #body="slotProps">
                                                    <span :class="getChangeClass(slotProps.data.change)">
                                                        {{ formatChange(slotProps.data.change) }}
                                                    </span>
                                                </template>
                                            </Column>
                                        </DataTable>
                                    </TabPanel>
                                    
                                    <TabPanel header="JPY">
                                        <DataTable :value="jpyData" stripedRows :paginator="true" :rows="10">
                                            <Column field="date" header="Date" sortable style="min-width: 150px"></Column>
                                            <Column field="rate" header="CHF/JPY Rate" sortable style="min-width: 150px"></Column>
                                            <Column field="change" header="Change %" sortable style="min-width: 120px">
                                                <template #body="slotProps">
                                                    <span :class="getChangeClass(slotProps.data.change)">
                                                        {{ formatChange(slotProps.data.change) }}
                                                    </span>
                                                </template>
                                            </Column>
                                        </DataTable>
                                    </TabPanel>
                                </TabView>
                            </template>
                        </Card>
                    </div>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.card {
    height: 100%;
}

.warning-message {
    margin-bottom: 2rem;
}

.warning-message :deep(.p-message-wrapper) {
    padding: 1rem 1.25rem;
}

.warning-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1rem;
}

.warning-content i {
    font-size: 1.25rem;
}

.warning-content strong {
    margin-right: 0.25rem;
}
</style>

