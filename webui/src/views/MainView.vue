<script setup>
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import TopBar from '@/views/topbar.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import { onMounted, ref } from 'vue'
import Card from 'primevue/card'

const userStore = useUserStore()

// Add your data fetching and state management here
const loading = ref(false)

onMounted(() => {
    // Initialize your data here
})
</script>

<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <TopBar />
        </template>
        <template #default>
            <HorizontalLayout
                :fullHeight="true"
                :centerContent="true"
                :verticalCenterContent="false"
            >
                <template #default>
                    <Placeholder :width="'960px'" :height="'auto'">
                        <div class="grid">
                            <!-- Quick Actions Card -->
                            <div class="col-12 md:col-6 lg:col-4">
                                <Card>
                                    <template #title>Quick Actions</template>
                                    <template #content>
                                        <div class="flex flex-column gap-2">
                                            <Button label="View Accounts" icon="pi pi-wallet" severity="secondary" />
                                            <Button label="View Status" icon="pi pi-chart-line" severity="secondary" />
                                            <Button label="Playground" icon="pi pi-code" severity="secondary" />
                                        </div>
                                    </template>
                                </Card>
                            </div>

                            <!-- Recent Activity Card -->
                            <div class="col-12 md:col-6 lg:col-4">
                                <Card>
                                    <template #title>Recent Activity</template>
                                    <template #content>
                                        <p>Your recent activities will appear here</p>
                                    </template>
                                </Card>
                            </div>

                            <!-- System Status Card -->
                            <div class="col-12 md:col-6 lg:col-4">
                                <Card>
                                    <template #title>System Status</template>
                                    <template #content>
                                        <p>System status information will appear here</p>
                                    </template>
                                </Card>
                            </div>
                        </div>
                    </Placeholder>
                </template>
            </HorizontalLayout>
        </template>
        <template #footer>
            <Placeholder :width="'100%'" :height="30" :color="12">Footer</Placeholder>
        </template>
    </VerticalLayout>
</template>

<style scoped>
.grid {
    display: grid;
    gap: 1rem;
    padding: 1rem;
}

.card {
    height: 100%;
}
</style> 