// webui/src/composables/useIndicators.ts
import { computed, type Ref } from 'vue'

// --- Pure computation functions ---

export function computeSMA(data: number[], period: number): (number | null)[] {
    const result: (number | null)[] = []
    for (let i = 0; i < data.length; i++) {
        if (i < period - 1) {
            result.push(null)
        } else {
            let sum = 0
            for (let j = i - period + 1; j <= i; j++) sum += data[j]
            result.push(sum / period)
        }
    }
    return result
}

export function computeEMA(data: number[], period: number): (number | null)[] {
    const result: (number | null)[] = []
    const k = 2 / (period + 1)
    let ema: number | null = null
    for (let i = 0; i < data.length; i++) {
        if (i < period - 1) {
            result.push(null)
        } else if (i === period - 1) {
            let sum = 0
            for (let j = 0; j < period; j++) sum += data[j]
            ema = sum / period
            result.push(ema)
        } else {
            ema = data[i] * k + ema! * (1 - k)
            result.push(ema)
        }
    }
    return result
}

export interface BollingerResult {
    middle: (number | null)[]
    upper: (number | null)[]
    lower: (number | null)[]
}

export function computeBollinger(data: number[], period: number, stdDev: number): BollingerResult {
    const middle = computeSMA(data, period)
    const upper: (number | null)[] = []
    const lower: (number | null)[] = []
    for (let i = 0; i < data.length; i++) {
        const m = middle[i]
        if (m === null) {
            upper.push(null)
            lower.push(null)
        } else {
            let sumSq = 0
            for (let j = i - period + 1; j <= i; j++) {
                sumSq += (data[j] - m) ** 2
            }
            const sd = Math.sqrt(sumSq / period)
            upper.push(m + stdDev * sd)
            lower.push(m - stdDev * sd)
        }
    }
    return { middle, upper, lower }
}

export function computeRSI(data: number[], period: number): (number | null)[] {
    const result: (number | null)[] = [null] // first element has no change
    if (data.length < 2) return result

    const gains: number[] = []
    const losses: number[] = []
    for (let i = 1; i < data.length; i++) {
        const change = data[i] - data[i - 1]
        gains.push(change > 0 ? change : 0)
        losses.push(change < 0 ? -change : 0)
    }

    // Need `period` changes to compute first RSI
    for (let i = 0; i < gains.length; i++) {
        if (i < period - 1) {
            result.push(null)
        } else if (i === period - 1) {
            let avgGain = 0
            let avgLoss = 0
            for (let j = 0; j < period; j++) {
                avgGain += gains[j]
                avgLoss += losses[j]
            }
            avgGain /= period
            avgLoss /= period
            const rs = avgLoss === 0 ? 100 : avgGain / avgLoss
            result.push(100 - 100 / (1 + rs))
        } else {
            let avgGain = 0
            let avgLoss = 0
            for (let j = i - period + 1; j <= i; j++) {
                avgGain += gains[j]
                avgLoss += losses[j]
            }
            avgGain /= period
            avgLoss /= period
            const rs = avgLoss === 0 ? 100 : avgGain / avgLoss
            result.push(100 - 100 / (1 + rs))
        }
    }
    return result
}

export interface MACDResult {
    macd: (number | null)[]
    signal: (number | null)[]
    histogram: (number | null)[]
}

export function computeMACD(
    data: number[],
    fastPeriod: number,
    slowPeriod: number,
    signalPeriod: number
): MACDResult {
    const fastEma = computeEMA(data, fastPeriod)
    const slowEma = computeEMA(data, slowPeriod)

    const macdLine: (number | null)[] = []
    for (let i = 0; i < data.length; i++) {
        if (fastEma[i] !== null && slowEma[i] !== null) {
            macdLine.push(fastEma[i]! - slowEma[i]!)
        } else {
            macdLine.push(null)
        }
    }

    // Compute signal line as EMA of non-null MACD values
    const nonNullMacd: number[] = []
    const nonNullIndices: number[] = []
    for (let i = 0; i < macdLine.length; i++) {
        if (macdLine[i] !== null) {
            nonNullMacd.push(macdLine[i]!)
            nonNullIndices.push(i)
        }
    }

    const signalEma = computeEMA(nonNullMacd, signalPeriod)

    const signal: (number | null)[] = new Array(data.length).fill(null)
    const histogram: (number | null)[] = new Array(data.length).fill(null)

    for (let j = 0; j < nonNullIndices.length; j++) {
        const idx = nonNullIndices[j]
        signal[idx] = signalEma[j]
        if (macdLine[idx] !== null && signalEma[j] !== null) {
            histogram[idx] = macdLine[idx]! - signalEma[j]!
        }
    }

    return { macd: macdLine, signal, histogram }
}

// --- Reactive composable ---

export interface IndicatorParams {
    sma: { enabled: boolean; period1: number; period2: number; showSecond: boolean }
    ema: { enabled: boolean; period1: number; period2: number; showSecond: boolean }
    bollinger: { enabled: boolean; period: number; stdDev: number }
    rsi: { enabled: boolean; period: number }
    macd: { enabled: boolean; fast: number; slow: number; signal: number }
}

export function useIndicators(closes: Ref<number[]>, params: Ref<IndicatorParams>) {
    const sma1 = computed(() =>
        params.value.sma.enabled ? computeSMA(closes.value, params.value.sma.period1) : []
    )
    const sma2 = computed(() =>
        params.value.sma.enabled && params.value.sma.showSecond
            ? computeSMA(closes.value, params.value.sma.period2)
            : []
    )
    const ema1 = computed(() =>
        params.value.ema.enabled ? computeEMA(closes.value, params.value.ema.period1) : []
    )
    const ema2 = computed(() =>
        params.value.ema.enabled && params.value.ema.showSecond
            ? computeEMA(closes.value, params.value.ema.period2)
            : []
    )
    const bollinger = computed(() =>
        params.value.bollinger.enabled
            ? computeBollinger(closes.value, params.value.bollinger.period, params.value.bollinger.stdDev)
            : { middle: [], upper: [], lower: [] }
    )
    const rsi = computed(() =>
        params.value.rsi.enabled ? computeRSI(closes.value, params.value.rsi.period) : []
    )
    const macd = computed(() =>
        params.value.macd.enabled
            ? computeMACD(closes.value, params.value.macd.fast, params.value.macd.slow, params.value.macd.signal)
            : { macd: [], signal: [], histogram: [] }
    )

    return { sma1, sma2, ema1, ema2, bollinger, rsi, macd }
}
