<script setup>
import { PrimeIcons } from '@primevue/core/api'

defineProps({
    icon: {
        type: String,
        required: true
    },
    text: {
        type: String,
        default: ''
    }
})
</script>

<template>
    <div class="app-icon-text">
        <i :class="icon" class="pi app-icon"></i>
        <span v-if="text" class="app-text">{{ text }}</span>
    </div>
</template>

<style scoped>
.app-icon-text {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-family: Arial, sans-serif;
    font-size: 1rem;
    color: var(--c-text-color);
}

.app-icon {
    font-size: 1.5rem;
    color: var(--c-primary-50);
}

.app-text {
    font-size: 150%;
    //font-weight: bold;
    color: var(--c-primary-50);
}
</style>
