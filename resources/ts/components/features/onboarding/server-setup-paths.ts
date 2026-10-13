import { pages } from '@/app/pages'

const base = pages.serverSetup.path

/** The screens of the server setup: the required URL, then a hub of optional steps. */
export const serverSetupPaths = {
    url: base,
    options: `${base}/more`,
    mail: `${base}/mail`,
    sso: `${base}/sso`,
} as const
