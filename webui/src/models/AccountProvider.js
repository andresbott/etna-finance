export default class AccountProvider {
    constructor({ id = '', name = '', description = '', accounts = [] } = {}) {
        this.id = id
        this.name = name
        this.description = description
        this.accounts = accounts
    }
}
