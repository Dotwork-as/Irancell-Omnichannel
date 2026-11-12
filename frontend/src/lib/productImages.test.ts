import { describe, expect, it } from 'vitest';
import { readFileSync, statSync } from 'node:fs';
import path from 'node:path';

import { PRODUCT_IMAGES, productImageFor } from './productImages';

// vitest runs from frontend/ (npm script); jsdom rewrites import.meta.url
// to an http origin, so the project root must come from process.cwd().
const publicDir = path.resolve(process.cwd(), 'public');

/**
 * Deliberate reuse: the Figma kit's Silver render is
 * shared by the two blue-toned products because the exported set has no
 * blue render. Any file listed here MUST actually be shared by the
 * listed slugs, and no sharing outside this list is allowed.
 */
const SHARED_IMAGE_ALLOWLIST: Record<string, string[]> = {
  '/products/iphone-14-plus/iphone-14-pro-silver.png': [
    'iphone-14-plus',
    'iphone-15-pro',
  ],
};

describe('productImages', () => {
  it('returns null for products the Figma kit does not cover', () => {
    expect(productImageFor('not-a-product', 'not-a-color')).toBeNull();
    expect(productImageFor('iphone-15-pro-max', 'not-a-color')).toBeNull();
  });

  it('stores every mapped asset under products/ with a descriptive filename', () => {
    for (const [slug, colors] of Object.entries(PRODUCT_IMAGES)) {
      for (const [colorKey, src] of Object.entries(colors)) {
        expect(src, `${slug}/${colorKey}`).toMatch(
          /^\/products\/[a-z0-9-]+\/[a-z0-9-]+\.png$/,
        );
      }
    }
  });

  it('has a non-empty file on disk for every mapped asset', () => {
    for (const colors of Object.values(PRODUCT_IMAGES)) {
      for (const src of Object.values(colors)) {
        const abs = path.join(publicDir, src);
        expect(statSync(abs).size, src).toBeGreaterThan(0);
      }
    }
  });

  it('shares files only through the documented allowlist, never by accident', () => {
    const byDigest = new Map<string, Set<string>>();
    const bySrc = new Map<string, Set<string>>();
    for (const [slug, colors] of Object.entries(PRODUCT_IMAGES)) {
      for (const src of Object.values(colors)) {
        expect(src, `${slug} → ${src}`).toBeTruthy();
        const digest = readFileSync(path.join(publicDir, src)).toString(
          'base64',
        );
        byDigest.set(digest, (byDigest.get(digest) ?? new Set()).add(src));
        bySrc.set(src, (bySrc.get(src) ?? new Set()).add(slug));
      }
    }

    // distinct files must not be byte-identical with each other
    expect(byDigest.size).toBe(new Set(bySrc.keys()).size);

    for (const [src, slugs] of bySrc) {
      const allowed = SHARED_IMAGE_ALLOWLIST[src];
      if (allowed) {
        expect([...slugs].sort(), src).toEqual([...allowed].sort());
      } else {
        expect(slugs.size, `${src} used by ${[...slugs]}`).toBe(1);
      }
    }
    // every allowlisted share must actually exist in the map
    for (const [src, slugs] of Object.entries(SHARED_IMAGE_ALLOWLIST)) {
      for (const slug of slugs) {
        expect(Object.values(PRODUCT_IMAGES[slug] ?? {}), `${slug} → ${src}`)
          .toContain(src);
      }
    }
  });
});
