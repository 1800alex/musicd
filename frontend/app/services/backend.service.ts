import type { Track, Playlist, Artist, APIResponse, ArtistResponse, AlbumResponse, PlaylistResponse } from "~/types";
import httpService from "~/services/http.service";

const backendService = {
	FetchPlaylists,
	FetchPlaylist,
	FetchTrackById,
	FetchTracks,
	CreatePlaylist,
	AddTrackToPlaylist,
	FetchArtists,
	FetchArtist,
	FetchArtistTracks,
	FetchAlbum,
	FetchAlbumTracks,
	SearchTracks,
	// New ID-based playlist functions
	FetchPlaylistTracks,
	AddTrackToPlaylistById,
	RemoveTrackFromPlaylistById,
	DeletePlaylistById,
	// Rescan functions
	TriggerRescan,
	FetchScanStatus
};

export default backendService;

function getPageSize(params?: Record<string, any>): number | null {
	let size: number | null = null;
	if (params && params.pageSize) {
		if ("number" === typeof params.pageSize) {
			size = params.pageSize;
		} else {
			const parsedSize = parseInt(params.pageSize, 10);
			if (!isNaN(parsedSize) && parsedSize > 0) {
				size = parsedSize;
			}
		}
	}

	return size;
}

async function paginatedFetch<T>(url: string, params?: Record<string, any>): Promise<APIResponse<T>> {
	const pageSize = getPageSize(params);

	if (!pageSize || pageSize > 100) {
		// Fetch all pages 100 at a time
		let currentPage = 1;

		const result: APIResponse<T> = {
			data: [],
			page: 1,
			pageSize: 100,
			totalPages: 1,
			search: params?.search || "",
			total: 0
		};

		while (true) {
			const adjustedParams = { ...params, pageSize: 100, page: currentPage };

			const response = await httpService.get<APIResponse<T>>(url, { params: adjustedParams });

			if (response.data && response.data.data && response.data.data.length > 0) {
				result.data = result.data.concat(response.data.data);
				result.total = response.data.data.length;
			}

			if (response.data.totalPages <= 1) {
				return result;
			}
			if (currentPage >= response.data.totalPages) {
				return result;
			}
			currentPage += 1;
		}
	}

	const response = await httpService.get<APIResponse<T>>(url, { params: params });
	if (response.data && !response.data.data) {
		response.data.data = [];
	}

	return response.data;
}

async function FetchArtists(params?: Record<string, any>): Promise<APIResponse<Artist>> {
	const response = await paginatedFetch<Artist>("/api/artists", params);
	if (!response.data) {
		response.data = [];
	}

	return response;
}

async function FetchArtist(id: string, params?: Record<string, any>): Promise<ArtistResponse> {
	const response = await httpService.get<ArtistResponse>(`/api/artist/${encodeURIComponent(id)}`, {
		params
	});
	if (!response.data.data) {
		response.data.data = [];
	}

	return response.data;
}

async function FetchArtistTracks(id: string, params?: Record<string, any>): Promise<APIResponse<Track>> {
	const response = await paginatedFetch<Track>(`/api/artist/${encodeURIComponent(id)}/tracks`, params);
	if (!response.data) {
		response.data = [];
	}

	return response;
}

async function FetchAlbum(id: string, params?: Record<string, any>): Promise<AlbumResponse> {
	const response = await httpService.get<AlbumResponse>(`/api/album/${encodeURIComponent(id)}`, {
		params
	});
	if (!response.data.data) {
		response.data.data = [];
	}

	return response.data;
}

async function FetchAlbumTracks(id: string, params?: Record<string, any>): Promise<APIResponse<Track>> {
	const response = await paginatedFetch<Track>(`/api/album/${encodeURIComponent(id)}/tracks`, params);
	if (!response.data) {
		response.data = [];
	}

	return response;
}

async function SearchTracks(query: string, params?: Record<string, any>): Promise<APIResponse<Track>> {
	const response = await paginatedFetch<Track>("/api/search", { ...params, q: query });
	if (!response.data) {
		response.data = [];
	}

	return response;
}

async function FetchTrackById(id: string): Promise<Track> {
	const response = await httpService.get<Track>(`/api/track/${encodeURIComponent(id)}`);
	return response.data;
}

async function FetchTracks(params?: Record<string, any>): Promise<APIResponse<Track>> {
	const response = await paginatedFetch<Track>("/api/tracks", params);
	if (!response.data) {
		response.data = [];
	}

	return response;
}

async function FetchPlaylists(params?: Record<string, any>): Promise<Playlist[]> {
	const response = await httpService.get<Playlist[]>("/api/playlists", { params });
	return response.data || [];
}

async function FetchPlaylist(id: string, params?: Record<string, any>): Promise<Playlist> {
	const response = await httpService.get<Playlist>(`/api/playlist/${encodeURIComponent(id)}`, { params });
	if (!response.data) {
		response.data = {} as Playlist;
	}

	return response.data;
}

async function FetchPlaylistTracks(id: string, params?: Record<string, any>): Promise<APIResponse<Track>> {
	const response = await paginatedFetch<Track>(`/api/playlist/${encodeURIComponent(id)}/tracks`, params);
	if (!response.data) {
		response.data = [];
	}

	return response;
}

async function CreatePlaylist(name: string): Promise<void> {
	if (!name.trim()) {
		return;
	}

	await httpService.post("/api/playlist", { name: name.trim() });
}

async function AddTrackToPlaylist(track: Track, playlistName: string): Promise<void> {
	await httpService.post(`/api/playlist/${encodeURIComponent(playlistName)}/add/${track.id}`, {});
}

async function AddTrackToPlaylistById(trackId: string, playlistId: string): Promise<void> {
	await httpService.post(`/api/playlist/${encodeURIComponent(playlistId)}/add/${encodeURIComponent(trackId)}`, {});
}

async function RemoveTrackFromPlaylistById(trackId: string, playlistId: string): Promise<void> {
	await httpService.delete(`/api/playlist/${encodeURIComponent(playlistId)}/remove/${encodeURIComponent(trackId)}`);
}

async function DeletePlaylistById(playlistId: string): Promise<void> {
	await httpService.delete(`/api/playlist/${encodeURIComponent(playlistId)}`);
}

async function TriggerRescan(): Promise<void> {
	await httpService.post("/api/rescan", {});
}

async function FetchScanStatus(): Promise<boolean> {
	const response = await httpService.get<{ scanning: boolean }>("/api/scan-status");
	return response.data?.scanning ?? false;
}
