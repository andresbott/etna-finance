<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import Message from 'primevue/message'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import { useDateFormat } from '@/composables/useDateFormat'
import { useJobs } from '@/composables/useJobs'

const { formatDateTime } = useDateFormat()
const router = useRouter()
const leftSidebarCollapsed = ref(true)

const { jobs, triggerJob, getStatusSeverity, triggeringJobId } = useJobs()

function formatExecutionTime(isoString) {
    if (!isoString) return '-'
    return formatDateTime(isoString)
}

function formatDuration(startedAt, finishedAt) {
    if (!finishedAt) return '—'
    const start = new Date(startedAt).getTime()
    const end = new Date(finishedAt).getTime()
    const sec = Math.round((end - start) / 1000)
    if (sec < 60) return `${sec}s`
    const m = Math.floor(sec / 60)
    const s = sec % 60
    return s ? `${m}m ${s}s` : `${m}m`
}

function getLastExecutionDuration(job) {
    const list = job.executions ?? []
    if (!list.length) return '—'
    const sorted = [...list].sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
    const last = sorted[0]
    return formatDuration(last.startedAt, last.finishedAt)
}

function onRowClick(event) {
    router.push({ name: 'job-detail', params: { id: event.data.id } })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <Message severity="info" :closable="false" class="info-message">
                    <div class="info-content">
                        <i class="pi pi-info-circle"></i>
                        <span>
                            View and manage scheduled and background jobs. Click a row to see execution history. Use
                            <strong>Run now</strong> to trigger a job manually. (Placeholder data.)
                        </span>
                    </div>
                </Message>

                <div class="grid">
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-briefcase"></i>
                                    <span>Jobs</span>
                                </div>
                            </template>
                            <template #content>
                                <DataTable
                                    :value="jobs"
                                    dataKey="id"
                                    stripedRows
                                    class="p-datatable-sm jobs-table clickable-rows"
                                    selectionMode="single"
                                    @row-click="onRowClick"
                                >
                                    <Column field="name" header="Name">
                                        <template #body="{ data }">
                                            <span class="font-semibold">{{ data.name }}</span>
                                        </template>
                                    </Column>
                                    <Column header="Last execution">
                                        <template #body="{ data }">
                                            {{ data.lastExecution ? formatExecutionTime(data.lastExecution) : '-' }}
                                        </template>
                                    </Column>
                                    <Column header="Duration">
                                        <template #body="{ data }">
                                            {{ getLastExecutionDuration(data) }}
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
                                    <Column header="Actions" style="width: 8rem">
                                        <template #body="{ data }">
                                            <Button
                                                label="Run now"
                                                icon="pi pi-play"
                                                size="small"
                                                :loading="triggeringJobId === data.id"
                                                :disabled="data.status === 'running'"
                                                @click.stop="triggerJob(data)"
                                            />
                                        </template>
                                    </Column>
                                </DataTable>
                            </template>
                        </Card>
                    </div>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.info-message {
    margin-bottom: 1.5rem;
}

.info-message :deep(.p-message-wrapper) {
    padding: 1rem 1.25rem;
}

.info-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1rem;
}

.info-content i {
    font-size: 1.25rem;
    flex-shrink: 0;
}

:deep(.jobs-table.clickable-rows .p-datatable-tbody > tr) {
    cursor: pointer;
}
</style>
