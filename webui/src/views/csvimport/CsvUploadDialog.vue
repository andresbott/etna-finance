<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { parseCSV } from '@/lib/api/CsvImport'
import { getApiErrorMessage } from '@/utils/apiError'

const props = defineProps({
    visible: {
        type: Boolean,
        required: true
    },
    accountId: {
        type: Number,
        default: null
    }
})

const emit = defineEmits(['update:visible'])

const router = useRouter()
const toast = useToast()

const selectedFile = ref(null)
const isParsing = ref(false)
const parseError = ref('')

const onFileChange = (event) => {
    const files = event.target.files
    if (files && files.length > 0) {
        selectedFile.value = files[0]
        parseError.value = ''
    }
}

const handleParse = async () => {
    if (!selectedFile.value || !props.accountId) return
    isParsing.value = true
    parseError.value = ''
    try {
        const result = await parseCSV(props.accountId, selectedFile.value)
        emit('update:visible', false)
        router.push({
            name: 'csv-import',
            params: { accountId: props.accountId },
            state: { parsedRows: JSON.stringify(result.rows) }
        })
    } catch (err) {
        parseError.value = getApiErrorMessage(err)
    } finally {
        isParsing.value = false
    }
}

const handleClose = () => {
    selectedFile.value = null
    parseError.value = ''
    isParsing.value = false
    emit('update:visible', false)
}
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="handleClose"
        header="Import CSV"
        modal
        :style="{ width: '30rem' }"
    >
        <div class="upload-form">
            <div class="file-input-wrapper">
                <input
                    type="file"
                    accept=".csv"
                    @change="onFileChange"
                    class="file-input"
                />
            </div>

            <Message v-if="parseError" severity="error" :closable="false" class="mt-3">
                {{ parseError }}
            </Message>
        </div>

        <template #footer>
            <Button
                label="Parse"
                icon="pi pi-upload"
                :loading="isParsing"
                :disabled="!selectedFile"
                @click="handleParse"
            />
            <Button
                label="Cancel"
                severity="secondary"
                @click="handleClose"
            />
        </template>
    </Dialog>
</template>

<style scoped>
.upload-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.file-input {
    width: 100%;
}
</style>
