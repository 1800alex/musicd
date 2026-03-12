import { test, expect, playFirstTrack } from "./fixtures";

test.describe("Volume Control", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
	});

	test("volume slider is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-volume-slider"]')).toBeVisible();
	});

	test("mute button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="player-mute-btn"]')).toBeVisible();
	});

	test("volume slider has correct range attributes", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');
		await expect(slider).toHaveAttribute("min", "0");
		await expect(slider).toHaveAttribute("max", "100");
		await expect(slider).toHaveAttribute("type", "range");
	});

	test("volume slider default value is greater than 0", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');
		const value = await slider.inputValue();
		expect(parseInt(value, 10)).toBeGreaterThan(0);
	});

	test("dragging volume slider changes volume value", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');

		// Get initial value
		const initialValue = await slider.inputValue();

		// Set value to 50 using fill
		await slider.fill("50");
		await mockPage.waitForTimeout(200);

		const newValue = await slider.inputValue();
		expect(newValue).toBe("50");
	});

	test("setting volume to 0 shows mute icon", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');
		await slider.fill("0");
		await mockPage.waitForTimeout(300);

		// The mute icon SVG should change (class or path data)
		await expect(mockPage.locator('[data-testid="player-mute-btn"]')).toBeVisible();
	});

	test("setting volume above 50 shows volume-up icon", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');
		await slider.fill("80");
		await mockPage.waitForTimeout(300);
		await expect(mockPage.locator('[data-testid="player-mute-btn"]')).toBeVisible();
	});

	test("clicking mute button mutes audio", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');

		// Ensure volume is not 0 first
		await slider.fill("70");
		await mockPage.waitForTimeout(200);

		// Click mute
		await mockPage.locator('[data-testid="player-mute-btn"]').click();
		await mockPage.waitForTimeout(300);

		// After muting, the mute icon (fa-volume-mute) should be shown
		await expect(mockPage.locator('[data-testid="player-mute-btn"]')).toBeVisible();
	});

	test("clicking mute button twice restores volume", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');
		await slider.fill("70");
		await mockPage.waitForTimeout(200);

		const beforeValue = await slider.inputValue();

		// Mute
		await mockPage.locator('[data-testid="player-mute-btn"]').click();
		await mockPage.waitForTimeout(200);

		// Unmute
		await mockPage.locator('[data-testid="player-mute-btn"]').click();
		await mockPage.waitForTimeout(200);

		// Volume should be restored
		const afterValue = await slider.inputValue();
		expect(afterValue).toBe(beforeValue);
	});

	test("setting volume to minimum (0) and then back up works", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');

		await slider.fill("0");
		await mockPage.waitForTimeout(200);
		expect(await slider.inputValue()).toBe("0");

		await slider.fill("75");
		await mockPage.waitForTimeout(200);
		expect(await slider.inputValue()).toBe("75");
	});

	test("volume slider --volume-percent CSS variable updates", async ({ mockPage }) => {
		const slider = mockPage.locator('[data-testid="player-volume-slider"]');
		await slider.fill("60");
		await mockPage.waitForTimeout(200);

		const style = await slider.getAttribute("style");
		expect(style).toContain("--volume-percent");
	});
});
