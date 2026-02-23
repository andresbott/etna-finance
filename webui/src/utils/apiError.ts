/**
 * Extracts a user-facing error message from an API/axios error (e.g. 4xx/5xx response body).
 */
export function getApiErrorMessage(err: unknown): string {
    if (err == null) return 'An error occurred'
    const anyErr = err as { response?: { data?: unknown; status?: number }; message?: string }
    const data = anyErr.response?.data
    if (data != null) {
        if (typeof data === 'string') return data
        if (typeof data === 'object' && data !== null && 'message' in data && typeof (data as { message: unknown }).message === 'string') {
            return (data as { message: string }).message
        }
        if (typeof data === 'object' && data !== null && 'error' in data && typeof (data as { error: unknown }).error === 'string') {
            return (data as { error: string }).error
        }
    }
    const status = anyErr.response?.status
    if (status === 400) return 'Invalid request. Please check your input.'
    if (status === 401) return 'You are not authorized.'
    if (status === 403) return 'You do not have permission.'
    if (status === 404) return 'The resource was not found.'
    if (status && status >= 500) return 'Server error. Please try again later.'
    return anyErr.message && typeof anyErr.message === 'string' ? anyErr.message : 'An error occurred'
}
