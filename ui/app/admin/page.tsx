"use client";

import { useAuth } from "@/components/AuthProvider";
import { BackendsConsole } from "@/components/BackendsConsole";
import { LoginCard } from "@/components/LoginCard";

export default function AdminPage() {
  const { isAuthenticated, claims } = useAuth();
  const isAdmin = claims?.role === "admin";

  if (!isAuthenticated) {
    return (
      <section className="flex min-h-[70vh] items-center justify-center">
        <LoginCard
          title="Restricted area"
          subtitle="Sign in with an administrator account to continue."
        />
      </section>
    );
  }

  if (!isAdmin) {
    return (
      <section className="rounded-3xl border border-border bg-card/70 p-6 text-center text-sm text-muted-foreground">
        <p>You need the admin role to access backend operations.</p>
      </section>
    );
  }

  return (
    <div className="space-y-8">
      <BackendsConsole />
    </div>
  );
}
