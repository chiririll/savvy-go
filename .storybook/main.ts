import type { StorybookConfig } from '@storybook/react-vite'

const config: StorybookConfig = {
    framework: '@storybook/react-vite',
    stories: ['../resources/ts/**/*.stories.@(ts|tsx)'],
    addons: ['@storybook/addon-a11y', '@storybook/addon-docs', '@storybook/addon-vitest'],
    staticDirs: ['./public'],
    core: { disableTelemetry: true },
}

export default config
