import AccountSelector from './AccountSelector.vue'
import { ref } from 'vue'
import AccountProvider from '@/models/AccountProvider'
import Account from '@/models/Account'
import { queryClient } from '../../.storybook/preview.js'

// Mock account data - this will be injected via the global query client
const mockAccountData = [
  new AccountProvider({
    id: 1,
    name: 'Bank of America',
    description: 'Primary banking provider',
    accounts: [
      new Account({ id: 101, name: 'Checking Account', currency: 'USD', type: 'checking' }),
      new Account({ id: 102, name: 'Savings Account', currency: 'USD', type: 'savings' }),
      new Account({ id: 103, name: 'Investment Account', currency: 'USD', type: 'investment' }),
    ]
  }),
  new AccountProvider({
    id: 2,
    name: 'Chase',
    description: 'Secondary bank',
    accounts: [
      new Account({ id: 201, name: 'Business Checking', currency: 'USD', type: 'checking' }),
      new Account({ id: 202, name: 'Credit Card', currency: 'USD', type: 'credit' }),
    ]
  }),
  new AccountProvider({
    id: 3,
    name: 'PayPal',
    description: 'Digital wallet',
    accounts: [
      new Account({ id: 301, name: 'PayPal Balance', currency: 'USD', type: 'wallet' }),
    ]
  }),
  new AccountProvider({
    id: 4,
    name: 'European Bank',
    description: 'European accounts',
    accounts: [
      new Account({ id: 401, name: 'Euro Checking', currency: 'EUR', type: 'checking' }),
      new Account({ id: 402, name: 'Euro Savings', currency: 'EUR', type: 'savings' }),
    ]
  }),
]

export default {
  title: 'Components/AccountSelector',
  component: AccountSelector,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: 'A hierarchical account selector that groups accounts by provider. Uses TreeSelect from PrimeVue and fetches data via the useAccounts composable.',
      },
    },
  },
}

/**
 * Default account selector with no selection
 */
export const Default = {
  render: () => ({
    components: { AccountSelector },
    setup() {
      const selectedAccount = ref(null)
      
      // Set mock data in the query cache
      queryClient.setQueryData(['accounts'], mockAccountData)
      
      return { selectedAccount }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <AccountSelector
          v-model="selectedAccount"
          name="account"
          placeholder="Select Account"
        />
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Value:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedAccount }}</pre>
        </div>
      </div>
    `,
  }),
}

/**
 * Account selector with a pre-selected account
 */
export const WithPreselectedAccount = {
  render: () => ({
    components: { AccountSelector },
    setup() {
      // Pre-select "Checking Account" from Bank of America (ID: 101)
      const selectedAccount = ref({ 101: true })
      queryClient.setQueryData(['accounts'], mockAccountData)
      return { selectedAccount }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <AccountSelector
          v-model="selectedAccount"
          name="account"
          placeholder="Select Account"
        />
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Value:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedAccount }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Shows the account selector with Bank of America\'s Checking Account pre-selected.',
      },
    },
  },
}

/**
 * Filter by checking accounts only
 */
export const FilterCheckingAccounts = {
  render: () => ({
    components: { AccountSelector },
    setup() {
      const selectedAccount = ref(null)
      queryClient.setQueryData(['accounts'], mockAccountData)
      return { selectedAccount }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <AccountSelector
          v-model="selectedAccount"
          name="account"
          placeholder="Select Checking Account"
          :accountTypes="['checking']"
        />
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Value:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedAccount }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Only shows checking accounts from all providers. Note how PayPal provider is hidden since it has no checking accounts.',
      },
    },
  },
}

/**
 * Disabled account selector
 */
export const Disabled = {
  render: () => ({
    components: { AccountSelector },
    setup() {
      const selectedAccount = ref({ 102: true })
      queryClient.setQueryData(['accounts'], mockAccountData)
      return { selectedAccount }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <AccountSelector
          v-model="selectedAccount"
          name="account"
          placeholder="Select Account"
          :disabled="true"
        />
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Value:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedAccount }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Account selector in disabled state with a pre-selected value.',
      },
    },
  },
}

/**
 * Interactive form example
 */
export const InFormContext = {
  render: () => ({
    components: { AccountSelector },
    setup() {
      const fromAccount = ref(null)
      const toAccount = ref(null)
      const amount = ref(100)
      
      queryClient.setQueryData(['accounts'], mockAccountData)
      
      const handleSubmit = () => {
        alert(`Transfer $${amount.value} from account ${JSON.stringify(fromAccount.value)} to ${JSON.stringify(toAccount.value)}`)
      }
      
      return { fromAccount, toAccount, amount, handleSubmit }
    },
    template: `
      <div style="max-width: 600px; margin: 2rem auto;">
        <h3 style="margin-bottom: 1.5rem;">Transfer Funds</h3>
        <form @submit.prevent="handleSubmit" style="display: flex; flex-direction: column; gap: 1.5rem;">
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">From Account:</label>
            <AccountSelector
              v-model="fromAccount"
              name="fromAccount"
              placeholder="Select source account"
              :required="true"
            />
          </div>
          
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">To Account:</label>
            <AccountSelector
              v-model="toAccount"
              name="toAccount"
              placeholder="Select destination account"
              :required="true"
            />
          </div>
          
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Amount:</label>
            <input 
              v-model.number="amount" 
              type="number" 
              step="0.01"
              style="width: 100%; padding: 0.75rem; border: 1px solid #ccc; border-radius: 4px;"
            />
          </div>
          
          <button 
            type="submit"
            :disabled="!fromAccount || !toAccount"
            style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem;"
            :style="{ opacity: (!fromAccount || !toAccount) ? 0.5 : 1, cursor: (!fromAccount || !toAccount) ? 'not-allowed' : 'pointer' }"
          >
            Transfer Funds
          </button>
        </form>
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Form State:</strong>
          <pre style="margin: 0.5rem 0 0 0; font-size: 0.875rem;">From: {{ fromAccount }}
To: {{ toAccount }}
Amount: ${{ amount }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example of AccountSelector used in a transfer form with validation.',
      },
    },
  },
}
