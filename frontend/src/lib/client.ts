import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { auth } from "./firebase";
import { API_BASE_URL } from "./config";

// ★ここがポイント: src/lib から見て src/gen は一つ上(..)の gen です
import { TimelineService } from "../gen/proto/pixicast/v1/timeline_connect";

const transport = createConnectTransport({
  baseUrl: API_BASE_URL,
  interceptors: [
    (next) => async (req) => {
      // Firebase IDトークンを取得してヘッダーに追加
      const user = auth.currentUser;
      if (user) {
        try {
          const token = await user.getIdToken();
          req.header.set("Authorization", `Bearer ${token}`);
          console.log("🎫 gRPC request with auth token");
        } catch (error) {
          console.error("❌ Failed to get ID token for gRPC:", error);
        }
      } else {
        console.warn("⚠️ gRPC request without auth (user not logged in)");
      }
      return await next(req);
    },
  ],
});

export const client = createClient(TimelineService, transport);
