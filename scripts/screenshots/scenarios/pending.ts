// Pages (their `path` in resources/ts/app/pages.ts) that no scenario goes through yet, each with
// the reason. The run fails for a page that is neither here nor in the `covers` of a scenario, and
// for one that is both. This list should only get shorter: a new page gets a scenario in
// flows/, not an entry here (unless it cannot have one; say why).
export const pending: Record<string, string> = {
    '/auth/sso/callback': 'todo: needs an identity provider',
    '/automation/:id/logs': 'todo: needs an automation to have run',
    '/admin/providers/:id/edit': 'todo: needs a provider to exist',
}
