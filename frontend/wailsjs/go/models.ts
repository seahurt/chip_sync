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
	        this.log_path = source["log_path"];
	    }
	}

}

