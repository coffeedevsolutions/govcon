'use client';

import { useMemo } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import { useOpportunitiesSearch } from '@/lib/hooks/useOpportunitiesSearch';
import { type SearchOpportunitiesParams } from '@/lib/api/client';
import { OpportunityCard } from './OpportunityCard';
import { FilterPanel } from './FilterPanel';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';

export function OpportunitiesFeed() {
  const searchParams = useSearchParams();
  const router = useRouter();

  // Extract filter values from URL - memoize to prevent infinite loops
  const filters: SearchOpportunitiesParams = useMemo(() => ({
    q: searchParams.get('q') || undefined,
    naics: searchParams.get('naics') || undefined,
    setAside: searchParams.get('setAside') || undefined,
    state: searchParams.get('state') || undefined,
    agency: searchParams.get('agency') || undefined,
    postedFrom: searchParams.get('postedFrom') || undefined,
    postedTo: searchParams.get('postedTo') || undefined,
    dueFrom: searchParams.get('dueFrom') || undefined,
    dueTo: searchParams.get('dueTo') || undefined,
    sort: (searchParams.get('sort') as 'posted_desc' | 'due_asc' | 'relevance') || 'posted_desc',
    limit: 25,
  }), [
    searchParams.get('q'),
    searchParams.get('naics'),
    searchParams.get('setAside'),
    searchParams.get('state'),
    searchParams.get('agency'),
    searchParams.get('postedFrom'),
    searchParams.get('postedTo'),
    searchParams.get('dueFrom'),
    searchParams.get('dueTo'),
    searchParams.get('sort'),
  ]);

  const { opportunities, loading, error, hasMore, loadMore } = useOpportunitiesSearch({
    params: filters,
    debounceMs: 300,
  });

  // Update URL when filters change
  const updateFilters = (updates: Partial<SearchOpportunitiesParams>) => {
    const newParams = new URLSearchParams(searchParams.toString());
    
    Object.entries(updates).forEach(([key, value]) => {
      if (value === undefined || value === '') {
        newParams.delete(key);
      } else {
        newParams.set(key, String(value));
      }
    });

    router.push(`/opportunities?${newParams.toString()}`);
  };

  const clearFilters = () => {
    router.push('/opportunities');
  };

  return (
    <div className="flex flex-col lg:flex-row gap-6 lg:gap-8">
      {/* Filter Panel */}
      <FilterPanel
        filters={filters}
        onFiltersChange={updateFilters}
        onClearFilters={clearFilters}
      />

      {/* Results Panel */}
      <div className="flex-1 min-w-0">
        {loading && opportunities.length === 0 && (
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="bg-white rounded-lg border border-blue-200 p-6">
                <Skeleton className="h-6 w-3/4 mb-4" />
                <Skeleton className="h-4 w-full mb-2" />
                <Skeleton className="h-4 w-2/3" />
              </div>
            ))}
          </div>
        )}

        {error && opportunities.length === 0 && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
            <p className="text-red-700 font-medium">Error: {error}</p>
            <p className="text-red-600 text-sm mt-2">Please try adjusting your filters or refresh the page.</p>
          </div>
        )}

        {opportunities.length > 0 && (
          <>
            <div className="mb-6 flex items-center justify-between">
              <p className="text-slate-600">
                Showing <span className="font-semibold text-slate-900">{opportunities.length}</span> opportunity{opportunities.length !== 1 ? 'ies' : ''}
              </p>
            </div>

            <div className="space-y-4">
              {opportunities.map((opportunity) => (
                <OpportunityCard key={opportunity.noticeId} opportunity={opportunity} />
              ))}
            </div>

            {hasMore && (
              <div className="mt-8 text-center">
                <Button
                  onClick={loadMore}
                  disabled={loading}
                  className="min-w-[120px]"
                >
                  {loading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Loading...
                    </>
                  ) : (
                    'Load More'
                  )}
                </Button>
              </div>
            )}

            {!hasMore && opportunities.length > 0 && (
              <div className="mt-8 text-center">
                <p className="text-slate-500 text-sm">No more opportunities to load</p>
              </div>
            )}
          </>
        )}

        {!loading && opportunities.length === 0 && !error && (
          <div className="bg-white rounded-lg border border-blue-200 p-12 text-center">
            <div className="max-w-md mx-auto">
              <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-blue-100 flex items-center justify-center">
                <svg className="w-8 h-8 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">No opportunities found</h3>
              <p className="text-slate-600 mb-4">Try adjusting your filters to find more results.</p>
              <Button onClick={clearFilters} variant="outline">
                Clear All Filters
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
