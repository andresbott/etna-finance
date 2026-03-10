<script setup>
/**
 * Category add/edit dialog. Uses local state + emit (parent persists) rather than
 * PrimeVue Form + Zod like AccountDialog/AccountProviderDialog. Intentional for
 * simpler category CRUD; align with Form+Zod if validation/consistency is needed.
 */
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import CategorySelect from '@/components/common/CategorySelect.vue'
import IconSelect from '@/components/common/IconSelect.vue'

const props = defineProps({
    visible: Boolean,
    categoryData: Object,
    expenseTreeData: { type: Array, default: () => [] },
    incomeTreeData: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:visible', 'save', 'reset', 'update:categoryData'])

const localCategory = ref({
    id: null,
    name: '',
    description: '',
    parentId: 0,
    type: 'expense',
    action: null,
    icon: 'pi-tag',
    ...props.categoryData
})

watch(
    () => props.categoryData,
    (newVal) => {
        if (newVal) {
            localCategory.value = {
                id: null,
                name: '',
                description: '',
                parentId: 0,
                type: 'expense',
                action: null,
                icon: 'pi-tag',
                ...newVal
            }
        }
    },
    { immediate: true, deep: true }
)

// Get dialog header title
const titleMap = {
    income: 'Income Category',
    expense: 'Expense Category'
}
const dialogHeaderTitle = computed(() => {
    const action = props.categoryData.action === 'edit' ? 'Edit' : 'Add New'
    const type = titleMap[props.categoryData.type]
    return `${action} ${type}`
})

/* ---------------------------
   Actions
---------------------------- */
const save = () => {
    emit('update:categoryData', localCategory.value)
    emit('save')
    emit('update:visible', false)
}

const reset = () => {
    localCategory.value = {
        id: null,
        name: '',
        description: '',
        parentId: 0,
        type: 'expense',
        action: null,
        icon: 'pi-tag'
    }

    emit('reset')
}

const cancel = () => {
    reset()
    emit('update:visible', false)
}

// Show parent selector only when editing
const showParentSelector = computed(() => {
    return localCategory.value && localCategory.value.action !== 'add'
})
</script>

<template>
    <Dialog
        :visible="visible"
        modal
        :header="dialogHeaderTitle"
        :draggable="false"
        class="entry-dialog"
        @update:visible="emit('update:visible', $event)"
        @hide="cancel"
    >
        <div>
            <!-- Name -->
            <div class="field">
                <label for="category-name" class="form-label">Name</label>
                <InputText id="category-name" v-model="localCategory.name" />
            </div>

            <!-- Description -->
            <div class="field">
                <label for="category-description" class="form-label">Description</label>
                <InputText
                    id="category-description"
                    v-model="localCategory.description"
                />
            </div>

            <!-- Icon -->
            <div class="field">
                <label for="category-icon" class="form-label">Icon</label>
                <IconSelect v-model="localCategory.icon" />
            </div>

            <!-- Parent selector -->
            <CategorySelect
                v-if="showParentSelector"
                v-model="localCategory.parentId"
                :type="localCategory.type"
            />
        </div>

        <template #footer>
            <Button label="Save" icon="pi pi-check" @click="save" :disabled="!localCategory.name" />
            <Button label="Cancel" icon="pi pi-times" text @click="cancel" />
        </template>
    </Dialog>
</template>
