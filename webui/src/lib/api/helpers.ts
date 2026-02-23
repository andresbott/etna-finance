/**
 * Shared API helpers (e.g. 404-as-null for optional resources).
 */

/**
 * Runs an API call and returns null when the response is 404; otherwise throws.
 */
export async function with404Null<T>(apiCall: () => Promise<T>): Promise<T | null> {
    try {
        return await apiCall()
    } catch (err: unknown) {
        if (typeof err === 'object' && err !== null && 'response' in err) {
            const ax = err as { response?: { status?: number } }
            if (ax.response?.status === 404) return null
        }
        throw err
    }
}
