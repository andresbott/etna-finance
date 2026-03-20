<script setup>
import { computed, ref, watch } from 'vue'
import Card from 'primevue/card'
import SelectButton from 'primevue/selectbutton'
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
import { useAccounts } from '@/composables/useAccounts'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'
import { useDateFormat } from '@/composables/useDateFormat'
import { rangeToStartEnd } from '@/utils/dateRange'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent])

const { accounts } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation
const { formatDate } = useDateFormat()

const dateRangeOptions = [
    { value: '3m', label: '3M' },
    { value: '1y', label: '1Y' }
]
const selectedRange = ref('3m')

const stepsForRange = (_range) => 90

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
        mergedData.value?.[0]?.reportData?.map((r) => formatDate(r.date)) || []

    const textColor = getTextColor()
    const surfaceColor = getSurfaceColor()

    const series =
        mergedData.value?.map((account, index) => ({
            name: account.name,
            type: 'line',
            smooth: 0.1,
            showSymbol: false,
            data: account.reportData.map((r) => r.sum),
            lineStyle: { color: getColor(index) },
            itemStyle: { color: getColor(index) },
            yAxisIndex: 0,
            _currency: account.currency || 'CHF'
        })) || []

    // Sum of all non-unvested accounts per date point
    const vestedAccounts = mergedData.value?.filter((a) => a.type !== 'unvested') || []
    const totalData = labels.map((_, i) =>
        vestedAccounts.reduce((sum, a) => sum + (a.reportData[i]?.sum || 0), 0)
    )

    series.push({
        name: 'Total',
        type: 'line',
        smooth: 0.1,
        showSymbol: false,
        clip: false,
        data: totalData,
        lineStyle: { color: textColor, width: 2.5, type: 'dashed' },
        itemStyle: { color: textColor },
        yAxisIndex: 1,
        _currency: null,
        endLabel: {
            show: true,
            formatter: (params) => formatAmount(params.value),
            color: textColor,
            fontWeight: 'bold'
        }
    })

    return {
        animation: true,
        animationDuration: 500,
        grid: {
            left: '3%',
            right: '19%',
            bottom: '3%',
            top: '3%',
            containLabel: true
        },
        tooltip: {
            trigger: 'axis',
            formatter: (params) => {
                let result = `<strong>${params[0].axisValueLabel}</strong><br/>`
                for (const p of params) {
                    const s = series[p.seriesIndex]
                    const currency = s?._currency
                    const formatted = formatAmount(p.value)
                    result += `${p.marker} ${p.seriesName}: ${formatted}${currency ? ` ${currency}` : ''}<br/>`
                }
                return result
            }
        },
        legend: {
            type: 'scroll',
            orient: 'vertical',
            right: 10,
            top: 'middle',
            textStyle: { color: textColor },
            icon: 'circle',
            itemWidth: 10,
            itemHeight: 10,
            itemGap: 12,
            selectedMode: 'multiple'
        },
        xAxis: {
            type: 'category',
            data: labels,
            axisLabel: { show: false },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        yAxis: [
            {
                type: 'value',
                axisLabel: { color: textColor },
                axisLine: { lineStyle: { color: surfaceColor } },
                splitLine: { lineStyle: { color: surfaceColor } }
            },
            {
                type: 'value',
                position: 'right',
                axisLabel: { color: textColor },
                axisLine: { show: true, lineStyle: { color: surfaceColor } },
                splitLine: { show: false }
            }
        ],
        series
    }
})

// Fetch balance reports when accounts are loaded or range changes
watch(
    [allAccounts, selectedRange],
    ([accountsList]) => {
        if (accountsList && accountsList.length > 0) {
            const accountIds = accountsList.map((account) => account.id).filter(Boolean)

            if (accountIds.length > 0) {
                const { start } = rangeToStartEnd(selectedRange.value)
                mutate({
                    accountIds,
                    steps: stepsForRange(selectedRange.value),
                    startDate: start
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
            <div style="display: flex; justify-content: flex-end; margin-bottom: 0.75rem;">
                <SelectButton
                    v-model="selectedRange"
                    :options="dateRangeOptions"
                    optionLabel="label"
                    optionValue="value"
                />
            </div>
            <VChart
                :option="chartOption"
                autoresize
                style="height: 450px"
            />
        </template>
    </Card>
</template>
