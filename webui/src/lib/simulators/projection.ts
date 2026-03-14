import type { CaseStudy, PortfolioSimulatorParams, RealEstateSimulatorParams, BuyVsRentSimulatorParams } from '@/lib/api/ToolsData'
import { computePortfolioNetWorth20Y } from './portfolio'
import { computeRealEstateNetWorth20Y } from './realEstate'
import { computeBuyVsRentNetWorth20Y } from './buyVsRent'

export function computeNetWorth20Y(caseStudy: CaseStudy): number[] {
    switch (caseStudy.toolType) {
        case 'portfolio-simulator':
            return computePortfolioNetWorth20Y(caseStudy.params as PortfolioSimulatorParams)
        case 'real-estate-simulator':
            return computeRealEstateNetWorth20Y(caseStudy.params as RealEstateSimulatorParams)
        case 'buy-vs-rent-simulator':
            return computeBuyVsRentNetWorth20Y(caseStudy.params as BuyVsRentSimulatorParams)
        default:
            return Array(21).fill(0)
    }
}
