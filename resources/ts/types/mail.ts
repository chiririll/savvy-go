/** How the SMTP connection is secured; mirrors the constants of internal/mailer. */
export const MAIL_PROTOCOLS = ['starttls', 'smtps', 'smtp'] as const
export type MailProtocol = (typeof MAIL_PROTOCOLS)[number]

/** Mail configuration of the instance. Read-only: it lives in the [mailer] section of config.toml. */
export interface MailStatus {
    /** Mail works: configured, valid and loaded. */
    enabled: boolean
    /** Why a configured [mailer] section was ignored; null when there is nothing wrong. */
    problem: string | null
    host: string
    port: number
    protocol: MailProtocol
    user: string
    from: string
    skipVerify: boolean
    /** The password itself is never sent. */
    passwordSet: boolean
    /** Links in emails need the public URL. */
    appUrlSet: boolean
}
