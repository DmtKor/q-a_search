import { useSettings } from '@/store/settings'
import { getLogBuffer, clearLogBuffer } from '@/lib/clientLog'

export function Settings() {
  const { token, baseURL, accessType, checked, checking, checkError, setToken, setBaseURL, saveToStorage } = useSettings()

  const downloadUiLogs = () => {
    const buf = getLogBuffer()
    const blob = new Blob([JSON.stringify(buf, null, 2)], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `ui-logs-${new Date().toISOString().slice(0, 19).replace(/T/g, '_')}.json`
    a.click()
    URL.revokeObjectURL(a.href)
  }

  return (
    <>
      <h1 className="page-title">Настройки</h1>
      <div className="card" style={{ maxWidth: 480 }}>
        <div className="form-group">
          <label htmlFor="token">Bearer token</label>
          <input
            id="token"
            type="password"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="Введите токен"
            autoComplete="off"
          />
        </div>
        <div className="form-group">
          <label htmlFor="baseURL">Базовый URL API</label>
          <input
            id="baseURL"
            type="text"
            value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)}
            placeholder="Напр. http://localhost:8080"
          />
          {!baseURL.trim() && (
            <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', marginTop: '0.35rem' }}>
              Если API на другом порту (UI на 5173, API на 8080), укажите базовый URL, например http://localhost:8080
            </p>
          )}
        </div>
        <div className="form-group">
          <button type="button" className="primary" onClick={saveToStorage} disabled={checking}>
            {checking ? 'Проверка доступа…' : 'Сохранить в браузере'}
          </button>
        </div>
        {checkError && (
          <p style={{ marginTop: '0.75rem', color: 'var(--error, #f85149)' }} role="alert">
            {checkError}
          </p>
        )}
        {checked && !checkError && (
          <p style={{ color: 'var(--text-muted)', marginTop: '1rem' }}>
            Текущий доступ: {accessType === 'staff' ? 'Staff' : accessType === 'app' ? 'App' : '— введите токен и сохраните'}
          </p>
        )}
      </div>
      {!token.trim() && (
        <p className="empty">Без валидного токена запросы не отправляются. Введите токен в настройках выше.</p>
      )}
      <div className="card" style={{ maxWidth: 480, marginTop: '1.5rem' }}>
        <h2 style={{ margin: '0 0 0.5rem', fontSize: '1rem' }}>Логи UI</h2>
        <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', margin: 0 }}>
          Действия и ошибки API пишутся в консоль браузера с префиксом [ui]. Буфер можно скачать.
        </p>
        <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.75rem' }}>
          <button type="button" onClick={downloadUiLogs}>
            Скачать логи UI
          </button>
          <button type="button" onClick={clearLogBuffer}>
            Очистить буфер
          </button>
        </div>
      </div>
    </>
  )
}
