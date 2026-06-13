<script setup>
import { HorizontalLayout, Placeholder } from '@/components/layout'

import Card from 'primevue/card'
import Button from 'primevue/button'
</script>

<template>
    <HorizontalLayout :fullHeight="true" :centerContent="true" :verticalCenterContent="true">
        <template #default>
            <Placeholder :width="'960px'" :height="'auto'">
                <Card>
                    <template #title>Page Not Found</template>
                    <template #content>
                        <div class="flex flex-column align-items-center gap-4">
                            <i
                                class="ti ti-alert-triangle"
                                style="font-size: 4rem; color: var(--c-warning-500)"
                            ></i>
                            <h2>404 - Page Not Found</h2>
                            <p>The page you are looking for does not exist or has been moved.</p>
                            <Button
                                label="Go to Home"
                                icon="ti ti-home"
                                @click="$router.push('/')"
                            />
                        </div>
                    </template>
                </Card>
            </Placeholder>
        </template>
    </HorizontalLayout>
</template>

<style scoped>
.flex {
    display: flex;
}
.flex-column {
    flex-direction: column;
}
.align-items-center {
    align-items: center;
}
.gap-4 {
    gap: 1rem;
}
</style>
