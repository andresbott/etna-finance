<script setup>
import { ref, computed, nextTick } from 'vue'
import Popover from 'primevue/popover'
import InputText from 'primevue/inputtext'
import IconField from 'primevue/iconfield'
import InputIcon from 'primevue/inputicon'
import { TABLER_ICONS } from '@/utils/tablerIcons'

const props = defineProps({
    modelValue: { type: String, default: 'wallet' },
    placeholder: { type: String, default: 'Select Icon' }
})

const emit = defineEmits(['update:modelValue'])

const popoverRef = ref(null)
const triggerRef = ref(null)
const searchInputRef = ref(null)
const searchQuery = ref('')
const isOpen = ref(false)
const popoverWidth = ref('360px')

const currentIconName = computed(() => props.modelValue)

const filteredIcons = computed(() => {
    if (!searchQuery.value) return TABLER_ICONS
    const query = searchQuery.value.toLowerCase()
    return TABLER_ICONS.filter(icon => icon.toLowerCase().includes(query))
})

const toggleDropdown = (event) => {
    popoverRef.value.toggle(event)
}

const onPopoverShow = () => {
    isOpen.value = true
    searchQuery.value = ''
    if (triggerRef.value) {
        popoverWidth.value = `${triggerRef.value.offsetWidth}px`
    }
    nextTick(() => {
        searchInputRef.value?.$el?.focus()
    })
}

const onPopoverHide = () => {
    isOpen.value = false
}

const selectIcon = (icon) => {
    emit('update:modelValue', icon)
    popoverRef.value.hide()
}
</script>

<template>
    <div class="icon-select">
        <button
            ref="triggerRef"
            type="button"
            class="icon-select-trigger p-inputtext flex align-items-center gap-3 w-full cursor-pointer text-left"
            :class="{ 'icon-select-trigger--open': isOpen }"
            @click="toggleDropdown"
        >
            <i :class="['ti', `ti-${modelValue}`]" class="text-xl text-primary flex-shrink-0"></i>
            <span class="flex-1 capitalize">{{ currentIconName }}</span>
            <i class="ti ti-chevron-down text-xs opacity-60 flex-shrink-0" :class="{ 'rotate-180': isOpen }" style="transition: transform 0.2s"></i>
        </button>

        <Popover
            ref="popoverRef"
            @show="onPopoverShow"
            @hide="onPopoverHide"
            class="icon-select-popover"
        >
            <div class="icon-picker-content" :style="{ width: popoverWidth }">
                <div class="p-3 pb-4">
                    <IconField>
                        <InputIcon class="ti ti-search" />
                        <InputText
                            ref="searchInputRef"
                            v-model="searchQuery"
                            placeholder="Search icons..."
                            fluid
                        />
                    </IconField>
                </div>

                <div class="icons-grid">
                    <button
                        v-for="icon in filteredIcons"
                        :key="icon"
                        type="button"
                        class="icon-item"
                        :class="{ 'icon-item--selected': modelValue === icon }"
                        @click="selectIcon(icon)"
                        :title="icon"
                    >
                        <i :class="['ti', `ti-${icon}`]" class="text-2xl"></i>
                    </button>
                </div>

                <div v-if="filteredIcons.length === 0" class="flex flex-column align-items-center justify-content-center p-4 text-color-secondary">
                    <i class="ti ti-search text-4xl mb-3 opacity-50"></i>
                    <p class="m-0">No icons found for "{{ searchQuery }}"</p>
                </div>
            </div>
        </Popover>
    </div>
</template>

<style>
.icon-select-trigger {
    line-height: 1.5rem;
}

.icon-select-popover.p-popover,
.icon-select-popover .p-popover-content {
    padding: 0;
}

.icon-select-popover .icon-picker-content {
    min-width: 280px;
    max-width: calc(100vw - 2rem);
}

.icon-select-popover .icons-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(44px, 1fr));
    gap: 0.5rem;
    max-height: 300px;
    overflow-y: auto;
    padding: 1rem;
}

.icon-select-popover .icon-item {
    display: flex;
    align-items: center;
    justify-content: center;
    aspect-ratio: 1;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--c-border-radius);
    cursor: pointer;
    color: var(--c-text-color);
}

.icon-select-popover .icon-item:hover {
    background: var(--c-surface-100);
}

.icon-select-popover .icon-item--selected {
    background: var(--c-highlight-background);
    color: var(--c-highlight-color);
    border-color: var(--c-highlight-color);
}
</style>
