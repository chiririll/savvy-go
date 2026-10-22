// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { TextRow } from './SettingRows'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))

const origin = window.location.origin
const save = vi.fn()

afterEach(() => {
    cleanup()
    save.mockReset()
})

function renderRow(props: { value?: string; readOnly?: boolean; suggestion?: string } = {}) {
    return render(
        <TextRow id="app-url" label="Public URL" description="d" value="" onSave={save} suggestion={origin} {...props} />,
    )
}

const input = () => screen.getByLabelText('Public URL') as HTMLInputElement
const useButton = () => screen.queryByRole('button', { name: 'system.server.useBrowserUrl' })

describe('TextRow suggestion', () => {
    it('fills the field from the suggestion on click and waits for Save', () => {
        renderRow()
        fireEvent.click(useButton()!)
        expect(input().value).toBe(origin)
        expect(save).not.toHaveBeenCalled()
        fireEvent.click(screen.getByRole('button', { name: 'actions.save' }))
        expect(save).toHaveBeenCalledWith(origin)
    })

    it('is hidden when the config is read-only', () => {
        renderRow({ readOnly: true })
        expect(useButton()).toBeNull()
    })

    it('is hidden once the field already holds the suggestion', () => {
        renderRow({ value: origin })
        expect(useButton()).toBeNull()
    })

    it('is hidden after the draft is set to the suggestion', () => {
        renderRow()
        fireEvent.click(useButton()!)
        expect(useButton()).toBeNull()
    })

    it('is not offered without a suggestion', () => {
        renderRow({ suggestion: undefined })
        expect(useButton()).toBeNull()
    })
})
