import { useCallback } from 'react'
import { apiRequest } from './client'
import { useError } from '@/context/error'
import { useSettings } from '@/store/settings'
import { clientLogError } from '@/lib/clientLog'

export function useApi() {
  const { setError } = useError()
  const { getTokenForRequest, getBaseURLForRequest } = useSettings()

  const request = useCallback(
    async <T,>(path: string, options: { method?: string; body?: unknown } = {}) => {
      const token = getTokenForRequest()
      const baseURL = getBaseURLForRequest()
      const result = await apiRequest<T>(path, {
        ...options,
        token,
        baseURLOverride: baseURL || undefined,
      })
      if ('error' in result) {
        setError(result.error)
        clientLogError('api_error', {
          path,
          method: options.method ?? 'GET',
          status: result.error.status,
          code: result.error.code,
        })
        return { data: undefined as T, error: result.error }
      }
      return { data: result.data, error: undefined }
    },
    [setError, getTokenForRequest, getBaseURLForRequest]
  )

  return { request, getTokenForRequest, getBaseURLForRequest }
}
