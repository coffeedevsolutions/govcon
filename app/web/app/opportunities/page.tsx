import { OpportunitiesFeed } from '@/components/opportunities/OpportunitiesFeed';

export default function OpportunitiesPage() {
  return (
    <main style={{ padding: 24, maxWidth: 1400, margin: '0 auto' }}>
      <h1 style={{ marginBottom: 32, fontSize: 28, fontWeight: 600, color: '#1a1a1a' }}>
        Government Contracting Opportunities
      </h1>
      <OpportunitiesFeed />
    </main>
  );
}

