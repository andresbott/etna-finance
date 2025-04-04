<script setup>
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'

import Topbar from '@/views/topbar.vue'

defineProps({
    type: {
        type: String,
        required: true
    }
})
</script>
<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <topbar />
        </template>
        <template #default>
            <HorizontalLayout :fullHeight="true" :contentMaxWidth="'100%'" :centerContent="true">
                <template #default>
                    <HorizontalLayout
                        :fullHeight="true"
                        :contentMaxWidth="'100%'"
                        :leftMinWidth="'300px'"
                        :containerMaxWidth="'960px'"
                    >
                        <template #left>
                            <Placeholder :width="'100%'" :height="'100%'">sidebar</Placeholder>
                        </template>
                        <template #default>
                            <Placeholder :width="'100%'" :height="1000"
                                >Main App content
                            </Placeholder>
                            <!--                                <Placeholder :width="'100%'" :height="'100%'"  >Main App content </Placeholder>-->
                        </template>
                    </HorizontalLayout>
                </template>
            </HorizontalLayout>
        </template>
        <template #footer>
            <Placeholder :width="'100%'" :height="30" :colorIndex="12">Footer</Placeholder>
        </template>
    </VerticalLayout>
</template>
