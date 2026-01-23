import IconSelect from './IconSelect.vue'

export default {
    title: 'Components/Common/IconSelect',
    component: IconSelect,
    tags: ['autodocs'],
    argTypes: {
        modelValue: {
            control: 'text',
            description: 'The selected icon class (e.g., "pi-wallet")'
        },
        placeholder: {
            control: 'text',
            description: 'Placeholder text when no icon is selected'
        }
    }
}

export const Default = {
    args: {
        modelValue: 'pi-wallet',
        placeholder: 'Select Icon'
    }
}

export const WithCreditCard = {
    args: {
        modelValue: 'pi-credit-card',
        placeholder: 'Select Icon'
    }
}

export const WithChartLine = {
    args: {
        modelValue: 'pi-chart-line',
        placeholder: 'Select Icon'
    }
}

export const WithBuilding = {
    args: {
        modelValue: 'pi-building',
        placeholder: 'Select Icon'
    }
}

export const WithMoneyBill = {
    args: {
        modelValue: 'pi-money-bill',
        placeholder: 'Select Icon'
    }
}
