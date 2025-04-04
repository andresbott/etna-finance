<script setup>
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'

const props = defineProps({
    name: {
        type: String,
        default: ''
    },
    title: {
        type: String,
        default: 'Confirm Deletion'
    },
    message: {
        type: String,
        default: 'Are you sure you want to delete this item?'
    },
    onConfirm: {
        type: Function,
        required: true
    }
})

const visible = defineModel('visible', { default: false })

async function handleConfirm() {
    try {
        await props.onConfirm()
        visible.value = false
    } catch (error) {
        console.error('Failed to delete item:', error)
        // You might want to show an error message to the user here
    }
}
</script>

<template>
    <Dialog
        v-model:visible="visible"
        modal
        :closable="true"
        :draggable="false"
        :header="title"
        :style="{ width: '50vw' }"
        @keydown.enter="handleConfirm"
    >
        <span class="block mb-4">{{ message }} {{ name ? `"${name}"` : '' }}</span>
        <div class="flex justify-content-end gap-3">
            <Button type="button" label="Ok" icon="pi pi-check" @click="handleConfirm"></Button>
            <Button
                type="button"
                label="Cancel"
                icon="pi pi-times"
                severity="secondary"
                @click="visible = false"
            ></Button>
        </div>
    </Dialog>
</template>

<style></style>
