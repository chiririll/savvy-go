// Renders resources/ts/components/shared/Logo.tsx into every static logo file:
// public/favicon.svg, the PNG/ICO icons in public/ and the README logos in docs/images/.
// Usage: npm run logos
import { writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import sharp from 'sharp'
import { Logo } from '../resources/ts/components/shared/Logo.tsx'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const out = (...parts: string[]) => path.join(root, ...parts)

const render = (color: string): string =>
    renderToStaticMarkup(createElement(Logo, { width: 100, height: 100, style: { color } }))

// Favicon SVG adapts to the browser theme, since `currentColor` is otherwise black.
const favicon = render('#111827').replace(
    /style="[^"]*"/,
    '',
).replace(
    '</svg>',
    '<style>@media (prefers-color-scheme: dark) { svg { color: #f9fafb; } }</style></svg>',
)

// README logo: rounded tile behind the mark.
const tile = (color: string, background: string): string =>
    render(color).replace('>', `><rect width="100" height="100" rx="20" fill="${background}"/>`)

// [file, size, background (null = transparent)]
const pngs: [string, number, string | null][] = [
    ['favicon-96x96.png', 96, null],
    ['web-app-manifest-192x192.png', 192, null],
    ['web-app-manifest-512x512.png', 512, null],
    ['apple-touch-icon.png', 180, '#ffffff'],
]
const icoSizes = [16, 32, 48]

const raster = (size: number, background: string | null): Promise<Buffer> => {
    const image = sharp(Buffer.from(render('#000000')), { density: (72 * size) / 100 }).resize(size, size)
    return (background ? image.flatten({ background }) : image).png().toBuffer()
}

// ICO container holding PNG-encoded images.
function ico(images: { size: number; data: Buffer }[]): Buffer {
    const head = Buffer.alloc(6 + 16 * images.length)
    head.writeUInt16LE(1, 2)
    head.writeUInt16LE(images.length, 4)
    let offset = head.length
    images.forEach(({ size, data }, i) => {
        const entry = 6 + 16 * i
        head.writeUInt8(size, entry)
        head.writeUInt8(size, entry + 1)
        head.writeUInt16LE(1, entry + 4)
        head.writeUInt16LE(32, entry + 6)
        head.writeUInt32LE(data.length, entry + 8)
        head.writeUInt32LE(offset, entry + 12)
        offset += data.length
    })
    return Buffer.concat([head, ...images.map((image) => image.data)])
}

await writeFile(out('public/favicon.svg'), favicon)
await writeFile(out('docs/images/logo-light.svg'), tile('#1f2937', '#f3f4f6'))
await writeFile(out('docs/images/logo-dark.svg'), tile('#f9fafb', '#374151'))
for (const [name, size, bg] of pngs) await writeFile(out('public', name), await raster(size, bg))
await writeFile(
    out('public/favicon.ico'),
    ico(await Promise.all(icoSizes.map(async (size) => ({ size, data: await raster(size, null) })))),
)
console.log('logos updated')
