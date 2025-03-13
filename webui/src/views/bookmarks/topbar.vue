<script setup>
import Horizontal from '@/lib/layout/horizontal.vue'
import UserProfile from '@/lib/user/UserProfile.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import Button from 'primevue/button'
import Search from '@/views/bookmarks/search.vue'
import AppTitle from '@/views/parts/AppTitle.vue'
import AppSelector from '@/views/parts/AppSelector.vue'

const user = useUserStore()
</script>

<template>
    <horizontal class="topbar" :horizontal-center="true" :vertical-center="true">
        <template #left>
            <!--            <AppSelector icon="pi pi-ellipsis-h" class="ml-4 mr-2" />-->
            <!--            <router-link to="/bookmarks" class="layout-topbar-logo">-->
            <AppTitle icon="" text="Bookmarks" class="ml-4 mr-2" />
            <!--            </router-link>-->
        </template>

        <template #default>
            <search />
        </template>

        <template #right>
            <UserProfile v-if="user.isLoggedIn" />
            <router-link v-if="!user.isLoggedIn" to="/login" class="layout-topbar-logo">
                <Button label="Login" icon="pi pi-sign-in" />
            </router-link>
        </template>
    </horizontal>
</template>

<style scoped>
.topbar {
    background-color: var(--c-surface-600);
    border-bottom: 1px solid #e5e5e5;
    height: 5rem;
}
.c-horizontal {
    grid-template-columns: 1fr 2fr 1fr;
}
</style>
