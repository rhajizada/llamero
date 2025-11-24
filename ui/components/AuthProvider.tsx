"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { User } from "@/lib/api/data-contracts";
import { createApiClient } from "@/lib/api-client";
import { clearStoredAuth, loadStoredAuth, persistAuth } from "@/lib/auth-storage";
import { decodeJwt, type JwtClaims } from "@/lib/jwt";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error-message";

interface AuthContextValue {
  token: string | null;
  isAuthenticated: boolean;
  expiresAt: number | null;
  profile: User | null;
  claims: JwtClaims | null;
  loading: boolean;
  error: string | null;
  login: () => void;
  logout: () => void;
  setSession: (token: string, expiresInSeconds?: number | null) => void;
  refreshProfile: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const [token, setToken] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<number | null>(null);
  const [profile, setProfile] = useState<User | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const stored = loadStoredAuth();
    if (stored.token) {
      if (stored.expiresAt && stored.expiresAt < Date.now()) {
        clearStoredAuth();
        return;
      }
      setToken(stored.token);
      setExpiresAt(stored.expiresAt ?? null);
    }
  }, []);

  const logout = useCallback(() => {
    clearStoredAuth();
    setToken(null);
    setExpiresAt(null);
    setProfile(null);
  }, []);

  const setSession = useCallback((newToken: string, expiresInSeconds?: number | null) => {
    const derivedExpiry = expiresInSeconds
      ? Date.now() + expiresInSeconds * 1000
      : null;
    setToken(newToken);
    setExpiresAt(derivedExpiry);
    persistAuth(newToken, derivedExpiry);
  }, []);

  const login = useCallback(() => {
    if (typeof window === "undefined") return;
    window.location.href = "/auth/login";
  }, []);

  const refreshProfile = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const api = createApiClient(token);
      const response = await api.profileList();
      setProfile(response.data ?? null);
    } catch (err) {
      console.error("load profile", err);
      const error = err as { status?: number };
      if (error?.status === 401) {
        logout();
        setError("Session expired. Please sign in again.");
      } else {
        const message = getErrorMessage(err, "Unable to load profile");
        setError(message);
        toast.error(message);
      }
    } finally {
      setLoading(false);
    }
  }, [logout, token]);

  useEffect(() => {
    if (!token) {
      setProfile(null);
      return;
    }
    refreshProfile();
  }, [token, refreshProfile]);

  const claims = useMemo(() => (token ? decodeJwt(token) : null), [token]);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      isAuthenticated: Boolean(token),
      expiresAt,
      profile,
      claims,
      loading,
      error,
      login,
      logout,
      setSession,
      refreshProfile,
    }),
    [claims, error, expiresAt, loading, login, logout, profile, setSession, token, refreshProfile],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
};
