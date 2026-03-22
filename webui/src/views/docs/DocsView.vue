<script setup lang="ts">
import { RouterView } from 'vue-router'
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import DocsNavSidebar from '@/components/docs/DocsNavSidebar.vue'

const leftSidebarCollapsed = ref(true)
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="grid">
                    <!-- Left panel -->
                    <div class="col-12 md:col-3">
                        <DocsNavSidebar />
                    </div>

                    <!-- Right content panel -->
                    <div class="col-12 md:col-9">
                        <RouterView />
                    </div>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>
