# Tray action icons

The refresh and settings buttons use Lucide's **refresh-cw** and **settings**
icons. Their SVG geometry is unchanged.

- Project: https://lucide.dev/
- Refresh: https://lucide.dev/icons/refresh-cw
- Settings: https://lucide.dev/icons/settings
- Upstream revision: `7b0196e249ca09ca27c584c497aca9a8cab8a7f5`
- Sources: `https://github.com/lucide-icons/lucide/tree/7b0196e249ca09ca27c584c497aca9a8cab8a7f5/icons`
- License: [ISC, including upstream third-party notices](LICENSE)

The transparent 128 × 128 PNGs are generated from the SVGs with
`node scripts/render-tray-action-icons.cjs` from the repository root. The script
requires `sharp` (or `SHARP_MODULE_PATH`) during development only. The app embeds
the PNG masks and license, scales the masks for the display DPI, and applies the
button's foreground/background colors. Building or running the app does not
require Node.js, an SVG library, network access, or an icon font.
