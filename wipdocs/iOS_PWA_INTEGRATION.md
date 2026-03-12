# iOS PWA Integration Guide

This guide explains the iOS PWA improvements that have been added to your music player application.

## What's New

### 1. **PWA Manifest** (`public/manifest.json`)

- Enables "Add to Home Screen" on iOS and Android
- Configures app display mode, colors, and icons
- Includes app shortcuts for quick access
- iOS recognizes the app as installable

### 2. **Service Worker** (`public/sw.js`)

- Caches essential static assets for offline access
- Implements network-first strategy for API calls
- Enables background sync capabilities
- Reduces bandwidth usage

### 3. **iOS Meta Tags** (in `nuxt.config.ts`)

- `apple-mobile-web-app-capable` - Allows standalone mode
- `apple-mobile-web-app-status-bar-style` - Black translucent status bar
- `viewport-fit=cover` - Safe area support (notch/Dynamic Island)
- Status bar styling

### 4. **Enhanced Media Session** (`useMediaSession.ts` composable)

Enables Apple's native music controls:

- **Lock Screen**: Track artwork, title, artist
- **Lock Screen Controls**: Play/Pause, Next, Previous buttons
- **Control Center**: Quick playback controls
- **Headphone Controls**: Use headphone play/pause button
- **Siri**: Voice control integration
- **Position Tracking**: Shows progress on lock screen

### 5. **PWA Features** (`usePWA.ts` composable)

- Service worker registration & updates
- Install prompt handling
- Detection of PWA installation status
- Update notifications

### 6. **App Initialization** (`useAppInitialization.ts`)

- Master composable that ties everything together
- Wake Lock API (keeps screen on while playing)
- iOS-specific touch optimizations

## Integration Steps

### Step 1: Update `app/layouts/default.vue`

Add these imports at the top of the `<script setup>` section:

```typescript
import { useAppInitialization } from "~/composables/useAppInitialization";
```

### Step 2: Initialize in `onMounted`

Add this to your `onMounted` hook in `app/layouts/default.vue` (around line 680-700):

```typescript
onMounted(async () => {
	await awaitAppState();

	const svc = new PlayerService(appState);
	player.value = svc;
	player.value.SetTopLevel(true);

	// ✅ ADD THIS:
	useAppInitialization(player.value);
	// END ADD

	// ... rest of initialization
});
```

### Step 3: Update Media Session Metadata

The Media Session API is already hooked into PlayerService. The new composable will automatically:

- Update lock screen artwork when tracks change
- Show track metadata (title, artist, album)
- Update position progress on lock screen
- Handle lock screen button presses

## How It Works on iOS

### For Local Playback (Scenario A - Primary)

1. **Add to Home Screen**
    - Open in Safari
    - Tap Share → Add to Home Screen
    - App launches in full-screen standalone mode

2. **Lock Screen Controls**
    - While playing, the lock screen shows:
        - Album artwork
        - Track title, artist, album
        - Play/Pause button
        - Next/Previous buttons
    - Use headphone buttons to play/pause
    - Use Control Center to control playback

3. **Siri Integration**
    - Say "Next track" or "Play/Pause" etc.

### For Remote Control (Scenario A - Secondary)

When controlling a remote device (via `/remote` page):

- Navigate to Remote → Select Device
- The app shows the remote device's currently playing track
- All Apple native controls work:
    - Lock screen shows what's playing on the remote device
    - Lock screen buttons send commands to the remote device
    - Control Center integrates with remote playback

## Required Icons

For full PWA experience, create these icon files in `public/`:

```
/public/
  ├── icon-192.png           (192x192 - app icon)
  ├── icon-192-maskable.png  (192x192 - maskable for iOS)
  ├── icon-512.png           (512x512 - larger icon)
  ├── icon-512-maskable.png  (512x512 - maskable for iOS)
  ├── screenshot-540.png     (540x720 - narrow screenshot)
  └── screenshot-1280.png    (1280x720 - wide screenshot)
```

### Icon Best Practices:

- Use solid colors in safe area (not edges)
- "Maskable" icons are cropped to a circle on iOS
- Put your logo in the center with padding

## Customization

### Status Bar Color

Edit `nuxt.config.ts` to change the status bar color:

```typescript
{ name: "apple-mobile-web-app-status-bar-style", content: "black-translucent" },
// Options: "default", "black", "black-translucent"
```

### Theme Colors

Edit `public/manifest.json`:

```json
{
	"theme_color": "#1a1a1a",
	"background_color": "#1a1a1a"
}
```

### App Name

The short name appears under the home screen icon:

```json
{
	"short_name": "Music", // Short name for home screen
	"name": "Music Player" // Full name
}
```

## Testing

### On macOS (Safari)

1. Develop → Experimental Features → Web App Manifest Debug Mode
2. Visit your app URL
3. You'll see install options

### On iOS

1. Open in Safari
2. Tap Share → Add to Home Screen
3. Launch from home screen
4. Lock device and play music to see lock screen controls

### On Android

1. Open in Chrome
2. Tap the menu → Install App
3. Lock device to test controls

## Troubleshooting

### Lock screen controls not working

- Ensure track info is being updated (title, artist, album)
- Check browser console for Media Session errors
- Verify audio element is properly initialized
- Lock screen requires metadata to be set first

### Icons not showing

- Icon files must be in `public/` directory
- Check file paths in `manifest.json`
- Ensure files are PNG format with proper dimensions
- Maskable icons must have content in center

### Wake Lock not working

- Only works on HTTPS (or localhost)
- Only activates when user interacts with the page
- Browser must support Wake Lock API

### Service Worker not registering

- Requires HTTPS (or localhost for development)
- Check browser console for registration errors
- Clear browser cache and try again
- Check that `/ui/sw.js` is accessible

## Browser Support

| Feature        | iOS Safari            | Android Chrome                 |
| -------------- | --------------------- | ------------------------------ |
| PWA Install    | ✅ Add to Home Screen | ✅ Install Prompt              |
| Media Session  | ✅ Lock Screen        | ✅ Lock Screen + Notifications |
| Wake Lock      | ✅ Limited            | ✅ Full                        |
| Service Worker | ✅                    | ✅                             |

## Notes

- iOS PWAs open in standalone mode (no browser UI)
- iOS has some limitations compared to native apps (no background audio by default)
- Local storage and IndexedDB work on iOS PWAs
- For persistent background playback, consider native app or web push

## Next Steps

1. Create the required icon files (192x192 and 512x512)
2. Add the integration code to `app/layouts/default.vue`
3. Test on iOS by adding to home screen
4. Customize colors and branding in `manifest.json`
5. Test lock screen controls while playing

## Remote Control Specifics

When using remote control mode:

1. The app connects to a remote music service
2. Player shows what's playing on the remote device
3. All controls (lock screen, headphones, etc.) send commands to the remote device
4. Metadata updates automatically from the remote device

This allows you to:

- Control your desktop music player from your iPhone lock screen
- Use Siri to control remote playback
- Use headphone buttons to skip tracks on the remote device
- See artwork and metadata from the remote device on your lock screen
