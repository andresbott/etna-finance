<script setup>
import Horizontal from '@/lib/layout/horizontal.vue'
import Vertical from '@/lib/layout/Vertical.vue'

import { useUserStore } from '@/lib/user/userstore.js'

const user = useUserStore()
</script>

<template>
    <vertical :center-content="false">
        <template v-slot:header> header </template>

        <template v-slot:main>
            <horizontal :center-content="true">
                <template v-slot:default>
                    <router-view />
                </template>
            </horizontal>
        </template>

        <template v-slot:footer>my footer </template>
    </vertical>
</template>
