"use client";

import { createContext, useContext, useEffect, useState, ReactNode } from "react";
import { auth } from "@/lib/firebase";
import {
  User,
  signInAnonymously,
  signInWithPopup,
  GoogleAuthProvider,
  signOut as firebaseSignOut,
  onAuthStateChanged,
} from "firebase/auth";

interface AuthContextType {
  user: User | null;
  loading: boolean;
  isAnonymous: boolean;
  signInWithGoogle: () => Promise<void>;
  signOut: () => Promise<void>;
  getIdToken: () => Promise<string | null>;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  isAnonymous: false,
  signInWithGoogle: async () => {},
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
      const provider = new GoogleAuthProvider();
      await signInWithPopup(auth, provider);
    } catch (error) {
      console.error("Googleログインエラー:", error);
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
        signOut,
        getIdToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);

