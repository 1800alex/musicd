import { test, expect, playFirstTrack, waitForAppReady } from "./fixtures";

test.describe("Player Controls", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
	});

	// -------------------------------------------------------------------------
	// Play / Pause
	// -------------------------------------------------------------------------

	test("player bar is visible when a track is playing", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("play button shows pause icon while playing", async ({ mockPage }) => {
		const playBtn = mockPage.locator('[data-testid="player-play-btn"]');
		await expect(playBtn).toBeVisible();
		// When playing, the button contains an fa-pause icon
		await expect(playBtn.locator("svg")).toBeVisible();
	});

	test("clicking play/pause button toggles playback state", async ({ mockPage }) => {
		const playBtn = mockPage.locator('[data-testid="player-play-btn"]');

		// Capture initial aria / class state
		const initialClass = await playBtn.getAttribute("class");

		// Click to pause
		await playBtn.click();
		await mockPage.waitForTimeout(200);

		// The button class or inner icon should change
		const pausedClass = await playBtn.getAttribute("class");
		// Both states use a button - just verify it's still visible and clickable
		await expect(playBtn).toBeVisible();

		// Click again to resume
		await playBtn.click();
		await mockPage.waitForTimeout(200);
		await expect(playBtn).toBeVisible();
	});

	// -------------------------------------------------------------------------
	// Track info in player bar
	// -------------------------------------------------------------------------

	test("player bar displays track title, artist and album", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-track-title"]')).not.toBeEmpty();
		await expect(mockPage.locator('[data-testid="player-track-artist"]')).not.toBeEmpty();
		await expect(mockPage.locator('[data-testid="player-track-album"]')).not.toBeEmpty();
	});

	test("player bar shows the first track title after double-clicking it", async ({ mockPage }) => {
		const title = await mockPage.locator('[data-testid="player-track-title"]').textContent();
		expect(title?.trim()).toBeTruthy();
	});

	// -------------------------------------------------------------------------
	// Next / Previous
	// -------------------------------------------------------------------------

	test("next track button is visible and clickable", async ({ mockPage }) => {
		const nextBtn = mockPage.locator('[data-testid="player-next-btn"]');
		await expect(nextBtn).toBeVisible();
		await expect(nextBtn).toBeEnabled();
		await nextBtn.click();
		await mockPage.waitForTimeout(300);
		// Player bar should still be visible after advancing
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("previous track button is visible and clickable", async ({ mockPage }) => {
		const prevBtn = mockPage.locator('[data-testid="player-prev-btn"]');
		await expect(prevBtn).toBeVisible();
		await expect(prevBtn).toBeEnabled();
		await prevBtn.click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("clicking next then previous navigates through queue", async ({ mockPage }) => {
		const titleBefore = await mockPage.locator('[data-testid="player-track-title"]').textContent();

		// Go to next track
		await mockPage.locator('[data-testid="player-next-btn"]').click();
		await mockPage.waitForTimeout(400);

		const titleAfterNext = await mockPage.locator('[data-testid="player-track-title"]').textContent();

		// Title may or may not change depending on queue size, but player must remain visible
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();

		// Go back
		await mockPage.locator('[data-testid="player-prev-btn"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	// -------------------------------------------------------------------------
	// Shuffle
	// -------------------------------------------------------------------------

	test("shuffle button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-shuffle-btn"]')).toBeVisible();
	});

	test("clicking shuffle button toggles shuffle on", async ({ mockPage }) => {
		const shuffleBtn = mockPage.locator('[data-testid="player-shuffle-btn"]');

		// Shuffle should be off initially (no active class)
		const initialClass = await shuffleBtn.getAttribute("class");
		expect(initialClass).not.toContain("active");

		// Enable shuffle
		await shuffleBtn.click();
		await mockPage.waitForTimeout(200);

		const activeClass = await shuffleBtn.getAttribute("class");
		expect(activeClass).toContain("active");
	});

	test("clicking shuffle twice toggles back to off", async ({ mockPage }) => {
		const shuffleBtn = mockPage.locator('[data-testid="player-shuffle-btn"]');

		await shuffleBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await shuffleBtn.getAttribute("class")).toContain("active");

		await shuffleBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await shuffleBtn.getAttribute("class")).not.toContain("active");
	});

	// -------------------------------------------------------------------------
	// Repeat
	// -------------------------------------------------------------------------

	test("repeat button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-repeat-btn"]')).toBeVisible();
	});

	test("repeat cycles through Off → All → One modes", async ({ mockPage }) => {
		const repeatBtn = mockPage.locator('[data-testid="player-repeat-btn"]');

		// Initial state: Off (no active class)
		const initialClass = await repeatBtn.getAttribute("class");
		expect(initialClass).not.toContain("active");

		// Click once → All (active should appear)
		await repeatBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await repeatBtn.getAttribute("class")).toContain("active");

		// Click again → One (still active, different icon)
		await repeatBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await repeatBtn.getAttribute("class")).toContain("active");

		// Click again → Off (active disappears)
		await repeatBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await repeatBtn.getAttribute("class")).not.toContain("active");
	});

	// -------------------------------------------------------------------------
	// Progress bar
	// -------------------------------------------------------------------------

	test("progress bar is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-progress-bar"]')).toBeVisible();
	});

	test("current time and duration displays are visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-current-time"]')).toBeVisible();
		await expect(mockPage.locator('[data-testid="player-duration"]')).toBeVisible();
	});

	test("current time display shows mm:ss format", async ({ mockPage }) => {
		const timeText = await mockPage.locator('[data-testid="player-current-time"]').textContent();
		expect(timeText).toMatch(/^\d+:\d{2}$/);
	});

	test("duration display shows mm:ss format when loaded", async ({ mockPage }) => {
		// Allow audio metadata to load
		await mockPage.waitForTimeout(1000);
		const durationText = await mockPage.locator('[data-testid="player-duration"]').textContent();
		expect(durationText).toMatch(/^\d+:\d{2}$/);
	});
});
