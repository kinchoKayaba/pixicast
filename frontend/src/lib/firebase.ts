import { initializeApp } from "firebase/app";
import { getAuth } from "firebase/auth";

const firebaseConfig = {
  apiKey: "AIzaSyDyrWPMYGJCYfPyBarRnyGe-9e_GtZ1AIA",
  authDomain: "pixicast-924a4.firebaseapp.com",
  projectId: "pixicast-924a4",
  storageBucket: "pixicast-924a4.firebasestorage.app",
  messagingSenderId: "741846019456",
  appId: "1:741846019456:web:47b246bbad9e3cb637e565",
  measurementId: "G-CH3LBGKEZ2"
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

