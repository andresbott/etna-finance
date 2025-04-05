<script setup>
import { ref } from 'vue'
import Button from 'primevue/button'
import Divider from 'primevue/divider'
import { useUserStore } from '@/lib/user/userstore.js'

const user = useUserStore()
const emit = defineEmits(['logout'])

const sections = [
    {
        title: 'Account',
        items: [
            { label: 'Profile', icon: 'pi pi-user', route: '/profile' ,disabled: true },
            { label: 'Settings', icon: 'pi pi-cog', route: '/settings' , disabled: true }
        ]
    },
    {
        title: 'Security',
        items: [
            { label: 'Change Password', icon: 'pi pi-lock', route: '/security/password', disabled: true },
            { label: 'Two-Factor Auth', icon: 'pi pi-shield', route: '/security/2fa', disabled: true }
        ]
    },
    {
        title: 'Application Data',
        items: [
            { label: 'CSV Import Profiles', icon: 'pi pi-file-import', route: '/setup/csv-profiles', disabled: true  },
            { label: 'Categories', icon: 'pi pi-tags', route: '/categories' },
            { label: 'Account Setup', icon: 'pi pi-wallet', route: '/accounts' }
        ]
    }
]
</script>

<template>
    <div class="user-profile-menu">
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
                    @click="!item.disabled && $router.push(item.route)"
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
                @click="$emit('logout')"
            />
        </div>
    </div>
</template>

<style scoped>
.user-profile-menu {
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