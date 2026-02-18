import type { ErrorEnvelope } from './types'

const STORAGE_TOKEN = 'demo_token'
const STORAGE_BASE_URL = 'demo_base_url'

const API_V1 = '/api/v1'

function normalizeBaseURL(raw: string): string {
  const base = raw.trim().replace(/\/+$/, '')
  if (!base) return ''
  return base.endsWith(API_V1) ? base : `${base}${API_V1}`
}

function getBaseURL(): string {
  const stored = localStorage.getItem(STORAGE_BASE_URL)
  if (stored != null && stored.trim() !== '') return normalizeBaseURL(stored)
  return `${window.location.origin}${API_V1}`
}

function getToken(): string | null {
  return localStorage.getItem(STORAGE_TOKEN)
}

export function getStoredToken(): string | null {
  return getToken()
}

export function getStoredBaseURL(): string {
  return localStorage.getItem(STORAGE_BASE_URL) ?? ''
}

export function setStoredToken(token: string): void {
  localStorage.setItem(STORAGE_TOKEN, token)
}

export function setStoredBaseURL(url: string): void {
  if (url.trim() === '') {
    localStorage.removeItem(STORAGE_BASE_URL)
  } else {
    localStorage.setItem(STORAGE_BASE_URL, url.trim())
  }
}

export interface ApiError {
  status: number
  code: string
  message: string
  details: Record<string, unknown> | null
}

async function parseErrorResponse(res: Response): Promise<ApiError> {
  let code = 'unknown'
  let message = res.statusText
  let details: Record<string, unknown> | null = null
  const ct = res.headers.get('content-type')
  if (ct?.includes('application/json')) {
    try {
      const body = (await res.json()) as ErrorEnvelope
      if (body?.error) {
        code = body.error.code ?? code
        message = body.error.message ?? message
        details = body.error.details ?? null
      }
    } catch {
      /* use defaults */
    }
  }
  return { status: res.status, code, message, details }
}

export interface RequestOptions {
  method?: string
  body?: unknown
  token: string | null
  baseURLOverride?: string
}

export async function apiRequest<T>(
  path: string,
  options: RequestOptions
): Promise<{ data: T } | { error: ApiError }> {
  const token = options.token
  if (!token?.trim()) {
    return {
      error: {
        status: 0,
        code: 'unauthorized',
        message: 'Введите токен в настройках',
        details: null,
      },
    }
  }

  let baseURL = options.baseURLOverride ?? getBaseURL()
  if (baseURL && !path.startsWith('http')) {
    baseURL = baseURL.trim().endsWith(API_V1) ? baseURL.trim() : normalizeBaseURL(baseURL)
  }
  const url = path.startsWith('http') ? path : `${baseURL.replace(/\/+$/, '')}/${path.replace(/^\//, '')}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  }

  const init: RequestInit = {
    method: options.method ?? 'GET',
    headers,
  }
  if (options.body != null && options.method !== 'GET') {
    init.body = JSON.stringify(options.body)
  }

  const res = await fetch(url, init)
  const ct = res.headers.get('content-type')

  if (!res.ok) {
    const err = await parseErrorResponse(res)
    return { error: err }
  }

  if (res.status === 204) {
    return { data: undefined as T }
  }

  const text = await res.text()
  if (!text || !text.trim()) {
    return { data: undefined as T }
  }
  if (ct?.includes('application/json')) {
    try {
      const data = JSON.parse(text) as T
      return { data }
    } catch {
      return {
        error: {
          status: res.status,
          code: 'invalid_response',
          message: 'Invalid JSON in response',
          details: null,
        },
      }
    }
  }

  return { data: undefined as T }
}
