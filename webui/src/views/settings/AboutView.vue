<script setup lang="ts">
import { computed } from 'vue'
import Card from 'primevue/card'
import Divider from 'primevue/divider'
import { useQuery } from '@tanstack/vue-query'
import { useSettingsStore } from '@/store/settingsStore'
import { useAccounts } from '@/composables/useAccounts'
import { useCategories } from '@/composables/useCategories'
import { getEntries } from '@/lib/api/Entry'

const settings = useSettingsStore()
const repoUrl = 'https://github.com/andresbott/etna'

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
</script>

<template>
    <Card>
        <template #title>About</template>
        <template #content>
            <div class="about-content">
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
