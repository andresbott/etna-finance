<script setup>
import { computed, watch } from 'vue'
import Card from 'primevue/card'
import Chart from 'primevue/chart'
import { useAccounts } from '@/composables/useAccounts.js'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'

const { accounts } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation

// Gather all accounts from all providers
const allAccounts = computed(() => {
    if (!accounts.value) return []
    
    const accountsList = []
    for (const provider of accounts.value) {
        if (provider.accounts && Array.isArray(provider.accounts)) {
            accountsList.push(...provider.accounts)
        }
    }
    return accountsList
})

// Merge accounts with their balance report data
const mergedData = computed(() => {
    if (!balanceReport.value || !allAccounts.value) return []
    
    return allAccounts.value
        .map((account) => {
            const reportData = balanceReport.value?.accounts?.[account.id]
            if (!reportData) return null // skip if no matching report data

            return {
                ...account,
                reportData
            }
        })
        .filter(Boolean)
})

const colors = [
    '#22c55e', // green
    '#3b82f6', // blue
    '#eab308', // yellow
    '#ef4444', // red
    '#8b5cf6', // purple
    '#ec4899', // pink
    '#14b8a6', // teal
    '#f97316', // orange
    '#06b6d4', // cyan
    '#84cc16', // lime
    '#f43f5e', // rose
    '#6366f1', // indigo
    '#10b981', // emerald
    '#f59e0b', // amber
    '#8b5cf6', // violet
    '#0ea5e9', // sky
    '#a855f7', // fuchsia
    '#ec4899', // hot pink
    '#14b8a6', // teal light
    '#fb923c'  // orange light
]

const getColor = (index) => {
    // Use gray for accounts beyond the 20th
    if (index >= 20) {
        return '#6b7280' // gray
    }
    return colors[index]
}

const chartData = computed(() => ({
    labels:
        mergedData.value?.[0]?.reportData?.map((r) =>
            new Date(r.Date).toLocaleDateString('en-GB', {
                day: '2-digit',
                month: 'short'
            })
        ) || [],

    datasets:
        mergedData.value?.map((account, index) => ({
            label: account.name,
            data: account.reportData.map((r) => r.Sum),
            fill: false,
            borderColor: getColor(index),
            tension: 0.1,
            currency: account.currency || 'CHF' // Store currency for tooltip
        })) || []
}))

// Get computed colors from CSS variables
const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-text-color').trim() || '#495057'
}

const getSurfaceColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-surface-300').trim() || '#ebedef'
}

const chartOptions = computed(() => ({
    maintainAspectRatio: false,
    aspectRatio: 0.6,
    animation: {
        duration: 500 // Fast animation in milliseconds (default is 1000)
    },
    plugins: {
        legend: {
            position: 'bottom',
            display: true,
            align: 'start',
            labels: {
                color: getTextColor(),
                boxWidth: 20,
                padding: 20,
                usePointStyle: true,
                pointStyle: 'circle'
            },
            onClick: (e, legendItem, legend) => {
                const chart = legend.chart
                const datasetIndex = legendItem.datasetIndex
                const clickedMeta = chart.getDatasetMeta(datasetIndex)
                
                // Check if the clicked dataset is currently the only visible one
                const visibleDatasets = chart.data.datasets.map((_, i) => 
                    chart.getDatasetMeta(i)
                ).filter(meta => !meta.hidden)
                
                const isOnlyOneVisible = visibleDatasets.length === 1 && !clickedMeta.hidden
                
                if (isOnlyOneVisible) {
                    // If only one is visible and we click it, show all datasets
                    chart.data.datasets.forEach((_, i) => {
                        chart.getDatasetMeta(i).hidden = false
                    })
                } else {
                    // Hide all datasets except the clicked one
                    chart.data.datasets.forEach((_, i) => {
                        const meta = chart.getDatasetMeta(i)
                        meta.hidden = i !== datasetIndex
                    })
                }
                
                chart.update()
            }
        },
        tooltip: {
            callbacks: {
                label: function(context) {
                    const label = context.dataset.label || ''
                    const value = context.parsed.y
                    const currency = context.dataset.currency || 'CHF'
                    return `${label}: ${formatAmount(value)} ${currency}`
                }
            }
        }
    },
    scales: {
        x: {
            ticks: {
                color: getTextColor()
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

// Fetch balance reports when accounts are loaded
watch(
    allAccounts,
    (accountsList) => {
        if (accountsList && accountsList.length > 0) {
            const accountIds = accountsList.map((account) => account.id).filter(Boolean)
            
            if (accountIds.length > 0) {
                // Calculate start date as 6 months ago
                const today = new Date()
                const sixMonthsAgo = new Date(today)
                sixMonthsAgo.setMonth(today.getMonth() - 6)
                const startDate = sixMonthsAgo.toISOString().split('T')[0]
                
                mutate({
                    accountIds,
                    steps: 15,
                    startDate
                })
            }
        }
    },
    { immediate: true }
)
</script>

<template>
    <Card>
        <template #title>Financial Overview</template>
        <template #content>
            <Chart
                type="line"
                :data="chartData"
                :options="chartOptions"
                style="height: 450px"
            />
        </template>
    </Card>
</template>

