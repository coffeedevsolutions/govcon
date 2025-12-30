/**
 * API Client for making requests to the backend
 */

export interface OpportunitiesResponse {
  items: Opportunity[];
  totalRecords: number;
  limit: number;
  offset: number;
  hasMore: boolean;
}

export interface Opportunity {
  noticeId: string;
  title: string;
  organizationType: string;
  postedDate: string;
  type: string;
  baseType: string;
  archiveType?: string;
  archiveDate?: string;
  typeOfSetAside?: string;
  typeOfSetAsideDesc?: string;
  responseDeadline?: string;
  naics?: Array<{
    code: string;
    description: string;
  }>;
  classificationCode?: string;
  active: boolean;
  pointOfContact?: Array<{
    fax?: string;
    type?: string;
    email?: string;
    phone?: string;
    title?: string;
    fullName?: string;
    additionalInfoLink?: string;
  }>;
  placeOfPerformance?: {
    streetAddress?: string;
    city?: string;
    state?: string;
    zip?: string;
    country?: string;
  };
  description?: string;
  department?: string;
  subTier?: string;
  office?: string;
  solicitationNumber?: string;
  agencyPathName?: string;
  links?: Array<{
    rel: string;
    href: string;
    type: string;
  }>;
  descriptionStatus?: string; // none | ready | not_found | error | available_unfetched
}

export interface OpportunitiesParams {
  postedFrom?: string;
  postedTo?: string;
  limit?: number;
  offset?: number;
  ptype?: string;
}

export interface SearchOpportunitiesParams {
  q?: string;
  naics?: string;
  setAside?: string;
  state?: string;
  agency?: string;
  postedFrom?: string;
  postedTo?: string;
  dueFrom?: string;
  dueTo?: string;
  sort?: 'posted_desc' | 'due_asc' | 'relevance';
  limit?: number;
  cursor?: string;
}

export interface SearchOpportunitiesResponse {
  items: Opportunity[];
  nextCursor: string | null;
  debug?: {
    sort: string;
    appliedFilters: Record<string, string>;
  };
}

export interface OpportunityDescription {
  noticeId: string;
  status: 'fetched' | 'not_found' | 'none' | 'error' | 'available_unfetched';
  sourceType: 'url' | 'inline' | 'none';
  sourceUrl?: string;
  rawText?: string;
  rawPostParseText?: string;
  normalizedText?: string;
  rawJsonResponse?: string;
  normalizationVersion?: number;
  fetchedAt?: string;
}

class APIClient {
  private baseURL: string;

  constructor() {
    // Use relative URL for same-origin requests (Next.js will proxy via rewrite)
    this.baseURL = typeof window !== 'undefined' ? '/api' : 'http://localhost:4000';
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    
    console.log('API Request:', { url, baseURL: this.baseURL, endpoint });
    
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    console.log('API Response status:', response.status, response.statusText);

    if (!response.ok) {
      // Try to get error message from response
      let errorMessage = `HTTP error! status: ${response.status}`;
      let errorData: any = {};
      
      try {
        const text = await response.text();
        console.error('API Error Response Text:', text);
        
        if (text) {
          try {
            errorData = JSON.parse(text);
            errorMessage = errorData.error || errorData.message || errorMessage;
          } catch {
            // If not JSON, use the text as error message
            errorMessage = text || errorMessage;
          }
        }
      } catch (err) {
        console.error('Failed to read error response:', err);
      }
      
      console.error('API Error Details:', {
        status: response.status,
        statusText: response.statusText,
        errorData,
        errorMessage,
      });
      
      throw new Error(errorMessage);
    }

    const data = await response.json();
    console.log('API Response data:', data);
    return data;
  }

  async getOpportunities(params: OpportunitiesParams = {}): Promise<OpportunitiesResponse> {
    const queryParams = new URLSearchParams();
    
    if (params.postedFrom) queryParams.append('postedFrom', params.postedFrom);
    if (params.postedTo) queryParams.append('postedTo', params.postedTo);
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.ptype) queryParams.append('ptype', params.ptype);

    const queryString = queryParams.toString();
    const endpoint = `/opportunities${queryString ? `?${queryString}` : ''}`;

    return this.request<OpportunitiesResponse>(endpoint);
  }

  async searchOpportunities(params: SearchOpportunitiesParams = {}): Promise<SearchOpportunitiesResponse> {
    const queryParams = new URLSearchParams();
    
    if (params.q) queryParams.append('q', params.q);
    if (params.naics) queryParams.append('naics', params.naics);
    if (params.setAside) queryParams.append('setAside', params.setAside);
    if (params.state) queryParams.append('state', params.state);
    if (params.agency) queryParams.append('agency', params.agency);
    if (params.postedFrom) queryParams.append('postedFrom', params.postedFrom);
    if (params.postedTo) queryParams.append('postedTo', params.postedTo);
    if (params.dueFrom) queryParams.append('dueFrom', params.dueFrom);
    if (params.dueTo) queryParams.append('dueTo', params.dueTo);
    if (params.sort) queryParams.append('sort', params.sort);
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.cursor) queryParams.append('cursor', params.cursor);

    const queryString = queryParams.toString();
    const endpoint = `/opportunities/search${queryString ? `?${queryString}` : ''}`;

    return this.request<SearchOpportunitiesResponse>(endpoint);
  }

  async getOpportunity(noticeId: string): Promise<Opportunity> {
    const endpoint = `/opportunities/${noticeId}`;
    return this.request<Opportunity>(endpoint);
  }

  async getOpportunityDescription(noticeId: string, refresh?: boolean): Promise<OpportunityDescription> {
    const queryParams = new URLSearchParams();
    if (refresh) {
      queryParams.append('refresh', 'true');
    }

    const queryString = queryParams.toString();
    const endpoint = `/opportunities/${noticeId}/description${queryString ? `?${queryString}` : ''}`;

    return this.request<OpportunityDescription>(endpoint);
  }
}

export const apiClient = new APIClient();

