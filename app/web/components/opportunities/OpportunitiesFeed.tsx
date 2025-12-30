'use client';

import { useMemo } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { useOpportunitiesSearch } from '@/lib/hooks/useOpportunitiesSearch';
import { type SearchOpportunitiesParams } from '@/lib/api/client';
import { OpportunityCard } from './OpportunityCard';

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
    <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
      {/* Filter Panel */}
      <div
        style={{
          width: 280,
          flexShrink: 0,
          padding: 20,
          backgroundColor: '#f9f9f9',
          borderRadius: 8,
          border: '1px solid #e0e0e0',
        }}
      >
        <div style={{ marginBottom: 24 }}>
          <h2 style={{ margin: '0 0 16px 0', fontSize: 18, fontWeight: 600, color: '#1a1a1a' }}>Filters</h2>
          <button
            onClick={clearFilters}
            style={{
              padding: '6px 12px',
              fontSize: 14,
              backgroundColor: '#fff',
              color: '#333',
              border: '1px solid #ccc',
              borderRadius: 4,
              cursor: 'pointer',
            }}
          >
            Clear All
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* Keyword Search */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              Keyword Search
            </label>
            <input
              type="text"
              value={filters.q || ''}
              onChange={(e) => updateFilters({ q: e.target.value || undefined })}
              placeholder="Search title, solicitation, agency..."
              style={{
                width: '100%',
                padding: '8px 12px',
                fontSize: 14,
                color: '#333',
                backgroundColor: '#fff',
                border: '1px solid #ccc',
                borderRadius: 4,
              }}
            />
          </div>

          {/* NAICS */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              NAICS Code
            </label>
            <input
              type="text"
              value={filters.naics || ''}
              onChange={(e) => updateFilters({ naics: e.target.value || undefined })}
              placeholder="e.g., 335311"
              style={{
                width: '100%',
                padding: '8px 12px',
                fontSize: 14,
                color: '#333',
                backgroundColor: '#fff',
                border: '1px solid #ccc',
                borderRadius: 4,
              }}
            />
          </div>

          {/* Set-Aside */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              Set-Aside
            </label>
            <input
              type="text"
              value={filters.setAside || ''}
              onChange={(e) => updateFilters({ setAside: e.target.value || undefined })}
              placeholder="e.g., SBA"
              style={{
                width: '100%',
                padding: '8px 12px',
                fontSize: 14,
                color: '#333',
                backgroundColor: '#fff',
                border: '1px solid #ccc',
                borderRadius: 4,
              }}
            />
          </div>

          {/* State */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              State
            </label>
            <input
              type="text"
              value={filters.state || ''}
              onChange={(e) => updateFilters({ state: e.target.value || undefined })}
              placeholder="e.g., MO"
              style={{
                width: '100%',
                padding: '8px 12px',
                fontSize: 14,
                color: '#333',
                backgroundColor: '#fff',
                border: '1px solid #ccc',
                borderRadius: 4,
              }}
            />
          </div>

          {/* Agency */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              Agency
            </label>
            <input
              type="text"
              value={filters.agency || ''}
              onChange={(e) => updateFilters({ agency: e.target.value || undefined })}
              placeholder="Agency name..."
              style={{
                width: '100%',
                padding: '8px 12px',
                fontSize: 14,
                color: '#333',
                backgroundColor: '#fff',
                border: '1px solid #ccc',
                borderRadius: 4,
              }}
            />
          </div>

          {/* Posted Date Range */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              Posted Date
            </label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <input
                type="date"
                value={filters.postedFrom || ''}
                onChange={(e) => updateFilters({ postedFrom: e.target.value || undefined })}
                placeholder="From"
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  fontSize: 14,
                  border: '1px solid #ccc',
                  borderRadius: 4,
                }}
              />
              <input
                type="date"
                value={filters.postedTo || ''}
                onChange={(e) => updateFilters({ postedTo: e.target.value || undefined })}
                placeholder="To"
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  fontSize: 14,
                  border: '1px solid #ccc',
                  borderRadius: 4,
                }}
              />
            </div>
          </div>

          {/* Due Date Range */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              Response Deadline
            </label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <input
                type="date"
                value={filters.dueFrom || ''}
                onChange={(e) => updateFilters({ dueFrom: e.target.value || undefined })}
                placeholder="From"
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  fontSize: 14,
                  border: '1px solid #ccc',
                  borderRadius: 4,
                }}
              />
              <input
                type="date"
                value={filters.dueTo || ''}
                onChange={(e) => updateFilters({ dueTo: e.target.value || undefined })}
                placeholder="To"
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  fontSize: 14,
                  border: '1px solid #ccc',
                  borderRadius: 4,
                }}
              />
            </div>
          </div>

          {/* Sort */}
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500, color: '#333' }}>
              Sort By
            </label>
            <select
              value={filters.sort || 'posted_desc'}
              onChange={(e) => updateFilters({ sort: e.target.value as any })}
              style={{
                width: '100%',
                padding: '8px 12px',
                fontSize: 14,
                color: '#333',
                backgroundColor: '#fff',
                border: '1px solid #ccc',
                borderRadius: 4,
              }}
            >
              <option value="posted_desc">Posted Date (Newest First)</option>
              <option value="due_asc">Response Deadline (Earliest First)</option>
              <option value="relevance">Relevance</option>
            </select>
          </div>
        </div>
      </div>

      {/* Results Panel */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {loading && opportunities.length === 0 && (
          <div style={{ textAlign: 'center', padding: 48 }}>
            <p style={{ color: '#666' }}>Loading opportunities...</p>
          </div>
        )}

        {error && opportunities.length === 0 && (
          <div style={{ textAlign: 'center', padding: 48 }}>
            <p style={{ color: '#d32f2f' }}>Error: {error}</p>
          </div>
        )}

        {opportunities.length > 0 && (
          <>
            <div style={{ marginBottom: 16, color: '#666' }}>
              Showing {opportunities.length} opportunity{opportunities.length !== 1 ? 'ies' : ''}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {opportunities.map((opportunity) => (
                <OpportunityCard key={opportunity.noticeId} opportunity={opportunity} />
              ))}
            </div>

            {hasMore && (
              <div style={{ marginTop: 32, textAlign: 'center' }}>
                <button
                  onClick={loadMore}
                  disabled={loading}
                  style={{
                    padding: '12px 24px',
                    fontSize: 16,
                    cursor: loading ? 'not-allowed' : 'pointer',
                    backgroundColor: loading ? '#ccc' : '#0070f3',
                    color: 'white',
                    border: 'none',
                    borderRadius: 4,
                  }}
                >
                  {loading ? 'Loading...' : 'Load More'}
                </button>
              </div>
            )}

            {!hasMore && opportunities.length > 0 && (
              <div style={{ marginTop: 32, textAlign: 'center', color: '#666' }}>
                <p>No more opportunities to load</p>
              </div>
            )}
          </>
        )}

        {!loading && opportunities.length === 0 && !error && (
          <div style={{ textAlign: 'center', padding: 48, color: '#666' }}>
            <p>No opportunities found. Try adjusting your filters.</p>
          </div>
        )}
      </div>
    </div>
  );
}
