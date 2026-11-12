/**
 * Frontend-owned device imagery: the backend never sends colors or
 * image URLs (docs/API.md §4) — the UI maps the stable backend
 * `colorKey`/`slug` pair to the illustration tint. Provisional values
 * mirroring the catalog; tweak here when the design changes.
 */
export const COLOR_HEX: Record<string, Record<string, string>> = {
  'iphone-15-pro-max': {
    natural: '#bfa89a',
    blue: '#3c4a5a',
    black: '#3c3c3d',
    white: '#e8e8e3',
  },
  'iphone-15-pro': { blue: '#3c4a5a' },
  'iphone-15': { pink: '#d7c4c8' },
  'iphone-14-plus': { blue: '#8fa8c8' },
  'iphone-14': { black: '#3c3c3d' },
  'iphone-se-3': { white: '#e8e8e3' },
};

const FALLBACK = '#3c3c3d';

export function colorHexFor(slug: string, colorKey: string): string {
  return COLOR_HEX[slug]?.[colorKey] ?? FALLBACK;
}
