export namespace editor {
	
	export class TranslationRow {
	    table: string;
	    id: string;
	    text: string;
	    original: string;
	
	    static createFrom(source: any = {}) {
	        return new TranslationRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.id = source["id"];
	        this.text = source["text"];
	        this.original = source["original"];
	    }
	}
	export class EditorData {
	    rows: TranslationRow[];
	    tempDir: string;
	
	    static createFrom(source: any = {}) {
	        return new EditorData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], TranslationRow);
	        this.tempDir = source["tempDir"];
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
	export class ToolingStatus {
	    ready: boolean;
	    bundleToolAvailable: boolean;
	    bundleToolPath: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolingStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.bundleToolAvailable = source["bundleToolAvailable"];
	        this.bundleToolPath = source["bundleToolPath"];
	        this.message = source["message"];
	    }
	}

}

export namespace main {
	
	export class AppStatus {
	    game: steam.GameInfo;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = this.convertValues(source["game"], steam.GameInfo);
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
	export class ApplyRequest {
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	    }
	}

}

export namespace patcher {
	
	export class ApplyResult {
	    targetPath: string;
	    backupPath: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetPath = source["targetPath"];
	        this.backupPath = source["backupPath"];
	        this.message = source["message"];
	    }
	}

}

export namespace steam {
	
	export class GameInfo {
	    installed: boolean;
	    path: string;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new GameInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}

}

