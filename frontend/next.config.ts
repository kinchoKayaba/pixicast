import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // ... (webpackの設定はそのまま) ...
  webpack: (config, { isServer }) => {
    if (!isServer) {
      config.resolve.fallback = {
        ...(config.resolve.fallback || {}),
        fs: false,
      };
    }
    return config;
  },

  async rewrites() {
    // 環境変数があればそれを使い、なければlocalhostを使う
    const backendUrl = process.env.BACKEND_URL || "http://localhost:8080";

    return [
      {
        source: "/pixicast.v1.TimelineService/:path*",
        // 変数を使って動的にする
        destination: `${backendUrl}/pixicast.v1.TimelineService/:path*`,
      },
    ];
  },
};

export default nextConfig;
