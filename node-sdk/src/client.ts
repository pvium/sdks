import { RequestOptions } from "./types";

export interface PviumSdkConfig {
  baseUrl?: string;
  apiKey?: string;
  clientId?: string;
  environment?: keyof typeof PVIUM_BASE_URLS | null;
  consentHost?: string;
  timeoutMs?: number;
  fetchFn?: typeof fetch;
  defaultHeaders?: Record<string, string>;
  logging?: {
    requests?: boolean;
    logger?: Pick<Console, 'debug' | 'error' | 'log' | 'warn'>;
  };
}

export interface HttpRequestConfig {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  query?: Record<string, string | number | boolean | undefined | null>;
  body?: unknown;
  options?: RequestOptions;
}

export class PviumApiError extends Error {
  public readonly status: number;
  public readonly statusText: string;
  public readonly body: unknown;

  constructor(params: { status: number; statusText: string; body: unknown }) {
    super(getPviumApiErrorMessage(params));
    this.name = "PviumApiError";
    this.status = params.status;
    this.statusText = params.statusText;
    this.body = params.body;
  }
}

export const PVIUM_BASE_URLS = {
  test: "http://localhost:4005/v1",
  sandbox: "https://api-sandbox.pvium.com/v1",
  production: "https://api.pvium.com/v1",
} as const;

export const PVIUM_CONSENT_HOSTS = {
  test: 'http://localhost:3000',
  sandbox: 'https://sandbox.pvium.com',
  production: 'https://pvium.com',
} as const;

const DEFAULT_BASE_URL = PVIUM_BASE_URLS.production;
const DEFAULT_CONSENT_HOST = PVIUM_CONSENT_HOSTS.production;

export function resolvePviumBaseUrl(config: PviumSdkConfig): string {
  const environment = config.environment ?? "production";
  return (
    config.baseUrl ||
    PVIUM_BASE_URLS[environment] ||
    DEFAULT_BASE_URL
  ).replace(/\/$/, "");
}

export function resolvePviumConsentHost(config: PviumSdkConfig): string {
  const environment = config.environment ?? "production";
  return (
    config.consentHost ||
    PVIUM_CONSENT_HOSTS[environment] ||
    DEFAULT_CONSENT_HOST
  ).replace(/\/$/, "");
}

export class PviumHttpClient {
  private readonly baseUrl: string;
  private readonly timeoutMs: number;
  private readonly fetchFn: typeof fetch;
  private apiKey?: string;
  private readonly defaultHeaders: Record<string, string>;
  private readonly logRequests: boolean;
  private readonly logger: Pick<Console, 'debug' | 'error' | 'log' | 'warn'>;

  constructor(config: PviumSdkConfig) {
    this.baseUrl = resolvePviumBaseUrl(config);
    this.timeoutMs = config.timeoutMs ?? 30000;
    this.fetchFn =
      config.fetchFn ?? ((input, init) => globalThis.fetch(input, init));
    this.apiKey = config.apiKey;
    this.defaultHeaders = config.defaultHeaders ?? {};
    this.logRequests = Boolean(config.logging?.requests);
    this.logger = config.logging?.logger ?? console;
  }

  setApiKey(key?: string): void {
    this.apiKey = key;
  }

  async request(config: HttpRequestConfig): Promise<Response> {
    const url = this.buildUrl(config.path, config.query);
    const headers: Record<string, string> = {
      Accept: 'application/json',
      ...this.defaultHeaders,
      ...config.options?.headers,
    };

    if (config.options?.accessToken) {
      headers.Authorization = `Bearer ${config.options.accessToken}`;
    } else {
      const apiKey = config.options?.apiKey ?? this.apiKey;
      if (apiKey && !config.options?.skipApiKey) {
        headers['x-api-key'] = apiKey;
      }
    }

    if (config.body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);
    const startedAt = Date.now();
    const requestLog = this.createRequestLog(config.method, url);

    if (this.logRequests) {
      this.logger.debug('[pvium-sdk] request', requestLog);
    }

    try {
      const response = await this.fetchFn(url, {
        method: config.method,
        headers,
        body:
          config.body !== undefined ? JSON.stringify(config.body) : undefined,
        signal: config.options?.signal ?? controller.signal,
      });
      if (this.logRequests) {
        this.logger.debug('[pvium-sdk] response', {
          ...requestLog,
          status: response.status,
          ok: response.ok,
          durationMs: Date.now() - startedAt,
        });
      }
      if (!response.ok) {
        throw new PviumApiError({
          status: response.status,
          statusText: response.statusText,
          body: await this.parseResponseBody<unknown>(response),
        });
      }
      return response;
    } catch (error) {
      if (this.logRequests) {
        this.logger.error('[pvium-sdk] request failed', {
          ...requestLog,
          durationMs: Date.now() - startedAt,
          error: this.serializeError(error),
        });
      }

      throw error;
    } finally {
      clearTimeout(timeout);
    }
  }

  private createRequestLog(method: HttpRequestConfig['method'], url: string) {
    const parsed = new URL(url);

    return {
      method,
      protocol: parsed.protocol,
      host: parsed.host,
      pathname: parsed.pathname,
      hasQuery: parsed.search.length > 0,
      timeoutMs: this.timeoutMs,
    };
  }

  private serializeError(error: unknown): unknown {
    if (this.isAggregateErrorLike(error)) {
      return {
        name: error.name,
        message: error.message,
        errors: error.errors.map((cause) => this.serializeError(cause)),
      };
    }

    if (error instanceof Error) {
      const errorWithDetails = error as Error & {
        code?: string;
        cause?: unknown;
      };

      return {
        name: error.name,
        message: error.message,
        code: errorWithDetails.code,
        cause:
          errorWithDetails.cause === undefined
            ? undefined
            : this.serializeError(errorWithDetails.cause),
      };
    }

    return error;
  }

  private isAggregateErrorLike(error: unknown): error is {
    name?: string;
    message?: string;
    errors: unknown[];
  } {
    return (
      Boolean(error) &&
      typeof error === 'object' &&
      Array.isArray((error as { errors?: unknown }).errors)
    );
  }

  private buildUrl(
    path: string,
    query?: Record<string, string | number | boolean | undefined | null>,
  ): string {
    const normalizedPath = this.normalizePath(path);
    const url = new URL(`${this.baseUrl}${normalizedPath}`);

    if (query) {
      for (const [key, value] of Object.entries(query)) {
        if (value === undefined || value === null) {
          continue;
        }

        url.searchParams.set(key, String(value));
      }
    }

    return url.toString();
  }

  private normalizePath(path: string): string {
    if (!path.startsWith('/')) {
      return `/${path}`;
    }

    if (this.baseUrl.endsWith('/v1') && path.startsWith('/v1/')) {
      return path.slice(3);
    }

    return path;
  }

  public async parseResponseBody<T>(response: Response): Promise<T> {
    const contentType = response.headers.get('content-type') ?? '';

    if (contentType.includes('application/json')) {
      return response.json() as Promise<T>;
    }

    const text = await response.text();
    return text.length > 0 ? (text as unknown as T) : (null as unknown as T);
  }
}

function getPviumApiErrorMessage(params: {
  status: number;
  statusText: string;
  body: unknown;
}) {
  const message = getPviumApiErrorBodyMessage(params.body);
  return message
    ? `Pvium API request failed with ${params.status}: ${message}`
    : `Pvium API request failed with ${params.status} ${params.statusText}`;
}

function getPviumApiErrorBodyMessage(body: unknown) {
  if (!body || typeof body !== "object") return undefined;
  const record = body as Record<string, unknown>;
  const meta = record.meta;

  if (meta && typeof meta === "object") {
    const metaRecord = meta as Record<string, unknown>;
    if (typeof metaRecord.developerMessage === "string") {
      return metaRecord.developerMessage;
    }
    if (typeof metaRecord.message === "string") {
      return metaRecord.message;
    }
  }

  if (typeof record.developerMessage === "string") return record.developerMessage;
  if (typeof record.message === "string") return record.message;
  if (typeof record.error === "string") return record.error;
  return undefined;
}
