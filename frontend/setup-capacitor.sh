#!/bin/bash

# Capacitor iOS Setup Script
# This script automates the Capacitor setup for your Nuxt Music Player app

set -e

echo "🎵 Setting up Capacitor for iOS Music Player..."
echo ""

# Step 1: Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v node &> /dev/null; then
  echo "❌ Node.js not found. Please install Node.js first."
  exit 1
fi

if ! command -v npm &> /dev/null; then
  echo "❌ npm not found. Please install npm first."
  exit 1
fi

echo "✅ Node.js and npm found"
echo ""

# Step 2: Install Capacitor
echo "📦 Installing Capacitor packages..."
npm install @capacitor/core @capacitor/cli --save
npm install @capacitor/ios --save-dev

echo "✅ Capacitor installed"
echo ""

# Step 3: Initialize Capacitor
echo "⚙️  Initializing Capacitor..."
echo ""
echo "You'll be asked for:"
echo "  App name: Music Player"
echo "  App ID: com.yourname.musicplayer (use reverse domain like com.example.musicplayer)"
echo "  Directory: dist"
echo "  URL: http://localhost:5173"
echo ""

npx cap init

echo ""
echo "✅ Capacitor initialized"
echo ""

# Step 4: Build Nuxt
echo "🔨 Building Nuxt app..."
npm run build
echo "✅ Nuxt built"
echo ""

# Step 5: Add iOS platform
echo "🍎 Adding iOS platform..."
npx cap add ios
echo "✅ iOS platform added"
echo ""

# Step 6: Install plugins
echo "📚 Installing Capacitor plugins..."
npm install @capacitor-community/background-audio
npm install @capacitor/app
npm install @capacitor/screen-reader

# Sync plugins
npx cap sync ios
echo "✅ Plugins installed and synced"
echo ""

# Step 7: Show next steps
echo "🎉 Capacitor setup complete!"
echo ""
echo "📝 Next steps:"
echo ""
echo "1. Open iOS project in Xcode:"
echo "   npx cap open ios"
echo ""
echo "2. Configure signing (in Xcode):"
echo "   - Select App target"
echo "   - Go to Signing & Capabilities"
echo "   - Select your Apple Developer team"
echo ""
echo "3. Run on simulator:"
echo "   npx cap run ios --target=\"iPhone 15\""
echo ""
echo "4. Or run on device:"
echo "   - Connect iPhone"
echo "   - In Xcode, select your device"
echo "   - Click the ▶️ Play button"
echo ""
echo "📖 Read CAPACITOR_SETUP.md for detailed instructions"
echo ""
echo "Happy coding! 🚀"
