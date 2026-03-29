import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./i18n/request.ts");

const nextConfig: NextConfig = {
  // Remove the X-Powered-By header to reduce response size and hide framework info
  poweredByHeader: false,

  // Enable React strict mode for catching potential issues
  reactStrictMode: true,

  // Optimize images: allow remote patterns for user-uploaded content (MinIO / R2)
  images: {
    formats: ["image/avif", "image/webp"],
    remotePatterns: [
      {
        protocol: "http",
        hostname: "localhost",
        port: "9000",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "192.168.1.156",
        port: "9000",
        pathname: "/**",
      },
      {
        protocol: "https",
        hostname: "**.r2.cloudflarestorage.com",
        pathname: "/**",
      },
      {
        protocol: "https",
        hostname: "**.r2.dev",
        pathname: "/**",
      },
    ],
  },

  // Enable gzip compression (useful for self-hosting)
  compress: true,

  // Proxy API calls through Next.js in production so cookies stay same-origin.
  // Without this, session_id cookie set by Railway won't be sent to Vercel.
  // Uses API_BACKEND_URL (server-only) for the rewrite destination.
  // In development, NEXT_PUBLIC_API_URL is set so the client calls directly — no proxy needed.
  async rewrites() {
    if (process.env.NEXT_PUBLIC_API_URL) return [];
    const backendUrl = process.env.API_BACKEND_URL;
    if (!backendUrl) return [];
    return [
      {
        source: "/api/:path*",
        destination: `${backendUrl}/api/:path*`,
      },
    ];
  },

  // Experimental performance optimizations
  experimental: {
    // Optimize package imports to reduce bundle size
    optimizePackageImports: ["lucide-react", "clsx", "@tanstack/react-query"],
  },
};

export default withNextIntl(nextConfig);
