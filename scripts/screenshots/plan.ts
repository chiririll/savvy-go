// Works out what to take: a job for each language and theme, user, space and viewport, with the
// pages the config asks for.
import path from 'node:path'
import { pages, type Page } from '../../resources/ts/app/pages.ts'
import { pageWanted, type Config } from './config.ts'
import type { Manifest, ManifestSpace } from './manifest.ts'
import { paramValue, routes, skip } from './routes.ts'
import { slug, type Job } from './shared.ts'

const pageName = (p: string) => slug(p.replace(/^\//, '').replace(/\//g, '-')) || 'dashboard'

/** The session (cookies) of a user, from signing them in. */
export type SignIn = (email: string, password: string) => Promise<Job['storageState']>

export async function plan(manifest: Manifest, config: Config, signIn: SignIn): Promise<Job[]> {
    // A page with a ":param" is captured only where the manifest has a value for it.
    const resolved = new Set<string>()
    const unresolved = new Set<string>()
    const fill = (page: Page, space?: ManifestSpace) => {
        if (skip.has(page.path)) return []
        let missing = false
        const p = page.path.replace(/:\w+/g, () => {
            const value = paramValue[page.path]?.(manifest, space)
            if (value === undefined) missing = true
            return String(value)
        })
        if (missing) {
            unresolved.add(page.path)
            return []
        }
        resolved.add(page.path)
        // The name leaves out the ids, which differ from run to run.
        return pageWanted(config, page.path) ? [{ path: p, name: pageName(page.path.replace(/\/:\w+/g, '')) }] : []
    }
    const pagesOf = (scope: Page['scope'], space?: ManifestSpace): Job['pages'] =>
        routes.filter((r) => r.scope === scope).flatMap((r) => fill(r, space))

    // Everything is taken once for each language and theme, in a directory of its own.
    const variants = config.languages.flatMap((language) => config.themes.map((theme) => ({ language, theme, name: `${language}-${theme}` })))

    const jobs: Job[] = []
    for (const v of variants) {
        for (const viewport of config.viewports) {
            const list = pagesOf('public')
            if (list.length) jobs.push({ label: `${v.name} public`, viewport, language: v.language, theme: v.theme, dir: path.join(v.name, 'public'), pages: list })
        }
    }

    for (const user of manifest.users) {
        if (config.users.length && !config.users.includes(user.key)) continue
        const storageState = await signIn(user.email, user.password)
        const mine = manifest.spaces
            .filter((s) => user.key in s.members)
            .filter((s) => !config.spaces.length || config.spaces.some((n) => n.toLowerCase() === s.name.toLowerCase()))
        for (const v of variants) {
            for (const viewport of config.viewports) {
                mine.forEach((sp, i) => {
                    const list = pagesOf('space', sp)
                    // On a phone the sidebar is a drawer behind a button, so it is not in any page's shot.
                    if (viewport === 'mobile' && config.sidebar) {
                        list.unshift({ path: pages.dashboard.path, name: 'sidebar', sidebar: true })
                    }
                    // Pages of the user and the server do not depend on the space: once is enough.
                    if (i === 0) {
                        list.push(...pagesOf('user'))
                        if (user.role === 'admin') list.push(...pagesOf('admin'))
                    }
                    if (!list.length) return
                    jobs.push({
                        label: `${v.name} ${user.key}/${slug(sp.name)}`, viewport, language: v.language, theme: v.theme,
                        storageState, spaceId: sp.id, pages: list,
                        dir: path.join(v.name, user.key, slug(sp.name)),
                    })
                })
            }
        }
    }
    // Some spaces have no value (no automation rules): only a page that has none anywhere is worth a warning.
    const never = [...unresolved].filter((p) => !resolved.has(p))
    if (never.length) console.warn(`Left out, no value for their parameters in the manifest: ${never.join(', ')}`)
    return jobs
}
