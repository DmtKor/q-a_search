import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Ticket } from '@/api/types'

export function TicketDetail() {
  const { id } = useParams<{ id: string }>()
  const { request } = useApi()
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [closing, setClosing] = useState(false)

  const fetchTicket = useCallback(() => {
    if (!id) return
    request<Ticket>(`tickets/${id}`).then(({ data }) => setTicket(data ?? null))
  }, [id, request])

  useEffect(() => {
    fetchTicket()
  }, [fetchTicket])

  const closeTicket = async () => {
    if (!id || !ticket || ticket.status === 'closed') return
    setClosing(true)
    const { error } = await request<Ticket>(`tickets/${id}`, {
      method: 'PUT',
      body: { status: 'closed' as const },
    })
    setClosing(false)
    if (!error) setTicket((prev) => (prev ? { ...prev, status: 'closed' } : null))
  }

  if (!ticket) return <p className="empty">Загрузка…</p>

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/tickets">Тикеты</Link>
        {' / '}
        <span>{ticket.id}</span>
      </nav>
      <h1 className="page-title">Тикет</h1>
      <div className="card">
        <p><strong>Запрос:</strong></p>
        <pre style={{ whiteSpace: 'pre-wrap', background: 'var(--bg)', padding: '0.75rem', borderRadius: 6 }}>
          {ticket.query}
        </pre>
        <p><strong>Категория:</strong> {ticket.category ?? '—'}</p>
        <p><strong>Confidence:</strong> {ticket.confidence != null ? (ticket.confidence * 100).toFixed(1) + '%' : '—'}</p>
        <p><strong>Статус:</strong> <span className={`badge ${ticket.status}`}>{ticket.status}</span></p>
        <p><strong>Назначен:</strong> {ticket.assigned_to ?? '—'}</p>
        <p><strong>Создан:</strong> {ticket.created_at ? new Date(ticket.created_at).toLocaleString() : '—'}</p>
        <p><strong>Обновлён:</strong> {ticket.updated_at ? new Date(ticket.updated_at).toLocaleString() : '—'}</p>
        {ticket.resolution_notes != null && ticket.resolution_notes !== '' && (
          <p><strong>Заметки по решению:</strong> {ticket.resolution_notes}</p>
        )}
        {ticket.converted_to_case_id && (
          <p>
            <strong>Конвертирован в кейс:</strong>{' '}
            <Link to={`/cases/${ticket.converted_to_case_id}`}>{ticket.converted_to_case_id}</Link>
          </p>
        )}
      </div>
      <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
        <Link to={`/tickets/${id}/edit`}>
          <button type="button" className="primary">Редактировать</button>
        </Link>
        {ticket.status !== 'closed' && (
          <button type="button" onClick={closeTicket} disabled={closing}>
            {closing ? 'Закрываю…' : 'Закрыть тикет'}
          </button>
        )}
        {!ticket.converted_to_case_id && (
          <Link to={`/tickets/${id}/convert`}>
            <button type="button">Конвертировать в кейс</button>
          </Link>
        )}
      </div>
    </>
  )
}
