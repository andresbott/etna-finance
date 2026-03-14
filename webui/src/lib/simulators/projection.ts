import type { CaseStudy, PortfolioSimulatorParams, RealEstateSimulatorParams, BuyVsRentSimulatorParams } from '@/lib/api/ToolsData'
import { computePortfolioNetWorth20Y } from './portfolio'
import { computeRealEstateNetWorth20Y } from './realEstate'
import { computeBuyVsRentNetWorth20Y } from './buyVsRent'

export function computeNetWorth20Y(caseStudy: CaseStudy, initialAmountOverride?: number, durationYears?: number): number[] {
    switch (caseStudy.toolType) {
        case 'portfolio-simulator': {
            const params = caseStudy.params as PortfolioSimulatorParams
            const effective = initialAmountOverride != null
                ? { ...params, initialContribution: initialAmountOverride }
                : params
            return computePortfolioNetWorth20Y(effective, durationYears)
        }
        case 'real-estate-simulator':
            return computeRealEstateNetWorth20Y(caseStudy.params as RealEstateSimulatorParams)
        case 'buy-vs-rent-simulator':
            return computeBuyVsRentNetWorth20Y(caseStudy.params as BuyVsRentSimulatorParams)
        default:
            return Array((durationYears ?? 20) + 1).fill(0)
    }
}
