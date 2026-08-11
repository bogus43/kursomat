import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react'
import type { Notice } from '../types'

interface NoticeBarProps {
  notice: Notice
  onClose: () => void
}

export function NoticeBar({ notice, onClose }: NoticeBarProps) {
  if (!notice) return null
  const Icon = notice.kind === 'success' ? CheckCircle2 : notice.kind === 'error' ? AlertCircle : Info
  return (
    <div className={`notice ${notice.kind}`} role={notice.kind === 'error' ? 'alert' : 'status'}>
      <Icon size={18} />
      <span>{notice.message}</span>
      <button className="icon-button" type="button" aria-label="Zamknij komunikat" title="Zamknij" onClick={onClose}>
        <X size={17} />
      </button>
    </div>
  )
}
