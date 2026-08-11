import { RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { api, errorMessage } from './api'
import { NoticeBar } from './components/NoticeBar'
import { SettingsDialog } from './components/SettingsDialog'
import { Sidebar } from './components/Sidebar'
import type { Currency, Dashboard, Notice, Theme, ViewName } from './types'
import { ConverterView } from './views/ConverterView'
import { DatabaseView } from './views/DatabaseView'
import { ImportView } from './views/ImportView'

export default function App() {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('kursomat-theme')
    if (saved === 'light' || saved === 'dark') return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })
  const [activeView, setActiveView] = useState<ViewName>('converter')
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
  const [currencies, setCurrencies] = useState<Currency[]>([])
  const [notice, setNotice] = useState<Notice>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [fatalError, setFatalError] = useState('')

  const refreshDashboard = useCallback(async () => {
    const data = await api.dashboard()
    setDashboard(data)
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setFatalError('')
    try {
      const data = await api.dashboard()
      setDashboard(data)
      try {
        setCurrencies(await api.currencies())
      } catch (error) {
        setNotice({ kind: 'error', message: errorMessage(error) })
      }
    } catch (error) {
      setFatalError(errorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])
  useEffect(() => {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('kursomat-theme', theme)
  }, [theme])

  if (loading) {
    return <main className="splash"><span className="brand-mark">K</span><RefreshCw className="spin" size={24} /><p>Uruchamianie Kursomatu...</p></main>
  }

  if (!dashboard || fatalError) {
    return (
      <main className="fatal-state">
        <span className="brand-mark">K</span>
        <h1>Nie udało się otworzyć aplikacji</h1>
        <p>{fatalError || 'Brak danych startowych.'}</p>
        <button className="button primary" type="button" onClick={load}><RefreshCw size={18} /> Spróbuj ponownie</button>
      </main>
    )
  }

  return (
    <div className="app-shell">
      <Sidebar
        active={activeView}
        theme={theme}
        onChange={setActiveView}
        onSettings={() => setSettingsOpen(true)}
        onThemeToggle={() => setTheme((current) => current === 'dark' ? 'light' : 'dark')}
      />
      <main className="main-content">
        <NoticeBar notice={notice} onClose={() => setNotice(null)} />
        {activeView === 'converter' && <ConverterView currencies={currencies} dashboard={dashboard} onNotice={setNotice} onChanged={refreshDashboard} />}
        {activeView === 'import' && <ImportView currencies={currencies} dashboard={dashboard} onNotice={setNotice} onChanged={refreshDashboard} />}
        {activeView === 'database' && <DatabaseView dashboard={dashboard} onNotice={setNotice} onChanged={refreshDashboard} />}
      </main>
      <SettingsDialog
        open={settingsOpen}
        config={dashboard.config}
        configPath={dashboard.config_path}
        onClose={() => setSettingsOpen(false)}
        onSaved={async () => {
          await refreshDashboard()
          setCurrencies(await api.currencies())
          setNotice({ kind: 'success', message: 'Ustawienia zostały zapisane.' })
        }}
        onError={(message) => setNotice({ kind: 'error', message })}
      />
    </div>
  )
}
