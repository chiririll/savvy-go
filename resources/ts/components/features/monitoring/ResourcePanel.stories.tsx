import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { GB, MB, resources } from './fixtures'
import { Gauge } from './Gauge'
import { ResourcePanel } from './ResourcePanel'

const meta = {
    title: 'features/monitoring/ResourcePanel',
    component: ResourcePanel,
    parameters: { layout: 'padded' },
    args: { data: resources() },
} satisfies Meta<typeof ResourcePanel>

export default meta
type Story = StoryObj<typeof meta>

export const Healthy: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('1.25.1')).toBeVisible()
        await expect(canvas.getByText('0.42')).toBeVisible()
        await expect(canvas.getByText('1,234')).toBeVisible()
        await expect(canvas.getByText('19% of limit')).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

// Over 90% turns red and says so under the uptime.
export const UnderLoad: Story = {
    args: {
        data: resources({
            cpu: { cores: 2, load: [3.9, 3.5, 2.8], load_percent: 97 },
            memory: { total_bytes: 2 * GB, used_bytes: 1.6 * GB, free_bytes: 0.4 * GB, used_percent: 80, source: 'host' },
        }),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('97%')).toBeVisible()
        await expect(canvas.getByText('80%')).toBeVisible()
    },
}

// A host that tells nothing: no load, no memory, no process limit.
export const NothingKnown: Story = {
    args: {
        data: resources({
            cpu: { cores: null, load: null, load_percent: null },
            memory: { total_bytes: null, used_bytes: null, free_bytes: null, used_percent: null, source: null },
            process: { memory_bytes: 50 * MB, peak_bytes: 80 * MB, limit_bytes: null, goroutines: 12, gc_cycles: 3 },
            runtime: { go_version: 'go1.25.1', app_version: 'dev', environment: 'local', uptime_seconds: null },
        }),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getAllByText('—').length).toBeGreaterThanOrEqual(3)
        await expect(canvas.getByText('peak 80 MB')).toBeVisible()
    },
}

export const JustStarted: Story = {
    args: { data: resources({ runtime: { ...resources().runtime, uptime_seconds: 125 } }) },
}

export const Gauges: Story = {
    render: () => (
        <div className="flex flex-wrap gap-6">
            <Gauge percent={12} label="CPU" size={150} />
            <Gauge percent={78} label="Memory" size={150} />
            <Gauge percent={95} label="Disk" />
            <Gauge percent={null} label="Unknown" size={150} />
            <Gauge percent={140} label="Clamped" size={150} />
        </div>
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('—')).toBeVisible()
    },
}
