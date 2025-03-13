<template>
    <div class="c-horizontal" :class="[{ 'v-cent': verticalCenter }]">
        <div class="c-left">
            <slot name="left"></slot>
        </div>
        <div class="c-main">
            <div
                :class="[{ 'h-cent': horizontalCenter }, { 'h-stretch': stretchContent }]"
                :style="maxWidth ? { maxWidth: maxWidth } : null"
            >
                <slot></slot>
            </div>
        </div>

        <div class="c-right">
            <slot name="right"></slot>
        </div>
    </div>
</template>

<script setup>
defineProps({
    horizontalCenter: {
        type: Boolean,
        default: false
    },
    verticalCenter: {
        type: Boolean,
        default: false
    },
    stretchContent: {
        type: Boolean,
        default: false
    },
    maxWidth: {
        type: [Number, String],
        default: null
    }
})
</script>

<style>
.c-horizontal {
    width: 100%;
    height: 100%;
    display: grid;
    grid-template-columns: auto 1fr auto;
    place-items: start normal;
}
.c-horizontal.v-cent {
    width: 100%;
    height: 100%;
    display: grid;
    grid-template-columns: auto 1fr auto;
    place-items: center normal;
}

.c-main {
    width: 100%;
    position: relative;
    display: grid;
    place-items: start;
}
.c-main .h-cent {
    margin: auto;
}
.c-main .h-stretch {
    width: 100%;
}
.c-right {
    width: 100%;
    height: 100%;
    display: grid;
    grid-auto-flow: column;
    justify-content: end;
}
.c-left {
    width: 100%;
    height: 100%;
    display: grid;
    grid-auto-flow: column;
    justify-content: start;
}
</style>
