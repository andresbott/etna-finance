<script setup>
// import { VerticalLayout, HorizontalLayout } from '@go-bumbu/vue-components/layout'
import { VerticalLayout, HorizontalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
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
