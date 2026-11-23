import { LogOut, UserRound } from 'lucide-react'

export default function Topbar({ page, session, onAuth, onLogout }) {
  return (
    <header className="topbar">
      <div className="topbar-location">
        <h1>{page === 'history' ? 'История файлов' : 'Фоторедактор'}</h1>
      </div>
      <div className="topbar-actions">
        {session ? (
          <div className="account-signed-in">
            <span className="account-name"><UserRound size={17} /><span>{session.username}</span></span>
            <button className="account-signout" onClick={onLogout} title="Выйти" aria-label="Выйти из аккаунта"><LogOut size={17}/></button>
          </div>
        ) : (
          <button className="account-btn" onClick={onAuth}><UserRound size={17} /><span>Войти</span></button>
        )}
      </div>
    </header>
  )
}
