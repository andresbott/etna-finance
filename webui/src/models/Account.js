export default class Account {
    constructor({ id = '', name = '', currency = '', type = '', icon = '' } = {}) {
        this.id = id
        this.name = name
        this.currency = currency
        this.type = type
        this.icon = icon
    }
}
