// Opens a page of the app by its address and waits until it has drawn itself.
import { expect, type Page } from 'playwright/test'

export async function visit(page: Page, path: string) {
    await page.goto(path)
    await expect(page).toHaveURL(new RegExp(`${path.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')}$`))
    await expect(page.locator('main').first()).toBeVisible()
    // The page is not found, or it failed to draw.
    await expect(page.getByText(/404|something went wrong/i)).toHaveCount(0)
}
