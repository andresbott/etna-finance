<script setup lang="ts">
import { computed } from 'vue'
import Card from 'primevue/card'
import Divider from 'primevue/divider'
import { useQuery } from '@tanstack/vue-query'
import { useSettingsStore } from '@/store/settingsStore'
import { useAccounts } from '@/composables/useAccounts'
import { useCategories } from '@/composables/useCategories'
import { getEntries } from '@/lib/api/Entry'
import { getStats } from '@/lib/api/Stats'

const settings = useSettingsStore()
const repoUrl = 'https://github.com/andresbott/etna'
const poweredByUrl = 'https://github.com/andresbott/etna-finance'

const { accounts: providers } = useAccounts()
const { incomeCategories, expenseCategories } = useCategories()

const farPast = new Date(2000, 0, 1)
const farFuture = new Date(2099, 11, 31)

const entriesCountQuery = useQuery({
    queryKey: ['entries-count'],
    queryFn: () => getEntries({ startDate: farPast, endDate: farFuture, page: 1, limit: 1 }),
})

const accountCount = computed(() => {
    if (!providers.value) return 0
    return providers.value.reduce((sum, p) => sum + (p.accounts?.length ?? 0), 0)
})

const categoryCount = computed(() => {
    const income = incomeCategories.data?.value?.length ?? 0
    const expense = expenseCategories.data?.value?.length ?? 0
    return income + expense
})

const transactionCount = computed(() => entriesCountQuery.data?.value?.total ?? 0)

const statsQuery = useQuery({
    queryKey: ['app-stats'],
    queryFn: getStats,
})

const stats = computed(() => statsQuery.data?.value)

function formatBytes(bytes: number): string {
    if (bytes <= 0) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
    const value = bytes / Math.pow(1024, i)
    return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

const marketDataSummary = computed(() => {
    const s = stats.value
    if (!s) return '—'
    const series = `${s.priceSeries.toLocaleString()} ${s.priceSeries === 1 ? 'symbol' : 'symbols'}`
    return `${series} · ${s.pricePoints.toLocaleString()} points`
})

const fxSummary = computed(() => {
    const s = stats.value
    if (!s) return '—'
    const series = `${s.fxSeries.toLocaleString()} ${s.fxSeries === 1 ? 'pair' : 'pairs'}`
    return `${series} · ${s.fxPoints.toLocaleString()} points`
})

const dbSizeDisplay = computed(() => (stats.value ? formatBytes(stats.value.dbSizeBytes) : '—'))
const attachmentsSizeDisplay = computed(() =>
    stats.value ? formatBytes(stats.value.attachmentsSizeBytes) : '—'
)
</script>

<template>
    <Card>
        <template #title>About</template>
        <template #content>
            <div class="about-content">
                <div class="about-row">
                    <span class="about-label">Powered by</span>
                    <a :href="poweredByUrl" target="_blank" rel="noopener noreferrer" class="about-link">
                        etna-finance
                        <i class="ti ti-external-link link-icon"></i>
                    </a>
                </div>
                <div class="about-row">
                    <span class="about-label">Version</span>
                    <span class="about-value">{{ settings.version || 'development' }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">License</span>
                    <span class="about-value">AGPL-3.0</span>
                </div>
                <div class="about-row">
                    <span class="about-label">Source code</span>
                    <a :href="repoUrl" target="_blank" rel="noopener noreferrer" class="about-link">
                        {{ repoUrl }}
                        <i class="ti ti-external-link link-icon"></i>
                    </a>
                </div>
            </div>

            <Divider />

            <h3 class="stats-title">Statistics</h3>
            <div class="about-content">
                <div class="about-row">
                    <span class="about-label">Transactions</span>
                    <span class="about-value">{{ transactionCount.toLocaleString() }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">Accounts</span>
                    <span class="about-value">{{ accountCount.toLocaleString() }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">Categories</span>
                    <span class="about-value">{{ categoryCount.toLocaleString() }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">Market data</span>
                    <span class="about-value">{{ marketDataSummary }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">FX rates</span>
                    <span class="about-value">{{ fxSummary }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">Database size</span>
                    <span class="about-value">{{ dbSizeDisplay }}</span>
                </div>
                <div class="about-row">
                    <span class="about-label">Attachments size</span>
                    <span class="about-value">{{ attachmentsSizeDisplay }}</span>
                </div>
            </div>
        </template>
    </Card>
</template>

<style scoped>
.about-content {
    max-width: 480px;
}

.about-row {
    display: flex;
    align-items: baseline;
    padding: 0.5rem 0;
    border-bottom: 1px solid var(--surface-border);
}

.about-row:last-child {
    border-bottom: none;
}

.about-label {
    font-weight: 600;
    color: var(--text-color);
    min-width: 120px;
}

.about-value {
    color: var(--text-color-secondary);
}

.about-link {
    color: var(--primary-color);
    text-decoration: none;
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
}

.about-link:hover {
    text-decoration: underline;
}

.link-icon {
    font-size: 0.875rem;
}

.stats-title {
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--text-color);
    margin-bottom: 0.5rem;
}
</style>
