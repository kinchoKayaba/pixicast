"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import AddChannelModal from "@/components/AddChannelModal";

interface Channel {
  user_id: number;
  platform: string;
  source_id: string;
  channel_id: string;
  handle: string;
  display_name: string;
  thumbnail_url: string;
  enabled: boolean;
}

export default function ChannelsPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const router = useRouter();
  const { user, getIdToken } = useAuth();

  useEffect(() => {
    fetchChannels();
  }, [user]);

  const fetchChannels = async () => {
    setLoading(true);
    try {
      if (!user) {
        console.log("⚠️ ChannelsPage: User not authenticated");
        setChannels([]);
        setLoading(false);
        return;
      }

      const idToken = await getIdToken();
      if (!idToken) {
        console.error("❌ ChannelsPage: Failed to get ID token");
        setChannels([]);
        setLoading(false);
        return;
      }

      const response = await fetch("http://localhost:8080/v1/subscriptions", {
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
      });

      if (!response.ok) {
        console.error("❌ ChannelsPage: API error:", response.status);
        setChannels([]);
        setLoading(false);
        return;
      }

      const data = await response.json();
      console.log("✅ ChannelsPage: Channels loaded:", data.subscriptions?.length || 0);
      setChannels(data.subscriptions || []);
    } catch (error) {
      console.error("❌ ChannelsPage: チャンネル取得エラー:", error);
      setChannels([]);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (channelId: string) => {
    if (!confirm("このチャンネルの登録を解除しますか？")) return;

    console.log("Deleting channel:", channelId);

    try {
      const idToken = await getIdToken();
      if (!idToken) {
        console.error("❌ ChannelsPage: Failed to get ID token for delete");
        alert("認証エラーが発生しました");
        return;
      }

      const response = await fetch(
        `http://localhost:8080/v1/subscriptions/${channelId}`,
        {
          method: "DELETE",
          headers: {
            Authorization: `Bearer ${idToken}`,
          },
        }
      );

      console.log("Delete response status:", response.status);

      if (!response.ok) {
        const errorText = await response.text();
        console.error("Delete failed:", errorText);
        throw new Error(
          `Failed to delete channel: ${response.status} ${errorText}`
        );
      }

      // リストを更新（UIの即座の反映用）
      setChannels(channels.filter((ch) => ch.channel_id !== channelId));

      // ホームにリダイレクト（サイドバーが自動的に更新される）
      router.push("/");
      router.refresh(); // Next.jsのキャッシュをリフレッシュ
    } catch (error) {
      console.error("チャンネル削除エラー:", error);
      alert(`削除に失敗しました: ${error}`);
    }
  };

  const handleChannelAdded = () => {
    // チャンネルリストを再取得
    fetchChannels();
  };

  return (
    <div className="min-h-screen bg-pink-50 p-8 pt-20">
      <div className="max-w-4xl">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold text-gray-800">チャンネル管理</h1>
          <button
            onClick={() => setIsModalOpen(true)}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors shadow-md"
            type="button"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            <span className="font-medium">チャンネルを追加</span>
          </button>
        </div>

        {loading ? (
          <p className="text-gray-500">読み込み中...</p>
        ) : channels.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-8 text-center">
            <p className="text-gray-500 mb-4">
              登録されているチャンネルがありません
            </p>
            <p className="text-sm text-gray-400">
              上の「+ チャンネルを追加」ボタンから登録できます
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {channels.map((channel) => (
              <div
                key={channel.channel_id}
                className="bg-white rounded-lg shadow p-6 flex items-center justify-between hover:shadow-md transition-shadow"
              >
                <div className="flex items-center gap-4">
                  <img
                    src={channel.thumbnail_url}
                    alt={channel.display_name}
                    className="w-16 h-16 rounded-full"
                  />
                  <div>
                    <h2 className="text-lg font-semibold text-gray-800">
                      {channel.display_name}
                    </h2>
                    <p className="text-sm text-gray-500">
                      {channel.platform === "youtube"
                        ? "YouTube"
                        : channel.platform}
                      {channel.handle && ` • @${channel.handle}`}
                    </p>
                  </div>
                </div>

                <button
                  onClick={() => handleDelete(channel.channel_id)}
                  className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors"
                >
                  削除
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* チャンネル追加モーダル */}
      <AddChannelModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSuccess={handleChannelAdded}
      />
    </div>
  );
}
