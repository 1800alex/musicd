// Music Player Types
export interface Track {
	id: string;
	title: string;
	artist: string;
	album: string;
	year?: number;
	filename: string;
	file_path: string;
	cover_art_id?: string;
	duration?: number;
	duration_seconds?: number;
	playlist_position_id?: string; // Unique ID per playlist position (for deduplication when same track appears multiple times)
}

export interface Playlist {
	id: string;
	name: string;
	path: string;
	track_count: number;
	cover_art_id?: string;
}

export interface Artist {
	id: string;
	name: string;
	albums: Album[];
	tracks: Track[];
	track_count: number;
	cover_art_id?: string;
}

export interface Album {
	id: string;
	name: string;
	artist: string;
	year?: number;
	tracks: Track[];
	track_count: number;
	cover_art_id?: string;
}

export interface APIResponse<T> {
	data: T[];
	page: number;
	pageSize: number;
	totalPages: number;
	search: string;
	total: number;
}

export interface PlaylistResponse {
	playlist: Playlist;
	data: Track[];
	page: number;
	pageSize: number;
	totalPages: number;
	search: string;
	total: number;
}

export interface ArtistResponse {
	artist: Artist;
	data: Track[];
	page: number;
	pageSize: number;
	totalPages: number;
	search: string;
	total: number;
}

export interface AlbumResponse {
	album: Album;
	artist: string;
	data: Track[];
	page: number;
	pageSize: number;
	totalPages: number;
	search: string;
	total: number;
}
