<script setup>
import { HorizontalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import UserProfile from '@/lib/user/UserProfile.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import Button from 'primevue/button'

const user = useUserStore()
</script>

<template>
    <HorizontalLayout :centerContent="true">
        <template v-slot:left>
            <router-link to="/app" class="layout-topbar-logo"> </router-link>
        </template>

        <!--        <InputText placeholder="Search" type="text" class="w-32 sm:w-auto" />-->
        <template v-slot:right>
            <UserProfile v-if="user.isLoggedIn" />
            <router-link v-if="!user.isLoggedIn" to="/login" class="layout-topbar-logo">
                <Button label="Login" icon="pi pi-sign-in" />
            </router-link>
        </template>
    </HorizontalLayout>
</template>
