import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useApi } from '@/api/request'
import { useError } from '@/context/error'
import type { App, AppSettings } from '@/api/types'

export function AppDetail() {
  const { id } = useParams<{ id: string }>()
  const { request, getTokenForRequest, getBaseURLForRequest } = useApi()
  const { setError } = useError()
  const [app, setApp] = useState<App | null>(null)
  const [settings, setSettings] = useState<AppSettings | null>(null)
  const [defaultTopK, setDefaultTopK] = useState<number | ''>('')
  const [confidenceThreshold, setConfidenceThreshold] = useState<number | ''>('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!id) return
    request<App>(`apps/${id}`).then(({ data }) => setApp(data ?? null))
  }, [id, request])

  useEffect(() => {
    if (!id) return
    request<AppSettings>(`apps/${id}/settings`).then(({ data }) => {
      if (data) {
        setSettings(data)
        setDefaultTopK(data.search?.default_top_k ?? '')
        setConfidenceThreshold(data.search?.confidence_threshold ?? '')
      }
    })
  }, [id, request])

  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    setSubmitting(true)
    const next: AppSettings = {
      ...settings,
      search: {
        ...(settings?.search ?? {}),
        ...(defaultTopK !== '' && { default_top_k: Number(defaultTopK) }),
        ...(confidenceThreshold !== '' && { confidence_threshold: Number(confidenceThreshold) }),
      },
    }
    const { error } = await request<AppSettings>(`apps/${id}/settings`, { method: 'PUT', body: next })
    setSubmitting(false)
    if (!error) setSettings(next)
  }

  const handleExportSettings = async () => {
    if (!id || !getTokenForRequest()) return
    const base = getBaseURLForRequest()
    const url = `${base.replace(/\/$/, '')}/apps/${id}/settings/export`
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${getTokenForRequest()}` },
    })
    if (!res.ok) {
      const ct = res.headers.get('content-type')
      if (ct?.includes('application/json')) {
        const err = await res.json().catch(() => ({}))
        setError({
          status: res.status,
          code: (err as { error?: { code?: string } })?.error?.code ?? 'unknown',
          message: (err as { error?: { message?: string } })?.error?.message ?? res.statusText,
          details: (err as { error?: { details?: Record<string, unknown> | null } })?.error?.details ?? null,
        })
      }
      return
    }
    const blob = await res.blob()
    const name = `app-settings-${id}-${new Date().toISOString().slice(0, 10)}.json`
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = name
    a.click()
    URL.revokeObjectURL(a.href)
  }

  const handleImportSettings = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !id) return
    const text = await file.text()
    let body: AppSettings
    try {
      body = JSON.parse(text) as AppSettings
    } catch {
      return
    }
    await request<AppSettings>(`apps/${id}/settings/import`, { method: 'POST', body })
    request<AppSettings>(`apps/${id}/settings`).then(({ data }) => {
      if (data) {
        setSettings(data)
        setDefaultTopK(data.search?.default_top_k ?? '')
        setConfidenceThreshold(data.search?.confidence_threshold ?? '')
      }
    })
    e.target.value = ''
  }

  if (!app) return <p className="empty">Загрузка…</p>

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/apps">Приложения</Link>
        {' / '}
        <span>{app.name}</span>
      </nav>
      <h1 className="page-title">{app.name}</h1>
      <div className="card">
        <p><strong>ID:</strong> {app.id}</p>
        <p><strong>Создан:</strong> {app.created_at ? new Date(app.created_at).toLocaleString() : '—'}</p>
        <p><strong>Обновлён:</strong> {app.updated_at ? new Date(app.updated_at).toLocaleString() : '—'}</p>
      </div>

      <h2 style={{ fontSize: '1rem', marginBottom: '0.5rem' }}>Настройки</h2>
      <form onSubmit={handleSaveSettings} className="card" style={{ maxWidth: 400 }}>
        <div className="form-group">
          <label>search.default_top_k</label>
          <input
            type="number"
            min={1}
            max={50}
            value={defaultTopK === '' ? '' : defaultTopK}
            onChange={(e) => setDefaultTopK(e.target.value === '' ? '' : Number(e.target.value))}
          />
        </div>
        <div className="form-group">
          <label>search.confidence_threshold</label>
          <input
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={confidenceThreshold === '' ? '' : confidenceThreshold}
            onChange={(e) => setConfidenceThreshold(e.target.value === '' ? '' : Number(e.target.value))}
          />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Сохранение…' : 'Сохранить настройки'}
        </button>
      </form>
      {settings && (
        <div className="card">
          <strong>Остальные ключи (read-only)</strong>
          <pre style={{ marginTop: '0.5rem', fontSize: '0.85rem', overflow: 'auto' }}>
            {JSON.stringify(settings, null, 2)}
          </pre>
        </div>
      )}
      <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}>
        <button type="button" onClick={handleExportSettings}>
          Экспорт настроек
        </button>
        <label>
          <span style={{ marginRight: '0.5rem' }}>Импорт настроек</span>
          <input type="file" accept=".json,application/json" onChange={handleImportSettings} style={{ display: 'inline-block', width: 'auto' }} />
        </label>
      </div>
    </>
  )
}
