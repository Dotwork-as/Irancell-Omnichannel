/**
 * Frontend-owned product imagery map, sourced from the Figma design
 * ("E-Commerce UI Kit (Community)", file JKFo1aAAUL186rx2Uh7dRS) — the
 * backend never sends image paths (docs/API.md §4). Exports of the
 * Figma kit supply one render per product; each entry maps
 * the stable backend `slug`/`colorKey` pair to that local asset under
 * `public/products/{slug}/`. PROVISIONAL mapping: renders are named
 * for the device they actually show — the kit imagery does not cover
 * every catalog color, so each product carries its closest kit
 * render. The kit's Silver render is shared by the two blue-toned
 * products (deliberate reuse from the exported set; see the test's
 * SHARED_IMAGE_ALLOWLIST). Keep this
 * map in sync with `src/lib/productImages.test.ts`, which asserts the
 * referenced files exist on disk.
 */
export const PRODUCT_IMAGES: Record<string, Record<string, string>> = {
  'iphone-15-pro-max': {
    natural: '/products/iphone-15-pro-max/iphone-14-pro-gold.png',
    blue: '/products/iphone-15-pro-max/iphone-14-pro-gold.png',
    black: '/products/iphone-15-pro-max/iphone-14-pro-gold.png',
    white: '/products/iphone-15-pro-max/iphone-14-pro-gold.png',
  },
  'iphone-15-pro': {
    blue: '/products/iphone-14-plus/iphone-14-pro-silver.png',
  },
  'iphone-15': {
    pink: '/products/iphone-15/iphone-13-pink.png',
  },
  'iphone-14-plus': {
    blue: '/products/iphone-14-plus/iphone-14-pro-silver.png',
  },
  'iphone-14': {
    black: '/products/iphone-14/iphone-14-pro-space-black.png',
  },
  'iphone-se-3': {
    white: '/products/iphone-se-3/iphone-11-white.png',
  },
};

/**
 * Image path for a product/color, or null when the Figma kit provides
 * no genuine image for that pair (renderers fall back to vector art).
 */
export function productImageFor(
  slug: string,
  colorKey: string,
): string | null {
  return PRODUCT_IMAGES[slug]?.[colorKey] ?? null;
}
