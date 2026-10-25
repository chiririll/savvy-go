// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { NameStep } from './NameStep'

vi.mock('@/components/shared/pickers/EmojiPicker', () => ({ EmojiPicker: () => <button type="button">emoji</button> }))

afterEach(() => cleanup())

function renderStep(props: { name: string; emoji?: string; color?: string }) {
    render(
        <NameStep
            name={props.name}
            emoji={props.emoji ?? ''}
            color={props.color ?? ''}
            onNameChange={vi.fn()}
            onLookChange={vi.fn()}
            onSubmit={vi.fn()}
        />,
    )
}

describe('NameStep preview', () => {
    it('shows the first letter of the name when there is no emoji', () => {
        renderStep({ name: 'trip home' })
        expect(screen.getByText('T')).toBeTruthy()
    })

    it('shows the question mark while the name is empty', () => {
        renderStep({ name: '  ' })
        expect(screen.getByText('?')).toBeTruthy()
    })

    it('prefers the emoji over the letter', () => {
        renderStep({ name: 'Trip', emoji: '🏖️' })
        expect(screen.getByText('🏖️')).toBeTruthy()
        expect(screen.queryByText('T')).toBeNull()
    })
})
