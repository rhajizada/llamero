"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { createApiClient } from "@/lib/api-client";
import { TokenExpiryPicker } from "@/components/TokenExpiryPicker";
import { useAuth } from "@/components/AuthProvider";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error-message";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type {
  PersonalAccessToken,
  PersonalAccessTokenResponse,
} from "@/lib/api/data-contracts";

const MIN_TOKEN_TTL_SECONDS = 60;

interface TokenFormState {
  name: string;
  scopes: string[];
  expiresAt: Date | null;
}

interface ScopeRow {
  entity: string;
  action: string;
}

const defaultExpiryDate = () => {
  const date = new Date();
  date.setDate(date.getDate() + 30);
  date.setHours(23, 59, 59, 999);
  return date;
};

const utcDateTimeFormatter = new Intl.DateTimeFormat("en-US", {
  timeZone: "UTC",
  year: "numeric",
  month: "short",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
});

const formatDate = (value?: string) => {
  if (!value) return "—";
  return utcDateTimeFormatter.format(new Date(value));
};

export const PatPanel = () => {
  const { token, claims } = useAuth();
  const [tokens, setTokens] = useState<PersonalAccessToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [issuedToken, setIssuedToken] =
    useState<PersonalAccessTokenResponse | null>(null);
  const [copied, setCopied] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showSecretModal, setShowSecretModal] = useState(false);
  const [tokenToRevoke, setTokenToRevoke] = useState<string | null>(null);
  const [form, setForm] = useState<TokenFormState>({
    name: "",
    scopes: [],
    expiresAt: null,
  });

  useEffect(() => {
    const userScopes = claims?.scopes ?? [];
    setForm((prev) => {
      const nextScopes = prev.scopes.length ? prev.scopes : userScopes;
      const nextExpiry = prev.expiresAt ?? defaultExpiryDate();
      if (nextScopes === prev.scopes && nextExpiry === prev.expiresAt) {
        return prev;
      }
      return { ...prev, scopes: nextScopes, expiresAt: nextExpiry };
    });
  }, [claims?.scopes]);

  const availableScopes = useMemo(
    () => Array.from(new Set(claims?.scopes ?? [])).sort(),
    [claims?.scopes],
  );

  const refreshTokens = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const api = createApiClient(token);
      const response = await api.profileTokensList();
      setTokens(response.data ?? []);
    } catch (err) {
      console.error("load tokens", err);
      const message = getErrorMessage(err, "Unable to load personal access tokens");
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    refreshTokens();
  }, [refreshTokens]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!token) return;
    setSubmitting(true);
    setError(null);
    setIssuedToken(null);
    try {
      const api = createApiClient(token);
      const expiresInSeconds = form.expiresAt
        ? Math.max(
            Math.floor((form.expiresAt.getTime() - Date.now()) / 1000),
            MIN_TOKEN_TTL_SECONDS,
          )
        : undefined;
      const response = await api.profileTokensCreate({
        name: form.name || undefined,
        expires_in: expiresInSeconds,
        scopes: form.scopes.length ? form.scopes : undefined,
      });
      setIssuedToken(response.data ?? null);
      setCopied(false);
      setShowCreateModal(false);
      setShowSecretModal(true);
      setForm({
        name: "",
        scopes: form.scopes,
        expiresAt: defaultExpiryDate(),
      });
      await refreshTokens();
    } catch (err) {
      console.error("create token", err);
      const message = getErrorMessage(err, "Unable to issue token");
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  };

  const onDelete = async (tokenId?: string) => {
    if (!token || !tokenId) return;
    try {
      const api = createApiClient(token);
      await api.profileTokensDelete(tokenId);
      await refreshTokens();
    } catch (err) {
      console.error("delete token", err);
      const message = getErrorMessage(err, "Unable to revoke token");
      toast.error(message);
    }
  };

  const availableScopeRows = useMemo<ScopeRow[]>(() => {
    const rows = new Map<string, ScopeRow>();
    availableScopes.forEach((entry) => {
      if (!entry) return;
      const [entity, action] = entry.split(":");
      const normalizedEntity = entity || "general";
      const normalizedAction = action || "full";
      const id = `${normalizedEntity}:${normalizedAction}`;
      if (!rows.has(id)) {
        rows.set(id, {
          entity: normalizedEntity,
          action: normalizedAction,
        });
      }
    });
    return Array.from(rows.values()).sort((a, b) =>
      a.entity.localeCompare(b.entity),
    );
  }, [availableScopes]);

  const toggleScope = (scope: string) => {
    setForm((prev) => {
      const exists = prev.scopes.includes(scope);
      const scopes = exists
        ? prev.scopes.filter((s) => s !== scope)
        : [...prev.scopes, scope];
      return { ...prev, scopes };
    });
  };

  const availableSet = useMemo(
    () => new Set(availableScopes),
    [availableScopes],
  );

  const scopesByEntity = useMemo(
    () =>
      availableScopeRows.reduce((acc, row) => {
        const existing = acc.get(row.entity) ?? [];
        if (!existing.includes(row.action)) {
          existing.push(row.action);
        }
        acc.set(row.entity, existing.sort());
        return acc;
      }, new Map<string, string[]>()),
    [availableScopeRows],
  );

  return (
    <section className="rounded-3xl border border-border bg-card/60 p-6 shadow-sm">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <h2 className="text-xl font-semibold text-foreground">
          Personal access tokens
        </h2>
        <button
          type="button"
          onClick={() => setShowCreateModal(true)}
          className="rounded-full border border-border px-4 py-2 text-sm font-semibold text-foreground transition hover:bg-muted"
        >
          New token
        </button>
      </header>
      <div className="grid min-w-0 gap-6 lg:grid-cols-[3fr,2fr]">
        <div className="min-w-0 space-y-4">
          <div className="w-full max-w-full overflow-hidden rounded-2xl border border-border bg-background/60">
            <div className="w-full overflow-x-auto">
              <table className="min-w-full table-auto text-left text-sm">
                <thead>
                  <tr className="text-muted-foreground">
                    <th className="px-4 py-3 font-normal">Name</th>
                    <th className="px-4 py-3 font-normal">Scopes</th>
                    <th className="px-4 py-3 font-normal">Expires</th>
                    <th className="px-4 py-3 font-normal">Status</th>
                    <th className="px-4 py-3 font-normal text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {tokens.length === 0 ? (
                    <tr>
                      <td
                        className="px-4 py-6 text-center text-muted-foreground"
                        colSpan={5}
                      >
                        {loading ? "Loading tokens…" : "No tokens yet"}
                      </td>
                    </tr>
                  ) : (
                    tokens.map((item) => {
                      const isRevoked = Boolean(item.revoked);
                      const badgeClass = isRevoked
                        ? "bg-destructive/20 text-destructive"
                        : "bg-emerald-500/20 text-emerald-500";
                      return (
                        <tr key={item.id} className="border-t border-border/60">
                          <td className="px-4 py-3 font-medium">
                            {item.name || "—"}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground">
                            {(item.scopes || []).join(", ") || "default"}
                          </td>
                          <td className="px-4 py-3 text-xs">
                            {formatDate(item.expires_at)}
                          </td>
                          <td className="px-4 py-3">
                            <span
                              className={`rounded-full px-2 py-1 text-xs font-semibold ${badgeClass}`}
                            >
                              {isRevoked ? "revoked" : "active"}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-right">
                            {isRevoked ? null : (
                              <button
                                type="button"
                                className="text-xs tracking-wide text-destructive hover:underline"
                                onClick={() => setTokenToRevoke(item.id ?? null)}
                              >
                                Revoke
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>
          {availableScopeRows.length ? <></> : null}
          {error ? (
            <div className="rounded-xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
              {error}
            </div>
          ) : null}
        </div>
      </div>
      {showCreateModal ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 backdrop-blur-sm">
          <div className="w-full max-w-4xl rounded-3xl border border-border bg-card p-6 shadow-2xl">
            <header className="flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground">New token</p>
                <h3 className="text-lg font-semibold">
                  Define scope and expiry
                </h3>
              </div>
              <button
                type="button"
                onClick={() => setShowCreateModal(false)}
                className="rounded-full border border-border px-3 py-1 text-sm text-muted-foreground hover:bg-muted"
              >
                Close
              </button>
            </header>
            <form className="mt-4 space-y-4" onSubmit={onSubmit}>
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="text-xs text-muted-foreground">
                    Token name
                  </label>
                  <input
                    type="text"
                    value={form.name}
                    onChange={(e) =>
                      setForm((prev) => ({ ...prev, name: e.target.value }))
                    }
                    className="mt-2 w-full rounded-xl border border-border bg-card/60 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">
                    Expires on
                  </label>
                  <div className="mt-2">
                    <TokenExpiryPicker
                      value={form.expiresAt}
                      onChange={(date) =>
                        setForm((prev) => ({ ...prev, expiresAt: date }))
                      }
                    />
                  </div>
                </div>
              </div>
              {availableScopeRows.length ? (
                <div className="rounded-2xl border border-border bg-background/60 p-4">
                  <h4 className="text-sm font-semibold text-foreground">
                    Scopes
                  </h4>
                  <p className="text-xs text-muted-foreground">
                    Toggle the permissions this token should include.
                  </p>
                  <div className="mt-3 space-y-3">
                    {Array.from(scopesByEntity.entries()).map(([entity, actions]) => (
                      <div
                        key={entity}
                        className="rounded-xl border border-border bg-card/40 p-3"
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-semibold text-foreground">
                            {entity}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            {actions.length} action{actions.length === 1 ? "" : "s"}
                          </span>
                        </div>
                        <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3">
                          {actions.map((action) => {
                            const scopeValue = `${entity}:${action}`;
                            const available = availableSet.has(scopeValue);
                            const checked = form.scopes.includes(scopeValue);
                            return (
                              <label
                                key={scopeValue}
                                className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm ${
                                  checked ? "border-foreground bg-foreground/10" : "border-border"
                                }`}
                              >
                                <input
                                  type="checkbox"
                                  disabled={!available}
                                  checked={checked && available}
                                  onChange={() => toggleScope(scopeValue)}
                                  className="h-4 w-4 rounded border border-border"
                                />
                                <span className="text-foreground">{action}</span>
                              </label>
                            );
                          })}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  You do not have any scopes assigned to your account.
                </p>
              )}
              <div className="flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="rounded-full border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="rounded-full bg-foreground px-4 py-2 text-sm font-semibold text-background transition hover:opacity-90 disabled:opacity-60"
                >
                  {submitting ? "Issuing…" : "Create token"}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
      {showSecretModal && issuedToken?.token ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 backdrop-blur-sm">
          <div className="w-full max-w-xl rounded-3xl border border-border bg-card p-6 shadow-2xl">
            <header className="mb-3">
              <h3 className="text-lg font-semibold text-foreground">
                Token issued
              </h3>
              <p className="text-sm text-muted-foreground">
                Copy this secret now — you will not be able to view it again.
              </p>
            </header>
            <div className="rounded-xl border border-border bg-background/80 p-3 text-xs">
              <code className="block break-all whitespace-pre-wrap">
                {issuedToken.token}
              </code>
            </div>
            <div className="mt-4 flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={() => {
                  if (issuedToken?.token) {
                    navigator.clipboard?.writeText(issuedToken.token).then(
                      () => {
                        setCopied(true);
                        setTimeout(() => setCopied(false), 2000);
                      },
                      () => setCopied(false),
                    );
                  }
                }}
                className="rounded-full border border-border px-4 py-2 text-sm font-semibold text-foreground hover:bg-muted"
              >
                {copied ? "Copied" : "Copy token"}
              </button>
              <button
                type="button"
                onClick={() => setShowSecretModal(false)}
                className="rounded-full bg-foreground px-4 py-2 text-sm font-semibold text-background hover:opacity-90"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      ) : null}
      <Dialog open={Boolean(tokenToRevoke)} onOpenChange={(open) => !open && setTokenToRevoke(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke token</DialogTitle>
            <DialogDescription>
              Revoked tokens cannot be used again. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4">
            <button
              type="button"
              className="rounded-full border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted"
              onClick={() => setTokenToRevoke(null)}
            >
              Cancel
            </button>
            <button
              type="button"
              className="rounded-full bg-destructive px-4 py-2 text-sm font-semibold text-destructive-foreground hover:opacity-90"
              onClick={() => {
                if (tokenToRevoke) {
                  onDelete(tokenToRevoke);
                  setTokenToRevoke(null);
                }
              }}
            >
              Revoke token
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
};
