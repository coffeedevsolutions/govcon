'use client';

import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { apiClient, type SearchOpportunitiesParams, type SearchOpportunitiesResponse, type Opportunity } from '@/lib/api/client';

interface UseOpportunitiesSearchOptions {
  params: SearchOpportunitiesParams;
  debounceMs?: number;
}

interface UseOpportunitiesSearchResult {
  opportunities: Opportunity[];
  loading: boolean;
  error: string | null;
  hasMore: boolean;
  loadMore: () => void;
  refetch: () => void;
}

export function useOpportunitiesSearch(
  options: UseOpportunitiesSearchOptions
): UseOpportunitiesSearchResult {
  const { params, debounceMs = 300 } = options;
  const [opportunities, setOpportunities] = useState<Opportunity[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const currentParamsKeyRef = useRef<string>('');
  
  const abortControllerRef = useRef<AbortController | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchOpportunities = useCallback(
    async (searchParams: SearchOpportunitiesParams, append: boolean = false) => {
      // Cancel previous request if still pending
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      try {
        setLoading(true);
        setError(null);

        const response: SearchOpportunitiesResponse = await apiClient.searchOpportunities(searchParams);

        if (abortController.signal.aborted) {
          return;
        }

        if (append) {
          setOpportunities((prev) => [...prev, ...response.items]);
        } else {
          setOpportunities(response.items);
        }

        setNextCursor(response.nextCursor);
      } catch (err) {
        if (abortController.signal.aborted) {
          return;
        }
        const errorMessage = err instanceof Error ? err.message : 'Failed to load opportunities';
        setError(errorMessage);
        console.error('Error fetching opportunities:', err);
      } finally {
        if (!abortController.signal.aborted) {
          setLoading(false);
        }
      }
    },
    []
  );

  // Serialize params for comparison (excluding cursor)
  const paramsKey = useMemo(() => {
    const paramsWithoutCursor = { ...params };
    delete paramsWithoutCursor.cursor;
    return JSON.stringify(paramsWithoutCursor);
  }, [params]);

  // Debounced effect for search params changes
  useEffect(() => {
    // Clear previous debounce timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Check if params actually changed
    const paramsChanged = paramsKey !== currentParamsKeyRef.current;
    if (paramsChanged) {
      setNextCursor(null);
      currentParamsKeyRef.current = paramsKey;
    }

    // Debounce the fetch
    debounceTimerRef.current = setTimeout(() => {
      const fetchParams = { ...params };
      if (paramsChanged) {
        delete fetchParams.cursor; // Reset cursor on param change
      }
      fetchOpportunities(fetchParams, false);
    }, debounceMs);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [paramsKey, debounceMs, fetchOpportunities, params]);

  const loadMore = useCallback(() => {
    if (nextCursor && !loading) {
      fetchOpportunities({ ...params, cursor: nextCursor }, true);
    }
  }, [nextCursor, loading, params, fetchOpportunities]);

  const refetch = useCallback(() => {
    const fetchParams = { ...params };
    delete fetchParams.cursor;
    fetchOpportunities(fetchParams, false);
  }, [params, fetchOpportunities]);

  return {
    opportunities,
    loading,
    error,
    hasMore: nextCursor !== null,
    loadMore,
    refetch,
  };
}

