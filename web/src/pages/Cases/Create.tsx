import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { CaseCreate as CaseCreateType, CategoriesResponse } from '@/api/types'

export function CaseCreate() {
  const navigate = useNavigate()
  const { request } = useApi()
  const [categories, setCategories] = useState<string[]>([])
  const [categoryOtherMode, setCategoryOtherMode] = useState(false)
  const [category, setCategory] = useState('')
  const [title, setTitle] = useState('')
  const [responseTemplate, setResponseTemplate] = useState('')
  const [questions, setQuestions] = useState('')
  const [keywords, setKeywords] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    request<CategoriesResponse>('cases/categories').then(({ data }) => {
      if (data?.categories) setCategories(data.categories)
    })
  }, [request])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!category.trim() || !title.trim() || !responseTemplate.trim()) return
    setSubmitting(true)
    const body: CaseCreateType = {
      category: category.trim(),
      title: title.trim(),
      response_template: responseTemplate.trim(),
      questions: questions.trim() ? questions.split('\n').map((s) => s.trim()).filter(Boolean) : [],
      keywords: keywords.trim() ? keywords.split('\n').map((s) => s.trim()).filter(Boolean) : [],
    }
    const { data, error } = await request<{ id: string }>('cases', { method: 'POST', body })
    setSubmitting(false)
    if (!error && data?.id) navigate(`/cases/${data.id}`)
  }

  return (
    <>
      <nav style={{ marginBottom: '1rem', color: 'var(--text-muted)' }}>
        <Link to="/cases">База знаний</Link>
        {' / Создать'}
      </nav>
      <h1 className="page-title">Создать кейс</h1>
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
          <textarea value={responseTemplate} onChange={(e) => setResponseTemplate(e.target.value)} rows={4} required />
        </div>
        <div className="form-group">
          <label>Вопросы (по одному на строку)</label>
          <textarea value={questions} onChange={(e) => setQuestions(e.target.value)} rows={3} placeholder="вопрос 1&#10;вопрос 2" />
        </div>
        <div className="form-group">
          <label>Ключевые слова (по одному на строку)</label>
          <textarea value={keywords} onChange={(e) => setKeywords(e.target.value)} rows={2} />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? 'Создание…' : 'Создать'}
        </button>
      </form>
    </>
  )
}
