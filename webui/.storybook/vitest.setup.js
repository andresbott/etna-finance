import { setProjectAnnotations } from '@storybook/vue3'
import * as previewAnnotations from './preview.js'

// Apply all the project-level annotations (decorators, parameters, etc.) 
// from Storybook preview to the Vitest tests
const annotations = setProjectAnnotations([previewAnnotations])

// If you have any global setup that needs to happen before each test, 
// you can export beforeAll or beforeEach hooks here
export const beforeAll = annotations.beforeAll

