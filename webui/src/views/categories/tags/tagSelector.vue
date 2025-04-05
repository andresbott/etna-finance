<script setup>
import { ref,  } from 'vue'
import AutoComplete from 'primevue/autocomplete'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import InputGroup from 'primevue/inputgroup'
import InputGroupAddon from 'primevue/inputgroupaddon'
import { useTagStore } from '@/stores/tags.js'

const tagStore = useTagStore() // Initialize the store

const props = defineProps({
  modelValue: {
        type: Array, // Important if this is not initialized tags won't be visible when using the component
        default: () => []
    }
})
const emit = defineEmits(['update:modelValue'])



const searchTerm = ref('')
const filteredTags = ref([])

const updateTags = (newTags) => {
    emit('update:modelValue', newTags) // Only pass tag IDs
}

const searchTags = (event) => {
    const query = event.query.toLowerCase()
    filteredTags.value = tagStore.tagPaths
        .filter(
            (tag) => tag.path.toLowerCase().includes(query) && !props.modelValue?.includes(tag.id) // Ensure tag ID is not in selected list
        )
        .map((tag) => tag.path) // Store only the `path` value
}
const addTag = () => {
    const tag = searchTerm.value.trim()
    if (tag) {
        const existingTag = tagStore.tagPaths.find(
            (path) => path.path.toLowerCase() === tag.toLowerCase()
        )
        const tagToAdd = existingTag ? existingTag.id : null
        if (tagToAdd && !props.modelValue.includes(tagToAdd)) {
            updateTags([...(props.modelValue || []), tagToAdd])
        }
    }
    searchTerm.value = ''
    filteredTags.value = []
}

const removeTag = (tagItem) => {
    updateTags((props.modelValue || []).filter((id) => id !== tagItem))
}
</script>

<template>
    <div class="flex flex-column gap-2">
        <InputGroup>
            <InputGroupAddon>
                <i class="pi pi-tags"></i>
            </InputGroupAddon>
            <AutoComplete
                class="gap-2"
                v-model="searchTerm"
                :suggestions="filteredTags"
                @complete="searchTags"
                @keyup.enter="addTag"
                placeholder="Add a tag..."
            />
            <Button icon="pi pi-plus" @click="addTag" label="Add" outlined />
        </InputGroup>

        <div class="flex flex-wrap gap-2">
            <Tag
                v-for="TagId in modelValue"
                :key="TagId"
                :value="tagStore.tagPaths.find((tag) => tag.id === TagId)?.path"
                class="p-mr-2 tag-clickable"
                removable
                @click="removeTag(TagId)"
                severity="info"
                rounded
                icon="pi pi-times"
            />
        </div>
    </div>
</template>

<style scoped>
.p-mr-2 {
    margin-right: 0.5rem;
}

:deep(.tag-clickable) {
    cursor: pointer;
}
</style>
