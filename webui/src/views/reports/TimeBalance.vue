<script setup>
import { computed, watch } from 'vue'
import Card from 'primevue/card'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
    GridComponent,
    TooltipComponent,
    LegendComponent,
    DataZoomComponent
} from 'echarts/components'
import { useAccounts } from '@/composables/useAccounts.js'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'
import { useDateFormat } from '@/composables/useDateFormat'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent])

const { accounts } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation
const { formatDate } = useDateFormat()

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
            if (!reportData) return null

            return {
                ...account,
                reportData
            }
        })
        .filter(Boolean)
})

const colors = [
    '#22c55e', '#3b82f6', '#eab308', '#ef4444',
    '#8b5cf6', '#ec4899', '#14b8a6', '#f97316',
    '#06b6d4', '#84cc16', '#f43f5e', '#6366f1',
    '#10b981', '#f59e0b', '#8b5cf6', '#0ea5e9',
    '#a855f7', '#ec4899', '#14b8a6', '#fb923c'
]

const getColor = (index) => {
    if (index >= 20) return '#6b7280'
    return colors[index]
}

const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-text-color').trim() || '#495057'
}

const getSurfaceColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-surface-300').trim() || '#ebedef'
}

const chartOption = computed(() => {
    const labels =
        mergedData.value?.[0]?.reportData?.map((r) => formatDate(r.Date)) || []

    const series =
        mergedData.value?.map((account, index) => ({
            name: account.name,
            type: 'line',
            smooth: 0.1,
            showSymbol: false,
            data: account.reportData.map((r) => r.Sum),
            lineStyle: { color: getColor(index) },
            itemStyle: { color: getColor(index) },
            _currency: account.currency || 'CHF'
        })) || []

    const textColor = getTextColor()
    const surfaceColor = getSurfaceColor()

    return {
        animation: true,
        animationDuration: 500,
        grid: {
            left: '3%',
            right: '4%',
            bottom: '18%',
            top: '3%',
            containLabel: true
        },
        tooltip: {
            trigger: 'axis',
            formatter: (params) => {
                let result = `<strong>${params[0].axisValueLabel}</strong><br/>`
                for (const p of params) {
                    const s = series[p.seriesIndex]
                    const currency = s?._currency || 'CHF'
                    result += `${p.marker} ${p.seriesName}: ${formatAmount(p.value)} ${currency}<br/>`
                }
                return result
            }
        },
        legend: {
            type: 'scroll',
            bottom: 0,
            left: 'left',
            textStyle: { color: textColor },
            icon: 'circle',
            itemWidth: 10,
            itemHeight: 10,
            itemGap: 20,
            selectedMode: 'multiple'
        },
        xAxis: {
            type: 'category',
            data: labels,
            axisLabel: { color: textColor },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: textColor },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        series
    }
})

// Fetch balance reports when accounts are loaded
watch(
    allAccounts,
    (accountsList) => {
        if (accountsList && accountsList.length > 0) {
            const accountIds = accountsList.map((account) => account.id).filter(Boolean)
            
            if (accountIds.length > 0) {
                const today = new Date()
                const sixMonthsAgo = new Date(today)
                sixMonthsAgo.setMonth(today.getMonth() - 6)
                const startDate = sixMonthsAgo.toISOString().split('T')[0]
                
                mutate({
                    accountIds,
                    steps: 90,
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
            <VChart
                :option="chartOption"
                autoresize
                style="height: 450px"
            />
        </template>
    </Card>
</template>
