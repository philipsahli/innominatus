import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  output: 'export',
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
