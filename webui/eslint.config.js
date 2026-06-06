import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'
import tsParser from '@typescript-eslint/parser'
import tsPlugin from '@typescript-eslint/eslint-plugin'
import globals from 'globals'

export default [
    {
        name: 'app/files-to-lint',
        files: ['**/*.{js,mjs,cjs,jsx,ts,mts,tsx,vue}']
    },
    {
        name: 'app/files-to-ignore',
        ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**']
    },
    {
        name: 'app/language-options',
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.browser,
                ...globals.node
            }
        }
    },
    js.configs.recommended,
    ...pluginVue.configs['flat/essential'],
    // Parse <script lang="ts"> blocks in .vue files with the TypeScript parser
    // (eslint-plugin-vue already sets vue-eslint-parser as the .vue parser).
    {
        name: 'app/typescript-in-vue',
        files: ['**/*.vue'],
        languageOptions: {
            parserOptions: { parser: tsParser }
        }
    },
    // Parse standalone TypeScript files with the TypeScript parser.
    {
        name: 'app/typescript',
        files: ['**/*.{ts,mts,tsx}'],
        languageOptions: { parser: tsParser }
    },
    // TypeScript-aware rules: the core no-unused-vars/no-undef rules misfire on
    // TS syntax (interface augmentation, function-type params, type-only refs).
    {
        name: 'app/typescript-rules',
        files: ['**/*.{ts,mts,tsx,vue}'],
        plugins: { '@typescript-eslint': tsPlugin },
        rules: {
            // The TS compiler already reports undefined identifiers.
            'no-undef': 'off',
            // Defer to the TS-aware rule; honor the `_` "intentionally unused" convention.
            'no-unused-vars': 'off',
            '@typescript-eslint/no-unused-vars': [
                'error',
                {
                    argsIgnorePattern: '^_',
                    varsIgnorePattern: '^_',
                    caughtErrorsIgnorePattern: '^_',
                    // Allow `const { omitted, ...rest } = obj` to drop a property.
                    ignoreRestSiblings: true
                }
            ]
        }
    },
    // Allow intentionally empty catch blocks (best-effort operations).
    {
        name: 'app/general-rules',
        rules: {
            'no-empty': ['error', { allowEmptyCatch: true }]
        }
    },
    skipFormatting
]
