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
    background-color: #f5f5f5;
    color: #525252;
    border-top: 1px solid #e5e5e5;

    font-size: 80%;
    padding: 0.5rem;
}
.footer a {
    color: #525252;
    text-decoration: none;
    margin-right: 0.5rem;
}
.footer a:hover {
    color: var(--c-primary-600);
}
.c-horizontal {
    grid-template-columns: 1fr 2fr 1fr;
}
</style>
