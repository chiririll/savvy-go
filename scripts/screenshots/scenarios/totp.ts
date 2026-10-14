// The one-time code of an authenticator app (RFC 6238: SHA-1, 6 digits, 30 s), for scenarios that
// set up or pass two-factor sign-in. The server accepts the code of the step before and after too.
import { createHmac } from 'node:crypto'

const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'

function base32(text: string): Buffer {
    let bits = ''
    for (const c of text.replace(/=+$/, '').toUpperCase()) bits += alphabet.indexOf(c).toString(2).padStart(5, '0')
    const bytes: number[] = []
    for (let i = 0; i + 8 <= bits.length; i += 8) bytes.push(parseInt(bits.slice(i, i + 8), 2))
    return Buffer.from(bytes)
}

export function totp(secret: string, at = Date.now()): string {
    const counter = Buffer.alloc(8)
    counter.writeBigUInt64BE(BigInt(Math.floor(at / 1000 / 30)))
    const sum = createHmac('sha1', base32(secret)).update(counter).digest()
    const offset = sum[sum.length - 1] & 0x0f
    const n = ((sum[offset] & 0x7f) << 24) | (sum[offset + 1] << 16) | (sum[offset + 2] << 8) | sum[offset + 3]
    return String(n % 1_000_000).padStart(6, '0')
}
