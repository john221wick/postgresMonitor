export namespace agentserver {
	
	export class ContainerInfo {
	    id: string;
	    name: string;
	    image: string;
	    status: string;
	    cpuPercent: number;
	    memUsedMB: number;
	    memLimitMB: number;
	
	    static createFrom(source: any = {}) {
	        return new ContainerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.image = source["image"];
	        this.status = source["status"];
	        this.cpuPercent = source["cpuPercent"];
	        this.memUsedMB = source["memUsedMB"];
	        this.memLimitMB = source["memLimitMB"];
	    }
	}
	export class ContainerReport {
	    available: boolean;
	    runtime: string;
	    error?: string;
	    containers: ContainerInfo[];
	
	    static createFrom(source: any = {}) {
	        return new ContainerReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.runtime = source["runtime"];
	        this.error = source["error"];
	        this.containers = this.convertValues(source["containers"], ContainerInfo);
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
	export class HostStats {
	    hostname: string;
	    osName: string;
	    kernel: string;
	    arch: string;
	    cpuModel: string;
	    uptimeSeconds: number;
	    cpuPercent: number;
	    cpuCores: number;
	    memTotalMB: number;
	    memUsedMB: number;
	    loadAvg: number[];
	    perCoreCPU: number[];
	
	    static createFrom(source: any = {}) {
	        return new HostStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.osName = source["osName"];
	        this.kernel = source["kernel"];
	        this.arch = source["arch"];
	        this.cpuModel = source["cpuModel"];
	        this.uptimeSeconds = source["uptimeSeconds"];
	        this.cpuPercent = source["cpuPercent"];
	        this.cpuCores = source["cpuCores"];
	        this.memTotalMB = source["memTotalMB"];
	        this.memUsedMB = source["memUsedMB"];
	        this.loadAvg = source["loadAvg"];
	        this.perCoreCPU = source["perCoreCPU"];
	    }
	}
	export class PgConnReq {
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	    db: string;
	    sslMode: string;
	
	    static createFrom(source: any = {}) {
	        return new PgConnReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.db = source["db"];
	        this.sslMode = source["sslMode"];
	    }
	}
	export class PgDeleteReq {
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	    db: string;
	    sslMode: string;
	    schema: string;
	    table: string;
	    ctid: string;
	
	    static createFrom(source: any = {}) {
	        return new PgDeleteReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.db = source["db"];
	        this.sslMode = source["sslMode"];
	        this.schema = source["schema"];
	        this.table = source["table"];
	        this.ctid = source["ctid"];
	    }
	}
	export class string {
	
	
	    static createFrom(source: any = {}) {
	        return new string(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PgInsertReq {
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	    db: string;
	    sslMode: string;
	    schema: string;
	    table: string;
	    values: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new PgInsertReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.db = source["db"];
	        this.sslMode = source["sslMode"];
	        this.schema = source["schema"];
	        this.table = source["table"];
	        this.values = source["values"];
	    }
	}
	export class PgPage {
	    columns: string[];
	    types: string[];
	    rows: string[][];
	    ctids: string[];
	    hasMore: boolean;
	    offset: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new PgPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.types = source["types"];
	        this.rows = source["rows"];
	        this.ctids = source["ctids"];
	        this.hasMore = source["hasMore"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	    }
	}
	export class PgRowsReq {
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	    db: string;
	    sslMode: string;
	    schema: string;
	    table: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new PgRowsReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.db = source["db"];
	        this.sslMode = source["sslMode"];
	        this.schema = source["schema"];
	        this.table = source["table"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class PgTable {
	    schema: string;
	    name: string;
	    rows: number;
	
	    static createFrom(source: any = {}) {
	        return new PgTable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	        this.rows = source["rows"];
	    }
	}
	export class PgUpdateReq {
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	    db: string;
	    sslMode: string;
	    schema: string;
	    table: string;
	    ctid: string;
	    column: string;
	    value?: string;
	
	    static createFrom(source: any = {}) {
	        return new PgUpdateReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.db = source["db"];
	        this.sslMode = source["sslMode"];
	        this.schema = source["schema"];
	        this.table = source["table"];
	        this.ctid = source["ctid"];
	        this.column = source["column"];
	        this.value = source["value"];
	    }
	}
	export class ProcInfo {
	    pid: number;
	    command: string;
	    cpuPercent: number;
	    memMB: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.command = source["command"];
	        this.cpuPercent = source["cpuPercent"];
	        this.memMB = source["memMB"];
	    }
	}

}

export namespace desktop {
	
	export class NodeInfo {
	    id: string;
	    name: string;
	    status: string;
	    localDir: string;
	    remoteDir: string;
	    arch: string;
	    os: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.localDir = source["localDir"];
	        this.remoteDir = source["remoteDir"];
	        this.arch = source["arch"];
	        this.os = source["os"];
	    }
	}
	export class NodeMonitorInfo {
	    nodeID: string;
	    nodeName: string;
	    reachable: boolean;
	    error?: string;
	    host: agentserver.HostStats;
	    containers: agentserver.ContainerReport;
	    processes: agentserver.ProcInfo[];
	    collectedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeMonitorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeID = source["nodeID"];
	        this.nodeName = source["nodeName"];
	        this.reachable = source["reachable"];
	        this.error = source["error"];
	        this.host = this.convertValues(source["host"], agentserver.HostStats);
	        this.containers = this.convertValues(source["containers"], agentserver.ContainerReport);
	        this.processes = this.convertValues(source["processes"], agentserver.ProcInfo);
	        this.collectedAt = source["collectedAt"];
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
	export class SavedNodeInfo {
	    id: string;
	    sshCommand: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedNodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sshCommand = source["sshCommand"];
	    }
	}

}

