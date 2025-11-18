"use client";

import Image from "next/image";
import Link from "next/link";
import { useAuth } from "@/components/AuthProvider";
import { ThemeToggle } from "@/components/ThemeToggle";

export const Navbar = () => {
  const { isAuthenticated, profile, claims, login, logout } = useAuth();
  const isAdmin = claims?.role === "admin";

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/80 backdrop-blur">
      <div className="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-4 lg:px-6">
        <Link href="/" className="flex items-center gap-4">
          <Image
            src="/assets/brand/logo.png"
            alt="Llamero"
            width={48}
            height={48}
            priority
            className="rounded-2xl saturate-0 brightness-200"
          />
          <div className="flex flex-col leading-tight">
            <span className="text-lg font-semibold tracking-tight">
              Llamero
            </span>
          </div>
        </Link>
        <div className="flex items-center gap-3 text-sm">
          {isAuthenticated ? (
            <>
              {isAdmin ? (
                <Link
                  href="/admin"
                  className="rounded-full border border-border px-4 py-2 text-sm font-medium transition hover:bg-muted"
                >
                  Admin
                </Link>
              ) : null}
            </>
          ) : null}
          <ThemeToggle />
          {isAuthenticated ? (
            <div className="flex items-center gap-3">
              <span className="hidden text-muted-foreground sm:inline-flex">
                {profile?.email || "Signed in"}
              </span>
              <button
                type="button"
                onClick={logout}
                className="rounded-full border border-border px-4 py-2 text-sm font-medium transition hover:bg-muted"
              >
                Sign out
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={login}
              className="rounded-full border border-border px-4 py-2 text-sm font-medium transition hover:bg-muted"
            >
              Sign in
            </button>
          )}
        </div>
      </div>
    </header>
  );
};
