<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Button from 'primevue/button'
import CategorySelect from '@/components/common/CategorySelect.vue'

const router = useRouter()
const toast = useToast()

const visible = ref(false)
const categoryId = ref<number | null>(null)
const pattern = ref('')
const isRegex = ref(false)

const open = () => {
    categoryId.value = null
    pattern.value = ''
    isRegex.value = false
    visible.value = true
}

const handleApply = () => {
    if (!categoryId.value) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Category is required', life: 3000 })
        return
    }
    if (!pattern.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Pattern is required', life: 3000 })
        return
    }
    visible.value = false
    router.push({
        name: 'reapply-rules',
        state: {
            adhocRule: {
                categoryId: categoryId.value,
                pattern: pattern.value.trim(),
                isRegex: isRegex.value,
            },
        },
    })
}

defineExpose({ open })
</script>

<template>
    <Dialog
        v-model:visible="visible"
        header="Apply Ad-hoc Rule"
        :modal="true"
        :closable="true"
        class="entry-dialog"
    >
        <div class="adhoc-dialog-content">
            <div class="field">
                <label for="adhocPattern">Pattern *</label>
                <InputText id="adhocPattern" v-model="pattern"
                    placeholder="e.g., AMAZON or .*amazon.*" class="w-full" />
            </div>

            <div class="flex align-items-center gap-2">
                <Checkbox v-model="isRegex" :binary="true" inputId="adhocRegex" />
                <label for="adhocRegex" class="text-sm">Is Regex</label>
            </div>

            <CategorySelect v-model="categoryId" type="all" label="Category *" />

            <div class="flex justify-content-end gap-2 mt-3">
                <Button label="Apply" icon="pi pi-bolt" @click="handleApply" />
                <Button label="Cancel" severity="secondary" text @click="visible = false" />
            </div>
        </div>
    </Dialog>
</template>

<style scoped>
.adhoc-dialog-content {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}
</style>
