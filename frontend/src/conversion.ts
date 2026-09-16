/** Decimal amounts accept either Polish comma or dot; no implicit JS coercion. */
export function parseAmount(raw: string): number | null {
  const text = raw.trim()
  if (!/^\d+(?:[.,]\d+)?$/.test(text)) return null
  const value = Number(text.replace(',', '.'))
  return Number.isFinite(value) ? value : null
}

export function localDate(date = new Date()): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
