export default class Account {
    constructor({ id = '', name = '', currency = '', type = '' } = {}) {
        this.id = id
        this.name = name
        this.currency = currency
        this.type = type
    }
}
