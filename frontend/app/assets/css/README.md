# Music App Theming

This directory contains the branded theme system for the music app.

## Files

- **theme.scss** - Main branded theme with color variables and component styling
- **site.css** - Legacy CSS for layout and positioning (colors removed to let SCSS take over)

## Theme Colors

### Primary Green Theme

- `$clr-primary-a0: #6db207` - Main brand green
- `$clr-primary-a10: #7fbb35` - Lighter green for hovers
- `$clr-primary-a20: #91c352` - Even lighter green for text links

### Surface Colors (Dark Theme)

- `$clr-surface-a0: #121212` - Main background
- `$clr-surface-a10: #282828` - Elevated surfaces
- `$clr-surface-a20: #3f3f3f` - Higher elevation

### Status Colors

- **Success:** `$clr-success-a0: #22946e`
- **Warning:** `$clr-warning-a0: #a87a2a`
- **Danger:** `$clr-danger-a0: #9c2121`
- **Info:** `$clr-info-a0: #21498a`

## Usage

The main app container uses the `.music-app` class which applies all the themed styling automatically.

### Utility Classes

```scss
.text-primary        // Primary green text
.text-success        // Success green text
.text-warning        // Warning amber text
.text-danger         // Danger red text
.text-muted          // Muted gray text

.bg-primary          // Primary green background
.bg-surface          // Main surface background
.bg-surface-elevated // Elevated surface background

.border-primary      // Primary green border
```

### CSS Custom Properties

Theme colors are also available as CSS custom properties:

```css
var(--clr-primary)
var(--clr-primary-light)
var(--clr-primary-dark)
var(--clr-surface-base)
var(--clr-surface-elevated)
var(--clr-surface-higher)
```

## Component Theming

All major UI components are automatically themed:

- Navigation bars
- Buttons
- Audio player
- Progress bars
- Track lists
- Tables
- Modals
- Forms
- Dropdowns
- Pagination
- Scrollbars

## Development Notes

- SCSS variables are defined in `theme.scss`
- Bulma framework variables are overridden with theme colors
- The `.music-app` wrapper class applies all component theming
- Legacy colors in `site.css` have been commented out to avoid conflicts
