/** Resolves a design token from styles.css (e.g. `--color-info`) for APIs that need a literal color, such as Leaflet and SweetAlert. */
export function themeColor(token: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(token).trim();
}
