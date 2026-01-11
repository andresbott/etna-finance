

/** @type { import('@storybook/vue3-vite').StorybookConfig } */
const config = {
  "stories": [
    "../src/**/*.mdx",
    "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"
  ],
  "addons": ["@storybook/addon-a11y", "@storybook/addon-vitest"],
  "framework": "@storybook/vue3-vite"
};
export default config;