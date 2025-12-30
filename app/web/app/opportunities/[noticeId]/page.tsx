'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiClient, type Opportunity, type OpportunityDescription } from '@/lib/api/client';

export default function OpportunityDetailPage() {
  const params = useParams();
  const router = useRouter();
  const noticeId = params.noticeId as string;

  const [opportunity, setOpportunity] = useState<Opportunity | null>(null);
  const [description, setDescription] = useState<OpportunityDescription | null>(null);
  const [activeTab, setActiveTab] = useState<'normalized' | 'raw'>('normalized');
  const [loading, setLoading] = useState(true);
  const [descriptionLoading, setDescriptionLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    async function fetchOpportunity() {
      try {
        setLoading(true);
        const opp = await apiClient.getOpportunity(noticeId);
        setOpportunity(opp);
      } catch (err: any) {
        setError(err.message || 'Failed to load opportunity');
      } finally {
        setLoading(false);
      }
    }

    if (noticeId) {
      fetchOpportunity();
    }
  }, [noticeId]);

  useEffect(() => {
    async function fetchDescription() {
      if (!noticeId) return;

      try {
        setDescriptionLoading(true);
        const desc = await apiClient.getOpportunityDescription(noticeId);
        setDescription(desc);
      } catch (err: any) {
        setError(err.message || 'Failed to load description');
      } finally {
        setDescriptionLoading(false);
      }
    }

    if (noticeId) {
      fetchDescription();
    }
  }, [noticeId]);

  const handleRefresh = async () => {
    if (!noticeId) return;

    try {
      setRefreshing(true);
      const desc = await apiClient.getOpportunityDescription(noticeId, true);
      setDescription(desc);
    } catch (err: any) {
      setError(err.message || 'Failed to refresh description');
    } finally {
      setRefreshing(false);
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleDateString();
    } catch {
      return dateString;
    }
  };

  const getStatusBadgeColor = (status?: string) => {
    switch (status) {
      case 'fetched':
        return '#4caf50';
      case 'available_unfetched':
        return '#ff9800';
      case 'not_found':
        return '#f44336';
      case 'error':
        return '#f44336';
      case 'none':
        return '#9e9e9e';
      default:
        return '#9e9e9e';
    }
  };

  const getStatusLabel = (status?: string) => {
    switch (status) {
      case 'fetched':
        return 'Ready';
      case 'available_unfetched':
        return 'Available (Not Fetched)';
      case 'not_found':
        return 'Not Found';
      case 'error':
        return 'Error';
      case 'none':
        return 'No Description';
      default:
        return 'Unknown';
    }
  };

  if (loading) {
    return (
      <main style={{ padding: 24, maxWidth: 1200, margin: '0 auto' }}>
        <p>Loading opportunity...</p>
      </main>
    );
  }

  if (error && !opportunity) {
    return (
      <main style={{ padding: 24, maxWidth: 1200, margin: '0 auto' }}>
        <p style={{ color: '#d32f2f' }}>Error: {error}</p>
        <button
          onClick={() => router.push('/opportunities')}
          style={{
            marginTop: 16,
            padding: '8px 16px',
            backgroundColor: '#0070f3',
            color: 'white',
            border: 'none',
            borderRadius: 4,
            cursor: 'pointer',
          }}
        >
          Back to Opportunities
        </button>
      </main>
    );
  }

  if (!opportunity) {
    return (
      <main style={{ padding: 24, maxWidth: 1200, margin: '0 auto' }}>
        <p>Opportunity not found</p>
      </main>
    );
  }

  return (
    <main style={{ padding: 24, maxWidth: 1200, margin: '0 auto' }}>
      <button
        onClick={() => router.push('/opportunities')}
        style={{
          marginBottom: 24,
          padding: '8px 16px',
          backgroundColor: '#f5f5f5',
          color: '#333',
          border: '1px solid #e0e0e0',
          borderRadius: 4,
          cursor: 'pointer',
        }}
      >
        ← Back to Opportunities
      </button>

      <div style={{ marginBottom: 32 }}>
        <h1 style={{ margin: '0 0 8px 0', fontSize: 32, color: '#1a1a1a' }}>
          {opportunity.title || 'Untitled Opportunity'}
        </h1>
        <p style={{ margin: '4px 0', fontSize: 14, color: '#666' }}>
          Notice ID: {opportunity.noticeId}
        </p>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
          gap: 16,
          marginBottom: 32,
          padding: 16,
          backgroundColor: '#f9f9f9',
          borderRadius: 8,
        }}
      >
        <div>
          <strong style={{ color: '#333' }}>Posted:</strong> {formatDate(opportunity.postedDate)}
        </div>
        {opportunity.responseDeadline && (
          <div>
            <strong style={{ color: '#333' }}>Deadline:</strong>{' '}
            {formatDate(opportunity.responseDeadline)}
          </div>
        )}
        <div>
          <strong style={{ color: '#333' }}>Type:</strong> {opportunity.type || 'N/A'}
        </div>
        {opportunity.organizationType && (
          <div>
            <strong style={{ color: '#333' }}>Organization:</strong> {opportunity.organizationType}
          </div>
        )}
        {opportunity.solicitationNumber && (
          <div>
            <strong style={{ color: '#333' }}>Solicitation #:</strong>{' '}
            {opportunity.solicitationNumber}
          </div>
        )}
        {opportunity.agencyPathName && (
          <div>
            <strong style={{ color: '#333' }}>Agency:</strong> {opportunity.agencyPathName}
          </div>
        )}
      </div>

      {/* Description Section */}
      <div
        style={{
          border: '1px solid #e0e0e0',
          borderRadius: 8,
          padding: 24,
          backgroundColor: '#fff',
          boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
        }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 16,
            paddingBottom: 16,
            borderBottom: '1px solid #e0e0e0',
          }}
        >
          <h2 style={{ margin: 0, fontSize: 24, color: '#1a1a1a' }}>Description</h2>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            {description && (
              <span
                style={{
                  fontSize: 12,
                  padding: '4px 12px',
                  borderRadius: 12,
                  backgroundColor: getStatusBadgeColor(description.status),
                  color: 'white',
                  fontWeight: 500,
                }}
              >
                {getStatusLabel(description.status)}
              </span>
            )}
            <button
              onClick={handleRefresh}
              disabled={refreshing || descriptionLoading}
              style={{
                padding: '8px 16px',
                fontSize: 14,
                backgroundColor: refreshing || descriptionLoading ? '#ccc' : '#0070f3',
                color: 'white',
                border: 'none',
                borderRadius: 4,
                cursor: refreshing || descriptionLoading ? 'not-allowed' : 'pointer',
              }}
            >
              {refreshing ? 'Refreshing...' : 'Refresh'}
            </button>
          </div>
        </div>

        {descriptionLoading && !description && (
          <p style={{ color: '#666' }}>Loading description...</p>
        )}

        {description && (
          <>
            {description.status === 'fetched' && (
              <>
                {/* Tabs */}
                <div style={{ display: 'flex', gap: 8, marginBottom: 16, borderBottom: '1px solid #e0e0e0' }}>
                  <button
                    onClick={() => setActiveTab('normalized')}
                    style={{
                      padding: '8px 16px',
                      fontSize: 14,
                      backgroundColor: 'transparent',
                      color: activeTab === 'normalized' ? '#0070f3' : '#666',
                      border: 'none',
                      borderBottom: activeTab === 'normalized' ? '2px solid #0070f3' : '2px solid transparent',
                      cursor: 'pointer',
                      fontWeight: activeTab === 'normalized' ? 600 : 400,
                    }}
                  >
                    Normalized
                  </button>
                  <button
                    onClick={() => setActiveTab('raw')}
                    style={{
                      padding: '8px 16px',
                      fontSize: 14,
                      backgroundColor: 'transparent',
                      color: activeTab === 'raw' ? '#0070f3' : '#666',
                      border: 'none',
                      borderBottom: activeTab === 'raw' ? '2px solid #0070f3' : '2px solid transparent',
                      cursor: 'pointer',
                      fontWeight: activeTab === 'raw' ? 600 : 400,
                    }}
                  >
                    Raw Post-Parse
                  </button>
                </div>

                {/* Content */}
                <div
                  style={{
                    padding: 16,
                    backgroundColor: '#f9f9f9',
                    borderRadius: 4,
                    maxHeight: '600px',
                    overflowY: 'auto',
                  }}
                >
                  <pre
                    style={{
                      margin: 0,
                      whiteSpace: 'pre-wrap',
                      wordWrap: 'break-word',
                      fontFamily: 'inherit',
                      fontSize: 14,
                      lineHeight: 1.6,
                      color: '#333',
                    }}
                  >
                    {activeTab === 'normalized'
                      ? description.normalizedText || 'No normalized text available'
                      : (() => {
                          // Show raw JSON response if available, otherwise fall back to raw post-parse text
                          if (description.rawJsonResponse) {
                            try {
                              // Try to pretty-print JSON
                              const parsed = JSON.parse(description.rawJsonResponse);
                              return JSON.stringify(parsed, null, 2);
                            } catch {
                              // If not valid JSON, show as-is
                              return description.rawJsonResponse;
                            }
                          }
                          return description.rawPostParseText || 'No raw post-parse text available';
                        })()}
                  </pre>
                </div>
              </>
            )}

            {description.status === 'none' && (
              <p style={{ color: '#666' }}>No description available for this opportunity.</p>
            )}

            {description.status === 'not_found' && (
              <p style={{ color: '#666' }}>
                Description not found. The description may not be available from the source.
              </p>
            )}

            {description.status === 'error' && (
              <p style={{ color: '#d32f2f' }}>
                Error loading description. Please try refreshing.
              </p>
            )}

            {description.status === 'available_unfetched' && (
              <p style={{ color: '#666' }}>
                Description is available but has not been fetched yet. Click Refresh to fetch it.
              </p>
            )}
          </>
        )}

        {!description && !descriptionLoading && (
          <p style={{ color: '#666' }}>No description data available.</p>
        )}
      </div>

      {/* Additional Details */}
      {opportunity.naics && opportunity.naics.length > 0 && (
        <div
          style={{
            marginTop: 32,
            padding: 16,
            backgroundColor: '#f9f9f9',
            borderRadius: 8,
          }}
        >
          <strong style={{ color: '#333' }}>NAICS Codes:</strong>
          <div style={{ marginTop: 8 }}>
            {opportunity.naics.map((naics, idx) => (
              <div key={idx} style={{ marginBottom: 4, color: '#333' }}>
                {naics.code} - {naics.description}
              </div>
            ))}
          </div>
        </div>
      )}

      {opportunity.pointOfContact && opportunity.pointOfContact.length > 0 && (
        <div
          style={{
            marginTop: 32,
            padding: 16,
            backgroundColor: '#f9f9f9',
            borderRadius: 8,
          }}
        >
          <strong style={{ color: '#333' }}>Point of Contact:</strong>
          {opportunity.pointOfContact.map((contact, idx) => (
            <div key={idx} style={{ marginTop: 8, fontSize: 14, color: '#333' }}>
              {contact.fullName && <div>{contact.fullName}</div>}
              {contact.email && <div>{contact.email}</div>}
              {contact.phone && <div>{contact.phone}</div>}
            </div>
          ))}
        </div>
      )}
    </main>
  );
}

