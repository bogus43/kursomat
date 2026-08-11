import { ArrowDownAZ, CalendarClock, Database, Download, RefreshCw, Search, Trash2, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { api, errorMessage } from '../api'
import { TrendChart } from '../components/TrendChart'
import type { CurrencyHistoryEntry, Dashboard, Notice } from '../types'

interface DatabaseViewProps {
  dashboard: Dashboard
  onNotice: (notice: Notice) => void
  onChanged: () => Promise<void>
}

type SortMode = 'code' | 'count' | 'date'
const number = new Intl.NumberFormat('pl-PL', { minimumFractionDigits: 4, maximumFractionDigits: 4 })
const formatDateTime = (value: string) => {
  if (!value) return 'Brak zapisu'
  const date = new Date(value)
  return Number.isNaN(date.getTime())
    ? value
    : date.toLocaleString('pl-PL', { dateStyle: 'medium', timeStyle: 'short' })
}

export function DatabaseView({ dashboard, onNotice, onChanged }: DatabaseViewProps) {
  const [filter, setFilter] = useState('')
  const [sort, setSort] = useState<SortMode>('code')
  const [selected, setSelected] = useState('')
  const [history, setHistory] = useState<CurrencyHistoryEntry[]>([])
  const [loadingHistory, setLoadingHistory] = useState(false)
  const [confirmClear, setConfirmClear] = useState(false)
  const [clearing, setClearing] = useState(false)
  const [exportFormat, setExportFormat] = useState<'csv' | 'json'>('csv')
  const [exporting, setExporting] = useState(false)

  const stats = useMemo(() => {
    const query = filter.trim().toLocaleLowerCase('pl')
    const result = dashboard.currencies.filter((item) =>
      item.rate_count > 0 && `${item.code} ${item.name}`.toLocaleLowerCase('pl').includes(query),
    )
    return result.sort((a, b) => {
      if (sort === 'count') return b.rate_count - a.rate_count || a.code.localeCompare(b.code)
      if (sort === 'date') return b.last_date.localeCompare(a.last_date) || a.code.localeCompare(b.code)
      return a.code.localeCompare(b.code)
    })
  }, [dashboard.currencies, filter, sort])

  useEffect(() => {
    if (!selected && stats.length > 0) setSelected(stats[0].code)
    if (selected && !stats.some((item) => item.code === selected)) setSelected(stats[0]?.code ?? '')
  }, [selected, stats])

  useEffect(() => {
    if (!selected) {
      setHistory([])
      return
    }
    let active = true
    setLoadingHistory(true)
    api.history(selected)
      .then((items) => active && setHistory(items))
      .catch((error) => active && onNotice({ kind: 'error', message: errorMessage(error) }))
      .finally(() => active && setLoadingHistory(false))
    return () => { active = false }
  }, [selected, dashboard.cache.entries, onNotice])

  const clear = async () => {
    setClearing(true)
    try {
      await api.clearCache()
      setSelected('')
      setHistory([])
      setConfirmClear(false)
      await onChanged()
      onNotice({ kind: 'success', message: 'Baza lokalna została wyczyszczona.' })
    } catch (error) {
      onNotice({ kind: 'error', message: errorMessage(error) })
    } finally {
      setClearing(false)
    }
  }

  const exportHistory = async () => {
    if (!selected) return
    setExporting(true)
    try {
      const path = await api.exportHistory(selected, exportFormat)
      if (path) onNotice({ kind: 'success', message: `Zapisano historię ${selected}: ${path}` })
    } catch (error) {
      onNotice({ kind: 'error', message: errorMessage(error) })
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="view database-view">
      <header className="view-header">
        <div><span className="eyebrow">Lokalne dane historyczne</span><h1>Baza</h1></div>
        <div className="header-actions">
          <div className="export-controls">
            <select aria-label="Format eksportu" value={exportFormat} onChange={(event) => setExportFormat(event.target.value as 'csv' | 'json')}>
              <option value="csv">CSV</option>
              <option value="json">JSON</option>
            </select>
            <button className="button secondary compact" type="button" disabled={!selected || history.length === 0 || exporting} onClick={exportHistory}>
              <Download size={17} /> {exporting ? 'Zapisywanie...' : 'Eksportuj'}
            </button>
          </div>
          <button className="icon-button bordered" type="button" aria-label="Odśwież bazę" title="Odśwież" onClick={onChanged}><RefreshCw size={18} /></button>
          <button className="button danger compact" type="button" disabled={dashboard.cache.entries === 0} onClick={() => setConfirmClear(true)}><Trash2 size={17} /> Wyczyść</button>
        </div>
      </header>

      <div className="metrics-band">
        <div><Database size={20} /><span><strong>{dashboard.cache.entries.toLocaleString('pl-PL')}</strong> kursów</span></div>
        <div><ArrowDownAZ size={20} /><span><strong>{dashboard.cache.currency_count}</strong> walut</span></div>
        <div><CalendarClock size={20} /><span><strong>{formatDateTime(dashboard.cache.last_saved_at)}</strong> ostatnia aktualizacja</span></div>
      </div>

      <div className="database-layout">
        <section className="currency-browser">
          <div className="database-toolbar">
            <label className="search-field"><Search size={17} /><input aria-label="Filtruj waluty w bazie" placeholder="Filtruj waluty" value={filter} onChange={(event) => setFilter(event.target.value)} /></label>
            <select aria-label="Sortuj waluty" value={sort} onChange={(event) => setSort(event.target.value as SortMode)}>
              <option value="code">Kod A—Z</option>
              <option value="count">Liczba kursów</option>
              <option value="date">Ostatnia data</option>
            </select>
          </div>
          <div className="currency-table" role="listbox" aria-label="Waluty w bazie">
            <div className="table-head"><span>Waluta</span><span>Kursy</span><span>Zakres</span></div>
            {stats.map((item) => (
              <button className={selected === item.code ? 'table-row selected' : 'table-row'} type="button" role="option" aria-selected={selected === item.code} key={item.code} onClick={() => setSelected(item.code)}>
                <span><strong>{item.code}</strong><small>{item.name}</small></span>
                <span>{item.rate_count}</span>
                <span><small>{item.first_date || '—'}</small><small>{item.last_date || '—'}</small></span>
              </button>
            ))}
            {stats.length === 0 && <div className="table-empty">Brak walut pasujących do filtra.</div>}
          </div>
        </section>

        <section className="history-browser">
          <div className="history-header">
            <div><span className="eyebrow">Historia kursów</span><h2>{selected || 'Wybierz walutę'}</h2></div>
            <span>Ostatnie 120 notowań</span>
          </div>
          {!loadingHistory && <TrendChart currency={selected} entries={history} />}
          <div className="history-table">
            <div className="history-row table-head"><span>Data</span><span>Kurs średni</span><span>Tabela</span></div>
            {loadingHistory ? <div className="table-empty"><RefreshCw className="spin" size={18} /> Ładowanie historii...</div> : history.map((entry) => (
              <div className="history-row" key={`${entry.effective_rate_date}-${entry.table_no}`}>
                <span>{entry.effective_rate_date}</span>
                <strong>{number.format(entry.mid)}</strong>
                <small>{entry.table_no || '—'}</small>
              </div>
            ))}
            {!loadingHistory && selected && history.length === 0 && <div className="table-empty">Brak zapisanych kursów dla tej waluty.</div>}
            {!selected && <div className="table-empty">Wybierz walutę z listy.</div>}
          </div>
        </section>
      </div>

      {confirmClear && (
        <div className="modal-backdrop" role="presentation">
          <section className="modal confirmation-dialog" role="alertdialog" aria-modal="true" aria-labelledby="clear-title">
            <header className="modal-header"><h2 id="clear-title">Wyczyścić bazę?</h2><button className="icon-button" type="button" aria-label="Anuluj" title="Zamknij" onClick={() => setConfirmClear(false)}><X size={19} /></button></header>
            <div className="modal-body"><p>Usunięte zostaną kursy, mapowania zapytań i zapisana lista walut. Tej operacji nie można cofnąć.</p></div>
            <footer className="modal-footer"><button className="button secondary" type="button" onClick={() => setConfirmClear(false)}>Anuluj</button><button className="button danger" type="button" disabled={clearing} onClick={clear}><Trash2 size={17} />{clearing ? 'Czyszczenie...' : 'Wyczyść bazę'}</button></footer>
          </section>
        </div>
      )}
    </div>
  )
}
