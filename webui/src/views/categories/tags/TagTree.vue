<script setup>
import Tree from 'primevue/tree'
import {ref, onMounted, computed,  watch} from 'vue'
import { useTagStore } from '@/stores/tags.js'
import Button from 'primevue/button'
import TagTreeEdit from '@/views/tags/TagTreeEdit.vue'

const tagStore = useTagStore()

const props = defineProps({
    editable: {
        type: Boolean,
        default: true
    }
})

const selectedKey = ref(null) // reference to check if the meta key is pressed
const isEditMode = ref(false) // reference to check if the tree is in edit mode
const expandedKeys = ref({ 0: true }) // reference to the keys that are expanded
const filterValue = ref(""); // reference to the string in the filter field

const expandMatchedNodes = (node, parentKeys = []) => {
  if (!node.children) return;

  for (const child of node.children) {
    if (child.label.toLowerCase().includes(filterValue.value.toLowerCase())) {
      parentKeys.forEach((key) => (expandedKeys.value[key] = true));
    }
    expandMatchedNodes(child, [...parentKeys, node.key]);
  }
};

watch(filterValue, () => {
  expandedKeys.value = {}; // Reset expanded nodes
  tagsTree.value.forEach((node) => expandMatchedNodes(node));
});




onMounted(() => {
    tagStore.Load()
})

const toggleEditMode = () => {
    if (props.editable) {
        isEditMode.value = !isEditMode.value
    }
}

const onNodeSelect = (node) => {
    console.log('select', node)
}

const onNodeUnselect = (node) => {
    console.log('unselect', node)
}

const tagsTree = computed(() => {
    if (!tagStore.tagTree.items || !Array.isArray(tagStore.tagTree.items)) {
        // still loading
        return {}
    }
    // Helper function to recursively process each node
    function transformNode(node) {
        const transformed = {
            key: node.id,
            label: `${node.name}`,
            selectable: true
            // icon: node.children ? 'pi pi-fw pi-chevron-up' : 'pi pi-fw pi-chevron-right',
        }
        if (node.children && Array.isArray(node.children)) {
            transformed.children = node.children.map(transformNode)
        }
        return transformed
    }
    // icon: node.children ? 'pi pi-fw pi-chevron-up' : 'pi pi-fw pi-chevron-right',
    return tagStore.tagTree.items.map(transformNode)
})

const checked = ref(false)
</script>
<template>
    <Tree
        v-if="!isEditMode"
        :value="tagsTree"
        :expandedKeys="expandedKeys"
        v-model:filterValue="filterValue"
        @nodeSelect="onNodeSelect"
        @nodeUnselect="onNodeUnselect"
        selectionMode="multiple"
        :metaKeySelection="true"
        v-model:selectionKeys="selectedKey"
        :filter="true"
        filterMode="lenient"
        class="w-full tag-tree"
    >
        <template #default="slotProps">
            <span>{{ slotProps.node.label }}</span>
        </template>
    </Tree>
    <TagTreeEdit v-if="isEditMode" :expandedKeys="expandedKeys" />

    <div v-if="props.editable" class="flex align-items-center justify-content-end">
        <Button
            :label="isEditMode ? 'Exit Edit Mode' : 'Enter Edit Mode'"
            :icon="isEditMode ? 'pi pi-times' : 'pi pi-pencil'"
            severity="secondary"
            @click="toggleEditMode"
        />
    </div>
</template>

<style></style>
