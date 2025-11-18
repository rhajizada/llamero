import type { Metadata } from "next";
import { AppShell } from "@/components/AppShell";
import "./globals.css";

export const metadata: Metadata = {
  title: "Llamero Control Plane",
  description: "Manage personal access tokens and Ollama backends from one place.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="bg-background font-sans antialiased">
        <AppShell>{children}</AppShell>
      </body>
    </html>
  );
}
