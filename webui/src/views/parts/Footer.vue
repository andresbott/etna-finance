<script setup>
import { HorizontalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
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
