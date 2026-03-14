# PWA Icon Setup

Your manifest.json references these icon files. You need to create or add them to `/public/`:

## Required Icon Files

```
/public/
├── icon-192.png              (192x192 pixels)
├── icon-192-maskable.png     (192x192 pixels - for iOS)
├── icon-512.png              (512x512 pixels)
└── icon-512-maskable.png     (512x512 pixels - for iOS)
```

## Optional: Screenshots for App Installation

```
/public/
├── screenshot-540.png        (540x720 pixels - mobile)
└── screenshot-1280.png       (1280x720 pixels - tablet)
```

## How to Create Icons

### Option 1: Using ImageMagick (if installed)

```bash
# Create a 512x512 icon from your logo
convert your-logo.png -resize 512x512 public/icon-512.png

# Create 192x192 version
convert your-logo.png -resize 192x192 public/icon-192.png

# For maskable icons (same as regular for now)
cp public/icon-512.png public/icon-512-maskable.png
cp public/icon-192.png public/icon-192-maskable.png
```

### Option 2: Using Online Tools

- https://www.favicon-generator.org/
- https://convertio.co/
- https://cloudconvert.com/

### Option 3: Creating a Simple Icon

If you don't have a logo yet, you can create a simple colored square:

```bash
# Using Python PIL (if available)
python3 << 'EOF'
from PIL import Image, ImageDraw

# Create 512x512 icon
img = Image.new('RGB', (512, 512), color='#1a1a1a')
draw = ImageDraw.Draw(img)

# Draw a music note in the center
# Simple approximation of a music note
draw.rectangle([200, 150, 312, 350], fill='#ffffff')  # First note
draw.ellipse([150, 300, 250, 400], fill='#ffffff')    # First dot
draw.ellipse([200, 250, 300, 350], fill='#ffffff')    # Second dot

img.save('public/icon-512.png')
img.thumbnail((192, 192))
img.save('public/icon-192.png')

# Create maskable versions (same content)
img.save('public/icon-512-maskable.png')
img.load()
img.thumbnail((192, 192))
img.save('public/icon-192-maskable.png')
EOF
```

### Option 4: Using Your Existing Favicon

If you have a favicon, just copy it:

```bash
cp public/favicon.ico public/icon-192.png
cp public/favicon.ico public/icon-512.png
# etc...
```

## Icon Requirements

### For "maskable" Icons (iOS)

- Icons should have all important content in the center (safe area)
- Should work well when cropped to a circle
- Typically 20-40% padding around the edge

### For Regular Icons

- Should look good at any size
- Should be recognizable even as a 32x32 icon
- Square shape with transparent or solid background

## Testing

1. **Desktop (macOS/Linux)**:

    ```bash
    npm run dev  # Start your dev server
    ```

    - Visit `http://localhost:3000/ui/`
    - Check browser DevTools → Application → Manifest

2. **iOS**:
    - Open in Safari
    - Tap Share → Add to Home Screen
    - Icon appears on home screen

3. **Android**:
    - Open in Chrome
    - Menu → Install App (or see prompt)
    - Icon appears on home screen

## Manifest Configuration

Your `manifest.json` already references these icons:

```json
{
	"icons": [
		{
			"src": "/ui/icon-192.png",
			"sizes": "192x192",
			"type": "image/png",
			"purpose": "any"
		},
		{
			"src": "/ui/icon-192-maskable.png",
			"sizes": "192x192",
			"type": "image/png",
			"purpose": "maskable"
		}
		// ... etc for 512x512
	]
}
```

## Troubleshooting

### Icons don't show on home screen

- Ensure files are in `/public/` directory
- Check file paths match manifest.json exactly
- Files must be PNG format
- Try different icon sizes

### Icon looks blurry

- Ensure source image is high resolution
- Use 1:1 aspect ratio (square)
- Avoid upscaling small images

### Icon doesn't have right colors

- Check image color mode (should be RGB, not CMYK)
- Ensure background is solid or transparent
- Test with different browsers

## Optional: Custom Splash Screens

iOS generates splash screens automatically from your icon. Android uses the `screenshots` field in manifest.json.

If you want custom splash screens, create:

- `screenshot-540.png` (540×720 - narrow/mobile)
- `screenshot-1280.png` (1280×720 - wide/tablet)

## Next Steps

1. Create or provide your 192x192 and 512x512 icon files
2. Place them in `/public/`
3. Restart your dev server
4. Test on iOS: Share → Add to Home Screen
5. Test lock screen controls while playing music
