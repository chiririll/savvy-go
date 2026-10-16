/** How the SMTP connection is secured; mirrors the constants of internal/mail/smtp. */
export const MAIL_PROTOCOLS = ['starttls', 'smtps', 'smtp'] as const
export type MailProtocol = (typeof MAIL_PROTOCOLS)[number]

/** Mail configuration of the instance, kept in the [mailer] section of config.toml. */
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
    /** Seconds to wait for the server. */
    timeout: number
    /** The password itself is never sent. */
    passwordSet: boolean
    /** Links in emails need the public URL. */
    appUrlSet: boolean
    /** False when config.toml is read-only: the values can be shown but not changed. */
    writable: boolean
}

/** What PUT /admin/mail takes: every setting. Without `password` the stored one is kept; '' clears it. */
export interface MailInput {
    enabled: boolean
    host: string
    port: number
    protocol: MailProtocol
    user: string
    password?: string
    from: string
    skipVerify: boolean
    timeout: number
}
