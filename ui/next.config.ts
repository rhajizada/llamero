import type { NextConfig } from "next";

const backendOrigin =
  process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ||
  process.env.LLAMERO_SERVER_EXTERNAL_URL ||
  "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${backendOrigin}/api/:path*`,
      },
      {
        source: "/auth/:path*",
        destination: `${backendOrigin}/auth/:path*`,
      },
      {
        source: "/healthz",
        destination: `${backendOrigin}/healthz`,
      },
    ];
  },
};

export default nextConfig;
