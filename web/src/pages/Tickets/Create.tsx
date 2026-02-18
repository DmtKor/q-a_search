import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Ticket, TicketCreate } from '@/api/types'

export function TicketCreate() {
  const navigate = useNavigate()
  const { request } = useApi()
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [confidence, setConfidence] = useState<number | ''>('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!query.trim()) return
    setSubmitting(true)
    const body: TicketCreate = {
      query: query.trim(),
      ...(category.trim() && { category: category.trim() }),
      ...(confidence !== '' && { confidence: Number(confidence) }),
    }
    const { data, error } = await request<Ticket>('tickets', { method: 'POST', body })
    setSubmitting(false)
    if (!error && data?.id) navigate(`/tickets/${data.id}`)
  }

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/tickets">Тикеты</Link>
        {' / Создать'}
      </nav>
      <h1 className="page-title">Создать тикет вручную</h1>
      <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 480 }}>
        <div className="form-group">
          <label>Запрос *</label>
          <textarea value={query} onChange={(e) => setQuery(e.target.value)} rows={3} required />
        </div>
        <div className="form-group">
          <label>Категория</label>
          <input value={category} onChange={(e) => setCategory(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Confidence (0–1)</label>
          <input
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={confidence === '' ? '' : confidence}
            onChange={(e) => setConfidence(e.target.value === '' ? '' : Number(e.target.value))}
          />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Создание…' : 'Создать'}
        </button>
      </form>
    </>
  )
}
