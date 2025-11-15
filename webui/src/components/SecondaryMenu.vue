<script setup>
import Button from 'primevue/button'
import Divider from 'primevue/divider'
import Drawer from 'primevue/drawer'
import Avatar from 'primevue/avatar'
import { useUserStore } from '@/lib/user/userstore.js'
import { useRouter } from 'vue-router'
import { useUiStore } from '@/store/uiStore.js'

const user = useUserStore()
const router = useRouter()
const uiStore = useUiStore()

const handleNavigation = (route) => {
    router.push(route)
    uiStore.closeSecondaryDrawer()
}

const handleLogout = () => {
    user.logout()
    uiStore.closeSecondaryDrawer()
}

const sections = [
    {
        title: 'Settings',
        items: [
            { label: 'Configuration', icon: 'pi pi-cog', route: '/settings', disabled: true }
        ]
    },
    {
        title: 'Application Data',
        items: [
            {
                label: 'CSV Import Profiles',
                icon: 'pi pi-file-import',
                route: '/setup/csv-profiles'
            },
            { label: 'Categories', icon: 'pi pi-tags', route: '/categories' },
            { label: 'Account Setup', icon: 'pi pi-wallet', route: '/accounts' }
        ]
    },
    {
        title: 'Maintenance',
        items: [
            {
                label: 'Backup/Restore',
                icon: 'pi pi-database',
                route: '/backup-restore'
            }
        ]
    }
]
</script>

<template>
    <Drawer
        v-model:visible="uiStore.isSecondaryDrawerVisible"
        style="width: 25rem"
        position="right"
    >
        <template #header>
            <div class="drawer-header">
                <Avatar
                    icon="pi pi-user"
                    size="large"
                    style="background-color: #ece9fc; color: #2a1261"
                />
                <span class="username">{{ user.loggedInUser }}</span>
            </div>
        </template>

        <div class="secondary-menu-content">
            <div v-for="section in sections" :key="section.title" class="section">
                <h3 class="section-title">{{ section.title }}</h3>
                <div class="menu-items">
                    <Button
                        v-for="item in section.items"
                        :key="item.label"
                        :label="item.label"
                        :icon="item.icon"
                        class="menu-item"
                        :class="{ 'disabled-item': item.disabled }"
                        :disabled="item.disabled"
                        text
                        @click="!item.disabled && handleNavigation(item.route)"
                    />
                </div>
                <Divider />
            </div>

            <div class="section">
                <Button
                    label="Logout"
                    icon="pi pi-sign-out"
                    severity="danger"
                    class="menu-item"
                    text
                    @click="handleLogout"
                />
            </div>
        </div>
    </Drawer>

</template>

<style scoped>
.drawer-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.5rem;
}

.username {
    font-size: 1.25rem;
    font-weight: 600;
}

.secondary-menu-content {
    padding: 1rem;
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

.menu-item:hover {
    background-color: var(--surface-hover);
}

.disabled-item {
    opacity: 0.5;
    cursor: not-allowed;
}

:deep(.p-button-label) {
    font-weight: 500;
}

:deep(.p-button:disabled) {
    opacity: 0.5;
    cursor: not-allowed;
}
</style>

