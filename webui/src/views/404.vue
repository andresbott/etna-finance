<script setup>
import Vertical from '@/lib/layout/Vertical.vue'
</script>

<template>
    <vertical>
        <template v-slot:header> head </template>

        404

        <template v-slot:footer> footer </template>
    </vertical>
</template>

<style>
@media (min-width: 1024px) {
    .about {
        min-height: 100vh;
        display: flex;
        align-items: center;
    }
}
</style>
