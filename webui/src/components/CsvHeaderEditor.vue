<script setup>
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InputText from 'primevue/inputtext'
import Dropdown from 'primevue/dropdown'
import Button from 'primevue/button'
import Message from 'primevue/message'

const props = defineProps({
    headers: {
        type: Array,
        default: () => []
    }
})

const emit = defineEmits(['update:headers', 'save'])

// Local copy of headers for editing
const localHeaders = ref([...props.headers])

// Available field types for mapping
const fieldTypes = [
    { label: 'Not Mapped', value: null },
    { label: 'Date', value: 'date' },
    { label: 'Description', value: 'description' },
    { label: 'Amount', value: 'amount' },
    { label: 'Category', value: 'category' },
    { label: 'Account', value: 'account' },
    { label: 'Type (Income/Expense)', value: 'type' },
    { label: 'Reference', value: 'reference' },
    { label: 'Notes', value: 'notes' }
]

// Date format options
const dateFormats = [
    { label: 'YYYY-MM-DD', value: 'YYYY-MM-DD' },
    { label: 'DD/MM/YYYY', value: 'DD/MM/YYYY' },
    { label: 'MM/DD/YYYY', value: 'MM/DD/YYYY' },
    { label: 'DD.MM.YYYY', value: 'DD.MM.YYYY' },
    { label: 'YYYY/MM/DD', value: 'YYYY/MM/DD' }
]

const selectedDateFormat = ref('YYYY-MM-DD')

// Validation
const validationErrors = computed(() => {
    const errors = []
    const mappedFields = localHeaders.value.filter(h => h.mappedTo).map(h => h.mappedTo)
    
    // Check for required fields
    if (!mappedFields.includes('date')) {
        errors.push('Date field is required')
    }
    if (!mappedFields.includes('amount')) {
        errors.push('Amount field is required')
    }
    if (!mappedFields.includes('description')) {
        errors.push('Description field is required')
    }
    
    // Check for duplicate mappings
    const duplicates = mappedFields.filter((item, index) => mappedFields.indexOf(item) !== index)
    if (duplicates.length > 0) {
        errors.push(`Duplicate mappings found: ${[...new Set(duplicates)].join(', ')}`)
    }
    
    return errors
})

const isValid = computed(() => validationErrors.value.length === 0)

// Add a new header
const addHeader = () => {
    localHeaders.value.push({
        id: Date.now(),
        name: `Column ${localHeaders.value.length + 1}`,
        mappedTo: null,
        example: ''
    })
}

// Remove a header
const removeHeader = (index) => {
    localHeaders.value.splice(index, 1)
}

// Save changes
const handleSave = () => {
    if (isValid.value) {
        emit('update:headers', localHeaders.value)
        emit('save', {
            headers: localHeaders.value,
            dateFormat: selectedDateFormat.value
        })
    }
}

// Reset to original
const handleReset = () => {
    localHeaders.value = [...props.headers]
}

// Update header name
const updateHeaderName = (index, value) => {
    localHeaders.value[index].name = value
}

// Update mapping
const updateMapping = (index, value) => {
    localHeaders.value[index].mappedTo = value
}

// Update example
const updateExample = (index, value) => {
    localHeaders.value[index].example = value
}
</script>

<template>
    <div class="csv-header-editor">
        <Card>
            <template #title>
                <div class="card-header">
                    <div class="title-section">
                        <i class="pi pi-table"></i>
                        CSV Header Mapping
                    </div>
                    <div class="header-actions">
                        <Button
                            icon="pi pi-plus"
                            label="Add Column"
                            @click="addHeader"
                            outlined
                            size="small"
                        />
                    </div>
                </div>
            </template>
            <template #content>
                <div class="editor-content">
                    <Message
                        v-if="validationErrors.length > 0"
                        severity="warn"
                        :closable="false"
                    >
                        <ul class="validation-errors">
                            <li v-for="error in validationErrors" :key="error">{{ error }}</li>
                        </ul>
                    </Message>

                    <div class="date-format-selector">
                        <label for="dateFormat">Date Format:</label>
                        <Dropdown
                            id="dateFormat"
                            v-model="selectedDateFormat"
                            :options="dateFormats"
                            optionLabel="label"
                            optionValue="value"
                            placeholder="Select date format"
                        />
                    </div>

                    <DataTable
                        :value="localHeaders"
                        stripedRows
                        responsiveLayout="scroll"
                        :scrollable="true"
                        class="header-table"
                    >
                        <template #empty>
                            <div class="empty-state">
                                <i class="pi pi-inbox"></i>
                                <p>No CSV headers defined. Click "Add Column" to start.</p>
                            </div>
                        </template>

                        <Column header="Column Name" style="min-width: 200px">
                            <template #body="{ data, index }">
                                <InputText
                                    :modelValue="data.name"
                                    @update:modelValue="updateHeaderName(index, $event)"
                                    placeholder="Enter column name"
                                    class="w-full"
                                />
                            </template>
                        </Column>

                        <Column header="Map To Field" style="min-width: 200px">
                            <template #body="{ data, index }">
                                <Dropdown
                                    :modelValue="data.mappedTo"
                                    @update:modelValue="updateMapping(index, $event)"
                                    :options="fieldTypes"
                                    optionLabel="label"
                                    optionValue="value"
                                    placeholder="Select field type"
                                    class="w-full"
                                />
                            </template>
                        </Column>

                        <Column header="Example Value" style="min-width: 200px">
                            <template #body="{ data, index }">
                                <InputText
                                    :modelValue="data.example"
                                    @update:modelValue="updateExample(index, $event)"
                                    placeholder="Sample data"
                                    class="w-full"
                                />
                            </template>
                        </Column>

                        <Column header="Actions" style="width: 100px" :exportable="false">
                            <template #body="{ index }">
                                <Button
                                    icon="pi pi-trash"
                                    severity="danger"
                                    text
                                    rounded
                                    @click="removeHeader(index)"
                                    v-tooltip.top="'Remove column'"
                                />
                            </template>
                        </Column>
                    </DataTable>

                    <div class="editor-actions">
                        <Button
                            label="Reset"
                            icon="pi pi-refresh"
                            severity="secondary"
                            outlined
                            @click="handleReset"
                        />
                        <Button
                            label="Save Mapping"
                            icon="pi pi-save"
                            @click="handleSave"
                            :disabled="!isValid"
                        />
                    </div>
                </div>
            </template>
        </Card>
    </div>
</template>

<style scoped lang="scss">
.csv-header-editor {
    width: 100%;
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

.title-section {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    
    i {
        color: var(--primary-color);
    }
}

.header-actions {
    display: flex;
    gap: 0.5rem;
}

.editor-content {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
}

.validation-errors {
    margin: 0;
    padding-left: 1.5rem;
    
    li {
        margin: 0.25rem 0;
    }
}

.date-format-selector {
    display: flex;
    align-items: center;
    gap: 1rem;
    
    label {
        font-weight: 600;
        color: var(--text-color);
    }
    
    :deep(.p-dropdown) {
        min-width: 200px;
    }
}

.header-table {
    :deep(.p-datatable-wrapper) {
        border-radius: 8px;
    }
}

.empty-state {
    text-align: center;
    padding: 3rem 1rem;
    color: var(--text-color-secondary);
    
    i {
        font-size: 3rem;
        margin-bottom: 1rem;
        opacity: 0.5;
    }
    
    p {
        margin: 0;
        font-size: 1rem;
    }
}

.editor-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    padding-top: 1rem;
}

:deep(.p-card-content) {
    padding-top: 0;
}

:deep(.p-inputtext),
:deep(.p-dropdown) {
    width: 100%;
}
</style>

