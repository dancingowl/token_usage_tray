// Regenerate the embedded tray action masks: node scripts/render-tray-action-icons.cjs
// Requires sharp for development only; optionally set SHARP_MODULE_PATH.
'use strict';
const fs = require('node:fs/promises');
const path = require('node:path');
const sharp = require(process.env.SHARP_MODULE_PATH || 'sharp');

async function main() {
  const directory = path.resolve(__dirname, '../vendor/fyne.io/systray/icons');
  for (const name of ['refresh-cw', 'settings']) {
    const source = await fs.readFile(path.join(directory, `${name}.svg`), 'utf8');
    // Keep upstream geometry, stroke width, rounded caps, and rounded joins.
    // Only alpha is used at runtime, so the source is rendered in black.
    const svg = Buffer.from(source.replaceAll('currentColor', '#000000'));
    await sharp(svg, { density: 384 })
      .resize(128, 128)
      .png()
      .toFile(path.join(directory, `${name}.png`));
    console.log(`${name}.png: 128 × 128 transparent mask`);
  }
}

main().catch(error => { console.error(error); process.exitCode = 1; });
