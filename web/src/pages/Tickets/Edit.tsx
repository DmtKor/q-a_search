import { useEffect, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Ticket, TicketUpdate, TicketStatus } from '@/api/types'

const STATUSES: TicketStatus[] = ['open', 'in_progress', 'resolved', 'closed']

export function TicketEdit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { request } = useApi()
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [status, setStatus] = useState<TicketStatus>('open')
  const [assignedTo, setAssignedTo] = useState('')
  const [resolutionNotes, setResolutionNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!id) return
    request<Ticket>(`tickets/${id}`).then(({ data }) => {
      if (data) {
        setTicket(data)
        setStatus(data.status)
        setAssignedTo(data.assigned_to ?? '')
        setResolutionNotes(data.resolution_notes ?? '')
      }
    })
  }, [id, request])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    setSubmitting(true)
    const body: TicketUpdate = {
      status,
      assigned_to: assignedTo.trim() || undefined,
      resolution_notes: resolutionNotes.trim() || undefined,
    }
    const { error } = await request<Ticket>(`tickets/${id}`, { method: 'PUT', body })
    setSubmitting(false)
    if (!error) navigate(`/tickets/${id}`)
  }

  if (!ticket) return <p className="empty">Загрузка…</p>

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/tickets">Тикеты</Link>
        {' / '}
        <Link to={`/tickets/${id}`}>{ticket.id}</Link>
        {' / Редактировать'}
      </nav>
      <h1 className="page-title">Редактировать тикет</h1>
      <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 480 }}>
        <div className="form-group">
          <label>Статус</label>
          <select value={status} onChange={(e) => setStatus(e.target.value as TicketStatus)}>
            {STATUSES.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>
        <div className="form-group">
          <label>Назначен (assigned_to)</label>
          <input value={assignedTo} onChange={(e) => setAssignedTo(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Заметки по решению</label>
          <textarea value={resolutionNotes} onChange={(e) => setResolutionNotes(e.target.value)} rows={3} />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Сохранение…' : 'Сохранить'}
        </button>
      </form>
    </>
  )
}
