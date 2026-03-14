import { test as base, expect, type Page } from "@playwright/test";

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

export const MOCK_TRACKS = {
	data: [
		{
			id: "track-1",
			title: "Midnight Drive",
			artist: "The Neon Pilots",
			album: "Electric Skies",
			year: 2022,
			filename: "midnight_drive.mp3",
			file_path: "test/midnight_drive.mp3",
			cover_art_id: "cover-1",
			duration_seconds: 210,
			duration: 210
		},
		{
			id: "track-2",
			title: "Ocean Waves",
			artist: "Coastal Echoes",
			album: "Deep Blue",
			year: 2021,
			filename: "ocean_waves.mp3",
			file_path: "test/ocean_waves.mp3",
			cover_art_id: "cover-2",
			duration_seconds: 185,
			duration: 185
		},
		{
			id: "track-3",
			title: "Mountain High",
			artist: "The Neon Pilots",
			album: "Electric Skies",
			year: 2022,
			filename: "mountain_high.mp3",
			file_path: "test/mountain_high.mp3",
			cover_art_id: "cover-1",
			duration_seconds: 240,
			duration: 240
		},
		{
			id: "track-4",
			title: "City Lights",
			artist: "Urban Pulse",
			album: "Concrete Jungle",
			year: 2023,
			filename: "city_lights.mp3",
			file_path: "test/city_lights.mp3",
			cover_art_id: "cover-3",
			duration_seconds: 195,
			duration: 195
		},
		{
			id: "track-5",
			title: "Stargazing",
			artist: "Coastal Echoes",
			album: "Deep Blue",
			year: 2021,
			filename: "stargazing.mp3",
			file_path: "test/stargazing.mp3",
			cover_art_id: "cover-2",
			duration_seconds: 220,
			duration: 220
		}
	],
	page: 1,
	pageSize: 25,
	totalPages: 1,
	total: 5,
	search: ""
};

export const MOCK_ARTISTS = {
	data: [
		{
			id: "artist-1",
			name: "The Neon Pilots",
			track_count: 2,
			cover_art_id: "cover-1",
			albums: [
				{
					id: "album-1",
					name: "Electric Skies",
					artist: "The Neon Pilots",
					year: 2022,
					track_count: 2,
					cover_art_id: "cover-1",
					tracks: []
				}
			],
			tracks: MOCK_TRACKS.data.filter((t) => t.artist === "The Neon Pilots")
		},
		{
			id: "artist-2",
			name: "Coastal Echoes",
			track_count: 2,
			cover_art_id: "cover-2",
			albums: [
				{
					id: "album-2",
					name: "Deep Blue",
					artist: "Coastal Echoes",
					year: 2021,
					track_count: 2,
					cover_art_id: "cover-2",
					tracks: []
				}
			],
			tracks: MOCK_TRACKS.data.filter((t) => t.artist === "Coastal Echoes")
		},
		{
			id: "artist-3",
			name: "Urban Pulse",
			track_count: 1,
			cover_art_id: "cover-3",
			albums: [
				{
					id: "album-3",
					name: "Concrete Jungle",
					artist: "Urban Pulse",
					year: 2023,
					track_count: 1,
					cover_art_id: "cover-3",
					tracks: []
				}
			],
			tracks: MOCK_TRACKS.data.filter((t) => t.artist === "Urban Pulse")
		}
	],
	page: 1,
	pageSize: 25,
	totalPages: 1,
	total: 3,
	search: ""
};

export const MOCK_PLAYLISTS = [
	{
		id: "playlist-1",
		name: "My Favorites",
		path: "playlists/favorites.m3u",
		track_count: 3,
		cover_art_id: "cover-1"
	},
	{
		id: "playlist-2",
		name: "Workout Mix",
		path: "playlists/workout.m3u",
		track_count: 5,
		cover_art_id: "cover-2"
	}
];

export const MOCK_PLAYLIST_TRACKS = {
	data: MOCK_TRACKS.data.slice(0, 3),
	page: 1,
	pageSize: 25,
	totalPages: 1,
	total: 3,
	search: ""
};

export const MOCK_ALBUM = {
	album: {
		id: "album-1",
		name: "Electric Skies",
		artist: "The Neon Pilots",
		year: 2022,
		track_count: 2,
		cover_art_id: "cover-1",
		tracks: []
	},
	tracks: {
		data: MOCK_TRACKS.data.filter((t) => t.album === "Electric Skies"),
		page: 1,
		pageSize: 25,
		totalPages: 1,
		total: 2,
		search: ""
	}
};

// ---------------------------------------------------------------------------
// Create a minimal silent WAV file for mocking audio playback
// ---------------------------------------------------------------------------
function createSilentWav(durationSeconds = 1): Buffer {
	const sampleRate = 8000;
	const numChannels = 1;
	const bitsPerSample = 8;
	const numSamples = sampleRate * numChannels * durationSeconds;
	const dataSize = numSamples * (bitsPerSample / 8);
	const buf = Buffer.alloc(44 + dataSize);
	let o = 0;
	buf.write("RIFF", o);
	o += 4;
	buf.writeUInt32LE(36 + dataSize, o);
	o += 4;
	buf.write("WAVE", o);
	o += 4;
	buf.write("fmt ", o);
	o += 4;
	buf.writeUInt32LE(16, o);
	o += 4;
	buf.writeUInt16LE(1, o);
	o += 2; // PCM
	buf.writeUInt16LE(numChannels, o);
	o += 2;
	buf.writeUInt32LE(sampleRate, o);
	o += 4;
	buf.writeUInt32LE(sampleRate * numChannels * (bitsPerSample / 8), o);
	o += 4;
	buf.writeUInt16LE(numChannels * (bitsPerSample / 8), o);
	o += 2;
	buf.writeUInt16LE(bitsPerSample, o);
	o += 2;
	buf.write("data", o);
	o += 4;
	buf.writeUInt32LE(dataSize, o);
	o += 4;
	buf.fill(0x80, o); // 8-bit unsigned silence = 0x80
	return buf;
}

const SILENT_WAV = createSilentWav(3);

// 1x1 pixel PNG for cover art mocking
const PLACEHOLDER_PNG = Buffer.from(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
	"base64"
);

// ---------------------------------------------------------------------------
// Setup all API mocks on a given page
// ---------------------------------------------------------------------------
export async function setupApiMocks(page: Page) {
	// Tracks list
	await page.route("**/api/tracks*", (route) => {
		const url = new URL(route.request().url());
		const search = url.searchParams.get("search") || "";
		const filtered = search
			? MOCK_TRACKS.data.filter(
					(t) =>
						t.title.toLowerCase().includes(search.toLowerCase()) ||
						t.artist.toLowerCase().includes(search.toLowerCase()) ||
						t.album.toLowerCase().includes(search.toLowerCase())
				)
			: MOCK_TRACKS.data;
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({ ...MOCK_TRACKS, data: filtered, total: filtered.length })
		});
	});

	// Individual track by id
	await page.route("**/api/track/*", (route) => {
		const url = route.request().url();
		const id = url.split("/api/track/")[1];
		const track = MOCK_TRACKS.data.find((t) => t.id === id);
		route.fulfill({
			status: track ? 200 : 404,
			contentType: "application/json",
			body: JSON.stringify(track || {})
		});
	});

	// Artists list
	await page.route("**/api/artists*", (route) => {
		const url = new URL(route.request().url());
		const search = url.searchParams.get("search") || "";
		const filtered = search
			? MOCK_ARTISTS.data.filter((a) => a.name.toLowerCase().includes(search.toLowerCase()))
			: MOCK_ARTISTS.data;
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({ ...MOCK_ARTISTS, data: filtered, total: filtered.length })
		});
	});

	// Individual artist
	await page.route("**/api/artist/*", (route) => {
		const url = route.request().url();
		const parts = url.split("/api/artist/");
		const id = parts[1]?.split("/")[0];
		const artist = MOCK_ARTISTS.data.find((a) => a.id === id);
		if (url.includes("/tracks")) {
			const tracks = MOCK_TRACKS.data.filter((t) => t.artist === artist?.name);
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					data: tracks,
					page: 1,
					pageSize: 25,
					totalPages: 1,
					total: tracks.length,
					search: ""
				})
			});
		} else {
			route.fulfill({
				status: artist ? 200 : 404,
				contentType: "application/json",
				body: JSON.stringify({
					artist,
					tracks: { data: [], page: 1, pageSize: 25, totalPages: 1, total: 0, search: "" }
				})
			});
		}
	});

	// Playlists list
	await page.route("**/api/playlists*", (route) => {
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(MOCK_PLAYLISTS)
		});
	});

	// Individual playlist
	await page.route("**/api/playlist/*", (route) => {
		const url = route.request().url();
		if (url.includes("/tracks")) {
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify(MOCK_PLAYLIST_TRACKS)
			});
		} else if (route.request().method() === "POST") {
			route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ ok: true }) });
		} else {
			const id = url.split("/api/playlist/")[1]?.split("/")[0];
			const playlist = MOCK_PLAYLISTS.find((p) => p.id === id);
			route.fulfill({
				status: playlist ? 200 : 404,
				contentType: "application/json",
				body: JSON.stringify(playlist || {})
			});
		}
	});

	// Album
	await page.route("**/api/album/*", (route) => {
		const url = route.request().url();
		if (url.includes("/tracks")) {
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify(MOCK_ALBUM.tracks)
			});
		} else {
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify(MOCK_ALBUM)
			});
		}
	});

	// Audio stream - return a silent WAV so audio element has real duration
	await page.route("**/api/music/**", (route) => {
		route.fulfill({
			status: 200,
			contentType: "audio/wav",
			body: SILENT_WAV
		});
	});

	// Cover art images - return a 1x1 PNG
	await page.route("**/api/cover-art/**", (route) => {
		route.fulfill({
			status: 200,
			contentType: "image/png",
			body: PLACEHOLDER_PNG
		});
	});

	// Sessions endpoint (for remote control feature)
	await page.route("**/api/sessions*", (route) => {
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify([])
		});
	});

	// Search
	await page.route("**/api/search*", (route) => {
		const url = new URL(route.request().url());
		const q = url.searchParams.get("q") || url.searchParams.get("search") || "";
		const filtered = MOCK_TRACKS.data.filter(
			(t) =>
				t.title.toLowerCase().includes(q.toLowerCase()) ||
				t.artist.toLowerCase().includes(q.toLowerCase()) ||
				t.album.toLowerCase().includes(q.toLowerCase())
		);
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({ data: filtered, page: 1, pageSize: 25, totalPages: 1, total: filtered.length, search: q })
		});
	});

	// Scan status
	await page.route("**/api/scan/status*", (route) => {
		route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(false) });
	});
}

// ---------------------------------------------------------------------------
// Wait for the Nuxt SPA to fully hydrate
// ---------------------------------------------------------------------------
export async function waitForAppReady(page: Page) {
	// Wait for the main content or player to be visible
	await page.waitForLoadState("networkidle", { timeout: 15000 }).catch(() => {});
	// Give Vue reactivity time to settle
	await page.waitForTimeout(500);
}

// ---------------------------------------------------------------------------
// Navigate to a track page and play the first track, returning after the
// audio player is visible in the UI.
// ---------------------------------------------------------------------------
export async function playFirstTrack(page: Page) {
	await page.goto("/ui/tracks");
	await waitForAppReady(page);
	// Wait for at least one track row to appear
	await page.waitForSelector('[data-testid="track-row"]', { timeout: 10000 });
	// Double-click first track to start playback
	await page.dblclick('[data-testid="track-row"]:first-child');
	// Wait for the player bar to appear
	await page.waitForSelector('[data-testid="audio-player"]', { timeout: 8000 });
	// Give the player state a moment to settle
	await page.waitForTimeout(300);
}

// ---------------------------------------------------------------------------
// Custom test fixture: auto-setup API mocks before each test
// ---------------------------------------------------------------------------
export const test = base.extend<{ mockPage: Page }>({
	mockPage: async ({ page }, use) => {
		await setupApiMocks(page);
		await use(page);
	}
});

export { expect };
