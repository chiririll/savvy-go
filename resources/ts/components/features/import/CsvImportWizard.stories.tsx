import type { Meta, StoryObj } from '@storybook/react-vite'
import { http, HttpResponse } from 'msw'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { currencyApi, fail, raw, reply, spy } from '@/test/api-mocks'
import { accounts, categories } from '@/test/fixtures'
import type { ImportState } from '@/types/import'

import { CsvImportWizard } from './CsvImportWizard'
import { parsed, preview, result } from './fixtures'

const state = (overrides: Partial<ImportState>): ImportState => ({
    importId: 'imp1',
    status: 'parsed',
    totalRows: 120,
    processedRows: 0,
    created: 0,
    skipped: 0,
    errors: 0,
    message: null,
    parse: null,
    result: null,
    ...overrides,
})

/**
 * Import requests belong to the current space, so their paths may start with /api/spaces/{id}.
 * The file goes up in parts to storage, then the server reads it. The status endpoint answers
 * for whichever job ran last: the parse, then the import.
 */
function server(status: { current: ImportState }) {
    return [
        raw('post', '/s3/multipart', { uploadId: 'u1', key: 'k1', bucket: 'b', partSize: 0, expiresAt: null }),
        raw('get', '/s3/multipart/u1/1', { url: '/storage/part-1', method: 'PUT', headers: {}, expires: 0 }),
        raw('post', '/s3/multipart/u1/complete', { location: 'storage/k1', key: 'k1', uploadId: 'u1' }),
        http.put('/storage/part-1', () => new HttpResponse(null, { headers: { ETag: '"e1"' } })),
        http.post(/\/api(\/spaces\/\d+)?\/transactions\/import\/parse$/, () => {
            status.current = state({ status: 'parsed', parse: parsed() })
            return HttpResponse.json({ data: state({ status: 'parsing' }) })
        }),
        http.get(/\/api(\/spaces\/\d+)?\/transactions\/import\/imp1$/, () => HttpResponse.json({ data: status.current })),
        reply('get', '/accounts', accounts),
        reply('get', '/categories', categories),
        ...currencyApi(),
    ]
}

const status = { current: state({}) }
const sent: { preview: unknown[]; execute: unknown[] } = { preview: [], execute: [] }

const meta = {
    title: 'features/import/CsvImportWizard',
    component: CsvImportWizard,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        status.current = state({})
        const previewSpy = spy('post', '/transactions/import/preview', preview())
        const execute = http.post(/\/api(\/spaces\/\d+)?\/transactions\/import\/execute$/, async ({ request }) => {
            sent.execute.push(await request.json())
            status.current = state({ status: 'completed', processedRows: 120, created: 117, result: result() })
            return HttpResponse.json({ data: state({ status: 'importing' }) })
        })
        sent.preview = previewSpy.bodies
        sent.execute = []
        msw.use(...server(status), previewSpy.handler, execute)
    },
} satisfies Meta<typeof CsvImportWizard>

export default meta
type Story = StoryObj<typeof meta>

const upload = async (canvasElement: HTMLElement) => {
    const input = canvasElement.querySelector<HTMLInputElement>('#csv-upload')!
    await userEvent.upload(input, new File(['Date,Amount\n2030-01-05,-42.50\n'], 'bank.csv', { type: 'text/csv' }))
    await expect(await within(canvasElement).findByText('120 rows detected', {}, { timeout: 5000 })).toBeVisible()
}

const button = (canvasElement: HTMLElement, name: RegExp) => within(canvasElement).getByRole('button', { name })

export const Start: Story = {
    play: async ({ canvasElement }) => {
        await expect(button(canvasElement, /^next$/i)).toBeDisabled()
        await expect(within(canvasElement).queryByRole('button', { name: /back/i })).toBeNull()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const Uploaded: Story = {
    play: async ({ canvasElement }) => {
        await upload(canvasElement)
        await userEvent.click(button(canvasElement, /^next$/i))
        await expect(await within(canvasElement).findByRole('combobox', { name: /date column/i })).toBeVisible()
    },
}

// The whole way, from the file to the result, with the options the server gets.
export const Imported: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await upload(canvasElement)
        await userEvent.click(button(canvasElement, /^next$/i))
        await userEvent.click(await canvas.findByRole('combobox', { name: /target account/i }))
        await userEvent.click(await screen.findByRole('option', { name: /euro card/i }))
        await userEvent.click(button(canvasElement, /^preview$/i))
        await expect(await canvas.findByText(/categories from the file/i)).toBeVisible()
        await waitFor(() => expect(sent.preview).toHaveLength(1))
        await expect(sent.preview[0]).toMatchObject({
            import_id: 'imp1',
            mapping: { date: 0, amount: 1, description: 2, category: 3, currency: 4 },
            options: { default_account_id: 2, skip_first_row: true, category_map: {} },
        })
        await userEvent.click(button(canvasElement, /import 3 transactions/i))
        await expect(await canvas.findByText(/import complete/i, {}, { timeout: 5000 })).toBeVisible()
        // The category choices travel with the import.
        await expect(sent.execute[0]).toMatchObject({
            options: { category_map: { Groceries: 1, 'Eating out': 'create', Salary: 2 } },
        })
        await userEvent.click(button(canvasElement, /import another file/i))
        await expect(canvas.getByText(/drop csv file here/i)).toBeVisible()
    },
}

export const BackKeepsTheFile: Story = {
    play: async ({ canvasElement }) => {
        await upload(canvasElement)
        await userEvent.click(button(canvasElement, /^next$/i))
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /back/i }))
        await expect(within(canvasElement).getByText('120 rows detected')).toBeVisible()
    },
}

export const ParseFailed: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            http.post(/\/api(\/spaces\/\d+)?\/transactions\/import\/parse$/, () => {
                status.current = state({ status: 'failed', message: 'The file has no rows' })
                return HttpResponse.json({ data: state({ status: 'parsing' }) })
            }),
        ),
    play: async ({ canvasElement }) => {
        const input = canvasElement.querySelector<HTMLInputElement>('#csv-upload')!
        await userEvent.upload(input, new File([''], 'empty.csv', { type: 'text/csv' }))
        await expect(
            await within(canvasElement).findByText('The file has no rows', {}, { timeout: 5000 }),
        ).toBeVisible()
        await expect(button(canvasElement, /^next$/i)).toBeDisabled()
    },
}

export const PreviewFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/transactions/import/preview', 'The amount column has text')),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await upload(canvasElement)
        await userEvent.click(button(canvasElement, /^next$/i))
        await userEvent.click(await canvas.findByRole('combobox', { name: /target account/i }))
        await userEvent.click(await screen.findByRole('option', { name: /euro card/i }))
        await userEvent.click(button(canvasElement, /^preview$/i))
        await expect(await canvas.findByRole('alert')).toHaveTextContent(/the amount column has text/i)
        // Still on the mapping step, to fix it.
        await expect(canvas.getByRole('combobox', { name: /date column/i })).toBeVisible()
    },
}
