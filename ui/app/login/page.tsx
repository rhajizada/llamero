"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";
import { LoginCard } from "@/components/LoginCard";

export default function LoginPage() {
  const router = useRouter();
  const { isAuthenticated, setSession } = useAuth();

  useEffect(() => {
    if (typeof window === "undefined") return;
    const hash = window.location.hash.startsWith("#")
      ? window.location.hash.slice(1)
      : window.location.hash;
    if (!hash) {
      if (isAuthenticated) {
        router.replace("/");
      }
      return;
    }

    const params = new URLSearchParams(hash);
    const token = params.get("token");
    const expires = params.get("expires_in");

    if (token) {
      const expiresIn = expires ? Number(expires) : undefined;
      setSession(token, expiresIn);
      window.location.hash = "";
      router.replace("/");
    }
  }, [isAuthenticated, router, setSession]);

  return (
    <section className="flex min-h-[70vh] items-center justify-center">
      <LoginCard
        title="Authenticate"
        subtitle="You will be redirected to the control plane after signing in."
      />
    </section>
  );
}
