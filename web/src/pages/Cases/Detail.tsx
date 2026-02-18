import { useEffect, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Case } from '@/api/types'

export function CaseDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { request } = useApi()
  const [case_, setCase] = useState<Case | null>(null)

  useEffect(() => {
    if (!id) return
    request<Case>(`cases/${id}`).then(({ data }) => setCase(data ?? null))
  }, [id])

  const handleDelete = async () => {
    if (!id || !window.confirm('Удалить (архивировать) этот кейс?')) return
    const { error } = await request(`cases/${id}`, { method: 'DELETE' })
    if (!error) navigate('/cases')
  }

  if (!case_) return <p className="empty">Загрузка…</p>

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/cases">База знаний</Link>
        {' / '}
        <span>{case_.title}</span>
      </nav>
      <h1 className="page-title">{case_.title}</h1>
      <div className="card">
        <p><strong>Категория:</strong> {case_.category}</p>
        <p><strong>Статус:</strong> <span className={`badge ${case_.status}`}>{case_.status}</span></p>
        <p><strong>Создан:</strong> {case_.created_by ?? '—'} {case_.created_at && new Date(case_.created_at).toLocaleString()}</p>
        <p><strong>Обновлён:</strong> {case_.updated_at && new Date(case_.updated_at).toLocaleString()}</p>
        {case_.questions?.length ? (
          <p><strong>Вопросы:</strong> {case_.questions.join(', ')}</p>
        ) : null}
        {case_.keywords?.length ? (
          <p><strong>Ключевые слова:</strong> {case_.keywords.join(', ')}</p>
        ) : null}
        <p><strong>Шаблон ответа:</strong></p>
        <pre style={{ whiteSpace: 'pre-wrap', background: 'var(--bg)', padding: '0.75rem', borderRadius: 6 }}>
          {case_.response_template}
        </pre>
        {case_.notes != null && case_.notes !== '' && (
          <p><strong>Заметки:</strong> {case_.notes}</p>
        )}
      </div>
      <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
        <Link to={`/cases/${id}/edit`}>
          <button type="button" className="primary">Редактировать</button>
        </Link>
        <Link to={`/cases/${id}/status`} onClick={(e) => e.stopPropagation?.()}>
          <button type="button">Сменить статус</button>
        </Link>
        <button type="button" onClick={handleDelete}>Удалить (архивировать)</button>
      </div>
    </>
  )
}
