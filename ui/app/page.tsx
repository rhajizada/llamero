"use client";

import { useAuth } from "@/components/AuthProvider";
import { LoginCard } from "@/components/LoginCard";
import { PatPanel } from "@/components/PatPanel";
import { ModelsPanel } from "@/components/ModelsPanel";

export default function HomePage() {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return (
      <section className="flex min-h-[70vh] items-center justify-center">
        <LoginCard />
      </section>
    );
  }

  return (
    <div className="space-y-8">
      <PatPanel />
      <ModelsPanel />
    </div>
  );
}
