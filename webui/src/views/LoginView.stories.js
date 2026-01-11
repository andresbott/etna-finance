import LoginView from './LoginView.vue'

export default {
  title: 'Pages/LoginView',
  component: LoginView,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'The Login page composition showing the full authentication screen with centered layout.',
      },
    },
  },
  decorators: [
    () => ({
      template: '<div style="height: 100vh; width: 100vw;"><story /></div>',
    }),
  ],
}

/**
 * Default login page state - ready for user input
 */
export const Default = {
  render: () => ({
    components: { LoginView },
    template: '<LoginView />',
  }),
}

/**
 * Login page with pre-filled test credentials for demonstration
 */
export const WithTestData = {
  render: () => ({
    components: { LoginView },
    setup() {
      // Simulate pre-filled form (this would typically be done via browser autofill)
      setTimeout(() => {
        const usernameInput = document.getElementById('username')
        const passwordInput = document.getElementById('password')
        if (usernameInput) usernameInput.value = 'testuser'
        if (passwordInput) passwordInput.value = '••••••••'
      }, 100)
    },
    template: '<LoginView />',
  }),
  parameters: {
    docs: {
      description: {
        story: 'Login form with pre-filled test credentials to show the filled state.',
      },
    },
  },
}

/**
 * Dark mode version of the login page
 */
export const DarkMode = {
  render: () => ({
    components: { LoginView },
    mounted() {
      document.documentElement.classList.add('dark-mode')
    },
    unmounted() {
      document.documentElement.classList.remove('dark-mode')
    },
    template: '<LoginView />',
  }),
  parameters: {
    docs: {
      description: {
        story: 'Login page in dark mode, showing the theme adaptation.',
      },
    },
  },
}

/**
 * Login page showing loading state
 */
export const Loading = {
  render: () => ({
    components: { LoginView },
    setup() {
      // Simulate loading state
      setTimeout(() => {
        const submitButton = document.getElementById('login-submit')
        if (submitButton) {
          submitButton.click()
        }
      }, 500)
    },
    template: '<LoginView />',
  }),
  parameters: {
    docs: {
      description: {
        story: 'Login page in loading state after form submission.',
      },
    },
  },
}

