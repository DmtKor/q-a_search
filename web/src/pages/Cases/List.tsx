import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '@/api/request'
import { clientLog } from '@/lib/clientLog'
import type { Case, CaseStatus } from '@/api/types'

function groupByCategory(cases: Case[]): Map<string, Case[]> {
  const map = new Map<string, Case[]>()
  for (const c of cases) {
    const cat = c.category || '(без категории)'
    if (!map.has(cat)) map.set(cat, [])
    map.get(cat)!.push(c)
  }
  return map
}

export function CasesList() {
  const { request } = useApi()
  const [list, setList] = useState<Case[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<CaseStatus | ''>('')
  const [categoryFilter, setCategoryFilter] = useState('')
  const [mineOnly, setMineOnly] = useState(false)
  const [appliedStatus, setAppliedStatus] = useState<CaseStatus | ''>('')
  const [appliedCategory, setAppliedCategory] = useState('')
  const [appliedMine, setAppliedMine] = useState(false)
  const [expandedCategories, setExpandedCategories] = useState<Set<string> | null>(null)

  const fetchList = useCallback(() => {
    setAppliedStatus(statusFilter)
    setAppliedCategory(categoryFilter)
    setAppliedMine(mineOnly)
    setLoading(true)
    const params = new URLSearchParams()
    if (statusFilter) params.set('status', statusFilter)
    if (categoryFilter.trim()) params.set('category', categoryFilter.trim())
    if (mineOnly) params.set('mine', 'true')
    const q = params.toString()
    request<Case[]>(`cases${q ? `?${q}` : ''}`).then(({ data }) => {
      setList(Array.isArray(data) ? data : [])
      setLoading(false)
      clientLog('cases_filter_applied', { status: statusFilter, category: categoryFilter.trim(), mine: mineOnly, count: Array.isArray(data) ? data.length : 0 })
    })
  }, [statusFilter, categoryFilter, mineOnly, request])

  useEffect(() => {
    fetchList()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps -- load once on mount

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

  const hasFilterChange =
    appliedStatus !== statusFilter || appliedCategory !== categoryFilter.trim() || appliedMine !== mineOnly

  return (
    <>
      <h1 className="page-title">База знаний (Кейсы)</h1>
      <div className="card" style={{ marginBottom: '1rem' }}>
        <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', alignItems: 'center' }}>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as CaseStatus | '')}
            style={{ width: 'auto' }}
            aria-label="Статус"
          >
            <option value="">Все статусы</option>
            <option value="draft">draft</option>
            <option value="pending_review">pending_review</option>
            <option value="approved">approved</option>
            <option value="archived">archived</option>
          </select>
          <input
            type="text"
            placeholder="Категория"
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
            style={{ width: 160 }}
            aria-label="Категория"
          />
          <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', margin: 0 }}>
            <input type="checkbox" checked={mineOnly} onChange={(e) => setMineOnly(e.target.checked)} />
            Только мои
          </label>
          <button type="button" className="primary" onClick={fetchList} disabled={loading}>
            {loading ? 'Загрузка…' : 'Применить фильтры'}
          </button>
        </div>
        {hasFilterChange && !loading && (
          <p style={{ margin: '0.5rem 0 0', fontSize: '0.85rem', color: 'var(--text-muted)' }}>
            Изменены фильтры. Нажмите «Применить фильтры».
          </p>
        )}
      </div>
      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', marginBottom: '1rem', alignItems: 'center' }}>
        <Link to="/cases/new">
          <button type="button" className="primary">Создать кейс</button>
        </Link>
        <Link to="/cases/import">
          <button type="button">Импорт</button>
        </Link>
        <Link to="/cases/export">
          <button type="button">Экспорт</button>
        </Link>
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

      {loading && list.length === 0 ? (
        <p className="empty">Загрузка…</p>
      ) : list.length === 0 ? (
        <p className="empty">Нет кейсов. Задайте фильтры и нажмите «Применить фильтры» или создайте кейс.</p>
      ) : (
        Array.from(grouped.entries()).map(([category, cases]) => {
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
                  ({cases.length})
                </span>
              </button>
              {isExpanded && (
                <ul style={{ listStyle: 'none', padding: '0.75rem 0 0 1.25rem', margin: 0 }}>
                  {cases.map((c) => (
                    <li key={c.id} style={{ marginBottom: '0.5rem' }}>
                      <Link to={`/cases/${c.id}`}>
                        <strong>{c.title}</strong>
                      </Link>
                      {' '}
                      <span className={`badge ${c.status}`}>{c.status}</span>
                      {c.updated_at && (
                        <span style={{ color: 'var(--text-muted)', marginLeft: '0.5rem', fontSize: '0.85rem' }}>
                          {new Date(c.updated_at).toLocaleString()}
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )
        })
      )}
    </>
  )
}
