<script setup>
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Message from 'primevue/message'

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
    error: {
        type: String,
        default: null
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
        <Message v-if="props.error" severity="error" :closable="false" class="mb-2">{{ props.error }}</Message>
        <div class="flex justify-content-end gap-3">
            <Button type="button" label="Ok" icon="ti ti-check" @click="handleConfirm"></Button>
            <Button
                type="button"
                label="Cancel"
                icon="ti ti-x"
                severity="secondary"
                @click="visible = false"
            ></Button>
        </div>
    </Dialog>
</template>

<style></style>
