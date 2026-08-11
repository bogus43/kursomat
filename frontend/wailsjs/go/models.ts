export namespace appcore {
	
	export class ConversionResult {
	    source_amount: number;
	    source_currency: string;
	    target_amount: number;
	    target_currency: string;
	    rate: models.RateResult;
	
	    static createFrom(source: any = {}) {
	        return new ConversionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_amount = source["source_amount"];
	        this.source_currency = source["source_currency"];
	        this.target_amount = source["target_amount"];
	        this.target_currency = source["target_currency"];
	        this.rate = this.convertValues(source["rate"], models.RateResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConvertRequest {
	    currency: string;
	    amount: number;
	    date: string;
	    direction: string;
	
	    static createFrom(source: any = {}) {
	        return new ConvertRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currency = source["currency"];
	        this.amount = source["amount"];
	        this.date = source["date"];
	        this.direction = source["direction"];
	    }
	}
	export class Dashboard {
	    config_path: string;
	    config: models.AppConfig;
	    cache: cache.Info;
	    currencies: cache.CurrencyStat[];
	
	    static createFrom(source: any = {}) {
	        return new Dashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config_path = source["config_path"];
	        this.config = this.convertValues(source["config"], models.AppConfig);
	        this.cache = this.convertValues(source["cache"], cache.Info);
	        this.currencies = this.convertValues(source["currencies"], cache.CurrencyStat);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportRequest {
	    currencies: string[];
	    start_date: string;
	    end_date: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currencies = source["currencies"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	    }
	}
	export class ImportSummary {
	    currency_count: number;
	    rate_count: number;
	    start_date: string;
	    end_date: string;
	    chunk_count: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currency_count = source["currency_count"];
	        this.rate_count = source["rate_count"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.chunk_count = source["chunk_count"];
	    }
	}

}

export namespace cache {
	
	export class CurrencyHistoryEntry {
	    effective_rate_date: string;
	    mid: number;
	    table_no: string;
	
	    static createFrom(source: any = {}) {
	        return new CurrencyHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.effective_rate_date = source["effective_rate_date"];
	        this.mid = source["mid"];
	        this.table_no = source["table_no"];
	    }
	}
	export class CurrencyStat {
	    code: string;
	    name: string;
	    rate_count: number;
	    first_date: string;
	    last_date: string;
	
	    static createFrom(source: any = {}) {
	        return new CurrencyStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.rate_count = source["rate_count"];
	        this.first_date = source["first_date"];
	        this.last_date = source["last_date"];
	    }
	}
	export class Info {
	    path: string;
	    entries: number;
	    query_mappings: number;
	    currency_count: number;
	    last_saved_at: string;
	    size_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.entries = source["entries"];
	        this.query_mappings = source["query_mappings"];
	        this.currency_count = source["currency_count"];
	        this.last_saved_at = source["last_saved_at"];
	        this.size_bytes = source["size_bytes"];
	    }
	}

}

export namespace models {
	
	export class AppConfig {
	    cache_path: string;
	    timeout_seconds: number;
	    retry_count: number;
	    max_lookback_days: number;
	    verbose: boolean;
	    last_from_date: string;
	    last_converter_date: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cache_path = source["cache_path"];
	        this.timeout_seconds = source["timeout_seconds"];
	        this.retry_count = source["retry_count"];
	        this.max_lookback_days = source["max_lookback_days"];
	        this.verbose = source["verbose"];
	        this.last_from_date = source["last_from_date"];
	        this.last_converter_date = source["last_converter_date"];
	    }
	}
	export class Currency {
	    code: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Currency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	    }
	}
	export class RateResult {
	    currency: string;
	    requested_date: string;
	    effective_rate_date: string;
	    mid: number;
	    table_no?: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new RateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currency = source["currency"];
	        this.requested_date = source["requested_date"];
	        this.effective_rate_date = source["effective_rate_date"];
	        this.mid = source["mid"];
	        this.table_no = source["table_no"];
	        this.source = source["source"];
	    }
	}

}

