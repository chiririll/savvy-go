import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { invitationsApi } from '@/api/spaces'
import { useAuthStore } from '@/stores/auth'
import { useSpaceStore } from '@/stores/space'
import { ApiError } from '@/types'

export interface RegisterInput {
    name: string
    email: string
    password: string
}

/** Joining a space or registering through an invitation link, with the state both share. */
export function useInviteActions(token: string) {
    const navigate = useNavigate()
    const queryClient = useQueryClient()
    const checkAuth = useAuthStore((state) => state.checkAuth)
    const setCurrentId = useSpaceStore((state) => state.setCurrentId)
    const [error, setError] = useState<string | null>(null)
    const [busy, setBusy] = useState(false)
    const [alreadyMember, setAlreadyMember] = useState(false)

    const run = async (action: () => Promise<void>) => {
        setBusy(true)
        setError(null)
        try {
            await action()
        } catch (e) {
            // 409: the account is already in this space, so the link would only change its role.
            if ((e as ApiError).code === '409') setAlreadyMember(true)
            else setError((e as ApiError).message)
        } finally {
            setBusy(false)
        }
    }

    const join = () =>
        run(async () => {
            const space = await invitationsApi.accept(token)
            setCurrentId(space.id)
            await queryClient.invalidateQueries({ queryKey: ['spaces'] })
            navigate('/')
        })

    const register = (input: RegisterInput) =>
        run(async () => {
            await invitationsApi.register(token, input)
            await checkAuth()
            setCurrentId(null)
            navigate('/')
        })

    return { busy, error, alreadyMember, join, register }
}
