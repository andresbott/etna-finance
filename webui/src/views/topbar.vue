<script setup>
import { HorizontalLayout as HL } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'

import UserProfile from '@/lib/user/UserProfile.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import Button from 'primevue/button'
import AppTitle from '@/views/parts/AppTitle.vue'

const user = useUserStore()
</script>

<template>
    <HL class="topbar" :centerContent="true" :verticalCenterContent="false">
        <template #left>
            <!--            <AppSelector icon="pi pi-ellipsis-h" class="ml-4 mr-2" />-->
            <!--            <router-link to="/bookmarks" class="layout-topbar-logo">-->
            <AppTitle icon="pi-money-bill" text="Etna" class="ml-4 mr-2" />
            <!--            </router-link>-->
        </template>

        <template #default> </template>

        <template #right>
            <UserProfile v-if="user.isLoggedIn" />
            <router-link v-if="!user.isLoggedIn" to="/login" class="layout-topbar-logo">
                <Button label="Login" icon="pi pi-sign-in" />
            </router-link>
        </template>
    </HL>
</template>

<style scoped>
.topbar {
    background-color: var(--c-primary-600);
    padding: 5px 0;
}
</style>
