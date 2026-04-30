import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
import path from "path";

const withNextIntl = createNextIntlPlugin("./i18n/request.ts");

const nextConfig: NextConfig = {
  // Scope Turbopack to web/ only — prevents watching backend/, admin/, mobile/
  turbopack: {
    root: path.resolve(__dirname, ".."),
  },
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

  // Security headers — applied to every Next.js response. The backend
  // serves the same headers via middleware.SecurityHeaders for API
  // responses; this block covers the static + SSR pages that Next
  // serves directly (Vercel/self-hosted) and never hit the Go backend.
  //
  // CSP allowlist: Stripe (Embedded Components + checkout), R2/MinIO
  // (uploaded media), and LiveKit (WebRTC signalling). Tightened
  // beyond the backend default so the browser side gets the same
  // protection as the API side.
  async headers() {
    const csp = [
      "default-src 'self'",
      // Next 16 + React Server Components rely on inline runtime
      // bootstraps; 'unsafe-inline' is required until we ship
      // hash-based or nonce-based CSP (tracked as follow-up).
      "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://js.stripe.com https://*.stripe.com",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: blob: https://*.r2.cloudflarestorage.com https://*.r2.dev http://localhost:9000 http://192.168.1.156:9000 https://*.stripe.com",
      "media-src 'self' blob: https://*.r2.cloudflarestorage.com https://*.r2.dev http://localhost:9000 http://192.168.1.156:9000",
      "font-src 'self' data:",
      "connect-src 'self' https://*.stripe.com https://api.stripe.com https://*.r2.cloudflarestorage.com https://*.r2.dev http://localhost:8083 https://*.livekit.cloud wss://*.livekit.cloud ws://localhost:8083 wss://localhost:8083",
      "frame-src https://js.stripe.com https://hooks.stripe.com https://*.stripe.com",
      "frame-ancestors 'none'",
      "object-src 'none'",
      "base-uri 'self'",
      "form-action 'self'",
    ].join("; ");

    return [
      {
        source: "/:path*",
        headers: [
          { key: "Content-Security-Policy", value: csp },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-XSS-Protection", value: "0" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
          // HSTS in production only — match backend behaviour. Vercel
          // serves only over HTTPS so it's safe to keep on at all
          // times when NODE_ENV=production.
          ...(process.env.NODE_ENV === "production"
            ? [{ key: "Strict-Transport-Security", value: "max-age=31536000; includeSubDomains" }]
            : []),
        ],
      },
    ];
  },

  // Proxy API calls through Next.js in production so cookies stay same-origin.
  // Without this, session_id cookie set by Railway won't be sent to Vercel.
  // Uses API_BACKEND_URL (server-only) for the rewrite destination.
  // In development, NEXT_PUBLIC_API_URL is set so the client calls directly — no proxy needed.
  async rewrites() {
    const backendUrl =
      process.env.API_BACKEND_URL ?? process.env.NEXT_PUBLIC_API_URL;
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
