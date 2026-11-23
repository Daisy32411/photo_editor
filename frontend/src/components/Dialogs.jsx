import { ArrowRight, LockKeyhole, X } from 'lucide-react'
import { useState } from 'react'
import { login, register } from '../api/client.js'

export function AuthDialog({ onClose, onSuccess }) {
  const [mode, setMode] = useState('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(event) {
    event.preventDefault()
    if (busy) return
    const name = username.trim()
    if (!name || !password) return
    setBusy(true)
    setError('')

    try {
      if (mode === 'register') await register(name, password)
      const { token } = await login(name, password)
      if (!token) throw new Error('Не удалось войти. Попробуй ещё раз.')
      onSuccess({ username: name, token })
    } catch (err) {
      if (err.status === 409) {
        setError('Это имя пользователя уже занято.')
      } else if (err.status === 401) {
        setError('Неверный логин или пароль.')
      } else {
        setError(err.message || 'Не удалось выполнить запрос.')
      }
    } finally {
      setBusy(false)
    }
  }

  function changeMode(nextMode) {
    setMode(nextMode)
    setError('')
  }

  return (
    <div className="modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !busy) onClose() }}>
      <div className="modal-panel auth-modal" role="dialog" aria-modal="true" aria-labelledby="auth-title">
        <button className="modal-close" onClick={onClose} disabled={busy} aria-label="Закрыть"><X size={18}/></button>
        <div className="modal-icon"><LockKeyhole size={23}/></div>
        <h2 id="auth-title">{mode === 'login' ? 'Войти в PhotoLab' : 'Регистрация'}</h2>
        <p>Войди или создай аккаунт, чтобы сохранять результаты обработки.</p>
        <div className="auth-tabs">
          <button type="button" className={mode === 'login' ? 'current' : ''} onClick={() => changeMode('login')} disabled={busy}>Войти</button>
          <button type="button" className={mode === 'register' ? 'current' : ''} onClick={() => changeMode('register')} disabled={busy}>Регистрация</button>
        </div>
        <form onSubmit={handleSubmit}>
          <label>
            Имя пользователя
            <input type="text" autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} placeholder="Например, creator" required maxLength={80}/>
          </label>
          <label>
            Пароль
            <input type="password" autoComplete={mode === 'register' ? 'new-password' : 'current-password'} value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Введите пароль" required/>
          </label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <button type="submit" className="modal-primary" disabled={busy}>
            {busy ? 'Подождите…' : mode === 'login' ? 'Войти' : 'Создать аккаунт'}
            {!busy && <ArrowRight size={17}/>}
          </button>
        </form>
      </div>
    </div>
  )
}
