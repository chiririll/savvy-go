export interface Settings {
    /** Per space: refresh this space's currency rates daily. */
    auto_update_currencies: boolean
    sso_allow_signup: boolean
    password_login_enabled: boolean
    sso_require_verified_email: boolean
    /** Size limit of a space in MB; 0 is unlimited. */
    space_quota_mb: number
    /** Backups a space keeps; 0 keeps all. */
    space_backups_max: number
    /** Spaces a user may administer; null is unlimited. */
    max_spaces_per_user: number | null
    /** Whether an invitation link may create an account. */
    space_invites_can_register: boolean
}
