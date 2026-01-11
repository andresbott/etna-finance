import CategorySelect from './categorySelect.vue'
import { ref } from 'vue'
import { queryClient } from '../../../.storybook/preview.js'

// Mock category tree data
const mockExpenseCategories = [
  {
    data: { id: 1, name: 'Housing', type: 'expense' },
    children: [
      { data: { id: 11, name: 'Rent', type: 'expense' }, children: [] },
      { data: { id: 12, name: 'Utilities', type: 'expense' }, children: [] },
      { data: { id: 13, name: 'Maintenance', type: 'expense' }, children: [] },
    ]
  },
  {
    data: { id: 2, name: 'Transportation', type: 'expense' },
    children: [
      { data: { id: 21, name: 'Gas', type: 'expense' }, children: [] },
      { data: { id: 22, name: 'Public Transit', type: 'expense' }, children: [] },
      { data: { id: 23, name: 'Car Payment', type: 'expense' }, children: [] },
    ]
  },
  {
    data: { id: 3, name: 'Food', type: 'expense' },
    children: [
      { data: { id: 31, name: 'Groceries', type: 'expense' }, children: [] },
      { data: { id: 32, name: 'Restaurants', type: 'expense' }, children: [] },
    ]
  },
  {
    data: { id: 4, name: 'Entertainment', type: 'expense' },
    children: []
  },
]

const mockIncomeCategories = [
  {
    data: { id: 101, name: 'Salary', type: 'income' },
    children: [
      { data: { id: 111, name: 'Primary Job', type: 'income' }, children: [] },
      { data: { id: 112, name: 'Secondary Job', type: 'income' }, children: [] },
    ]
  },
  {
    data: { id: 102, name: 'Investments', type: 'income' },
    children: [
      { data: { id: 121, name: 'Dividends', type: 'income' }, children: [] },
      { data: { id: 122, name: 'Interest', type: 'income' }, children: [] },
    ]
  },
  {
    data: { id: 103, name: 'Freelance', type: 'income' },
    children: []
  },
]

export default {
  title: 'Components/Common/CategorySelect',
  component: CategorySelect,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: 'A hierarchical category selector using TreeSelect from PrimeVue. Displays categories in a tree structure for expense or income types.',
      },
    },
  },
}

/**
 * Expense category selector
 */
export const ExpenseCategories = {
  render: () => ({
    components: { CategorySelect },
    setup() {
      const selectedCategory = ref(0)
      
      // Set mock expense category data
      queryClient.setQueryData(['categories', 'expense'], mockExpenseCategories)
      
      return { selectedCategory }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <CategorySelect
          v-model="selectedCategory"
          type="expense"
        />
        
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Category ID:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedCategory }}</pre>
        </div>
      </div>
    `,
  }),
}

/**
 * Income category selector
 */
export const IncomeCategories = {
  render: () => ({
    components: { CategorySelect },
    setup() {
      const selectedCategory = ref(0)
      
      // Set mock income category data
      queryClient.setQueryData(['categories', 'income'], mockIncomeCategories)
      
      return { selectedCategory }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <CategorySelect
          v-model="selectedCategory"
          type="income"
        />
        
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Category ID:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedCategory }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Category selector configured for income categories.',
      },
    },
  },
}

/**
 * With pre-selected category
 */
export const WithPreselection = {
  render: () => ({
    components: { CategorySelect },
    setup() {
      // Pre-select "Groceries" (ID: 31)
      const selectedCategory = ref(31)
      
      queryClient.setQueryData(['categories', 'expense'], mockExpenseCategories)
      
      return { selectedCategory }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <CategorySelect
          v-model="selectedCategory"
          type="expense"
        />
        
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Category ID:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedCategory }}</pre>
          <p style="margin-top: 0.5rem; font-size: 0.875rem; color: #666;">
            Pre-selected: Food / Groceries (ID: 31)
          </p>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Category selector with a pre-selected nested category.',
      },
    },
  },
}

/**
 * Root category selected
 */
export const RootSelected = {
  render: () => ({
    components: { CategorySelect },
    setup() {
      const selectedCategory = ref(0)
      
      queryClient.setQueryData(['categories', 'expense'], mockExpenseCategories)
      
      return { selectedCategory }
    },
    template: `
      <div style="max-width: 500px; margin: 2rem auto;">
        <h4 style="margin-bottom: 1rem;">Root Category (Uncategorized)</h4>
        <CategorySelect
          v-model="selectedCategory"
          type="expense"
        />
        
        <div style="margin-top: 1rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Category ID:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ selectedCategory }}</pre>
          <p style="margin-top: 0.5rem; font-size: 0.875rem; color: #666;">
            ID 0 represents "Root Category" - uncategorized transactions
          </p>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Shows the root category selection (ID: 0) for uncategorized items.',
      },
    },
  },
}

/**
 * In a transaction form
 */
export const InTransactionForm = {
  render: () => ({
    components: { CategorySelect },
    setup() {
      const amount = ref(50.00)
      const description = ref('Weekly groceries')
      const selectedCategory = ref(31) // Groceries
      const transactionType = ref('expense')
      
      // Set both types of categories
      queryClient.setQueryData(['categories', 'expense'], mockExpenseCategories)
      queryClient.setQueryData(['categories', 'income'], mockIncomeCategories)
      
      const handleSubmit = () => {
        alert(`Transaction: $${amount.value} - ${description.value} (Category: ${selectedCategory.value})`)
      }
      
      return { amount, description, selectedCategory, transactionType, handleSubmit }
    },
    template: `
      <div style="max-width: 600px; margin: 2rem auto;">
        <h3 style="margin-bottom: 1.5rem;">Add Transaction</h3>
        
        <form @submit.prevent="handleSubmit" style="display: flex; flex-direction: column; gap: 1.5rem;">
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Type:</label>
            <select v-model="transactionType" style="width: 100%; padding: 0.75rem; border: 1px solid #ccc; border-radius: 4px;">
              <option value="expense">Expense</option>
              <option value="income">Income</option>
            </select>
          </div>
          
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Description:</label>
            <input 
              v-model="description"
              type="text"
              style="width: 100%; padding: 0.75rem; border: 1px solid #ccc; border-radius: 4px;"
            />
          </div>
          
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Amount:</label>
            <input 
              v-model.number="amount"
              type="number"
              step="0.01"
              style="width: 100%; padding: 0.75rem; border: 1px solid #ccc; border-radius: 4px;"
            />
          </div>
          
          <CategorySelect
            v-model="selectedCategory"
            :type="transactionType"
          />
          
          <button 
            type="submit"
            style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem;"
          >
            Add Transaction
          </button>
        </form>
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Form State:</strong>
          <pre style="margin: 0.5rem 0 0 0; font-size: 0.875rem;">Type: {{ transactionType }}
Amount: ${{ amount }}
Category: {{ selectedCategory }}
Description: {{ description }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example of CategorySelect used in a transaction entry form with dynamic category type.',
      },
    },
  },
}

