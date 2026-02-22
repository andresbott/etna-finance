<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import Card from 'primevue/card'
import Message from 'primevue/message'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import ProgressSpinner from 'primevue/progressspinner'
import { useDateFormat } from '@/composables/useDateFormat'
import { useTasks } from '@/composables/useTasks'

const { formatDateTime } = useDateFormat()
const leftSidebarCollapsed = ref(true)

const {
    tasks,
    executions,
    triggerTask,
    cancelTaskExecution,
    cancelMutation,
    upsertTask,
    patchTask,
    deleteTaskSchedule,
    getStatusSeverity,
    getStatusLabel,
    triggeringTaskId,
    tasksQuery,
    executionsQuery
} = useTasks()

function isExecutionCancelable(status) {
    return status === 'waiting' || status === 'running'
}

function isCancelingExecution(id) {
    return cancelMutation.isPending.value && cancelMutation.variables.value === id
}

// Quartz format: second minute hour day-of-month month day-of-week (6 fields)
const SCHEDULE_PRESETS = [
    { label: 'Daily', cron: '0 0 0 * * *' },
    { label: 'Weekly', cron: '0 0 0 * * 0' },
    { label: 'Monthly', cron: '0 0 0 1 * *' }
]

function scheduleSummary(task) {
    if (!task.schedule) return 'Not scheduled'
    const cron = task.schedule.cron_expression
    const preset = SCHEDULE_PRESETS.find((p) => p.cron === cron)
    const label = preset ? preset.label : cron
    return task.schedule.enabled ? label : `${label} (paused)`
}

const scheduleDialogVisible = ref(false)
const scheduleDialogTask = ref(null)
const scheduleDialogForm = ref({ cronExpression: '', enabled: true })
const scheduleDialogSaving = ref(false)
const scheduleDialogError = ref('')

function openScheduleDialog(task) {
    scheduleDialogTask.value = task
    scheduleDialogForm.value = {
        cronExpression: task.schedule?.cron_expression ?? '',
        enabled: task.schedule?.enabled ?? true
    }
    scheduleDialogError.value = ''
    scheduleDialogVisible.value = true
}

function setPresetCron(cron) {
    scheduleDialogForm.value.cronExpression = cron
    scheduleDialogError.value = ''
}

async function saveScheduleDialog() {
    const task = scheduleDialogTask.value
    if (!task) return
    const { cronExpression, enabled } = scheduleDialogForm.value
    const cron = cronExpression.trim()
    if (!cron) {
        scheduleDialogError.value = 'Cron expression is required'
        return
    }
    scheduleDialogError.value = ''
    scheduleDialogSaving.value = true
    try {
        if (task.schedule) {
            await patchTask(task.id, { cron_expression: cron, enabled })
        } else {
            await upsertTask(task.id, { cron_expression: cron, enabled })
        }
        scheduleDialogVisible.value = false
    } catch (e) {
        const data = e.response?.data
        scheduleDialogError.value =
            typeof data === 'string' ? data : data?.message ?? e.message ?? 'Failed to save schedule'
    } finally {
        scheduleDialogSaving.value = false
    }
}

async function removeScheduleDialog() {
    const task = scheduleDialogTask.value
    if (!task?.schedule) return
    scheduleDialogError.value = ''
    scheduleDialogSaving.value = true
    try {
        await deleteTaskSchedule(task.id)
        scheduleDialogVisible.value = false
    } catch (e) {
        const data = e.response?.data
        scheduleDialogError.value =
            typeof data === 'string' ? data : data?.message ?? e.message ?? 'Failed to remove schedule'
    } finally {
        scheduleDialogSaving.value = false
    }
}

function closeScheduleDialog() {
    scheduleDialogVisible.value = false
    scheduleDialogTask.value = null
    scheduleDialogError.value = ''
}

// Queue order is defined by the backend (waiting → running → terminal, newest first within each).
function formatExecutionTime(isoString) {
    if (!isoString) return '-'
    return formatDateTime(isoString)
}

function formatDuration(executionStartedAt, finishedAt) {
    if (!executionStartedAt || !finishedAt) return ''
    const start = new Date(executionStartedAt).getTime()
    const end = new Date(finishedAt).getTime()
    const sec = Math.round((end - start) / 1000)
    if (sec < 0) return ''
    // Guard against zero/unset start (e.g. canceled before run) producing huge duration
    const MAX_REASONABLE_SEC = 86400 * 365 // 1 year
    if (sec > MAX_REASONABLE_SEC) return ''
    if (sec < 60) return `${sec}s`
    const m = Math.floor(sec / 60)
    const s = sec % 60
    return s ? `${m}m ${s}s` : `${m}m`
}

function taskDisplayName(taskName) {
    const t = tasks.value.find((x) => x.id === taskName)
    return t ? t.name : taskName
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <Message
                    v-if="tasksQuery.isError.value || executionsQuery.isError.value"
                    severity="error"
                    :closable="false"
                    class="error-message"
                >
                    {{
                        tasksQuery.error.value?.message ||
                        executionsQuery.error.value?.message ||
                        'Failed to load tasks'
                    }}
                </Message>

                <div class="grid">
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-briefcase"></i>
                                    <span>Tasks</span>
                                </div>
                            </template>
                            <template #content>
                                <ProgressSpinner
                                    v-if="tasksQuery.isLoading.value"
                                    style="width: 3rem; height: 3rem"
                                    stroke-width="4"
                                />
                                <DataTable
                                    v-else
                                    :value="tasks"
                                    dataKey="id"
                                    stripedRows
                                    class="p-datatable-sm tasks-table"
                                >
                                    <Column field="name" header="Name">
                                        <template #body="{ data }">
                                            <span class="font-semibold">{{
                                                data.name
                                            }}</span>
                                            <p
                                                v-if="data.description"
                                                class="text-sm text-color-secondary mt-0 mb-0 mt-1"
                                            >
                                                {{ data.description }}
                                            </p>
                                        </template>
                                    </Column>
                                    <Column header="Schedule" style="width: 14rem">
                                        <template #body="{ data }">
                                            <span class="schedule-summary text-color-secondary">
                                                {{ scheduleSummary(data) }}
                                            </span>
                                            <Button
                                                :label="data.schedule ? 'Edit' : 'Schedule'"
                                                icon="pi pi-calendar"
                                                size="small"
                                                text
                                                class="ml-2"
                                                @click.stop="openScheduleDialog(data)"
                                            />
                                        </template>
                                    </Column>
                                    <Column
                                        header="Actions"
                                        style="width: 8rem"
                                    >
                                        <template #body="{ data }">
                                            <Button
                                                label="Run now"
                                                icon="pi pi-play"
                                                size="small"
                                                :loading="
                                                    triggeringTaskId === data.id
                                                "
                                                :disabled="
                                                    triggeringTaskId !== null
                                                "
                                                @click.stop="triggerTask(data)"
                                            />
                                        </template>
                                    </Column>
                                </DataTable>
                            </template>
                        </Card>
                    </div>

                    <Dialog
                        v-model:visible="scheduleDialogVisible"
                        :header="scheduleDialogTask ? `Schedule: ${scheduleDialogTask.name}` : 'Schedule'"
                        modal
                        class="entry-dialog"
                        :closable="true"
                        @hide="closeScheduleDialog"
                    >
                        <div v-if="scheduleDialogTask" class="flex flex-column gap-3 py-2">
                            <div class="flex flex-wrap gap-2 align-items-center">
                                <span class="font-medium">Preset:</span>
                                <Button
                                    v-for="p in SCHEDULE_PRESETS"
                                    :key="p.cron"
                                    :label="p.label"
                                    size="small"
                                    :severity="scheduleDialogForm.cronExpression === p.cron ? 'primary' : 'secondary'"
                                    @click="setPresetCron(p.cron)"
                                />
                            </div>
                            <div class="flex flex-column gap-1">
                                <label for="schedule-cron">Cron expression</label>
                                <InputText
                                    id="schedule-cron"
                                    v-model="scheduleDialogForm.cronExpression"
                                    placeholder="e.g. 0 0 0 * * *"
                                    class="w-full"
                                    @input="scheduleDialogError = ''"
                                />
                            </div>
                            <div class="flex align-items-center gap-2">
                                <Checkbox
                                    id="schedule-enabled"
                                    v-model="scheduleDialogForm.enabled"
                                    :binary="true"
                                    input-id="schedule-enabled"
                                />
                                <label for="schedule-enabled">Enabled</label>
                            </div>
                            <Message
                                v-if="scheduleDialogError"
                                severity="error"
                                :closable="false"
                                class="mt-0"
                            >
                                {{ scheduleDialogError }}
                            </Message>
                        </div>
                        <template #footer>
                            <Button
                                v-if="scheduleDialogTask?.schedule"
                                label="Remove schedule"
                                severity="danger"
                                text
                                :loading="scheduleDialogSaving"
                                @click="removeScheduleDialog"
                            />
                            <span class="flex-grow-1" />
                            <Button label="Cancel" text severity="secondary" @click="closeScheduleDialog" />
                            <Button
                                label="Save"
                                icon="pi pi-check"
                                :loading="scheduleDialogSaving"
                                @click="saveScheduleDialog"
                            />
                        </template>
                    </Dialog>

                    <div class="col-12">
                        <Card class="queue-card">
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-list"></i>
                                    <span>Queue</span>
                                </div>
                            </template>
                            <template #content>
                                <ProgressSpinner
                                    v-if="executionsQuery.isLoading.value"
                                    style="width: 3rem; height: 3rem"
                                    stroke-width="4"
                                />
                                <DataTable
                                    v-else
                                    :value="executions"
                                    dataKey="id"
                                    stripedRows
                                    class="p-datatable-sm queue-table"
                                    :paginator="executions.length > 50"
                                    :rows="50"
                                >
                                    <Column header="Task" :sortable="false">
                                        <template #body="{ data }">
                                            {{ taskDisplayName(data.task_name) }}
                                        </template>
                                    </Column>
                                    <Column header="Queued at" :sortable="false">
                                        <template #body="{ data }">
                                            {{
                                                formatExecutionTime(
                                                    data.queuedAt
                                                )
                                            }}
                                        </template>
                                    </Column>
                                    <Column header="Duration" :sortable="false">
                                        <template #body="{ data }">
                                            {{
                                                formatDuration(
                                                    data.executionStartedAt,
                                                    data.finishedAt
                                                )
                                            }}
                                        </template>
                                    </Column>
                                    <Column header="Status" :sortable="false">
                                        <template #body="{ data }">
                                            <Tag
                                                :value="getStatusLabel(data.status)"
                                                :severity="
                                                    getStatusSeverity(
                                                        data.status
                                                    )
                                                "
                                            />
                                        </template>
                                    </Column>
                                    <Column header="Actions" style="width: 8rem" :sortable="false">
                                        <template #body="{ data }">
                                            <Button
                                                v-if="isExecutionCancelable(data.status)"
                                                label="Cancel"
                                                icon="pi pi-times"
                                                size="small"
                                                severity="secondary"
                                                :loading="isCancelingExecution(data.id)"
                                                :disabled="cancelMutation.isPending.value"
                                                @click.stop="cancelTaskExecution(data.id)"
                                            />
                                        </template>
                                    </Column>
                                </DataTable>
                                <p
                                    v-if="
                                        !executionsQuery.isLoading.value &&
                                        !executions.length
                                    "
                                    class="text-color-secondary mt-2 mb-0 text-sm"
                                >
                                    Queue is empty.
                                </p>
                            </template>
                        </Card>
                    </div>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.error-message {
    margin-bottom: 1rem;
}

.queue-card {
    margin-top: 1.5rem;
}

.schedule-summary {
    font-size: 0.875rem;
}
</style>
