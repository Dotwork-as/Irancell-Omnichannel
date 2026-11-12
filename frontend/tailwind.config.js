/**
 * Tailwind v4 runs CSS-first: the tokens in `src/styles/globals.css`
 * (`@theme`) are the single source of truth and generate the utilities.
 * This file only mirrors them for tooling that reads a JS config; keep the
 * two in sync when a token changes.
 */

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/app/**/*.{js,ts,jsx,tsx}',
    './src/components/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        ink: '#0C0C0C',
        paper: '#FFFFFF',
        brand: '#FFBE00',
        coal: '#2E2E2E',
        ash: '#979797',
        line: '#E4E4E4',
        smoke: '#F5F5F5',
      },
    },
  },
  plugins: [],
};
