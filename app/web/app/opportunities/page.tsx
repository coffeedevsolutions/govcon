import { Suspense } from 'react';
import { OpportunitiesFeed } from '@/components/opportunities/OpportunitiesFeed';
import { Skeleton } from '@/components/ui/skeleton';

function OpportunitiesFeedSkeleton() {
  return (
    <div className="flex flex-col lg:flex-row gap-6 lg:gap-8">
      <div className="w-full lg:w-80">
        <div className="bg-white rounded-lg border border-blue-200 shadow-sm p-6">
          <Skeleton className="h-6 w-24 mb-6" />
          <div className="space-y-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </div>
      </div>
      <div className="flex-1 space-y-4">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="bg-white rounded-lg border border-blue-200 p-6">
            <Skeleton className="h-6 w-3/4 mb-4" />
            <Skeleton className="h-4 w-full mb-2" />
            <Skeleton className="h-4 w-2/3" />
          </div>
        ))}
      </div>
    </div>
  );
}

export default function OpportunitiesPage() {
  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8 animate-fade-in">
          <h1 className="text-4xl font-bold text-slate-900 mb-2 bg-gradient-to-r from-blue-900 to-blue-700 bg-clip-text text-transparent">
            Government Contracting Opportunities
          </h1>
          <p className="text-slate-600 text-lg">
            Discover and explore federal contracting opportunities
          </p>
        </div>
        <Suspense fallback={<OpportunitiesFeedSkeleton />}>
          <OpportunitiesFeed />
        </Suspense>
      </div>
    </main>
  );
}
