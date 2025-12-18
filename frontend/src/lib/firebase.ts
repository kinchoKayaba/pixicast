import { initializeApp } from "firebase/app";
import { getAuth } from "firebase/auth";

const firebaseConfig = {
  apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
  authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
  storageBucket: process.env.NEXT_PUBLIC_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: process.env.NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID,
  appId: process.env.NEXT_PUBLIC_FIREBASE_APP_ID,
  measurementId: process.env.NEXT_PUBLIC_FIREBASE_MEASUREMENT_ID
};

console.log("🔥 Firebase初期化開始");
console.log("Firebase Config:", {
  projectId: firebaseConfig.projectId,
  authDomain: firebaseConfig.authDomain,
});

let app;
let auth;

try {
  app = initializeApp(firebaseConfig);
  console.log("✅ Firebase App初期化成功");
  auth = getAuth(app);
  console.log("✅ Firebase Auth初期化成功");
} catch (error) {
  console.error("❌ Firebase初期化エラー:", error);
  throw error;
}

export { auth };

