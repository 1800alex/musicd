# 📱 Capacitor iOS App Setup Guide

Convert your Nuxt PWA to a native iOS app with Capacitor. Reuse 95% of your code and add background audio support!

## What You'll Get

- ✅ Native iOS app that runs your Nuxt code
- ✅ Background audio playback (the main win!)
- ✅ Full Media Session API support
- ✅ App Store ready
- ✅ Same codebase for web + iOS
- ✅ Easy updates via web version

## Prerequisites

- **macOS** (required for iOS development)
- **Xcode** (free, from App Store)
- **Node.js** & npm/yarn (you already have)
- **CocoaPods** (for iOS dependencies)
- **Apple Developer Account** (free, $99/year for App Store)

## Step 1: Install Capacitor

In your project root (`/workspace/playground/web/music/frontend`):

```bash
npm install @capacitor/core @capacitor/cli
npm install -D @capacitor/ios

# Or with yarn
yarn add @capacitor/core @capacitor/cli
yarn add -D @capacitor/ios
```

## Step 2: Initialize Capacitor

```bash
npx cap init
```

Answer the prompts:
- **App name**: `Music Player`
- **App ID**: `com.yourname.musicplayer` (e.g., `com.example.musicplayer`)
  - ⚠️ Important: Use reverse domain naming (com.yourname.appname)
  - This must match your Apple Developer ID later
- **Directory**: `dist` (Nuxt builds to dist/)
- **URL**: `http://localhost:5173` (for dev, will be localhost for production)

Creates: `capacitor.config.ts`

## Step 3: Build Your Nuxt App

```bash
npm run build
```

This generates the `dist/` folder that Capacitor will wrap.

## Step 4: Add iOS Platform

```bash
npx cap add ios
```

This creates the `ios/` folder with the Xcode project.

## Step 5: Install iOS Plugins

Add plugins for background audio and other features:

```bash
npm install @capacitor-community/background-audio
npm install @capacitor/media-session
npm install @capacitor/app

npx cap sync ios
```

### What These Do:
- **background-audio**: Continues playing when app goes to background
- **media-session**: Enhanced lock screen controls (already have this via web API)
- **app**: App lifecycle management

## Step 6: Configure Background Audio

Create/edit `ios/App/App/Info.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<!-- ... existing config ... -->

	<!-- Add these capabilities -->
	<key>UIBackgroundModes</key>
	<array>
		<string>audio</string>
	</array>

	<!-- Allow background audio playback -->
	<key>AVAudioDefaultToSpeaker</key>
	<true/>

	<!-- Microphone permissions -->
	<key>NSMicrophoneUsageDescription</key>
	<string>Music Player uses microphone for voice control</string>

</dict>
</plist>
```

## Step 7: Update Capacitor Config

Edit `capacitor.config.ts`:

```typescript
import { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.yourname.musicplayer',
  appName: 'Music Player',
  webDir: 'dist',
  server: {
    androidScheme: 'https',
    iosPlatform: 'localhost'
  },
  plugins: {
    'BackgroundAudio': {
      enabled: true
    }
  }
};

export default config;
```

## Step 8: Update for Production

Create `capacitor.production.config.ts`:

```typescript
import { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.yourname.musicplayer',
  appName: 'Music Player',
  webDir: 'dist',
  server: {
    url: 'https://yourdomain.com/ui/',  // Production URL
    cleartext: false
  },
  plugins: {
    'BackgroundAudio': {
      enabled: true
    }
  }
};

export default config;
```

Then add to `package.json`:

```json
{
  "scripts": {
    "build": "nuxt build",
    "build:prod": "CAPACITOR_ENV=production nuxt build && npx cap sync ios"
  }
}
```

## Step 9: Initialize Native Audio Handler

Update `app/layouts/default.vue` to use Capacitor audio:

```typescript
import { BackgroundAudio } from '@capacitor-community/background-audio';

// In your player initialization
onMounted(async () => {
  // ... existing code ...

  // Initialize background audio for iOS
  if (typeof window !== 'undefined') {
    try {
      await BackgroundAudio.init();
      console.log('Background audio initialized');
    } catch (err) {
      console.warn('Background audio not available (web app):', err);
    }
  }
});
```

## Step 10: Open in Xcode

```bash
npx cap open ios
```

This opens Xcode with your iOS project ready to build/run.

## Step 11: Configure Signing (First Time)

In Xcode:

1. Select the **App** target
2. Go to **Signing & Capabilities**
3. Select your **Team** (your Apple Developer account)
4. Let Xcode auto-manage provisioning

## Step 12: Run on Simulator or Device

**Simulator** (free):
```bash
npx cap run ios --target="iPhone 15"
```

**Physical Device**:
1. Connect iPhone via USB
2. In Xcode, select your device from top bar
3. Click the ▶️ Play button
4. App builds and installs!

## Step 13: Rebuild After Changes

When you update your Nuxt code:

```bash
npm run build          # Build Nuxt
npx cap sync ios      # Sync to iOS
npx cap open ios      # Open Xcode (if needed)
```

Or in Xcode, just press Cmd+R to rebuild.

---

## Testing Background Audio

1. Launch app on iPhone
2. Play music
3. Lock the device or switch apps
4. **Music should keep playing!** 🎵
5. Lock screen shows artwork & controls

---

## Before App Store Submission

### Create Icons & Splash Screens

Capacitor needs iOS-specific assets:

```bash
npm install --save-dev @capacitor/assets
npx cap-assets generate --imageInputPath=./public/icon-512.png
```

This generates all required icon sizes automatically.

### Configure App.xcconfig

Edit `ios/App/App/App.xcconfig`:

```
// Add these
DEVELOPMENT_TEAM = YOUR_TEAM_ID
CODE_SIGN_IDENTITY = iPhone Developer
```

### Update Info.plist

Ensure these are set:
```xml
<key>CFBundleDisplayName</key>
<string>Music Player</string>

<key>CFBundleShortVersionString</key>
<string>1.0.0</string>

<key>CFBundleVersion</key>
<string>1</string>
```

---

## Deploy to App Store

### 1. Create App in App Store Connect

1. Go to https://appstoreconnect.apple.com
2. Click "My Apps"
3. Click "+" and "New App"
4. Fill in details

### 2. Build for Archive

In Xcode:
1. Select "Generic iOS Device" (top bar)
2. Product → Archive
3. Wait for build to complete
4. Organizer window opens

### 3. Distribute

1. Click "Distribute App"
2. Select "App Store Connect"
3. Follow wizard
4. App uploads to Apple for review

### 4. Review & Release

- Apple reviews (1-3 days typically)
- App appears in App Store
- Users can install!

---

## Troubleshooting

### Background audio not working

```typescript
// Make sure this is in Info.plist
<key>UIBackgroundModes</key>
<array>
  <string>audio</string>
</array>
```

### Build errors

```bash
# Clear Xcode cache
rm -rf ~/Library/Developer/Xcode/DerivedData/*

# Rebuild
npx cap sync ios
npx cap open ios
```

### Changes not reflecting

```bash
# Always rebuild Nuxt first
npm run build

# Then sync
npx cap sync ios

# Refresh in Xcode (Cmd+Shift+K to clean)
```

### Signing issues

```bash
# Update signing
npx cap open ios

# In Xcode:
# 1. Select App target
# 2. Signing & Capabilities
# 3. Select your team
# 4. Xcode auto-provisions
```

---

## Development Workflow

### Hot Reload (Fastest Development)

```bash
# Terminal 1: Watch Nuxt
npm run dev

# Terminal 2: Open iOS project
npx cap open ios

# In Xcode: Keep running, it auto-refreshes from dev server
```

### Building for Testing

```bash
npm run build
npx cap sync ios
npx cap run ios
```

### Building for Distribution

```bash
npm run build:prod
npx cap sync ios
npx cap open ios
# Then Product → Archive in Xcode
```

---

## Project Structure After Setup

```
/workspace/playground/web/music/
├── frontend/
│   ├── app/              # Your Nuxt app (unchanged)
│   ├── dist/             # Built Nuxt (generated)
│   ├── ios/              # ← iOS project (new!)
│   ├── node_modules/
│   ├── capacitor.config.ts
│   └── package.json
└── ...
```

---

## Next Steps

1. ✅ Install Capacitor & iOS platform
2. ✅ Configure background audio in Info.plist
3. ✅ Generate app icons
4. ✅ Test on simulator
5. ✅ Test on physical device
6. ✅ Create Apple Developer account
7. ✅ Submit to App Store

---

## Resources

- [Capacitor Docs](https://capacitorjs.com/docs)
- [Capacitor iOS](https://capacitorjs.com/docs/ios)
- [Background Audio Plugin](https://github.com/jepiqueau/capacitor-background-audio)
- [App Store Connect](https://appstoreconnect.apple.com)
- [Xcode Guide](https://developer.apple.com/xcode/)

---

## Example Commands Quick Reference

```bash
# Setup
npm install @capacitor/core @capacitor/cli @capacitor/ios
npx cap init
npm run build
npx cap add ios

# Development
npm run dev
npx cap open ios

# Testing
npx cap run ios --target="iPhone 15"

# Updates
npm run build && npx cap sync ios

# Distribution
npm run build:prod && npx cap sync ios && npx cap open ios
```

---

## Questions?

- `capacitor.config.ts` - Check docs for config options
- iOS errors - Check Xcode console (bottom right)
- Background audio - Verify Info.plist settings
- App Store - Check App Store Connect docs
