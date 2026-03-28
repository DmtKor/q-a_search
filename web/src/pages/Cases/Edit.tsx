import { useEffect, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Case, CaseUpdate, ReadableSegment, CategoriesResponse } from '@/api/types'
import { TemplateReadableBlock } from '@/components/TemplateReadableBlock'

export function CaseEdit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { request } = useApi()
  const [case_, setCase] = useState<Case | null>(null)
  const [category, setCategory] = useState('')
  const [title, setTitle] = useState('')
  const [responseTemplate, setResponseTemplate] = useState('')
  const [questions, setQuestions] = useState('')
  const [keywords, setKeywords] = useState('')
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewJson, setPreviewJson] = useState('{}')
  const [previewResult, setPreviewResult] = useState<string | null>(null)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [readableView, setReadableView] = useState(false)
  const [readableSegments, setReadableSegments] = useState<ReadableSegment[] | null>(null)
  const [readableLoading, setReadableLoading] = useState(false)
  const [categories, setCategories] = useState<string[]>([])
  const [categoryOtherMode, setCategoryOtherMode] = useState(false)

  useEffect(() => {
    request<CategoriesResponse>('cases/categories').then(({ data }) => {
      if (data?.categories) setCategories(data.categories)
    })
  }, [request])

  useEffect(() => {
    if (!id) return
    request<Case>(`cases/${id}`).then(({ data }) => {
      if (data) {
        setCase(data)
        const cat = data.category ?? ''
        setCategory(cat)
        setCategoryOtherMode(!!(cat && !categories.includes(cat)))
        setTitle(data.title ?? '')
        setResponseTemplate(data.response_template ?? '')
        setQuestions((data.questions ?? []).join('\n'))
        setKeywords((data.keywords ?? []).join('\n'))
        setNotes(data.notes ?? '')
      }
    })
  }, [id, categories])

  const handleRenderPreview = async () => {
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
      body: { template: responseTemplate, user_context: userContext },
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
      setReadableLoading(true)
      const { data } = await request<{ segments: ReadableSegment[] }>('cases/template-readable', {
        method: 'POST',
        body: { template: responseTemplate },
      })
      setReadableLoading(false)
      setReadableSegments(data?.segments ?? null)
    }
    setReadableView((v) => !v)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    setSubmitting(true)
    const body: CaseUpdate = {
      category: category.trim(),
      title: title.trim(),
      response_template: responseTemplate.trim(),
      questions: questions.trim() ? questions.split('\n').map((s) => s.trim()).filter(Boolean) : [],
      keywords: keywords.trim() ? keywords.split('\n').map((s) => s.trim()).filter(Boolean) : [],
      notes: notes.trim() || undefined,
    }
    const { error } = await request<Case>(`cases/${id}`, { method: 'PUT', body })
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
        {' / Редактировать'}
      </nav>
      <h1 className="page-title">Редактировать кейс</h1>
      <form onSubmit={handleSubmit} className="card" style={{ maxWidth: 560 }}>
        <div className="form-group">
          <label>Категория *</label>
          <select
            value={categoryOtherMode ? '__other__' : (categories.includes(category) ? category : '')}
            onChange={(e) => {
              const v = e.target.value
              setCategoryOtherMode(v === '__other__')
              setCategory(v === '__other__' ? '' : v)
            }}
            style={{ width: '100%', maxWidth: 280 }}
            required={!categoryOtherMode && !category}
          >
            <option value="">Выберите или введите свою ниже</option>
            {categories.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
            <option value="__other__">— ввести новую —</option>
          </select>
          {(categoryOtherMode || (category && !categories.includes(category))) && (
            <input
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="Новая категория"
              style={{ marginTop: '0.5rem', width: '100%', maxWidth: 280 }}
              required
            />
          )}
        </div>
        <div className="form-group">
          <label>Заголовок *</label>
          <input value={title} onChange={(e) => setTitle(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Шаблон ответа *</label>
          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap', marginBottom: '0.5rem' }}>
            <button type="button" onClick={handleToggleReadableView} disabled={readableLoading}>
              {readableView ? 'Код шаблона' : 'Читаемый вид'}
            </button>
            {readableLoading && <span style={{ color: 'var(--text-muted)' }}>Загрузка…</span>}
          </div>
          {readableView && readableSegments ? (
            <div style={{ background: 'var(--bg)', padding: '0.75rem', borderRadius: 6 }}>
              <TemplateReadableBlock segments={readableSegments} />
            </div>
          ) : (
            <textarea value={responseTemplate} onChange={(e) => setResponseTemplate(e.target.value)} rows={4} required />
          )}
        </div>
        <div className="form-group">
          <label>Вопросы (по одному на строку)</label>
          <textarea value={questions} onChange={(e) => setQuestions(e.target.value)} rows={3} />
        </div>
        <div className="form-group">
          <label>Ключевые слова (по одному на строку)</label>
          <textarea value={keywords} onChange={(e) => setKeywords(e.target.value)} rows={2} />
        </div>
        <div className="form-group">
          <label>Заметки</label>
          <textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
        </div>
        <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
          <button type="submit" className="primary" disabled={submitting}>
            {submitting ? 'Сохранение…' : 'Сохранить'}
          </button>
          <button
            type="button"
            className="secondary"
            onClick={() => setPreviewOpen((v) => !v)}
          >
            {previewOpen ? 'Скрыть тест' : 'Тест'}
          </button>
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
      </form>
    </>
  )
}
