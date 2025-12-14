import CustomDrawer from './CustomDrawer.vue'
import { ref } from 'vue'

export default {
  title: 'Components/Common/CustomDrawer',
  component: CustomDrawer,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'A custom drawer/sidebar component that slides in from left or right. Includes backdrop and slide animations.',
      },
    },
  },
}

/**
 * Drawer from the left side
 */
export const LeftDrawer = {
  render: () => ({
    components: { CustomDrawer },
    setup() {
      const visible = ref(false)
      
      const openDrawer = () => {
        visible.value = true
      }
      
      return { visible, openDrawer }
    },
    template: `
      <div>
        <button 
          @click="openDrawer"
          style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 2rem;"
        >
          Open Left Drawer
        </button>
        
        <CustomDrawer
          v-model:visible="visible"
          position="left"
          header="Navigation"
        >
          <div style="padding: 1rem;">
            <h3>Menu Items</h3>
            <ul style="list-style: none; padding: 0;">
              <li style="padding: 0.5rem 0; cursor: pointer;">Dashboard</li>
              <li style="padding: 0.5rem 0; cursor: pointer;">Transactions</li>
              <li style="padding: 0.5rem 0; cursor: pointer;">Reports</li>
              <li style="padding: 0.5rem 0; cursor: pointer;">Settings</li>
            </ul>
          </div>
        </CustomDrawer>
      </div>
    `,
  }),
}

/**
 * Drawer from the right side
 */
export const RightDrawer = {
  render: () => ({
    components: { CustomDrawer },
    setup() {
      const visible = ref(false)
      
      const openDrawer = () => {
        visible.value = true
      }
      
      return { visible, openDrawer }
    },
    template: `
      <div>
        <button 
          @click="openDrawer"
          style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 2rem;"
        >
          Open Right Drawer
        </button>
        
        <CustomDrawer
          v-model:visible="visible"
          position="right"
          header="Settings"
        >
          <div style="padding: 1rem;">
            <h3>User Settings</h3>
            <div style="display: flex; flex-direction: column; gap: 1rem; margin-top: 1rem;">
              <label>
                <input type="checkbox" checked /> Email notifications
              </label>
              <label>
                <input type="checkbox" /> SMS notifications
              </label>
              <label>
                <input type="checkbox" checked /> Dark mode
              </label>
            </div>
          </div>
        </CustomDrawer>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Drawer sliding in from the right side, useful for settings or additional options.',
      },
    },
  },
}

/**
 * Drawer without header
 */
export const WithoutHeader = {
  render: () => ({
    components: { CustomDrawer },
    setup() {
      const visible = ref(false)
      
      const openDrawer = () => {
        visible.value = true
      }
      
      return { visible, openDrawer }
    },
    template: `
      <div>
        <button 
          @click="openDrawer"
          style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 2rem;"
        >
          Open Drawer (No Header)
        </button>
        
        <CustomDrawer
          v-model:visible="visible"
          position="left"
        >
          <div style="padding: 2rem;">
            <h2>Custom Content</h2>
            <p>This drawer has no header, giving you full control over the content.</p>
            <button 
              @click="visible = false"
              style="margin-top: 1rem; padding: 0.5rem 1rem; background: #dc2626; color: white; border: none; border-radius: 4px; cursor: pointer;"
            >
              Close
            </button>
          </div>
        </CustomDrawer>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Drawer without a header for custom layouts.',
      },
    },
  },
}

/**
 * Drawer with rich content
 */
export const WithRichContent = {
  render: () => ({
    components: { CustomDrawer },
    setup() {
      const visible = ref(false)
      
      const openDrawer = () => {
        visible.value = true
      }
      
      return { visible, openDrawer }
    },
    template: `
      <div>
        <button 
          @click="openDrawer"
          style="padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 2rem;"
        >
          Open Filter Panel
        </button>
        
        <CustomDrawer
          v-model:visible="visible"
          position="right"
          header="Filters"
        >
          <div style="padding: 1rem; display: flex; flex-direction: column; gap: 1.5rem;">
            <div>
              <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Date Range</label>
              <select style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;">
                <option>Last 7 days</option>
                <option>Last 30 days</option>
                <option>Last 90 days</option>
                <option>Custom</option>
              </select>
            </div>
            
            <div>
              <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Category</label>
              <select style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;">
                <option>All Categories</option>
                <option>Income</option>
                <option>Expenses</option>
                <option>Transfer</option>
              </select>
            </div>
            
            <div>
              <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Amount Range</label>
              <div style="display: flex; gap: 0.5rem;">
                <input type="number" placeholder="Min" style="flex: 1; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;" />
                <input type="number" placeholder="Max" style="flex: 1; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;" />
              </div>
            </div>
            
            <div style="display: flex; gap: 0.5rem; margin-top: auto; padding-top: 1rem; border-top: 1px solid #e5e5e5;">
              <button style="flex: 1; padding: 0.75rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer;">
                Apply
              </button>
              <button 
                @click="visible = false"
                style="flex: 1; padding: 0.75rem; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;"
              >
                Cancel
              </button>
            </div>
          </div>
        </CustomDrawer>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example of a drawer with a filter form containing multiple inputs.',
      },
    },
  },
}

/**
 * Open by default for preview
 */
export const OpenByDefault = {
  render: () => ({
    components: { CustomDrawer },
    setup() {
      const visible = ref(true)
      
      return { visible }
    },
    template: `
      <div>
        <p style="margin: 2rem;">Drawer is open by default for preview</p>
        
        <CustomDrawer
          v-model:visible="visible"
          position="left"
          header="Example Drawer"
        >
          <div style="padding: 1rem;">
            <p>This is the drawer content.</p>
            <p>Click outside or press the X to close.</p>
          </div>
        </CustomDrawer>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Drawer opened by default to preview its appearance.',
      },
    },
  },
}

