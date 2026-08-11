export type ViewName = 'converter' | 'import' | 'database'
export type Direction = 'pln_to_foreign' | 'foreign_to_pln'
export type Theme = 'light' | 'dark'
export type Notice = { kind: 'success' | 'error' | 'info'; message: string } | null

export interface Currency {
  code: string
  name: string
}

export interface AppConfig {
  cache_path: string
  timeout_seconds: number
  retry_count: number
  max_lookback_days: number
  verbose: boolean
  last_from_date: string
  last_converter_date: string
}

export interface CacheInfo {
  path: string
  entries: number
  query_mappings: number
  currency_count: number
  last_saved_at: string
  size_bytes: number
}

export interface CurrencyStat {
  code: string
  name: string
  rate_count: number
  first_date: string
  last_date: string
}

export interface CurrencyHistoryEntry {
  effective_rate_date: string
  mid: number
  table_no: string
}

export interface Dashboard {
  config_path: string
  config: AppConfig
  cache: CacheInfo
  currencies: CurrencyStat[]
}

export interface RateResult {
  currency: string
  requested_date: string
  effective_rate_date: string
  mid: number
  table_no?: string
  source: string
}

export interface ConvertRequest {
  currency: string
  amount: number
  date: string
  direction: Direction
}

export interface ConversionResult {
  source_amount: number
  source_currency: string
  target_amount: number
  target_currency: string
  rate: RateResult
}

export interface ImportRequest {
  currencies: string[]
  start_date: string
  end_date: string
}

export interface ImportProgress {
  completed_chunks: number
  total_chunks: number
  currency: string
  chunk_start: string
  chunk_end: string
  rate_count: number
}

export interface ImportSummary {
  currency_count: number
  rate_count: number
  start_date: string
  end_date: string
  chunk_count: number
}
