<script setup>
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import Chart from 'primevue/chart'

// Mock data - to be replaced with real data later
const pieChartData = ref({
    labels: ['Cash', 'Bank', 'Investment', 'Credit'],
    datasets: [
        {
            data: [3000, 5000, 10000, 2000],
            backgroundColor: ['#22c55e', '#3b82f6', '#eab308', '#ef4444']
        }
    ]
})

// Get computed colors from CSS variables
const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--c-text-color').trim() || '#495057'
}

const pieChartOptions = computed(() => ({
    maintainAspectRatio: false,
    aspectRatio: 0.8,
    plugins: {
        legend: {
            labels: {
                color: getTextColor()
            }
        }
    }
}))
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
            <Chart
                type="pie"
                :data="pieChartData"
                :options="pieChartOptions"
                style="height: 300px"
            />
        </template>
    </Card>
</template>

