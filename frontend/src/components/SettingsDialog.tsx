import { FolderOpen, Save, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { api, errorMessage } from '../api'
import type { AppConfig } from '../types'

interface SettingsDialogProps {
  open: boolean
  config: AppConfig
  configPath: string
  onClose: () => void
  onSaved: () => Promise<void>
  onError: (message: string) => void
}

export function SettingsDialog({ open, config, configPath, onClose, onSaved, onError }: SettingsDialogProps) {
  const [draft, setDraft] = useState(config)
  const [saving, setSaving] = useState(false)

  useEffect(() => setDraft(config), [config, open])
  if (!open) return null

  const choosePath = async () => {
    try {
      const path = await api.chooseCachePath()
      if (path) setDraft((current) => ({ ...current, cache_path: path }))
    } catch (error) {
      onError(errorMessage(error))
    }
  }

  const save = async () => {
    setSaving(true)
    try {
      await api.saveSettings(draft)
      await onSaved()
      onClose()
    } catch (error) {
      onError(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className="modal settings-dialog" role="dialog" aria-modal="true" aria-labelledby="settings-title">
        <header className="modal-header">
          <div>
            <span className="eyebrow">Konfiguracja aplikacji</span>
            <h2 id="settings-title">Ustawienia</h2>
          </div>
          <button className="icon-button" type="button" aria-label="Zamknij ustawienia" title="Zamknij" onClick={onClose}>
            <X size={19} />
          </button>
        </header>
        <div className="modal-body settings-grid">
          <label className="field span-full">
            <span>Plik bazy SQLite</span>
            <div className="path-input">
              <input value={draft.cache_path} onChange={(event) => setDraft({ ...draft, cache_path: event.target.value })} />
              <button className="icon-button bordered" type="button" aria-label="Wybierz plik bazy" title="Wybierz plik" onClick={choosePath}>
                <FolderOpen size={18} />
              </button>
            </div>
          </label>
          <label className="field">
            <span>Limit czasu (s)</span>
            <input type="number" min="1" max="300" value={draft.timeout_seconds} onChange={(event) => setDraft({ ...draft, timeout_seconds: Number(event.target.value) })} />
          </label>
          <label className="field">
            <span>Liczba ponowień</span>
            <input type="number" min="0" max="10" value={draft.retry_count} onChange={(event) => setDraft({ ...draft, retry_count: Number(event.target.value) })} />
          </label>
          <label className="field">
            <span>Wyszukiwanie wstecz (dni)</span>
            <input type="number" min="1" max="3660" value={draft.max_lookback_days} onChange={(event) => setDraft({ ...draft, max_lookback_days: Number(event.target.value) })} />
          </label>
          <label className="toggle-field">
            <input type="checkbox" checked={draft.verbose} onChange={(event) => setDraft({ ...draft, verbose: event.target.checked })} />
            <span>
              <strong>Logowanie diagnostyczne</strong>
              <small>Zapisuj szczegóły komunikacji z NBP.</small>
            </span>
          </label>
          <div className="config-location span-full">
            <span>Plik ustawień</span>
            <code>{configPath}</code>
          </div>
        </div>
        <footer className="modal-footer">
          <button className="button secondary" type="button" onClick={onClose}>Anuluj</button>
          <button className="button primary" type="button" disabled={saving} onClick={save}>
            <Save size={17} />
            {saving ? 'Zapisywanie...' : 'Zapisz ustawienia'}
          </button>
        </footer>
      </section>
    </div>
  )
}
