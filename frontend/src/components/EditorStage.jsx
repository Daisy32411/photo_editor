import { useLayoutEffect, useRef, useState } from 'react'
import { ArrowDownToLine, ArrowLeftRight, FileImage, ImagePlus, Minus, Plus, RotateCcw, Sparkles, UploadCloud, X } from 'lucide-react'
import { formatFileSize, getPreviewStyle } from '../lib/filters'

export default function EditorStage({ photo, filter, sigma, processedUrl, zoom, setZoom, compare, setCompare, split, setSplit, isDragging, onUpload, onDrop, onDragOver, onDragLeave, onRemovePhoto, onDownloadOriginal, onDownloadResult }) {
  const stageRef = useRef(null)
  const [stageSize, setStageSize] = useState({ width: 0, height: 0, paddingX: 22, paddingY: 22 })

  useLayoutEffect(() => {
    const stage = stageRef.current
    if (!stage) return undefined

    function measure() {
      const image = stage.querySelector('.stage-photo-layer img')
      const style = image ? window.getComputedStyle(image) : null
      setStageSize({
        width: stage.clientWidth,
        height: stage.clientHeight,
        paddingX: style ? parseFloat(style.paddingLeft) || 0 : 22,
        paddingY: style ? parseFloat(style.paddingTop) || 0 : 22,
      })
    }

    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(stage)
    window.addEventListener('resize', measure)
    return () => {
      observer.disconnect()
      window.removeEventListener('resize', measure)
    }
  }, [photo?.id])

  const availableWidth = Math.max(0, stageSize.width - 2 * stageSize.paddingX)
  const availableHeight = Math.max(0, stageSize.height - 2 * stageSize.paddingY)
  const displayedWidth = photo?.width && photo?.height
    ? Math.min(availableWidth, availableHeight * photo.width / photo.height)
    : 0
  const pixelScale = photo?.width ? displayedWidth / photo.width : 0
  const originalStyle = { transform: `scale(${zoom / 100})`, transformOrigin: 'center' }
  const previewStyle = processedUrl ? originalStyle : getPreviewStyle(filter, sigma, zoom, pixelScale)
  const previewUrl = processedUrl ?? photo?.url

  return (
    <section className="editor-col">
      <div className="section-heading">
<div><h2>Рабочая область</h2></div>
      </div>

      <div className="editor-card">
        <div className="editor-toolbar">
          <div className="editor-file-info">
            <span className="toolbar-file-icon"><FileImage size={20}/></span>
            <div><strong title={photo?.name}>{photo ? photo.name : 'Новый проект'}</strong><small>{photo ? `${photo.width} × ${photo.height} px  ·  ${formatFileSize(photo.size)}` : 'Изображение не выбрано'}</small></div>
          </div>
          <div className="toolbar-controls">
            {photo && <button className="toolbar-button subtle" onClick={onRemovePhoto} title="Убрать текущую фотографию"><X size={16}/><span>Закрыть</span></button>}
            <button className="toolbar-button" onClick={onUpload}><ImagePlus size={17}/><span>Открыть PNG</span></button>
          </div>
        </div>

        <div ref={stageRef} className={`image-stage ${photo ? 'has-image' : 'no-image'} ${isDragging ? 'dragging' : ''}`} onDrop={onDrop} onDragOver={onDragOver} onDragLeave={onDragLeave}>
          <div className="grid-overlay" />
          {photo ? (
            <>
              <div className="stage-photo-layer original-layer"><img src={photo.url} alt="Исходная фотография" draggable="false" style={originalStyle} /></div>
              {compare ? (
                <>
                  <div className="stage-photo-layer filtered-layer" style={{ clipPath: `inset(0 ${100 - split}% 0 0)` }}><img src={previewUrl} alt={processedUrl ? "Обработанное изображение" : "Предпросмотр выбранного фильтра"} draggable="false" style={previewStyle} /></div>
                  <div className="compare-divider" style={{ left: `${split}%` }}><span className="compare-handle"><ArrowLeftRight size={17}/></span></div>
                  <input className="compare-range" type="range" min="0" max="100" value={split} onChange={(event) => setSplit(Number(event.target.value))} aria-label="Разделитель сравнения до и после" title="Перемещай курсор или ползунок, чтобы сравнивать" />
                  <span className="stage-caption stage-caption-left">{processedUrl ? "РЕЗУЛЬТАТ" : "ПРЕДПРОСМОТР"}</span><span className="stage-caption stage-caption-right">ОРИГИНАЛ</span>
                </>
              ) : (
                <div className="stage-photo-layer filtered-layer"><img src={previewUrl} alt={processedUrl ? "Обработанное изображение" : "Предпросмотр выбранного фильтра"} draggable="false" style={previewStyle} /></div>
              )}
              {isDragging && <div className="drop-shield"><UploadCloud size={44}/><strong>Отпусти, чтобы заменить изображение</strong></div>}
            </>
          ) : (
            <div className="empty-stage-content">
              <div className="empty-art" aria-hidden="true">
                <div className="art-glow" /><div className="art-card art-card-back"><div className="art-sun"/><div className="art-mountain one"/><div className="art-mountain two"/></div>
                <div className="art-card art-card-front"><Sparkles size={32} strokeWidth={1.5}/><div className="art-curve art-curve-one"/><div className="art-curve art-curve-two"/></div>
                <div className="art-float art-float-a">✦</div><div className="art-float art-float-b">✳</div>
              </div>
              <h3>Начни с чистого кадра</h3>
              <p>Перетащи PNG сюда или выбери файл с компьютера.</p>
              <button className="primary-cta empty-upload" onClick={onUpload}><ImagePlus size={19} /> Выбрать изображение <span>↗</span></button>
              <div className="empty-file-details"><span>PNG до 70 МБ</span></div>
              {isDragging && <div className="drop-shield"><UploadCloud size={44}/><strong>Отпусти файл, чтобы открыть</strong></div>}
            </div>
          )}
        </div>

        <div className="stage-bottom-toolbar">
          <div className="stage-view-left">
            
            {photo && <span className="stage-dimensions">{photo.width} × {photo.height}</span>}
          </div>
          <div className="stage-view-right">
            {photo && <button className={`compare-toggle ${compare ? 'is-on' : ''}`} onClick={() => setCompare(!compare)}><ArrowLeftRight size={16}/> Сравнить</button>}
            <div className="zoom-tool"><button disabled={!photo} aria-label="Уменьшить" onClick={() => setZoom(Math.max(50, zoom - 10))}><Minus size={15}/></button><span>{zoom}%</span><button disabled={!photo} aria-label="Увеличить" onClick={() => setZoom(Math.min(150, zoom + 10))}><Plus size={15}/></button></div>
            <button className="zoom-reset" disabled={!photo || zoom === 100} title="Сбросить масштаб" aria-label="Сбросить масштаб" onClick={() => setZoom(100)}><RotateCcw size={15}/></button>
          </div>
        </div>
      </div>

      {photo && (
        <div className="editor-footnotes">
          <button className="download-original" onClick={onDownloadOriginal}><ArrowDownToLine size={17}/> Скачать оригинал</button>
          {processedUrl && <button className="download-result" onClick={onDownloadResult}><ArrowDownToLine size={17}/> Скачать результат PNG</button>}
        </div>
      )}
    </section>
  )
}
