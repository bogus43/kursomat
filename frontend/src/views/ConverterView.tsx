import { ArrowRight, CalendarDays, Copy, RefreshCw, Star } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { ClipboardSetText } from '../../wailsjs/runtime/runtime'
import { api, errorMessage } from '../api'
import type { ConversionResult, Currency, Dashboard, Direction, Notice } from '../types'

interface ConverterViewProps {
  currencies: Currency[]
  dashboard: Dashboard
  onNotice: (notice: Notice) => void
  onChanged: () => Promise<void>
}

const today = new Date().toISOString().slice(0, 10)
const number = new Intl.NumberFormat('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 4 })
const favoritesKey = 'kursomat-favorite-currencies'
const autoConvertKey = 'kursomat-auto-convert'

function readFavorites(): string[] {
  try {
    const saved = JSON.parse(localStorage.getItem(favoritesKey) ?? '[]')
    return Array.isArray(saved) ? saved.filter((item): item is string => typeof item === 'string') : []
  } catch {
    return []
  }
}

export function ConverterView({ currencies, dashboard, onNotice, onChanged }: ConverterViewProps) {
  const [currency, setCurrency] = useState('USD')
  const [amount, setAmount] = useState('100,00')
  const [date, setDate] = useState(dashboard.config.last_converter_date || today)
  const [direction, setDirection] = useState<Direction>('pln_to_foreign')
  const [result, setResult] = useState<ConversionResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [autoConvert, setAutoConvert] = useState(() => localStorage.getItem(autoConvertKey) !== 'false')
  const [favorites, setFavorites] = useState<string[]>(readFavorites)
  const requestSequence = useRef(0)

  const selectedCurrency = useMemo(
    () => currencies.find((item) => item.code === currency),
    [currencies, currency],
  )
  const sortedCurrencies = useMemo(() => [...currencies].sort((left, right) => {
    const leftFavorite = favorites.includes(left.code)
    const rightFavorite = favorites.includes(right.code)
    if (leftFavorite !== rightFavorite) return leftFavorite ? -1 : 1
    return left.code.localeCompare(right.code)
  }), [currencies, favorites])

  useEffect(() => {
    if (currencies.length === 0) return
    if (!currencies.some((item) => item.code === currency)) {
      setCurrency(currencies.find((item) => item.code === 'USD')?.code ?? currencies[0].code)
    }
  }, [currencies, currency])

  const runConversion = useCallback(async (notify: boolean) => {
    const parsedAmount = Number(amount.trim().replace(',', '.'))
    if (!Number.isFinite(parsedAmount) || parsedAmount < 0) {
      if (notify) onNotice({ kind: 'error', message: 'Podaj poprawną, nieujemną kwotę.' })
      return
    }
    if (!currency) {
      if (notify) onNotice({ kind: 'error', message: 'Wybierz walutę.' })
      return
    }
    const sequence = ++requestSequence.current
    setLoading(true)
    try {
      const converted = await api.convert({ currency, amount: parsedAmount, date, direction })
      if (sequence !== requestSequence.current) return
      setResult(converted)
      if (notify) onNotice({ kind: 'success', message: `Przeliczono według kursu z ${converted.rate.effective_rate_date}.` })
      if (converted.rate.source !== 'cache') await onChanged()
    } catch (error) {
      if (sequence === requestSequence.current) onNotice({ kind: 'error', message: errorMessage(error) })
    } finally {
      if (sequence === requestSequence.current) setLoading(false)
    }
  }, [amount, currency, date, direction, onChanged, onNotice])

  useEffect(() => {
    localStorage.setItem(autoConvertKey, String(autoConvert))
    requestSequence.current += 1
    setLoading(false)
    if (!autoConvert || currencies.length === 0) return
    const timer = window.setTimeout(() => void runConversion(false), 450)
    return () => window.clearTimeout(timer)
  }, [autoConvert, currencies.length, runConversion])

  const submit = (event: React.FormEvent) => {
    event.preventDefault()
    void runConversion(true)
  }

  const toggleFavorite = () => {
    if (!currency) return
    const updated = favorites.includes(currency)
      ? favorites.filter((code) => code !== currency)
      : [...favorites, currency]
    setFavorites(updated)
    localStorage.setItem(favoritesKey, JSON.stringify(updated))
  }

  const copyResult = async () => {
    if (!result) return
    try {
      const copied = await ClipboardSetText(`${number.format(result.target_amount)} ${result.target_currency}`)
      if (!copied) throw new Error('Nie udało się zapisać tekstu w schowku.')
      onNotice({ kind: 'success', message: 'Wynik skopiowano do schowka.' })
    } catch (error) {
      onNotice({ kind: 'error', message: errorMessage(error) })
    }
  }

  return (
    <div className="view converter-view">
      <header className="view-header">
        <div>
          <span className="eyebrow">Kalkulator walutowy</span>
          <h1>Konwerter</h1>
        </div>
        <div className="source-badge"><span /> Oficjalne dane NBP</div>
      </header>

      <div className="converter-grid">
        <form className="work-panel converter-form" onSubmit={submit}>
          <div className="section-heading">
            <div>
              <span className="step-number">01</span>
              <h2>Parametry przeliczenia</h2>
            </div>
          </div>

          <div className="segmented" aria-label="Kierunek przeliczenia">
            <button type="button" className={direction === 'pln_to_foreign' ? 'selected' : ''} onClick={() => setDirection('pln_to_foreign')}>
              PLN na walutę
            </button>
            <button type="button" className={direction === 'foreign_to_pln' ? 'selected' : ''} onClick={() => setDirection('foreign_to_pln')}>
              Waluta na PLN
            </button>
          </div>

          <label className="field amount-field">
            <span>Kwota źródłowa</span>
            <div className="input-with-suffix">
              <input inputMode="decimal" value={amount} onChange={(event) => setAmount(event.target.value)} autoFocus />
              <strong>{direction === 'pln_to_foreign' ? 'PLN' : currency}</strong>
            </div>
          </label>

          <label className="field">
            <span>Waluta</span>
            <div className="currency-picker">
              <select value={currency} disabled={currencies.length === 0} onChange={(event) => setCurrency(event.target.value)}>
                {sortedCurrencies.map((item) => <option key={item.code} value={item.code}>{favorites.includes(item.code) ? '★ ' : ''}{item.code} · {item.name}</option>)}
              </select>
              <button className={favorites.includes(currency) ? 'icon-button bordered favorite' : 'icon-button bordered'} type="button" aria-label={favorites.includes(currency) ? 'Usuń z ulubionych' : 'Dodaj do ulubionych'} title={favorites.includes(currency) ? 'Usuń z ulubionych' : 'Dodaj do ulubionych'} onClick={toggleFavorite}>
                <Star size={17} fill={favorites.includes(currency) ? 'currentColor' : 'none'} />
              </button>
            </div>
          </label>

          <label className="field">
            <span>Data kursu</span>
            <div className="input-icon">
              <CalendarDays size={17} />
              <input type="date" max={today} value={date} onChange={(event) => setDate(event.target.value)} />
            </div>
          </label>

          <label className="live-convert-toggle">
            <input type="checkbox" checked={autoConvert} onChange={(event) => setAutoConvert(event.target.checked)} />
            <span>Przeliczaj automatycznie</span>
          </label>

          <button className="button primary full-width" type="submit" disabled={loading || currencies.length === 0}>
            {loading ? <RefreshCw className="spin" size={18} /> : <ArrowRight size={18} />}
            {loading ? 'Pobieranie kursu...' : 'Przelicz kwotę'}
          </button>
        </form>

        <section className={result ? 'result-panel has-result' : 'result-panel'} aria-live="polite">
          <div className="section-heading">
            <div>
              <span className="step-number">02</span>
              <h2>Wynik</h2>
            </div>
            {result && <button className="icon-button bordered" type="button" aria-label="Kopiuj wynik" title="Kopiuj wynik" onClick={copyResult}><Copy size={17} /></button>}
          </div>
          {result ? (
            <>
              <div className="conversion-result">
                <div>
                  <span>Kwota źródłowa</span>
                  <strong>{number.format(result.source_amount)} <small>{result.source_currency}</small></strong>
                </div>
                <ArrowRight size={24} aria-hidden="true" />
                <div>
                  <span>Kwota po przeliczeniu</span>
                  <strong>{number.format(result.target_amount)} <small>{result.target_currency}</small></strong>
                </div>
              </div>
              <dl className="rate-details">
                <div><dt>Kurs średni</dt><dd>{number.format(result.rate.mid)} PLN</dd></div>
                <div><dt>Data żądana</dt><dd>{result.rate.requested_date}</dd></div>
                <div><dt>Data kursu</dt><dd>{result.rate.effective_rate_date}</dd></div>
                <div><dt>Tabela NBP</dt><dd>{result.rate.table_no || '—'}</dd></div>
                <div><dt>Źródło</dt><dd>{result.rate.source === 'cache' ? 'Lokalna baza' : result.rate.source}</dd></div>
                <div><dt>Waluta</dt><dd>{selectedCurrency?.name ?? result.rate.currency}</dd></div>
              </dl>
            </>
          ) : (
            <div className="empty-result">
              <ArrowRight size={28} />
              <p>Wynik pojawi się po wykonaniu przeliczenia.</p>
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
