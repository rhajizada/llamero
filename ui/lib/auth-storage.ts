const TOKEN_KEY = "llamero.token";
const EXP_KEY = "llamero.token_exp";

export type StoredAuth = {
  token: string | null;
  expiresAt: number | null;
};

const readNumber = (value: string | null): number | null => {
  if (!value) return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

export const loadStoredAuth = (): StoredAuth => {
  if (typeof window === "undefined") {
    return { token: null, expiresAt: null };
  }

  return {
    token: window.localStorage.getItem(TOKEN_KEY),
    expiresAt: readNumber(window.localStorage.getItem(EXP_KEY)),
  };
};

export const persistAuth = (token: string, expiresAt?: number | null) => {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
  if (expiresAt && Number.isFinite(expiresAt)) {
    window.localStorage.setItem(EXP_KEY, String(expiresAt));
  } else {
    window.localStorage.removeItem(EXP_KEY);
  }
};

export const clearStoredAuth = () => {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(EXP_KEY);
};
