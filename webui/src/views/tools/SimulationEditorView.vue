<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PortfolioSimulatorView from './PortfolioSimulatorView.vue'
import RealEstateSimulatorView from './RealEstateSimulatorView.vue'
import BuyVsRentSimulatorView from './BuyVsRentSimulatorView.vue'

const route = useRoute()
const toolType = computed(() => route.params.toolType as string)
const caseId = computed(() => Number(route.params.id))
</script>

<template>
  <PortfolioSimulatorView v-if="toolType === 'portfolio-simulator'" :caseId="caseId" />
  <RealEstateSimulatorView v-else-if="toolType === 'real-estate-simulator'" :caseId="caseId" />
  <BuyVsRentSimulatorView v-else-if="toolType === 'buy-vs-rent-simulator'" :caseId="caseId" />
  <div v-else>Unknown simulation type: {{ toolType }}</div>
</template>
