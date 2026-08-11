import { ArrowLeftRight, Database, Download, Moon, Settings, Sun } from 'lucide-react'
import type { Theme, ViewName } from '../types'

interface SidebarProps {
  active: ViewName
  theme: Theme
  onChange: (view: ViewName) => void
  onSettings: () => void
  onThemeToggle: () => void
}

const navigation = [
  { id: 'converter' as const, label: 'Konwerter', icon: ArrowLeftRight },
  { id: 'import' as const, label: 'Dane NBP', icon: Download },
  { id: 'database' as const, label: 'Baza', icon: Database },
]

export function Sidebar({ active, theme, onChange, onSettings, onThemeToggle }: SidebarProps) {
  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">K</span>
        <div>
          <strong>Kursomat</strong>
          <span>Kursy średnie NBP</span>
        </div>
      </div>
      <nav className="navigation" aria-label="Główna nawigacja">
        {navigation.map(({ id, label, icon: Icon }) => (
          <button
            className={active === id ? 'nav-item active' : 'nav-item'}
            key={id}
            type="button"
            aria-current={active === id ? 'page' : undefined}
            onClick={() => onChange(id)}
          >
            <Icon size={19} strokeWidth={1.8} />
            <span>{label}</span>
          </button>
        ))}
      </nav>
      <div className="sidebar-tools">
        <button className="theme-toggle" type="button" aria-label={theme === 'dark' ? 'Włącz tryb jasny' : 'Włącz tryb nocny'} title={theme === 'dark' ? 'Tryb jasny' : 'Tryb nocny'} onClick={onThemeToggle}>
          {theme === 'dark' ? <Sun size={19} strokeWidth={1.8} /> : <Moon size={19} strokeWidth={1.8} />}
        </button>
        <button className="nav-item settings-link" type="button" onClick={onSettings}>
          <Settings size={19} strokeWidth={1.8} />
          <span>Ustawienia</span>
        </button>
      </div>
    </aside>
  )
}
