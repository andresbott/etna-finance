<script setup>
import { ref, computed, nextTick } from 'vue'
import Popover from 'primevue/popover'
import InputText from 'primevue/inputtext'
import IconField from 'primevue/iconfield'
import InputIcon from 'primevue/inputicon'

// Complete list of PrimeIcons
const ALL_ICONS = [
    // Financial & Commerce
    { name: 'wallet', class: 'pi pi-wallet' },
    { name: 'money-bill', class: 'pi pi-money-bill' },
    { name: 'credit-card', class: 'pi pi-credit-card' },
    { name: 'dollar', class: 'pi pi-dollar' },
    { name: 'euro', class: 'pi pi-euro' },
    { name: 'pound', class: 'pi pi-pound' },
    { name: 'indian-rupee', class: 'pi pi-indian-rupee' },
    { name: 'turkish-lira', class: 'pi pi-turkish-lira' },
    { name: 'bitcoin', class: 'pi pi-bitcoin' },
    { name: 'ethereum', class: 'pi pi-ethereum' },
    { name: 'percentage', class: 'pi pi-percentage' },
    { name: 'calculator', class: 'pi pi-calculator' },
    { name: 'receipt', class: 'pi pi-receipt' },
    { name: 'shopping-cart', class: 'pi pi-shopping-cart' },
    { name: 'shopping-bag', class: 'pi pi-shopping-bag' },
    { name: 'cart-plus', class: 'pi pi-cart-plus' },
    { name: 'cart-minus', class: 'pi pi-cart-minus' },
    { name: 'cart-arrow-down', class: 'pi pi-cart-arrow-down' },
    { name: 'shop', class: 'pi pi-shop' },
    { name: 'gift', class: 'pi pi-gift' },
    { name: 'ticket', class: 'pi pi-ticket' },
    { name: 'paypal', class: 'pi pi-paypal' },
    { name: 'amazon', class: 'pi pi-amazon' },
    
    // Buildings & Places
    { name: 'building', class: 'pi pi-building' },
    { name: 'building-columns', class: 'pi pi-building-columns' },
    { name: 'warehouse', class: 'pi pi-warehouse' },
    { name: 'home', class: 'pi pi-home' },
    { name: 'map', class: 'pi pi-map' },
    { name: 'map-marker', class: 'pi pi-map-marker' },
    { name: 'globe', class: 'pi pi-globe' },
    { name: 'compass', class: 'pi pi-compass' },
    
    // Charts & Data
    { name: 'chart-line', class: 'pi pi-chart-line' },
    { name: 'chart-bar', class: 'pi pi-chart-bar' },
    { name: 'chart-pie', class: 'pi pi-chart-pie' },
    { name: 'chart-scatter', class: 'pi pi-chart-scatter' },
    { name: 'gauge', class: 'pi pi-gauge' },
    { name: 'wave-pulse', class: 'pi pi-wave-pulse' },
    
    // People & Users
    { name: 'user', class: 'pi pi-user' },
    { name: 'users', class: 'pi pi-users' },
    { name: 'user-plus', class: 'pi pi-user-plus' },
    { name: 'user-minus', class: 'pi pi-user-minus' },
    { name: 'user-edit', class: 'pi pi-user-edit' },
    { name: 'id-card', class: 'pi pi-id-card' },
    { name: 'address-book', class: 'pi pi-address-book' },
    
    // Files & Folders
    { name: 'file', class: 'pi pi-file' },
    { name: 'file-o', class: 'pi pi-file-o' },
    { name: 'file-plus', class: 'pi pi-file-plus' },
    { name: 'file-check', class: 'pi pi-file-check' },
    { name: 'file-edit', class: 'pi pi-file-edit' },
    { name: 'file-excel', class: 'pi pi-file-excel' },
    { name: 'file-pdf', class: 'pi pi-file-pdf' },
    { name: 'file-word', class: 'pi pi-file-word' },
    { name: 'file-import', class: 'pi pi-file-import' },
    { name: 'file-export', class: 'pi pi-file-export' },
    { name: 'file-arrow-up', class: 'pi pi-file-arrow-up' },
    { name: 'folder', class: 'pi pi-folder' },
    { name: 'folder-open', class: 'pi pi-folder-open' },
    { name: 'folder-plus', class: 'pi pi-folder-plus' },
    { name: 'clipboard', class: 'pi pi-clipboard' },
    { name: 'copy', class: 'pi pi-copy' },
    { name: 'clone', class: 'pi pi-clone' },
    
    // Communication
    { name: 'envelope', class: 'pi pi-envelope' },
    { name: 'inbox', class: 'pi pi-inbox' },
    { name: 'send', class: 'pi pi-send' },
    { name: 'comment', class: 'pi pi-comment' },
    { name: 'comments', class: 'pi pi-comments' },
    { name: 'phone', class: 'pi pi-phone' },
    { name: 'bell', class: 'pi pi-bell' },
    { name: 'bell-slash', class: 'pi pi-bell-slash' },
    { name: 'megaphone', class: 'pi pi-megaphone' },
    { name: 'microphone', class: 'pi pi-microphone' },
    
    // Objects & Things
    { name: 'box', class: 'pi pi-box' },
    { name: 'briefcase', class: 'pi pi-briefcase' },
    { name: 'key', class: 'pi pi-key' },
    { name: 'lock', class: 'pi pi-lock' },
    { name: 'lock-open', class: 'pi pi-lock-open' },
    { name: 'unlock', class: 'pi pi-unlock' },
    { name: 'shield', class: 'pi pi-shield' },
    { name: 'verified', class: 'pi pi-verified' },
    { name: 'crown', class: 'pi pi-crown' },
    { name: 'trophy', class: 'pi pi-trophy' },
    { name: 'graduation-cap', class: 'pi pi-graduation-cap' },
    { name: 'lightbulb', class: 'pi pi-lightbulb' },
    { name: 'book', class: 'pi pi-book' },
    { name: 'paperclip', class: 'pi pi-paperclip' },
    { name: 'pencil', class: 'pi pi-pencil' },
    { name: 'pen-to-square', class: 'pi pi-pen-to-square' },
    { name: 'eraser', class: 'pi pi-eraser' },
    { name: 'hammer', class: 'pi pi-hammer' },
    { name: 'wrench', class: 'pi pi-wrench' },
    { name: 'palette', class: 'pi pi-palette' },
    { name: 'camera', class: 'pi pi-camera' },
    { name: 'image', class: 'pi pi-image' },
    { name: 'images', class: 'pi pi-images' },
    { name: 'video', class: 'pi pi-video' },
    { name: 'headphones', class: 'pi pi-headphones' },
    
    // Transportation
    { name: 'car', class: 'pi pi-car' },
    { name: 'truck', class: 'pi pi-truck' },
    
    // Technology
    { name: 'desktop', class: 'pi pi-desktop' },
    { name: 'mobile', class: 'pi pi-mobile' },
    { name: 'tablet', class: 'pi pi-tablet' },
    { name: 'server', class: 'pi pi-server' },
    { name: 'database', class: 'pi pi-database' },
    { name: 'cloud', class: 'pi pi-cloud' },
    { name: 'cloud-upload', class: 'pi pi-cloud-upload' },
    { name: 'cloud-download', class: 'pi pi-cloud-download' },
    { name: 'wifi', class: 'pi pi-wifi' },
    { name: 'qrcode', class: 'pi pi-qrcode' },
    { name: 'barcode', class: 'pi pi-barcode' },
    { name: 'microchip', class: 'pi pi-microchip' },
    { name: 'microchip-ai', class: 'pi pi-microchip-ai' },
    { name: 'code', class: 'pi pi-code' },
    { name: 'link', class: 'pi pi-link' },
    { name: 'sitemap', class: 'pi pi-sitemap' },
    
    // Time & Calendar
    { name: 'calendar', class: 'pi pi-calendar' },
    { name: 'calendar-plus', class: 'pi pi-calendar-plus' },
    { name: 'calendar-minus', class: 'pi pi-calendar-minus' },
    { name: 'calendar-times', class: 'pi pi-calendar-times' },
    { name: 'calendar-clock', class: 'pi pi-calendar-clock' },
    { name: 'clock', class: 'pi pi-clock' },
    { name: 'stopwatch', class: 'pi pi-stopwatch' },
    { name: 'hourglass', class: 'pi pi-hourglass' },
    { name: 'history', class: 'pi pi-history' },
    
    // Tags & Labels
    { name: 'tag', class: 'pi pi-tag' },
    { name: 'tags', class: 'pi pi-tags' },
    { name: 'bookmark', class: 'pi pi-bookmark' },
    { name: 'bookmark-fill', class: 'pi pi-bookmark-fill' },
    { name: 'hashtag', class: 'pi pi-hashtag' },
    { name: 'thumbtack', class: 'pi pi-thumbtack' },
    { name: 'flag', class: 'pi pi-flag' },
    { name: 'flag-fill', class: 'pi pi-flag-fill' },
    
    // Actions & Controls
    { name: 'cog', class: 'pi pi-cog' },
    { name: 'sliders-h', class: 'pi pi-sliders-h' },
    { name: 'sliders-v', class: 'pi pi-sliders-v' },
    { name: 'filter', class: 'pi pi-filter' },
    { name: 'filter-fill', class: 'pi pi-filter-fill' },
    { name: 'filter-slash', class: 'pi pi-filter-slash' },
    { name: 'search', class: 'pi pi-search' },
    { name: 'search-plus', class: 'pi pi-search-plus' },
    { name: 'search-minus', class: 'pi pi-search-minus' },
    { name: 'sync', class: 'pi pi-sync' },
    { name: 'refresh', class: 'pi pi-refresh' },
    { name: 'replay', class: 'pi pi-replay' },
    { name: 'undo', class: 'pi pi-undo' },
    { name: 'save', class: 'pi pi-save' },
    { name: 'print', class: 'pi pi-print' },
    { name: 'upload', class: 'pi pi-upload' },
    { name: 'download', class: 'pi pi-download' },
    { name: 'trash', class: 'pi pi-trash' },
    { name: 'delete-left', class: 'pi pi-delete-left' },
    { name: 'power-off', class: 'pi pi-power-off' },
    { name: 'sign-in', class: 'pi pi-sign-in' },
    { name: 'sign-out', class: 'pi pi-sign-out' },
    { name: 'external-link', class: 'pi pi-external-link' },
    { name: 'expand', class: 'pi pi-expand' },
    { name: 'window-maximize', class: 'pi pi-window-maximize' },
    { name: 'window-minimize', class: 'pi pi-window-minimize' },
    { name: 'share-alt', class: 'pi pi-share-alt' },
    { name: 'reply', class: 'pi pi-reply' },
    { name: 'eject', class: 'pi pi-eject' },
    
    // Status & Feedback
    { name: 'check', class: 'pi pi-check' },
    { name: 'check-circle', class: 'pi pi-check-circle' },
    { name: 'check-square', class: 'pi pi-check-square' },
    { name: 'list-check', class: 'pi pi-list-check' },
    { name: 'times', class: 'pi pi-times' },
    { name: 'times-circle', class: 'pi pi-times-circle' },
    { name: 'plus', class: 'pi pi-plus' },
    { name: 'plus-circle', class: 'pi pi-plus-circle' },
    { name: 'minus', class: 'pi pi-minus' },
    { name: 'minus-circle', class: 'pi pi-minus-circle' },
    { name: 'ban', class: 'pi pi-ban' },
    { name: 'exclamation-circle', class: 'pi pi-exclamation-circle' },
    { name: 'exclamation-triangle', class: 'pi pi-exclamation-triangle' },
    { name: 'question', class: 'pi pi-question' },
    { name: 'question-circle', class: 'pi pi-question-circle' },
    { name: 'info', class: 'pi pi-info' },
    { name: 'info-circle', class: 'pi pi-info-circle' },
    { name: 'spinner', class: 'pi pi-spinner' },
    { name: 'spinner-dotted', class: 'pi pi-spinner-dotted' },
    
    // Shapes & Symbols
    { name: 'star', class: 'pi pi-star' },
    { name: 'star-fill', class: 'pi pi-star-fill' },
    { name: 'star-half', class: 'pi pi-star-half' },
    { name: 'star-half-fill', class: 'pi pi-star-half-fill' },
    { name: 'heart', class: 'pi pi-heart' },
    { name: 'heart-fill', class: 'pi pi-heart-fill' },
    { name: 'circle', class: 'pi pi-circle' },
    { name: 'circle-fill', class: 'pi pi-circle-fill' },
    { name: 'circle-on', class: 'pi pi-circle-on' },
    { name: 'circle-off', class: 'pi pi-circle-off' },
    { name: 'bolt', class: 'pi pi-bolt' },
    { name: 'sparkles', class: 'pi pi-sparkles' },
    { name: 'sun', class: 'pi pi-sun' },
    { name: 'moon', class: 'pi pi-moon' },
    { name: 'face-smile', class: 'pi pi-face-smile' },
    { name: 'thumbs-up', class: 'pi pi-thumbs-up' },
    { name: 'thumbs-up-fill', class: 'pi pi-thumbs-up-fill' },
    { name: 'thumbs-down', class: 'pi pi-thumbs-down' },
    { name: 'thumbs-down-fill', class: 'pi pi-thumbs-down-fill' },
    { name: 'bullseye', class: 'pi pi-bullseye' },
    { name: 'at', class: 'pi pi-at' },
    { name: 'asterisk', class: 'pi pi-asterisk' },
    { name: 'equals', class: 'pi pi-equals' },
    { name: 'prime', class: 'pi pi-prime' },
    { name: 'venus', class: 'pi pi-venus' },
    { name: 'mars', class: 'pi pi-mars' },
    
    // Arrows & Direction
    { name: 'arrow-up', class: 'pi pi-arrow-up' },
    { name: 'arrow-down', class: 'pi pi-arrow-down' },
    { name: 'arrow-left', class: 'pi pi-arrow-left' },
    { name: 'arrow-right', class: 'pi pi-arrow-right' },
    { name: 'arrow-up-right', class: 'pi pi-arrow-up-right' },
    { name: 'arrow-up-left', class: 'pi pi-arrow-up-left' },
    { name: 'arrow-down-right', class: 'pi pi-arrow-down-right' },
    { name: 'arrow-down-left', class: 'pi pi-arrow-down-left' },
    { name: 'arrow-circle-up', class: 'pi pi-arrow-circle-up' },
    { name: 'arrow-circle-down', class: 'pi pi-arrow-circle-down' },
    { name: 'arrow-circle-left', class: 'pi pi-arrow-circle-left' },
    { name: 'arrow-circle-right', class: 'pi pi-arrow-circle-right' },
    { name: 'arrow-right-arrow-left', class: 'pi pi-arrow-right-arrow-left' },
    { name: 'arrows-h', class: 'pi pi-arrows-h' },
    { name: 'arrows-v', class: 'pi pi-arrows-v' },
    { name: 'arrows-alt', class: 'pi pi-arrows-alt' },
    { name: 'chevron-up', class: 'pi pi-chevron-up' },
    { name: 'chevron-down', class: 'pi pi-chevron-down' },
    { name: 'chevron-left', class: 'pi pi-chevron-left' },
    { name: 'chevron-right', class: 'pi pi-chevron-right' },
    { name: 'angle-up', class: 'pi pi-angle-up' },
    { name: 'angle-down', class: 'pi pi-angle-down' },
    { name: 'angle-left', class: 'pi pi-angle-left' },
    { name: 'angle-right', class: 'pi pi-angle-right' },
    { name: 'angle-double-up', class: 'pi pi-angle-double-up' },
    { name: 'angle-double-down', class: 'pi pi-angle-double-down' },
    { name: 'angle-double-left', class: 'pi pi-angle-double-left' },
    { name: 'angle-double-right', class: 'pi pi-angle-double-right' },
    { name: 'caret-up', class: 'pi pi-caret-up' },
    { name: 'caret-down', class: 'pi pi-caret-down' },
    { name: 'caret-left', class: 'pi pi-caret-left' },
    { name: 'caret-right', class: 'pi pi-caret-right' },
    { name: 'directions', class: 'pi pi-directions' },
    { name: 'directions-alt', class: 'pi pi-directions-alt' },
    
    // Layout & UI
    { name: 'bars', class: 'pi pi-bars' },
    { name: 'list', class: 'pi pi-list' },
    { name: 'th-large', class: 'pi pi-th-large' },
    { name: 'table', class: 'pi pi-table' },
    { name: 'objects-column', class: 'pi pi-objects-column' },
    { name: 'ellipsis-h', class: 'pi pi-ellipsis-h' },
    { name: 'ellipsis-v', class: 'pi pi-ellipsis-v' },
    { name: 'align-left', class: 'pi pi-align-left' },
    { name: 'align-center', class: 'pi pi-align-center' },
    { name: 'align-right', class: 'pi pi-align-right' },
    { name: 'align-justify', class: 'pi pi-align-justify' },
    
    // Sorting
    { name: 'sort', class: 'pi pi-sort' },
    { name: 'sort-up', class: 'pi pi-sort-up' },
    { name: 'sort-down', class: 'pi pi-sort-down' },
    { name: 'sort-up-fill', class: 'pi pi-sort-up-fill' },
    { name: 'sort-down-fill', class: 'pi pi-sort-down-fill' },
    { name: 'sort-alt', class: 'pi pi-sort-alt' },
    { name: 'sort-alt-slash', class: 'pi pi-sort-alt-slash' },
    { name: 'sort-alpha-up', class: 'pi pi-sort-alpha-up' },
    { name: 'sort-alpha-down', class: 'pi pi-sort-alpha-down' },
    { name: 'sort-alpha-up-alt', class: 'pi pi-sort-alpha-up-alt' },
    { name: 'sort-alpha-down-alt', class: 'pi pi-sort-alpha-down-alt' },
    { name: 'sort-numeric-up', class: 'pi pi-sort-numeric-up' },
    { name: 'sort-numeric-down', class: 'pi pi-sort-numeric-down' },
    { name: 'sort-numeric-up-alt', class: 'pi pi-sort-numeric-up-alt' },
    { name: 'sort-numeric-down-alt', class: 'pi pi-sort-numeric-down-alt' },
    { name: 'sort-amount-up', class: 'pi pi-sort-amount-up' },
    { name: 'sort-amount-down', class: 'pi pi-sort-amount-down' },
    { name: 'sort-amount-up-alt', class: 'pi pi-sort-amount-up-alt' },
    { name: 'sort-amount-down-alt', class: 'pi pi-sort-amount-down-alt' },
    
    // Media Controls
    { name: 'play', class: 'pi pi-play' },
    { name: 'play-circle', class: 'pi pi-play-circle' },
    { name: 'pause', class: 'pi pi-pause' },
    { name: 'pause-circle', class: 'pi pi-pause-circle' },
    { name: 'stop', class: 'pi pi-stop' },
    { name: 'stop-circle', class: 'pi pi-stop-circle' },
    { name: 'forward', class: 'pi pi-forward' },
    { name: 'backward', class: 'pi pi-backward' },
    { name: 'fast-forward', class: 'pi pi-fast-forward' },
    { name: 'fast-backward', class: 'pi pi-fast-backward' },
    { name: 'step-forward', class: 'pi pi-step-forward' },
    { name: 'step-backward', class: 'pi pi-step-backward' },
    { name: 'step-forward-alt', class: 'pi pi-step-forward-alt' },
    { name: 'step-backward-alt', class: 'pi pi-step-backward-alt' },
    { name: 'volume-up', class: 'pi pi-volume-up' },
    { name: 'volume-down', class: 'pi pi-volume-down' },
    { name: 'volume-off', class: 'pi pi-volume-off' },
    
    // Viewing
    { name: 'eye', class: 'pi pi-eye' },
    { name: 'eye-slash', class: 'pi pi-eye-slash' },
    
    // Social
    { name: 'facebook', class: 'pi pi-facebook' },
    { name: 'twitter', class: 'pi pi-twitter' },
    { name: 'instagram', class: 'pi pi-instagram' },
    { name: 'linkedin', class: 'pi pi-linkedin' },
    { name: 'youtube', class: 'pi pi-youtube' },
    { name: 'vimeo', class: 'pi pi-vimeo' },
    { name: 'github', class: 'pi pi-github' },
    { name: 'discord', class: 'pi pi-discord' },
    { name: 'slack', class: 'pi pi-slack' },
    { name: 'whatsapp', class: 'pi pi-whatsapp' },
    { name: 'telegram', class: 'pi pi-telegram' },
    { name: 'twitch', class: 'pi pi-twitch' },
    { name: 'tiktok', class: 'pi pi-tiktok' },
    { name: 'pinterest', class: 'pi pi-pinterest' },
    { name: 'reddit', class: 'pi pi-reddit' },
    { name: 'google', class: 'pi pi-google' },
    { name: 'apple', class: 'pi pi-apple' },
    { name: 'android', class: 'pi pi-android' },
    { name: 'microsoft', class: 'pi pi-microsoft' },
    
    // Language
    { name: 'language', class: 'pi pi-language' },
    
    // Chevron Circle
    { name: 'chevron-circle-up', class: 'pi pi-chevron-circle-up' },
    { name: 'chevron-circle-down', class: 'pi pi-chevron-circle-down' },
    { name: 'chevron-circle-left', class: 'pi pi-chevron-circle-left' },
    { name: 'chevron-circle-right', class: 'pi pi-chevron-circle-right' }
]

const props = defineProps({
    modelValue: { type: String, default: 'pi pi-wallet' },
    placeholder: { type: String, default: 'Select Icon' }
})

const emit = defineEmits(['update:modelValue'])

const popoverRef = ref(null)
const searchInputRef = ref(null)
const searchQuery = ref('')
const isOpen = ref(false)

// Find current icon name for display
const currentIconName = computed(() => {
    const icon = ALL_ICONS.find(i => i.class === props.modelValue)
    return icon ? icon.name : props.modelValue.replace('pi pi-', '')
})

// Filter icons based on search
const filteredIcons = computed(() => {
    if (!searchQuery.value) return ALL_ICONS
    const query = searchQuery.value.toLowerCase()
    return ALL_ICONS.filter(icon => icon.name.toLowerCase().includes(query))
})

const toggleDropdown = (event) => {
    popoverRef.value.toggle(event)
}

const onPopoverShow = () => {
    isOpen.value = true
    searchQuery.value = ''
    nextTick(() => {
        searchInputRef.value?.$el?.focus()
    })
}

const onPopoverHide = () => {
    isOpen.value = false
}

const selectIcon = (icon) => {
    emit('update:modelValue', icon.class)
    popoverRef.value.hide()
}
</script>

<template>
    <div class="icon-select">
        <!-- Trigger Button -->
        <button 
            type="button" 
            class="icon-select-trigger p-inputtext"
            :class="{ 'icon-select-trigger--open': isOpen }"
            @click="toggleDropdown"
        >
            <i :class="modelValue" class="selected-icon"></i>
            <span class="selected-name">{{ currentIconName }}</span>
            <i class="pi pi-chevron-down trigger-arrow" :class="{ 'trigger-arrow--open': isOpen }"></i>
        </button>

        <!-- Icon Picker Dropdown -->
        <Popover 
            ref="popoverRef"
            @show="onPopoverShow"
            @hide="onPopoverHide"
            class="icon-picker-popover"
        >
            <div class="icon-picker-content">
                <!-- Search Input -->
                <div class="icon-search">
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
                        :key="icon.name"
                        type="button"
                        class="icon-item"
                        :class="{ 'icon-item--selected': modelValue === icon.class }"
                        @click="selectIcon(icon)"
                        :title="icon.name"
                    >
                        <i :class="icon.class" class="icon-preview"></i>
                    </button>
                </div>

                <!-- No Results -->
                <div v-if="filteredIcons.length === 0" class="no-results">
                    <i class="pi pi-search"></i>
                    <p>No icons found for "{{ searchQuery }}"</p>
                </div>
            </div>
        </Popover>
    </div>
</template>

<style scoped>
.icon-select {
    width: 100%;
    position: relative;
}

/* Trigger inherits from p-inputtext, we just add flexbox layout */
.icon-select-trigger {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    cursor: pointer;
    text-align: left;
    line-height: 1.5rem;
}

.selected-icon {
    font-size: 1.25rem;
    color: var(--c-primary-color);
    flex-shrink: 0;
}

.selected-name {
    flex: 1;
    text-align: left;
    text-transform: capitalize;
}

.trigger-arrow {
    font-size: 0.75rem;
    opacity: 0.6;
    transition: transform 0.2s;
    flex-shrink: 0;
}

.trigger-arrow--open {
    transform: rotate(180deg);
}

/* Popover panel - inherit PrimeVue popover styling */
:deep(.p-popover) {
    padding: 0;
}

:deep(.p-popover-content) {
    padding: 0;
}

/* Popover Content Styles */
.icon-picker-content {
    width: 360px;
    max-width: calc(100vw - 2rem);
}

.icon-search {
    padding: 0.75rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--c-content-border-color);
}

.icons-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(44px, 1fr));
    gap: var(--c-inputtext-padding-x);
    max-height: 300px;
    overflow-y: scroll;
    overflow-x: hidden;
    padding: 1rem 0.75rem 0.75rem 0.75rem;
    scrollbar-width: thin;
    scrollbar-color: var(--c-surface-400) var(--c-surface-100);
}

.icons-grid::-webkit-scrollbar {
    width: 8px;
}

.icons-grid::-webkit-scrollbar-track {
    background: var(--c-surface-100);
    border-radius: var(--c-border-radius);
}

.icons-grid::-webkit-scrollbar-thumb {
    background: var(--c-surface-400);
    border-radius: var(--c-border-radius);
}

.icons-grid::-webkit-scrollbar-thumb:hover {
    background: var(--c-surface-500);
}

.icon-item {
    display: flex;
    align-items: center;
    justify-content: center;
    aspect-ratio: 1;
    padding: var(--c-inputtext-padding-y);
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--c-listbox-option-border-radius);
    cursor: pointer;
    transition: background var(--c-listbox-transition-duration), color var(--c-listbox-transition-duration), border-color var(--c-listbox-transition-duration);
    color: var(--c-listbox-option-color);
}

.icon-item:hover {
    background: var(--c-listbox-option-focus-background);
    color: var(--c-listbox-option-focus-color);
}

.icon-item--selected {
    background: var(--c-listbox-option-selected-background);
    color: var(--c-listbox-option-selected-color);
    border-color: var(--c-listbox-option-selected-color);
}

.icon-item--selected:hover {
    background: var(--c-listbox-option-selected-focus-background);
    color: var(--c-listbox-option-selected-focus-color);
    border-color: var(--c-listbox-option-selected-focus-color);
}

.icon-preview {
    font-size: 1.5rem;
}

.no-results {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--c-overlay-popover-padding);
    color: var(--c-text-muted-color);
}

.no-results i {
    font-size: 2rem;
    margin-bottom: var(--c-inputtext-padding-x);
    opacity: 0.5;
}

.no-results p {
    margin: 0;
}
</style>
