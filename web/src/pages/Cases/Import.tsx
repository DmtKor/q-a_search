import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { ImportResult } from '@/api/types'

type Mode = 'merge' | 'draft' | 'replace'

export function CaseImport() {
  const { request } = useApi()
  const [mode, setMode] = useState<Mode>('merge')
  const [file, setFile] = useState<File | null>(null)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!file) return
    setSubmitting(true)
    setResult(null)
    const text = await file.text()
    let body: unknown
    try {
      body = JSON.parse(text)
    } catch {
      setResult({ imported: 0, updated: 0, errors: ['Invalid JSON'] })
      setSubmitting(false)
      return
    }
    const { data, error } = await request<ImportResult>(`cases/import?mode=${mode}`, { method: 'POST', body })
    setSubmitting(false)
    if (!error && data) setResult(data)
  }

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/cases">База знаний</Link>
        {' / Импорт'}
      </nav>
      <h1 className="page-title">Импорт кейсов</h1>
      <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 400 }}>
        <div className="form-group">
          <label>Режим</label>
          <select value={mode} onChange={(e) => setMode(e.target.value as Mode)}>
            <option value="merge">merge</option>
            <option value="draft">draft</option>
            <option value="replace">replace</option>
          </select>
        </div>
        <div className="form-group">
          <label>Файл JSON</label>
          <input
            type="file"
            accept=".json,application/json"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </div>
        <button type="submit" className="primary" disabled={submitting || !file}>
          {submitting ? 'Импорт…' : 'Импорт'}
        </button>
      </form>
      {result && (
        <div className="card">
          <p>Импортировано: {result.imported}, обновлено: {result.updated}</p>
          {result.errors?.length ? (
            <ul>
              {result.errors.map((err, i) => (
                <li key={i} style={{ color: 'var(--error)' }}>{err}</li>
              ))}
            </ul>
          ) : null}
        </div>
      )}
    </>
  )
}
