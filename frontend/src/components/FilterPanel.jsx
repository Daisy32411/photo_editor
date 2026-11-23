import { Check, ChevronRight, LoaderCircle, Settings2, SlidersHorizontal, WandSparkles, Zap } from 'lucide-react'
import { FILTERS } from '../lib/filters'

export default function FilterPanel({ photo, filter, onChangeFilter, sigma, setSigma, compare, setCompare, onProcess, processing, isCurrentResult }) {
  const selected = FILTERS.find((item) => item.id === filter) ?? FILTERS[0]
  return (
    <aside className="filter-col">
      <div className="inspector-heading"><div><h2>Инструменты <Settings2 size={18}/></h2></div><span className="inspector-count">04</span></div>
      <div className="inspector-body">
        <div className="process-area">
          <button className="process-button" onClick={onProcess} disabled={!photo || Boolean(processing)}>
            {processing ? <LoaderCircle size={18} className="spin-icon" /> : <WandSparkles size={18} />}
            <span>{processing ? processing.label : isCurrentResult ? 'Обработать ещё раз' : 'Применить фильтр'}</span>
          </button>
          {processing && <div className="process-status" role="status">{processing.detail}</div>}
          {isCurrentResult && !processing && <div className="process-success"><Check size={14}/> Результат готов к скачиванию</div>}
        </div>
        <div className="filter-title-line"><span>Фильтры изображения</span><SlidersHorizontal size={16}/></div>
        <p className="filter-intro">Выбери эффект, чтобы изменить фотографию.</p>
        <div className="filter-list" role="radiogroup" aria-label="Выбор фильтра">
          {FILTERS.map((item) => {
            const Icon = item.icon
            const active = item.id === filter
            return (
              <button key={item.id} className={`filter-item ${active ? 'selected' : ''}`} role="radio" aria-checked={active} onClick={() => onChangeFilter(item.id)}>
                <span className={`filter-icon color-${item.color}`}><Icon size={20} strokeWidth={1.9}/></span>
                <span className="filter-text"><strong>{item.title}</strong><small>{item.description}</small></span>
                <span className={`filter-check ${active ? 'checked' : ''}`}>{active ? <Check size={14} strokeWidth={3}/> : <ChevronRight size={15}/>}</span>
              </button>
            )
          })}
        </div>

        <div className="setting-divider"/>
        <div className="settings-block">
          <div className="settings-title"><span>Параметры эффекта</span><span className="settings-tag">{selected.english}</span></div>
          {selected.hasSigma ? (
            <div className="sigma-card">
              <div className="sigma-head"><div><strong>Интенсивность</strong><small>Сила эффекта</small></div><div className="sigma-value">{sigma.toFixed(1)}</div></div>
              <input type="range" className="sigma-slider" min="0.5" max="20" step="0.5" value={sigma} onChange={(event) => setSigma(Number(event.target.value))} aria-label="Значение sigma" style={{ '--progress': `${((sigma - 0.5) / 19.5) * 100}%` }}/>
              <div className="range-ends"><span>Легче</span><span>Сильнее</span></div>
            </div>
          ) : (
            <div className="no-settings"><span className="no-settings-icon"><Zap size={17}/></span><div><strong>Без дополнительных настроек</strong></div></div>
          )}
        </div>

        <div className="setting-divider"/>
        <div className="inspector-toggle-line"><div><strong>Сравнение до / после</strong><small>Перетаскивай разделитель на фото</small></div><button className={`switch ${compare ? 'on' : ''}`} onClick={() => setCompare(!compare)} aria-pressed={compare} aria-label="Включить сравнение"><span /></button></div>


      </div>
    </aside>
  )
}
