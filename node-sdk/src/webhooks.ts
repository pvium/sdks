import HmacSHA256 from "crypto-js/hmac-sha256";
import SHA256 from "crypto-js/sha256";
import Base64 from "crypto-js/enc-base64";
import Utf8 from "crypto-js/enc-utf8";

export interface PviumWebhookTokenPayload<TData = Record<string, unknown>> {
  event?: string;
  data?: TData;
  iat?: number;
  exp?: number;
  [key: string]: unknown;
}

export interface VerifyPviumWebhookTokenOptions {
  expectedEvent?: string;
  now?: Date | number;
  allowHashedSecretFallback?: boolean;
}

export function verifyPviumWebhookToken<
  TData = Record<string, unknown>,
>(
  token: string,
  secret: string,
  options: VerifyPviumWebhookTokenOptions = {},
): PviumWebhookTokenPayload<TData> {
  const parts = token.split(".");
  if (parts.length !== 3) {
    throw new Error("Invalid Pvium webhook token");
  }

  const [encodedHeader, encodedPayload, encodedSignature] = parts;
  const header = parseBase64UrlJson(encodedHeader);
  if (!header || header.alg !== "HS256") {
    throw new Error("Unsupported Pvium webhook token algorithm");
  }

  const signingInput = `${encodedHeader}.${encodedPayload}`;
  const secrets = [secret];
  if (options.allowHashedSecretFallback !== false) {
    const hashedSecret = SHA256(secret).toString();
    if (hashedSecret !== secret) {
      secrets.push(hashedSecret);
    }
  }

  const signatureValid = secrets.some((candidate) =>
    safeEqual(
      encodedSignature,
      hmacSha256Base64Url(signingInput, candidate),
    ),
  );

  if (!signatureValid) {
    throw new Error("Invalid Pvium webhook token signature");
  }

  const payload = parseBase64UrlJson(encodedPayload) as
    | PviumWebhookTokenPayload<TData>
    | null;
  if (!payload) {
    throw new Error("Invalid Pvium webhook token payload");
  }

  const nowSeconds =
    typeof options.now === "number"
      ? Math.floor(options.now / 1000)
      : Math.floor((options.now ?? new Date()).getTime() / 1000);
  if (typeof payload.exp === "number" && nowSeconds >= payload.exp) {
    throw new Error("Expired Pvium webhook token");
  }

  if (
    options.expectedEvent &&
    payload.event &&
    payload.event !== options.expectedEvent
  ) {
    throw new Error("Pvium webhook token event mismatch");
  }

  return payload;
}

export function resolvePviumWebhookPayload<TData = Record<string, unknown>>(
  body: {
    event?: string;
    type?: string;
    token?: string;
    data?: TData;
  },
  secret: string,
  options: VerifyPviumWebhookTokenOptions = {},
): { event?: string; data: TData; tokenPayload?: PviumWebhookTokenPayload<TData> } {
  if (!body.token) {
    return {
      event: body.event ?? body.type,
      data: body.data ?? ({} as TData),
    };
  }

  const tokenPayload = verifyPviumWebhookToken<TData>(body.token, secret, {
    ...options,
    expectedEvent: options.expectedEvent ?? body.event ?? body.type,
  });

  return {
    event: tokenPayload.event ?? body.event ?? body.type,
    data: tokenPayload.data ?? ({} as TData),
    tokenPayload,
  };
}

function hmacSha256Base64Url(value: string, secret: string): string {
  return HmacSHA256(value, secret)
    .toString(Base64)
    .replace(/=/g, "")
    .replace(/\+/g, "-")
    .replace(/\//g, "_");
}

function parseBase64UrlJson(value: string): Record<string, unknown> | null {
  try {
    const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(
      base64.length + ((4 - (base64.length % 4)) % 4),
      "=",
    );
    const json = Base64.parse(padded).toString(Utf8);
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
}

function safeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;

  let diff = 0;
  for (let index = 0; index < a.length; index += 1) {
    diff |= a.charCodeAt(index) ^ b.charCodeAt(index);
  }

  return diff === 0;
}
