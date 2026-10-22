import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { BucketBreakdown } from './BucketBreakdown'
import { GB, storage } from './fixtures'
import { StorageStats } from './StorageStats'
import { VolumePanel } from './VolumePanel'

// The storage side of the monitoring page: the volume, the totals and the buckets.
const meta = { title: 'features/monitoring/StoragePanels', parameters: { layout: 'padded' } } satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Volume: Story = {
    render: () => <VolumePanel data={storage()} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('local · /var/lib/savvy/uploads')).toBeVisible()
        await expect(canvas.getByText('40%')).toBeVisible()
        // Savvy's own 3 GB of the 40 GB used; the rest is other files on the disk.
        await expect(canvas.getByText('37 GB')).toBeVisible()
    },
}

export const VolumeInRussian: Story = { ...Volume, play: undefined, globals: { locale: 'ru' } }
export const VolumeInDarkTheme: Story = { ...Volume, play: undefined, globals: { theme: 'dark' } }

export const VolumeFillingUp: Story = {
    render: () => (
        <VolumePanel
            data={storage({
                volume: { ...storage().volume, free_bytes: 20 * GB, used_bytes: 80 * GB, used_percent: 80 },
            })}
        />
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/filling up/i)).toBeVisible()
    },
}

export const VolumeCritical: Story = {
    render: () => (
        <VolumePanel
            data={storage({ volume: { ...storage().volume, free_bytes: 3 * GB, used_bytes: 97 * GB, used_percent: 97 } })}
        />
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/critical/i)).toBeVisible()
    },
}

export const VolumeUnknown: Story = {
    render: () => (
        <VolumePanel
            data={storage({
                volume: { disk: 's3', path: 'bucket/savvy', total_bytes: null, free_bytes: null, used_bytes: null, used_percent: null },
            })}
        />
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText('%')).toBeNull()
    },
}

export const Totals: Story = {
    render: () => <StorageStats data={storage()} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('3 GB')).toBeVisible()
        await expect(canvas.getByText('4,300')).toBeVisible()
    },
}

export const TotalsInRussian: Story = { ...Totals, play: undefined, globals: { locale: 'ru' } }

// Statuses the server did not report count as none.
export const TotalsEmpty: Story = {
    render: () => (
        <StorageStats
            data={storage({
                managed: { used_bytes: 0, objects: 0, pending_bytes: 0, buckets: [] },
                uploads: { total: 0, by_status: {} },
                imports: { total: 0, by_status: {} },
            })}
        />
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByText('0 B')).toHaveLength(2)
    },
}

export const Buckets: Story = {
    render: () => <BucketBreakdown data={storage()} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('backups')).toBeVisible()
        await expect(canvas.getByText('83%')).toBeVisible()
        await expect(canvas.getByText('17%')).toBeVisible()
    },
}

export const NoBuckets: Story = {
    render: () => <BucketBreakdown data={storage({ managed: { ...storage().managed, buckets: [] } })} />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('list')).toBeNull()
    },
}
