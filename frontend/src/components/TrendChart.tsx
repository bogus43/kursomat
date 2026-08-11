import type { CurrencyHistoryEntry } from '../types'

interface TrendChartProps {
  currency: string
  entries: CurrencyHistoryEntry[]
}

const chartNumber = new Intl.NumberFormat('pl-PL', {
  minimumFractionDigits: 4,
  maximumFractionDigits: 4,
})

export function TrendChart({ currency, entries }: TrendChartProps) {
  if (entries.length < 2) return null

  const chronological = [...entries].reverse()
  const values = chronological.map((entry) => entry.mid)
  const minimum = Math.min(...values)
  const maximum = Math.max(...values)
  const spread = maximum - minimum || 1
  const points = chronological.map((entry, index) => {
    const x = chronological.length === 1 ? 50 : (index / (chronological.length - 1)) * 100
    const y = 90 - ((entry.mid - minimum) / spread) * 80
    return `${x.toFixed(2)},${y.toFixed(2)}`
  }).join(' ')

  return (
    <figure className="trend-chart" aria-label={`Trend kursu ${currency}`}>
      <figcaption>
        <span>Min. <strong>{chartNumber.format(minimum)}</strong></span>
        <span>Trend {currency}</span>
        <span>Maks. <strong>{chartNumber.format(maximum)}</strong></span>
      </figcaption>
      <svg viewBox="0 0 100 100" preserveAspectRatio="none" role="img">
        <title>Trend ostatnich {entries.length} notowań waluty {currency}</title>
        <line x1="0" y1="90" x2="100" y2="90" />
        <line x1="0" y1="50" x2="100" y2="50" />
        <line x1="0" y1="10" x2="100" y2="10" />
        <polyline points={points} />
      </svg>
    </figure>
  )
}
