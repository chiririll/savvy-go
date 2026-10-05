// Compares two screenshot runs and writes the changed screenshots, with the changes boxed in red,
// to screenshots/<old>_vs_<new>/. See docs/scripts.md.
//
//   npm run screenshots:diff -- <old> <new>     (run directory names; "latest" is the newest run)
import { mkdir, readdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', '..', 'screenshots')

/** A channel difference above this is a change; lower is font smoothing. */
const THRESHOLD = 24
/** Changed pixels are grouped in cells of this size, and cells this close are one box. */
const CELL = 8
const PAD = 4

interface Box {
    x: number
    y: number
    w: number
    h: number
}

const isRun = (name: string) => !name.includes('_vs_')

async function runs(): Promise<string[]> {
    const entries = await readdir(root, { withFileTypes: true }).catch(() => [])
    return entries
        .filter((e) => e.isDirectory() && isRun(e.name))
        .map((e) => e.name)
        .sort()
}

async function resolveRun(name: string): Promise<string> {
    const all = await runs()
    if (name === 'latest') {
        if (!all.length) throw new Error(`no runs in ${root}`)
        return all[all.length - 1]
    }
    if (!all.includes(name)) throw new Error(`no run "${name}"; there are ${all.join(', ') || 'none'}`)
    return name
}

async function pngs(dir: string): Promise<string[]> {
    const out: string[] = []
    const walk = async (rel: string) => {
        for (const e of await readdir(path.join(dir, rel), { withFileTypes: true })) {
            const next = path.join(rel, e.name)
            if (e.isDirectory()) await walk(next)
            else if (e.name.endsWith('.png')) out.push(next.split(path.sep).join('/'))
        }
    }
    await walk('')
    return out.sort()
}

const load = async (file: string) => {
    const { data, info } = await sharp(file).ensureAlpha().raw().toBuffer({ resolveWithObject: true })
    return { data, width: info.width, height: info.height }
}

/** Boxes around the changed areas of the part both images have, and how many pixels changed. */
function compare(a: Awaited<ReturnType<typeof load>>, b: Awaited<ReturnType<typeof load>>) {
    const w = Math.min(a.width, b.width)
    const h = Math.min(a.height, b.height)
    const cols = Math.ceil(w / CELL)
    const rows = Math.ceil(h / CELL)
    const cells = new Array<Box | null>(cols * rows).fill(null)
    let changed = 0
    for (let y = 0; y < h; y++) {
        for (let x = 0; x < w; x++) {
            const i = (y * a.width + x) * 4
            const j = (y * b.width + x) * 4
            const d = Math.max(
                Math.abs(a.data[i] - b.data[j]),
                Math.abs(a.data[i + 1] - b.data[j + 1]),
                Math.abs(a.data[i + 2] - b.data[j + 2]),
            )
            if (d <= THRESHOLD) continue
            changed++
            const c = Math.floor(y / CELL) * cols + Math.floor(x / CELL)
            const box = cells[c]
            if (!box) cells[c] = { x, y, w: 1, h: 1 }
            else {
                const x2 = Math.max(box.x + box.w, x + 1)
                const y2 = Math.max(box.y + box.h, y + 1)
                box.x = Math.min(box.x, x)
                box.y = Math.min(box.y, y)
                box.w = x2 - box.x
                box.h = y2 - box.y
            }
        }
    }

    // Cells next to each other (a gap of one empty cell included) are one box.
    const seen = new Array<boolean>(cells.length).fill(false)
    const boxes: Box[] = []
    for (let start = 0; start < cells.length; start++) {
        if (!cells[start] || seen[start]) continue
        let merged = { ...(cells[start] as Box) }
        const stack = [start]
        seen[start] = true
        while (stack.length) {
            const c = stack.pop() as number
            const cx = c % cols
            const cy = Math.floor(c / cols)
            for (let dy = -2; dy <= 2; dy++) {
                for (let dx = -2; dx <= 2; dx++) {
                    const nx = cx + dx
                    const ny = cy + dy
                    if (nx < 0 || ny < 0 || nx >= cols || ny >= rows) continue
                    const n = ny * cols + nx
                    const box = cells[n]
                    if (!box || seen[n]) continue
                    seen[n] = true
                    stack.push(n)
                    const x2 = Math.max(merged.x + merged.w, box.x + box.w)
                    const y2 = Math.max(merged.y + merged.h, box.y + box.h)
                    merged = { x: Math.min(merged.x, box.x), y: Math.min(merged.y, box.y), w: 0, h: 0 }
                    merged.w = x2 - merged.x
                    merged.h = y2 - merged.y
                }
            }
        }
        boxes.push(merged)
    }
    return { boxes, changed, compared: w * h }
}

async function main() {
    const [oldArg, newArg] = process.argv.slice(2)
    if (!oldArg || !newArg) throw new Error('usage: npm run screenshots:diff -- <old run> <new run>   ("latest" is the newest run)')
    const oldRun = await resolveRun(oldArg)
    const newRun = await resolveRun(newArg)
    if (oldRun === newRun) throw new Error('both are the same run')

    const day = (run: string) => run.slice(0, 10)
    if (day(oldRun) !== day(newRun)) {
        console.warn(`warning: the runs are from different days (${day(oldRun)}, ${day(newRun)}); dates, periods and amounts follow the day, so expect many changes`)
    }

    const oldDir = path.join(root, oldRun)
    const newDir = path.join(root, newRun)
    const outDir = path.join(root, `${oldRun}_vs_${newRun}`)
    const [oldFiles, newFiles] = await Promise.all([pngs(oldDir), pngs(newDir)])
    const oldSet = new Set(oldFiles)
    const newSet = new Set(newFiles)
    const added = newFiles.filter((f) => !oldSet.has(f))
    const removed = oldFiles.filter((f) => !newSet.has(f))
    const both = newFiles.filter((f) => oldSet.has(f))

    const lines: string[] = []
    let changedFiles = 0
    for (const file of both) {
        const [a, b] = await Promise.all([load(path.join(oldDir, file)), load(path.join(newDir, file))])
        const { boxes, changed, compared } = compare(a, b)
        const sizeChanged = a.width !== b.width || a.height !== b.height
        if (!boxes.length && !sizeChanged) continue
        changedFiles++

        // Whatever the new screenshot has beyond the old one's size is a change as well.
        const extra: Box[] = []
        if (b.height > a.height) extra.push({ x: 0, y: a.height, w: b.width, h: b.height - a.height })
        if (b.width > a.width) extra.push({ x: a.width, y: 0, w: b.width - a.width, h: Math.min(a.height, b.height) })

        const rects = [...boxes, ...extra]
            .map((r) => {
                const x = Math.max(0, r.x - PAD)
                const y = Math.max(0, r.y - PAD)
                const w = Math.min(b.width - x, r.w + PAD * 2)
                const h = Math.min(b.height - y, r.h + PAD * 2)
                return `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="rgba(255,0,0,0.08)" stroke="#ff0000" stroke-width="3"/>`
            })
            .join('')
        const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${b.width}" height="${b.height}">${rects}</svg>`

        const target = path.join(outDir, file)
        await mkdir(path.dirname(target), { recursive: true })
        await sharp(path.join(newDir, file)).composite([{ input: Buffer.from(svg) }]).png().toFile(target)

        const percent = ((changed / Math.max(1, compared)) * 100).toFixed(2)
        const size = sizeChanged ? `, size ${a.width}x${a.height} -> ${b.width}x${b.height}` : ''
        lines.push(`${file}: ${boxes.length} area${boxes.length === 1 ? '' : 's'}, ${percent}% of pixels${size}`)
    }

    const report = [
        `${oldRun} -> ${newRun}`,
        `${both.length} compared: ${changedFiles} changed, ${both.length - changedFiles} the same; ${added.length} only in the new run, ${removed.length} only in the old`,
        '',
        ...lines,
        ...(added.length ? ['', 'only in the new run:', ...added.map((f) => `  ${f}`)] : []),
        ...(removed.length ? ['', 'only in the old run:', ...removed.map((f) => `  ${f}`)] : []),
        '',
    ].join('\n')
    await mkdir(outDir, { recursive: true })
    await writeFile(path.join(outDir, 'report.txt'), report)

    console.log(report.split('\n').slice(0, 2).join('\n'))
    console.log(`written to ${outDir}`)
}

main().catch((e) => {
    console.error((e as Error).message)
    process.exit(1)
})
