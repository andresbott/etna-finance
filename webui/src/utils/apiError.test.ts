import { describe, it, expect } from 'vitest'
import { getApiErrorMessage } from './apiError'

describe('getApiErrorMessage', () => {
  // --- null/undefined input ---
  it('returns generic message for null', () => {
    expect(getApiErrorMessage(null)).toBe('An error occurred')
  })

  it('returns generic message for undefined', () => {
    expect(getApiErrorMessage(undefined)).toBe('An error occurred')
  })

  // --- response.data extractions ---
  it('returns response.data when it is a string', () => {
    const err = { response: { data: 'Something broke' } }
    expect(getApiErrorMessage(err)).toBe('Something broke')
  })

  it('returns response.data.message when present', () => {
    const err = { response: { data: { message: 'Validation failed' } } }
    expect(getApiErrorMessage(err)).toBe('Validation failed')
  })

  it('returns response.data.error when present', () => {
    const err = { response: { data: { error: 'Bad gateway' } } }
    expect(getApiErrorMessage(err)).toBe('Bad gateway')
  })

  it('prefers data.message over data.error when both present', () => {
    const err = { response: { data: { message: 'msg', error: 'err' } } }
    expect(getApiErrorMessage(err)).toBe('msg')
  })

  it('ignores non-string data.message and falls through to data.error', () => {
    const err = { response: { data: { message: 123, error: 'fallback error' } } }
    expect(getApiErrorMessage(err)).toBe('fallback error')
  })

  it('ignores non-string data.error and falls through to status code', () => {
    const err = { response: { data: { error: 42 }, status: 400 } }
    expect(getApiErrorMessage(err)).toBe('Invalid request. Please check your input.')
  })

  // --- status code mappings ---
  it('maps status 400', () => {
    const err = { response: { status: 400 } }
    expect(getApiErrorMessage(err)).toBe('Invalid request. Please check your input.')
  })

  it('maps status 401', () => {
    const err = { response: { status: 401 } }
    expect(getApiErrorMessage(err)).toBe('You are not authorized.')
  })

  it('maps status 403', () => {
    const err = { response: { status: 403 } }
    expect(getApiErrorMessage(err)).toBe('You do not have permission.')
  })

  it('maps status 404', () => {
    const err = { response: { status: 404 } }
    expect(getApiErrorMessage(err)).toBe('The resource was not found.')
  })

  it('maps status 500', () => {
    const err = { response: { status: 500 } }
    expect(getApiErrorMessage(err)).toBe('Server error. Please try again later.')
  })

  it('maps status 502', () => {
    const err = { response: { status: 502 } }
    expect(getApiErrorMessage(err)).toBe('Server error. Please try again later.')
  })

  it('maps status 503', () => {
    const err = { response: { status: 503 } }
    expect(getApiErrorMessage(err)).toBe('Server error. Please try again later.')
  })

  // --- status code with data present (data takes priority) ---
  it('data.message takes priority over status code', () => {
    const err = { response: { data: { message: 'Custom msg' }, status: 500 } }
    expect(getApiErrorMessage(err)).toBe('Custom msg')
  })

  // --- fallback to err.message ---
  it('falls back to err.message when no response', () => {
    const err = { message: 'Network Error' }
    expect(getApiErrorMessage(err)).toBe('Network Error')
  })

  it('falls back to err.message when response has no data and unknown status', () => {
    const err = { response: { status: 418 }, message: 'I am a teapot' }
    expect(getApiErrorMessage(err)).toBe('I am a teapot')
  })

  // --- generic fallback ---
  it('returns generic message for object without message or response', () => {
    expect(getApiErrorMessage({})).toBe('An error occurred')
  })

  it('returns generic message for non-object input (number)', () => {
    expect(getApiErrorMessage(42)).toBe('An error occurred')
  })

  it('returns generic message for non-object input (boolean)', () => {
    expect(getApiErrorMessage(true)).toBe('An error occurred')
  })

  it('returns generic message when err.message is not a string', () => {
    const err = { message: 123 }
    expect(getApiErrorMessage(err)).toBe('An error occurred')
  })
})
