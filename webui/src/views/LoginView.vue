<script setup>
import { VerticalLayout, HorizontalLayout } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import UserLogin from '@/lib/user/UserLogin.vue'
</script>

<template>
    <VerticalLayout :centerContent="true">
        <template #default>
            <HorizontalLayout :verticalCenterContent="true" :centerContent="true">
                <template #default>
                    <UserLogin />
                </template>
            </HorizontalLayout>
        </template>
        <template #footer> </template>
    </VerticalLayout>
</template>
