export interface Security {
    id: number
    symbol: string
    name: string
    currency: string
}

export interface CreateSecurityDTO {
    symbol: string
    name: string
    currency: string
}

export interface UpdateSecurityDTO {
    symbol?: string
    name?: string
    currency?: string
}
