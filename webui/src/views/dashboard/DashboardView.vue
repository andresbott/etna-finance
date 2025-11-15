<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed, onMounted } from 'vue'
import Card from 'primevue/card'
import Chart from 'primevue/chart'
import { useAccounts } from '@/composables/useAccounts.js'
import { useGetBalanceReport } from '@/composables/useGetBalanceReport'

const { accounts } = useAccounts()
const { mutate, data: balanceReport } = useGetBalanceReport()

const mergedData = computed(() => {
    if (!balanceReport.value) return []
    const accountsData = accounts.value
        ? accounts.value
              .map((account) => {
                  const reportData = balanceReport.value?.accounts?.[account.id]
                  if (!reportData) return null // skip if no matching report data

                  return {
                      ...account,
                      reportData
                  }
              })
              .filter(Boolean)
        : []

    return accountsData
})
const generateRandomData = (count) => {
    return Array.from({ length: count }, () => Math.floor(Math.random() * 10000))
}

const getRandomColor = () => {
    const colors = [
        '#22c55e', // green
        '#3b82f6', // blue
        '#eab308', // yellow
        '#ef4444', // red
        '#8b5cf6', // purple
        '#ec4899', // pink
        '#14b8a6', // teal
        '#f97316' // orange
    ]
    return colors[Math.floor(Math.random() * colors.length)]
}

const chartLabels = ['January', 'February', 'March', 'April', 'May', 'June', 'July']

const chartData = computed(() => ({
    labels:
        mergedData.value?.[0]?.reportData?.map((r) =>
            new Date(r.Date).toLocaleDateString('en-GB', {
                day: '2-digit',
                month: 'short'
            })
        ) || [],

    datasets:
        mergedData.value?.map((account) => ({
            label: account.name,
            data: account.reportData.map((r) => r.Sum),
            fill: false,
            borderColor: getRandomColor(),
            tension: 0.1
        })) || []
}))

const pieChartData = ref({
    labels: ['Cash', 'Bank', 'Investment', 'Credit'],
    datasets: [
        {
            data: [3000, 5000, 10000, 2000],
            backgroundColor: ['#22c55e', '#3b82f6', '#eab308', '#ef4444']
        }
    ]
})

const pieChartOptions = ref({
    maintainAspectRatio: false,
    aspectRatio: 0.8,
    plugins: {
        legend: {
            labels: {
                color: '#495057'
            }
        }
    }
})

const chartOptions = ref({
    maintainAspectRatio: false,
    aspectRatio: 0.6,
    plugins: {
        legend: {
            position: 'bottom',
            display: true,
            align: 'start',
            labels: {
                color: '#495057',
                boxWidth: 20,
                padding: 20,
                usePointStyle: true,
                pointStyle: 'circle'
            },
            onClick: (e, legendItem, legend) => {
                const chart = legend.chart
                const datasetIndex = legendItem.datasetIndex
                const meta = chart.getDatasetMeta(datasetIndex)

                meta.hidden =
                    meta.hidden === null ? !chart.data.datasets[datasetIndex].hidden : null

                legendItem.fontStyle =
                    legendItem.fontStyle === 'line-through' ? 'normal' : 'line-through'

                if (meta.hidden) {
                    console.log('Hiding dataset', datasetIndex)
                } else {
                    console.log('Showing dataset', datasetIndex)
                }

                chart.update()
            }
        }
    },
    scales: {
        x: {
            ticks: {
                color: '#495057'
            },
            grid: {
                color: '#ebedef'
            }
        },
        y: {
            ticks: {
                color: '#495057'
            },
            grid: {
                color: '#ebedef'
            }
        }
    }
})

const leftSidebarCollapsed = ref(true)

const getAccountTypeIcon = (type) => {
    switch (type) {
        case 'Cash':
            return 'pi pi-money-bill'
        case 'Bank':
            return 'pi pi-credit-card'
        case 'Investment':
            return 'pi pi-chart-line'
        case 'Credit':
            return 'pi pi-credit-card'
        default:
            return 'pi pi-wallet'
    }
}

onMounted(() => {
    mutate({
        accountIds: [2, 3],
        steps: 30,
        startDate: '2025-01-03'
    })
})
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="grid p-3">
                <!-- Chart Card -->
                <div class="col-12">
                    <Card>
                        <template #title>Financial Overview</template>
                        <template #content>
                            <Chart
                                type="line"
                                :data="chartData"
                                :options="chartOptions"
                                style="height: 300px"
                            />
                        </template>
                    </Card>
                </div>

                <!-- Account Types List Card -->
                <div class="col-12 lg:col-6">
                    <Card>
                        <template #title>Account Types</template>
                        <template #content>
                            <div class="flex flex-column gap-2">
                                <div
                                    v-for="(amount, type) in {
                                        Cash: 3000,
                                        Bank: 5000,
                                        Investment: 10000,
                                        Credit: 2000
                                    }"
                                    :key="type"
                                    class="flex justify-content-between align-items-center p-2 border-round"
                                    style="background: var(--surface-ground)"
                                >
                                    <div class="flex align-items-center gap-2">
                                        <i :class="getAccountTypeIcon(type)"></i>
                                        <span>{{ type }}</span>
                                    </div>
                                    <div class="flex align-items-center gap-2">
                                        <span class="font-bold">{{ amount }}</span>
                                        <span>CHF</span>
                                    </div>
                                </div>
                            </div>
                        </template>
                    </Card>
                </div>

                <!-- Account Types Distribution Card -->
                <div class="col-12 lg:col-6">
                    <Card>
                        <template #title>Account Distribution</template>
                        <template #content>
                            <Chart
                                type="pie"
                                :data="pieChartData"
                                :options="pieChartOptions"
                                style="height: 300px"
                            />
                        </template>
                    </Card>
                </div>

                <!-- Accounts Card -->
                <div class="col-12">
                    <Card>
                        <template #title>Accounts</template>
                        <template #content>
                            <div class="flex flex-column gap-2">
                                <div
                                    v-for="account in mergedData"
                                    :key="account.id"
                                    class="flex justify-content-between align-items-center p-2 border-round"
                                    style="background: var(--surface-ground)"
                                >
                                    <div class="flex align-items-center gap-2">
                                        <i class="pi pi-wallet"></i>
                                        <span>{{ account.name }}</span>
                                    </div>
                                    <div class="flex align-items-center gap-2">
                                        <span class="font-bold">{{
                                            account.reportData.pop()['Sum'] || 0
                                        }}</span>
                                        <!-- <span>{{ account.accounts.currency }}</span> -->
                                    </div>
                                </div>
                            </div>
                        </template>
                    </Card>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.card {
    height: 100%;
}
</style>
