"use client";

import { createContext, useContext, useEffect, useState, ReactNode } from "react";
import { auth } from "@/lib/firebase";
import {
  User,
  signInAnonymously,
  signInWithPopup,
  linkWithPopup,
  GoogleAuthProvider,
  signOut as firebaseSignOut,
  onAuthStateChanged,
} from "firebase/auth";

interface AuthContextType {
  user: User | null;
  loading: boolean;
  isAnonymous: boolean;
  signInWithGoogle: () => Promise<void>;
  signInAnonymously: () => Promise<void>;
  signOut: () => Promise<void>;
  getIdToken: () => Promise<string | null>;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  isAnonymous: false,
  signInWithGoogle: async () => {},
  signInAnonymously: async () => {},
  signOut: async () => {},
  getIdToken: async () => null,
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    console.log("🔥 Firebase Auth 初期化開始");
    const unsubscribe = onAuthStateChanged(auth, async (user) => {
      console.log("🔥 onAuthStateChanged triggered, user:", user?.uid || "null");
      if (user) {
        console.log("✅ 既存ユーザー検出:", user.uid, "isAnonymous:", user.isAnonymous);
        setUser(user);
        setLoading(false);
      } else {
        console.log("⚠️ ユーザーなし、匿名ログイン開始...");
        try {
          const result = await signInAnonymously(auth);
          console.log("✅ 匿名ログイン成功:", result.user.uid);
          setUser(result.user);
          setLoading(false);
        } catch (error) {
          console.error("❌ 匿名ログインエラー:", error);
          console.error("Error details:", JSON.stringify(error, null, 2));
          setLoading(false);
        }
      }
    });

    return () => {
      console.log("🔥 Firebase Auth クリーンアップ");
      unsubscribe();
    };
  }, []);

  const signInWithGoogle = async () => {
    try {
      console.log("🔐 Googleログイン開始, 現在のユーザー:", user?.uid, "isAnonymous:", user?.isAnonymous);
      
      const provider = new GoogleAuthProvider();
      
      // 匿名ユーザーの場合は、linkWithPopupで同じUIDのまま昇格を試みる
      if (user?.isAnonymous) {
        console.log("⬆️ 匿名ユーザーを正規ユーザーに昇格（同じUID維持）");
        try {
          const result = await linkWithPopup(user, provider);
          console.log("✅ アカウントリンク成功:", result.user.uid, "email:", result.user.email);
          console.log("🎉 UIDは維持されたまま正規ユーザーに昇格しました");
        } catch (linkError: any) {
          // credential-already-in-use の場合は、既存アカウントでログイン
          if (linkError?.code === 'auth/credential-already-in-use') {
            console.log("⚠️ このGoogleアカウントは既に存在します。既存アカウントでログインします");
            // 匿名ユーザーを削除してから既存アカウントでログイン
            await firebaseSignOut(auth);
            const result = await signInWithPopup(auth, provider);
            console.log("✅ 既存アカウントでログイン成功:", result.user.uid, "email:", result.user.email);
          } else {
            throw linkError;
          }
        }
      } else {
        // 既に正規ユーザーの場合は通常のログイン
        const result = await signInWithPopup(auth, provider);
        console.log("✅ Googleログイン成功:", result.user.uid, "email:", result.user.email);
      }
    } catch (error) {
      console.error("❌ Googleログインエラー:", error);
      throw error;
    }
  };

  const signOut = async () => {
    try {
      await firebaseSignOut(auth);
    } catch (error) {
      console.error("ログアウトエラー:", error);
      throw error;
    }
  };

  const doSignInAnonymously = async () => {
    try {
      const result = await signInAnonymously(auth);
      console.log("✅ 匿名ログイン成功:", result.user.uid);
      setUser(result.user);
    } catch (error) {
      console.error("❌ 匿名ログインエラー:", error);
      throw error;
    }
  };

  const getIdToken = async (): Promise<string | null> => {
    console.log("🎫 getIdToken called, user:", user?.uid || "null");
    if (!user) {
      console.error("❌ getIdToken: user is null");
      return null;
    }
    try {
      const token = await user.getIdToken();
      console.log("✅ Token取得成功, length:", token?.length || 0);
      return token;
    } catch (error) {
      console.error("❌ トークン取得エラー:", error);
      return null;
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAnonymous: user?.isAnonymous ?? false,
        signInWithGoogle,
        signInAnonymously: doSignInAnonymously,
        signOut,
        getIdToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);

