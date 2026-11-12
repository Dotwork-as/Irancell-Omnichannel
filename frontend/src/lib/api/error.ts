import { errorEnvelopeSchema } from './types/schemas';

/**
 * Normalized API error: every service rejects with this shape so
 * hooks and components handle one predictable type (docs/API.md §3).
 */
export class ApiError extends Error {
  /** HTTP status; 0 when the request never reached the server. */
  readonly status: number;
  /** Machine-readable §4.3 code (e.g. STOCK_UNAVAILABLE). */
  readonly code: string;
  /** §4.3 details payload (per-field validation, stock conflicts). */
  readonly details: unknown;

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }

  /** Validation detail entries, when the backend sent that shape. */
  validationDetails(): { field: string; code: string; message: string }[] {
    if (!Array.isArray(this.details)) return [];
    return this.details.filter(
      (d): d is { field: string; code: string; message: string } =>
        typeof d === 'object' &&
        d !== null &&
        typeof (d as Record<string, unknown>).field === 'string' &&
        typeof (d as Record<string, unknown>).code === 'string' &&
        typeof (d as Record<string, unknown>).message === 'string',
    );
  }

  static is(value: unknown): value is ApiError {
    return value instanceof ApiError;
  }
}

/** Raw rejection shape produced by the axios response interceptor. */
interface interceptorRejection {
  status?: number;
  data?: unknown;
  message?: string;
}

/** Best-effort read of `error.message` from an unparseable body. */
function rawEnvelopeMessage(data: unknown): string | undefined {
  if (typeof data !== 'object' || data === null) return undefined;
  const error = (data as Record<string, unknown>).error;
  if (typeof error !== 'object' || error === null) return undefined;
  const message = (error as Record<string, unknown>).message;
  return typeof message === 'string' ? message : undefined;
}

/**
 * Normalizes any thrown value (interceptor rejection, network error,
 * programming mistake) into an ApiError without leaking internals.
 */
export function normalizeApiError(thrown: unknown): ApiError {
  if (ApiError.is(thrown)) return thrown;

  const rejection = thrown as interceptorRejection | undefined;
  const status = typeof rejection?.status === 'number' ? rejection.status : 0;
  const rawMessage = rawEnvelopeMessage(rejection?.data);

  const parsed = errorEnvelopeSchema.safeParse(rejection?.data);
  if (parsed.success) {
    const message = parsed.data.error.message || rawMessage || '';
    return new ApiError(status, parsed.data.error.code, message, parsed.data.error.details);
  }

  if (status === 0) {
    return new ApiError(0, 'NETWORK_ERROR', rawMessage ?? 'No response from server');
  }
  return new ApiError(status, 'UNKNOWN', rawMessage ?? 'Unexpected error');
}

