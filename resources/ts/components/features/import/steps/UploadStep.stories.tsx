import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { parsed } from '../fixtures'
import { UploadStep } from './UploadStep'

const meta = {
    tags: ['desktop-only'],
    title: 'features/import/UploadStep',
    component: UploadStep,
    parameters: { layout: 'padded' },
    args: {
        onFileSelect: fn(),
        onCancelUpload: fn(),
        parseResult: null,
        isUploading: false,
        isParsing: false,
        uploadPercentage: 0,
        error: null,
    },
} satisfies Meta<typeof UploadStep>

export default meta
type Story = StoryObj<typeof meta>

const fileInput = (canvasElement: HTMLElement) => canvasElement.querySelector<HTMLInputElement>('#csv-upload')!

export const Empty: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/drop csv file here/i)).toBeVisible()
    },
}

export const FileChosen: Story = {
    play: async ({ canvasElement, args }) => {
        const file = new File(['date,amount\n'], 'bank.csv', { type: 'text/csv' })
        await userEvent.upload(fileInput(canvasElement), file)
        await expect(args.onFileSelect).toHaveBeenCalledWith(file)
        const canvas = within(canvasElement)
        await expect(canvas.getByText('bank.csv')).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /clear/i }))
        await expect(canvas.getByText(/drop csv file here/i)).toBeVisible()
    },
}

export const Uploading: Story = {
    args: { isUploading: true, uploadPercentage: 42 },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('42%')).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /cancel/i }))
        await expect(args.onCancelUpload).toHaveBeenCalled()
    },
}

export const Parsing: Story = {
    args: { isParsing: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/reading the file/i)).toBeVisible()
    },
}

export const Failed: Story = {
    args: { error: 'The file is not valid UTF-8' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('The file is not valid UTF-8')).toBeVisible()
    },
}

// The detected formats and the first five rows; currencies of every precision come through as text.
export const Parsed: Story = {
    args: { parseResult: parsed() },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('120 rows detected')).toBeVisible()
        await expect(canvas.getByText('-12.345')).toBeVisible()
        await expect(canvas.getAllByRole('row')).toHaveLength(6)
        await expect(canvas.getByText(/and 115 more rows/i)).toBeVisible()
    },
}

export const TabSeparated: Story = {
    args: {
        parseResult: parsed({
            detectedFormats: { dateFormat: 'DD.MM.YYYY', amountFormat: 'EU', hasHeader: true, delimiter: '\t' },
            previewRows: parsed().previewRows.slice(0, 2),
            totalRows: 2,
        }),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('TAB')).toBeVisible()
        await expect(canvas.queryByText(/more rows/i)).toBeNull()
    },
}
