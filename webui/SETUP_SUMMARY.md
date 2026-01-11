# Storybook Vitest Setup - Quick Start

## What Was Set Up

Your project now has **Storybook integrated with Vitest** for running component tests directly from story files.

### Installed Packages
- `@storybook/addon-vitest` (v10.1.8)
- `@storybook/test` (v8.6.14) 
- `vitest` (upgraded to v3.2.4)
- `@vitest/browser` (v3.2.4)
- `@vitest/coverage-v8` (v3.2.4)
- `playwright` (v1.57.0)

### Configuration Files Created/Modified
1. **`.storybook/main.js`** - Added vitest addon
2. **`.storybook/vitest.setup.js`** - Storybook configuration for tests (NEW)
3. **`vitest.config.js`** - Updated with dual project configuration
4. **`package.json`** - Added new test scripts

## Running Tests

```bash
# Run all tests (unit + storybook)
npm test
# OR
make test

# Run only Storybook tests
npm run test:storybook
# OR
make test-storybook

# Run only unit tests
npm run test:unit
# OR
make test-unit

# Watch mode
npm run test:watch
# OR
make test-watch

# With UI
npm run test:ui
# OR
make test-ui

# With coverage
npm run test:coverage
# OR
make test-coverage
```

## Example Component with Tests

A complete example was created at:
- **Component**: `src/components/common/Button.vue`
- **Story with tests**: `src/components/common/Button.stories.js`

The example demonstrates:
- ✅ Multiple story variants
- ✅ Testing render output with `expect()`
- ✅ Testing user interactions with `userEvent`
- ✅ Testing disabled states
- ✅ Using `step()` for organized test output
- ✅ Mock functions with `fn()`

## Test Results

Current status: **53 of 55 tests passing** ✅

The 2 failing tests are from existing stories with pre-existing issues (not related to this setup):
- `categorySelect.stories.js > In Transaction Form` - has a template error
- `confirmDialog.stories.js > Default` - missing directive

## Writing Your First Test

```javascript
import { expect, userEvent, within, fn } from '@storybook/test'
import MyComponent from './MyComponent.vue'

export const MyStory = {
  args: {
    label: 'Click me',
    onClick: fn()
  },
  play: async ({ canvasElement, args, step }) => {
    const canvas = within(canvasElement)
    
    await step('Should render button', async () => {
      const button = canvas.getByRole('button')
      expect(button).toBeInTheDocument()
    })
    
    await step('Should handle clicks', async () => {
      const button = canvas.getByRole('button')
      await userEvent.click(button)
      expect(args.onClick).toHaveBeenCalled()
    })
  }
}
```

## Documentation

Full documentation available at: **`STORYBOOK_TESTING.md`**

## Next Steps

1. Add tests to existing stories using the `play` function
2. Run `npm run storybook` to see tests in the interactive UI
3. Run `npm run test:storybook` in CI/CD pipelines
4. Check coverage with `npm run test:coverage`

## Benefits

✅ Tests live alongside component documentation  
✅ Tests run in real browser (Playwright)  
✅ Visual debugging in Storybook UI  
✅ No duplicate test setup - reuse story data  
✅ CI/CD ready (headless mode)  

