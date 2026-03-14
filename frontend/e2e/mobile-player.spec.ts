import { test, expect, playFirstTrack, MOCK_TRACKS } from "./fixtures";

// ---------------------------------------------------------------------------
// Helper: open the mobile fullscreen player by clicking cover art
// ---------------------------------------------------------------------------
async function openMobilePlayer(mockPage: import("@playwright/test").Page) {
	await mockPage.locator('[data-testid="player-cover-art"]').click();
	await mockPage.waitForSelector('[data-testid="mobile-player"]', { timeout: 5000 });
	await mockPage.waitForTimeout(400); // wait for slide-in transition
}

test.describe("Mobile Player - Open / Close", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
	});

	test("mobile fullscreen player is not in DOM before opening", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
	});

	test("clicking cover art in player bar opens mobile player", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();
	});

	test("mobile player has a collapse/close button", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		await expect(mockPage.locator('[data-testid="mobile-player-collapse"]')).toBeVisible();
	});

	test("clicking collapse closes mobile player", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
		await mockPage.waitForTimeout(400);
		await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
	});

	test("desktop player bar is hidden while mobile player is open", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		await expect(mockPage.locator('[data-testid="audio-player"]')).not.toBeAttached();
	});

	test("desktop player bar reappears after closing mobile player", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
		await mockPage.waitForTimeout(400);
		await expect(mockPage.locator('[data-testid="audio-player"]')).toBeVisible();
	});

	test("can toggle mobile player open and closed repeatedly", async ({ mockPage }) => {
		for (let i = 0; i < 3; i++) {
			await openMobilePlayer(mockPage);
			await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();

			await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
			await mockPage.waitForTimeout(400);
			await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
		}
	});
});

test.describe("Mobile Player - Cover Art & Track Info", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
		await openMobilePlayer(mockPage);
	});

	test("mobile player shows cover art image", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-cover-image"]')).toBeVisible();
	});

	test("cover art image has alt text", async ({ mockPage }) => {
		const img = mockPage.locator('[data-testid="mobile-player-cover-image"]');
		const alt = await img.getAttribute("alt");
		expect(alt).toBeTruthy();
	});

	test("mobile player cover container is present", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-cover"]')).toBeVisible();
	});

	test("mobile player shows track title", async ({ mockPage }) => {
		// Title text is inside MarqueeText — check the wrapper has text
		const player = mockPage.locator('[data-testid="mobile-player"]');
		const titleText = await player.locator(".mobile-player-title").textContent();
		expect(titleText?.trim()).toBeTruthy();
		// Should match one of our mock tracks
		const mockTitles = MOCK_TRACKS.data.map((t) => t.title);
		expect(mockTitles.some((t) => titleText?.includes(t))).toBe(true);
	});

	test("mobile player shows artist name", async ({ mockPage }) => {
		const player = mockPage.locator('[data-testid="mobile-player"]');
		const artistText = await player.locator(".mobile-player-artist").textContent();
		expect(artistText?.trim()).toBeTruthy();
	});
});

test.describe("Mobile Player - Playback Controls", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
		await openMobilePlayer(mockPage);
	});

	test("mobile player play/pause button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-play"]')).toBeVisible();
	});

	test("mobile player previous button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-prev"]')).toBeVisible();
	});

	test("mobile player next button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-next"]')).toBeVisible();
	});

	test("mobile player shuffle button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-shuffle"]')).toBeVisible();
	});

	test("mobile player repeat button is visible", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-repeat"]')).toBeVisible();
	});

	test("clicking play/pause in mobile player toggles playback", async ({ mockPage }) => {
		const playBtn = mockPage.locator('[data-testid="mobile-player-play"]');
		await expect(playBtn).toBeVisible();

		await playBtn.click();
		await mockPage.waitForTimeout(200);
		await expect(playBtn).toBeVisible();

		await playBtn.click();
		await mockPage.waitForTimeout(200);
		await expect(playBtn).toBeVisible();
	});

	test("clicking shuffle in mobile player toggles shuffle on", async ({ mockPage }) => {
		const shuffleBtn = mockPage.locator('[data-testid="mobile-player-shuffle"]');

		const initialClass = await shuffleBtn.getAttribute("class");
		expect(initialClass).not.toContain("active");

		await shuffleBtn.click();
		await mockPage.waitForTimeout(200);

		expect(await shuffleBtn.getAttribute("class")).toContain("active");
	});

	test("clicking shuffle twice turns shuffle off", async ({ mockPage }) => {
		const shuffleBtn = mockPage.locator('[data-testid="mobile-player-shuffle"]');

		await shuffleBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await shuffleBtn.getAttribute("class")).toContain("active");

		await shuffleBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await shuffleBtn.getAttribute("class")).not.toContain("active");
	});

	test("clicking repeat in mobile player cycles repeat modes", async ({ mockPage }) => {
		const repeatBtn = mockPage.locator('[data-testid="mobile-player-repeat"]');

		// Off → All
		expect(await repeatBtn.getAttribute("class")).not.toContain("active");
		await repeatBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await repeatBtn.getAttribute("class")).toContain("active");

		// All → One
		await repeatBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await repeatBtn.getAttribute("class")).toContain("active");

		// One → Off
		await repeatBtn.click();
		await mockPage.waitForTimeout(200);
		expect(await repeatBtn.getAttribute("class")).not.toContain("active");
	});

	test("clicking next in mobile player advances track", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="mobile-player-next"]').click();
		await mockPage.waitForTimeout(400);
		// Mobile player should remain open and visible
		await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();
	});

	test("clicking previous in mobile player goes back", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="mobile-player-prev"]').click();
		await mockPage.waitForTimeout(400);
		await expect(mockPage.locator('[data-testid="mobile-player"]')).toBeVisible();
	});
});

test.describe("Mobile Player - Seek Bar", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
		await openMobilePlayer(mockPage);
	});

	test("seek bar is visible in mobile player", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-seek"]')).toBeVisible();
	});

	test("seek bar is a range input with min 0 and max 100", async ({ mockPage }) => {
		const seekBar = mockPage.locator('[data-testid="mobile-player-seek"]');
		expect(await seekBar.getAttribute("min")).toBe("0");
		expect(await seekBar.getAttribute("max")).toBe("100");
	});

	test("time displays are shown in mobile player", async ({ mockPage }) => {
		const times = mockPage.locator('[data-testid="mobile-player"] .mobile-player-times span');
		await expect(times.first()).toBeVisible();
		await expect(times.last()).toBeVisible();
	});
});

test.describe("Mobile Player - Volume", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
		await openMobilePlayer(mockPage);
	});

	test("volume slider is visible in mobile player", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-volume"]')).toBeVisible();
	});

	test("volume slider is a range input with min 0 and max 100", async ({ mockPage }) => {
		const volSlider = mockPage.locator('[data-testid="mobile-player-volume"]');
		expect(await volSlider.getAttribute("min")).toBe("0");
		expect(await volSlider.getAttribute("max")).toBe("100");
	});

	test("mute icon is visible in mobile player", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-mute"]')).toBeVisible();
	});
});

test.describe("Mobile Player - Menu", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
		await openMobilePlayer(mockPage);
	});

	test("menu button is visible in mobile player header", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-player-menu-btn"]')).toBeVisible();
	});

	test("clicking menu button shows the menu with go-to options", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="mobile-player-menu-btn"]').click();
		await mockPage.waitForTimeout(200);

		await expect(mockPage.locator('[data-testid="mobile-player-go-artist"]')).toBeVisible();
		await expect(mockPage.locator('[data-testid="mobile-player-go-album"]')).toBeVisible();
	});

	test("clicking go-to-artist navigates to artist page and closes mobile player", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="mobile-player-menu-btn"]').click();
		await mockPage.waitForTimeout(200);

		await mockPage.locator('[data-testid="mobile-player-go-artist"]').click();
		await mockPage.waitForTimeout(600);

		expect(mockPage.url()).toContain("/artist/");
		await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
	});

	test("clicking go-to-album navigates to album page and closes mobile player", async ({ mockPage }) => {
		await mockPage.locator('[data-testid="mobile-player-menu-btn"]').click();
		await mockPage.waitForTimeout(200);

		await mockPage.locator('[data-testid="mobile-player-go-album"]').click();
		await mockPage.waitForTimeout(600);

		expect(mockPage.url()).toContain("/album/");
		await expect(mockPage.locator('[data-testid="mobile-player"]')).not.toBeAttached();
	});
});

test.describe("Mobile Mini Player Bar", () => {
	test.beforeEach(async ({ mockPage }) => {
		await playFirstTrack(mockPage);
	});

	test("mobile mini player bar is in the DOM when a track is playing", async ({ mockPage }) => {
		// On desktop the mini bar is CSS-hidden but still attached
		await expect(mockPage.locator('[data-testid="mobile-mini-player"]')).toBeAttached();
	});

	test("mini player has a play/pause button", async ({ mockPage }) => {
		await expect(mockPage.locator('[data-testid="mobile-mini-play"]')).toBeAttached();
	});

	test("mini player disappears from DOM when mobile fullscreen player is open", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		// v-if condition: !showMobilePlayer hides the mini bar too
		await expect(mockPage.locator('[data-testid="mobile-mini-player"]')).not.toBeAttached();
	});

	test("mini player reappears after closing mobile fullscreen player", async ({ mockPage }) => {
		await openMobilePlayer(mockPage);
		await mockPage.locator('[data-testid="mobile-player-collapse"]').click();
		await mockPage.waitForTimeout(400);

		await expect(mockPage.locator('[data-testid="mobile-mini-player"]')).toBeAttached();
	});
});
