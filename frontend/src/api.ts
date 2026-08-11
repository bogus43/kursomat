import {
  CancelImport,
  ChooseCachePath,
  ClearCache,
  Convert,
  ExportCurrencyHistory,
  GetCurrencies,
  GetCurrencyHistory,
  GetDashboard,
  ImportRates,
  SaveSettings,
} from '../wailsjs/go/main/DesktopApp'
import { EventsOn } from '../wailsjs/runtime/runtime'
import type {
  AppConfig,
  ConversionResult,
  ConvertRequest,
  Currency,
  CurrencyHistoryEntry,
  Dashboard,
  ImportProgress,
  ImportRequest,
  ImportSummary,
} from './types'

export const api = {
  dashboard: () => GetDashboard() as Promise<Dashboard>,
  currencies: () => GetCurrencies() as Promise<Currency[]>,
  convert: (request: ConvertRequest) => Convert(request) as Promise<ConversionResult>,
  importRates: (request: ImportRequest) => ImportRates(request) as Promise<ImportSummary>,
  cancelImport: () => CancelImport() as Promise<boolean>,
  history: (currency: string, limit = 120) =>
    GetCurrencyHistory(currency, limit) as Promise<CurrencyHistoryEntry[]>,
  exportHistory: (currency: string, format: 'csv' | 'json') =>
    ExportCurrencyHistory(currency, format) as Promise<string>,
  clearCache: () => ClearCache() as Promise<void>,
  saveSettings: (config: AppConfig) => SaveSettings(config) as Promise<Dashboard>,
  chooseCachePath: () => ChooseCachePath() as Promise<string>,
  onImportProgress: (callback: (progress: ImportProgress) => void) =>
    EventsOn('import:progress', callback),
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  return 'Wystąpił nieoczekiwany błąd'
}
