<script setup>
import { VerticalLayout, Placeholder, ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import TopBar from '@/views/topbar.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import { onMounted, ref, computed } from 'vue'
import Card from 'primevue/card'
import Chart from 'primevue/chart'
import { useAccounts } from '@/composables/useAccounts.js'

const { accounts } = useAccounts()

// Add your data fetching and state management here
const loading = ref(false)

const generateRandomData = (count) => {
    return Array.from({ length: count }, () => Math.floor(Math.random() * 10000))
}

const fetchReportData = async () => {
    try {
        const response = await fetch(
            '/api/v0/fin/report/balance?accountIds=1&steps=30&startDate=2025-01-03'
        )

        if (!response.ok) {
            throw new Error('Network response was not ok')
        }
        const data = await response.json()

        console.log('Fetched report data:', data)
    } catch (error) {
        console.error('Error fetching report data:', error)
    }
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
    labels: chartLabels,
    datasets:
        accounts.value?.map((account) => ({
            label: account.name,
            data: generateRandomData(7),
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

onMounted(() => {
    // Initialize your data here
    fetchReportData()
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
                                    v-for="account in accounts"
                                    :key="account.id"
                                    class="flex justify-content-between align-items-center p-2 border-round"
                                    style="background: var(--surface-ground)"
                                >
                                    <div class="flex align-items-center gap-2">
                                        <i class="pi pi-wallet"></i>
                                        <span>{{ account.name }}</span>
                                    </div>
                                    <div class="flex align-items-center gap-2">
                                        <span class="font-bold">{{ account.balance || 0 }}</span>
                                        <span>{{ account.currency }}</span>
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
