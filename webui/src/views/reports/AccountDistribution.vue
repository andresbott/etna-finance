<script setup>
import { computed } from 'vue'
import Card from 'primevue/card'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { useAccountTypesData } from '@/composables/useAccountTypesData'
import { getAccountTypeLabel, ACCOUNT_TYPES } from '@/types/account'
import { formatAmount } from '@/utils/currency'

use([CanvasRenderer, PieChart, TooltipComponent, LegendComponent])

const { accountsByType, totalInMainCurrency, mainCurrency } = useAccountTypesData()

const colors = [
    '#22c55e', '#3b82f6', '#eab308', '#ef4444',
    '#8b5cf6', '#ec4899', '#14b8a6', '#f97316'
]

const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-text-color').trim() || '#495057'
}

const chartOption = computed(() => {
    const textColor = getTextColor()
    const hiddenTypes = [ACCOUNT_TYPES.RESTRICTED_STOCK, ACCOUNT_TYPES.PREPAID_EXPENSE]
    const rows = (accountsByType.value ?? []).filter((row) => !hiddenTypes.includes(row.type))
    const data = rows.map((row, index) => ({
        value: totalInMainCurrency(row),
        name: getAccountTypeLabel(row.type),
        itemStyle: { color: colors[index % colors.length] }
    })).filter((d) => d.value > 0)

    return {
        tooltip: {
            trigger: 'item',
            formatter: (params) => {
                const pct = params.percent != null ? params.percent.toFixed(1) : '0'
                return `${params.name}: ${formatAmount(params.value)} ${mainCurrency.value} (${pct}%)`
            }
        },
        legend: {
            bottom: 0,
            textStyle: { color: textColor }
        },
        series: [
            {
                type: 'pie',
                radius: ['0%', '70%'],
                center: ['50%', '45%'],
                avoidLabelOverlap: true,
                label: {
                    show: false
                },
                emphasis: {
                    label: {
                        show: true,
                        fontSize: 14,
                        fontWeight: 'bold'
                    }
                },
                labelLine: {
                    show: false
                },
                data
            }
        ]
    }
})
</script>

<template>
    <Card>
        <template #title>Account Distribution</template>
        <template #content>
            <div v-if="!accountsByType?.length" class="text-center p-3 text-500">
                No data available
            </div>
            <VChart
                v-else
                :option="chartOption"
                autoresize
                style="height: 300px"
            />
        </template>
    </Card>
</template>
