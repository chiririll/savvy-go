import { useEffect, useRef } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { ssoApi } from '@/api/sso'
import { useAuthStore } from '@/stores/auth'
import i18n from '@/lib/i18n'
import { SSO_ERROR_CODES, ssoErrorLabel, type SsoErrorCode } from '@/lib/labels'

function ssoErrorMessage(code: string): string {
    if ((SSO_ERROR_CODES as readonly string[]).includes(code)) {
        return ssoErrorLabel(i18n.t, code as SsoErrorCode)
    }

    return i18n.t('auth:sso.errors.unknown', { code })
}

export default function SsoCallbackPage() {
    const [params] = useSearchParams()
    const navigate = useNavigate()
    const applySession = useAuthStore((state) => state.applySession)
    const ran = useRef(false)

    useEffect(() => {
        if (ran.current) return
        ran.current = true

        const error = params.get('error')
        const ticket = params.get('ticket')

        if (error || !ticket) {
            toast.error(error ? ssoErrorMessage(error) : i18n.t('auth:ssoFailed'))
            navigate('/login', { replace: true })
            return
        }

        ssoApi
            .exchange(ticket)
            .then((res) => {
                if ('requires_2fa' in res) {
                    navigate('/login', { replace: true, state: { twoFactorToken: res.two_factor_token, rememberMe: true } })
                    return
                }

                applySession(res)
                toast.success(i18n.t('auth:welcomeToast'))
                navigate('/', { replace: true })
            })
            .catch(() => {
                toast.error(i18n.t('auth:ssoFailed'))
                navigate('/login', { replace: true })
            })
    }, [params, navigate, applySession])

    return (
        <div className="min-h-screen flex items-center justify-center bg-background">
            <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
    )
}
