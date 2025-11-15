<template>
    <transition name="slide">
        <div v-if="visible" class="fixed inset-0 z-50 flex" @click.self="close">
            <!-- Backdrop -->
            <div class="bg-black/30 absolute inset-0"></div>

            <!-- Drawer Panel -->
            <div
                :class="[
                    'relative bg-white shadow-xl h-full flex flex-col transform transition-transform duration-300 ease-in-out',
                    position === 'left'
                        ? 'w-64 translate-x-0'
                        : position === 'right'
                          ? 'w-64 translate-x-0 ml-auto'
                          : 'translate-x-0'
                ]"
                @click.stop
            >
                <header
                    v-if="header"
                    class="px-4 py-3 font-semibold border-b text-gray-700 flex justify-between items-center"
                >
                    <span>{{ header }}</span>
                    <button
                        @click="close"
                        class="pi pi-times text-gray-600 hover:text-gray-800"
                    ></button>
                </header>

                <!-- Drawer Content -->
                <div class="flex-1 overflow-y-auto">
                    <slot></slot>
                </div>
            </div>
        </div>
    </transition>
</template>

<script setup>
const props = defineProps({
    visible: Boolean,
    position: { type: String, default: 'left' },
    header: { type: String, default: '' }
})

const emit = defineEmits(['update:visible'])

const close = () => {
    emit('update:visible', false)
}
</script>

<style scoped>
.slide-enter-active,
.slide-leave-active {
    transition:
        opacity 0.3s ease,
        transform 0.3s ease;
}
.slide-enter-from,
.slide-leave-to {
    opacity: 0;
}
</style>
