import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import axios from 'axios'
import { unref, computed } from 'vue'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const ENTRIES_ENDPOINT = `${API_BASE_URL}/fin/report/income-expense`

/**
 * Fetches entries from the API with date range filtering
 * @param {Date} startDate - Start date for filtering
 * @param {Date} endDate - End date for filtering
 * @returns {Promise<Entry[]>}
 */
const fetchIncomeExpense = async (startDate, endDate) => {
    const params = new URLSearchParams({
        startDate,
        endDate
    })

    const { data } = await axios.get(`${ENTRIES_ENDPOINT}?${params}`)

    return data || []
}

export { fetchIncomeExpense }
