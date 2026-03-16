import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Remove the X-Powered-By header to reduce response size and hide framework info
  poweredByHeader: false,

  // Enable React strict mode for catching potential issues
  reactStrictMode: true,

  // Optimize images: allow remote patterns for user-uploaded content
  images: {
    formats: ["image/avif", "image/webp"],
  },

  // Enable gzip compression (useful for self-hosting)
  compress: true,

  // Experimental performance optimizations
  experimental: {
    // Optimize package imports to reduce bundle size
    optimizePackageImports: ["lucide-react", "clsx", "@tanstack/react-query"],
  },
};

export default nextConfig;
