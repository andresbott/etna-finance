<script setup lang="ts">
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import Topbar from './views/topbar.vue'
import Footer from './views/parts/Footer.vue'
</script>

<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <Topbar />
        </template>
        <template #default>
            <router-view />
        </template>
        <template #footer>
            <Footer />
        </template>
    </VerticalLayout>
</template>
