import axios from 'axios'
const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0

export const apiClient = axios.create({
    baseURL: API_BASE_URL, // adjust
    headers: {
        'Content-Type': 'application/json'
    }
})
