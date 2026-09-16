import { CalendarRange, CheckSquare2, Download, Search, Square, XCircle } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { api, errorMessage } from '../api'
import { localDate } from '../conversion'
import type { Currency, Dashboard, ImportProgress, Notice } from '../types'

interface ImportViewProps {
  currencies: Currency[]
  dashboard: Dashboard
  onNotice: (notice: Notice) => void
  onChanged: () => Promise<void>
}

function monthAgo() {
  const date = new Date()
  const previous = new Date(date.getFullYear(), date.getMonth() - 1, 1)
  const lastDay = new Date(date.getFullYear(), date.getMonth(), 0).getDate()
  previous.setDate(Math.min(date.getDate(), lastDay))
  return localDate(previous)
}

export function ImportView({ currencies, dashboard, onNotice, onChanged }: ImportViewProps) {
  const today = localDate()
  const [startDate, setStartDate] = useState(dashboard.config.last_from_date || monthAgo())
  const [endDate, setEndDate] = useState(today)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [search, setSearch] = useState('')
  const [busy, setBusy] = useState(false)
  const [progress, setProgress] = useState<ImportProgress | null>(null)

  useEffect(() => {
    if (currencies.length > 0 && selected.size === 0) {
      setSelected(new Set(currencies.map((item) => item.code)))
    }
  }, [currencies, selected.size])

  useEffect(() => api.onImportProgress((update) => setProgress(update)), [])

  const visibleCurrencies = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('pl')
    if (!query) return currencies
    return currencies.filter((item) => `${item.code} ${item.name}`.toLocaleLowerCase('pl').includes(query))
  }, [currencies, search])

  const toggleCurrency = (code: string) => {
    setSelected((current) => {
      const next = new Set(current)
      if (next.has(code)) next.delete(code)
      else next.add(code)
      return next
    })
  }

  const toggleAll = () => {
    setSelected(selected.size === currencies.length ? new Set() : new Set(currencies.map((item) => item.code)))
  }

  const startImport = async () => {
    if (selected.size === 0) {
      onNotice({ kind: 'error', message: 'Wybierz co najmniej jedną walutę.' })
      return
    }
    if (!startDate || !endDate || endDate < startDate) {
      onNotice({ kind: 'error', message: 'Podaj prawidłowy zakres dat.' })
      return
    }
    setBusy(true)
    setProgress(null)
    try {
      const summary = await api.importRates({ currencies: [...selected], start_date: startDate, end_date: endDate })
      onNotice({ kind: 'success', message: `Zapisano ${summary.rate_count} kursów dla ${summary.currency_count} walut.` })
      await onChanged()
    } catch (error) {
      onNotice({ kind: 'error', message: errorMessage(error) })
    } finally {
      setBusy(false)
    }
  }

  const cancelImport = async () => {
    await api.cancelImport()
  }

  const percent = progress ? Math.round((progress.completed_chunks / progress.total_chunks) * 100) : 0

  return (
    <div className="view import-view">
      <header className="view-header">
        <div>
          <span className="eyebrow">Lokalny zbiór danych</span>
          <h1>Dane NBP</h1>
        </div>
        <div className="selection-count">Wybrano <strong>{selected.size}</strong> z {currencies.length}</div>
      </header>

      <div className="import-layout">
        <section className="work-panel import-parameters">
          <div className="section-heading">
            <div><span className="step-number">01</span><h2>Zakres importu</h2></div>
          </div>
          <div className="date-range">
            <label className="field"><span>Od</span><input type="date" max={endDate} value={startDate} onChange={(event) => setStartDate(event.target.value)} /></label>
            <CalendarRange size={20} aria-hidden="true" />
            <label className="field"><span>Do</span><input type="date" min={startDate} max={today} value={endDate} onChange={(event) => setEndDate(event.target.value)} /></label>
          </div>

          <div className="section-heading currency-heading">
            <div><span className="step-number">02</span><h2>Waluty</h2></div>
            <button className="text-button" type="button" onClick={toggleAll}>
              {selected.size === currencies.length ? <CheckSquare2 size={17} /> : <Square size={17} />}
              Wszystkie
            </button>
          </div>
          <label className="search-field">
            <Search size={17} />
            <input aria-label="Szukaj waluty" placeholder="Szukaj kodu lub nazwy" value={search} onChange={(event) => setSearch(event.target.value)} />
          </label>
          <div className="currency-checklist">
            {visibleCurrencies.map((item) => (
              <label className={selected.has(item.code) ? 'currency-option checked' : 'currency-option'} key={item.code}>
                <input type="checkbox" checked={selected.has(item.code)} onChange={() => toggleCurrency(item.code)} />
                <strong>{item.code}</strong>
                <span>{item.name}</span>
              </label>
            ))}
          </div>
        </section>

        <aside className="import-summary">
          <span className="eyebrow">Podsumowanie</span>
          <h2>{startDate} <span>—</span> {endDate}</h2>
          <dl>
            <div><dt>Waluty</dt><dd>{selected.size}</dd></div>
            <div><dt>Tryb</dt><dd>Średnie kursy, tabela A</dd></div>
            <div><dt>Miejsce zapisu</dt><dd>{dashboard.cache.path}</dd></div>
          </dl>

          {busy && (
            <div className="progress-block" aria-live="polite">
              <div className="progress-label">
                <span>{progress ? `${progress.currency}: ${progress.chunk_start} — ${progress.chunk_end}` : 'Przygotowywanie importu...'}</span>
                <strong>{percent}%</strong>
              </div>
              <div className="progress-track"><span style={{ width: `${percent}%` }} /></div>
              {progress && <small>{progress.rate_count} zapisanych kursów</small>}
            </div>
          )}

          {busy ? (
            <button className="button danger full-width" type="button" onClick={cancelImport}>
              <XCircle size={18} /> Anuluj import
            </button>
          ) : (
            <button className="button primary full-width" type="button" disabled={currencies.length === 0} onClick={startImport}>
              <Download size={18} /> Pobierz do bazy
            </button>
          )}
        </aside>
      </div>
    </div>
  )
}
