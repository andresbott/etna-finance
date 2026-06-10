export interface Instrument {
    id: number
    symbol: string
    name: string
    currency: string
    notes: string
    type: string
    exchange: string
}

export interface CreateInstrumentDTO {
    symbol: string
    name: string
    currency: string
    notes: string
    type: string
    exchange: string
}

export interface UpdateInstrumentDTO {
    symbol?: string
    name?: string
    currency?: string
    notes?: string
    type?: string
    exchange?: string
}

export interface InstrumentLookup {
    name: string
    currency: string
    type: string
    exchange: string
    notes: string
}
