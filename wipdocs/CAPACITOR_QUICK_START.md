# 🚀 Capacitor iOS App - Quick Start

Get your native iOS app running in **5 easy steps**!

## ⚡ Prerequisites (5 min)

- [ ] **macOS** (Intel or Apple Silicon)
- [ ] **Xcode** (free from App Store)
- [ ] **CocoaPods** (install via: `sudo gem install cocoapods`)

Verify:
```bash
xcode-select --install          # Install Xcode Command Line Tools
pod --version                    # Check CocoaPods (should see version number)
```

---

## 🛠️ Setup Steps (30 min total)

### Step 1: Install Capacitor (5 min)

Go to `frontend/` folder:
```bash
cd /workspace/playground/web/music/frontend

npm install @capacitor/core @capacitor/cli --save
npm install @capacitor/ios --save-dev
```

### Step 2: Initialize Capacitor (2 min)

```bash
npx cap init
```

When prompted:
- **App name**: `Music Player`
- **App ID**: `com.yourname.musicplayer` ← Use your domain!
- **Directory**: `dist`
- **URL**: `http://localhost:5173`

✅ Creates: `capacitor.config.ts`

### Step 3: Build Your App (3 min)

```bash
npm run build
```

✅ Creates: `dist/` folder with your compiled app

### Step 4: Add iOS Platform (10 min)

```bash
npx cap add ios
```

✅ Creates: `ios/` folder (your Xcode project!)

### Step 5: Open in Xcode (5 min)

```bash
npx cap open ios
```

✅ Xcode opens with your iOS project ready!

---

## ✅ Configure Signing (2 min)

In Xcode:

1. **Left sidebar**: Select **App** target
2. **Top menu**: Go to **Signing & Capabilities** tab
3. **Team dropdown**: Select your Apple ID / team
4. ✅ Let Xcode auto-sign

---

## 🏃 Run on Simulator (1 min)

Terminal:
```bash
npx cap run ios --target="iPhone 15"
```

Or in Xcode:
- Top bar: Select **iPhone 15** simulator
- Click ▶️ **Play button**
- App builds and launches!

---

## 📱 Run on Real Device (1 min)

1. **Plug in iPhone** via USB
2. **Unlock** and tap "Trust" if prompted
3. In Xcode:
   - Top bar: Select your iPhone
   - Click ▶️ **Play button**
4. App installs and launches!

---

## ✨ Test Background Audio

1. **Play** music in app
2. **Lock** device or **switch** to another app
3. **Music keeps playing!** 🎵
4. **Lock screen** shows artwork & controls
5. **Tap controls** to play/pause/skip

---

## 📝 After Setup

### Update Your Code

Edit: `app/layouts/default.vue`

Add to imports:
```typescript
import { App as CapApp } from '@capacitor/app';
import { BackgroundAudio } from '@capacitor-community/background-audio';
```

In `onMounted()`:
```typescript
// Initialize background audio (iOS)
try {
  await BackgroundAudio.init();
} catch (err) {
  console.warn('Background audio not available');
}
```

### Rebuild After Changes

```bash
npm run build           # Build Nuxt
npx cap sync ios      # Update iOS project
# Then rebuild in Xcode (Cmd+R)
```

---

## 🐛 Troubleshooting

| Problem | Solution |
|---------|----------|
| Build fails | Clear cache: `rm -rf ~/Library/Developer/Xcode/DerivedData/*` |
| Changes not showing | Run `npm run build && npx cap sync ios` first |
| Can't sign | Go to Xcode → Signing & Capabilities → Select team |
| No background audio | Verify `ios/App/App/Info.plist` has `UIBackgroundModes` → `audio` |
| Simulator won't start | Try: `npx cap run ios --target="iPhone 15"` |

---

## 📚 Full Documentation

Read the complete guide: [CAPACITOR_SETUP.md](./CAPACITOR_SETUP.md)

---

## 🎯 What's Next

After getting it running:

### Test on Real Device
- [ ] Build for physical iPhone
- [ ] Test background audio
- [ ] Test lock screen controls
- [ ] Test remote control feature

### App Store (Later)
- [ ] Create Apple Developer account ($99/year)
- [ ] Generate app icons (512x512 PNG)
- [ ] Register app in App Store Connect
- [ ] Submit for review
- [ ] Get published! 🎉

---

## 💡 Pro Tips

**Hot Reload Development** (fastest):
```bash
npm run dev              # Terminal 1: Keep running
npx cap open ios        # Terminal 2: Open Xcode
# App auto-refreshes as you code!
```

**Build for Testing**:
```bash
npm run build && npx cap sync ios && npx cap run ios
```

**Clean Everything**:
```bash
rm -rf dist ios node_modules
npm install
npm run build
npx cap add ios
```

---

## ❓ FAQ

**Q: Can I still use the web version?**
A: Yes! Your web PWA still works. Now you just have an iOS app too.

**Q: Do I need to know Swift?**
A: No! Capacitor handles all the native code. You just use JavaScript/Vue.

**Q: How do I update the app on devices?**
A: Just rebuild and redistribute via App Store, TestFlight, or enterprise distribution.

**Q: What about Android?**
A: Same code works! Just run `npx cap add android` and follow same steps.

---

## 🚀 You're Ready!

```bash
npm run build
npx cap open ios
# Click play button in Xcode
# Your iOS app launches! 🎵
```

Enjoy building! 🎉
