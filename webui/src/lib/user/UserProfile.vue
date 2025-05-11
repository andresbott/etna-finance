<script setup>
import Avatar from 'primevue/avatar'
import Drawer from 'primevue/drawer'
import { ref } from 'vue'
import { useUserStore } from '@/lib/user/userstore.js'
import router from '@/router/index.js'
import UserProfileMenu from './UserProfileMenu.vue'

const user = useUserStore()
const visible = ref(false)

user.registerLogoutAction(() => {
    router.push({ path: '/', force: true })
})

const logOut = () => {
    user.logout()
}
</script>
<template>
    <Avatar
        icon="pi pi-user"
        class="mr-2 ml-2"
        size="large"
        @click="visible = true"
        style="background-color: #ece9fc; color: #2a1261; cursor: pointer"
    />

    <Drawer v-model:visible="visible" style="width: 25rem" position="right">
        <template #header>
            <div class="drawer-header">
                <Avatar
                    icon="pi pi-user"
                    size="large"
                    style="background-color: #ece9fc; color: #2a1261"
                />
                <span class="username">{{ user.loggedInUser }}</span>
            </div>
        </template>
        <UserProfileMenu @logout="logOut" />
    </Drawer>
</template>

<style scoped>
.drawer-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.5rem;
}

.username {
    font-size: 1.25rem;
    font-weight: 600;
}
</style>
