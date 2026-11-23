import { ArrowRight, Clock3, FolderOpen, ImagePlus, Trash2 } from 'lucide-react'
import { formatFileSize } from '../lib/filters'

export default function HistoryView({ photos, onOpenPhoto, onUpload, onClearHistory }) {
  return (
    <section className="history-page">
      <div className="history-header"><div><h2>Недавние фотографии</h2></div><div className="history-actions">{photos.length > 0 && <button className="outline-btn" onClick={onClearHistory}><Trash2 size={16}/> Очистить</button>}<button className="primary-cta history-upload" onClick={onUpload}><ImagePlus size={18}/> Загрузить PNG</button></div></div>
      {photos.length > 0 ? (
        <div className="history-grid">{photos.map((photo) => (
          <button className="history-card" key={photo.id} onClick={() => onOpenPhoto(photo.id)}>
            <div className="history-cover"><img src={photo.url} alt={photo.name} /><span className="history-cover-icon"><ArrowRight size={20}/></span></div>
            <div className="history-card-body"><strong title={photo.name}>{photo.name}</strong><small>{photo.width} × {photo.height} px · {formatFileSize(photo.size)}</small><span><Clock3 size={13}/> {new Intl.DateTimeFormat('ru-RU', { hour: '2-digit', minute: '2-digit' }).format(photo.createdAt)}</span></div>
          </button>
        ))}</div>
      ) : (
        <div className="history-empty"><span><FolderOpen size={32}/></span><h3>Здесь ещё пусто</h3><p>Добавь первую фотографию,<br />и она появится здесь.</p><button className="primary-cta" onClick={onUpload}><ImagePlus size={17}/> Загрузить PNG</button></div>
      )}
    </section>
  )
}
