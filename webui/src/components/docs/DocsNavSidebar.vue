<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Divider from 'primevue/divider'

const router = useRouter()

const sections = computed(() => {
    return [
        {
            title: 'Getting Started',
            items: [
                { label: 'Overview', icon: 'ti ti-home', route: '/docs/overview' },
                { label: 'Configuration', icon: 'ti ti-settings', route: '/docs/getting-started/configuration' },
            ]
        },
        {
            title: 'Basics',
            items: [
                { label: 'Accounts', icon: 'ti ti-wallet', route: '/docs/concepts/accounts' },
                { label: 'Categories', icon: 'ti ti-tags', route: '/docs/concepts/categories' },
                { label: 'Category Rules', icon: 'ti ti-filter', route: '/docs/concepts/category-rules' },
                { label: 'CSV Import Profiles', icon: 'ti ti-file-import', route: '/docs/concepts/csv-import-profiles' },
            ]
        },
        {
            title: 'Guides',
            items: [
                { label: 'Handling RSUs', icon: 'ti ti-chart-bar', route: '/docs/guides/handling-rsus' },
                { label: 'Handling ESPP', icon: 'ti ti-shopping-cart', route: '/docs/guides/handling-espp' },
            ]
        },
    ]
})
</script>

<template>
    <Card>
        <template #title>Documentation</template>
        <template #content>
            <div class="nav-content">
                <div v-for="(section, index) in sections" :key="section.title" class="section">
                    <h3 class="section-title">{{ section.title }}</h3>
                    <div class="menu-items">
                        <Button
                            v-for="item in section.items"
                            :key="item.label"
                            :label="item.label"
                            :icon="item.icon"
                            class="menu-item"
                            text
                            @click="router.push(item.route)"
                        />
                    </div>
                    <Divider v-if="index < sections.length - 1" />
                </div>
            </div>
        </template>
    </Card>
</template>

<style scoped>
.nav-content {
    padding: 0.5rem 0;
}

.section {
    margin-bottom: 1rem;
}

.section-title {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-color-secondary);
    margin-bottom: 0.5rem;
    padding: 0 0.5rem;
}

.menu-items {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
}

.menu-item {
    justify-content: flex-start;
    padding: 0.5rem;
    width: 100%;
    border-radius: 4px;
}

:deep(.menu-item:hover) {
    background-color: var(--surface-hover);
}

:deep(.p-button-label) {
    font-weight: 500;
}

:deep(.p-button-icon) {
    font-size: 1.35rem;
}
</style>
