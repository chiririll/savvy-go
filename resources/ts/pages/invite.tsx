import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { invitationsApi } from '@/api/spaces'
import { useAuthStore } from '@/stores/auth'
import { useSpaceStore } from '@/stores/space'
import { ApiError } from '@/types'
import { spaceRoleLabel } from '@/lib/labels'

/** Opened from an invitation link: join the space, or create an account for it. */
export default function InvitePage() {
    const { t } = useTranslation('settings')
    const { token = '' } = useParams<{ token: string }>()
    const navigate = useNavigate()
    const queryClient = useQueryClient()
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
    const authLoading = useAuthStore((state) => state.isLoading)
    const checkAuth = useAuthStore((state) => state.checkAuth)
    const setCurrentId = useSpaceStore((state) => state.setCurrentId)
    const [form, setForm] = useState({ name: '', email: '', password: '' })
    const [error, setError] = useState<string | null>(null)
    const [busy, setBusy] = useState(false)

    useEffect(() => {
        void checkAuth()
    }, [checkAuth])

    const { data: invitation, isPending, isError } = useQuery({
        queryKey: ['invitation', token],
        queryFn: () => invitationsApi.preview(token),
        retry: false,
    })

    useEffect(() => {
        if (invitation?.email) setForm((f) => ({ ...f, email: invitation.email ?? '' }))
    }, [invitation?.email])

    const run = async (action: () => Promise<void>) => {
        setBusy(true)
        setError(null)
        try {
            await action()
        } catch (e) {
            setError((e as ApiError).message)
        } finally {
            setBusy(false)
        }
    }

    const join = () => run(async () => {
        const space = await invitationsApi.accept(token)
        setCurrentId(space.id)
        await queryClient.invalidateQueries({ queryKey: ['spaces'] })
        navigate('/')
    })

    const register = (e: React.FormEvent) => {
        e.preventDefault()
        void run(async () => {
            await invitationsApi.register(token, form)
            await checkAuth()
            setCurrentId(null)
            navigate('/')
        })
    }

    return (
        <div className="flex min-h-screen items-center justify-center p-4">
            <Card className="w-full max-w-md">
                {isPending || authLoading ? (
                    <CardContent className="flex justify-center py-10">
                        <Loader2 className="size-6 animate-spin text-muted-foreground" />
                    </CardContent>
                ) : isError || !invitation ? (
                    <CardHeader>
                        <CardTitle>{t('invite.invalidTitle')}</CardTitle>
                        <CardDescription>{t('invite.invalidText')}</CardDescription>
                    </CardHeader>
                ) : (
                    <>
                        <CardHeader>
                            <CardTitle>{t('invite.title', { space: invitation.spaceName })}</CardTitle>
                            <CardDescription>
                                {t('invite.text', {
                                    inviter: invitation.inviter || t('invite.someone'),
                                    role: spaceRoleLabel(t, invitation.role),
                                })}
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {error && <p className="text-sm text-destructive">{error}</p>}
                            {isAuthenticated ? (
                                <Button className="w-full" disabled={busy} onClick={join}>
                                    {t('invite.join')}
                                </Button>
                            ) : (
                                <>
                                    {invitation.canRegister && (
                                        <form onSubmit={register} className="space-y-3">
                                            <div className="space-y-1.5">
                                                <Label htmlFor="name">{t('invite.name')}</Label>
                                                <Input id="name" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                                            </div>
                                            <div className="space-y-1.5">
                                                <Label htmlFor="email">{t('invite.email')}</Label>
                                                <Input id="email" type="email" required readOnly={!!invitation.email} value={form.email}
                                                    onChange={(e) => setForm({ ...form, email: e.target.value })} />
                                            </div>
                                            <div className="space-y-1.5">
                                                <Label htmlFor="password">{t('invite.password')}</Label>
                                                <Input id="password" type="password" minLength={8} required value={form.password}
                                                    onChange={(e) => setForm({ ...form, password: e.target.value })} />
                                            </div>
                                            <Button type="submit" className="w-full" disabled={busy}>{t('invite.register')}</Button>
                                        </form>
                                    )}
                                    <p className="text-center text-sm text-muted-foreground">
                                        {t('invite.haveAccount')}{' '}
                                        <Link to="/login" state={{ from: `/invite/${token}` }} className="font-medium text-foreground underline">
                                            {t('invite.signIn')}
                                        </Link>
                                    </p>
                                </>
                            )}
                        </CardContent>
                    </>
                )}
            </Card>
        </div>
    )
}
