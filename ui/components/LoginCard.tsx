"use client";

import { useAuth } from "@/components/AuthProvider";

interface LoginCardProps {
  title?: string;
  subtitle?: string;
}

export const LoginCard = ({
  title = "Sign in to Llamero",
  subtitle = "Authenticate with your corporate identity provider to manage access tokens and Ollama backends.",
}: LoginCardProps) => {
  const { login, error } = useAuth();

  return (
    <div className="w-full max-w-md rounded-2xl border border-border bg-card/80 p-8 shadow-lg shadow-black/5">
      <div className="mb-6 space-y-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="text-sm text-muted-foreground">{subtitle}</p>
      </div>
      {error ? (
        <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-2 text-sm text-destructive">
          {error}
        </div>
      ) : null}
      <button
        type="button"
        onClick={login}
        className="w-full rounded-xl bg-foreground px-4 py-3 text-sm font-semibold tracking-wide text-background transition hover:opacity-90"
      >
        Continue with OAuth
      </button>
    </div>
  );
};
