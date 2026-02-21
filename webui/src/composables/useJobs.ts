import { ref, computed, unref, type MaybeRefOrGetter } from 'vue'

export const JOB_STATUS = {
    idle: 'idle',
    running: 'running',
    success: 'success',
    error: 'error'
} as const

export const EXECUTION_STATUS = {
    running: 'running',
    success: 'success',
    error: 'error'
} as const

export interface JobExecution {
    id: number
    startedAt: string
    finishedAt: string | null
    status: string
}

export interface Job {
    id: string
    name: string
    lastExecution: string | null
    status: string
    executions: JobExecution[]
}

// Shared placeholder data so list and detail views stay in sync
const jobs = ref<Job[]>([
    {
        id: 'sync-market-data',
        name: 'Sync market data',
        lastExecution: '2025-02-19T14:30:00',
        status: JOB_STATUS.success,
        executions: [
            { id: 1, startedAt: '2025-02-19T14:30:00', finishedAt: '2025-02-19T14:30:12', status: EXECUTION_STATUS.success },
            { id: 2, startedAt: '2025-02-18T08:00:00', finishedAt: '2025-02-18T08:00:15', status: EXECUTION_STATUS.success },
            { id: 3, startedAt: '2025-02-17T08:00:00', finishedAt: '2025-02-17T08:00:18', status: EXECUTION_STATUS.error }
        ]
    },
    {
        id: 'refresh-rates',
        name: 'Refresh exchange rates',
        lastExecution: '2025-02-20T06:00:00',
        status: JOB_STATUS.success,
        executions: [
            { id: 1, startedAt: '2025-02-20T06:00:00', finishedAt: '2025-02-20T06:00:03', status: EXECUTION_STATUS.success },
            { id: 2, startedAt: '2025-02-19T06:00:00', finishedAt: '2025-02-19T06:00:02', status: EXECUTION_STATUS.success }
        ]
    },
    {
        id: 'cleanup-logs',
        name: 'Cleanup old logs',
        lastExecution: null,
        status: JOB_STATUS.idle,
        executions: []
    }
])

const triggeringJobId = ref<string | null>(null)

export function useJobs() {
    const getJobById = (id: string) => jobs.value.find((j) => j.id === id)

    const triggerJob = (job: Job) => {
        triggeringJobId.value = job.id
        const newExec: JobExecution = {
            id: Date.now(),
            startedAt: new Date().toISOString().slice(0, 19),
            finishedAt: null,
            status: EXECUTION_STATUS.running
        }
        job.executions = [newExec, ...job.executions]
        job.lastExecution = newExec.startedAt
        job.status = JOB_STATUS.running

        setTimeout(() => {
            newExec.finishedAt = new Date().toISOString().slice(0, 19)
            newExec.status = EXECUTION_STATUS.success
            job.status = JOB_STATUS.success
            triggeringJobId.value = null
        }, 1500)
    }

    const getStatusSeverity = (status: string) => {
        if (status === JOB_STATUS.success || status === EXECUTION_STATUS.success) return 'success'
        if (status === JOB_STATUS.error || status === EXECUTION_STATUS.error) return 'danger'
        if (status === JOB_STATUS.running || status === EXECUTION_STATUS.running) return 'info'
        return 'secondary'
    }

    return {
        jobs,
        triggeringJobId,
        getJobById,
        triggerJob,
        getStatusSeverity
    }
}

export function useJobExecutions(jobId: MaybeRefOrGetter<string | undefined>) {
    return computed(() => {
        const id = typeof jobId === 'function' ? jobId() : unref(jobId)
        const j = id ? jobs.value.find((x) => x.id === id) : null
        const list = j?.executions ?? []
        return [...list].sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
    })
}
