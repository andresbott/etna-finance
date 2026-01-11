import SecondaryMenu from './SecondaryMenu.vue'
import { useUiStore } from '@/store/uiStore.js'
import { useUserStore } from '@/lib/user/userstore.js'

export default {
  title: 'Components/SecondaryMenu',
  component: SecondaryMenu,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'A right-side drawer menu for user settings, configuration, and logout. Uses Pinia stores for state management.',
      },
    },
  },
}

/**
 * Closed by default
 */
export const Default = {
  render: () => ({
    components: { SecondaryMenu },
    setup() {
      const uiStore = useUiStore()
      const userStore = useUserStore()
      
      // Set a mock username
      userStore.loggedInUser = 'john.doe'
      
      const openMenu = () => {
        uiStore.openSecondaryDrawer()
      }
      
      return { openMenu }
    },
    template: `
      <div>
        <button 
          @click="openMenu"
          style="position: fixed; top: 1rem; right: 1rem; padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; z-index: 1000;"
        >
          Open Settings Menu
        </button>
        
        <SecondaryMenu />
      </div>
    `,
  }),
}

/**
 * Open by default
 */
export const OpenByDefault = {
  render: () => ({
    components: { SecondaryMenu },
    setup() {
      const uiStore = useUiStore()
      const userStore = useUserStore()
      
      userStore.loggedInUser = 'jane.smith'
      
      // Open the drawer on mount
      uiStore.openSecondaryDrawer()
      
      return {}
    },
    template: `
      <div>
        <p style="margin: 2rem;">Secondary menu is open for preview</p>
        <SecondaryMenu />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Secondary menu opened by default to show all menu options.',
      },
    },
  },
}

/**
 * Different username
 */
export const DifferentUser = {
  render: () => ({
    components: { SecondaryMenu },
    setup() {
      const uiStore = useUiStore()
      const userStore = useUserStore()
      
      userStore.loggedInUser = 'admin@company.com'
      uiStore.openSecondaryDrawer()
      
      return {}
    },
    template: `
      <div>
        <p style="margin: 2rem;">Logged in as: admin@company.com</p>
        <SecondaryMenu />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Shows the menu with a different username displayed.',
      },
    },
  },
}

/**
 * Navigation example
 */
export const NavigationExample = {
  render: () => ({
    components: { SecondaryMenu },
    setup() {
      const uiStore = useUiStore()
      const userStore = useUserStore()
      
      userStore.loggedInUser = 'demo.user'
      
      const openMenu = () => {
        uiStore.openSecondaryDrawer()
      }
      
      return { openMenu }
    },
    template: `
      <div>
        <div style="padding: 2rem;">
          <h2>Finance Dashboard</h2>
          <p>Click the button to open the settings menu</p>
          <p style="margin-top: 1rem; padding: 1rem; background: #eff6ff; border-radius: 4px;">
            <strong>Available sections:</strong><br/>
            • Settings - Configuration options<br/>
            • Application Data - Categories, Accounts, CSV Profiles<br/>
            • Maintenance - Backup/Restore<br/>
            • Logout
          </p>
        </div>
        
        <button 
          @click="openMenu"
          style="position: fixed; top: 1rem; right: 1rem; padding: 0.75rem; background: #335c67; color: white; border: none; border-radius: 50%; cursor: pointer; z-index: 1000; width: 48px; height: 48px;"
          title="Settings"
        >
          <i class="pi pi-cog"></i>
        </button>
        
        <SecondaryMenu />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Example showing how the secondary menu is typically accessed via a top-right button.',
      },
    },
  },
}

/**
 * With context explanation
 */
export const WithExplanation = {
  render: () => ({
    components: { SecondaryMenu },
    setup() {
      const uiStore = useUiStore()
      const userStore = useUserStore()
      
      userStore.loggedInUser = 'test.user'
      uiStore.openSecondaryDrawer()
      
      return {}
    },
    template: `
      <div>
        <div style="max-width: 800px; margin: 2rem; padding: 2rem; background: white; border: 1px solid #e5e5e5; border-radius: 8px;">
          <h2>Secondary Menu Features</h2>
          
          <h3 style="margin-top: 1.5rem;">Settings Section</h3>
          <p>Access application configuration and preferences.</p>
          
          <h3 style="margin-top: 1.5rem;">Application Data</h3>
          <ul>
            <li><strong>CSV Import Profiles:</strong> Manage CSV import templates</li>
            <li><strong>Categories:</strong> Organize transaction categories</li>
            <li><strong>Account Setup:</strong> Configure financial accounts</li>
          </ul>
          
          <h3 style="margin-top: 1.5rem;">Maintenance</h3>
          <p>Backup and restore your financial data.</p>
          
          <h3 style="margin-top: 1.5rem;">Logout</h3>
          <p>Securely sign out of the application.</p>
        </div>
        
        <SecondaryMenu />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Shows the menu with contextual explanation of each section.',
      },
    },
  },
}

