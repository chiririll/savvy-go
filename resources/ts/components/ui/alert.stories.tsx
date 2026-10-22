import type { Meta, StoryObj } from '@storybook/react-vite'
import { AlertCircle, Info } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from './alert'

const meta = {
    title: 'ui/Alert',
    component: Alert,
    args: {
        children: (
            <>
                <Info />
                <AlertTitle>Heads up</AlertTitle>
                <AlertDescription>Rates were last refreshed an hour ago.</AlertDescription>
            </>
        ),
    },
} satisfies Meta<typeof Alert>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Destructive: Story = {
    args: {
        variant: 'destructive',
        children: (
            <>
                <AlertCircle />
                <AlertTitle>Could not save</AlertTitle>
                <AlertDescription>The server refused the change.</AlertDescription>
            </>
        ),
    },
}

export const TitleOnly: Story = { args: { children: <AlertTitle>Saved</AlertTitle> } }
