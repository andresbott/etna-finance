import { describe, it, expect } from 'vitest'
import { with404Null } from '@/lib/api/helpers'

describe('with404Null', () => {
    it('returns the resolved value on success', async () => {
        const result = await with404Null(() => Promise.resolve('ok'))
        expect(result).toBe('ok')
    })

    it('returns null when the error has response.status === 404', async () => {
        const err = { response: { status: 404 } }
        const result = await with404Null(() => Promise.reject(err))
        expect(result).toBeNull()
    })

    it('re-throws when response.status is not 404', async () => {
        const err = { response: { status: 500 } }
        await expect(with404Null(() => Promise.reject(err))).rejects.toBe(err)
    })

    it('re-throws when the error has no response property', async () => {
        const err = new Error('network error')
        await expect(with404Null(() => Promise.reject(err))).rejects.toBe(err)
    })

    it('re-throws when error is a plain string', async () => {
        await expect(with404Null(() => Promise.reject('boom'))).rejects.toBe('boom')
    })

    it('re-throws when error is null', async () => {
        await expect(with404Null(() => Promise.reject(null))).rejects.toBeNull()
    })

    it('returns null-ish values from a successful call without confusing them as errors', async () => {
        const result = await with404Null(() => Promise.resolve(null))
        expect(result).toBeNull()

        const result2 = await with404Null(() => Promise.resolve(undefined))
        expect(result2).toBeUndefined()
    })
})
