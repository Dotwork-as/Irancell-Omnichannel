import { ApiListResponse } from "./api/types/shared";

/**
 * Shortens a long string for UI display
 * Example: 0x1234…abcd
 */
export function shortenAddress(address: string, start = 6, end = 4): string {
  if (!address) return '';

  if (address.length <= start + end) return address;

  return `${address.slice(0, start)}\u2026${address.slice(-end)}`;
}

/**
 * Converts ISO timestamp to relative time (e.g. "5 minutes ago")
 */
export function getRelativeTime(timestamp: string): string {
  if (!timestamp) return '';

  const date = new Date(timestamp);
  const now = new Date();

  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return 'Just now';

  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} minute${diffMin > 1 ? 's' : ''} ago`;

  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} hour${diffHour > 1 ? 's' : ''} ago`;

  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay} day${diffDay > 1 ? 's' : ''} ago`;
}

export function normalizeListResponse<T>(
  response: T[] | ApiListResponse<T>,
): T[] {
  return Array.isArray(response) ? response : response.items ?? [];
}
