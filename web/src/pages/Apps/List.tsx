import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '@/api/request'
import type { App } from '@/api/types'

export function AppsList() {
  const { request } = useApi()
  const [list, setList] = useState<App[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    request<App[]>('apps').then(({ data }) => {
      if (!cancelled && data) setList(Array.isArray(data) ? data : [])
      if (!cancelled) setLoading(false)
    })
    return () => { cancelled = true }
  }, [request])

  return (
    <>
      <h1 className="page-title">Приложения</h1>
      <div style={{ marginBottom: '1rem' }}>
        <Link to="/apps/new">
          <button type="button" className="primary">Создать приложение</button>
        </Link>
      </div>
      {loading ? (
        <p className="empty">Загрузка…</p>
      ) : list.length === 0 ? (
        <p className="empty">Нет приложений</p>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Название</th>
                <th>ID</th>
                <th>Обновлён</th>
              </tr>
            </thead>
            <tbody>
              {list.map((app) => (
                <tr key={app.id} className="clickable-row">
                  <td>
                    <Link to={`/apps/${app.id}`}>{app.name}</Link>
                  </td>
                  <td>{app.id}</td>
                  <td>{app.updated_at ? new Date(app.updated_at).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}
