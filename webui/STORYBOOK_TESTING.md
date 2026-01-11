# Storybook + Vitest Testing Setup

This project is configured to run unit tests directly from Storybook stories using the `@storybook/addon-vitest` addon.

## Overview

The setup allows you to:
- Write component tests as Storybook stories with `play` functions
- Run tests in a real browser environment using Playwright
- Use Storybook's testing utilities (`@storybook/test`) including `expect`, `userEvent`, and `within`
- Keep visual documentation and tests in the same place
- Run tests independently or as part of your CI/CD pipeline

## Setup Details

### Installed Packages
- `@storybook/addon-vitest` - Storybook addon for Vitest integration
- `@storybook/test` - Testing utilities (expect, userEvent, within, etc.)
- `@vitest/browser` - Browser mode for Vitest
- `@vitest/coverage-v8` - Code coverage support
- `playwright` - Browser automation for running tests
- `vitest@^3.0.0` - Test runner (upgraded from 2.x)

### Configuration Files

#### `.storybook/main.js`
The Storybook addon is registered here:
```js
addons: ["@storybook/addon-vitest"]
```

#### `.storybook/vitest.setup.js`
This file applies Storybook's configuration (decorators, parameters, etc.) to Vitest tests.

#### `vitest.config.js`
Configured with a workspace setup:
- **unit**: Regular unit tests (uses jsdom, excludes stories)
- **storybook**: Story-based tests (uses Playwright browser, includes only stories)

## Writing Tests

### Basic Structure

Create a story with a `play` function to define tests:

```js
import { expect, userEvent, within } from '@storybook/test'
import MyComponent from './MyComponent.vue'

export default {
  title: 'Components/MyComponent',
  component: MyComponent,
}

export const Default = {
  args: {
    label: 'Click me'
  },
  play: async ({ canvasElement, args, step }) => {
    const canvas = within(canvasElement)
    
    await step('Component should render', async () => {
      const button = canvas.getByRole('button')
      expect(button).toBeInTheDocument()
    })
    
    await step('Button should be clickable', async () => {
      const button = canvas.getByRole('button')
      await userEvent.click(button)
      expect(args.onClick).toHaveBeenCalled()
    })
  }
}
```

### Example: Button Component

See `src/components/common/Button.vue` and `Button.stories.js` for a complete example demonstrating:
- Multiple story variants (Primary, Secondary, Danger, Disabled)
- Testing render output
- Testing user interactions
- Testing disabled states
- Using `step()` for organized test output

## Running Tests

### Run All Tests
```bash
npm test
# OR
make test
```

### Run Only Unit Tests
```bash
npm run test:unit
# OR
make test-unit
```

### Run Only Storybook Tests
```bash
npm run test:storybook
# OR
make test-storybook
```

### Watch Mode
```bash
npm run test:watch
# OR
make test-watch
```

### With UI
```bash
npm run test:ui
# OR
make test-ui
```

### With Coverage
```bash
npm run test:coverage
# OR
make test-coverage
```

## Running Storybook

Start Storybook to view components and run tests interactively:

```bash
npm run storybook
```

Then navigate to the Storybook UI. Stories with `play` functions will show test results in the "Interactions" panel.

## Testing Utilities

The `@storybook/test` package provides:

### `expect` (from Vitest)
Assertion library:
```js
expect(button).toBeInTheDocument()
expect(button).toHaveClass('btn--primary')
expect(button).not.toBeDisabled()
```

### `within` (from Testing Library)
Query elements within a container:
```js
const canvas = within(canvasElement)
const button = canvas.getByRole('button')
const input = canvas.getByLabelText('Email')
```

### `userEvent` (from Testing Library)
Simulate user interactions:
```js
await userEvent.click(button)
await userEvent.type(input, 'test@example.com')
await userEvent.selectOptions(select, 'option1')
```

### `step`
Organize tests into logical steps:
```js
await step('Setup', async () => { /* ... */ })
await step('User interaction', async () => { /* ... */ })
await step('Verification', async () => { /* ... */ })
```

## Play Function Parameters

The `play` function receives:
- `canvasElement` - The DOM element containing the story
- `args` - Story arguments/props
- `step` - Function to organize tests into steps
- `parameters` - Story parameters
- `globals` - Global Storybook values

## Best Practices

1. **Use `step()` for clarity** - Group related assertions into named steps
2. **Test user perspective** - Query by role, label, or text (not by class or ID)
3. **Keep stories focused** - One story = one scenario
4. **Test interactions** - Use `userEvent` to simulate real user behavior
5. **Document with stories** - Stories serve as both tests and documentation
6. **Avoid test IDs** - Prefer semantic queries (role, label, text)

## CI/CD Integration

To run tests in CI:

```bash
# Run all tests (unit + storybook)
npm test

# Or run them separately
npm run test:unit
npm run test:storybook

# With coverage
npm run test:coverage
```

The tests run in headless Chromium via Playwright, so no display server is needed.

## Troubleshooting

### Playwright Installation
If you encounter issues with Playwright browsers:
```bash
npx playwright install chromium
```

### Port Conflicts
If Storybook port 6006 is in use, tests may fail. Ensure no other Storybook instance is running.

### Timeout Issues
If tests timeout, you can increase the timeout in `vitest.config.js`:
```js
test: {
  testTimeout: 30000, // 30 seconds
}
```

## Further Reading

- [Storybook Testing Handbook](https://storybook.js.org/docs/writing-tests)
- [Vitest Documentation](https://vitest.dev/)
- [Testing Library Queries](https://testing-library.com/docs/queries/about)
- [Playwright Documentation](https://playwright.dev/)

