"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function LandingPage() {
  const { user, isAnonymous, signInWithGoogle, signInAnonymously } = useAuth();
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(false);

  // ログイン済みユーザーは自動的にタイムラインへリダイレクト
  useEffect(() => {
    if (user && !isAnonymous) {
      router.push("/timeline");
    }
  }, [user, isAnonymous, router]);

  const handleGoogleLogin = async () => {
    try {
      setIsLoading(true);
      await signInWithGoogle();
      router.push("/timeline");
    } catch (error) {
      console.error("Googleログインエラー:", error);
      alert("ログインに失敗しました");
    } finally {
      setIsLoading(false);
    }
  };

  const handleAnonymousStart = async () => {
    try {
      setIsLoading(true);
      await signInAnonymously();
      router.push("/timeline");
    } catch (error) {
      console.error("匿名ログインエラー:", error);
      alert("開始に失敗しました");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-pink-50 to-white">
      {/* ヒーローセクション */}
      <section className="container mx-auto px-6 py-20 text-center">
        <div className="max-w-4xl mx-auto">
          {/* ロゴ & タイトル */}
          <div className="mb-8">
            <h1 className="text-6xl font-bold text-gray-900 mb-4">
              🎬 Pixicast
            </h1>
            <p className="text-2xl font-semibold text-pink-600 mb-6">
              自分専用のコンテンツ編成表
            </p>
          </div>

          {/* 説明文 */}
          <p className="text-xl text-gray-600 mb-8 leading-relaxed">
            YouTube、Twitch、Podcastなどの配信スケジュールを<br />
            ひとつのタイムラインで管理・可視化
          </p>

          {/* CTAボタン */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center items-center mb-12">
            <button
              onClick={handleGoogleLogin}
              disabled={isLoading}
              className="flex items-center gap-3 bg-white border-2 border-gray-300 px-8 py-4 rounded-xl hover:bg-gray-50 transition-all shadow-lg hover:shadow-xl disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <svg className="w-6 h-6" viewBox="0 0 24 24">
                <path
                  fill="#4285F4"
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                />
                <path
                  fill="#34A853"
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                />
                <path
                  fill="#FBBC05"
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                />
                <path
                  fill="#EA4335"
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                />
              </svg>
              <span className="text-lg font-semibold text-gray-700">
                Googleでログイン
              </span>
            </button>

            <button
              onClick={handleAnonymousStart}
              disabled={isLoading}
              className="px-8 py-4 bg-pink-500 text-white rounded-xl hover:bg-pink-600 transition-all shadow-lg hover:shadow-xl text-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
            >
              とりあえず始める
            </button>
          </div>

          {/* 補足説明 */}
          <p className="text-sm text-gray-500">
            ログインなしで5チャンネルまで無料で試せます
          </p>
        </div>
      </section>

      {/* 機能紹介セクション */}
      <section className="container mx-auto px-6 py-20 bg-white">
        <h2 className="text-4xl font-bold text-center text-gray-900 mb-16">
          主な機能
        </h2>

        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8 max-w-6xl mx-auto">
          {/* マルチプラットフォーム */}
          <div className="text-center p-6 rounded-xl bg-pink-50 hover:bg-pink-100 transition-colors">
            <div className="text-5xl mb-4">📺</div>
            <h3 className="text-xl font-bold text-gray-900 mb-3">
              マルチプラットフォーム
            </h3>
            <p className="text-gray-600">
              YouTube、Twitch、Podcastなど複数のプラットフォームに対応
            </p>
          </div>

          {/* パーソナライズド */}
          <div className="text-center p-6 rounded-xl bg-pink-50 hover:bg-pink-100 transition-colors">
            <div className="text-5xl mb-4">✨</div>
            <h3 className="text-xl font-bold text-gray-900 mb-3">
              自分専用タイムライン
            </h3>
            <p className="text-gray-600">
              登録したチャンネルだけを表示。あなた専用の編成表を作成
            </p>
          </div>

          {/* リアルタイム */}
          <div className="text-center p-6 rounded-xl bg-pink-50 hover:bg-pink-100 transition-colors">
            <div className="text-5xl mb-4">⚡</div>
            <h3 className="text-xl font-bold text-gray-900 mb-3">
              リアルタイム更新
            </h3>
            <p className="text-gray-600">
              ライブ配信の開始/終了を自動検知して表示
            </p>
          </div>

          {/* 段階的認証 */}
          <div className="text-center p-6 rounded-xl bg-pink-50 hover:bg-pink-100 transition-colors">
            <div className="text-5xl mb-4">🚀</div>
            <h3 className="text-xl font-bold text-gray-900 mb-3">
              段階的に使える
            </h3>
            <p className="text-gray-600">
              未ログインで試用→ログインで無制限→Proで広告なし
            </p>
          </div>
        </div>
      </section>

      {/* プラン比較表 */}
      <section className="container mx-auto px-6 py-20">
        <h2 className="text-4xl font-bold text-center text-gray-900 mb-16">
          料金プラン
        </h2>

        <div className="grid md:grid-cols-3 gap-8 max-w-5xl mx-auto">
          {/* 匿名プラン */}
          <div className="border-2 border-gray-200 rounded-2xl p-8 bg-white hover:shadow-xl transition-shadow">
            <div className="text-center mb-6">
              <h3 className="text-2xl font-bold text-gray-900 mb-2">
                匿名プラン
              </h3>
              <div className="text-4xl font-bold text-gray-900 mb-2">
                ¥0
              </div>
              <p className="text-gray-500">とりあえず試したい方向け</p>
            </div>

            <ul className="space-y-4 mb-8">
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">最大5チャンネル登録</span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">タイムライン表示</span>
              </li>
              <li className="flex items-start">
                <span className="text-red-500 mr-2">✗</span>
                <span className="text-gray-400">お気に入り機能</span>
              </li>
              <li className="flex items-start">
                <span className="text-red-500 mr-2">✗</span>
                <span className="text-gray-400">デバイス間同期</span>
              </li>
              <li className="flex items-start">
                <span className="text-yellow-500 mr-2">⚠</span>
                <span className="text-gray-500 text-sm">
                  30日間未アクセスでデータ削除
                </span>
              </li>
            </ul>

            <button
              onClick={handleAnonymousStart}
              disabled={isLoading}
              className="w-full py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors font-semibold disabled:opacity-50"
            >
              今すぐ始める
            </button>
          </div>

          {/* ベーシックプラン */}
          <div className="border-4 border-pink-500 rounded-2xl p-8 bg-white hover:shadow-2xl transition-shadow relative">
            <div className="absolute -top-4 left-1/2 -translate-x-1/2 bg-pink-500 text-white px-4 py-1 rounded-full text-sm font-bold">
              おすすめ
            </div>

            <div className="text-center mb-6">
              <h3 className="text-2xl font-bold text-gray-900 mb-2">
                ベーシックプラン
              </h3>
              <div className="text-4xl font-bold text-gray-900 mb-2">
                ¥0
              </div>
              <p className="text-gray-500">本格的に使いたい方向け</p>
            </div>

            <ul className="space-y-4 mb-8">
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700 font-semibold">
                  無制限チャンネル登録
                </span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">お気に入り機能</span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">デバイス間同期</span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">データ永久保存</span>
              </li>
              <li className="flex items-start">
                <span className="text-yellow-500 mr-2">⚠</span>
                <span className="text-gray-500 text-sm">広告表示あり</span>
              </li>
            </ul>

            <button
              onClick={handleGoogleLogin}
              disabled={isLoading}
              className="w-full py-3 bg-pink-500 text-white rounded-lg hover:bg-pink-600 transition-colors font-semibold disabled:opacity-50"
            >
              Googleでログイン
            </button>
          </div>

          {/* プロプラン */}
          <div className="border-2 border-purple-200 rounded-2xl p-8 bg-gradient-to-br from-purple-50 to-white hover:shadow-xl transition-shadow">
            <div className="text-center mb-6">
              <h3 className="text-2xl font-bold text-gray-900 mb-2">
                プロプラン
              </h3>
              <div className="text-4xl font-bold text-gray-900 mb-2">
                ¥500
                <span className="text-lg text-gray-500">/月</span>
              </div>
              <p className="text-gray-500">快適に使いたい方向け</p>
            </div>

            <ul className="space-y-4 mb-8">
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700 font-semibold">
                  ベーシックの全機能
                </span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700 font-semibold">広告なし</span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">優先サポート</span>
              </li>
              <li className="flex items-start">
                <span className="text-green-500 mr-2">✓</span>
                <span className="text-gray-700">新機能の優先アクセス</span>
              </li>
            </ul>

            <button
              disabled
              className="w-full py-3 bg-gray-300 text-gray-500 rounded-lg cursor-not-allowed font-semibold"
            >
              近日公開
            </button>
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section className="container mx-auto px-6 py-20 bg-white">
        <h2 className="text-4xl font-bold text-center text-gray-900 mb-16">
          よくある質問
        </h2>

        <div className="max-w-3xl mx-auto space-y-6">
          {/* FAQ 1 */}
          <details className="group border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-shadow">
            <summary className="font-bold text-lg text-gray-900 cursor-pointer flex justify-between items-center">
              <span>無料で使えますか？</span>
              <span className="text-gray-400 group-open:rotate-180 transition-transform">
                ▼
              </span>
            </summary>
            <p className="mt-4 text-gray-600 leading-relaxed">
              はい、ベーシックプラン（Googleログイン後）は完全無料で、無制限にチャンネルを登録できます。匿名プランでも5チャンネルまで登録可能です。
            </p>
          </details>

          {/* FAQ 2 */}
          <details className="group border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-shadow">
            <summary className="font-bold text-lg text-gray-900 cursor-pointer flex justify-between items-center">
              <span>どのプラットフォームに対応していますか？</span>
              <span className="text-gray-400 group-open:rotate-180 transition-transform">
                ▼
              </span>
            </summary>
            <p className="mt-4 text-gray-600 leading-relaxed">
              現在、YouTube、Twitch、Podcastに対応しています。今後、Radiko、アニメ情報、TV番組情報などの対応も予定しています。
            </p>
          </details>

          {/* FAQ 3 */}
          <details className="group border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-shadow">
            <summary className="font-bold text-lg text-gray-900 cursor-pointer flex justify-between items-center">
              <span>匿名プランのデータは本当に削除されますか？</span>
              <span className="text-gray-400 group-open:rotate-180 transition-transform">
                ▼
              </span>
            </summary>
            <p className="mt-4 text-gray-600 leading-relaxed">
              はい、最終アクセスから30日間経過すると、自動的にデータが削除されます。データを永久保存したい場合は、Googleログインでベーシックプランにアップグレードしてください。
            </p>
          </details>

          {/* FAQ 4 */}
          <details className="group border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-shadow">
            <summary className="font-bold text-lg text-gray-900 cursor-pointer flex justify-between items-center">
              <span>スマホでも使えますか？</span>
              <span className="text-gray-400 group-open:rotate-180 transition-transform">
                ▼
              </span>
            </summary>
            <p className="mt-4 text-gray-600 leading-relaxed">
              はい、レスポンシブデザインでスマートフォン、タブレット、PCすべてに対応しています。ベーシックプラン以上であれば、デバイス間でデータが自動的に同期されます。
            </p>
          </details>

          {/* FAQ 5 */}
          <details className="group border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-shadow">
            <summary className="font-bold text-lg text-gray-900 cursor-pointer flex justify-between items-center">
              <span>ライブ配信の通知は受け取れますか？</span>
              <span className="text-gray-400 group-open:rotate-180 transition-transform">
                ▼
              </span>
            </summary>
            <p className="mt-4 text-gray-600 leading-relaxed">
              現在、通知機能は未実装ですが、今後のアップデートで追加予定です。タイムラインページでライブ配信中のコンテンツは「放送中」ラベルが表示されます。
            </p>
          </details>

          {/* FAQ 6 */}
          <details className="group border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-shadow">
            <summary className="font-bold text-lg text-gray-900 cursor-pointer flex justify-between items-center">
              <span>プロプランの決済方法は？</span>
              <span className="text-gray-400 group-open:rotate-180 transition-transform">
                ▼
              </span>
            </summary>
            <p className="mt-4 text-gray-600 leading-relaxed">
              プロプランは現在準備中です。リリース時にはクレジットカード決済（Stripe）に対応予定です。
            </p>
          </details>
        </div>
      </section>

      {/* フッター */}
      <footer className="bg-gray-900 text-white py-12">
        <div className="container mx-auto px-6 text-center">
          <h3 className="text-2xl font-bold mb-4">🎬 Pixicast</h3>
          <p className="text-gray-400 mb-8">
            自分専用のコンテンツ編成表で、見逃しゼロ
          </p>

          <div className="flex justify-center gap-8 mb-8">
            <a
              href="/timeline"
              className="text-gray-400 hover:text-white transition-colors"
            >
              タイムライン
            </a>
            <a
              href="/channels"
              className="text-gray-400 hover:text-white transition-colors"
            >
              チャンネル管理
            </a>
          </div>

          <p className="text-gray-500 text-sm">
            © 2026 Pixicast. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
