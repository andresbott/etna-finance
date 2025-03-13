<script setup>
import Horizontal from '@/lib/layout/horizontal.vue'
import Vertical from '@/lib/layout/Vertical.vue'
import TopBar from '@/views/bookmarks/topbar.vue'
import { useUserStore } from '@/lib/user/userstore.js'
import BookmarkList from '@/views/bookmarks/bookmarkList.vue'
import Tree from 'primevue/tree'
import Footer from '@/views/parts/Footer.vue'

const user = useUserStore()

import { ref, computed } from 'vue'

const nodes = ref([
    {
        key: 'a',
        label: 'label a',
        icon: 'pi pi-fw pi-chevron-up ',
        children: [
            {
                key: 'b',
                label: 'ba',
                icon: 'pi pi-fw pi-chevron-right '
            },
            {
                key: 'c',
                label: 'c',
                icon: 'pi pi-fw pi-chevron-right '
            }
        ]
    },
    {
        key: 'd',
        label: 'label d',
        icon: 'pi pi-fw pi-chevron-up ',
        children: [
            {
                key: 'e',
                label: 'e',
                icon: 'pi pi-fw pi-chevron-right '
            },
            {
                key: 'f',
                label: 'f',
                icon: 'pi pi-fw pi-chevron-right '
            }
        ]
    }
])
</script>

<template>
    <vertical :center-content="false">
        <template #header>
            <TopBar />
        </template>

        <template #default>
            <horizontal :horizontal-center="true" :stretch-content="true" :max-width="'60rem'">
                <template #left>
                    <div>
                        <Tree :value="nodes" class="w-full md:w-[30rem]"></Tree>
                    </div>
                </template>
                <template #default>
                    <bookmark-list />
                </template>
            </horizontal>
        </template>

        <template #footer>
            <Footer />
        </template>
    </vertical>
</template>
