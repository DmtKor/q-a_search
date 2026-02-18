import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { App, AppCreate } from '@/api/types'

export function AppCreate() {
  const navigate = useNavigate()
  const { request } = useApi()
  const [name, setName] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    setSubmitting(true)
    const body: AppCreate = { name: name.trim() }
    const { data, error } = await request<App>('apps', { method: 'POST', body })
    setSubmitting(false)
    if (!error && data?.id) navigate(`/apps/${data.id}`)
  }

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/apps">Приложения</Link>
        {' / Создать'}
      </nav>
      <h1 className="page-title">Создать приложение</h1>
      <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 400 }}>
        <div className="form-group">
          <label>Название *</label>
          <input value={name} onChange={(e) => setName(e.target.value)} required />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Создание…' : 'Создать'}
        </button>
      </form>
    </>
  )
}
