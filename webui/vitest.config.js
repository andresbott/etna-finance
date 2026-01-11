import { fileURLToPath } from 'node:url'
import { defineConfig, configDefaults } from 'vitest/config'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import path from 'node:path'

const dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  test: {
    // Use projects configuration to separate regular tests and Storybook tests
    projects: [
      {
        // Regular unit tests configuration
        extends: './vite.config.js',
        test: {
          name: 'unit',
          environment: 'jsdom',
          exclude: [...configDefaults.exclude, 'e2e/*', '**/*.stories.*'],
          root: fileURLToPath(new URL('./', import.meta.url))
        }
      },
      {
        // Storybook tests configuration
        extends: './vite.config.js',
        plugins: [
          storybookTest({
            configDir: path.join(dirname, '.storybook'),
            storybookScript: 'npm run storybook -- --no-open --ci'
          })
        ],
        test: {
          name: 'storybook',
          browser: {
            enabled: true,
            provider: 'playwright',
            headless: true,
            instances: [{ browser: 'chromium' }]
          },
          setupFiles: ['./.storybook/vitest.setup.js']
        }
      }
    ]
  }
})
