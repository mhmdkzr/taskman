// Minimal RFC 6238 TOTP code generator, so specs can act as the user's
// authenticator app during enrollment/login without a real device or a
// third-party TOTP library.
import { createHmac } from "node:crypto"

const BASE32_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

/** @param {string} base32 */
function base32Decode(base32) {
  let bits = ""
  for (const char of base32.toUpperCase().replace(/=+$/, "")) {
    const value = BASE32_ALPHABET.indexOf(char)
    if (value === -1) throw new Error(`invalid base32 character: ${char}`)
    bits += value.toString(2).padStart(5, "0")
  }
  const bytes = []
  for (let i = 0; i + 8 <= bits.length; i += 8) {
    bytes.push(parseInt(bits.slice(i, i + 8), 2))
  }
  return Buffer.from(bytes)
}

/**
 * Generates the current 6-digit TOTP code for a base32 secret, per RFC 6238
 * (HMAC-SHA1, 30-second step) — matching ZITADEL's TOTP enrollment.
 * @param {string} secret
 * @param {number} [atMs] time to generate the code for, in epoch millis (for testing)
 */
export function generateTOTP(secret, atMs) {
  const key = base32Decode(secret)
  const counter = Math.floor((atMs ?? Date.now()) / 1000 / 30)
  const counterBuffer = Buffer.alloc(8)
  counterBuffer.writeBigUInt64BE(BigInt(counter))
  const hmac = createHmac("sha1", key).update(counterBuffer).digest()
  const offset = hmac[hmac.length - 1] & 0x0f
  const binary =
    ((hmac[offset] & 0x7f) << 24) | ((hmac[offset + 1] & 0xff) << 16) | ((hmac[offset + 2] & 0xff) << 8) | (hmac[offset + 3] & 0xff)
  return String(binary % 1_000_000).padStart(6, "0")
}
