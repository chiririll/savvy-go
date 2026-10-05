// Test ids: stable marks on what a script needs to find in the interface, such as a button
// that opens a dialog (see scripts/screenshots). They exist only in a build made with
// APP_ENV=screenshots (vite.config.ts); in any other build the constant is false and
// nothing is rendered.
//
// testId(id) marks a button or menu item, in the attribute Playwright's getByTestId reads.
// An id starting with "layout-" belongs to the page frame (header, sidebar), not to one page.
// testIdMenu(id) marks the button of a menu whose items carry testId().
// testIdTabs() marks a group of choices that is not built from Radix tabs (those are found
// by their role), and testIdTab(id, active) each choice in it.

export const testId = (id: string): { 'data-testid'?: string } => (__TEST_IDS__ ? { 'data-testid': id } : {})

export const testIdMenu = (id: string): { 'data-testid-menu'?: string } =>
    __TEST_IDS__ ? { 'data-testid-menu': id } : {}

export const testIdTabs = (): { 'data-testid-tabs'?: string } => (__TEST_IDS__ ? { 'data-testid-tabs': '' } : {})

export const testIdTab = (id: string, active: boolean): { 'data-testid-tab'?: string; 'data-testid-active'?: string } =>
    __TEST_IDS__ ? { 'data-testid-tab': id, 'data-testid-active': String(active) } : {}

// Controls: a group of choices that changes what a page shows, such as the type of a chart or
// the period of a report. testIdControls(name) marks the group and testIdControl(id, active) each
// choice; testIdSelect(name) marks a select (a drop-down) with the same meaning. A name starting
// with "global-" is for a control that is on the whole page, not in one of its tabs.

export const testIdControls = (name: string): { 'data-testid-controls'?: string } =>
    __TEST_IDS__ ? { 'data-testid-controls': name } : {}

export const testIdControl = (id: string, active: boolean): { 'data-testid-control'?: string; 'data-testid-active'?: string } =>
    __TEST_IDS__ ? { 'data-testid-control': id, 'data-testid-active': String(active) } : {}

export const testIdSelect = (name: string): { 'data-testid-select'?: string } =>
    __TEST_IDS__ ? { 'data-testid-select': name } : {}
