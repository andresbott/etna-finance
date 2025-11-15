import { ref, onMounted, onUnmounted } from 'vue'
import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', () => {
    const isDrawerVisible = ref(false)

    const openDrawer = () => {
        isDrawerVisible.value = true
    }

    const closeDrawer = () => {
        isDrawerVisible.value = false
    }

    const toggleDrawer = () => {
        isDrawerVisible.value = !isDrawerVisible.value
    }

    const checkScreenWidth = () => {
        if (window.innerWidth >= 1024) {
            openDrawer()
        } else {
            closeDrawer()
        }
    }

    const initUi = () => {
        checkScreenWidth()
        window.addEventListener('resize', checkScreenWidth)
    }

    const cleanupUi = () => {
        window.removeEventListener('resize', checkScreenWidth)
    }

    return {
        isDrawerVisible,
        openDrawer,
        closeDrawer,
        toggleDrawer,
        initUi,
        cleanupUi
    }
})
