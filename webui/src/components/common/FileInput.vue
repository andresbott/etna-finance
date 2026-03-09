<script setup>
import { ref, computed } from 'vue'
import Button from 'primevue/button'

const props = defineProps({
    accept: { type: String, default: '' },
    modelValue: { type: [File, null], default: null },
    label: { type: String, default: 'Choose file' },
    icon: { type: String, default: 'pi pi-upload' },
    disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const fileInputRef = ref(null)

const fileName = computed(() => props.modelValue?.name || null)

const triggerFileInput = () => {
    fileInputRef.value?.click()
}

const onFileChange = (event) => {
    const file = event.target.files?.[0] || null
    emit('update:modelValue', file)
    // Reset so the same file can be re-selected
    event.target.value = ''
}

const clearFile = () => {
    emit('update:modelValue', null)
}
</script>

<template>
    <div class="file-input-styled">
        <input
            ref="fileInputRef"
            type="file"
            :accept="accept"
            @change="onFileChange"
            class="file-input-hidden"
        />
        <div v-if="fileName" class="file-input-selected">
            <span class="file-input-name">
                <i class="pi pi-file"></i>
                {{ fileName }}
            </span>
            <Button
                icon="pi pi-times"
                text
                rounded
                severity="danger"
                size="small"
                @click="clearFile"
            />
        </div>
        <Button
            v-else
            :label="label"
            :icon="icon"
            severity="secondary"
            outlined
            :disabled="disabled"
            @click="triggerFileInput"
            class="file-input-button"
        />
    </div>
</template>

<style scoped>
.file-input-hidden {
    display: none;
}

.file-input-selected {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--p-content-border-color);
    border-radius: var(--p-border-radius);
    background: var(--p-content-background);
}

.file-input-name {
    flex: 1;
    font-size: 0.875rem;
    color: var(--p-text-color);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.file-input-button {
    width: 100%;
}
</style>
