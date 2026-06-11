import { apiClient } from './client'

export interface AppStats {
    dbSizeBytes: number
    attachmentsSizeBytes: number
    priceSeries: number
    pricePoints: number
    fxSeries: number
    fxPoints: number
    logLevel: string
}

export const getStats = async (): Promise<AppStats> => {
    const { data } = await apiClient.get('/stats')
    return {
        dbSizeBytes: data.dbSizeBytes ?? 0,
        attachmentsSizeBytes: data.attachmentsSizeBytes ?? 0,
        priceSeries: data.priceSeries ?? 0,
        pricePoints: data.pricePoints ?? 0,
        fxSeries: data.fxSeries ?? 0,
        fxPoints: data.fxPoints ?? 0,
        logLevel: data.logLevel ?? ''
    }
}
