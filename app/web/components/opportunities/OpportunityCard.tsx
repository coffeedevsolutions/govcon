import Link from 'next/link';
import { Calendar, MapPin, Building2, FileText, ExternalLink, Clock } from 'lucide-react';
import { type Opportunity } from '@/lib/api/client';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

interface OpportunityCardProps {
  opportunity: Opportunity;
}

export function OpportunityCard({ opportunity }: OpportunityCardProps) {
  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return dateString;
    }
  };

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

  const hasDeadline = opportunity.responseDeadline;
  const isDeadlineSoon = hasDeadline && opportunity.responseDeadline ? new Date(opportunity.responseDeadline) < new Date(Date.now() + 7 * 24 * 60 * 60 * 1000) : false;

  return (
    <Card className="group hover:shadow-lg transition-all duration-200 hover:border-blue-300">
      <CardHeader className="pb-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <Link
              href={`/opportunities/${opportunity.noticeId}`}
              className="block group-hover:text-blue-700 transition-colors"
            >
              <h3 className="text-xl font-semibold text-slate-900 mb-2 line-clamp-2 group-hover:text-blue-700">
                {opportunity.title || 'Untitled Opportunity'}
              </h3>
            </Link>
            <div className="flex items-center gap-3 flex-wrap">
              <p className="text-sm text-slate-500 font-mono">
                {opportunity.noticeId}
              </p>
              {opportunity.descriptionStatus && (
                <Badge variant={getStatusBadgeVariant(opportunity.descriptionStatus)}>
                  {getStatusLabel(opportunity.descriptionStatus)}
                </Badge>
              )}
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Key Information Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="flex items-start gap-3">
            <Calendar className="h-5 w-5 text-blue-600 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wide">Posted</p>
              <p className="text-sm font-medium text-slate-900">{formatDate(opportunity.postedDate)}</p>
            </div>
          </div>

          {opportunity.responseDeadline && (
            <div className="flex items-start gap-3">
              <Clock className={`h-5 w-5 mt-0.5 flex-shrink-0 ${isDeadlineSoon ? 'text-red-600' : 'text-blue-600'}`} />
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wide">Deadline</p>
                <p className={`text-sm font-medium ${isDeadlineSoon ? 'text-red-700' : 'text-slate-900'}`}>
                  {formatDate(opportunity.responseDeadline)}
                </p>
              </div>
            </div>
          )}

          <div className="flex items-start gap-3">
            <FileText className="h-5 w-5 text-blue-600 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wide">Type</p>
              <p className="text-sm font-medium text-slate-900">{opportunity.type || 'N/A'}</p>
            </div>
          </div>

          {opportunity.organizationType && (
            <div className="flex items-start gap-3">
              <Building2 className="h-5 w-5 text-blue-600 mt-0.5 flex-shrink-0" />
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wide">Organization</p>
                <p className="text-sm font-medium text-slate-900">{opportunity.organizationType}</p>
              </div>
            </div>
          )}
        </div>

        {/* NAICS Codes */}
        {opportunity.naics && opportunity.naics.length > 0 && (
          <div className="pt-2 border-t border-blue-100">
            <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">NAICS Codes</p>
            <div className="flex flex-wrap gap-2">
              {opportunity.naics.map((naics, idx) => (
                <Badge key={idx} variant="outline" className="text-xs">
                  {naics.code} - {naics.description}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* Point of Contact */}
        {opportunity.pointOfContact && opportunity.pointOfContact.length > 0 && (
          <div className="pt-2 border-t border-blue-100">
            <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Point of Contact</p>
            <div className="space-y-1">
              {opportunity.pointOfContact.slice(0, 1).map((contact, idx) => (
                <div key={idx} className="text-sm">
                  {contact.fullName && (
                    <p className="font-medium text-slate-900">{contact.fullName}</p>
                  )}
                  {contact.email && (
                    <p className="text-slate-600">{contact.email}</p>
                  )}
                  {contact.phone && (
                    <p className="text-slate-600">{contact.phone}</p>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center gap-3 pt-2 border-t border-blue-100">
          <Link href={`/opportunities/${opportunity.noticeId}`} className="flex-1">
            <Button className="w-full" size="sm">
              View Details
              <ExternalLink className="h-4 w-4 ml-2" />
            </Button>
          </Link>
          {opportunity.uiLink && (
            <Button
              variant="outline"
              size="sm"
              asChild
            >
              <a
                href={opportunity.uiLink}
                target="_blank"
                rel="noopener noreferrer"
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
