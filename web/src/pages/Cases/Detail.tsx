import { useEffect, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Case, ReadableSegment } from '@/api/types'
import { TemplateReadableBlock } from '@/components/TemplateReadableBlock'

export function CaseDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { request } = useApi()
  const [case_, setCase] = useState<Case | null>(null)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewJson, setPreviewJson] = useState('{}')
  const [previewResult, setPreviewResult] = useState<string | null>(null)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [readableView, setReadableView] = useState(false)
  const [readableSegments, setReadableSegments] = useState<ReadableSegment[] | null>(null)
  const [readableLoading, setReadableLoading] = useState(false)

  useEffect(() => {
    if (!id) return
    request<Case>(`cases/${id}`).then(({ data }) => setCase(data ?? null))
  }, [id])

  const handleRenderPreview = async () => {
    if (!case_) return
    setPreviewError(null)
    setPreviewResult(null)
    let userContext: Record<string, unknown>
    try {
      userContext = JSON.parse(previewJson || '{}') as Record<string, unknown>
    } catch {
      setPreviewError('Неверный JSON в поле user_context')
      return
    }
    setPreviewLoading(true)
    const { data, error } = await request<{ text: string }>('cases/render-preview', {
      method: 'POST',
      body: { template: case_.response_template ?? '', user_context: userContext },
    })
    setPreviewLoading(false)
    if (error) {
      setPreviewError(error.message)
      return
    }
    setPreviewResult(data?.text ?? '')
  }

  const handleToggleReadableView = async () => {
    if (!readableView) {
      if (!case_) return
      setReadableLoading(true)
      const { data } = await request<{ segments: ReadableSegment[] }>('cases/template-readable', {
        method: 'POST',
        body: { template: case_.response_template ?? '' },
      })
      setReadableLoading(false)
      setReadableSegments(data?.segments ?? null)
    }
    setReadableView((v) => !v)
  }

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
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap' }}>
          <button type="button" onClick={handleToggleReadableView} disabled={readableLoading}>
            {readableView ? 'Код шаблона' : 'Читаемый вид'}
          </button>
          {readableLoading && <span style={{ color: 'var(--text-muted)' }}>Загрузка…</span>}
        </div>
        {readableView && readableSegments ? (
          <div style={{ marginTop: '0.5rem', background: 'var(--bg)', padding: '0.75rem', borderRadius: 6 }}>
            <TemplateReadableBlock segments={readableSegments} />
          </div>
        ) : (
          <pre style={{ whiteSpace: 'pre-wrap', background: 'var(--bg)', padding: '0.75rem', borderRadius: 6, marginTop: '0.5rem' }}>
            {case_.response_template}
          </pre>
        )}
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
        <button type="button" onClick={() => setPreviewOpen((v) => !v)}>
          {previewOpen ? 'Скрыть тест' : 'Тест'}
        </button>
        <button type="button" onClick={handleDelete}>Удалить (архивировать)</button>
      </div>
      {previewOpen && (
        <div className="card" style={{ marginTop: '1rem', padding: '1rem', maxWidth: 560 }}>
          <div className="form-group">
            <label>user_context (JSON)</label>
            <textarea
              value={previewJson}
              onChange={(e) => setPreviewJson(e.target.value)}
              rows={4}
              placeholder='{"name": "Иван", "product": "Подписка"}'
              style={{ fontFamily: 'monospace' }}
            />
          </div>
          <button type="button" className="primary" onClick={handleRenderPreview} disabled={previewLoading}>
            {previewLoading ? 'Отрендерить…' : 'Отрендерить'}
          </button>
          {previewError && <p className="error" style={{ marginTop: '0.75rem' }}>{previewError}</p>}
          {previewResult !== null && !previewError && (
            <div style={{ marginTop: '0.75rem' }}>
              <label style={{ display: 'block', marginBottom: '0.25rem', color: 'var(--text-muted)' }}>
                Результат
              </label>
              <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', margin: 0 }}>{previewResult}</pre>
            </div>
          )}
        </div>
      )}
    </>
  )
}
