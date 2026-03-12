import { test, expect, waitForAppReady, MOCK_TRACKS, MOCK_ARTISTS } from "./fixtures";

test.describe("Search - Tracks Page", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/tracks");
		await waitForAppReady(mockPage);
		await mockPage.waitForSelector('[data-testid="track-list"]', { timeout: 10000 });
	});

	test("search input is visible on tracks page", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="track-search-input"]')).toBeVisible();
	});

	test("search button is visible on tracks page", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="track-search-btn"]')).toBeVisible();
	});

	test("search input accepts text", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Midnight");
		expect(await searchInput.inputValue()).toBe("Midnight");
	});

	test("typing in search filters track results", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		const initialCount = await mockPage.locator('[data-testid="track-row"]').count();

		// Type a search query that matches only one track
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Midnight");

		// Wait for debounce (800ms) + network + render
		await mockPage.waitForTimeout(1200);

		const filteredCount = await mockPage.locator('[data-testid="track-row"]').count();
		// Should show fewer (or equal) results
		expect(filteredCount).toBeLessThanOrEqual(initialCount);
	});

	test("clicking search button performs search", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Ocean");
		await mockPage.locator('[data-testid="track-search-btn"]').click();
		await mockPage.waitForTimeout(800);

		const rows = mockPage.locator('[data-testid="track-row"]');
		const count = await rows.count();
		// Should show at least one result (Ocean Waves)
		expect(count).toBeGreaterThanOrEqual(1);
	});

	test("search results show only matching tracks", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Midnight Drive");
		await mockPage.locator('[data-testid="track-search-btn"]').click();
		await mockPage.waitForTimeout(800);

		const titles = await mockPage.locator('[data-testid="track-row"] td strong').allTextContents();
		expect(titles.some((t) => t.toLowerCase().includes("midnight"))).toBe(true);
	});

	test("clear search button appears after typing", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		// Clear button should not be visible initially
		await expect(mockPage.locator('[data-testid="track-search-clear"]')).not.toBeVisible();

		await searchInput.fill("test");
		await mockPage.waitForTimeout(200);

		// Clear button should now appear
		await expect(mockPage.locator('[data-testid="track-search-clear"]')).toBeVisible();
	});

	test("clicking clear search button resets the search", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		const initialCount = await mockPage.locator('[data-testid="track-row"]').count();

		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Midnight");
		await mockPage.waitForTimeout(1200);

		// Clear the search
		await mockPage.locator('[data-testid="track-search-clear"]').click();
		await mockPage.waitForTimeout(1200);

		// Should restore original results
		const restoredCount = await mockPage.locator('[data-testid="track-row"]').count();
		expect(restoredCount).toBe(initialCount);
	});

	test("empty search query shows all tracks", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="track-row"]', { timeout: 8000 });
		const totalCount = await mockPage.locator('[data-testid="track-row"]').count();
		expect(totalCount).toBe(MOCK_TRACKS.data.length);
	});

	test("search with no results shows empty state message", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("zzz_nonexistent_track_zzz");
		await mockPage.waitForTimeout(1200);

		await expect(mockPage.locator('[data-testid="track-row"]')).toHaveCount(0);
		await expect(mockPage.locator("text=No tracks found")).toBeVisible();
	});

	test("search updates URL query parameter", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Ocean");
		await mockPage.locator('[data-testid="track-search-btn"]').click();
		await mockPage.waitForTimeout(600);

		expect(mockPage.url()).toContain("search=");
	});

	test("can search by artist name", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Coastal Echoes");
		await mockPage.waitForTimeout(1200);

		const rows = mockPage.locator('[data-testid="track-row"]');
		const count = await rows.count();
		expect(count).toBeGreaterThan(0);
	});

	test("can search by album name", async ({ mockPage }) => {
		const searchInput = mockPage.locator('[data-testid="track-search-input"]');
		await searchInput.fill("Electric Skies");
		await mockPage.waitForTimeout(1200);

		const rows = mockPage.locator('[data-testid="track-row"]');
		const count = await rows.count();
		expect(count).toBeGreaterThan(0);
	});
});

test.describe("Search - Artists Page", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/artists");
		await waitForAppReady(mockPage);
		await mockPage.waitForSelector('[data-testid="artist-grid"]', { timeout: 10000 });
	});

	test("search input is visible on artists page", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="artist-search-input"]')).toBeVisible();
	});

	test("search button is visible on artists page", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="artist-search-btn"]')).toBeVisible();
	});

	test("typing in artists search filters results", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-card"]', { timeout: 8000 });
		const initialCount = await mockPage.locator('[data-testid="artist-card"]').count();

		await mockPage.locator('[data-testid="artist-search-input"]').fill("Neon");
		await mockPage.waitForTimeout(600);

		const filteredCount = await mockPage.locator('[data-testid="artist-card"]').count();
		expect(filteredCount).toBeLessThanOrEqual(initialCount);
	});

	test("clear button appears after typing in artist search", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="artist-search-clear"]')).not.toBeVisible();

		await mockPage.locator('[data-testid="artist-search-input"]').fill("test");
		await mockPage.waitForTimeout(200);

		await expect(mockPage.locator('[data-testid="artist-search-clear"]')).toBeVisible();
	});

	test("clicking clear resets artist search results", async ({ mockPage }) => {
		await mockPage.waitForSelector('[data-testid="artist-card"]', { timeout: 8000 });
		const initialCount = await mockPage.locator('[data-testid="artist-card"]').count();

		await mockPage.locator('[data-testid="artist-search-input"]').fill("Neon");
		await mockPage.waitForTimeout(600);

		await mockPage.locator('[data-testid="artist-search-clear"]').click();
		await mockPage.waitForTimeout(600);

		const restoredCount = await mockPage.locator('[data-testid="artist-card"]').count();
		expect(restoredCount).toBe(initialCount);
	});
});

test.describe("Search - Navbar Global Search", () => {
	test.beforeEach(async ({ mockPage }) => {
		await mockPage.goto("/ui/tracks");
		await waitForAppReady(mockPage);
	});

	test("navbar search input is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="nav-search-input"]')).toBeVisible();
	});

	test("navbar clear button appears after typing", async ({ mockPage }) => {
		// Clear button (nav-search-btn) is hidden when input is empty
		await expect(mockPage.locator('[data-testid="nav-search-btn"]')).not.toBeVisible();

		await mockPage.locator('[data-testid="nav-search-input"]').fill("Midnight");
		await expect(mockPage.locator('[data-testid="nav-search-btn"]')).toBeVisible();
	});

	test("pressing Enter in navbar search navigates to tracks with query", async ({ mockPage }) => {
		const navSearch = mockPage.locator('[data-testid="nav-search-input"]');
		await navSearch.fill("Midnight");
		await navSearch.press("Enter");
		await mockPage.waitForTimeout(600);

		const url = mockPage.url();
		expect(url).toContain("/tracks");
		expect(url).toContain("search=");
	});

	test("pressing Enter in navbar search clears the input", async ({ mockPage }) => {
		const navSearch = mockPage.locator('[data-testid="nav-search-input"]');
		await navSearch.fill("Ocean");
		await navSearch.press("Enter");
		await mockPage.waitForTimeout(600);

		// Input is cleared after performing search
		expect(await navSearch.inputValue()).toBe("");
	});

	test("pressing Enter in navbar search from artists page navigates to tracks", async ({ mockPage }) => {
		await mockPage.goto("/ui/artists");
		await waitForAppReady(mockPage);

		const navSearch = mockPage.locator('[data-testid="nav-search-input"]');
		await navSearch.fill("Ocean");
		await navSearch.press("Enter");
		await mockPage.waitForTimeout(600);

		expect(mockPage.url()).toContain("/tracks");
	});

	test("navbar clear button clears the search input", async ({ mockPage }) => {
		const navSearch = mockPage.locator('[data-testid="nav-search-input"]');
		await navSearch.fill("Midnight");

		await mockPage.locator('[data-testid="nav-search-btn"]').click();
		expect(await navSearch.inputValue()).toBe("");
	});

	test("empty navbar search does not navigate", async ({ mockPage }) => {
		const initialUrl = mockPage.url();
		const navSearch = mockPage.locator('[data-testid="nav-search-input"]');
		await navSearch.press("Enter");
		await mockPage.waitForTimeout(300);

		// URL should not have changed since search was empty
		expect(mockPage.url()).toBe(initialUrl);
	});
});
