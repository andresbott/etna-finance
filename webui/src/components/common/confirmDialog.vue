<script setup>
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'

defineProps({
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
    }
})

const visible = defineModel('visible', { default: false })
const emit = defineEmits(['confirm'])

function handleConfirm() {
    emit('confirm')
}
</script>

<template>
    <Dialog
        v-model:visible="visible"
        modal
        :closable="true"
        :draggable="false"
        :header="title"
        class="entry-dialog"
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
