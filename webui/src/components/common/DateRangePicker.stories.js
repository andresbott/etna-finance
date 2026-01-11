import DateRangePicker from './DateRangePicker.vue'
import { ref } from 'vue'

export default {
  title: 'Components/Common/DateRangePicker',
  component: DateRangePicker,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: 'A date range picker with start and end dates, plus quick select options for common ranges (current month, year, etc.).',
      },
    },
  },
}

/**
 * Default date range picker
 */
export const Default = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date(2024, 0, 1)) // January 1, 2024
      const endDate = ref(new Date(2024, 11, 31)) // December 31, 2024
      
      const handleChange = (range) => {
        console.log('Date range changed:', range)
      }
      
      return { startDate, endDate, handleChange }
    },
    template: `
      <div>
        <DateRangePicker
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          @change="handleChange"
        />
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Range:</strong>
          <pre style="margin: 0.5rem 0 0 0;">Start: {{ startDate?.toLocaleDateString() }}
End: {{ endDate?.toLocaleDateString() }}</pre>
        </div>
      </div>
    `,
  }),
}

/**
 * Custom labels
 */
export const CustomLabels = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date())
      const endDate = ref(new Date())
      
      return { startDate, endDate }
    },
    template: `
      <div>
        <DateRangePicker
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          startLabel="Start:"
          endLabel="End:"
          startPlaceholder="Pick start date"
          endPlaceholder="Pick end date"
        />
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Range:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ startDate?.toLocaleDateString() }} - {{ endDate?.toLocaleDateString() }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Date range picker with custom labels and placeholders.',
      },
    },
  },
}

/**
 * Without icon
 */
export const WithoutIcon = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date(2024, 0, 1))
      const endDate = ref(new Date(2024, 2, 31))
      
      return { startDate, endDate }
    },
    template: `
      <div>
        <DateRangePicker
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          :showIcon="false"
        />
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Range:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ startDate?.toLocaleDateString() }} - {{ endDate?.toLocaleDateString() }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Date picker without calendar icons.',
      },
    },
  },
}

/**
 * Without button bar
 */
export const WithoutButtonBar = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date())
      const endDate = ref(new Date())
      
      return { startDate, endDate }
    },
    template: `
      <div>
        <DateRangePicker
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          :showButtonBar="false"
        />
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Range:</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ startDate?.toLocaleDateString() }} - {{ endDate?.toLocaleDateString() }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Date picker without the "Today" and "Clear" button bar.',
      },
    },
  },
}

/**
 * Different date format
 */
export const DifferentFormat = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date(2024, 5, 1)) // June 1, 2024
      const endDate = ref(new Date(2024, 5, 30)) // June 30, 2024
      
      return { startDate, endDate }
    },
    template: `
      <div>
        <DateRangePicker
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          dateFormat="mm/dd/yy"
        />
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Range (MM/DD/YY format):</strong>
          <pre style="margin: 0.5rem 0 0 0;">{{ startDate?.toLocaleDateString() }} - {{ endDate?.toLocaleDateString() }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Date picker with MM/DD/YY format instead of the default DD/MM/Y.',
      },
    },
  },
}

/**
 * Quick select demonstration
 */
export const QuickSelectDemo = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date())
      const endDate = ref(new Date())
      
      return { startDate, endDate }
    },
    template: `
      <div>
        <h3 style="margin-bottom: 1rem;">Try the Quick Select dropdown</h3>
        <p style="margin-bottom: 1rem; color: #666;">
          Use the "Quick Select" dropdown to quickly set common date ranges:
          Previous Month, Current Month, Previous Year, or Current Year.
        </p>
        
        <DateRangePicker
          v-model:startDate="startDate"
          v-model:endDate="endDate"
        />
        
        <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
          <strong>Selected Range:</strong>
          <pre style="margin: 0.5rem 0 0 0;">Start: {{ startDate?.toLocaleDateString() }}
End: {{ endDate?.toLocaleDateString() }}</pre>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Demonstrates the quick select feature for common date ranges.',
      },
    },
  },
}

/**
 * In a form context
 */
export const InFormContext = {
  render: () => ({
    components: { DateRangePicker },
    setup() {
      const startDate = ref(new Date(2024, 0, 1))
      const endDate = ref(new Date(2024, 11, 31))
      const reportType = ref('income-expense')
      
      const generateReport = () => {
        alert(`Generating ${reportType.value} report for ${startDate.value.toLocaleDateString()} to ${endDate.value.toLocaleDateString()}`)
      }
      
      return { startDate, endDate, reportType, generateReport }
    },
    template: `
      <div style="max-width: 800px; margin: 0 auto;">
        <h3 style="margin-bottom: 1.5rem;">Generate Financial Report</h3>
        
        <form @submit.prevent="generateReport" style="display: flex; flex-direction: column; gap: 1.5rem;">
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Report Type:</label>
            <select v-model="reportType" style="width: 100%; padding: 0.75rem; border: 1px solid #ccc; border-radius: 4px;">
              <option value="income-expense">Income & Expense</option>
              <option value="balance">Balance Sheet</option>
              <option value="cash-flow">Cash Flow</option>
              <option value="budget">Budget Analysis</option>
            </select>
          </div>
          
          <div>
            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Date Range:</label>
            <DateRangePicker
              v-model:startDate="startDate"
              v-model:endDate="endDate"
            />
          </div>
          
          <button 
            type="submit"
            style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem;"
          >
            Generate Report
          </button>
        </form>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example of DateRangePicker used in a report generation form.',
      },
    },
  },
}

