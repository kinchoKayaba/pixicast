import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

// ★ここがポイント: src/lib から見て src/gen は一つ上(..)の gen です
import { TimelineService } from "../gen/proto/pixicast/v1/timeline_connect";

const transport = createConnectTransport({
  baseUrl: "", // プロキシ経由なので空文字でOK
});

export const client = createClient(TimelineService, transport);
