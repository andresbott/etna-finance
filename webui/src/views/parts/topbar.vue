<script setup>
import { HorizontalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { useUserStore } from '@/lib/user/userstore.js'
import { useUiStore } from '@/store/uiStore.js'
import Button from 'primevue/button'
import Avatar from 'primevue/avatar'
import { useRouter } from 'vue-router'

const user = useUserStore()
const uiStore = useUiStore()
const router = useRouter()

const toggleSecondaryMenu = () => {
    uiStore.toggleSecondaryDrawer()
}

// Register logout action
user.registerLogoutAction(() => {
    router.push({ path: '/', force: true })
})

</script>

<template>
    <HorizontalLayout :centerContent="true">
        <template v-slot:left>
            <router-link to="/app" class="layout-topbar-logo"> </router-link>
        </template>

        <!--        <InputText placeholder="Search" type="text" class="w-32 sm:w-auto" />-->
        <template v-slot:right>
            <Avatar
                v-if="user.isLoggedIn"
                icon="pi pi-user"
                class="mr-2 ml-2"
                size="large"
                @click="toggleSecondaryMenu"
                style="background-color: #ece9fc; color: #2a1261; cursor: pointer"
            />
            <router-link v-if="!user.isLoggedIn" to="/login" class="layout-topbar-logo">
                <Button label="Login" icon="pi pi-sign-in" />
            </router-link>
        </template>
    </HorizontalLayout>


</template>
