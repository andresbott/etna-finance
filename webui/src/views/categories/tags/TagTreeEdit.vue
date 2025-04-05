<script setup>
import TreeTable from 'primevue/treetable'
import Column from 'primevue/column'
import { ref, onMounted, computed, nextTick, onUpdated } from 'vue'
import { useTagStore } from '@/stores/tags.js'
import TagDialog from '@/views/tags/TagDialog.vue'
import DeleteTagDialog from '@/views/tags/deleteTagDialog.vue'
import Tree from 'primevue/tree'
const tagStore = useTagStore()

const props = defineProps({
    expandedKeys: {
        type: Object,
        default: {}
    }
})

// onMounted(() => {
//     tagStore.Load()
// })

onUpdated(() => {
    setItemsDraggable()
})

const onNodeExpand = (node) => {
    nextTick(() => {
        setItemsDraggable()
    })
}

function setItemsDraggable() {
    // todo: selection needs to be done at ID level not at class level in case more than one tree is added
    const draggableItems = document.querySelectorAll('.tag-treetable .p-treetable-tbody tr')
    let isDragging = false,
        currentElement = null
    draggableItems.forEach((item) => {
        const handle = item.querySelector('.handle')
        if (handle) {
            handle.onmousedown = function (e) {
                item.setAttribute('draggable', 'true')
            }

            handle.onmouseup = function (e) {
                item.setAttribute('draggable', 'false')
            }

            item.ondragstart = function (e) {
                if (handle.dataset.id) {
                    e.dataTransfer.setData('itemID', handle.dataset.id)
                }
                createDropZones(item)
                isDragging = true
                currentElement = item
                item.style.cursor = 'grabbing'
            }

            item.ondragend = () => {
                item.setAttribute('draggable', 'false')
                if (isDragging) {
                    removeDropZones()
                }
                isDragging = false
                item.style.cursor = ''
            }
        }
    })
}
// createDropZones iterates over the tree and makes all items droppable
// also it should create placeholders to sort items on the same levels
function createDropZones(excludeItem) {
    const droppableItems = document.querySelectorAll('.tag-treetable .p-treetable-tbody tr')
    droppableItems.forEach((item) => {
        if (item !== excludeItem) {
            item.ondragover = dragoverHandler
            item.ondrop = dropHandler
            item.classList.add('dropzone')
        }
    })

    // todo: sort on the same level
    // const draggableItems = document.querySelectorAll('.tag-tree li.p-tree-node');
    // draggableItems.forEach(item => {
    //
    //   let dropItem = document.createElement("div");
    //   dropItem.classList.add("drop");
    //   dropItem.id = 'dropZone';
    //   dropItem.ondrop = dropHandler;
    //   dropItem.ondragover = dragoverHandler;
    //
    //   item.after(dropItem)
    // })
}

// Drag handlers
function dragoverHandler(ev) {
    ev.preventDefault()
    ev.dataTransfer.dropEffect = 'move'
}

function dropHandler(ev) {
    ev.preventDefault()
    const draggedId = ev.dataTransfer.getData('itemID')
    const dropZone = ev.target.closest('tr')
    const dzh = dropZone.querySelector('.handle')

    if (dzh && dzh.dataset.id) {
        const targetId = dzh.dataset.id
        console.log('move ' + draggedId + ' to ' + targetId)
    }
}

function removeDropZones() {
    const dropItems = document.querySelectorAll('.tag-tree .drop')
    dropItems.forEach((item) => {
        item.remove()
    })
    const dropZones = document.querySelectorAll('.tag-treetable .dropzone')
    dropZones.forEach((item) => {
        item.classList.remove('dropzone')
    })
}

const tagsTree = computed(() => {
    // still loading
    if (!tagStore.tagTree.items || !Array.isArray(tagStore.tagTree.items)) {
        return {}
    }
    // Helper function to recursively process each node
    function transformNode(node) {
        const transformed = {
            key: `${node.id}`,
            data: {
                id: node.id,
                name: `${node.name}`
            }
            // icon: node.children ? 'pi pi-fw pi-chevron-up' : 'pi pi-fw pi-chevron-right',
        }

        if (node.children && Array.isArray(node.children)) {
            transformed.children = node.children.map(transformNode)
        }

        if (!Array.isArray(node.children)) {
            transformed.children = []
        }
        return transformed
    }
    const nodes = tagStore.tagTree.items.map(transformNode)

    const root = {
        key: '0',
        data: {
            id: 0,
            name: `Root`
        }
    }
    root.children = nodes
    // nodes.unshift()

    // icon: node.children ? 'pi pi-fw pi-chevron-up' : 'pi pi-fw pi-chevron-right',
    return [root]
})
</script>
<template>
    <TreeTable
        :value="tagsTree"
        :expandedKeys="props.expandedKeys"
        :reorderableColumns="false"
        class="w-full tag-treetable"
        @nodeExpand="onNodeExpand"
    >
        <Column field="type" header="move">
            <template #body="slotProps">
                <i
                    v-if="slotProps.node.data.id !== 0"
                    class="pi pi-fw pi-arrows-alt handle"
                    :data-id="slotProps.node.data.id"
                ></i>
            </template>
        </Column>
        <Column field="name" header="Name" expander>
            <template #body="slotProps">
                <div class="flex flex-wrap gap-2">
                    {{ slotProps.node.data.name }}
                </div>
            </template>
        </Column>
        <Column field="name" header="Name">
            <template #body="slotProps">
                <div class="flex flex-wrap gap-2">
                    <TagDialog
                        v-if="slotProps.node.data.id !== 0"
                        :isEdit="true"
                        :itemId="slotProps.node.data.id"
                        :name="slotProps.node.data.name"
                    />
                    <TagDialog :isEdit="false" :parentId="slotProps.node.data.id" />
                    <DeleteTagDialog
                        v-if="slotProps.node.data.id !== 0"
                        :name="slotProps.node.data.name"
                        :itemId="slotProps.node.data.id"
                    />
                </div>
            </template>
        </Column>
    </TreeTable>
</template>

<style>
.drop {
    width: 100%;
    height: 20px;
    border: 1px solid;
}
.dropzone {
    position: relative;
}

.dropzone:after {
    content: ''; /* Required for pseudo-elements */
    position: absolute; /* Position relative to the <tr> */
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    margin: 4px;
    border: 2px dashed #a4a4a4; /* Red dashed lines */
    box-sizing: border-box; /* Ensure borders are included in dimensions */
    pointer-events: none; /* Allow interactions with the table row itself */
    z-index: 1; /* Ensure it stays in the background or adjust as needed */
}
</style>
