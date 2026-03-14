import { test, expect, playFirstTrack } from "./fixtures";

test.describe("Album Art", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
	});

	// -------------------------------------------------------------------------
	// Cover art thumbnail in player bar
	// -------------------------------------------------------------------------

	test("cover art thumbnail is visible in the player bar", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-cover-art"]')).toBeVisible();
	});

	test("cover art thumbnail contains an img element", async ({ mockPage }) => {
		const coverArt = mockPage.locator('[data-testid="player-cover-art"]');
		await expect(coverArt.locator("img")).toBeVisible();
	});

	test("cover art thumbnail img has correct alt text", async ({ mockPage }) => {
		const img = mockPage.locator('[data-testid="player-cover-art"] img');
		const alt = await img.getAttribute("alt");
		expect(alt).toBeTruthy();
		expect(alt!.length).toBeGreaterThan(0);
	});

	// -------------------------------------------------------------------------
	// Mobile fullscreen player: open via cover art click
	// -------------------------------------------------------------------------

	test("clicking cover art opens the mobile fullscreen player", async ({ mockPage }) => {
		// Mobile player should not be in DOM initially
		await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();

		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();
	});

	test("mobile player shows the cover image", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="mobile-player-cover-image"]')).toBeVisible();
	});

	test("mobile player has a collapse button", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="mobile-player-collapse"]')).toBeVisible();
	});

	test("desktop player bar is hidden when mobile player is open", async ({ mockPage }) => {
		// Desktop player bar is visible before opening mobile player
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();

		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);

		// v-if removes it from DOM entirely
		await expect(mockPage.locator('[data-testid="audio-player"]')).not.toBeAttached();
	});

	test("clicking collapse button dismisses the mobile player", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);
		await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();

		await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
	});

	test("desktop player bar is restored after closing mobile player", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);

		await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("mobile player cover image has alt text", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-cover-art"]').click();
		await mockPage.waitForTimeout(400);

		const img = mockPage.locator('[data-testid="mobile-player-cover-image"]');
		const alt = await img.getAttribute("alt");
		expect(alt).toBeTruthy();
	});

	test("can open and close mobile player multiple times", async ({ mockPage }) => {
		for (let i = 0; i < 3; i++) {
			await mockPage.locator('[data-testid="player-cover-art"]').click();
			await mockPage.waitForTimeout(400);
			await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();

			await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
			await mockPage.waitForTimeout(400);
			await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
		}
	});
});
