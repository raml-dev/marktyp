export namespace document {
	
	export class AppConfig {
	    version: number;
	    preferredMode: string;
	    lastOpenedPath: string;
	    autosave: boolean;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.preferredMode = source["preferredMode"];
	        this.lastOpenedPath = source["lastOpenedPath"];
	        this.autosave = source["autosave"];
	        this.theme = source["theme"];
	    }
	}
	export class AppInfo {
	    name: string;
	    tagline: string;
	    version: string;
	    modes: string[];
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.tagline = source["tagline"];
	        this.version = source["version"];
	        this.modes = source["modes"];
	    }
	}
	export class DocumentState {
	    id: string;
	    title: string;
	    path: string;
	    markdown: string;
	    draftMarkdown?: string;
	    hasDraft: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DocumentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.path = source["path"];
	        this.markdown = source["markdown"];
	        this.draftMarkdown = source["draftMarkdown"];
	        this.hasDraft = source["hasDraft"];
	    }
	}
	export class ExportHTMLRequest {
	    title: string;
	    html: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportHTMLRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.html = source["html"];
	    }
	}
	export class ExportPDFRequest {
	    title: string;
	    markdown: string;
	    html: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportPDFRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.markdown = source["markdown"];
	        this.html = source["html"];
	    }
	}
	export class NoteSummary {
	    id: string;
	    title: string;
	    path: string;
	    createdAt: string;
	    updatedAt: string;
	    updatedLabel: string;
	
	    static createFrom(source: any = {}) {
	        return new NoteSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.path = source["path"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.updatedLabel = source["updatedLabel"];
	    }
	}
	export class SaveDocumentRequest {
	    path: string;
	    title: string;
	    markdown: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveDocumentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.markdown = source["markdown"];
	    }
	}
	export class UpdatePreferencesRequest {
	    preferredMode: string;
	    autosave: boolean;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdatePreferencesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preferredMode = source["preferredMode"];
	        this.autosave = source["autosave"];
	        this.theme = source["theme"];
	    }
	}
	export class WorkspaceData {
	    appInfo: AppInfo;
	    config: AppConfig;
	    notes: NoteSummary[];
	    activeDoc: DocumentState;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appInfo = this.convertValues(source["appInfo"], AppInfo);
	        this.config = this.convertValues(source["config"], AppConfig);
	        this.notes = this.convertValues(source["notes"], NoteSummary);
	        this.activeDoc = this.convertValues(source["activeDoc"], DocumentState);
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

}

