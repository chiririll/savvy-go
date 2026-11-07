import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import MarkdownContent from './MarkdownContent'

vi.mock('react-i18next', async (orig) => ({ ...(await orig<object>()), useTranslation: () => ({ t: (key: string) => key }) }))

const html = (md: string) => renderToStaticMarkup(<MarkdownContent>{md}</MarkdownContent>)

describe('MarkdownContent', () => {
    it('renders bold, italic, code and line breaks', () => {
        const out = html('**b** *i* `c`\nnext')
        expect(out).toContain('<strong>b</strong>')
        expect(out).toContain('<em>i</em>')
        expect(out).toContain('c</code>')
        expect(out).toContain('next')
    })

    it('keeps a new line of the editor as a new line, in paragraphs and in list items', () => {
        expect(html('one\ntwo')).toMatch(/one<br\/>\s*two/)
        expect(html('- one\n  two')).toMatch(/one<br\/>\s*two/)
    })

    it('renders bullet and numbered lists', () => {
        const out = html('- milk\n- bread\n\n1. one\n2. two')
        expect(out).toContain('<ul')
        expect(out.match(/<li/g)).toHaveLength(4)
        expect(out).toContain('<ol')
    })

    it('renders task lists as disabled checkboxes', () => {
        const out = html('- [ ] eggs\n- [x] milk')
        expect(out.match(/type="checkbox"/g)).toHaveLength(2)
        expect(out.match(/disabled/g)).toHaveLength(2)
        expect(out).toMatch(/checked/)
    })

    it('opens links safely', () => {
        const out = html('[site](https://example.com)')
        expect(out).toContain('href="https://example.com"')
        expect(out).toContain('rel="noopener noreferrer"')
        expect(out).toContain('target="_blank"')
    })

    it('drops javascript: links but keeps their text', () => {
        const out = html('[click](javascript:alert(1))')
        expect(out).not.toContain('javascript')
        expect(out).not.toContain('<a')
        expect(out).toContain('click')
    })

    it('never renders raw HTML as elements (it shows as escaped text)', () => {
        const out = html('hi <script>alert(1)</script> <b onclick="x()">bold</b>')
        expect(out).not.toContain('<script')
        expect(out).not.toContain('<b')
        expect(out).toContain('&lt;script&gt;')
    })

    it('drops images', () => {
        const out = html('![alt](https://example.com/a.png)')
        expect(out).not.toContain('<img')
    })

    it('shows headings as plain text', () => {
        const out = html('# Title')
        expect(out).not.toMatch(/<h1/)
        expect(out).toContain('Title')
    })
})
