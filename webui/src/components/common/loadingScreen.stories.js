import LoadingScreen from './loadingScreen.vue'

export default {
  title: 'Components/LoadingScreen',
  component: LoadingScreen,
  tags: ['autodocs'],
  parameters: {
    // More on how to position stories at: https://storybook.js.org/docs/configure/story-layout
    layout: 'fullscreen',
  },
}

export const Default = {
  render: () => ({
    components: { LoadingScreen },
    template: '<LoadingScreen />',
  }),
}

