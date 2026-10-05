// The pages to capture are the app's own (resources/ts/app/pages.ts), so a new page
// is picked up without touching this script. routes.test.ts requires that a page
// with a ":param" is either left out below or has a value for it.
import { pages, type Page } from '../../resources/ts/app/pages.ts'
import type { Manifest, ManifestSpace } from './manifest.ts'

export const routes: Page[] = Object.values(pages)

/** Pages that are not worth a screenshot: they only work mid-flow or redirect when signed in. */
export const skip = new Set<string>([
    pages.setup.path,
    pages.setup2fa.path,
    pages.ssoCallback.path,
    pages.setPassword.path,
    // The demo has no single sign-on providers to edit.
    pages.providerEdit.path,
])

/** The value of the ":param" of a page, from the manifest; undefined when there is none. */
export const paramValue: Record<string, (m: Manifest, space?: ManifestSpace) => string | number | undefined> = {
    [pages.invite.path]: (m) => m.invitations[0]?.token,
    [pages.automationLogs.path]: (_, space) => space?.automations?.[0],
}
