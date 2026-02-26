"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { API_BASE_URL } from "@/lib/config";

export interface ChannelSearchResult {
  id: string;
  platform_id: string;
  external_id: string;
  handle?: string;
  display_name: string;
  thumbnail_url?: string;
  subscriber_count?: number;
  is_subscribed: boolean;
  source: string;
}

export interface SearchChannelsResponse {
  results: ChannelSearchResult[];
  total_count: number;
  source: string;
  quota_warning?: string;
}

export interface PopularChannelsResponse {
  results: ChannelSearchResult[];
  total_count: number;
}

const RECENT_SEARCHES_KEY = "pixicast_recent_searches";
const MAX_RECENT_SEARCHES = 5;

export interface RecentSearch {
  query: string;
  timestamp: number;
}

export function getRecentSearches(): RecentSearch[] {
  if (typeof window === "undefined") return [];
  try {
    const stored = localStorage.getItem(RECENT_SEARCHES_KEY);
    if (!stored) return [];
    return JSON.parse(stored) as RecentSearch[];
  } catch {
    return [];
  }
}

export function addRecentSearch(query: string): void {
  if (typeof window === "undefined") return;
  const trimmed = query.trim();
  if (trimmed.length < 2) return;

  const searches = getRecentSearches().filter((s) => s.query !== trimmed);
  searches.unshift({ query: trimmed, timestamp: Date.now() });
  const limited = searches.slice(0, MAX_RECENT_SEARCHES);
  localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(limited));
}

export function clearRecentSearches(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(RECENT_SEARCHES_KEY);
}

export function useChannelSearch(getIdToken: () => Promise<string | null>) {
  const [query, setQuery] = useState("");
  const [platform, setPlatform] = useState("");
  const [results, setResults] = useState<ChannelSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [quotaWarning, setQuotaWarning] = useState("");
  const abortControllerRef = useRef<AbortController | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isComposingRef = useRef(false);

  const search = useCallback(
    async (searchQuery: string, searchPlatform: string) => {
      // Cancel in-flight request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const trimmed = searchQuery.trim();
      if (trimmed.length < 2) {
        setResults([]);
        setError(null);
        setQuotaWarning("");
        return;
      }

      setLoading(true);
      setError(null);

      const controller = new AbortController();
      abortControllerRef.current = controller;

      try {
        const idToken = await getIdToken();
        if (!idToken) {
          setError("認証トークンの取得に失敗しました");
          setLoading(false);
          return;
        }

        const params = new URLSearchParams({ q: trimmed });
        if (searchPlatform) {
          params.set("platform", searchPlatform);
        }

        const response = await fetch(
          `${API_BASE_URL}/v1/channels/search?${params.toString()}`,
          {
            headers: { Authorization: `Bearer ${idToken}` },
            signal: controller.signal,
          }
        );

        if (!response.ok) {
          const data = await response.json();
          if (data.hint === "url_detected") {
            setError("url_detected");
          } else {
            setError(data.error || "検索に失敗しました");
          }
          setResults([]);
          setLoading(false);
          return;
        }

        const data: SearchChannelsResponse = await response.json();
        setResults(data.results || []);
        setQuotaWarning(data.quota_warning || "");

        // Save to recent searches
        addRecentSearch(trimmed);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") {
          return; // Aborted, ignore
        }
        setError("検索中にエラーが発生しました");
        setResults([]);
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    },
    [getIdToken]
  );

  // Debounced search on query/platform change
  useEffect(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    if (query.trim().length < 2) {
      setResults([]);
      setError(null);
      setLoading(false);
      return;
    }

    // IME変換中は検索しない
    if (isComposingRef.current) {
      return;
    }

    debounceTimerRef.current = setTimeout(() => {
      search(query, platform);
    }, 400);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [query, platform, search]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  const onCompositionStart = useCallback(() => {
    isComposingRef.current = true;
  }, []);

  const onCompositionEnd = useCallback(() => {
    isComposingRef.current = false;
    // IME確定後に検索をトリガー（queryのuseEffectはcomposing中スキップされるため）
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      search(query, platform);
    }, 400);
  }, [query, platform, search]);

  return {
    query,
    setQuery,
    platform,
    setPlatform,
    results,
    loading,
    error,
    quotaWarning,
    onCompositionStart,
    onCompositionEnd,
  };
}
