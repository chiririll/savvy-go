// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { InviteMethods } from '@/types/spaces'
import { MembersStep } from './MembersStep'
import type { DraftInvitation } from './draft'

let methods: InviteMethods | undefined

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/hooks/use-spaces', () => ({ useInviteMethods: () => ({ data: methods }) }))

/** Keeps the list in state, as the modal does with its draft. */
function Harness({ onList }: { onList: (list: DraftInvitation[]) => void }) {
    const [list, setList] = useState<DraftInvitation[]>([])
    return (
        <MembersStep
            invitations={list}
            onChange={(next) => {
                setList(next)
                onList(next)
            }}
        />
    )
}

const type = (value: string) =>
    fireEvent.change(screen.getByPlaceholderText('spaces.invitations.emailPlaceholder'), { target: { value } })
const add = () => fireEvent.click(screen.getByRole('button', { name: 'spaces.create.members.add' }))

afterEach(() => cleanup())

describe('MembersStep', () => {
    it('lists addresses without sending anything', () => {
        methods = { email: true, link: true }
        const onList = vi.fn()
        render(<Harness onList={onList} />)

        type('bob@example.org')
        add()

        expect(screen.getByText('bob@example.org')).toBeTruthy()
        expect(onList).toHaveBeenLastCalledWith([expect.objectContaining({ email: 'bob@example.org', role: 'editor' })])
    })

    it('refuses an address twice and one that is not an address', () => {
        methods = { email: true, link: true }
        render(<Harness onList={vi.fn()} />)

        type('bob@example.org')
        add()
        type('BOB@example.org')
        add()
        expect(screen.getByText('spaces.create.members.duplicate')).toBeTruthy()

        type('nope')
        add()
        expect(screen.getByText('spaces.create.members.invalid')).toBeTruthy()
        expect(screen.getAllByText('bob@example.org')).toHaveLength(1)
    })

    it('removes an entry before anything is sent', () => {
        methods = { email: true, link: true }
        const onList = vi.fn()
        render(<Harness onList={onList} />)
        type('bob@example.org')
        add()

        fireEvent.click(screen.getByRole('button', { name: 'spaces.create.members.remove' }))

        expect(screen.queryByText('bob@example.org')).toBeNull()
        expect(onList).toHaveBeenLastCalledWith([])
    })

    it('offers only links when the server cannot email', () => {
        methods = { email: false, link: true, emailReason: 'Mail is not configured.' }
        const onList = vi.fn()
        render(<Harness onList={onList} />)

        expect(screen.queryByPlaceholderText('spaces.invitations.emailPlaceholder')).toBeNull()
        add()
        expect(onList).toHaveBeenLastCalledWith([expect.objectContaining({ email: null })])
    })

    it('offers only addresses when the server requires verified ones', () => {
        methods = { email: true, link: false }
        render(<Harness onList={vi.fn()} />)

        expect(screen.getByPlaceholderText('spaces.invitations.emailPlaceholder')).toBeTruthy()
        // No switch between the methods: there is only one.
        expect(screen.queryByRole('button', { name: 'spaces.invitations.methodLink' })).toBeNull()
    })
})
