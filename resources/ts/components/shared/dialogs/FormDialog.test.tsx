// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { FormDialog, FormDialogFooterStart } from './FormDialog'

vi.mock('react-i18next', async (orig) => ({ ...(await orig<object>()), useTranslation: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/providers/ReadOnlyProvider', () => ({ useReadOnly: () => false }))
vi.mock('@/hooks/use-mobile', () => ({ useIsMobile: () => false }))

afterEach(cleanup)

describe('FormDialog footer', () => {
    it('keeps Cancel and Save together, apart from the footer-start slot', async () => {
        render(
            <FormDialog open onOpenChange={() => {}} title="T" formId="f">
                <FormDialogFooterStart>
                    <span>Active</span>
                    <span>Estimated</span>
                </FormDialogFooterStart>
            </FormDialog>,
        )

        const cancel = await screen.findByText('actions.cancel')
        const save = screen.getByText('actions.create')
        expect(cancel.parentElement).toBe(save.parentElement)
        expect(cancel.parentElement?.className).toContain('shrink-0')
        expect(cancel.parentElement).not.toBe(screen.getByText('Active').parentElement)
    })
})
