<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Card from 'primevue/card'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Checkbox from 'primevue/checkbox'
import Tag from 'primevue/tag'
import Divider from 'primevue/divider'
import { useToast } from 'primevue/usetoast'
import CategorySelect from '@/components/common/CategorySelect.vue'
import AdHocCategoryRuleDialog from '@/components/common/AdHocCategoryRuleDialog.vue'
import { useCategoryUtils } from '@/utils/categoryUtils'
import type { CategoryRuleGroup, CategoryRulePattern } from '@/types/csvimport'

type PatternDraft = CategoryRulePattern | { id: null; pattern: string; isRegex: boolean }
import {
    getCategoryRuleGroups,
    createCategoryRuleGroup,
    updateCategoryRuleGroup,
    deleteCategoryRuleGroup,
    createCategoryRulePattern,
    updateCategoryRulePattern,
    deleteCategoryRulePattern,
} from '@/lib/api/CsvImport'

const router = useRouter()
const toast = useToast()
const { getCategoryName } = useCategoryUtils()

// ============ Category Rule Groups ============

const categoryRuleGroups = ref<CategoryRuleGroup[]>([])
const isLoadingRules = ref(false)
const showGroupDialog = ref(false)
const editingGroup = ref<CategoryRuleGroup | null>(null)
const isSavingRule = ref(false)

// ============ Ad-hoc Rule Dialog ============
const adhocDialogRef = ref<InstanceType<typeof AdHocCategoryRuleDialog> | null>(null)

const formGroupName = ref('')
const formGroupCategoryId = ref<number | null>(null)
const formGroupPriority = ref(0)
const formGroupPatterns = ref<PatternDraft[]>([])

const newPatternValue = ref('')
const newPatternIsRegex = ref(false)

const resolveCategoryName = (categoryId: number) => {
    let name = getCategoryName(categoryId, 'expense')
    if (name === 'Unknown') {
        name = getCategoryName(categoryId, 'income')
    }
    return name
}

const loadRules = async () => {
    isLoadingRules.value = true
    try {
        categoryRuleGroups.value = await getCategoryRuleGroups()
        categoryRuleGroups.value.sort((a, b) => a.priority - b.priority)
    } catch (error: unknown) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load category rule groups: ' + (error as Error).message, life: 3000 })
    } finally {
        isLoadingRules.value = false
    }
}

const openCreateGroupDialog = () => {
    editingGroup.value = null
    formGroupName.value = ''
    formGroupCategoryId.value = null
    formGroupPriority.value = categoryRuleGroups.value.length
    formGroupPatterns.value = []
    newPatternValue.value = ''
    newPatternIsRegex.value = false
    showGroupDialog.value = true
}

const openEditGroupDialog = (group: CategoryRuleGroup) => {
    editingGroup.value = group
    formGroupName.value = group.name
    formGroupCategoryId.value = group.categoryId
    formGroupPriority.value = group.priority
    formGroupPatterns.value = (group.patterns || []).map((p: CategoryRulePattern): PatternDraft => ({ ...p }))
    newPatternValue.value = ''
    newPatternIsRegex.value = false
    showGroupDialog.value = true
}

const addPattern = () => {
    if (!newPatternValue.value.trim()) return
    formGroupPatterns.value.push({
        id: null,
        pattern: newPatternValue.value.trim(),
        isRegex: newPatternIsRegex.value,
    })
    newPatternValue.value = ''
    newPatternIsRegex.value = false
}

const removePattern = (index: number) => {
    formGroupPatterns.value.splice(index, 1)
}

const handleSaveGroup = async () => {
    if (!formGroupName.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Name is required', life: 3000 })
        return
    }
    if (!formGroupCategoryId.value) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Category is required', life: 3000 })
        return
    }

    isSavingRule.value = true
    try {
        if (editingGroup.value) {
            await updateCategoryRuleGroup(editingGroup.value.id, {
                name: formGroupName.value.trim(),
                categoryId: formGroupCategoryId.value,
                priority: formGroupPriority.value ?? 0,
                patterns: editingGroup.value.patterns || [],
            })

            const oldPatterns = editingGroup.value.patterns || []
            const newPatterns = formGroupPatterns.value
            const newIds = new Set(newPatterns.filter(p => p.id).map(p => p.id))

            for (const old of oldPatterns) {
                if (!newIds.has(old.id)) {
                    await deleteCategoryRulePattern(editingGroup.value.id, old.id)
                }
            }
            for (const p of newPatterns) {
                if (p.id) {
                    const old = oldPatterns.find(o => o.id === p.id)
                    if (old && (old.pattern !== p.pattern || old.isRegex !== p.isRegex)) {
                        await updateCategoryRulePattern(editingGroup.value.id, p.id, { pattern: p.pattern, isRegex: p.isRegex })
                    }
                } else {
                    await createCategoryRulePattern(editingGroup.value.id, { pattern: p.pattern, isRegex: p.isRegex })
                }
            }
            toast.add({ severity: 'success', summary: 'Success', detail: 'Group updated successfully', life: 3000 })
        } else {
            const created = await createCategoryRuleGroup({
                name: formGroupName.value.trim(),
                categoryId: formGroupCategoryId.value,
                priority: formGroupPriority.value ?? 0,
                patterns: [],
            })
            for (const p of formGroupPatterns.value) {
                await createCategoryRulePattern(created.id, { pattern: p.pattern, isRegex: p.isRegex })
            }
            toast.add({ severity: 'success', summary: 'Success', detail: 'Group created successfully', life: 3000 })
        }
        showGroupDialog.value = false
        await loadRules()
    } catch (error: unknown) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save group: ' + (error as Error).message, life: 3000 })
    } finally {
        isSavingRule.value = false
    }
}

const handleDeleteGroup = async (group: CategoryRuleGroup) => {
    if (!confirm(`Are you sure you want to delete the group "${group.name}" and all its patterns?`)) {
        return
    }
    try {
        await deleteCategoryRuleGroup(group.id)
        toast.add({ severity: 'success', summary: 'Success', detail: 'Group deleted successfully', life: 3000 })
        await loadRules()
    } catch (error: unknown) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete group: ' + (error as Error).message, life: 3000 })
    }
}

onMounted(() => {
    loadRules()
})
</script>

<template>
    <div>
        <div class="mb-4">
            <h1 class="text-2xl font-bold mb-2 text-color">Category Matching Rules</h1>
            <p class="text-color-secondary m-0 mb-3 text-base">
                Define rule groups to automatically assign categories to imported transactions based on description matching. Groups are evaluated in priority order; the first match wins.
            </p>
            <div class="flex gap-2 justify-content-end">
                <Button
                    label="Apply Ad-hoc Rule"
                    icon="ti ti-bolt"
                    severity="secondary"
                    @click="adhocDialogRef?.open()"
                />
                <Button
                    label="Re-apply Rules"
                    icon="ti ti-refresh"
                    severity="secondary"
                    @click="router.push('/settings/reapply-rules')"
                />
                <Button
                    label="New Group"
                    icon="ti ti-plus"
                    @click="openCreateGroupDialog"
                />
            </div>
        </div>

        <Card>
            <template #content>
                <DataTable
                    :value="categoryRuleGroups"
                    :loading="isLoadingRules"
                    dataKey="id"
                    stripedRows
                    :paginator="categoryRuleGroups.length > 50"
                    :rows="50"
                    responsiveLayout="scroll"
                >
                    <template #empty>
                        <div class="empty-state">
                            <i class="ti ti-inbox"></i>
                            <p>No category matching rule groups found</p>
                            <Button label="Create Your First Group" icon="ti ti-plus" @click="openCreateGroupDialog" outlined />
                        </div>
                    </template>

                    <Column field="priority" header="Priority" :sortable="true" style="width: 100px">
                        <template #body="{ data }">
                            <span class="font-semibold">{{ data.priority }}</span>
                        </template>
                    </Column>
                    <Column field="name" header="Name">
                        <template #body="{ data }">
                            <span class="font-semibold">{{ data.name }}</span>
                        </template>
                    </Column>
                    <Column field="categoryId" header="Category">
                        <template #body="{ data }">
                            {{ resolveCategoryName(data.categoryId) }}
                        </template>
                    </Column>
                    <Column header="Patterns" style="width: 120px">
                        <template #body="{ data }">
                            <Tag :value="String(data.patterns?.length || 0)" severity="info" />
                        </template>
                    </Column>
                    <Column header="Actions" :exportable="false" style="width: 100px">
                        <template #body="{ data }">
                            <div class="flex gap-1 justify-content-center">
                                <Button icon="ti ti-pencil" text rounded class="p-1"
                                    @click="openEditGroupDialog(data)" v-tooltip.top="'Edit group'" />
                                <Button icon="ti ti-trash" severity="danger" text rounded class="p-1"
                                    @click="handleDeleteGroup(data)" v-tooltip.top="'Delete group'" />
                            </div>
                        </template>
                    </Column>
                </DataTable>
            </template>
        </Card>

        <!-- Group Edit/Create Dialog -->
        <Dialog
            v-model:visible="showGroupDialog"
            :header="editingGroup ? 'Edit Rule Group' : 'Create Rule Group'"
            :modal="true"
            :closable="true"
            class="entry-dialog entry-dialog--wide"
        >
            <div class="rule-dialog-content">
                <div class="field">
                    <label for="groupName">Name *</label>
                    <InputText id="groupName" v-model="formGroupName"
                        placeholder="e.g., Amazon, Grocery Stores" class="w-full" />
                </div>

                <CategorySelect v-model="formGroupCategoryId" type="all" label="Category *" />

                <div class="field">
                    <label for="groupPriority">Priority</label>
                    <InputNumber id="groupPriority" v-model="formGroupPriority" :min="0" class="w-full" />
                    <small class="text-color-secondary">Lower value = higher priority. First matching group wins.</small>
                </div>

                <Divider />

                <div class="patterns-section">
                    <label class="font-semibold text-color">Patterns</label>
                    <div class="patterns-list" v-if="formGroupPatterns.length > 0">
                        <div v-for="(p, index) in formGroupPatterns" :key="index" class="pattern-row">
                            <span class="pattern-text">{{ p.pattern }}</span>
                            <Tag :value="p.isRegex ? 'Regex' : 'Substring'" :severity="p.isRegex ? 'warn' : 'info'" class="flex-shrink-0" />
                            <Button icon="ti ti-x" severity="danger" text rounded size="small"
                                @click="removePattern(index)" v-tooltip.top="'Remove pattern'" />
                        </div>
                    </div>
                    <div v-else class="text-color-secondary text-sm">No patterns yet. Add one below.</div>

                    <div class="add-pattern-row">
                        <InputText v-model="newPatternValue" placeholder="e.g., AMAZON or .*amazon.*"
                            class="flex-grow-1" size="small" @keydown.enter="addPattern" />
                        <div class="flex align-items-center gap-2 flex-shrink-0">
                            <Checkbox v-model="newPatternIsRegex" :binary="true" inputId="newPatternRegex" />
                            <label for="newPatternRegex" class="text-sm white-space-nowrap">Regex</label>
                        </div>
                        <Button label="Add" icon="ti ti-plus" outlined @click="addPattern" :disabled="!newPatternValue.trim()" />
                    </div>
                </div>

                <div class="flex justify-content-end gap-2 mt-3">
                    <Button label="Cancel" severity="secondary" text @click="showGroupDialog = false" />
                    <Button :label="editingGroup ? 'Update' : 'Create'" icon="ti ti-check"
                        :loading="isSavingRule" @click="handleSaveGroup" />
                </div>
            </div>
        </Dialog>

        <AdHocCategoryRuleDialog ref="adhocDialogRef" />
    </div>
</template>

<style scoped>
.pattern-text {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: monospace;
    font-size: 0.9rem;
    color: var(--primary-700, var(--primary-color));
}

.rule-dialog-content {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.patterns-section {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.patterns-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.pattern-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.5rem;
    border-radius: 6px;
    background-color: var(--surface-50);
    border-left: 3px solid var(--primary-color);
}

.add-pattern-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}
</style>
