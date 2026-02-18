import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { AccessType } from '@/api/types'
import {
  apiRequest,
  getStoredBaseURL,
  getStoredToken,
  setStoredBaseURL,
  setStoredToken,
} from '@/api/client'

type SetToken = (token: string) => void
type SetBaseURL = (url: string) => void
type SaveToStorage = () => void
type CheckAccess = () => Promise<void>

const defaultBase = getStoredBaseURL()
const defaultToken = getStoredToken() ?? ''

const SettingsContext = createContext<{
  token: string
  baseURL: string
  accessType: AccessType
  checked: boolean
  checking: boolean
  checkError: string | null
  setToken: SetToken
  setBaseURL: SetBaseURL
  saveToStorage: SaveToStorage
  checkAccess: CheckAccess
  getTokenForRequest: () => string | null
  getBaseURLForRequest: () => string
} | null>(null)

const API_V1 = '/api/v1'

function normalizeBaseURL(raw: string): string {
  const base = raw.trim().replace(/\/+$/, '')
  if (!base) return ''
  return base.endsWith(API_V1) ? base : `${base}${API_V1}`
}

function getBaseURLForRequest(baseURLInput: string): string {
  const { hostname, port } = window.location
  let base = baseURLInput.trim()
  if (base === '') {
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      base = port === '8080' ? `${window.location.origin}` : 'http://localhost:8080'
    } else {
      return `${window.location.origin}${API_V1}`
    }
  }
  return normalizeBaseURL(base)
}

export function SettingsProvider({ children }: { children: React.ReactNode }) {
  const [token, setTokenState] = useState(defaultToken)
  const [baseURL, setBaseURLState] = useState(defaultBase)
  const [accessType, setAccessType] = useState<AccessType>(null)
  const [checked, setChecked] = useState(false)
  const [checking, setChecking] = useState(false)
  const [checkError, setCheckError] = useState<string | null>(null)

  const getBaseURLResolved = useCallback(() => getBaseURLForRequest(baseURL), [baseURL])

  const checkAccess = useCallback(async () => {
    setCheckError(null)
    if (!token.trim()) {
      setAccessType(null)
      setChecked(true)
      return
    }
    setChecking(true)
    const base = getBaseURLForRequest(baseURL)
    const result = await apiRequest<unknown>('cases?limit=1', {
      method: 'GET',
      token,
      baseURLOverride: base,
    })
    setChecking(false)
    setChecked(true)
    if ('error' in result) {
      if (result.error.status === 403) {
        setAccessType('app')
      } else {
        setAccessType(null)
        const status = result.error.status
        const msg = result.error.message || ''
        if (status === 0 || status === 404) {
          setCheckError(
            'Не удалось подключиться к API. Укажите базовый URL API (например http://localhost:8080), если сервер на другом порту.'
          )
        } else {
          setCheckError(`Ошибка ${status}: ${msg}`)
        }
      }
    } else {
      setAccessType('staff')
    }
  }, [token, baseURL])

  useEffect(() => {
    checkAccess()
  }, []) // run once on mount: use stored token/baseURL

  const setToken: SetToken = useCallback((t) => setTokenState(t), [])
  const setBaseURL: SetBaseURL = useCallback((u) => setBaseURLState(u), [])

  const saveToStorage: SaveToStorage = useCallback(() => {
    setStoredToken(token)
    setStoredBaseURL(baseURL)
    checkAccess()
  }, [token, baseURL, checkAccess])

  const getTokenForRequest = useCallback(() => (token.trim() ? token : null), [token])

  const value = useMemo(
    () => ({
      token,
      baseURL,
      accessType,
      checked,
      checking,
      checkError,
      setToken,
      setBaseURL,
      saveToStorage,
      checkAccess,
      getTokenForRequest,
      getBaseURLForRequest: getBaseURLResolved,
    }),
    [
      token,
      baseURL,
      accessType,
      checked,
      checking,
      checkError,
      setToken,
      setBaseURL,
      saveToStorage,
      checkAccess,
      getTokenForRequest,
      getBaseURLResolved,
    ]
  )

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

export function useSettings() {
  const ctx = useContext(SettingsContext)
  if (!ctx) throw new Error('useSettings must be used within SettingsProvider')
  return ctx
}
