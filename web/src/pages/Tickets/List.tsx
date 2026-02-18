import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Ticket, TicketStatus } from '@/api/types'

const QUERY_PREVIEW_LEN = 60

function groupByCategory(tickets: Ticket[]): Map<string, Ticket[]> {
  const map = new Map<string, Ticket[]>()
  for (const t of tickets) {
    const cat = t.category?.trim() || '(без категории)'
    if (!map.has(cat)) map.set(cat, [])
    map.get(cat)!.push(t)
  }
  return map
}

export function TicketsList() {
  const { request } = useApi()
  const [list, setList] = useState<Ticket[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<TicketStatus | ''>('')
  const [categoryFilter, setCategoryFilter] = useState('')
  const [closingId, setClosingId] = useState<string | null>(null)
  const [expandedCategories, setExpandedCategories] = useState<Set<string> | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    const params = new URLSearchParams()
    if (statusFilter) params.set('status', statusFilter)
    if (categoryFilter) params.set('category', categoryFilter)
    const q = params.toString()
    request<Ticket[]>(`tickets${q ? `?${q}` : ''}`).then(({ data }) => {
      if (!cancelled) {
        setList(Array.isArray(data) ? data : [])
        setLoading(false)
      }
    })
    return () => { cancelled = true }
  }, [statusFilter, categoryFilter, request])

  const closeTicket = async (id: string, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setClosingId(id)
    const { error } = await request<Ticket>(`tickets/${id}`, {
      method: 'PUT',
      body: { status: 'closed' as const },
    })
    setClosingId(null)
    if (!error) {
      setList((prev) => prev.map((t) => (t.id === id ? { ...t, status: 'closed' as const } : t)))
    }
  }

  const grouped = useMemo(() => groupByCategory(list), [list])

  const toggleCategory = (category: string) => {
    setExpandedCategories((prev) => {
      const base = prev === null ? new Set(grouped.keys()) : new Set(prev)
      const next = new Set(base)
      if (next.has(category)) next.delete(category)
      else next.add(category)
      return next.size === 0 ? new Set() : next
    })
  }

  const expandAll = () => setExpandedCategories(null)
  const collapseAll = () => setExpandedCategories(new Set())

  return (
    <>
      <h1 className="page-title">Тикеты</h1>
      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', marginBottom: '1rem', alignItems: 'center' }}>
        <Link to="/tickets/new">
          <button type="button" className="primary">Создать тикет вручную</button>
        </Link>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as TicketStatus | '')}
          style={{ width: 'auto' }}
        >
          <option value="">Все статусы</option>
          <option value="open">open</option>
          <option value="in_progress">in_progress</option>
          <option value="resolved">resolved</option>
          <option value="closed">closed</option>
        </select>
        <input
          type="text"
          placeholder="Категория"
          value={categoryFilter}
          onChange={(e) => setCategoryFilter(e.target.value)}
          style={{ width: 160 }}
        />
        {grouped.size > 0 && (
          <>
            <button type="button" onClick={expandAll} style={{ fontSize: '0.9rem' }}>
              Развернуть все
            </button>
            <button type="button" onClick={collapseAll} style={{ fontSize: '0.9rem' }}>
              Свернуть все
            </button>
          </>
        )}
      </div>

      {loading ? (
        <p className="empty">Загрузка…</p>
      ) : list.length === 0 ? (
        <p className="empty">Нет тикетов</p>
      ) : (
        Array.from(grouped.entries()).map(([category, tickets]) => {
          const isExpanded = expandedCategories === null || expandedCategories.has(category)
          return (
            <div key={category} className="card" style={{ marginBottom: '0.75rem' }}>
              <button
                type="button"
                onClick={() => toggleCategory(category)}
                style={{
                  width: '100%',
                  textAlign: 'left',
                  background: 'none',
                  border: 'none',
                  color: 'inherit',
                  cursor: 'pointer',
                  padding: 0,
                  margin: 0,
                  fontSize: '1rem',
                  fontWeight: 600,
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.5rem',
                }}
                aria-expanded={isExpanded}
              >
                <span style={{ userSelect: 'none' }}>{isExpanded ? '▼' : '▶'}</span>
                {category}
                <span style={{ color: 'var(--text-muted)', fontWeight: 400, fontSize: '0.9rem' }}>
                  ({tickets.length})
                </span>
              </button>
              {isExpanded && (
                <div className="table-wrap" style={{ marginTop: '0.75rem' }}>
                  <table>
                    <thead>
                      <tr>
                        <th>Запрос</th>
                        <th>Статус</th>
                        <th>Назначен</th>
                        <th>Создан</th>
                        <th style={{ width: 100 }}>Действия</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tickets.map((t) => (
                        <tr key={t.id} className="clickable-row">
                          <td>
                            <Link to={`/tickets/${t.id}`}>
                              {t.query.length > QUERY_PREVIEW_LEN ? t.query.slice(0, QUERY_PREVIEW_LEN) + '…' : t.query}
                            </Link>
                          </td>
                          <td><span className={`badge ${t.status}`}>{t.status}</span></td>
                          <td>{t.assigned_to ?? '—'}</td>
                          <td>{t.created_at ? new Date(t.created_at).toLocaleString() : '—'}</td>
                          <td onClick={(e) => e.stopPropagation()}>
                            {t.status !== 'closed' && (
                              <button
                                type="button"
                                onClick={(e) => closeTicket(t.id, e)}
                                disabled={closingId === t.id}
                                style={{ fontSize: '0.85rem', padding: '0.25rem 0.5rem' }}
                              >
                                {closingId === t.id ? '…' : 'Закрыть'}
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )
        })
      )}
    </>
  )
}
