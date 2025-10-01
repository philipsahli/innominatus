import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  // Use 'standalone' for Docker, 'export' for static hosting
  output: process.env.DOCKER_BUILD === 'true' ? 'standalone' : 'export',
  turbopack: {
    root: path.resolve(__dirname)
  },
  trailingSlash: true,
  images: {
    unoptimized: true
  },
  skipTrailingSlashRedirect: true
};

export default nextConfig;
