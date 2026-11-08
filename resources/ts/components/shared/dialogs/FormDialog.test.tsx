// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
    FormDialog,
    FormDialogActions,
    FormDialogCancel,
    FormDialogFooter,
    FormDialogFooterStart,
    FormDialogFooterStartSlot,
    FormDialogSubmit,
} from './FormDialog'

const env = vi.hoisted(() => ({ readOnly: false, mobile: false }))

vi.mock('react-i18next', async (orig) => ({ ...(await orig<object>()), useTranslation: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/providers/ReadOnlyProvider', () => ({ useReadOnly: () => env.readOnly }))
vi.mock('@/hooks/use-mobile', () => ({ useIsMobile: () => env.mobile }))

beforeEach(() => {
    env.readOnly = false
    env.mobile = false
})
afterEach(cleanup)

const open = (props: Partial<React.ComponentProps<typeof FormDialog>> = {}, body: React.ReactNode = null) =>
    render(
        <FormDialog open onOpenChange={() => {}} title="T" formId="f" {...props}>
            {body}
        </FormDialog>,
    )

describe('FormDialog footer', () => {
    it('keeps Cancel and Save together, apart from the footer-start slot', async () => {
        open(
            {},
            <FormDialogFooterStart>
                <span>Active</span>
                <span>Estimated</span>
            </FormDialogFooterStart>,
        )

        const cancel = await screen.findByText('actions.cancel')
        const save = screen.getByText('actions.create')
        expect(cancel.parentElement).toBe(save.parentElement)
        expect(cancel.parentElement?.className).toContain('shrink-0')
        expect(cancel.parentElement).not.toBe(screen.getByText('Active').parentElement)
    })

    it('hides Save for the read-only role', async () => {
        env.readOnly = true
        open()

        await screen.findByText('actions.cancel')
        expect(screen.queryByText('actions.create')).toBeNull()
    })

    it('footerEnd replaces only the right side', async () => {
        open({ footerEnd: <button>Extra</button> }, <FormDialogFooterStart>Active</FormDialogFooterStart>)

        await screen.findByText('Extra')
        expect(screen.queryByText('actions.cancel')).toBeNull()
        expect(screen.queryByText('actions.create')).toBeNull()
        expect(screen.getByText('Active')).toBeTruthy()
    })

    it('footerStart replaces only the left side', async () => {
        open({ footerStart: <span>Mine</span> }, <FormDialogFooterStart>Active</FormDialogFooterStart>)

        await screen.findByText('Mine')
        expect(screen.queryByText('Active')).toBeNull()
        expect(screen.getByText('actions.cancel')).toBeTruthy()
        expect(screen.getByText('actions.create')).toBeTruthy()
    })

    it('footer replaces the whole row', async () => {
        open({ footer: <div>Whole</div> }, <FormDialogFooterStart>Active</FormDialogFooterStart>)

        await screen.findByText('Whole')
        expect(screen.queryByText('actions.cancel')).toBeNull()
        expect(screen.queryByText('Active')).toBeNull()
    })

    it('gives render functions the dialog state', async () => {
        const seen: unknown[] = []
        open({
            formId: 'my-form',
            isEdit: true,
            isSubmitting: true,
            footerEnd: (ctx) => {
                seen.push(ctx)
                return <button onClick={ctx.close}>Dismiss</button>
            },
        })

        await screen.findByText('Dismiss')
        expect(seen[0]).toMatchObject({ formId: 'my-form', isEdit: true, isSubmitting: true, isReadOnly: false })
    })

    it('lets an override reuse Cancel and Submit next to its own button', async () => {
        open({
            footerEnd: (
                <>
                    <button>Delete</button>
                    <FormDialogCancel />
                    <FormDialogSubmit />
                </>
            ),
        })

        await screen.findByText('Delete')
        expect(screen.getByText('actions.cancel')).toBeTruthy()
        expect(screen.getByText('actions.create')).toBeTruthy()
    })

    it('lets a full footer be composed from the default parts', async () => {
        open({
            footer: (
                <FormDialogFooter>
                    <FormDialogFooterStartSlot />
                    <button>Before</button>
                    <FormDialogActions />
                </FormDialogFooter>
            ),
        })

        await screen.findByText('Before')
        expect(screen.getByText('actions.cancel')).toBeTruthy()
    })

    it('hides the reused Submit for the read-only role', async () => {
        env.readOnly = true
        open({ footerEnd: <FormDialogSubmit /> })

        await screen.findByRole('dialog')
        expect(screen.queryByText('actions.create')).toBeNull()
    })

    it('works the same in the mobile sheet', async () => {
        env.mobile = true
        open({ footerEnd: <button>Extra</button> }, <FormDialogFooterStart>Active</FormDialogFooterStart>)

        await screen.findByText('Extra')
        expect(screen.getByText('Active')).toBeTruthy()
        expect(screen.queryByText('actions.cancel')).toBeNull()
    })
})
