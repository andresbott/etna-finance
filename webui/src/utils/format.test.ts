import { describe, it, expect } from 'vitest'
import { formatPct, getChangeSeverity } from './format'

describe('formatPct', () => {
  it('formats positive percentage with + prefix', () => {
    expect(formatPct(2.5)).toBe('+2.50%')
  })
  it('formats negative percentage', () => {
    expect(formatPct(-1.75)).toBe('-1.75%')
  })
  it('formats zero with + prefix', () => {
    expect(formatPct(0)).toBe('+0.00%')
  })
  it('returns dash for null', () => {
    expect(formatPct(null)).toBe('-')
  })
  it('returns dash for undefined', () => {
    expect(formatPct(undefined)).toBe('-')
  })
})

describe('getChangeSeverity', () => {
  it('returns success for positive', () => {
    expect(getChangeSeverity(1)).toBe('success')
  })
  it('returns danger for negative', () => {
    expect(getChangeSeverity(-1)).toBe('danger')
  })
  it('returns secondary for zero', () => {
    expect(getChangeSeverity(0)).toBe('secondary')
  })
  it('returns secondary for null', () => {
    expect(getChangeSeverity(null)).toBe('secondary')
  })
  it('returns secondary for undefined', () => {
    expect(getChangeSeverity(undefined)).toBe('secondary')
  })
})
