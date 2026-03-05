/**
 * Types for instrument providers and related security/backend entities.
 */

export interface InstrumentProvider {
    id: number
    name: string
    description?: string
    icon?: string
}

export interface CreateInstrumentProviderDTO {
    name: string
    description?: string
    icon?: string
}

export interface UpdateInstrumentProviderDTO {
    name?: string
    description?: string
    icon?: string
}
