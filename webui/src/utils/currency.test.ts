import { describe, it, expect } from 'vitest'
import { formatCurrency, formatAmount } from './currency'

describe('formatCurrency', () => {
  it('formats positive number with defaults', () => {
    expect(formatCurrency(1234.5)).toBe('1,234.50')
  })

  it('formats zero', () => {
    expect(formatCurrency(0)).toBe('0.00')
  })

  it('formats negative number', () => {
    expect(formatCurrency(-500.1)).toBe('-500.10')
  })

  it('respects custom fraction digits', () => {
    expect(formatCurrency(1.5, 0, 0)).toBe('2')
  })

  it('formats with more decimal places when requested', () => {
    expect(formatCurrency(1.23456, 2, 4)).toBe('1.2346')
  })

  it('pads with zeros to meet minimum fraction digits', () => {
    expect(formatCurrency(5, 3, 3)).toBe('5.000')
  })

  it('formats very large numbers', () => {
    expect(formatCurrency(1234567890.12)).toBe('1,234,567,890.12')
  })

  it('formats small decimal numbers', () => {
    expect(formatCurrency(0.01)).toBe('0.01')
  })

  it('formats negative zero as zero', () => {
    expect(formatCurrency(-0)).toBe('0.00')
  })

  it('formats tiny negative values that round to zero as zero', () => {
    expect(formatCurrency(-0.001)).toBe('0.00')
    expect(formatCurrency(-0.004)).toBe('0.00')
    expect(formatCurrency(-0.0000001)).toBe('0.00')
  })

  it('still formats values that round to -0.01', () => {
    expect(formatCurrency(-0.006)).toBe('-0.01')
    expect(formatCurrency(-0.01)).toBe('-0.01')
  })
})

describe('formatAmount', () => {
  it('delegates to formatCurrency with defaults', () => {
    expect(formatAmount(42)).toBe('42.00')
  })

  it('formats large numbers', () => {
    expect(formatAmount(1000000)).toBe('1,000,000.00')
  })

  it('formats zero', () => {
    expect(formatAmount(0)).toBe('0.00')
  })

  it('formats negative numbers', () => {
    expect(formatAmount(-123.45)).toBe('-123.45')
  })

  it('rounds to two decimal places', () => {
    expect(formatAmount(1.999)).toBe('2.00')
  })

  it('formats negative zero as zero', () => {
    expect(formatAmount(-0)).toBe('0.00')
  })
})
