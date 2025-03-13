<!--
## model
* collapsed (false): control the sidebar visibility status

## props
* :breakpoints="{ 1199: 'medium', 575: 'small' }"
allow to define a css class to be added to the main item
sidebar with can be controlled either with the variable     --ss-width: 300px;
or by overwriting the values with css

* :mobileWidth="450"
consider a mobile view if the device width is bellow this value

* :autoClose="true"
autoclose the menu if transitioning to mobile view

* :mobileOpen="false"
if initially the menu should load open on mobile view

-->

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
    breakpoints: {
        type: Object,
        default: () => ({})
    },
    mobileWidth: {
        type: Number,
        default: 450
    },
    autoClose: {
        type: Boolean,
        default: true
    },
    mobileOpen: {
        type: Boolean,
        default: false
    }
})

const sidebarOpen = defineModel('open')
// const toggleSidebar = () =>{
//     collapsed.value=!collapsed.value
// }
const closeSidebar = () => {
    sidebarOpen.value = false
}
const mobileView = ref(false)

// tell if view is mobile based on viewport with in relation to the auto-close width
function isMobile() {
    return props.mobileWidth >= getViewportWidth()
}

function getViewportWidth() {
    return window.innerWidth / window.devicePixelRatio
}

const sizeClass = ref('default')
let previousSize = null

function checkViewportSize() {
    // prevent resize animation
    document.body.classList.add('resize-animation-stopper')
    setTimeout(() => {
        document.body.classList.remove('resize-animation-stopper')
    }, 300)

    const width = getViewportWidth()
    mobileView.value = isMobile()

    // handle auto close on resize
    if (props.autoClose) {
        if (mobileView.value && sidebarOpen.value) {
            sidebarOpen.value = false
        }
        if (!mobileView.value && !sidebarOpen.value) {
            sidebarOpen.value = true
        }
    }

    // add css classes based on prop breakpoints
    let currentClass = null
    // Determine the current size based on breakpoints
    for (const [breakpoint, cssClass] of Object.entries(props.breakpoints)) {
        if (width < breakpoint) {
            currentClass = cssClass
            break
        }
    }
    if (!currentClass) {
        currentClass = 'default'
    }
    if (currentClass !== previousSize) {
        sizeClass.value = currentClass
        previousSize = currentClass
    }
}

// limit the amount of resize events happening
function throttle(callback, delay) {
    let isThrottled = false

    function wrapper() {
        if (isThrottled) {
            return
        }
        callback.apply(this, arguments)
        isThrottled = true

        // After the delay, allow the next call
        setTimeout(function () {
            isThrottled = false
        }, delay)
    }

    return wrapper
}

const throttledCheckViewportSize = throttle(checkViewportSize, 100)

// used to prevent animations when moutning the component
const isMounted = ref(false)
onMounted(() => {
    if (isMobile()) {
        sidebarOpen.value = props.mobileOpen
    }
    mobileView.value = isMobile()

    window.addEventListener('resize', throttledCheckViewportSize)
    // prevent animations to run on initial load
    setTimeout(() => {
        isMounted.value = true
    }, 20)
})

onBeforeUnmount(() => {
    // Clean up the event listener when the component is unmounted
    window.removeEventListener('resize', throttledCheckViewportSize)
})
</script>

<template>
    <div
        class="ss-wrapper"
        :class="[sizeClass, { open: sidebarOpen }, { mobile: mobileView }, { anim: isMounted }]"
    >
        <div class="ss-left" :class="{ anim: isMounted }">
            <slot name="menu"></slot>
        </div>
        <div class="ss-main" :class="{ anim: isMounted }">
            <slot></slot>
        </div>
        <div class="ss-overlay" @click="closeSidebar" :class="{ open: sidebarOpen }"></div>
    </div>
</template>

<style lang="scss">
:root {
    --ss-width: 300px;
}

.resize-animation-stopper * {
    animation: none !important;
    transition: none !important;
}

.ss-wrapper {
    .ss-left {
        width: var(--ss-width);
        position: fixed;
        top: 0;
        left: calc(var(--ss-width) * -1);
        height: 100%;
        z-index: 100;

        &.anim {
            transition: left 0.3s;
        }
    }

    &.open .ss-left {
        left: 0;
    }

    .ss-main {
        margin-left: 0;
        z-index: 0;

        &.anim {
            transition: margin-left 0.3s;
        }
    }

    &.open .ss-main {
        margin-left: var(--ss-width);
    }

    &.open.mobile .ss-main {
        margin-left: 0;
    }

    &.mobile .ss-overlay {
        z-index: 99;
        background: #424242;
        transition: opacity 0.3s;
        opacity: 0;

        cursor: pointer;
        &.open {
            opacity: 80%;
            transition: opacity 0.3s;
            position: fixed;
            top: 0;
            bottom: 0;
            left: 0;
            right: 0;
        }
    }
}
</style>
