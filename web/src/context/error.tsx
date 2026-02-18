import React, { createContext, useCallback, useContext, useState } from 'react'
import type { ApiError } from '@/api/client'

interface ErrorContextValue {
  error: ApiError | null
  setError: (err: ApiError | null) => void
  clearError: () => void
}

const ErrorContext = createContext<ErrorContextValue | null>(null)

export function ErrorProvider({ children }: { children: React.ReactNode }) {
  const [error, setError] = useState<ApiError | null>(null)
  const clearError = useCallback(() => setError(null), [])
  return (
    <ErrorContext.Provider value={{ error, setError, clearError }}>
      {children}
    </ErrorContext.Provider>
  )
}

export function useError() {
  const ctx = useContext(ErrorContext)
  if (!ctx) throw new Error('useError must be used within ErrorProvider')
  return ctx
}
