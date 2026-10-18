import { InputOTP, InputOTPGroup, InputOTPSeparator, InputOTPSlot } from '@/components/ui/input-otp'
import { cn } from '@/lib/utils'

interface OtpCodeInputProps {
    value: string
    onChange: (value: string) => void
    onComplete?: () => void
    className?: string
}

/** The six-digit code field of two-factor sign-in and setup, in two groups of three. */
export function OtpCodeInput({ value, onChange, onComplete, className }: OtpCodeInputProps) {
    return (
        <div className={cn('flex justify-center', className)}>
            <InputOTP maxLength={6} value={value} onChange={onChange} onComplete={onComplete}>
                <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                </InputOTPGroup>
                <InputOTPSeparator />
                <InputOTPGroup>
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                </InputOTPGroup>
            </InputOTP>
        </div>
    )
}
