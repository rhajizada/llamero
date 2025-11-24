"use client";

import { usePathname } from "next/navigation";
import { AuthProvider } from "./AuthProvider";
import { Navbar } from "./Navbar";
import { ThemeProvider } from "./ThemeProvider";
import { Toaster } from "@/components/ui/sonner";

export const AppShell = ({ children }: { children: React.ReactNode }) => {
  const pathname = usePathname();
  const hideNav = pathname?.startsWith("/login");

  return (
    <ThemeProvider>
      <AuthProvider>
        <div className="min-h-screen bg-background text-foreground">
          {hideNav ? null : <Navbar />}
          <main className="mx-auto w-full max-w-6xl px-4 pb-12 pt-8 lg:px-6">
            {children}
          </main>
          <Toaster position="bottom-right" closeButton richColors />
        </div>
      </AuthProvider>
    </ThemeProvider>
  );
};
