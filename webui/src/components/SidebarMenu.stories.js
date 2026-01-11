import SidebarMenu from './SidebarMenu.vue'
import { useUiStore } from '@/store/uiStore.js'
import { queryClient } from '../../.storybook/preview.js'
import AccountProvider from '@/models/AccountProvider'
import Account from '@/models/Account'

// Mock account data for the sidebar
const mockAccountData = [
  new AccountProvider({
    id: 1,
    name: 'Bank of America',
    description: 'Primary banking',
    accounts: [
      new Account({ id: 101, name: 'Checking', currency: 'USD', type: 'bank' }),
      new Account({ id: 102, name: 'Savings', currency: 'USD', type: 'savings' }),
    ]
  }),
  new AccountProvider({
    id: 2,
    name: 'Chase',
    description: 'Credit cards',
    accounts: [
      new Account({ id: 201, name: 'Chase Freedom', currency: 'USD', type: 'credit' }),
      new Account({ id: 202, name: 'Chase Sapphire', currency: 'USD', type: 'credit' }),
    ]
  }),
  new AccountProvider({
    id: 3,
    name: 'Vanguard',
    description: 'Investments',
    accounts: [
      new Account({ id: 301, name: '401k', currency: 'USD', type: 'investment' }),
      new Account({ id: 302, name: 'Roth IRA', currency: 'USD', type: 'investment' }),
      new Account({ id: 303, name: 'Brokerage', currency: 'USD', type: 'investment' }),
    ]
  }),
]

export default {
  title: 'Components/SidebarMenu',
  component: SidebarMenu,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'The main sidebar navigation menu with hierarchical account structure. Features expandable sections for Transactions, Reports, and Market Data.',
      },
    },
  },
}

/**
 * Default sidebar (closed on mobile)
 */
export const Default = {
  render: () => ({
    components: { SidebarMenu },
    setup() {
      const uiStore = useUiStore()
      
      // Set mock account data
      queryClient.setQueryData(['accounts'], mockAccountData)
      
      const openSidebar = () => {
        uiStore.openDrawer()
      }
      
      return { openSidebar, uiStore }
    },
    template: `
      <div>
        <button 
          v-if="!uiStore.isDrawerVisible"
          @click="openSidebar"
          style="position: fixed; top: 1rem; left: 1rem; padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer; z-index: 1000;"
        >
          <i class="pi pi-bars" style="margin-right: 0.5rem;"></i>
          Open Menu
        </button>
        
        <SidebarMenu />
        
        <div style="margin-left: 0; padding: 2rem; transition: margin-left 0.3s;">
          <h2>Main Content Area</h2>
          <p>Click "Open Menu" to show the sidebar navigation.</p>
        </div>
      </div>
    `,
  }),
}

/**
 * Open by default
 */
export const OpenByDefault = {
  render: () => ({
    components: { SidebarMenu },
    setup() {
      const uiStore = useUiStore()
      
      queryClient.setQueryData(['accounts'], mockAccountData)
      
      // Open the drawer
      uiStore.openDrawer()
      
      return {}
    },
    template: `
      <div style="display: flex;">
        <SidebarMenu />
        
        <div style="flex: 1; padding: 2rem; margin-left: 300px;">
          <h2>Dashboard</h2>
          <p>Sidebar is open by default showing all navigation options.</p>
          
          <div style="margin-top: 2rem; padding: 1rem; background: #f5f5f5; border-radius: 4px;">
            <h3>Navigation Sections:</h3>
            <ul>
              <li><strong>Transactions:</strong> View all transactions or filter by account</li>
              <li><strong>Reports:</strong> Overview and Income/Expense analysis</li>
              <li><strong>Market Data:</strong> Currency exchange and stock market data</li>
            </ul>
          </div>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Sidebar open by default to show the full navigation structure.',
      },
    },
  },
}

/**
 * With account section expanded
 */
export const AccountsExpanded = {
  render: () => ({
    components: { SidebarMenu },
    setup() {
      const uiStore = useUiStore()
      
      queryClient.setQueryData(['accounts'], mockAccountData)
      uiStore.openDrawer()
      
      // Simulate expanding accounts after a short delay
      setTimeout(() => {
        const accountToggle = document.querySelector('.menu-item .menu-toggle')
        if (accountToggle) {
          accountToggle.closest('.menu-item').click()
        }
      }, 500)
      
      return {}
    },
    template: `
      <div style="display: flex;">
        <SidebarMenu />
        
        <div style="flex: 1; padding: 2rem; margin-left: 300px;">
          <h2>Account View</h2>
          <p>The "By Account" section is expanded showing all account providers and their accounts.</p>
          
          <div style="margin-top: 2rem; padding: 1rem; background: #eff6ff; border-radius: 4px;">
            <strong>Account Structure:</strong>
            <ul>
              <li>Bank of America
                <ul>
                  <li>Checking</li>
                  <li>Savings</li>
                </ul>
              </li>
              <li>Chase
                <ul>
                  <li>Chase Freedom (Credit)</li>
                  <li>Chase Sapphire (Credit)</li>
                </ul>
              </li>
              <li>Vanguard
                <ul>
                  <li>401k (Investment)</li>
                  <li>Roth IRA (Investment)</li>
                  <li>Brokerage (Investment)</li>
                </ul>
              </li>
            </ul>
          </div>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Shows the sidebar with the account section expanded to display all providers and accounts.',
      },
    },
  },
}

/**
 * Full layout example
 */
export const FullLayoutExample = {
  render: () => ({
    components: { SidebarMenu },
    setup() {
      const uiStore = useUiStore()
      
      queryClient.setQueryData(['accounts'], mockAccountData)
      uiStore.openDrawer()
      
      const toggleSidebar = () => {
        uiStore.toggleDrawer()
      }
      
      return { toggleSidebar, uiStore }
    },
    template: `
      <div style="display: flex; height: 100vh;">
        <SidebarMenu />
        
        <div :style="{ flex: 1, display: 'flex', flexDirection: 'column', marginLeft: uiStore.isDrawerVisible ? '300px' : '0', transition: 'margin-left 0.3s' }">
          <!-- Top Bar -->
          <div style="padding: 1rem; background: white; border-bottom: 1px solid #e5e5e5; display: flex; align-items: center; gap: 1rem;">
            <button 
              @click="toggleSidebar"
              style="padding: 0.5rem; background: transparent; border: 1px solid #ccc; border-radius: 4px; cursor: pointer;"
            >
              <i class="pi pi-bars"></i>
            </button>
            <h2 style="margin: 0;">Finance Dashboard</h2>
          </div>
          
          <!-- Main Content -->
          <div style="flex: 1; padding: 2rem; overflow-y: auto; background: #f5f5f5;">
            <div style="max-width: 1200px;">
              <h3>Welcome to Etna Finance</h3>
              <p>Use the sidebar to navigate between different sections:</p>
              
              <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1rem; margin-top: 1.5rem;">
                <div style="padding: 1.5rem; background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
                  <i class="pi pi-list" style="font-size: 2rem; color: #335c67;"></i>
                  <h4>Transactions</h4>
                  <p style="color: #666;">View and manage all your financial transactions</p>
                </div>
                
                <div style="padding: 1.5rem; background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
                  <i class="pi pi-chart-line" style="font-size: 2rem; color: #335c67;"></i>
                  <h4>Reports</h4>
                  <p style="color: #666;">Analyze your income and expenses</p>
                </div>
                
                <div style="padding: 1.5rem; background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
                  <i class="pi pi-dollar" style="font-size: 2rem; color: #335c67;"></i>
                  <h4>Market Data</h4>
                  <p style="color: #666;">Track currencies and stocks</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Full application layout example with sidebar, top bar, and main content area.',
      },
    },
  },
}

/**
 * Responsive behavior demo
 */
export const ResponsiveBehavior = {
  render: () => ({
    components: { SidebarMenu },
    setup() {
      const uiStore = useUiStore()
      
      queryClient.setQueryData(['accounts'], mockAccountData)
      
      const toggleSidebar = () => {
        uiStore.toggleDrawer()
      }
      
      return { toggleSidebar, uiStore }
    },
    template: `
      <div>
        <div style="padding: 2rem; max-width: 800px;">
          <h2>Responsive Sidebar Menu</h2>
          <p>The sidebar automatically opens on desktop (≥1024px) and closes on mobile.</p>
          
          <button 
            @click="toggleSidebar"
            style="margin-top: 1rem; padding: 0.75rem 1.5rem; background: #335c67; color: white; border: none; border-radius: 4px; cursor: pointer;"
          >
            {{ uiStore.isDrawerVisible ? 'Close' : 'Open' }} Sidebar
          </button>
          
          <div style="margin-top: 2rem; padding: 1rem; background: #fef3c7; border-radius: 4px;">
            <strong>Current State:</strong> {{ uiStore.isDrawerVisible ? 'Open' : 'Closed' }}
          </div>
        </div>
        
        <SidebarMenu />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'Demonstrates the responsive behavior and toggle functionality of the sidebar.',
      },
    },
  },
}

