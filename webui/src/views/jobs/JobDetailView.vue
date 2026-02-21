<script setup lang="ts">
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import { useDateFormat } from '@/composables/useDateFormat'
import { useJobs, useJobExecutions, type Job } from '@/composables/useJobs'

const route = useRoute()
const router = useRouter()
const { formatDate, formatDateTime } = useDateFormat()
const { getJobById, triggerJob, getStatusSeverity, triggeringJobId } = useJobs()

const leftSidebarCollapsed = ref(true)
const jobId = computed(() => route.params.id as string)
const job = computed<Job | undefined>(() => getJobById(jobId.value))
const executions = useJobExecutions(jobId)

// Placeholder log messages (same for any execution for now)
const placeholderLogs = [
    { time: '14:30:00', level: 'INFO', message: 'Job started' },
    { time: '14:30:01', level: 'INFO', message: 'Connecting to provider...' },
    { time: '14:30:03', level: 'INFO', message: 'Fetched 12 symbols' },
    { time: '14:30:05', level: 'WARN', message: 'Rate limit approaching' },
    { time: '14:30:12', level: 'INFO', message: 'Job completed successfully' }
]

function formatExecutionTime(isoString: string | null) {
    if (!isoString) return '-'
    return formatDateTime(isoString)
}

function formatDuration(startedAt: string, finishedAt: string | null): string {
    if (!finishedAt) return '—'
    const start = new Date(startedAt).getTime()
    const end = new Date(finishedAt).getTime()
    const sec = Math.round((end - start) / 1000)
    if (sec < 60) return `${sec}s`
    const m = Math.floor(sec / 60)
    const s = sec % 60
    return s ? `${m}m ${s}s` : `${m}m`
}

const logsDialogVisible = ref(false)
const logsDialogExecution = ref<{ id: number; startedAt: string } | null>(null)

function openLogs(execution: { id: number; startedAt: string }) {
    logsDialogExecution.value = execution
    logsDialogVisible.value = true
}

const logsDialogHeader = computed(() => {
    const ex = logsDialogExecution.value
    if (!ex) return 'Logs'
    return `Logs — ${formatExecutionTime(ex.startedAt)}`
})

function goBack() {
    router.push({ name: 'jobs' })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="header-nav">
                    <Button
                        icon="pi pi-arrow-left"
                        label="Back to Jobs"
                        text
                        severity="secondary"
                        @click="goBack"
                    />
                </div>

                <div v-if="!job" class="empty-state">
                    <Card>
                        <template #content>
                            <div class="text-center p-4">
                                <i class="pi pi-exclamation-triangle" style="font-size: 2rem; color: var(--p-text-muted-color)"></i>
                                <p class="mt-3">Job not found.</p>
                                <Button label="Back to Jobs" @click="goBack" class="mt-2" />
                            </div>
                        </template>
                    </Card>
                </div>

                <template v-else>
                    <Card>
                        <template #title>
                            <div class="flex align-items-center gap-2 flex-wrap">
                                <span class="font-bold">{{ job.name }}</span>
                                <Tag :value="job.status" :severity="getStatusSeverity(job.status)" />
                                <Button
                                    label="Run now"
                                    icon="pi pi-play"
                                    size="small"
                                    :loading="triggeringJobId === job.id"
                                    :disabled="job.status === 'running'"
                                    class="ml-auto"
                                    @click="triggerJob(job)"
                                />
                            </div>
                        </template>
                        <template #content>
                            <h4 class="section-title">Executions</h4>
                            <DataTable
                                :value="executions"
                                dataKey="id"
                                stripedRows
                                class="p-datatable-sm"
                                :paginator="executions.length > 10"
                                :rows="10"
                            >
                                <Column header="Started">
                                    <template #body="{ data }">
                                        {{ formatExecutionTime(data.startedAt) }}
                                    </template>
                                </Column>
                                <Column header="Duration">
                                    <template #body="{ data }">
                                        {{ formatDuration(data.startedAt, data.finishedAt) }}
                                    </template>
                                </Column>
                                <Column header="Status">
                                    <template #body="{ data }">
                                        <Tag
                                            :value="data.status"
                                            :severity="getStatusSeverity(data.status)"
                                        />
                                    </template>
                                </Column>
                                <Column header="Logs" class="logs-column" style="width: 4rem; min-width: 4rem">
                                    <template #body="{ data }">
                                        <div class="logs-cell">
                                            <Button
                                                label="Logs"
                                                icon="pi pi-list"
                                                text
                                                size="small"
                                                class="p-0 logs-btn"
                                                @click.stop="openLogs(data)"
                                            />
                                        </div>
                                    </template>
                                </Column>
                            </DataTable>
                            <p v-if="!executions.length" class="text-color-secondary mt-2 mb-0 text-sm">
                                No executions yet.
                            </p>

                            <Dialog
                                v-model:visible="logsDialogVisible"
                                :header="logsDialogHeader"
                                modal
                                :style="{ width: '90vw', maxWidth: '56rem' }"
                                class="logs-dialog"
                                :contentStyle="{ maxHeight: '80vh' }"
                            >
                                <div class="log-viewer log-viewer-big">
                                    <div
                                        v-for="(line, idx) in placeholderLogs"
                                        :key="idx"
                                        class="log-line"
                                        :class="'log-' + line.level.toLowerCase()"
                                    >
                                        <span class="log-time">{{ line.time }}</span>
                                        <span class="log-level">{{ line.level }}</span>
                                        <span class="log-message">{{ line.message }}</span>
                                    </div>
                                </div>
                                <template #footer>
                                    <Button label="Close" @click="logsDialogVisible = false" />
                                </template>
                            </Dialog>
                        </template>
                    </Card>
                </template>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.header-nav {
    margin-bottom: 1rem;
}

.section-title {
    font-size: 0.875rem;
    font-weight: 600;
    margin: 0 0 0.75rem 0;
    color: var(--text-color-secondary);
}

.logs-column :deep(.p-datatable-column-header-content) {
    justify-content: flex-end;
}

.logs-cell {
    display: flex;
    justify-content: flex-end;
    align-items: center;
}

.logs-btn {
    min-width: 0;
    padding: 0.25rem;
}

.log-viewer {
    background-color: var(--surface-100);
    border: 1px solid var(--surface-200);
    border-radius: 6px;
    padding: 0.75rem 1rem;
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    overflow-x: auto;
    overflow-y: auto;
}

.log-viewer-big {
    max-height: 70vh;
    min-height: 12rem;
}

.log-line {
    display: flex;
    gap: 1rem;
    padding: 0.15rem 0;
    white-space: pre-wrap;
    word-break: break-word;
}

.log-time {
    flex-shrink: 0;
    color: var(--text-color-secondary);
}

.log-level {
    flex-shrink: 0;
    font-weight: 600;
    min-width: 4rem;
}

.log-info .log-level {
    color: var(--blue-600);
}

.log-warn .log-level {
    color: var(--orange-600);
}

.log-error .log-level {
    color: var(--red-600);
}

.log-message {
    flex: 1;
    color: var(--text-color);
}
</style>
