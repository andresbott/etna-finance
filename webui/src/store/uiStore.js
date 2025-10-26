import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', () => {
    const isDrawerVisible = ref(false)

    const openDrawer = () => {
        isDrawerVisible.value = true
    }

    const closeDrawer = () => {
        isDrawerVisible.value = false
    }

    return { isDrawerVisible, openDrawer, closeDrawer }
})
