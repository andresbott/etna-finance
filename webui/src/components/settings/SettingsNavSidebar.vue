<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Divider from 'primevue/divider'
import { useSettingsStore } from '@/store/settingsStore'

const router = useRouter()
const settings = useSettingsStore()

const sections = computed(() => {
    const result = [
        {
            title: 'Settings',
            items: [
                { label: 'Configuration', icon: 'pi pi-cog', route: '/settings/configuration' }
            ]
        },
        {
            title: 'Accounts',
            items: [
                { label: 'Account Setup', icon: 'pi pi-wallet', route: '/settings/accounts' },
                { label: 'CSV Import', icon: 'pi pi-file-import', route: '/settings/csv-profiles' },
            ]
        },
        {
            title: 'Categories',
            items: [
                { label: 'Categories', icon: 'pi pi-tags', route: '/settings/categories' },
                { label: 'Category Rules', icon: 'pi pi-bolt', route: '/settings/category-rules' },
            ]
        },
    ]

    if (settings.instruments) {
        result.push({
            title: 'Investments',
            items: [
                { label: 'Investment Products', icon: 'pi pi-chart-bar', route: '/settings/instruments' }
            ]
        })
    }

    result.push({
        title: 'Maintenance',
        items: [
            { label: 'Backup/Restore', icon: 'pi pi-database', route: '/settings/backup-restore' },
            { label: 'Tasks', icon: 'pi pi-briefcase', route: '/settings/tasks' },
        ]
    })

    return result
})
</script>

<template>
    <Card>
        <template #title>Settings</template>
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

:deep(.p-button:disabled) {
    opacity: 0.5;
    cursor: not-allowed;
}
</style>
