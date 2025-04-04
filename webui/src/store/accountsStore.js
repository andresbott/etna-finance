import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

const accountsEndpoint = import.meta.env.VITE_SERVER_URL_V0 + '/fin/accounts'
const accountEndpoint = import.meta.env.VITE_SERVER_URL_V0 + '/fin/account'

export const useAccountsStore = defineStore('accounts', () => {
    const Accounts = ref([])
    const isDataLoaded = ref(false)
    const isLoading = ref(false)
    const isErr = ref(false)

    const processPayload = (payload) => {
        if (payload.items && Array.isArray(payload.items)) {
            // Iterate over each bookmark in the payload
            payload.items.forEach((item) => {
                console.debug(item)
                Accounts.value.push({
                    id: item.id,
                    name: item.name,
                    currency: item.currency,
                    type: item.type
                    // TODO add other data
                })
            })
        } else {
            console.log('No data found in the payload.')
        }
    }

    // load all Accounts info
    const Load = () => {
        isDataLoaded.value = true
        isErr.value = false
        isLoading.value = true
        Accounts.value.splice(0, Accounts.value.length)
        axios
            .get(accountsEndpoint, {})
            .then((res) => {
                if (res.status === 200) {
                    processPayload(res.data)
                } else {
                    isErr.value = true
                    console.log(res)
                    // error?
                }
            })
            .catch((err) => {
                isErr.value = true
                console.log(err)
            })
            .finally(() => {
                isLoading.value = false
            })
    }

    const Add = (url, name) => {
        const bookmarkPayload = {
            url: url,
            name: name
        }
        isErr.value = false
        isLoading.value = true

        return axios
            .post(accountsEndpoint, bookmarkPayload, {
                headers: {
                    'Content-Type': 'application/json'
                }
            })
            .then((res) => {
                console.log(res)
                if (res.status === 200) {
                    const item = res.data
                    if (item) {
                        Accounts.value.push({
                            id: item.id
                            // TODO add other data
                        })
                    }
                } else {
                    isErr.value = true
                    console.log(res)
                    // error?
                }
            })
            .catch((err) => {
                isErr.value = true
                console.log(err)
            })
            .finally(() => {
                isLoading.value = false
            })
    }

    const removeItem = (id) => {
        const index = Accounts.value.findIndex((item) => item.id === id)
        Accounts.value.splice(index, 1)
    }

    const Delete = (id) => {
        isErr.value = false
        isLoading.value = true
        return axios
            .delete(accountEndpoint + '/' + id)
            .then((res) => {
                if (res.status === 200) {
                    removeItem(id)
                } else {
                    isErr.value = true
                    console.log(res)
                    // error?
                }
            })
            .catch((err) => {
                isErr.value = true
                console.log(err)
            })
            .finally(() => {
                isLoading.value = false
            })
    }

    const Reset = () => {
        Accounts.value = []
        isDataLoaded.value = false
    }

    return {
        Accounts,
        isLoading, // if the store is in a loading state
        isErr,
        Load, // initial load of bookmarks
        Add,
        Delete,
        Reset // reset on logout
    }
})
