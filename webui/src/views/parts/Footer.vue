<script setup>
import { HorizontalLayout } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import UserProfile from '@/lib/user/UserProfile.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import Button from 'primevue/button'
import Search from '@/views/bookmarks/search.vue'
import AppTitle from '@/views/parts/AppTitle.vue'
import AppSelector from '@/views/parts/AppSelector.vue'

const user = useUserStore()
</script>

<template>
    <HorizontalLayout class="footer" :centerContent="true">
        <template #left> Powered by.. TODO </template>

        <template #default> </template>

        <template #right> </template>
    </HorizontalLayout>
</template>

<style scoped>
.footer {
    background-color: var(--c-surface-200);
    color: var(--c-primary-500);
    font-size: 80%;
    padding: 0.2rem;
}
.c-horizontal {
    grid-template-columns: 1fr 2fr 1fr;
}
</style>
