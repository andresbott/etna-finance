<script setup>
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { useBookmarkStore } from '@/stores/bookmark.js'
import { useTagStore } from '@/stores/tags.js'

const props = defineProps({
    itemId: {
        type: Number,
        default: 0
    },
    name: {
        type: String,
        default: ''
    }
})

const tagStore = useTagStore()

function detenteTag(id) {
    tagStore.Delete(id)
    visible.value = false
}

const visible = ref(false)
</script>

<template>
    <div>
        <Button
            label=""
            severity="danger"
            variant="text"
            icon="pi pi-trash"
            @click="visible = true"
        />

        <Dialog
            v-model:visible="visible"
            modal
            :closable="true"
            :draggable="false"
            header="Confirm Deletion"
            :style="{ width: '50vw' }"
            @keydown.enter="detenteTag(itemId)"
        >
            <span class="block mb-4">Are you sure you want to bookmark: "{{ props.name }}"</span>
            <div class="flex justify-content-end gap-3">
                <Button
                    type="button"
                    label="Ok"
                    icon="pi pi-check"
                    @click="detenteTag(itemId)"
                ></Button>
                <Button
                    type="button"
                    label="Cancel"
                    icon="pi pi-times"
                    severity="secondary"
                    @click="visible = false"
                ></Button>
            </div>
        </Dialog>
    </div>
</template>

<style></style>
