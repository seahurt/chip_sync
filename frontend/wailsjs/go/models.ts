export namespace config {
	
	export class Config {
	    rsync_path: string;
	    remote_host: string;
	    remote_port: number;
	    remote_module: string;
	    username: string;
	    password: string;
	    local_path: string;
	    sync_interval_seconds: number;
	    stable_hours: number;
	    log_path: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rsync_path = source["rsync_path"];
	        this.remote_host = source["remote_host"];
	        this.remote_port = source["remote_port"];
	        this.remote_module = source["remote_module"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.local_path = source["local_path"];
	        this.sync_interval_seconds = source["sync_interval_seconds"];
	        this.stable_hours = source["stable_hours"];
	        this.log_path = source["log_path"];
	    }
	}

}

export namespace main {
	
	export class ChipDirInfo {
	    name: string;
	    is_stable: boolean;
	    last_modified: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ChipDirInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.is_stable = source["is_stable"];
	        this.last_modified = source["last_modified"];
	        this.status = source["status"];
	    }
	}

}

