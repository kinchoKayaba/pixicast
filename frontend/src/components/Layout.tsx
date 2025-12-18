"use client";

import { useState, Suspense } from "react";
import Sidebar from "./Sidebar";
import MenuButton from "./MenuButton";

export default function Layout({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(true);

  return (
    <>
      <MenuButton onClick={() => setSidebarOpen(!sidebarOpen)} />
      <Suspense fallback={<div>Loading...</div>}>
        <Sidebar isOpen={sidebarOpen} onToggle={() => setSidebarOpen(!sidebarOpen)} />
      </Suspense>
      <main
        className={`transition-all duration-300 ${
          sidebarOpen ? "ml-64" : "ml-16"
        }`}
      >
        <Suspense fallback={<div>Loading...</div>}>
          {children}
        </Suspense>
      </main>
    </>
  );
}

