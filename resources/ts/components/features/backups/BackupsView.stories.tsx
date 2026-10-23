import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import type { Backup } from '@/types/backup'

import { BackupsView } from './BackupsView'
import type { BackupSource } from './source'

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

const create = fn()
const upload = fn()
const restore = fn()
const remove = fn()
const laravelRestore = fn()

const source = (backups: Backup[], withLaravel = false): BackupSource => ({
    key: ['story-backups'],
    list: async () => backups,
    create: async (note) => {
        create(note)
        return backup()
    },
    upload: async (file, note) => {
        upload(file.name, note)
        return backup()
    },
    downloadUrl: (filename) => `/api/admin/backups/${filename}`,
    restore: async (filename) => restore(filename),
    remove: async (filename) => remove(filename),
    laravel: withLaravel
        ? { targets: ['server', 'new'], restore: async (file, target, appKey) => laravelRestore(file.name, target, appKey) }
        : undefined,
})

const two = [backup(), backup({ filename: 'older.zip', note: null, status: 'unsigned', signature: 'unsigned' })]

const meta = {
    title: 'features/backups/BackupsView',
    component: BackupsView,
    parameters: { a11y: overlayA11y, layout: 'fullscreen' },
    beforeEach: () => {
        for (const spy of [create, upload, restore, remove, laravelRestore]) spy.mockClear()
    },
    args: { source: source(two), title: 'Backups', description: 'Copies of the whole server.' },
} satisfies Meta<typeof BackupsView>

export default meta
type Story = StoryObj<typeof meta>

export const WithBackups: Story = {
    play: async () => {
        await expect(await screen.findByText('Before the upgrade')).toBeVisible()
    },
}
export const NoBackups: Story = { args: { source: source([]) } }
export const WithExtraActions: Story = { args: { actions: <button type="button">Extra</button> } }

export const CreateAsksForANote: Story = {
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /create/i }))
        await expect(await screen.findByLabelText(/note/i)).toBeVisible()
    },
}

export const Created: Story = {
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /^create/i }))
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.type(dialog.getByLabelText(/note/i), 'Monthly')
        await userEvent.click(dialog.getByRole('button', { name: /^create$/i }))
        await waitFor(() => expect(create).toHaveBeenCalledWith('Monthly'))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

export const CreatedWithoutANote: Story = {
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /^create/i }))
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /^create$/i }))
        await waitFor(() => expect(create).toHaveBeenCalledWith(undefined))
    },
}

export const UploadNeedsAFile: Story = {
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /upload/i }))
        const dialog = within(await screen.findByRole('dialog'))
        await expect(dialog.getByRole('button', { name: /upload/i })).toBeDisabled()
    },
}

export const Uploaded: Story = {
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /upload/i }))
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.upload(dialog.getByLabelText(/backup file/i), new File(['x'], 'mine.zip'))
        await userEvent.type(dialog.getByLabelText(/note/i), 'Old laptop')
        await userEvent.click(dialog.getByRole('button', { name: /upload/i }))
        await waitFor(() => expect(upload).toHaveBeenCalledWith('mine.zip', 'Old laptop'))
    },
}

// A bare database file is a Laravel one: it asks what to make of it instead of for a note.
export const LaravelFileAsksWhatToDo: Story = {
    args: { source: source(two, true) },
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /upload/i }))
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.upload(dialog.getByLabelText(/backup file/i), new File(['x'], 'old.sqlite'))
        await expect(await dialog.findAllByRole('radio')).toHaveLength(2)
        await expect(dialog.queryByLabelText(/^note/i)).toBeNull()
    },
}

export const LaravelFileUpgraded: Story = {
    args: { source: source(two, true) },
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /upload/i }))
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.upload(dialog.getByLabelText(/backup file/i), new File(['x'], 'old.sqlite'))
        await userEvent.type(await dialog.findByLabelText(/app_key/i), ' base64:abc ')
        await userEvent.click(dialog.getByRole('button', { name: /upgrade and save/i }))
        await waitFor(() => expect(laravelRestore).toHaveBeenCalledWith('old.sqlite', 'server', 'base64:abc'))
    },
}

export const RestoreAsksFirst: Story = {
    play: async () => {
        await userEvent.click((await screen.findAllByRole('button', { name: /restore/i }))[0])
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByRole('button', { name: /restore/i })).toBeVisible()
        await expect(restore).not.toHaveBeenCalled()
    },
}

export const RestoreUnsignedWarns: Story = {
    play: async () => {
        await userEvent.click((await screen.findAllByRole('button', { name: /restore/i }))[1])
        await expect(within(await screen.findByRole('alertdialog')).getAllByText(/./).length).toBeGreaterThan(3)
    },
}

export const Restored: Story = {
    play: async () => {
        await userEvent.click((await screen.findAllByRole('button', { name: /restore/i }))[0])
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /restore/i }))
        await waitFor(() => expect(restore).toHaveBeenCalledWith('savvy-2030-01-05.zip'))
    },
}

export const DeleteAsksFirst: Story = {
    play: async () => {
        await userEvent.click((await screen.findAllByRole('button', { name: /delete/i }))[0])
        await expect(await screen.findByRole('alertdialog')).toBeVisible()
        await expect(remove).not.toHaveBeenCalled()
    },
}

export const Deleted: Story = {
    play: async () => {
        await userEvent.click((await screen.findAllByRole('button', { name: /delete/i }))[1])
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await waitFor(() => expect(remove).toHaveBeenCalledWith('older.zip'))
    },
}

export const DeleteCancelled: Story = {
    play: async () => {
        await userEvent.click((await screen.findAllByRole('button', { name: /delete/i }))[0])
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull())
        await expect(remove).not.toHaveBeenCalled()
    },
}
