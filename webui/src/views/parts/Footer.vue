<script setup>
import { HorizontalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
</script>

<template>
    <HorizontalLayout class="footer" :centerContent="true">
        <template #left> </template>

        <template #default> Powered by <a href="https://github.com/andresbott/etna-finance" target="_blank">https://github.com/andresbott/etna-finance</a></template>

        <template #right> </template>
    </HorizontalLayout>
</template>

<style>
.footer {
    background-color: var(--c-primary-600);
    color: var(--c-primary-100);

    font-size: 80%;
    padding: 0.5rem;
}
.footer a {
    color: var(--c-primary-100);
    text-decoration: none;
    margin-right: 0.5rem;
}
.c-horizontal {
    grid-template-columns: 1fr 2fr 1fr;
}
</style>
