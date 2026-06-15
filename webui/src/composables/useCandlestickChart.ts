// webui/src/composables/useCandlestickChart.ts
import { computed, type Ref } from 'vue'
import type { OhlcvData } from '@/composables/useTsdbPrices'
import type { IndicatorParams, BollingerResult, MACDResult } from '@/composables/useIndicators'
import { getGreenColor, getRedColor } from '@/utils/chartColors'
import { useDateFormat } from '@/composables/useDateFormat'
import { formatCompact } from '@/utils/currency'

interface IndicatorData {
    sma1: (number | null)[]
    sma2: (number | null)[]
    ema1: (number | null)[]
    ema2: (number | null)[]
    bollinger: BollingerResult
    rsi: (number | null)[]
    macd: MACDResult
}

// Read green/red from CSS custom properties at runtime for theme-awareness.
// Evaluated inside computed() so they react to theme changes.
const GREEN = () => getGreenColor()
const RED = () => getRedColor()

// Fixed-size sections (px)
const TOP_PAD = 20
const GAP = 12
const DATAZOOM_HEIGHT = 36
const BOTTOM_PAD = 8

// Proportional weights for distributing available space
const MAIN_WEIGHT = 5
const VOLUME_WEIGHT = 1
const SUB_PANEL_WEIGHT = 2

export function useCandlestickChart(
    ohlcv: Ref<OhlcvData>,
    indicators: Ref<IndicatorData>,
    params: Ref<IndicatorParams>,
    availableHeight: Ref<number>,
    peRatio?: Ref<(number | null)[]>,
    peEnabled?: Ref<boolean>
) {
    const { formatDate } = useDateFormat()

    const chartOption = computed(() => {
        const { dates, opens, highs, lows, closes, volumes } = ohlcv.value
        if (dates.length === 0) return {}

        const rsiEnabled = params.value.rsi.enabled
        const macdEnabled = params.value.macd.enabled
        const peActive = peEnabled?.value && (peRatio?.value?.length ?? 0) > 0
        const subPanelCount = (rsiEnabled ? 1 : 0) + (macdEnabled ? 1 : 0) + (peActive ? 1 : 0)

        // Distribute available space proportionally
        const fixedSpace = TOP_PAD + DATAZOOM_HEIGHT + BOTTOM_PAD + GAP * (2 + subPanelCount)
        const flexSpace = Math.max(200, availableHeight.value - fixedSpace)
        const totalWeight = MAIN_WEIGHT + VOLUME_WEIGHT + subPanelCount * SUB_PANEL_WEIGHT
        const unit = flexSpace / totalWeight

        const mainHeight = Math.round(unit * MAIN_WEIGHT)
        const volumeHeight = Math.round(unit * VOLUME_WEIGHT)
        const subPanelHeight = Math.round(unit * SUB_PANEL_WEIGHT)

        const grids: any[] = []
        const xAxes: any[] = []
        const yAxes: any[] = []
        const series: any[] = []
        let top = TOP_PAD

        // Grid 0: Main candlestick
        grids.push({ left: 60, right: 16, top, height: mainHeight })
        xAxes.push({ type: 'category', data: dates, gridIndex: 0, show: false, boundaryGap: true })
        yAxes.push({
            type: 'value', gridIndex: 0, scale: true,
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.3 } }
        })
        top += mainHeight + GAP

        // Candlestick series
        const candlestickData = dates.map((_, i) => [opens[i], closes[i], lows[i], highs[i]])
        series.push({
            type: 'candlestick',
            data: candlestickData,
            xAxisIndex: 0,
            yAxisIndex: 0,
            itemStyle: {
                color: GREEN(),
                color0: RED(),
                borderColor: GREEN(),
                borderColor0: RED()
            }
        })

        // Overlay indicators on main grid
        if (params.value.sma.enabled && indicators.value.sma1.length) {
            series.push({
                type: 'line', data: indicators.value.sma1, xAxisIndex: 0, yAxisIndex: 0,
                smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#f59e0b' },
                name: `SMA ${params.value.sma.period1}`
            })
            if (params.value.sma.showSecond && indicators.value.sma2.length) {
                series.push({
                    type: 'line', data: indicators.value.sma2, xAxisIndex: 0, yAxisIndex: 0,
                    smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#8b5cf6' },
                    name: `SMA ${params.value.sma.period2}`
                })
            }
        }
        if (params.value.ema.enabled && indicators.value.ema1.length) {
            series.push({
                type: 'line', data: indicators.value.ema1, xAxisIndex: 0, yAxisIndex: 0,
                smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#06b6d4' },
                name: `EMA ${params.value.ema.period1}`
            })
            if (params.value.ema.showSecond && indicators.value.ema2.length) {
                series.push({
                    type: 'line', data: indicators.value.ema2, xAxisIndex: 0, yAxisIndex: 0,
                    smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#ec4899' },
                    name: `EMA ${params.value.ema.period2}`
                })
            }
        }
        if (params.value.bollinger.enabled && indicators.value.bollinger.upper.length) {
            const bb = indicators.value.bollinger
            series.push({
                type: 'line', data: bb.upper, xAxisIndex: 0, yAxisIndex: 0,
                smooth: false, showSymbol: false, lineStyle: { width: 1, color: '#94a3b8', type: 'dashed' },
                name: 'BB Upper'
            })
            series.push({
                type: 'line', data: bb.middle, xAxisIndex: 0, yAxisIndex: 0,
                smooth: false, showSymbol: false, lineStyle: { width: 1, color: '#94a3b8' },
                name: 'BB Middle'
            })
            series.push({
                type: 'line', data: bb.lower, xAxisIndex: 0, yAxisIndex: 0,
                smooth: false, showSymbol: false, lineStyle: { width: 1, color: '#94a3b8', type: 'dashed' },
                name: 'BB Lower',
                areaStyle: { color: 'rgba(148,163,184,0.08)', origin: 'auto' }
            })
        }

        // Grid 1: Volume
        const volGridIndex = grids.length
        grids.push({ left: 60, right: 16, top, height: volumeHeight })
        xAxes.push({ type: 'category', data: dates, gridIndex: volGridIndex, show: false, boundaryGap: true })
        yAxes.push({
            type: 'value', gridIndex: volGridIndex, scale: true, show: false,
            splitLine: { show: false }
        })
        top += volumeHeight + GAP

        const volumeColors = dates.map((_, i) => (closes[i] >= opens[i] ? GREEN() : RED()))
        series.push({
            type: 'bar',
            data: volumes.map((v, i) => ({
                value: v,
                itemStyle: { color: volumeColors[i], opacity: 0.5 }
            })),
            xAxisIndex: volGridIndex,
            yAxisIndex: volGridIndex,
            name: 'Volume'
        })

        // Grid: RSI (optional)
        if (rsiEnabled && indicators.value.rsi.length) {
            const rsiGridIndex = grids.length
            grids.push({ left: 60, right: 16, top, height: subPanelHeight })
            xAxes.push({ type: 'category', data: dates, gridIndex: rsiGridIndex, show: false, boundaryGap: true })
            yAxes.push({
                type: 'value', gridIndex: rsiGridIndex, scale: false, min: 0, max: 100,
                splitLine: { lineStyle: { type: 'dashed', opacity: 0.3 } },
                axisLabel: { fontSize: 10 }
            })
            top += subPanelHeight + GAP

            series.push({
                type: 'line', data: indicators.value.rsi, xAxisIndex: rsiGridIndex, yAxisIndex: rsiGridIndex,
                smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#a855f7' },
                name: 'RSI',
                markLine: {
                    silent: true, symbol: 'none',
                    lineStyle: { type: 'dashed', color: '#9ca3af', width: 1 },
                    data: [
                        { yAxis: 70, label: { formatter: '70', position: 'end', fontSize: 9 } },
                        { yAxis: 30, label: { formatter: '30', position: 'end', fontSize: 9 } }
                    ]
                }
            })
        }

        // Grid: MACD (optional)
        if (macdEnabled && indicators.value.macd.macd.length) {
            const macdGridIndex = grids.length
            grids.push({ left: 60, right: 16, top, height: subPanelHeight })
            xAxes.push({
                type: 'category', data: dates, gridIndex: macdGridIndex,
                axisLabel: { rotate: 45, fontSize: 10, formatter: (v: string) => formatDate(v) }, boundaryGap: true
            })
            yAxes.push({
                type: 'value', gridIndex: macdGridIndex, scale: true,
                splitLine: { lineStyle: { type: 'dashed', opacity: 0.3 } },
                axisLabel: { fontSize: 10 }
            })

            series.push({
                type: 'line', data: indicators.value.macd.macd,
                xAxisIndex: macdGridIndex, yAxisIndex: macdGridIndex,
                smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#3b82f6' },
                name: 'MACD'
            })
            series.push({
                type: 'line', data: indicators.value.macd.signal,
                xAxisIndex: macdGridIndex, yAxisIndex: macdGridIndex,
                smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#f97316' },
                name: 'Signal'
            })
            series.push({
                type: 'bar',
                data: (indicators.value.macd.histogram ?? []).map(v => ({
                    value: v,
                    itemStyle: { color: (v ?? 0) >= 0 ? GREEN() : RED(), opacity: 0.6 }
                })),
                xAxisIndex: macdGridIndex, yAxisIndex: macdGridIndex,
                name: 'Histogram'
            })
            top += subPanelHeight + GAP
        }

        // Grid: P/E Ratio (optional)
        if (peActive && peRatio?.value) {
            const peGridIndex = grids.length
            grids.push({ left: 60, right: 16, top, height: subPanelHeight })
            xAxes.push({ type: 'category', data: dates, gridIndex: peGridIndex, show: false, boundaryGap: true })
            yAxes.push({
                type: 'value', gridIndex: peGridIndex, scale: true,
                splitLine: { lineStyle: { type: 'dashed', opacity: 0.3 } },
                axisLabel: { fontSize: 10 }
            })
            // Advance the cursor to stay symmetric with the other panel blocks, so a
            // panel added after P/E keeps the correct offset.
            // eslint-disable-next-line no-useless-assignment
            top += subPanelHeight + GAP

            series.push({
                type: 'line', data: peRatio.value, xAxisIndex: peGridIndex, yAxisIndex: peGridIndex,
                smooth: false, showSymbol: false, lineStyle: { width: 1.5, color: '#0ea5e9' },
                name: 'P/E Ratio'
            })
        }

        // Show x-axis labels on the last grid
        const lastXAxisIndex = xAxes.length - 1
        xAxes[lastXAxisIndex] = {
            ...xAxes[lastXAxisIndex],
            show: true,
            axisLabel: { rotate: 45, fontSize: 10, formatter: (v: string) => formatDate(v) }
        }

        // dataZoom controls all xAxes
        const dataZoom = [
            {
                type: 'inside',
                xAxisIndex: xAxes.map((_: any, i: number) => i),
                start: 0,
                end: 100
            },
            {
                type: 'slider',
                xAxisIndex: xAxes.map((_: any, i: number) => i),
                bottom: BOTTOM_PAD,
                height: DATAZOOM_HEIGHT,
                start: 0,
                end: 100
            }
        ]

        return {
            animation: true,
            animationDuration: 300,
            grid: grids,
            xAxis: xAxes,
            yAxis: yAxes,
            series,
            dataZoom,
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' },
                formatter(params: any) {
                    if (!Array.isArray(params) || params.length === 0) return ''
                    const date = formatDate(params[0].axisValue ?? params[0].name ?? '')
                    let html = `<div style="font-weight:600;margin-bottom:4px">${date}</div>`
                    // Render OHLC first, then Volume, then the remaining series so the
                    // tooltip order stays readable regardless of ECharts' series order.
                    const candle = params.find((p: any) => p.seriesType === 'candlestick' && Array.isArray(p.data))
                    if (candle) {
                        // ECharts prepends the data index to candlestick tooltip data,
                        // so p.data is [dataIndex, open, close, low, high] — skip the index.
                        const [, open, close, low, high] = candle.data.map((v: number) => v.toFixed(2))
                        html += `${candle.marker ?? ''} O: ${open}  H: ${high}  L: ${low}  C: ${close}<br/>`
                    }
                    const volume = params.find((p: any) => p.seriesName === 'Volume')
                    if (volume) {
                        const raw = typeof volume.value === 'number' ? volume.value
                            : typeof volume.value?.value === 'number' ? volume.value.value
                            : null
                        html += `${volume.marker ?? ''} Volume: ${raw === null ? volume.value : formatCompact(raw)}<br/>`
                    }
                    for (const p of params) {
                        if (p === candle || p === volume) continue
                        const marker = p.marker ?? ''
                        const name = p.seriesName ?? ''
                        const raw = typeof p.value === 'number' ? p.value
                            : typeof p.value?.value === 'number' ? p.value.value
                            : null
                        const val = raw === null ? p.value : raw.toFixed(3)
                        html += `${marker} ${name}: ${val}<br/>`
                    }
                    return html
                }
            },
            axisPointer: {
                link: [{ xAxisIndex: 'all' }]
            }
        }
    })

    return { chartOption }
}
