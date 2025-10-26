<script setup>
import { HorizontalLayout as HL } from '@go-bumbu/vue-layouts'

import UserProfile from '@/lib/user/UserProfile.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import Button from 'primevue/button'
import AppTitle from '@/views/parts/AppTitle.vue'
import { useUiStore } from '@/store/uiStore.js'
import Menubar from 'primevue/menubar'
import { useRouter } from 'vue-router'

const user = useUserStore()
const router = useRouter()
const uiStore = useUiStore()

const menuItems = [
    {
        label: 'Entries',
        command: () => {
            router.push('/entries')
        }
    },
    {
        label: 'Reports',
        command: () => {
            router.push('/reports')
        }
    }
]
</script>

<template>
    <HL class="topbar" :centerContent="true" :verticalCenterContent="false">
        <template #left>
            <router-link to="/start" class="layout-topbar-logo">
                <AppTitle icon="pi-money-bill" text="Etna" class="ml-4 mr-2" />
            </router-link>
        </template>

        <template #default>
            <Menubar :model="menuItems" class="nav-menu hidden lg:block" />
            <Button icon="pi pi-bars" @click="uiStore.openDrawer" class="block lg:hidden" />
        </template>

        <template #right>
            <UserProfile v-if="user.isLoggedIn" />
            <router-link v-if="!user.isLoggedIn" to="/login" class="layout-topbar-logo">
                <Button label="Login" icon="pi pi-sign-in" />
            </router-link>
        </template>
    </HL>
</template>

<style scoped lang="scss">
.topbar {
    background-color: var(--c-primary-600);
    padding: 5px 0;
}

.layout-topbar-logo {
    text-decoration: none;
    color: inherit;
}

.nav-menu {
    //background: transparent;
    border: none;
}

:deep(.p-menubar) {
    //background: transparent;
    border: none;
    padding: 0;
}

:deep(.p-menubar-root-list) {
    gap: 2rem;
}

:deep(.p-menubar .p-menuitem > .p-menuitem-link .p-menuitem-text),
:deep(.p-menubar .p-menuitem > .p-menuitem-link) {
    //color: white !important;
}

:deep(.p-menuitem-link) {
    //padding: 0.5rem 0;
}

//:deep(.p-menuitem-link:hover) {
//    background: transparent;
//    opacity: 0.8;
//}
</style>
