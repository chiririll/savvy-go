import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'
import { overlayA11y } from '@/test/a11y'
import { currencyApi, reply } from '@/test/api-mocks'
import { accounts } from '@/test/fixtures'

import { parsed } from '../fixtures'
import { MappingStep } from './MappingStep'

const meta = {
    title: 'features/import/MappingStep',
    component: MappingStep,
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/accounts', accounts), ...currencyApi()),
    args: { parseResult: parsed(), onSubmit: fn(), isLoading: false },
    // The wizard's footer submits the form from outside it.
    render: (args) => (
        <div className="space-y-6">
            <MappingStep {...args} />
            <Button type="submit" form="mapping-form">
                Preview
            </Button>
        </div>
    ),
} satisfies Meta<typeof MappingStep>

export default meta
type Story = StoryObj<typeof meta>

const submit = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: 'Preview' }))

export const Suggested: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('combobox', { name: /date column/i })).toHaveTextContent('1: Date')
        await expect(canvas.getByRole('combobox', { name: /amount column/i })).toHaveTextContent('2: Amount')
        await expect(canvas.getByRole('combobox', { name: /type column/i })).toHaveTextContent(/auto-detect/i)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

// The account is not guessed: everything lands in it.
export const AccountRequired: Story = {
    play: async ({ canvasElement, args }) => {
        await submit(canvasElement)
        await expect(await within(canvasElement).findByText('Account is required')).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const NothingSuggested: Story = {
    args: { parseResult: parsed({ suggestedMapping: {} }) },
    play: async ({ canvasElement }) => {
        await submit(canvasElement)
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Date column is required')).toBeVisible()
        await expect(canvas.getByText('Amount column is required')).toBeVisible()
    },
}

export const Submitted: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('combobox', { name: /target account/i }))
        await userEvent.click(await screen.findByRole('option', { name: /kuwait savings/i }))
        await userEvent.click(canvas.getByRole('combobox', { name: /amount format/i }))
        await userEvent.click(await screen.findByRole('option', { name: /EU/ }))
        await userEvent.click(canvas.getByRole('combobox', { name: /description column/i }))
        await userEvent.click(await screen.findByRole('option', { name: /none/i }))
        await submit(canvasElement)
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        await expect(args.onSubmit).toHaveBeenCalledWith(
            { date: 0, amount: 1, description: null, type: null, category: 3, tags: null, currency: 4 },
            {
                dateFormat: 'ISO',
                amountFormat: 'EU',
                defaultAccountId: 4,
                defaultType: 'expense',
                skipFirstRow: true,
                createMissingCurrencies: true,
                createMissingTags: true,
                createMissingCategories: true,
            },
        )
    },
}

// A file without a header row starts reading at the first row.
export const WithoutHeader: Story = {
    args: {
        parseResult: parsed({
            headers: ['Column 1', 'Column 2'],
            detectedFormats: { dateFormat: 'DD/MM/YYYY', amountFormat: 'US', hasHeader: false, delimiter: ';' },
            suggestedMapping: { date: 0, amount: 1 },
        }),
    },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('combobox', { name: /^date format/i })).toHaveTextContent('DD/MM/YYYY')
        await userEvent.click(canvas.getByRole('combobox', { name: /target account/i }))
        await userEvent.click(await screen.findByRole('option', { name: /^cash/i }))
        await submit(canvasElement)
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        const options = (args.onSubmit as ReturnType<typeof fn>).mock.calls[0][1] as { skipFirstRow: boolean }
        await expect(options.skipFirstRow).toBe(false)
    },
}

export const Loading: Story = {
    args: { isLoading: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('combobox', { name: /date column/i })).toBeDisabled()
    },
}
