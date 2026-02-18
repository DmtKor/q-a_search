import { useEffect, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Case, CaseStatus, StatusChangeRequest } from '@/api/types'

const STATUS_OPTIONS: CaseStatus[] = ['draft', 'pending_review', 'approved', 'archived']

export function CaseStatus() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { request } = useApi()
  const [case_, setCase] = useState<Case | null>(null)
  const [status, setStatus] = useState<CaseStatus>('pending_review')
  const [comment, setComment] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!id) return
    request<Case>(`cases/${id}`).then(({ data }) => {
      if (data) setCase(data)
    })
  }, [id])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    setSubmitting(true)
    const body: StatusChangeRequest = { status, ...(comment.trim() && { comment: comment.trim() }) }
    const { error } = await request<Case>(`cases/${id}/status`, { method: 'POST', body })
    setSubmitting(false)
    if (!error) navigate(`/cases/${id}`)
  }

  if (!case_) return <p className="empty">Загрузка…</p>

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/cases">База знаний</Link>
        {' / '}
        <Link to={`/cases/${id}`}>{case_.title}</Link>
        {' / Сменить статус'}
      </nav>
      <h1 className="page-title">Сменить статус</h1>
      <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 400 }}>
        <div className="form-group">
          <label>Статус</label>
          <select value={status} onChange={(e) => setStatus(e.target.value as CaseStatus)}>
            {STATUS_OPTIONS.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>
        <div className="form-group">
          <label>Комментарий</label>
          <textarea value={comment} onChange={(e) => setComment(e.target.value)} rows={2} />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Сохранение…' : 'Сохранить'}
        </button>
      </form>
    </>
  )
}
