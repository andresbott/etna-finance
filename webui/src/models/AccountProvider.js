export default class AccountProvider {
    constructor({ id = '', name = '', description = '', icon = '', accounts = [] } = {}) {
        this.id = id
        this.name = name
        this.description = description
        this.icon = icon
        this.accounts = accounts
    }
}
