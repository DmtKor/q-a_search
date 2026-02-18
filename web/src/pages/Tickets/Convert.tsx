import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { ConvertToCaseRequest, ConvertToCaseResponse } from '@/api/types'

export function TicketConvert() {
  const { id } = useParams<{ id: string }>()
  const { request } = useApi()
  const [title, setTitle] = useState('')
  const [category, setCategory] = useState('')
  const [responseTemplate, setResponseTemplate] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [result, setResult] = useState<ConvertToCaseResponse | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    setSubmitting(true)
    setResult(null)
    const body: ConvertToCaseRequest = {}
    if (title.trim()) body.title = title.trim()
    if (category.trim()) body.category = category.trim()
    if (responseTemplate.trim()) body.response_template = responseTemplate.trim()
    const { data, error } = await request<ConvertToCaseResponse>(`tickets/${id}/convert-to-case`, { method: 'POST', body })
    setSubmitting(false)
    if (!error && data) setResult(data)
  }

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/tickets">Тикеты</Link>
        {' / '}
        <Link to={`/tickets/${id}`}>{id}</Link>
        {' / Конвертировать в кейс'}
      </nav>
      <h1 className="page-title">Конвертировать тикет в кейс</h1>
      {result ? (
        <div className="card">
          <p>Создан кейс: <strong>{result.case_id}</strong></p>
          <p>URL: {result.url}</p>
          <Link to={`/cases/${result.case_id}`}>
            <button type="button" className="primary">Перейти к кейсу</button>
          </Link>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 480 }}>
          <div className="form-group">
            <label>Заголовок кейса</label>
            <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Опционально" />
          </div>
          <div className="form-group">
            <label>Категория</label>
            <input value={category} onChange={(e) => setCategory(e.target.value)} placeholder="Опционально" />
          </div>
          <div className="form-group">
            <label>Шаблон ответа</label>
            <textarea value={responseTemplate} onChange={(e) => setResponseTemplate(e.target.value)} rows={3} placeholder="Опционально" />
          </div>
          <button type="submit" className="primary" disabled={submitting}>
            {submitting ? 'Конвертация…' : 'Конвертировать'}
          </button>
        </form>
      )}
    </>
  )
}
