<script setup>
import { computed, ref, watch } from 'vue'
import Card from 'primevue/card'
import SelectButton from 'primevue/selectbutton'
import Tag from 'primevue/tag'
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
import { ACCOUNT_TYPES } from '@/types/account'
import { useAccountTypesData } from '@/composables/useAccountTypesData'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'
import { formatPct, getChangeSeverity } from '@/utils/format'
import { formatChange } from '@/composables/useMarketData'
import { useDateFormat } from '@/composables/useDateFormat'
import { rangeToStartEnd } from '@/utils/dateRange'
import { useSettingsStore } from '@/store/settingsStore.js'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent])

const { accounts } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation
const { formatDate } = useDateFormat()
const settings = useSettingsStore()

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

// Aggregate balance data by provider
const providerData = computed(() => {
    if (!balanceReport.value || !accounts.value) return []

    return accounts.value
        .map((provider) => {
            if (!provider.accounts || provider.accounts.length === 0) return null

            // Collect report data for each account in this provider
            const accountReports = provider.accounts
                .map((account) => ({
                    type: account.type,
                    reportData: balanceReport.value?.accounts?.[account.id]
                }))
                .filter((a) => a.reportData)

            if (accountReports.length === 0) return null

            // Sum balances across accounts per date point
            const reportData = accountReports[0].reportData.map((point, i) => ({
                date: point.date,
                sum: accountReports.reduce((total, a) => total + (a.reportData[i]?.sum || 0), 0),
                unconverted: accountReports.some((a) => a.reportData[i]?.unconverted)
            }))

            return {
                id: provider.id,
                name: provider.name,
                icon: provider.icon,
                reportData
            }
        })
        .filter(Boolean)
})

// Account types excluded from the total (not liquid / not accessible)
const excludedFromTotal = new Set([ACCOUNT_TYPES.RESTRICTED_STOCK, ACCOUNT_TYPES.PREPAID_EXPENSE])

// Per-account report data for the total line (excluding restricted stock & prepaid)
const totalAccountReports = computed(() => {
    if (!balanceReport.value || !accounts.value) return []
    const reports = []
    for (const provider of accounts.value) {
        for (const account of provider.accounts ?? []) {
            if (excludedFromTotal.has(account.type)) continue
            const reportData = balanceReport.value?.accounts?.[account.id]
            if (reportData) reports.push(reportData)
        }
    }
    return reports
})

// Use the same data source as AccountTypesList for the header total
const { accountsByType, totalInMainCurrency } = useAccountTypesData()

const totalCurrentValue = computed(() => {
    const rows = accountsByType.value ?? []
    if (rows.length === 0) return null
    return rows
        .filter((row) => !excludedFromTotal.has(row.type))
        .reduce((sum, row) => sum + totalInMainCurrency(row), 0)
})

const totalFirstValue = computed(() => {
    const reports = totalAccountReports.value
    if (reports.length === 0) return null
    return reports.reduce((sum, r) => sum + (r[0]?.sum || 0), 0)
})

const totalChange = computed(() => {
    if (totalCurrentValue.value == null || totalFirstValue.value == null) return null
    return totalCurrentValue.value - totalFirstValue.value
})

const totalChangePct = computed(() => {
    if (totalChange.value == null || totalFirstValue.value == null || totalFirstValue.value === 0) return null
    return (totalChange.value / totalFirstValue.value) * 100
})

const mainCurrency = computed(() => settings.mainCurrency || 'CHF')

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
        providerData.value?.[0]?.reportData?.map((r) => formatDate(r.date)) || []

    const textColor = getTextColor()
    const surfaceColor = getSurfaceColor()

    const series =
        providerData.value?.map((provider, index) => ({
            name: provider.name,
            type: 'line',
            smooth: 0.1,
            showSymbol: false,
            data: provider.reportData.map((r) => r.sum),
            lineStyle: { color: getColor(index) },
            itemStyle: { color: getColor(index) },
            yAxisIndex: 0
        })) || []

    // Sum of accounts excluding restricted stock & prepaid per date point
    const reports = totalAccountReports.value
    const totalData = labels.map((_, i) =>
        reports.reduce((sum, r) => sum + (r[i]?.sum || 0), 0)
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
                    const formatted = formatAmount(p.value)
                    result += `${p.marker} ${p.seriesName}: ${formatted} ${mainCurrency.value}<br/>`
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
            <div class="chart-controls">
                <div v-if="totalCurrentValue != null" class="price-row">
                    <span class="current-price">{{ formatAmount(totalCurrentValue) }}</span>
                    <span class="currency-label">{{ mainCurrency }}</span>
                    <Tag
                        :value="formatPct(totalChangePct)"
                        :severity="getChangeSeverity(totalChangePct)"
                        class="ml-2"
                    />
                    <span
                        :class="(totalChange ?? 0) >= 0 ? 'text-green-600' : 'text-red-600'"
                        class="change-value ml-2"
                    >
                        {{ formatChange(totalChange) }}
                    </span>
                </div>
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

<style scoped>
.chart-controls {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.75rem;
    flex-wrap: wrap;
    gap: 0.5rem;
}

.price-row {
    display: flex;
    align-items: center;
    gap: 0.25rem;
}

.current-price {
    font-size: 1.6rem;
    font-weight: 700;
}

.currency-label {
    font-size: 0.9rem;
    color: var(--p-text-muted-color);
    margin-left: 0.25rem;
}

.change-value {
    font-weight: 600;
    font-size: 0.95rem;
}
</style>
