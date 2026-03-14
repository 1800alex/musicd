import { test, expect, playFirstTrack } from "./fixtures";

test.describe("Visualizer", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
	});

	test("visualizer toggle button is visible in player bar", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-visualizer-btn"]')).toBeVisible();
	});

	test("visualizer overlay is not visible by default", async ({ mockPage }) => {
		// The visualizer overlay has class hidden-visualizer when not active
		const overlay = mockPage.locator('[data-testid="visualizer-overlay"]');
		await expect(overlay).toBeAttached();
		// It should not be visible (hidden via CSS class)
		await expect(overlay).not.toBeVisible();
	});

	test("clicking visualizer button shows the overlay", async ({ mockPage }) => {
		const vizBtn = mockPage.locator('[data-testid="player-visualizer-btn"]');

		// Initial state: not active (no active-info class)
		const initialClass = await vizBtn.getAttribute("class");
		expect(initialClass).not.toContain("active-info");

		// Click to enable
		await vizBtn.click();
		await mockPage.waitForTimeout(300);

		// Button should now have active-info class
		const activeClass = await vizBtn.getAttribute("class");
		expect(activeClass).toContain("active-info");
	});

	test("visualizer overlay becomes visible after clicking toggle", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);

		// Overlay should be visible now (hidden-visualizer class removed)
		const overlay = mockPage.locator('[data-testid="visualizer-overlay"]');
		await expect(overlay).toBeVisible();
	});

	test("visualizer canvas is present inside the overlay", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);

		await expect(mockPage.locator('[data-testid="visualizer-canvas"]')).toBeAttached();
	});

	test("visualizer overlay has a close button", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);

		await expect(mockPage.locator('[data-testid="visualizer-close-btn"]')).toBeVisible();
	});

	test("clicking close button hides the visualizer overlay", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="visualizer-overlay"]')).toBeVisible();

		await mockPage.locator('[data-testid="visualizer-close-btn"]').click();
		await mockPage.waitForTimeout(300);

		await expect(mockPage.locator('[data-testid="visualizer-overlay"]')).not.toBeVisible();
	});

	test("clicking toggle button again hides the visualizer", async ({ mockPage }) => {
		const vizBtn = mockPage.locator('[data-testid="player-visualizer-btn"]');

		// Open
		await vizBtn.click();
		await mockPage.waitForTimeout(300);
		expect(await vizBtn.getAttribute("class")).toContain("active-info");

		// Close
		await vizBtn.click();
		await mockPage.waitForTimeout(300);
		expect(await vizBtn.getAttribute("class")).not.toContain("active-info");
		await expect(mockPage.locator('[data-testid="visualizer-overlay"]')).not.toBeVisible();
	});

	test("pressing Escape closes the visualizer", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="visualizer-overlay"]')).toBeVisible();

		await mockPage.keyboard.press("Escape");
		await mockPage.waitForTimeout(300);

		await expect(mockPage.locator('[data-testid="visualizer-overlay"]')).not.toBeVisible();
	});

	test("visualizer shows current track info", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);

		// The visualizer overlay shows track title and artist
		const overlay = mockPage.locator('[data-testid="visualizer-overlay"]');
		await expect(overlay.locator(".visualizer-track-info")).toBeVisible();
	});

	test("player controls remain accessible while visualizer is open", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="player-visualizer-btn"]').click();
		await mockPage.waitForTimeout(300);

		// Player bar should still be accessible
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
		await expect(mockPage.locator('[data-testid="player-play-btn"]')).toBeVisible();
	});

	test("can toggle visualizer on and off repeatedly", async ({ mockPage }) => {
		const vizBtn = mockPage.locator('[data-testid="player-visualizer-btn"]');
		const overlay = mockPage.locator('[data-testid="visualizer-overlay"]');

		for (let i = 0; i < 3; i++) {
			await vizBtn.click();
			await mockPage.waitForTimeout(200);
			await expect(overlay).toBeVisible();

			await vizBtn.click();
			await mockPage.waitForTimeout(200);
			await expect(overlay).not.toBeVisible();
		}
	});
});
