export namespace main {
	
	export class Novel {
	    id: string;
	    title: string;
	    catalogUrl: string;
	    ruleId: string;
	
	    static createFrom(source: any = {}) {
	        return new Novel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.catalogUrl = source["catalogUrl"];
	        this.ruleId = source["ruleId"];
	    }
	}
	export class RegexReplacementRule {
	    pattern: string;
	    replace: string;
	    removeLine: boolean;
	    replaceFirst: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RegexReplacementRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.replace = source["replace"];
	        this.removeLine = source["removeLine"];
	        this.replaceFirst = source["replaceFirst"];
	        this.enabled = source["enabled"];
	    }
	}
	export class TextReplacementRule {
	    match: string;
	    replace: string;
	    caseSensitive: boolean;
	    replaceFirst: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TextReplacementRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.match = source["match"];
	        this.replace = source["replace"];
	        this.caseSensitive = source["caseSensitive"];
	        this.replaceFirst = source["replaceFirst"];
	        this.enabled = source["enabled"];
	    }
	}
	export class SiteRule {
	    id: string;
	    name: string;
	    matchDomains: string[];
	    catalogSectionHeadingText: string;
	    catalogSectionContainer: string;
	    catalogChapterLinkSelector: string;
	    chapterTitleSelector: string;
	    chapterContentSelector: string;
	    nextPageSelector: string;
	    nextChapterSelector: string;
	    contentCleanupSelectors: string[];
	    contentStopTexts: string[];
	    removeMatchingLines: string[];
	    textReplacementRules: TextReplacementRule[];
	    regexReplacementRules: RegexReplacementRule[];
	    skipChapterTitlePatterns: string[];
	    requestHeaders: Record<string, string>;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new SiteRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.matchDomains = source["matchDomains"];
	        this.catalogSectionHeadingText = source["catalogSectionHeadingText"];
	        this.catalogSectionContainer = source["catalogSectionContainer"];
	        this.catalogChapterLinkSelector = source["catalogChapterLinkSelector"];
	        this.chapterTitleSelector = source["chapterTitleSelector"];
	        this.chapterContentSelector = source["chapterContentSelector"];
	        this.nextPageSelector = source["nextPageSelector"];
	        this.nextChapterSelector = source["nextChapterSelector"];
	        this.contentCleanupSelectors = source["contentCleanupSelectors"];
	        this.contentStopTexts = source["contentStopTexts"];
	        this.removeMatchingLines = source["removeMatchingLines"];
	        this.textReplacementRules = this.convertValues(source["textReplacementRules"], TextReplacementRule);
	        this.regexReplacementRules = this.convertValues(source["regexReplacementRules"], RegexReplacementRule);
	        this.skipChapterTitlePatterns = source["skipChapterTitlePatterns"];
	        this.requestHeaders = source["requestHeaders"];
	        this.notes = source["notes"];
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
	export class AppState {
	    rules: SiteRule[];
	    novels: Novel[];
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rules = this.convertValues(source["rules"], SiteRule);
	        this.novels = this.convertValues(source["novels"], Novel);
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
	export class CacheEntry {
	    ruleId: string;
	    ruleName: string;
	    fileCount: number;
	    totalBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new CacheEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleId = source["ruleId"];
	        this.ruleName = source["ruleName"];
	        this.fileCount = source["fileCount"];
	        this.totalBytes = source["totalBytes"];
	    }
	}
	export class CatalogChapter {
	    title: string;
	    url: string;
	    cached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CatalogChapter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.cached = source["cached"];
	    }
	}
	export class CatalogAnalysis {
	    ruleId: string;
	    ruleName: string;
	    novelTitle: string;
	    chapterCount: number;
	    chapters: CatalogChapter[];
	
	    static createFrom(source: any = {}) {
	        return new CatalogAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleId = source["ruleId"];
	        this.ruleName = source["ruleName"];
	        this.novelTitle = source["novelTitle"];
	        this.chapterCount = source["chapterCount"];
	        this.chapters = this.convertValues(source["chapters"], CatalogChapter);
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
	
	export class CatalogRequest {
	    catalogUrl: string;
	    ruleId: string;
	    novelId: string;
	
	    static createFrom(source: any = {}) {
	        return new CatalogRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalogUrl = source["catalogUrl"];
	        this.ruleId = source["ruleId"];
	        this.novelId = source["novelId"];
	    }
	}
	export class ChapterReadRequest {
	    catalogUrl: string;
	    ruleId: string;
	    novelId: string;
	    chapterUrl: string;
	    chapterTitle: string;
	
	    static createFrom(source: any = {}) {
	        return new ChapterReadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalogUrl = source["catalogUrl"];
	        this.ruleId = source["ruleId"];
	        this.novelId = source["novelId"];
	        this.chapterUrl = source["chapterUrl"];
	        this.chapterTitle = source["chapterTitle"];
	    }
	}
	export class ChapterReadResult {
	    ruleId: string;
	    novelTitle: string;
	    chapterTitle: string;
	    chapterUrl: string;
	    content: string;
	    cached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChapterReadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleId = source["ruleId"];
	        this.novelTitle = source["novelTitle"];
	        this.chapterTitle = source["chapterTitle"];
	        this.chapterUrl = source["chapterUrl"];
	        this.content = source["content"];
	        this.cached = source["cached"];
	    }
	}
	export class ExportFailure {
	    index: number;
	    title: string;
	    url: string;
	    error: string;
	    retries: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.error = source["error"];
	        this.retries = source["retries"];
	    }
	}
	export class ExportRequest {
	    catalogUrl: string;
	    ruleId: string;
	    novelTitle: string;
	    novelId: string;
	    selectedChapterUrls: string[];
	    startChapter: number;
	    endChapter: number;
	    maxChapters: number;
	    retryCount: number;
	    skipOnFailure: boolean;
	    skipFilteredTitle: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalogUrl = source["catalogUrl"];
	        this.ruleId = source["ruleId"];
	        this.novelTitle = source["novelTitle"];
	        this.novelId = source["novelId"];
	        this.selectedChapterUrls = source["selectedChapterUrls"];
	        this.startChapter = source["startChapter"];
	        this.endChapter = source["endChapter"];
	        this.maxChapters = source["maxChapters"];
	        this.retryCount = source["retryCount"];
	        this.skipOnFailure = source["skipOnFailure"];
	        this.skipFilteredTitle = source["skipFilteredTitle"];
	    }
	}
	export class ExportResult {
	    filePath: string;
	    ruleId: string;
	    novelTitle: string;
	    exportedCount: number;
	    failureCount: number;
	    failures: ExportFailure[];
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.ruleId = source["ruleId"];
	        this.novelTitle = source["novelTitle"];
	        this.exportedCount = source["exportedCount"];
	        this.failureCount = source["failureCount"];
	        this.failures = this.convertValues(source["failures"], ExportFailure);
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
	
	export class NovelCacheEntry {
	    novelId: string;
	    novelTitle: string;
	    ruleId: string;
	    ruleName: string;
	    fileCount: number;
	    totalBytes: number;
	    cachedTitles?: string[];
	
	    static createFrom(source: any = {}) {
	        return new NovelCacheEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.novelId = source["novelId"];
	        this.novelTitle = source["novelTitle"];
	        this.ruleId = source["ruleId"];
	        this.ruleName = source["ruleName"];
	        this.fileCount = source["fileCount"];
	        this.totalBytes = source["totalBytes"];
	        this.cachedTitles = source["cachedTitles"];
	    }
	}
	
	

}

