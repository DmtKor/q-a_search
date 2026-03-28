// OpenAPI-aligned types

export type AccessType = 'app' | 'staff' | null

export interface ErrorEnvelope {
  error: {
    code: string
    message: string
    details: Record<string, unknown> | null
  }
}

export interface CategoriesResponse {
  categories: string[]
}

export interface SearchRequest {
  query: string
  category?: string
  top_k?: number
  user_context?: Record<string, unknown>
  no_ticket_on_low_confidence?: boolean
}

export interface Chunk {
  case_id: string
  title: string
  text: string
  confidence: number
}

export interface TicketRef {
  id: string
  url: string
}

export interface SearchResponse {
  chunks?: Chunk[]
  ticket?: TicketRef
}

export type CaseStatus = 'draft' | 'pending_review' | 'approved' | 'archived'

export interface Case {
  id: string
  category: string
  title: string
  questions: string[]
  keywords: string[]
  response_template: string
  status: CaseStatus
  created_by: string | null
  created_at: string
  updated_by: string | null
  updated_at: string
  approved_by: string | null
  approved_at: string | null
  notes: string | null
}

export interface CaseCreate {
  category: string
  title: string
  questions?: string[]
  keywords?: string[]
  response_template: string
}

export interface CaseUpdate {
  category?: string
  title?: string
  questions?: string[]
  keywords?: string[]
  response_template?: string
  notes?: string
}

export interface StatusChangeRequest {
  status: CaseStatus
  comment?: string
}

export interface ImportResult {
  imported: number
  updated: number
  errors: string[]
}

export type TicketStatus = 'open' | 'in_progress' | 'resolved' | 'closed'

export interface Ticket {
  id: string
  query: string
  category: string | null
  confidence: number | null
  status: TicketStatus
  assigned_to: string | null
  created_at: string
  updated_at: string
  resolved_at: string | null
  resolution_notes: string | null
  converted_to_case_id: string | null
}

export interface TicketCreate {
  query: string
  category?: string
  confidence?: number
}

export interface TicketUpdate {
  status?: TicketStatus
  assigned_to?: string
  resolution_notes?: string
}

export interface ConvertToCaseRequest {
  title?: string
  category?: string
  response_template?: string
}

export interface ConvertToCaseResponse {
  case_id: string
  url: string
}

export interface AppSettings {
  search?: {
    default_top_k?: number
    confidence_threshold?: number
  }
  [key: string]: unknown
}

export interface App {
  id: string
  name: string
  settings?: AppSettings
  created_at: string
  updated_at: string
}

export interface AppCreate {
  name: string
  settings?: AppSettings
}

export interface AppUpdate {
  name?: string
  settings?: AppSettings
}

export interface ReadableSegment {
  type: 'literal' | 'readable' | 'raw'
  text?: string
  code?: string
  description?: string
}

export interface TemplateReadableResponse {
  segments: ReadableSegment[]
}
