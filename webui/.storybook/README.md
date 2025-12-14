# Storybook Configuration

This project uses Storybook v10 for component development and documentation.

## Running Storybook

To start the Storybook development server:

```bash
make storybook
```

Or directly with npm:

```bash
npm run storybook
```

Storybook will be available at http://localhost:6006/

## Creating Stories

Stories are located alongside their components with the `.stories.js` extension.

### Component Story Example

```javascript
import MyComponent from './MyComponent.vue'

export default {
  title: 'Components/ComponentName',
  component: MyComponent,
  tags: ['autodocs'],
}

export const Default = {
  render: () => ({
    components: { MyComponent },
    template: '<MyComponent />',
  }),
}
```

### Page Story Example

Page stories (full views/screens) require additional setup for router and state management:

```javascript
import MyView from './MyView.vue'

export default {
  title: 'Pages/MyView',
  component: MyView,
  parameters: {
    layout: 'fullscreen', // Important for page stories
  },
  decorators: [
    () => ({
      template: '<div style="height: 100vh;"><story /></div>',
    }),
  ],
}
```

### Mocking Data with Vue Query

For components that use composables with `@tanstack/vue-query`, you can mock data using the query client:

```javascript
import { queryClient } from '../../.storybook/preview.js'

export const Default = {
  render: () => ({
    components: { MyComponent },
    setup() {
      queryClient.setQueryData(['myQuery'], mockData)
      return {}
    },
    template: '<MyComponent />',
  }),
}
```

## Configuration

- **Main Config**: `.storybook/main.js` - Storybook configuration and addons
- **Preview Config**: `.storybook/preview.js` - Global decorators and parameters
- **Theme**: PrimeVue with custom theme is automatically configured
- **Global Setup**: Pinia, Vue Router, Vue Query, and common directives are configured globally

## Available Addons

- `@chromatic-com/storybook` - Visual testing
- `@storybook/addon-a11y` - Accessibility testing
- `@storybook/addon-docs` - Auto-generated documentation

## Current Stories

### Components

#### AccountSelector
**Path:** `src/components/AccountSelector.stories.js`

Hierarchical account selector using TreeSelect with provider grouping.

**Stories:** Default, WithPreselectedAccount, Disabled, FilterCheckingAccounts, FilterSavingsAndInvestment, Required, InFormContext, Loading

#### CategorySelect
**Path:** `src/components/common/categorySelect.stories.js`

Category selector with tree structure for expense or income categories.

**Stories:** ExpenseCategories, IncomeCategories, WithPreselection, RootSelected, InTransactionForm

#### ConfirmDialog
**Path:** `src/components/common/confirmDialog.stories.js`

Reusable confirmation dialog for delete operations.

**Stories:** Default, CustomContent, WithoutName, OpenByDefault, WithError

#### CustomDrawer
**Path:** `src/components/common/CustomDrawer.stories.js`

Drawer component that slides in from left or right with backdrop.

**Stories:** LeftDrawer, RightDrawer, WithoutHeader, WithRichContent, OpenByDefault

#### DateRangePicker
**Path:** `src/components/common/DateRangePicker.stories.js`

Date range picker with quick select options.

**Stories:** Default, CustomLabels, WithoutIcon, WithoutButtonBar, DifferentFormat, QuickSelectDemo, InFormContext

#### LoadingScreen
**Path:** `src/components/common/loadingScreen.stories.js`

Simple loading spinner overlay.

**Stories:** Default

#### CsvHeaderEditor
**Path:** `src/components/CsvHeaderEditor.stories.js`

Comprehensive CSV header mapping editor with validation.

**Stories:** Empty, WithSampleHeaders, FullyConfigured, WithValidationErrors, BankStatementChase, CreditCardStatement, InteractiveDemo

#### SecondaryMenu
**Path:** `src/components/SecondaryMenu.stories.js`

Right-side drawer for settings and user options.

**Stories:** Default, OpenByDefault, DifferentUser, NavigationExample, WithExplanation

#### SidebarMenu
**Path:** `src/components/SidebarMenu.stories.js`

Main sidebar navigation with hierarchical account structure.

**Stories:** Default, OpenByDefault, AccountsExpanded, FullLayoutExample, ResponsiveBehavior

### Pages

#### LoginView
**Path:** `src/views/LoginView.stories.js`

Full login page composition.

**Stories:** Default, WithTestData, DarkMode, Loading

## Story Organization

All stories are organized in the Storybook UI as follows:

```
Components/
├── AccountSelector
├── CsvHeaderEditor
├── SecondaryMenu
├── SidebarMenu
└── Common/
    ├── CategorySelect
    ├── ConfirmDialog
    ├── CustomDrawer
    ├── DateRangePicker
    └── LoadingScreen

Pages/
└── LoginView
```

## Tips

- Use `layout: 'fullscreen'` for page stories
- Use `layout: 'padded'` for component stories (default)
- Use `layout: 'centered'` for small isolated components
- Mock external dependencies (API calls, stores) in decorators
- Use `tags: ['autodocs']` to enable automatic documentation generation
- Import and use `queryClient` from preview.js for mocking Vue Query data
- Store state (Pinia) is globally available in all stories

## Statistics

- **Total Stories:** 10 component files + 1 page file = 11 files
- **Total Story Variants:** 50+ individual story variants
- **Coverage:** All major components documented with multiple use cases
