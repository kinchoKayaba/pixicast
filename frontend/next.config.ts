import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  webpack: (config, { isServer }) => {
    if (!isServer) {
      // ブラウザ側バンドルで Node.js の fs モジュールを解決しようとした場合は
      // 何もしないダミーとして扱う
      config.resolve = config.resolve || {};
      config.resolve.fallback = {
        ...(config.resolve.fallback || {}),
        fs: false,
      };
    }
    return config;
  },
  async rewrites() {
    return [
      {
        // ブラウザからのリクエストパス
        source: "/pixicast.v1.TimelineService/:path*",
        // Goサーバーへの転送先
        destination: "http://localhost:8080/pixicast.v1.TimelineService/:path*",
      },
    ];
  },
};

export default nextConfig;
