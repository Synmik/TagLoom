export namespace app {
	
	export class FileMetadata {
	    filename: string;
	    extension: string;
	    format_name: string;
	    mime_type: string;
	    size_bytes: number;
	    date_created: string;
	    date_modified: string;
	    resolution_width: number;
	    resolution_height: number;
	    duration_seconds: number;
	    dominant_colors: string[];
	
	    static createFrom(source: any = {}) {
	        return new FileMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.extension = source["extension"];
	        this.format_name = source["format_name"];
	        this.mime_type = source["mime_type"];
	        this.size_bytes = source["size_bytes"];
	        this.date_created = source["date_created"];
	        this.date_modified = source["date_modified"];
	        this.resolution_width = source["resolution_width"];
	        this.resolution_height = source["resolution_height"];
	        this.duration_seconds = source["duration_seconds"];
	        this.dominant_colors = source["dominant_colors"];
	    }
	}
	export class ThumbnailInfo {
	    file_id: number;
	    thumbnail_path: string;
	    exists: boolean;
	    size_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new ThumbnailInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_id = source["file_id"];
	        this.thumbnail_path = source["thumbnail_path"];
	        this.exists = source["exists"];
	        this.size_bytes = source["size_bytes"];
	    }
	}

}

export namespace config {
	
	export class Settings {
	    auto_tag_by_folder: boolean;
	    excluded_folders: string[];
	    thumbnail_size: number;
	    thumbnail_quality: number;
	    default_sort_field: string;
	    default_sort_order: string;
	    grid_thumbnail_size: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_tag_by_folder = source["auto_tag_by_folder"];
	        this.excluded_folders = source["excluded_folders"];
	        this.thumbnail_size = source["thumbnail_size"];
	        this.thumbnail_quality = source["thumbnail_quality"];
	        this.default_sort_field = source["default_sort_field"];
	        this.default_sort_order = source["default_sort_order"];
	        this.grid_thumbnail_size = source["grid_thumbnail_size"];
	    }
	}
	export class VaultConfig {
	    name: string;
	    version: number;
	    created_at: string;
	    settings: Settings;
	
	    static createFrom(source: any = {}) {
	        return new VaultConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.created_at = source["created_at"];
	        this.settings = this.convertValues(source["settings"], Settings);
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

export namespace db {
	
	export class File {
	    id: number;
	    vault_path: string;
	    thumbnail_path?: string;
	    name?: string;
	    notes?: string;
	    link?: string;
	    rating: number;
	    is_favorite: number;
	    folder_path: string;
	    indexed_at: string;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.vault_path = source["vault_path"];
	        this.thumbnail_path = source["thumbnail_path"];
	        this.name = source["name"];
	        this.notes = source["notes"];
	        this.link = source["link"];
	        this.rating = source["rating"];
	        this.is_favorite = source["is_favorite"];
	        this.folder_path = source["folder_path"];
	        this.indexed_at = source["indexed_at"];
	    }
	}
	export class FileFilter {
	    folder_path: string;
	    tag_ids: number[];
	    file_formats: string[];
	    min_rating: number;
	    favorites_only: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder_path = source["folder_path"];
	        this.tag_ids = source["tag_ids"];
	        this.file_formats = source["file_formats"];
	        this.min_rating = source["min_rating"];
	        this.favorites_only = source["favorites_only"];
	    }
	}
	export class FilePage {
	    files: File[];
	    total_count: number;
	    page: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new FilePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = this.convertValues(source["files"], File);
	        this.total_count = source["total_count"];
	        this.page = source["page"];
	        this.limit = source["limit"];
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
	export class FileUpdate {
	    id: number;
	    name: string;
	    notes: string;
	    link: string;
	    rating: number;
	    is_favorite: number;
	
	    static createFrom(source: any = {}) {
	        return new FileUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.notes = source["notes"];
	        this.link = source["link"];
	        this.rating = source["rating"];
	        this.is_favorite = source["is_favorite"];
	    }
	}
	export class FolderNode {
	    path: string;
	    name: string;
	    file_count: number;
	    children: FolderNode[];
	
	    static createFrom(source: any = {}) {
	        return new FolderNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.file_count = source["file_count"];
	        this.children = this.convertValues(source["children"], FolderNode);
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
	export class SortOpts {
	    field: string;
	    order: string;
	
	    static createFrom(source: any = {}) {
	        return new SortOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.order = source["order"];
	    }
	}
	export class Tag {
	    id: number;
	    name: string;
	    color: string;
	    parent_id?: number;
	    is_category: number;
	    sort_order: number;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Tag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.parent_id = source["parent_id"];
	        this.is_category = source["is_category"];
	        this.sort_order = source["sort_order"];
	        this.created_at = source["created_at"];
	    }
	}
	export class TagCreate {
	    name: string;
	    color: string;
	    parent_id?: number;
	    is_category: number;
	    sort_order: number;
	    aliases: string;
	
	    static createFrom(source: any = {}) {
	        return new TagCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.parent_id = source["parent_id"];
	        this.is_category = source["is_category"];
	        this.sort_order = source["sort_order"];
	        this.aliases = source["aliases"];
	    }
	}
	export class TagUpdate {
	    id: number;
	    name: string;
	    color: string;
	    parent_id?: number;
	    is_category: number;
	    sort_order: number;
	    aliases: string;
	
	    static createFrom(source: any = {}) {
	        return new TagUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.parent_id = source["parent_id"];
	        this.is_category = source["is_category"];
	        this.sort_order = source["sort_order"];
	        this.aliases = source["aliases"];
	    }
	}
	export class VaultInfo {
	    path: string;
	    name: string;
	    created_at: string;
	    file_count: number;
	
	    static createFrom(source: any = {}) {
	        return new VaultInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.created_at = source["created_at"];
	        this.file_count = source["file_count"];
	    }
	}

}

