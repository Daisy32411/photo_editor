import { Aperture, ArrowUpRight, Clock3, FolderOpen, ImagePlus, Trash2, WandSparkles } from 'lucide-react'
import { formatFileSize } from '../lib/filters'

export default function Sidebar({ page, setPage, photos, selectedPhotoId, onSelectPhoto, onUpload, onClearHistory }) {
  return (
    <aside className="sidebar">
      <button className="brand" onClick={() => setPage('editor')} aria-label="Открыть редактор PhotoLab">
        <span className="brand-symbol"><Aperture size={23} strokeWidth={2.2} /></span>
        <span className="brand-type"><strong>photo<span>lab</span></strong></span>
      </button>

      <div className="sidebar-block">
        <p className="sidebar-label">РАБОЧЕЕ ПРОСТРАНСТВО</p>
        <nav className="primary-nav" aria-label="Основная навигация">
          <button className={`nav-link ${page === 'editor' ? 'active' : ''}`} onClick={() => setPage('editor')}>
            <WandSparkles size={19} /><span>Редактор</span>{page === 'editor' && <span className="nav-active-dot" />}
          </button>
          <button className={`nav-link ${page === 'history' ? 'active' : ''}`} onClick={() => setPage('history')}>
            <Clock3 size={19} /><span>История</span><span className="nav-count">{photos.length}</span>
          </button>
        </nav>
      </div>

      <div className="sidebar-block sidebar-library">
        <div className="sidebar-label-row">
          <p className="sidebar-label">НЕДАВНИЕ ФАЙЛЫ</p>
          {photos.length > 0 && <button className="small-icon-button" aria-label="Очистить историю" title="Очистить историю" onClick={onClearHistory}><Trash2 size={14}/></button>}
        </div>
        {photos.length === 0 ? (
          <div className="library-empty"><FolderOpen size={20} strokeWidth={1.5} /><span>Здесь появятся<br />твои фотографии</span></div>
        ) : (
          <div className="file-list">
            {photos.slice(0, 5).map((photo) => (
              <button key={photo.id} className={`recent-file ${photo.id === selectedPhotoId && page === 'editor' ? 'selected' : ''}`} onClick={() => onSelectPhoto(photo.id)}>
                <img src={photo.url} alt="" />
                <span className="recent-meta"><strong title={photo.name}>{photo.name}</strong><small>{formatFileSize(photo.size)}</small></span>
                <ArrowUpRight size={13} className="recent-arrow" />
              </button>
            ))}
          </div>
        )}
        <button className="sidebar-upload" onClick={onUpload}><ImagePlus size={17} /> Добавить изображение</button>
      </div>

    </aside>
  )
}
