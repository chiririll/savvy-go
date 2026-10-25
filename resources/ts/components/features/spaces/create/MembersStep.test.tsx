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

const add = () => fireEvent.click(screen.getByRole('button', { name: 'spaces.create.members.add' }))

afterEach(() => cleanup())

describe('MembersStep', () => {
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
