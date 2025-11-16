<script setup>

import { onMounted, onUnmounted } from 'vue'
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import Topbar from './views/topbar.vue'
import Footer from './views/parts/Footer.vue'
import SidebarMenu from './components/SidebarMenu.vue'
import SecondaryMenu from './components/SecondaryMenu.vue'
import { useUiStore } from '@/store/uiStore.js'
import { useUserStore } from '@/lib/user/userstore.js'
import Toast from 'primevue/toast'
// import { VueQueryDevtools } from '@tanstack/vue-query-devtools'

const uiStore = useUiStore()
const user = useUserStore()

onMounted(() => {
    uiStore.initUi()
})

onUnmounted(() => {
    uiStore.cleanupUi()
})
</script>

<template>
    <Toast />
    <SecondaryMenu v-if="user.isLoggedIn" />
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <Topbar />
        </template>
        <template #default>
            <div class="content">
                <SidebarMenu v-if="user.isLoggedIn" />
                <router-view />
            </div>
        </template>
        <template #footer>
            <Footer />
        </template>
    </VerticalLayout>
<!--    <VueQueryDevtools  />-->
</template>

<style lang="css">
.content {
    position: relative;
    display: flex;
    height: 100%;
}
</style>
