import Link from 'next/link';
import { type Opportunity } from '@/lib/api/client';

interface OpportunityCardProps {
  opportunity: Opportunity;
}

export function OpportunityCard({ opportunity }: OpportunityCardProps) {
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
      case 'ready':
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
      case 'ready':
        return 'Description Ready';
      case 'available_unfetched':
        return 'Description Available';
      case 'not_found':
        return 'No Description';
      case 'error':
        return 'Error';
      case 'none':
        return 'No Description';
      default:
        return '';
    }
  };

  return (
    <div
      style={{
        border: '1px solid #e0e0e0',
        borderRadius: 8,
        padding: 20,
        backgroundColor: '#fff',
        boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
      }}
    >
      <div style={{ marginBottom: 12 }}>
        <Link
          href={`/opportunities/${opportunity.noticeId}`}
          style={{ textDecoration: 'none', color: 'inherit' }}
        >
          <h2 style={{ margin: 0, fontSize: 20, color: '#0070f3', cursor: 'pointer' }}>
            {opportunity.title || 'Untitled Opportunity'}
          </h2>
        </Link>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
          <p style={{ margin: 0, fontSize: 14, color: '#666' }}>
            Notice ID: {opportunity.noticeId}
          </p>
          {opportunity.descriptionStatus && (
            <span
              style={{
                fontSize: 11,
                padding: '2px 8px',
                borderRadius: 12,
                backgroundColor: getStatusBadgeColor(opportunity.descriptionStatus),
                color: 'white',
                fontWeight: 500,
              }}
            >
              {getStatusLabel(opportunity.descriptionStatus)}
            </span>
          )}
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 12, marginBottom: 12, color: '#333' }}>
        <div>
          <strong style={{ color: '#333' }}>Posted:</strong> {formatDate(opportunity.postedDate)}
        </div>
        {opportunity.responseDeadline && (
          <div>
            <strong style={{ color: '#333' }}>Deadline:</strong> {formatDate(opportunity.responseDeadline)}
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
      </div>


      {opportunity.naics && opportunity.naics.length > 0 && (
        <div style={{ marginBottom: 12, color: '#333' }}>
          <strong style={{ color: '#333' }}>NAICS Codes:</strong>{' '}
          {opportunity.naics.map((naics, idx) => (
            <span key={idx} style={{ marginLeft: 8, color: '#333' }}>
              {naics.code} - {naics.description}
            </span>
          ))}
        </div>
      )}

      {opportunity.pointOfContact && opportunity.pointOfContact.length > 0 && (
        <div style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid #e0e0e0', color: '#333' }}>
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

      {opportunity.links && opportunity.links.length > 0 && (
        <div style={{ marginTop: 12 }}>
          {opportunity.links.map((link, idx) => (
            <a
              key={idx}
              href={link.href}
              target="_blank"
              rel="noopener noreferrer"
              style={{
                marginRight: 12,
                color: '#0070f3',
                textDecoration: 'none',
              }}
            >
              {link.rel || 'View Details'}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}

