import { useEffect, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { Case, CaseUpdate } from '@/api/types'

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

  useEffect(() => {
    if (!id) return
    request<Case>(`cases/${id}`).then(({ data }) => {
      if (data) {
        setCase(data)
        setCategory(data.category ?? '')
        setTitle(data.title ?? '')
        setResponseTemplate(data.response_template ?? '')
        setQuestions((data.questions ?? []).join('\n'))
        setKeywords((data.keywords ?? []).join('\n'))
        setNotes(data.notes ?? '')
      }
    })
  }, [id])

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
          <input value={category} onChange={(e) => setCategory(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Заголовок *</label>
          <input value={title} onChange={(e) => setTitle(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Шаблон ответа *</label>
          <textarea value={responseTemplate} onChange={(e) => setResponseTemplate(e.target.value)} rows={4} required />
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
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Сохранение…' : 'Сохранить'}
        </button>
      </form>
    </>
  )
}
