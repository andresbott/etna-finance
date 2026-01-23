<script setup>
import { ref, computed, nextTick } from 'vue'
import Popover from 'primevue/popover'
import InputText from 'primevue/inputtext'
import IconField from 'primevue/iconfield'
import InputIcon from 'primevue/inputicon'

// Complete list of PrimeIcons (just the icon names without 'pi-' prefix)
const ALL_ICONS = [
    // Financial & Commerce
    'wallet', 'money-bill', 'credit-card', 'dollar', 'euro', 'pound',
    'indian-rupee', 'turkish-lira', 'bitcoin', 'ethereum', 'percentage',
    'calculator', 'receipt', 'shopping-cart', 'shopping-bag', 'cart-plus',
    'cart-minus', 'cart-arrow-down', 'shop', 'gift', 'ticket', 'paypal', 'amazon',
    
    // Buildings & Places
    'building', 'building-columns', 'warehouse', 'home', 'map', 'map-marker',
    'globe', 'compass',
    
    // Charts & Data
    'chart-line', 'chart-bar', 'chart-pie', 'chart-scatter', 'gauge', 'wave-pulse',
    
    // People & Users
    'user', 'users', 'user-plus', 'user-minus', 'user-edit', 'id-card', 'address-book',
    
    // Files & Folders
    'file', 'file-o', 'file-plus', 'file-check', 'file-edit', 'file-excel',
    'file-pdf', 'file-word', 'file-import', 'file-export', 'file-arrow-up',
    'folder', 'folder-open', 'folder-plus', 'clipboard', 'copy', 'clone',
    
    // Communication
    'envelope', 'inbox', 'send', 'comment', 'comments', 'phone', 'bell',
    'bell-slash', 'megaphone', 'microphone',
    
    // Objects & Things
    'box', 'briefcase', 'key', 'lock', 'lock-open', 'unlock', 'shield',
    'verified', 'crown', 'trophy', 'graduation-cap', 'lightbulb', 'book',
    'paperclip', 'pencil', 'pen-to-square', 'eraser', 'hammer', 'wrench',
    'palette', 'camera', 'image', 'images', 'video', 'headphones',
    
    // Transportation
    'car', 'truck',
    
    // Technology
    'desktop', 'mobile', 'tablet', 'server', 'database', 'cloud',
    'cloud-upload', 'cloud-download', 'wifi', 'qrcode', 'barcode',
    'microchip', 'microchip-ai', 'code', 'link', 'sitemap',
    
    // Time & Calendar
    'calendar', 'calendar-plus', 'calendar-minus', 'calendar-times',
    'calendar-clock', 'clock', 'stopwatch', 'hourglass', 'history',
    
    // Tags & Labels
    'tag', 'tags', 'bookmark', 'bookmark-fill', 'hashtag', 'thumbtack',
    'flag', 'flag-fill',
    
    // Actions & Controls
    'cog', 'sliders-h', 'sliders-v', 'filter', 'filter-fill', 'filter-slash',
    'search', 'search-plus', 'search-minus', 'sync', 'refresh', 'replay',
    'undo', 'save', 'print', 'upload', 'download', 'trash', 'delete-left',
    'power-off', 'sign-in', 'sign-out', 'external-link', 'expand',
    'window-maximize', 'window-minimize', 'share-alt', 'reply', 'eject',
    
    // Status & Feedback
    'check', 'check-circle', 'check-square', 'list-check', 'times',
    'times-circle', 'plus', 'plus-circle', 'minus', 'minus-circle', 'ban',
    'exclamation-circle', 'exclamation-triangle', 'question', 'question-circle',
    'info', 'info-circle', 'spinner', 'spinner-dotted',
    
    // Shapes & Symbols
    'star', 'star-fill', 'star-half', 'star-half-fill', 'heart', 'heart-fill',
    'circle', 'circle-fill', 'circle-on', 'circle-off', 'bolt', 'sparkles',
    'sun', 'moon', 'face-smile', 'thumbs-up', 'thumbs-up-fill', 'thumbs-down',
    'thumbs-down-fill', 'bullseye', 'at', 'asterisk', 'equals', 'prime',
    'venus', 'mars',
    
    // Arrows & Direction
    'arrow-up', 'arrow-down', 'arrow-left', 'arrow-right', 'arrow-up-right',
    'arrow-up-left', 'arrow-down-right', 'arrow-down-left', 'arrow-circle-up',
    'arrow-circle-down', 'arrow-circle-left', 'arrow-circle-right',
    'arrow-right-arrow-left', 'arrows-h', 'arrows-v', 'arrows-alt',
    'chevron-up', 'chevron-down', 'chevron-left', 'chevron-right',
    'angle-up', 'angle-down', 'angle-left', 'angle-right',
    'angle-double-up', 'angle-double-down', 'angle-double-left', 'angle-double-right',
    'caret-up', 'caret-down', 'caret-left', 'caret-right', 'directions', 'directions-alt',
    
    // Layout & UI
    'bars', 'list', 'th-large', 'table', 'objects-column', 'ellipsis-h',
    'ellipsis-v', 'align-left', 'align-center', 'align-right', 'align-justify',
    
    // Sorting
    'sort', 'sort-up', 'sort-down', 'sort-up-fill', 'sort-down-fill',
    'sort-alt', 'sort-alt-slash', 'sort-alpha-up', 'sort-alpha-down',
    'sort-alpha-up-alt', 'sort-alpha-down-alt', 'sort-numeric-up',
    'sort-numeric-down', 'sort-numeric-up-alt', 'sort-numeric-down-alt',
    'sort-amount-up', 'sort-amount-down', 'sort-amount-up-alt', 'sort-amount-down-alt',
    
    // Media Controls
    'play', 'play-circle', 'pause', 'pause-circle', 'stop', 'stop-circle',
    'forward', 'backward', 'fast-forward', 'fast-backward', 'step-forward',
    'step-backward', 'step-forward-alt', 'step-backward-alt', 'volume-up',
    'volume-down', 'volume-off',
    
    // Viewing
    'eye', 'eye-slash',
    
    // Social
    'facebook', 'twitter', 'instagram', 'linkedin', 'youtube', 'vimeo',
    'github', 'discord', 'slack', 'whatsapp', 'telegram', 'twitch',
    'tiktok', 'pinterest', 'reddit', 'google', 'apple', 'android', 'microsoft',
    
    // Language
    'language',
    
    // Chevron Circle
    'chevron-circle-up', 'chevron-circle-down', 'chevron-circle-left', 'chevron-circle-right'
]

const props = defineProps({
    modelValue: { type: String, default: 'pi-wallet' },
    placeholder: { type: String, default: 'Select Icon' }
})

const emit = defineEmits(['update:modelValue'])

const popoverRef = ref(null)
const triggerRef = ref(null)
const searchInputRef = ref(null)
const searchQuery = ref('')
const isOpen = ref(false)
const popoverWidth = ref('360px')

// Find current icon name for display
const currentIconName = computed(() => {
    return props.modelValue.replace('pi-', '')
})

// Filter icons based on search
const filteredIcons = computed(() => {
    if (!searchQuery.value) return ALL_ICONS
    const query = searchQuery.value.toLowerCase()
    return ALL_ICONS.filter(icon => icon.toLowerCase().includes(query))
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
    emit('update:modelValue', `pi-${icon}`)
    popoverRef.value.hide()
}
</script>

<template>
    <div class="icon-select">
        <!-- Trigger Button -->
        <button 
            ref="triggerRef"
            type="button" 
            class="icon-select-trigger p-inputtext flex align-items-center gap-3 w-full cursor-pointer text-left"
            :class="{ 'icon-select-trigger--open': isOpen }"
            @click="toggleDropdown"
        >
            <i :class="['pi', modelValue]" class="text-xl text-primary flex-shrink-0"></i>
            <span class="flex-1 capitalize">{{ currentIconName }}</span>
            <i class="pi pi-chevron-down text-xs opacity-60 flex-shrink-0" :class="{ 'rotate-180': isOpen }" style="transition: transform 0.2s"></i>
        </button>

        <!-- Icon Picker Dropdown -->
        <Popover 
            ref="popoverRef"
            @show="onPopoverShow"
            @hide="onPopoverHide"
            class="icon-select-popover"
        >
            <div class="icon-picker-content" :style="{ width: popoverWidth }">
                <!-- Search Input -->
                <div class="p-3 pb-4">
                    <IconField>
                        <InputIcon class="pi pi-search" />
                        <InputText 
                            ref="searchInputRef"
                            v-model="searchQuery" 
                            placeholder="Search icons..." 
                            fluid
                        />
                    </IconField>
                </div>

                <!-- Icons Grid -->
                <div class="icons-grid">
                    <button
                        v-for="icon in filteredIcons"
                        :key="icon"
                        type="button"
                        class="icon-item"
                        :class="{ 'icon-item--selected': modelValue === `pi-${icon}` }"
                        @click="selectIcon(icon)"
                        :title="icon"
                    >
                        <i :class="['pi', `pi-${icon}`]" class="text-2xl"></i>
                    </button>
                </div>

                <!-- No Results -->
                <div v-if="filteredIcons.length === 0" class="flex flex-column align-items-center justify-content-center p-4 text-color-secondary">
                    <i class="pi pi-search text-4xl mb-3 opacity-50"></i>
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
