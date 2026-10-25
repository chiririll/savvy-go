import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import type { Backup } from '@/types/backup'

import { BackupsTable } from './BackupsTable'

const backup = (overrides: Partial<Backup> = {}): Backup => ({
    filename: 'savvy-2030-01-05.zip',
    size: 2_345_678,
    note: 'Before the upgrade',
    appVersion: '1.4.0',
    kind: 'server',
    status: 'current',
    signature: 'own',
    restorable: true,
    createdAt: '2030-01-05T10:00:00Z',
    ...overrides,
})

const meta = {
    tags: ['desktop-only'],
    title: 'features/backups/BackupsTable',
    component: BackupsTable,
    parameters: { a11y: overlayA11y },
    args: {
        backups: [
            backup(),
            backup({ filename: 'b.zip', note: null, appVersion: null, status: 'unsigned', signature: 'unsigned' }),
            backup({ filename: 'c.zip', status: 'invalid', restorable: false }),
        ],
        isLoading: false,
        onDownload: fn(),
        onRestore: fn(),
        onDelete: fn(),
    },
} satisfies Meta<typeof BackupsTable>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Loading: Story = { args: { isLoading: true } }
export const Empty: Story = {
    args: { backups: [] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/no backups/i)).toBeVisible()
    },
}
export const LongNote: Story = {
    args: { backups: [backup({ note: 'A note that goes on far beyond what a narrow phone column can hold on one line' })] },
}

export const DownloadAsked: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /download/i })[0])
        await expect(args.onDownload).toHaveBeenCalledWith(args.backups![0])
    },
}

export const RestoreAsked: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /restore/i })[0])
        await expect(args.onRestore).toHaveBeenCalledWith(args.backups![0])
    },
}

// A backup this app cannot read cannot be restored, and the button says why.
export const UnreadableCannotBeRestored: Story = {
    play: async ({ canvasElement }) => {
        const rows = within(canvasElement).getAllByRole('row')
        const last = within(rows[rows.length - 1]).getAllByRole('button')
        await expect(last[1]).toBeDisabled()
    },
}

export const DeleteAsked: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /delete/i })[1])
        await expect(args.onDelete).toHaveBeenCalledWith(args.backups![1])
    },
}
