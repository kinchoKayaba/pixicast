import { auth } from "@/auth";
import Timeline from "@/components/Timeline";
import { SignInButton, SignOutButton } from "@/components/AuthButtons";

export default async function Home() {
  // サーバーサイドでセッション情報を取得
  const session = await auth();

  return (
    <>
      {/* ログインしている場合 */}
      {session?.user ? (
        <div className="min-h-screen bg-pink-50">
          <div className="absolute top-4 right-4 z-10 flex items-center gap-2">
            <span className="text-xs text-gray-600">
              {session.user.name} さん
            </span>
            {/* アイコンがあれば表示 */}
            {session.user.image && (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={session.user.image}
                alt="icon"
                className="w-8 h-8 rounded-full border border-gray-300"
              />
            )}
            <SignOutButton />
          </div>
          {/* 番組表を表示 */}
          <Timeline />
        </div>
      ) : (
        /* ログインしていない場合 */
        <div className="flex flex-col items-center justify-center min-h-screen bg-pink-50">
          <h1 className="text-3xl font-bold text-gray-800 mb-8">Pixicast</h1>
          <div className="bg-white p-8 rounded-xl shadow-lg text-center">
            <p className="mb-6 text-gray-600">自分専用のタイムラインを作ろう</p>
            <SignInButton />
          </div>
        </div>
      )}
    </>
  );
}
