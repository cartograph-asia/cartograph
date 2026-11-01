# Map fonts

Glyph PBFs for the base style's labels (`web/style.json` → `"glyphs": "/fonts/{fontstack}/{range}.pbf"`), served with the frontend so the map renders without any third-party host.

- Fonts: Noto Sans Regular and Noto Sans Medium, all 256 Unicode ranges (most are empty placeholders, but MapLibre needs a 200 for every range it asks for).
- Source: https://github.com/protomaps/basemaps-assets (`fonts/`), downloaded 2026-11-01.
- Licence: SIL Open Font License 1.1, see [OFL.txt](OFL.txt).

To add a font, copy its directory from the same repository and reference its exact directory name in a `text-font` entry.
