'use client';

import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { type Opportunity } from '@/lib/api/client';
import { useRouter } from 'next/navigation';

interface OpportunityHeaderProps {
  opportunity: Opportunity;
}

export function OpportunityHeader({ opportunity }: OpportunityHeaderProps) {
  const router = useRouter();

  const getStatusBadgeVariant = (status?: string): "default" | "secondary" | "destructive" | "success" | "outline" => {
    switch (status) {
      case 'ready':
        return 'success';
      case 'available_unfetched':
        return 'secondary';
      case 'not_found':
      case 'error':
        return 'destructive';
      case 'none':
        return 'outline';
      default:
        return 'outline';
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
    <div className="mb-8">
      <Button
        onClick={() => router.push('/opportunities')}
        variant="ghost"
        className="mb-6 text-slate-600 hover:text-blue-700"
      >
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Opportunities
      </Button>

      <div className="bg-gradient-to-r from-blue-50 to-white rounded-lg border border-blue-200 p-6 mb-6">
        <div className="flex items-start justify-between gap-4 mb-4">
          <div className="flex-1">
            <h1 className="text-3xl font-bold text-slate-900 mb-2">
              {opportunity.title || 'Untitled Opportunity'}
            </h1>
            <p className="text-sm text-slate-500 font-mono">
              Notice ID: {opportunity.noticeId}
            </p>
          </div>
          {opportunity.descriptionStatus && (
            <Badge variant={getStatusBadgeVariant(opportunity.descriptionStatus)} className="text-sm">
              {getStatusLabel(opportunity.descriptionStatus)}
            </Badge>
          )}
        </div>

        {opportunity.solicitationNumber && (
          <div className="mt-4 pt-4 border-t border-blue-200">
            <p className="text-sm text-slate-600">
              <span className="font-medium">Solicitation Number:</span> {opportunity.solicitationNumber}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

