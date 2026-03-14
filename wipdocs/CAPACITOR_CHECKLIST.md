# ✅ Capacitor iOS App Implementation Checklist

Track your progress as you build your native iOS music player!

---

## 🟢 Phase 1: Initial Setup (30 min)

### Prerequisites
- [ ] macOS installed
- [ ] Xcode installed (App Store)
- [ ] Xcode Command Line Tools: `xcode-select --install`
- [ ] CocoaPods installed: `sudo gem install cocoapods`
- [ ] Node.js & npm working

**Time**: 10-15 min (mostly waiting for downloads)

### Installation
- [ ] Navigate to `frontend/` folder
- [ ] Install Capacitor:
  ```bash
  npm install @capacitor/core @capacitor/cli --save
  npm install @capacitor/ios --save-dev
  ```
- [ ] Initialize Capacitor: `npx cap init`
  - App name: `Music Player`
  - App ID: `com.yourname.musicplayer`
  - Directory: `dist`
  - URL: `http://localhost:5173`
- [ ] Build Nuxt: `npm run build`
- [ ] Add iOS: `npx cap add ios`

**Time**: 15-20 min

---

## 🟢 Phase 2: Configuration (15 min)

### Background Audio Setup
- [ ] Open `ios/App/App/Info.plist`
- [ ] Add to root dict:
  ```xml
  <key>UIBackgroundModes</key>
  <array>
    <string>audio</string>
  </array>
  ```
- [ ] Add audio default to speaker:
  ```xml
  <key>AVAudioDefaultToSpeaker</key>
  <true/>
  ```

**Reference**: See `ios-plist-config.xml` for full config

### Capacitor Config
- [ ] Check `capacitor.config.ts` is correct:
  ```typescript
  appId: 'com.yourname.musicplayer',
  appName: 'Music Player',
  webDir: 'dist',
  ```

**Time**: 5-10 min

---

## 🟢 Phase 3: First Run (20 min)

### Open in Xcode
- [ ] Run: `npx cap open ios`
- [ ] Xcode opens automatically

### Configure Signing
- [ ] In Xcode, select **App** target (left sidebar)
- [ ] Go to **Signing & Capabilities** tab
- [ ] Under "Team", select your Apple ID or team
- [ ] Xcode auto-provisions
- [ ] No errors in signing section ✅

**Time**: 3-5 min

### Run on Simulator
- [ ] In Xcode top bar, select **iPhone 15** (or similar)
- [ ] Click **▶️ Play** button
- [ ] Wait for build... (takes 1-3 min first time)
- [ ] Simulator launches app
- [ ] App loads with your UI ✅

**Or Terminal**:
```bash
npx cap run ios --target="iPhone 15"
```

**Time**: 5-10 min

---

## 🟢 Phase 4: Test Core Features (30 min)

### Lock Screen
- [ ] Play a track
- [ ] Lock device (Cmd+L in simulator)
- [ ] See album artwork ✅
- [ ] See track info (title, artist) ✅
- [ ] See play/pause button ✅

### Controls
- [ ] Tap play/pause on lock screen → works ✅
- [ ] Tap next → skips track ✅
- [ ] Tap previous → goes back ✅
- [ ] Progress bar shows position ✅

### Background Audio (THE KEY TEST!)
- [ ] Play music
- [ ] Switch to another app (Simulator: Cmd+H)
- [ ] **Music keeps playing!** ✅ (This is the win!)
- [ ] Come back to app
- [ ] Music still playing ✅

### Control Center
- [ ] Play music
- [ ] Swipe down from top-right
- [ ] Control Center shows ✅
- [ ] Controls work ✅

**Time**: 15-20 min testing

---

## 🟡 Phase 5: Code Integration (Optional)

### Add Background Audio Plugin (if needed)

```bash
npm install @capacitor-community/background-audio
npx cap sync ios
```

### Update Layout (optional)

Edit `app/layouts/default.vue`:

```typescript
import { BackgroundAudio } from '@capacitor-community/background-audio';

onMounted(async () => {
  // ... existing code ...

  // iOS background audio init
  try {
    await BackgroundAudio.init();
    console.log('Background audio ready');
  } catch (err) {
    console.warn('Background audio unavailable (web app):', err);
  }
});
```

**Time**: 5-10 min (optional)

---

## 🟡 Phase 6: Real Device Testing (30 min)

### Connect iPhone
- [ ] Plug iPhone via USB
- [ ] Unlock & tap "Trust"
- [ ] Select device in Xcode top bar
- [ ] Click **▶️ Play**
- [ ] App builds (2-5 min first time)
- [ ] App installs on device ✅

### Test on Device
- [ ] Play music
- [ ] Lock device
- [ ] Music keeps playing ✅
- [ ] Lock screen controls work ✅
- [ ] Switch apps - music continues ✅

**Time**: 20-30 min

---

## 🟡 Phase 7: Refinements (Variable)

### Icon Generation
```bash
npm install --save-dev @capacitor/assets
npx cap-assets generate --imageInputPath=./public/icon-512.png
```
- [ ] Icons generated ✅
- [ ] Rebuild and verify ✅

### Splash Screens
- [ ] Create splash screens (optional)
- [ ] Configure in `capacitor.config.ts`

### Environment Configuration

For production:
- [ ] Create `capacitor.production.config.ts`
- [ ] Update package.json build script
- [ ] Test production build

**Time**: 15-30 min

---

## 🟣 Phase 8: App Store Preparation (Before Submission)

### Developer Account
- [ ] Create Apple Developer Account: https://developer.apple.com
- [ ] Join Apple Developer Program ($99/year)
- [ ] Accept agreements

### App Store Connect
- [ ] Go to https://appstoreconnect.apple.com
- [ ] Create new app
- [ ] Fill in app details
- [ ] Create bundle ID: `com.yourname.musicplayer`

### Build for Archive
- [ ] `npm run build` (production)
- [ ] `npx cap sync ios`
- [ ] `npx cap open ios`
- [ ] In Xcode: Select "Generic iOS Device"
- [ ] Product → Archive
- [ ] Wait for build

### Distribute
- [ ] In Organizer: Click "Distribute"
- [ ] Select "App Store Connect"
- [ ] Follow upload wizard
- [ ] Sign with Apple ID

### App Review
- [ ] Apple reviews (usually 1-3 days)
- [ ] Check status in App Store Connect
- [ ] Once approved → appears in App Store!

**Time**: Variable (mostly waiting for Apple review)

---

## 🟣 Phase 9: Post-Launch (Ongoing)

### Updates
- [ ] Update code in Nuxt
- [ ] `npm run build`
- [ ] `npx cap sync ios`
- [ ] Rebuild & test
- [ ] Archive & submit new version

### Monitoring
- [ ] Check crash logs in App Store Connect
- [ ] Monitor user reviews
- [ ] Track downloads/ratings

### Maintenance
- [ ] Keep dependencies updated
- [ ] Test on new iOS versions
- [ ] Fix bugs reported by users

---

## 📊 Progress Summary

Print this and mark off as you go:

```
Phase 1: Setup              ☐ (30 min)
Phase 2: Configuration      ☐ (15 min)
Phase 3: First Run          ☐ (20 min)
Phase 4: Test Features      ☐ (30 min)
Phase 5: Code Integration   ☐ (10 min, optional)
Phase 6: Real Device        ☐ (30 min)
Phase 7: Refinements        ☐ (15-30 min, optional)
Phase 8: App Store          ☐ (Variable)
Phase 9: Post-Launch        ☐ (Ongoing)
```

**Total Time to App Store**: ~4-6 hours (plus Apple review time)

---

## 🆘 Common Issues & Fixes

| Issue | Solution |
|-------|----------|
| Build fails in Xcode | Clear cache: `rm -rf ~/Library/Developer/Xcode/DerivedData/*` |
| Changes don't appear | Run `npm run build && npx cap sync ios` |
| Can't sign | Xcode → Signing & Capabilities → Select Team |
| No background audio | Check `Info.plist` has `UIBackgroundModes` → `audio` |
| Simulator won't run | Try: `npx cap run ios --target="iPhone 15"` |
| "Podfile lock" error | Delete `ios/Podfile.lock` and rebuild |

---

## 📚 Reference Files

- **Setup Guide**: `CAPACITOR_SETUP.md` (detailed)
- **Quick Start**: `CAPACITOR_QUICK_START.md` (quick reference)
- **This File**: `CAPACITOR_CHECKLIST.md` (progress tracking)
- **iOS Config**: `ios-plist-config.xml` (Info.plist reference)

---

## 🎯 Success Criteria

Your iOS app is ready when:

- ✅ App runs on simulator
- ✅ App runs on real device
- ✅ Music plays in foreground
- ✅ Music continues in background
- ✅ Lock screen shows artwork & controls
- ✅ All buttons (play/pause/next/prev) work
- ✅ No console errors

---

## 🚀 You're Ready!

Start with Phase 1 and work your way through. Each phase builds on the previous one.

**Estimated total time**: 2-4 hours to get running, then ~1 week for App Store submission/approval.

Good luck! 🎵
