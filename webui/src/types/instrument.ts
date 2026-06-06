export interface Instrument {
    id: number
    symbol: string
    name: string
    currency: string
    notes: string
}

export interface CreateInstrumentDTO {
    symbol: string
    name: string
    currency: string
    notes: string
}

export interface UpdateInstrumentDTO {
    symbol?: string
    name?: string
    currency?: string
    notes?: string
}
