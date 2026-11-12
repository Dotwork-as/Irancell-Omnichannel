import axios, { AxiosError } from 'axios';

import { defaultLocale, localeCookieName } from '@/src/i18n/config';

/**
 * Centralized API client (docs/API.md §1/§5): every request carries
 * credentials (the guest_sid cookie) and Accept-Language derived from
 * the same locale cookie that drives next-intl, so UI language and
 * API localization never diverge.
 */
const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080/api/v1',
  timeout: 10000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

/** Reads the next-intl locale cookie (client-side only). */
function currentLocale(): string {
  if (typeof document === 'undefined') return defaultLocale;
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${localeCookieName}=([^;]*)`),
  );
  const value = match?.[1];
  return value === 'en' || value === 'fa' ? value : defaultLocale;
}

api.interceptors.request.use((config) => {
  config.headers.set('Accept-Language', currentLocale());
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response) {
      return Promise.reject({
        status: error.response.status,
        data: error.response.data,
        message: error.message,
      });
    }

    if (error.request) {
      return Promise.reject({
        status: 0,
        message: 'No response from server',
      });
    }

    return Promise.reject({
      status: 0,
      message: error.message,
    });
  },
);

export default api;
