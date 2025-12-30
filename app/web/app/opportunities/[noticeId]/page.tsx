'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { 
  Calendar, 
  Clock, 
  FileText, 
  MapPin, 
  Building2, 
  Mail, 
  Phone, 
  Printer, 
  User, 
  ExternalLink,
  Download,
  RefreshCw,
  AlertCircle
} from 'lucide-react';
import { apiClient, type Opportunity, type OpportunityDescription } from '@/lib/api/client';
import { OpportunityHeader } from '@/components/opportunities/OpportunityHeader';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Skeleton } from '@/components/ui/skeleton';

type MainTab = 'overview' | 'details' | 'location' | 'contacts' | 'resources' | 'description';
type DescriptionTab = 'normalized' | 'raw';

export default function OpportunityDetailPage() {
  const params = useParams();
  const noticeId = params.noticeId as string;

  const [opportunity, setOpportunity] = useState<Opportunity | null>(null);
  const [description, setDescription] = useState<OpportunityDescription | null>(null);
  const [activeMainTab, setActiveMainTab] = useState<MainTab>('overview');
  const [activeDescriptionTab, setActiveDescriptionTab] = useState<DescriptionTab>('normalized');
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
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'long',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return dateString;
    }
  };

  const formatDateTime = (dateString?: string) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString('en-US', {
        month: 'long',
        day: 'numeric',
        year: 'numeric',
        hour: 'numeric',
        minute: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  const getStatusBadgeVariant = (status?: string): "default" | "secondary" | "destructive" | "success" | "outline" => {
    switch (status) {
      case 'fetched':
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

  const renderField = (label: string, value: string | null | undefined, icon?: React.ReactNode) => {
    if (!value) return null;
    return (
      <div className="space-y-1">
        <div className="flex items-center gap-2 text-sm text-slate-500">
          {icon}
          <span className="font-medium">{label}</span>
        </div>
        <p className="text-slate-900 pl-6">{value}</p>
      </div>
    );
  };

  const renderPlaceOfPerformance = () => {
    if (!opportunity?.placeOfPerformance) return null;

    const pop = opportunity.placeOfPerformance;
    const cityStr = typeof pop.city === 'string' ? pop.city : pop.city?.name || pop.city?.code || '';
    const stateStr = typeof pop.state === 'string' ? pop.state : pop.state?.name || pop.state?.code || '';
    const countryStr = typeof pop.country === 'string' ? pop.country : pop.country?.name || pop.country?.code || '';

    if (!cityStr && !stateStr && !countryStr && !pop.streetAddress && !pop.zip) return null;

    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5 text-blue-600" />
            Place of Performance
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {pop.streetAddress && renderField('Street Address', pop.streetAddress)}
          {cityStr && renderField('City', cityStr)}
          {stateStr && renderField('State', stateStr)}
          {pop.zip && renderField('ZIP', pop.zip)}
          {countryStr && renderField('Country', countryStr)}
        </CardContent>
      </Card>
    );
  };

  if (loading) {
    return (
      <main className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-8">
          <Skeleton className="h-12 w-64 mb-6" />
          <Skeleton className="h-96 w-full" />
        </div>
      </main>
    );
  }

  if (error && !opportunity) {
    return (
      <main className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-8">
          <Card className="border-red-200 bg-red-50">
            <CardContent className="pt-6">
              <div className="flex items-center gap-3 mb-4">
                <AlertCircle className="h-5 w-5 text-red-600" />
                <h2 className="text-lg font-semibold text-red-900">Error</h2>
              </div>
              <p className="text-red-700 mb-4">{error}</p>
              <Button onClick={() => window.location.href = '/opportunities'}>
                Back to Opportunities
              </Button>
            </CardContent>
          </Card>
        </div>
      </main>
    );
  }

  if (!opportunity) {
    return (
      <main className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-8">
          <Card>
            <CardContent className="pt-6">
              <p className="text-slate-600">Opportunity not found</p>
            </CardContent>
          </Card>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-8">
        <OpportunityHeader opportunity={opportunity} />

        <Tabs value={activeMainTab} onValueChange={(v) => setActiveMainTab(v as MainTab)} className="w-full">
          <TabsList className="grid w-full grid-cols-3 lg:grid-cols-6 mb-6">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="location">Location</TabsTrigger>
            <TabsTrigger value="contacts">Contacts</TabsTrigger>
            <TabsTrigger value="resources">Resources</TabsTrigger>
            <TabsTrigger value="description">Description</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Overview</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {renderField('Posted Date', formatDate(opportunity.postedDate), <Calendar className="h-4 w-4" />)}
                  {renderField('Response Deadline', formatDateTime(opportunity.responseDeadline), <Clock className="h-4 w-4" />)}
                  {renderField('Type', opportunity.type, <FileText className="h-4 w-4" />)}
                  {renderField('Base Type', opportunity.baseType, <FileText className="h-4 w-4" />)}
                  {renderField('Archive Type', opportunity.archiveType, <FileText className="h-4 w-4" />)}
                  {renderField('Archive Date', formatDate(opportunity.archiveDate), <Calendar className="h-4 w-4" />)}
                  {renderField('Organization Type', opportunity.organizationType, <Building2 className="h-4 w-4" />)}
                  {renderField('Active', opportunity.active ? 'Yes' : 'No')}
                  {renderField('Solicitation Number', opportunity.solicitationNumber, <FileText className="h-4 w-4" />)}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="details" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Details</CardTitle>
              </CardHeader>
              <CardContent className="space-y-6">
                {renderField('Full Parent Path Name', opportunity.fullParentPathName)}
                {renderField('Full Parent Path Code', opportunity.fullParentPathCode)}
                {renderField('Agency Path Name', opportunity.agencyPathName)}
                {renderField('Classification Code', opportunity.classificationCode)}
                {renderField('Type of Set Aside', opportunity.typeOfSetAside)}
                {renderField('Type of Set Aside Description', opportunity.typeOfSetAsideDescription)}
                {renderField('Award', opportunity.award ? JSON.stringify(opportunity.award) : null)}

                {opportunity.naics && opportunity.naics.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-3">NAICS Codes</h3>
                    <div className="space-y-2">
                      {opportunity.naics.map((naics, idx) => (
                        <Card key={idx} className="bg-blue-50 border-blue-200">
                          <CardContent className="pt-4">
                            <div className="flex items-center gap-2 mb-2">
                              <Badge variant="secondary">{naics.code}</Badge>
                            </div>
                            {naics.description && (
                              <p className="text-sm text-slate-700">{naics.description}</p>
                            )}
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  </div>
                )}

                {opportunity.naicsCode && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-2">NAICS Code</h3>
                    <p className="text-slate-900">{opportunity.naicsCode}</p>
                  </div>
                )}

                {opportunity.naicsCodes && opportunity.naicsCodes.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-2">NAICS Codes Array</h3>
                    <div className="flex flex-wrap gap-2">
                      {opportunity.naicsCodes.map((code, idx) => (
                        <Badge key={idx} variant="outline">{code}</Badge>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="location" className="space-y-4">
            {opportunity.officeAddress && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Building2 className="h-5 w-5 text-blue-600" />
                    Office Address
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {renderField('City', opportunity.officeAddress.city)}
                  {renderField('State', opportunity.officeAddress.state)}
                  {renderField('ZIP Code', opportunity.officeAddress.zipcode)}
                  {renderField('Country Code', opportunity.officeAddress.countryCode)}
                </CardContent>
              </Card>
            )}

            {renderPlaceOfPerformance()}
          </TabsContent>

          <TabsContent value="contacts" className="space-y-4">
            {opportunity.pointOfContact && opportunity.pointOfContact.length > 0 ? (
              <div className="space-y-4">
                {opportunity.pointOfContact.map((contact, idx) => (
                  <Card key={idx}>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <User className="h-5 w-5 text-blue-600" />
                        {contact.type ? contact.type.charAt(0).toUpperCase() + contact.type.slice(1) : 'Point of Contact'}
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      {renderField('Full Name', contact.fullName, <User className="h-4 w-4" />)}
                      {renderField('Title', contact.title, <User className="h-4 w-4" />)}
                      {renderField('Email', contact.email, <Mail className="h-4 w-4" />)}
                      {renderField('Phone', contact.phone, <Phone className="h-4 w-4" />)}
                      {renderField('Fax', contact.fax, <Printer className="h-4 w-4" />)}
                      {contact.additionalInfoLink && (
                        <div className="pt-4 border-t border-blue-200">
                          <Button variant="outline" asChild>
                            <a
                              href={contact.additionalInfoLink}
                              target="_blank"
                              rel="noopener noreferrer"
                            >
                              Additional Info Link
                              <ExternalLink className="h-4 w-4 ml-2" />
                            </a>
                          </Button>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <Card>
                <CardContent className="pt-6">
                  <p className="text-slate-500 text-center">No point of contact information available.</p>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="resources" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Resources</CardTitle>
              </CardHeader>
              <CardContent className="space-y-6">
                {opportunity.resourceLinks && opportunity.resourceLinks.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-3">Downloadable Resources</h3>
                    <div className="space-y-2">
                      {opportunity.resourceLinks.map((link, idx) => (
                        <Button
                          key={idx}
                          variant="outline"
                          className="w-full justify-start"
                          asChild
                        >
                          <a
                            href={link}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            <Download className="h-4 w-4 mr-2" />
                            Download Resource {idx + 1}
                          </a>
                        </Button>
                      ))}
                    </div>
                  </div>
                )}

                {opportunity.uiLink && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-2">UI Link</h3>
                    <Button variant="outline" asChild>
                      <a
                        href={opportunity.uiLink}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        View on SAM.gov
                        <ExternalLink className="h-4 w-4 ml-2" />
                      </a>
                    </Button>
                  </div>
                )}

                {opportunity.additionalInfoLink && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-2">Additional Info Link</h3>
                    <Button variant="outline" asChild>
                      <a
                        href={opportunity.additionalInfoLink}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Additional Information
                        <ExternalLink className="h-4 w-4 ml-2" />
                      </a>
                    </Button>
                  </div>
                )}

                {opportunity.links && opportunity.links.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 mb-3">API Links</h3>
                    <div className="space-y-2">
                      {opportunity.links.map((link, idx) => (
                        <Button
                          key={idx}
                          variant="ghost"
                          className="w-full justify-start text-left"
                          asChild
                        >
                          <a
                            href={link.href}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            {link.rel}: {link.href}
                            <ExternalLink className="h-4 w-4 ml-2" />
                          </a>
                        </Button>
                      ))}
                    </div>
                  </div>
                )}

                {(!opportunity.resourceLinks || opportunity.resourceLinks.length === 0) &&
                  !opportunity.uiLink &&
                  !opportunity.additionalInfoLink &&
                  (!opportunity.links || opportunity.links.length === 0) && (
                    <p className="text-slate-500 text-center">No resource links available.</p>
                  )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="description" className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>Description</CardTitle>
                  <div className="flex items-center gap-3">
                    {description && (
                      <Badge variant={getStatusBadgeVariant(description.status)}>
                        {getStatusLabel(description.status)}
                      </Badge>
                    )}
                    <Button
                      onClick={handleRefresh}
                      disabled={refreshing || descriptionLoading}
                      variant="outline"
                      size="sm"
                    >
                      <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                      {refreshing ? 'Refreshing...' : 'Refresh'}
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {descriptionLoading && !description && (
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-4 w-3/4" />
                  </div>
                )}

                {description && (
                  <>
                    {description.status === 'fetched' && (
                      <>
                        <Tabs value={activeDescriptionTab} onValueChange={(v) => setActiveDescriptionTab(v as DescriptionTab)}>
                          <TabsList className="mb-4">
                            <TabsTrigger value="normalized">Normalized</TabsTrigger>
                            <TabsTrigger value="raw">Raw Post-Parse</TabsTrigger>
                          </TabsList>

                          <TabsContent value="normalized">
                            <ScrollArea className="h-[600px] rounded-md border border-blue-200 p-4">
                              <div
                                className="prose prose-sm max-w-none"
                                dangerouslySetInnerHTML={{
                                  __html: description.normalizedText || 'No normalized text available',
                                }}
                              />
                            </ScrollArea>
                          </TabsContent>

                          <TabsContent value="raw">
                            <ScrollArea className="h-[600px] rounded-md border border-blue-200 p-4">
                              <pre className="text-xs font-mono whitespace-pre-wrap break-words">
                                {(() => {
                                  if (description.rawJsonResponse) {
                                    try {
                                      const parsed = JSON.parse(description.rawJsonResponse);
                                      return JSON.stringify(parsed, null, 2);
                                    } catch {
                                      return description.rawJsonResponse;
                                    }
                                  }
                                  return description.rawPostParseText || 'No raw post-parse text available';
                                })()}
                              </pre>
                            </ScrollArea>
                          </TabsContent>
                        </Tabs>
                      </>
                    )}

                    {description.status === 'none' && (
                      <p className="text-slate-500 text-center py-8">No description available for this opportunity.</p>
                    )}

                    {description.status === 'not_found' && (
                      <div className="text-center py-8">
                        <AlertCircle className="h-12 w-12 text-slate-400 mx-auto mb-4" />
                        <p className="text-slate-600">
                          Description not found. The description may not be available from the source.
                        </p>
                      </div>
                    )}

                    {description.status === 'error' && (
                      <div className="text-center py-8">
                        <AlertCircle className="h-12 w-12 text-red-400 mx-auto mb-4" />
                        <p className="text-red-600 mb-4">
                          Error loading description. Please try refreshing.
                        </p>
                        <Button onClick={handleRefresh} variant="outline">
                          Try Again
                        </Button>
                      </div>
                    )}

                    {description.status === 'available_unfetched' && (
                      <div className="text-center py-8">
                        <p className="text-slate-600 mb-4">
                          Description is available but has not been fetched yet. Click Refresh to fetch it.
                        </p>
                        <Button onClick={handleRefresh}>
                          Fetch Description
                        </Button>
                      </div>
                    )}
                  </>
                )}

                {!description && !descriptionLoading && (
                  <p className="text-slate-500 text-center py-8">No description data available.</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </main>
  );
}
