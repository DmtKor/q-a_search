import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '@/api/request'
import { useSettings } from '@/store/settings'
import type { CategoriesResponse, SearchRequest, SearchResponse, Chunk } from '@/api/types'

function ChunkRow({ chunk, isStaff }: { chunk: Chunk; isStaff: boolean }) {
  const [expanded, setExpanded] = useState(false)
  const snippet = chunk.text.slice(0, 120) + (chunk.text.length > 120 ? '…' : '')
  return (
    <div className="card" style={{ cursor: 'pointer' }} onClick={() => setExpanded(!expanded)}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
        <div>
          <strong>{chunk.title}</strong>
          {isStaff && (
            <span style={{ marginLeft: '0.5rem' }}>
              <Link to={`/cases/${chunk.case_id}`} onClick={(e) => e.stopPropagation()}>
                Кейс
              </Link>
            </span>
          )}
          <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '0.25rem' }}>
            {expanded ? chunk.text : snippet}
          </div>
        </div>
        <span className="badge" style={{ background: 'var(--bg-hover)' }}>
          {(chunk.confidence * 100).toFixed(0)}%
        </span>
      </div>
    </div>
  )
}

export function Search() {
  const { request, getTokenForRequest } = useApi()
  const { accessType } = useSettings()
  const isStaff = accessType === 'staff'
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [topK, setTopK] = useState<number | ''>(10)
  const [userContextRaw, setUserContextRaw] = useState('')
  const [noTicketOnLowConfidence, setNoTicketOnLowConfidence] = useState(true)
  const [result, setResult] = useState<SearchResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [categories, setCategories] = useState<string[]>([])
  const [categoryOtherMode, setCategoryOtherMode] = useState(false)

  useEffect(() => {
    request<CategoriesResponse>('cases/categories').then(({ data }) => {
      if (data?.categories) setCategories(data.categories)
    })
  }, [request])

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!query.trim()) return
    if (!getTokenForRequest()) return
    setLoading(true)
    setResult(null)
    let user_context: Record<string, unknown> | undefined
    if (userContextRaw.trim()) {
      try {
        user_context = JSON.parse(userContextRaw) as Record<string, unknown>
      } catch {
        setResult({ chunks: [] })
        setLoading(false)
        return
      }
    }
    const body: SearchRequest = {
      query: query.trim(),
      ...(category.trim() && { category: category.trim() }),
      ...(topK !== '' && Number(topK) >= 1 && Number(topK) <= 50 && { top_k: Number(topK) }),
      ...(user_context && { user_context }),
      ...(noTicketOnLowConfidence && { no_ticket_on_low_confidence: true }),
    }
    try {
      const { data } = await request<SearchResponse>('search', { method: 'POST', body })
      if (data) setResult(data)
      else setResult({ chunks: [] })
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <h1 className="page-title">Поиск</h1>
      <form onSubmit={handleSearch} className="card" style={{ maxWidth: 560, marginBottom: '1.5rem' }}>
        <div className="form-group">
          <label htmlFor="query">Запрос *</label>
          <input
            id="query"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Текст запроса"
            required
          />
        </div>
        <div className="form-group">
          <label htmlFor="category">Категория</label>
          <select
            id="category"
            value={categoryOtherMode ? '__other__' : (categories.includes(category) ? category : '')}
            onChange={(e) => {
              const v = e.target.value
              setCategoryOtherMode(v === '__other__')
              setCategory(v === '__other__' ? '' : v)
            }}
            style={{ width: '100%', maxWidth: 280 }}
          >
            <option value="">Не выбрано</option>
            {categories.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
            <option value="__other__">— ввести свою —</option>
          </select>
          {(categoryOtherMode || (category && !categories.includes(category))) && (
            <input
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="Введите категорию"
              style={{ marginTop: '0.5rem', width: '100%', maxWidth: 280 }}
            />
          )}
        </div>
        <div className="form-group">
          <label htmlFor="topK">Количество результатов (top_k)</label>
          <input
            id="topK"
            type="number"
            min={1}
            max={50}
            value={topK === '' ? '' : topK}
            onChange={(e) => setTopK(e.target.value === '' ? '' : Number(e.target.value))}
          />
        </div>
        <div className="form-group">
          <label htmlFor="user_context">user_context (JSON)</label>
          <textarea
            id="user_context"
            value={userContextRaw}
            onChange={(e) => setUserContextRaw(e.target.value)}
            placeholder='{"key": "value"}'
            rows={3}
            style={{ maxWidth: '100%' }}
          />
        </div>
        <button type="submit" className="primary" disabled={loading}>
          {loading ? 'Поиск…' : 'Искать'}
        </button>
        <div style={{ marginTop: '1rem', textAlign: 'left' }}>
          <label style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem', cursor: 'pointer', whiteSpace: 'nowrap' }}>
            <input
              type="checkbox"
              checked={noTicketOnLowConfidence}
              onChange={(e) => setNoTicketOnLowConfidence(e.target.checked)}
            />
            Не создавать тикет при плохом результате
          </label>
        </div>
      </form>

      {result && (
        <>
          {result.ticket && (
            <div className="card" style={{ borderColor: 'var(--accent)', marginBottom: '1rem' }}>
              <strong>Создан тикет для ручной обработки</strong>
              <div style={{ marginTop: '0.5rem' }}>
                <Link to={`/tickets/${result.ticket.id}`}>Тикет {result.ticket.id}</Link>
                {result.ticket.url && (
                  <span style={{ marginLeft: '0.5rem', color: 'var(--text-muted)' }}>{result.ticket.url}</span>
                )}
              </div>
            </div>
          )}
          {result.chunks && result.chunks.length > 0 ? (
            result.chunks.map((chunk) => (
              <ChunkRow key={`${chunk.case_id}-${chunk.title}-${chunk.confidence}`} chunk={chunk} isStaff={isStaff} />
            ))
          ) : (
            !result.ticket && <p className="empty">Нет результатов</p>
          )}
        </>
      )}
    </>
  )
}
