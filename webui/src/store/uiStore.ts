import { ref, onMounted, onUnmounted } from 'vue'
import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', () => {
    // Main sidebar drawer
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

    // Secondary menu drawer (user menu)
    const isSecondaryDrawerVisible = ref(false)

    const openSecondaryDrawer = () => {
        isSecondaryDrawerVisible.value = true
    }

    const closeSecondaryDrawer = () => {
        isSecondaryDrawerVisible.value = false
    }

    const toggleSecondaryDrawer = () => {
        isSecondaryDrawerVisible.value = !isSecondaryDrawerVisible.value
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
        isSecondaryDrawerVisible,
        openSecondaryDrawer,
        closeSecondaryDrawer,
        toggleSecondaryDrawer,
        initUi,
        cleanupUi
    }
})
