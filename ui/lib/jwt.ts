export interface JwtClaims {
  aud?: string | string[];
  email?: string;
  exp?: number;
  iat?: number;
  iss?: string;
  jti?: string;
  role?: string;
  scopes?: string[];
  sub?: string;
  [key: string]: unknown;
}

const decodeBase64 = (segment: string) => {
  try {
    return atob(segment.replace(/-/g, "+").replace(/_/g, "/"));
  } catch (err) {
    console.warn("failed to decode jwt", err);
    return null;
  }
};

export const decodeJwt = (token: string): JwtClaims | null => {
  if (!token) return null;
  const parts = token.split(".");
  if (parts.length < 2) return null;
  const payload = decodeBase64(parts[1]);
  if (!payload) return null;
  try {
    return JSON.parse(payload);
  } catch (err) {
    console.warn("invalid jwt payload", err);
    return null;
  }
};
