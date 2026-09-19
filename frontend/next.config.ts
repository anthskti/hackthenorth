import type { NextConfig } from "next";

/** Go backend (see backend/main.go). Override for remote Pi / docker. */
const backendOrigin = process.env.BACKEND_URL ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  reactCompiler: true,
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${backendOrigin}/api/:path*`,
      },
      {
        source: "/video/:path*",
        destination: `${backendOrigin}/video/:path*`,
      },
      {
        source: "/ws/:path*",
        destination: `${backendOrigin}/ws/:path*`,
      },
      {
        source: "/ping",
        destination: `${backendOrigin}/ping`,
      },
    ];
  },
};

export default nextConfig;
