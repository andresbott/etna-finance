<script setup>
import { computed } from 'vue'
import Card from 'primevue/card'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, PieChart, TooltipComponent, LegendComponent])

// Get computed colors from CSS variables
const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-text-color').trim() || '#495057'
}

// Mock data - to be replaced with real data later
const chartOption = computed(() => {
    const textColor = getTextColor()

    return {
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c} ({d}%)'
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
                data: [
                    { value: 3000, name: 'Cash', itemStyle: { color: '#22c55e' } },
                    { value: 5000, name: 'Bank', itemStyle: { color: '#3b82f6' } },
                    { value: 10000, name: 'Investment', itemStyle: { color: '#eab308' } },
                    { value: 2000, name: 'Credit', itemStyle: { color: '#ef4444' } }
                ]
            }
        ]
    }
})
</script>

<template>
    <Card>
        <template #title>
            <div>
                <div>Account Distribution</div>
                <div class="text-sm font-normal mt-1" style="color: red">
                    This is just a mock component
                </div>
            </div>
        </template>
        <template #content>
            <VChart
                :option="chartOption"
                autoresize
                style="height: 300px"
            />
        </template>
    </Card>
</template>
