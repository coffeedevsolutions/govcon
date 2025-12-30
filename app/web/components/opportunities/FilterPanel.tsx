'use client';

import { useMemo } from 'react';
import { Search, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { type SearchOpportunitiesParams } from '@/lib/api/client';

interface FilterPanelProps {
  filters: SearchOpportunitiesParams;
  onFiltersChange: (updates: Partial<SearchOpportunitiesParams>) => void;
  onClearFilters: () => void;
}

export function FilterPanel({ filters, onFiltersChange, onClearFilters }: FilterPanelProps) {
  // Count active filters
  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (filters.q) count++;
    if (filters.naics) count++;
    if (filters.setAside) count++;
    if (filters.state) count++;
    if (filters.agency) count++;
    if (filters.postedFrom) count++;
    if (filters.postedTo) count++;
    if (filters.dueFrom) count++;
    if (filters.dueTo) count++;
    return count;
  }, [filters]);

  const hasActiveFilters = activeFilterCount > 0;

  return (
    <div className="w-full lg:w-80 flex-shrink-0">
      <div className="bg-white rounded-lg border border-blue-200 shadow-sm p-6 sticky top-8">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-slate-900">Filters</h2>
          {hasActiveFilters && (
            <Badge variant="secondary" className="bg-blue-100 text-blue-700">
              {activeFilterCount}
            </Badge>
          )}
        </div>

        {hasActiveFilters && (
          <Button
            onClick={onClearFilters}
            variant="outline"
            size="sm"
            className="w-full mb-6 border-blue-200 hover:bg-blue-50"
          >
            <X className="h-4 w-4 mr-2" />
            Clear All Filters
          </Button>
        )}

        <div className="space-y-6">
          {/* Keyword Search */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              <Search className="inline h-4 w-4 mr-1" />
              Keyword Search
            </label>
            <Input
              type="text"
              value={filters.q || ''}
              onChange={(e) => onFiltersChange({ q: e.target.value || undefined })}
              placeholder="Search title, solicitation, agency..."
              className="w-full"
            />
          </div>

          {/* NAICS */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              NAICS Code
            </label>
            <Input
              type="text"
              value={filters.naics || ''}
              onChange={(e) => onFiltersChange({ naics: e.target.value || undefined })}
              placeholder="e.g., 335311"
              className="w-full"
            />
          </div>

          {/* Set-Aside */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Set-Aside
            </label>
            <Input
              type="text"
              value={filters.setAside || ''}
              onChange={(e) => onFiltersChange({ setAside: e.target.value || undefined })}
              placeholder="e.g., SBA"
              className="w-full"
            />
          </div>

          {/* State */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              State
            </label>
            <Input
              type="text"
              value={filters.state || ''}
              onChange={(e) => onFiltersChange({ state: e.target.value || undefined })}
              placeholder="e.g., MO"
              className="w-full"
            />
          </div>

          {/* Agency */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Agency
            </label>
            <Input
              type="text"
              value={filters.agency || ''}
              onChange={(e) => onFiltersChange({ agency: e.target.value || undefined })}
              placeholder="Agency name..."
              className="w-full"
            />
          </div>

          {/* Posted Date Range */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Posted Date
            </label>
            <div className="space-y-2">
              <Input
                type="date"
                value={filters.postedFrom || ''}
                onChange={(e) => onFiltersChange({ postedFrom: e.target.value || undefined })}
                placeholder="From"
                className="w-full"
              />
              <Input
                type="date"
                value={filters.postedTo || ''}
                onChange={(e) => onFiltersChange({ postedTo: e.target.value || undefined })}
                placeholder="To"
                className="w-full"
              />
            </div>
          </div>

          {/* Due Date Range */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Response Deadline
            </label>
            <div className="space-y-2">
              <Input
                type="date"
                value={filters.dueFrom || ''}
                onChange={(e) => onFiltersChange({ dueFrom: e.target.value || undefined })}
                placeholder="From"
                className="w-full"
              />
              <Input
                type="date"
                value={filters.dueTo || ''}
                onChange={(e) => onFiltersChange({ dueTo: e.target.value || undefined })}
                placeholder="To"
                className="w-full"
              />
            </div>
          </div>

          {/* Sort */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Sort By
            </label>
            <select
              value={filters.sort || 'posted_desc'}
              onChange={(e) => onFiltersChange({ sort: e.target.value as any })}
              className="flex h-10 w-full rounded-md border border-blue-200 bg-white px-3 py-2 text-sm ring-offset-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
            >
              <option value="posted_desc">Posted Date (Newest First)</option>
              <option value="due_asc">Response Deadline (Earliest First)</option>
              <option value="relevance">Relevance</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  );
}

