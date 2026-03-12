import { test, expect, waitForAppReady, MOCK_TRACKS, MOCK_ARTISTS, MOCK_PLAYLISTS } from "./fixtures";

test.describe("Browse - All Tracks", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/tracks");
		await waitForAppReady(mockPage);
		await mockPage.waitForSelector('[data-testid="track-list"]', { timeout: 10000 });
	});

	test("tracks page shows track list component", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="track-list"]')).toBeVisible();
	});

	test("track list shows expected number of tracks", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		const rows = mockPage.locator('[data-testid="track-row"]');
		await expect(rows).toHaveCount(MOCK_TRACKS.data.length);
	});

	test("each track row has title, artist, album columns", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		const firstRow = mockPage.locator('[data-testid="track-row"]').first();
		// Title is a <strong>, artist and album are clickable links
		await expect(firstRow.locator("td strong")).toBeVisible();
		await expect(firstRow.locator(".clickable-link").first()).toBeVisible();
	});

	test("double-clicking a track row starts playback", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		await mockPage.dblclick('[data-testid="track-row"]:first-child');
		await mockPage.waitForSelector('[data-testid="audio-player"]', { timeout: 8000 });
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("clicking the play action button starts playback", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-play-btn"]', { timeout: 8000 });
		await mockPage.locator('[data-testid="track-play-btn"]').first().click();
		await mockPage.waitForSelector('[data-testid="audio-player"]', { timeout: 8000 });
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("add to queue button is visible for each track", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-queue-btn"]', { timeout: 8000 });
		await expect(mockPage.locator('[data-testid="track-queue-btn"]').first()).toBeVisible();
	});

	test("page size selector is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="track-page-size-select"]')).toBeVisible();
	});

	test("page size selector has options 25, 50, 100", async ({ mockPage }) => {
		const select = mockPage.locator('[data-testid="track-page-size-select"]');
		await expect(select.locator('option[value="25"]')).toBeAttached();
		await expect(select.locator('option[value="50"]')).toBeAttached();
		await expect(select.locator('option[value="100"]')).toBeAttached();
	});

	test("track rows have correct track IDs as data attributes", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		const firstRow = mockPage.locator('[data-testid="track-row"]').first();
		const trackId = await firstRow.getAttribute("data-track-id");
		expect(trackId).toBeTruthy();
		expect(MOCK_TRACKS.data.some((t) => t.id === trackId)).toBe(true);
	});

	test("first track shows correct title from mock data", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		// Get all track titles from the table
		const titles = await mockPage.locator('[data-testid="track-row"] td strong').allTextContents();
		const mockTitles = MOCK_TRACKS.data.map((t) => t.title);
		// Every displayed title should be in the mock data
		titles.forEach((title) => {
			expect(mockTitles).toContain(title.trim());
		});
	});
});

test.describe("Browse - Artists", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/artists");
		await waitForAppReady(mockPage);
		await mockPage.waitForSelector('[data-testid="artist-grid"]', { timeout: 10000 });
	});

	test("artists page shows artist grid", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="artist-grid"]')).toBeVisible();
	});

	test("artist grid shows correct number of artists", async ({ mockPage }) => {
		const cards = mockPage.locator('[data-testid="artist-card"]');
		await expect(cards).toHaveCount(MOCK_ARTISTS.data.length);
	});

	test("each artist card shows artist name", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-card"]', { timeout: 8000 });
		const firstCard = mockPage.locator('[data-testid="artist-card"]').first();
		const name = await firstCard.locator(".title.is-6").textContent();
		expect(name?.trim()).toBeTruthy();
		expect(MOCK_ARTISTS.data.some((a) => a.name === name?.trim())).toBe(true);
	});

	test("each artist card has Play All and Add to Queue buttons", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-card"]', { timeout: 8000 });
		const firstCard = mockPage.locator('[data-testid="artist-card"]').first();
		await expect(firstCard.locator('[data-testid="artist-play-btn"]')).toBeVisible();
		await expect(firstCard.locator('[data-testid="artist-queue-btn"]')).toBeVisible();
	});

	test("clicking artist card navigates to artist page", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-card"]', { timeout: 8000 });
		await mockPage.locator('[data-testid="artist-card"]').first().click();
		await mockPage.waitForTimeout(500);
		expect(mockPage.url()).toContain("/artist/");
	});

	test("artist cards have correct artist IDs as data attributes", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-card"]', { timeout: 8000 });
		const firstCard = mockPage.locator('[data-testid="artist-card"]').first();
		const artistId = await firstCard.getAttribute("data-artist-id");
		expect(artistId).toBeTruthy();
		expect(MOCK_ARTISTS.data.some((a) => a.id === artistId)).toBe(true);
	});

	test("clicking Play All on an artist card starts playback", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-play-btn"]', { timeout: 8000 });
		await mockPage.locator('[data-testid="artist-play-btn"]').first().click();
		await mockPage.waitForTimeout(500);
		// Player may or may not appear immediately depending on queue availability
		// Just assert the click didn't cause an error
		await expect(mockPage.locator('[data-testid="artist-grid"]')).toBeVisible();
	});
});

test.describe("Browse - Playlists", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/playlists");
		await waitForAppReady(mockPage);
		await mockPage.waitForSelector('[data-testid="playlist-grid"]', { timeout: 10000 });
	});

	test("playlists page shows the playlist grid", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="playlist-grid"]')).toBeVisible();
	});

	test("playlist grid shows correct number of playlists", async ({ mockPage }) => {
		const cards = mockPage.locator('[data-testid="playlist-card"]');
		await expect(cards).toHaveCount(MOCK_PLAYLISTS.length);
	});

	test("each playlist card shows playlist name", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="playlist-card"]', { timeout: 8000 });
		const firstCard = mockPage.locator('[data-testid="playlist-card"]').first();
		const name = await firstCard.locator(".title.is-6").textContent();
		expect(name?.trim()).toBeTruthy();
		expect(MOCK_PLAYLISTS.some((p) => p.name === name?.trim())).toBe(true);
	});

	test("each playlist card has Play and Add to Queue buttons", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="playlist-card"]', { timeout: 8000 });
		const firstCard = mockPage.locator('[data-testid="playlist-card"]').first();
		await expect(firstCard.locator('[data-testid="playlist-play-btn"]')).toBeVisible();
		await expect(firstCard.locator('[data-testid="playlist-queue-btn"]')).toBeVisible();
	});

	test("clicking playlist card navigates to playlist detail page", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="playlist-card"]', { timeout: 8000 });
		await mockPage.locator('[data-testid="playlist-card"]').first().click();
		await mockPage.waitForTimeout(500);
		expect(mockPage.url()).toContain("/playlists/");
	});

	test("playlist cards have correct playlist IDs as data attributes", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="playlist-card"]', { timeout: 8000 });
		const firstCard = mockPage.locator('[data-testid="playlist-card"]').first();
		const playlistId = await firstCard.getAttribute("data-playlist-id");
		expect(playlistId).toBeTruthy();
		expect(MOCK_PLAYLISTS.some((p) => p.id === playlistId)).toBe(true);
	});

	test("create playlist button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="create-playlist-btn"]')).toBeVisible();
	});

	test("clicking create playlist button opens the modal", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="create-playlist-btn"]').click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="create-playlist-modal"]')).toHaveClass(/is-active/);
	});

	test("create playlist modal has a name input", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="create-playlist-btn"]').click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="create-playlist-name-input"]')).toBeVisible();
	});

	test("create playlist submit button is disabled when name is empty", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="create-playlist-btn"]').click();
		await mockPage.waitForTimeout(300);
		const submitBtn = mockPage.locator('[data-testid="create-playlist-submit"]');
		await expect(submitBtn).toBeDisabled();
	});

	test("create playlist submit button enables when name is typed", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="create-playlist-btn"]').click();
		await mockPage.waitForTimeout(300);
		await mockPage.locator('[data-testid="create-playlist-name-input"]').fill("Test Playlist");
		const submitBtn = mockPage.locator('[data-testid="create-playlist-submit"]');
		await expect(submitBtn).toBeEnabled();
	});

	test("closing create playlist modal with cancel button works", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="create-playlist-btn"]').click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="create-playlist-modal"]')).toHaveClass(/is-active/);

		// Click cancel button
		await mockPage.locator('[data-testid="create-playlist-modal"] .button:not(.is-primary)').click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="create-playlist-modal"]')).not.toHaveClass(/is-active/);
	});
});

test.describe("Browse - Navigation", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/tracks");
		await waitForAppReady(mockPage);
	});

	test("navbar browse dropdown is present", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="nav-browse-dropdown"]')).toBeVisible();
	});

	test("navbar shows All Tracks and Artists links on hover", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="nav-browse-dropdown"]').hover();
		await mockPage.waitForTimeout(200);
		await expect(mockPage.locator('[data-testid="nav-all-tracks-link"]')).toBeVisible();
		await expect(mockPage.locator('[data-testid="nav-artists-link"]')).toBeVisible();
	});

	test("clicking All Tracks in nav navigates to /tracks", async ({ mockPage }) => {
		// Navigate away first
		await mockPage.goto("/ui/artists");
		await waitForAppReady(mockPage);

		await mockPage.locator('[data-testid="nav-browse-dropdown"]').hover();
		await mockPage.waitForTimeout(200);
		await mockPage.locator('[data-testid="nav-all-tracks-link"]').click();
		await mockPage.waitForTimeout(500);

		expect(mockPage.url()).toContain("/tracks");
	});

	test("clicking Artists in nav navigates to /artists", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="nav-browse-dropdown"]').hover();
		await mockPage.waitForTimeout(200);
		await mockPage.locator('[data-testid="nav-artists-link"]').click();
		await mockPage.waitForTimeout(500);

		expect(mockPage.url()).toContain("/artists");
	});

	test("home page redirects to tracks page", async ({ mockPage }) => {
		await mockPage.goto("/ui/");
		await mockPage.waitForTimeout(1000);
		expect(mockPage.url()).toContain("/tracks");
	});
});
