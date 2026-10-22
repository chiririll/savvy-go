import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { LaravelRestoreOptions, isLaravelFile, type LaravelTarget } from './LaravelRestoreOptions'

type Props = { targets: LaravelTarget[]; initial?: LaravelTarget }

function Stateful({ targets, initial }: Props) {
    const [target, setTarget] = useState<LaravelTarget>(initial ?? targets[0])
    const [appKey, setAppKey] = useState('')
    return (
        <div className="max-w-md">
            <LaravelRestoreOptions targets={targets} target={target} onTarget={setTarget} appKey={appKey} onAppKey={setAppKey} />
        </div>
    )
}

const meta = {
    title: 'features/backups/LaravelRestoreOptions',
    component: Stateful,
    args: { targets: ['server', 'new'] },
} satisfies Meta<typeof Stateful>

export default meta
type Story = StoryObj<typeof meta>

export const ForTheServer: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByLabelText(/app_key/i)).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const ForASpace: Story = {
    args: { targets: ['space', 'new'] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByLabelText(/app_key/i)).toBeNull()
    },
}
export const OnlyNewSpace: Story = { args: { targets: ['new'] } }
export const AllTargets: Story = { args: { targets: ['server', 'space', 'new'] } }

export const ChoosingNewHidesTheKey: Story = {
    play: async ({ canvasElement }) => {
        const options = within(canvasElement).getAllByRole('radio')
        await userEvent.click(options[options.length - 1])
        await expect(within(canvasElement).queryByLabelText(/app_key/i)).toBeNull()
    },
}

export const TypingTheKey: Story = {
    play: async ({ canvasElement }) => {
        const key = within(canvasElement).getByLabelText(/app_key/i)
        await userEvent.type(key, 'base64:abc')
        await expect(key).toHaveValue('base64:abc')
    },
}

export const RecognisesALaravelDatabase: Story = {
    render: () => <p>{[isLaravelFile(new File([], 'x.SQLITE')), isLaravelFile(new File([], 'x.zip')), isLaravelFile(null)].join()}</p>,
    play: async ({ canvasElement }) => {
        await expect(canvasElement).toHaveTextContent('true,false,false')
    },
}
