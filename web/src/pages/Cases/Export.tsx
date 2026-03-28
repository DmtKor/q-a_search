import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Case, CaseStatus, CategoriesResponse } from '@/api/types'

export function CaseExport() {
  const { request, getTokenForRequest } = useApi()
  const [categories, setCategories] = useState<string[]>([])
  const [categoryOtherMode, setCategoryOtherMode] = useState(false)
  const [status, setStatus] = useState<CaseStatus | ''>('')
  const [category, setCategory] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    request<CategoriesResponse>('cases/categories').then(({ data }) => {
      if (data?.categories) setCategories(data.categories)
    })
  }, [request])

  const handleExport = async () => {
    const token = getTokenForRequest()
    if (!token) return
    setLoading(true)
    const params = new URLSearchParams()
    if (status) params.set('status', status)
    if (category) params.set('category', category)
    const q = params.toString()
    const { data, error } = await request<Case[]>(`cases/export${q ? `?${q}` : ''}`)
    setLoading(false)
    if (error || !data) return
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const name = `cases-export-${new Date().toISOString().slice(0, 19).replace(/T/, 'T').replace(/:/g, '-')}.json`
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = name
    a.click()
    URL.revokeObjectURL(a.href)
  }

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/cases">База знаний</Link>
        {' / Экспорт'}
      </nav>
      <h1 className="page-title">Экспорт кейсов</h1>
      <div className="card" style={{ maxWidth: 400 }}>
        <div className="form-group">
          <label>Статус</label>
          <select value={status} onChange={(e) => setStatus(e.target.value as CaseStatus | '')} style={{ width: 'auto' }}>
            <option value="">Все</option>
            <option value="draft">draft</option>
            <option value="pending_review">pending_review</option>
            <option value="approved">approved</option>
            <option value="archived">archived</option>
          </select>
        </div>
        <div className="form-group">
          <label>Категория</label>
          <select
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
        <button type="button" className="primary" onClick={handleExport} disabled={loading}>
          {loading ? 'Экспорт…' : 'Скачать JSON'}
        </button>
      </div>
    </>
  )
}
