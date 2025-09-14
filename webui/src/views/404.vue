<script setup>
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

import TopBar from '@/views/topbar.vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import { useUserStore } from '@/lib/user/userstore.js'

const user = useUserStore()
</script>

<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template v-if="user.isLoggedIn" #header>
            <TopBar />
        </template>
        <template #default>
            <HorizontalLayout
                :fullHeight="true"
                :centerContent="true"
                :verticalCenterContent="true"
            >
                <template #default>
                    <Placeholder :width="'960px'" :height="'auto'">
                        <Card>
                            <template #title>Page Not Found</template>
                            <template #content>
                                <div class="flex flex-column align-items-center gap-4">
                                    <i
                                        class="pi pi-exclamation-triangle"
                                        style="font-size: 4rem; color: var(--c-warning-500)"
                                    ></i>
                                    <h2>404 - Page Not Found</h2>
                                    <p>
                                        The page you are looking for does not exist or has been
                                        moved.
                                    </p>
                                    <Button
                                        label="Go to Home"
                                        icon="pi pi-home"
                                        @click="$router.push('/')"
                                    />
                                </div>
                            </template>
                        </Card>
                    </Placeholder>
                </template>
            </HorizontalLayout>
        </template>
        <template v-if="user.isLoggedIn" #footer>
            <Placeholder :width="'100%'" :height="30" :color="12">Footer</Placeholder>
        </template>
    </VerticalLayout>
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
