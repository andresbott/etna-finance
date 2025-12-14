import ConfirmDialog from './confirmDialog.vue'
import { ref } from 'vue'

export default {
  title: 'Components/Common/ConfirmDialog',
  component: ConfirmDialog,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: 'A reusable confirmation dialog component for delete operations or other confirmations. Uses PrimeVue Dialog.',
      },
    },
  },
}

/**
 * Default confirmation dialog
 */
export const Default = {
  render: () => ({
    components: { ConfirmDialog },
    setup() {
      const visible = ref(false)
      const itemName = ref('My Important Item')
      
      const handleConfirm = () => {
        return new Promise((resolve) => {
          setTimeout(() => {
            alert('Item deleted!')
            visible.value = false
            resolve()
          }, 500)
        })
      }
      
      const openDialog = () => {
        visible.value = true
      }
      
      return { visible, itemName, handleConfirm, openDialog }
    },
    template: `
      <div>
        <button 
          @click="openDialog"
          style="padding: 0.75rem 1.5rem; background: #dc2626; color: white; border: none; border-radius: 4px; cursor: pointer;"
        >
          Delete Item
        </button>
        
        <ConfirmDialog
          v-model:visible="visible"
          :name="itemName"
          :onConfirm="handleConfirm"
        />
      </div>
    `,
  }),
}

/**
 * Custom title and message
 */
export const CustomContent = {
  render: () => ({
    components: { ConfirmDialog },
    setup() {
      const visible = ref(false)
      
      const handleConfirm = () => {
        return new Promise((resolve) => {
          setTimeout(() => {
            alert('Action confirmed!')
            visible.value = false
            resolve()
          }, 500)
        })
      }
      
      const openDialog = () => {
        visible.value = true
      }
      
      return { visible, handleConfirm, openDialog }
    },
    template: `
      <div>
        <button 
          @click="openDialog"
          style="padding: 0.75rem 1.5rem; background: #ea580c; color: white; border: none; border-radius: 4px; cursor: pointer;"
        >
          Archive Account
        </button>
        
        <ConfirmDialog
          v-model:visible="visible"
          title="Archive Account"
          message="Are you sure you want to archive this account? This action can be undone later."
          name="Savings Account"
          :onConfirm="handleConfirm"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Dialog with custom title and message for archive operation.',
      },
    },
  },
}

/**
 * Without item name
 */
export const WithoutName = {
  render: () => ({
    components: { ConfirmDialog },
    setup() {
      const visible = ref(false)
      
      const handleConfirm = () => {
        return new Promise((resolve) => {
          setTimeout(() => {
            alert('Confirmed!')
            visible.value = false
            resolve()
          }, 500)
        })
      }
      
      const openDialog = () => {
        visible.value = true
      }
      
      return { visible, handleConfirm, openDialog }
    },
    template: `
      <div>
        <button 
          @click="openDialog"
          style="padding: 0.75rem 1.5rem; background: #7c3aed; color: white; border: none; border-radius: 4px; cursor: pointer;"
        >
          Clear All Data
        </button>
        
        <ConfirmDialog
          v-model:visible="visible"
          title="Clear All Data"
          message="Are you sure you want to clear all data? This action cannot be undone."
          :onConfirm="handleConfirm"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Dialog without an item name, useful for general confirmations.',
      },
    },
  },
}

/**
 * Open by default (for testing)
 */
export const OpenByDefault = {
  render: () => ({
    components: { ConfirmDialog },
    setup() {
      const visible = ref(true)
      
      const handleConfirm = () => {
        return new Promise((resolve) => {
          setTimeout(() => {
            alert('Confirmed!')
            visible.value = false
            resolve()
          }, 500)
        })
      }
      
      return { visible, handleConfirm }
    },
    template: `
      <div>
        <p>Dialog is open by default for preview</p>
        <ConfirmDialog
          v-model:visible="visible"
          name="Test Category"
          title="Delete Category"
          message="Are you sure you want to delete"
          :onConfirm="handleConfirm"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Dialog opened by default to show the dialog appearance.',
      },
    },
  },
}

/**
 * Error handling example
 */
export const WithError = {
  render: () => ({
    components: { ConfirmDialog },
    setup() {
      const visible = ref(false)
      const errorMessage = ref('')
      
      const handleConfirm = () => {
        return new Promise((resolve, reject) => {
          setTimeout(() => {
            errorMessage.value = 'Failed to delete item. Please try again.'
            reject(new Error('Delete failed'))
            // Dialog stays open on error
          }, 500)
        })
      }
      
      const openDialog = () => {
        visible.value = true
        errorMessage.value = ''
      }
      
      return { visible, errorMessage, handleConfirm, openDialog }
    },
    template: `
      <div>
        <button 
          @click="openDialog"
          style="padding: 0.75rem 1.5rem; background: #dc2626; color: white; border: none; border-radius: 4px; cursor: pointer;"
        >
          Delete (Will Fail)
        </button>
        
        <div v-if="errorMessage" style="margin-top: 1rem; padding: 1rem; background: #fee2e2; color: #991b1b; border-radius: 4px;">
          {{ errorMessage }}
        </div>
        
        <ConfirmDialog
          v-model:visible="visible"
          name="Protected Item"
          :onConfirm="handleConfirm"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example showing error handling - dialog stays open when confirmation fails.',
      },
    },
  },
}

