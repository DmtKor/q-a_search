import { Link } from 'react-router-dom'
import { useError } from '@/context/error'

export function ErrorBanner() {
  const { error, clearError } = useError()
  if (!error) return null
  return (
    <div className="error-banner" role="alert">
      <strong>Ошибка API</strong>
      {' '}
      {error.status > 0 && <span>(HTTP {error.status})</span>}
      {' — '}
      <span>{error.code}: {error.message}</span>
      {error.details != null && Object.keys(error.details).length > 0 && (
        <pre style={{ margin: '0.5rem 0 0', fontSize: '0.85rem', overflow: 'auto' }}>
          {JSON.stringify(error.details, null, 2)}
        </pre>
      )}
      {error.status === 401 && (
        <>
          {' '}
          <Link to="/settings" onClick={clearError}>
            Перейти в Настройки и обновить токен
          </Link>
        </>
      )}
      <button type="button" onClick={clearError} style={{ marginLeft: '1rem' }}>
        Закрыть
      </button>
    </div>
  )
}
