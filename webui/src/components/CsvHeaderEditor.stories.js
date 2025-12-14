import CsvHeaderEditor from './CsvHeaderEditor.vue'
import { ref } from 'vue'

export default {
  title: 'Components/CsvHeaderEditor',
  component: CsvHeaderEditor,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: 'A comprehensive CSV header mapping editor for configuring how CSV columns map to transaction fields. Includes validation, date format selection, and dynamic header management.',
      },
    },
  },
}

/**
 * Empty editor
 */
export const Empty = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([])
      
      const handleSave = (data) => {
        console.log('Saved:', data)
        alert(`Saved mapping with ${data.headers.length} columns and date format: ${data.dateFormat}`)
      }
      
      return { headers, handleSave }
    },
    template: `
      <div>
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
      </div>
    `,
  }),
}

/**
 * With sample headers
 */
export const WithSampleHeaders = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([
        { id: 1, name: 'Transaction Date', mappedTo: null, example: '2024-01-15' },
        { id: 2, name: 'Description', mappedTo: null, example: 'Grocery Store' },
        { id: 3, name: 'Amount', mappedTo: null, example: '-45.50' },
        { id: 4, name: 'Balance', mappedTo: null, example: '1234.56' },
      ])
      
      const handleSave = (data) => {
        console.log('Saved:', data)
        alert(`Saved mapping:\n${JSON.stringify(data, null, 2)}`)
      }
      
      return { headers, handleSave }
    },
    template: `
      <div>
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'CSV header editor with sample unmapped headers ready to be configured.',
      },
    },
  },
}

/**
 * Fully configured mapping
 */
export const FullyConfigured = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([
        { id: 1, name: 'Date', mappedTo: 'date', example: '01/15/2024' },
        { id: 2, name: 'Description', mappedTo: 'description', example: 'Costco Wholesale' },
        { id: 3, name: 'Debit', mappedTo: 'amount', example: '125.50' },
        { id: 4, name: 'Memo', mappedTo: 'notes', example: 'Monthly shopping' },
        { id: 5, name: 'Category', mappedTo: 'category', example: 'Groceries' },
      ])
      
      const handleSave = (data) => {
        console.log('Saved:', data)
        alert('Configuration saved successfully!')
      }
      
      return { headers, handleSave }
    },
    template: `
      <div>
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'A fully configured CSV mapping with all required fields mapped.',
      },
    },
  },
}

/**
 * With validation errors
 */
export const WithValidationErrors = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([
        { id: 1, name: 'Transaction Date', mappedTo: 'date', example: '2024-01-15' },
        { id: 2, name: 'Merchant', mappedTo: null, example: 'Amazon' },
        { id: 3, name: 'Total', mappedTo: null, example: '99.99' },
      ])
      
      const handleSave = (data) => {
        console.log('Save attempted:', data)
        alert('Cannot save - please fix validation errors first!')
      }
      
      return { headers, handleSave }
    },
    template: `
      <div>
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
        <div style="margin-top: 1rem; padding: 1rem; background: #fef2f2; border: 1px solid #fecaca; border-radius: 4px; color: #991b1b;">
          <strong>⚠️ Validation Issues:</strong>
          <ul style="margin: 0.5rem 0 0 1.5rem;">
            <li>Amount field is required but not mapped</li>
            <li>Description field is required but not mapped</li>
          </ul>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Shows validation errors when required fields are missing. The Save button will be disabled.',
      },
    },
  },
}

/**
 * Bank statement example (Chase)
 */
export const BankStatementChase = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([
        { id: 1, name: 'Details', mappedTo: 'notes', example: 'DEBIT' },
        { id: 2, name: 'Posting Date', mappedTo: 'date', example: '01/15/2024' },
        { id: 3, name: 'Description', mappedTo: 'description', example: 'WHOLEFDS MKT' },
        { id: 4, name: 'Amount', mappedTo: 'amount', example: '-52.34' },
        { id: 5, name: 'Type', mappedTo: 'type', example: 'DEBIT' },
        { id: 6, name: 'Balance', mappedTo: null, example: '2145.67' },
        { id: 7, name: 'Check or Slip #', mappedTo: 'reference', example: '' },
      ])
      
      const handleSave = (data) => {
        alert('Chase Bank CSV profile saved!')
      }
      
      return { headers, handleSave }
    },
    template: `
      <div>
        <h3 style="margin-bottom: 1rem;">Chase Bank Statement Import</h3>
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example configuration for importing Chase bank statements.',
      },
    },
  },
}

/**
 * Credit card statement example
 */
export const CreditCardStatement = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([
        { id: 1, name: 'Transaction Date', mappedTo: 'date', example: '2024/01/15' },
        { id: 2, name: 'Post Date', mappedTo: null, example: '2024/01/16' },
        { id: 3, name: 'Description', mappedTo: 'description', example: 'NETFLIX.COM' },
        { id: 4, name: 'Category', mappedTo: 'category', example: 'Entertainment' },
        { id: 5, name: 'Type', mappedTo: 'type', example: 'Sale' },
        { id: 6, name: 'Amount', mappedTo: 'amount', example: '15.99' },
      ])
      
      const handleSave = (data) => {
        alert('Credit card CSV profile saved!')
      }
      
      return { headers, handleSave }
    },
    template: `
      <div>
        <h3 style="margin-bottom: 1rem;">Credit Card Statement Import</h3>
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example configuration for importing credit card statements with categories.',
      },
    },
  },
}

/**
 * Interactive demo
 */
export const InteractiveDemo = {
  render: () => ({
    components: { CsvHeaderEditor },
    setup() {
      const headers = ref([
        { id: 1, name: 'Column A', mappedTo: null, example: '2024-01-15' },
        { id: 2, name: 'Column B', mappedTo: null, example: 'Sample Transaction' },
        { id: 3, name: 'Column C', mappedTo: null, example: '100.00' },
      ])
      
      const saveMessage = ref('')
      
      const handleSave = (data) => {
        saveMessage.value = `✓ Saved successfully! Mapped ${data.headers.filter(h => h.mappedTo).length} columns with date format: ${data.dateFormat}`
        setTimeout(() => {
          saveMessage.value = ''
        }, 5000)
      }
      
      return { headers, handleSave, saveMessage }
    },
    template: `
      <div>
        <div style="margin-bottom: 1.5rem; padding: 1rem; background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 4px;">
          <strong>Instructions:</strong>
          <ol style="margin: 0.5rem 0 0 1.5rem; padding: 0;">
            <li>Edit column names to match your CSV file</li>
            <li>Select the appropriate field mapping for each column</li>
            <li>Add example data to help identify the correct mapping</li>
            <li>Choose your date format</li>
            <li>Click "Save Mapping" (requires Date, Description, and Amount mapped)</li>
          </ol>
        </div>
        
        <CsvHeaderEditor
          :headers="headers"
          @save="handleSave"
        />
        
        <div v-if="saveMessage" style="margin-top: 1rem; padding: 1rem; background: #d1fae5; border: 1px solid #a7f3d0; border-radius: 4px; color: #047857;">
          {{ saveMessage }}
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Interactive demo - try mapping the columns and saving the configuration.',
      },
    },
  },
}

