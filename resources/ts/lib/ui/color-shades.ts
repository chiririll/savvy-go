/**
 * `count` shades of one colour for a parent's subcategories: the colour itself
 * first, then alternately lighter and darker steps, kept away from white and
 * black so each stays visible on either theme. A colour that is not hex is
 * repeated as it is.
 */
export function colorShades(color: string, count: number): string[] {
    const rgb = parseHex(color)
    if (!rgb) return Array.from({ length: count }, () => color)
    const [h, s, l] = toHsl(rgb)
    const out: string[] = []
    for (let i = 0; i < count; i++) {
        if (i === 0) {
            out.push(toHex(rgb))
            continue
        }
        // 1 → lighter, 2 → darker, 3 → lighter still, …
        const step = Math.ceil(i / 2) * 0.12 * (i % 2 === 1 ? 1 : -1)
        let next = l + step
        // Out of range on one side: take the same step on the other.
        if (next > MAX_L || next < MIN_L) next = l - step
        out.push(toHex(fromHsl(h, s, clamp(next, MIN_L, MAX_L))))
    }
    return out
}

const MIN_L = 0.25
const MAX_L = 0.8

type Rgb = [number, number, number]

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v))

function parseHex(color: string): Rgb | null {
    const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(color.trim())
    if (!m) return null
    const hex = m[1].length === 3 ? [...m[1]].map((c) => c + c).join('') : m[1]
    return [0, 2, 4].map((i) => parseInt(hex.slice(i, i + 2), 16)) as Rgb
}

const toHex = (rgb: Rgb) => '#' + rgb.map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')

function toHsl([r, g, b]: Rgb): Rgb {
    const [rr, gg, bb] = [r / 255, g / 255, b / 255]
    const max = Math.max(rr, gg, bb)
    const min = Math.min(rr, gg, bb)
    const l = (max + min) / 2
    if (max === min) return [0, 0, l]
    const d = max - min
    const s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    const h = max === rr ? (gg - bb) / d + (gg < bb ? 6 : 0) : max === gg ? (bb - rr) / d + 2 : (rr - gg) / d + 4
    return [h / 6, s, l]
}

function fromHsl(h: number, s: number, l: number): Rgb {
    if (s === 0) return [l * 255, l * 255, l * 255]
    const q = l < 0.5 ? l * (1 + s) : l + s - l * s
    const p = 2 * l - q
    const hue = (t: number) => {
        const x = t < 0 ? t + 1 : t > 1 ? t - 1 : t
        if (x < 1 / 6) return p + (q - p) * 6 * x
        if (x < 1 / 2) return q
        if (x < 2 / 3) return p + (q - p) * (2 / 3 - x) * 6
        return p
    }
    return [hue(h + 1 / 3) * 255, hue(h) * 255, hue(h - 1 / 3) * 255]
}
