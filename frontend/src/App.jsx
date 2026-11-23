import { useEffect, useRef, useState } from 'react'
import { AlertCircle, X } from 'lucide-react'
import Sidebar from './components/Sidebar.jsx'
import Topbar from './components/Topbar.jsx'
import EditorStage from './components/EditorStage.jsx'
import FilterPanel from './components/FilterPanel.jsx'
import HistoryView from './components/HistoryView.jsx'
import { AuthDialog } from './components/Dialogs.jsx'
import { createTask, getResult, getStatus } from './api/client.js'

const MAX_FILE_SIZE = 70 * 1024 * 1024
const TASK_TIMEOUT_MS = 180_000
const SESSION_KEY = 'photolab_session'

function loadSession() {
  try {
    const data = JSON.parse(sessionStorage.getItem(SESSION_KEY))
    return data?.token && data?.username ? data : null
  } catch {
    return null
  }
}

function getImageSize(objectUrl) {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve({ width: img.naturalWidth, height: img.naturalHeight })
    img.onerror = () => reject(new Error('Не удалось прочитать изображение. Убедись, что это настоящий PNG.'))
    img.src = objectUrl
  })
}

function wait(ms, signal) {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException('Отменено', 'AbortError'))
      return
    }
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    function onAbort() {
      clearTimeout(timer)
      reject(new DOMException('Отменено', 'AbortError'))
    }
    signal.addEventListener('abort', onAbort, { once: true })
  })
}

export default function App() {
  const [page, setPage] = useState('editor')
  const [photos, setPhotos] = useState([])
  const [selectedPhotoId, setSelectedPhotoId] = useState(null)
  const [filter, setFilter] = useState('Negative')
  const [sigma, setSigma] = useState(2.5)
  const [zoom, setZoom] = useState(100)
  const [compare, setCompare] = useState(true)
  const [split, setSplit] = useState(50)
  const [isDragging, setIsDragging] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [dialog, setDialog] = useState(null)
  const [toast, setToast] = useState(null)
  const [session, setSession] = useState(loadSession)
  const [processing, setProcessing] = useState(null)

  const fileInputRef = useRef(null)
  const urlsRef = useRef(new Set())
  const processingRef = useRef(false)
  const processingControllerRef = useRef(null)

  const selectedPhoto = photos.find((photo) => photo.id === selectedPhotoId) ?? null
  const currentResult = selectedPhoto?.result?.filter === filter &&
    (filter !== 'Blur' && filter !== 'Sharpen' || selectedPhoto.result.sigma === sigma)
    ? selectedPhoto.result : null
  const processingCurrent = processing?.photoId === selectedPhotoId ? processing : null

  useEffect(() => {
    const urls = urlsRef.current
    return () => {
      processingControllerRef.current?.abort()
      for (const url of urls) URL.revokeObjectURL(url)
      urls.clear()
    }
  }, [])

  useEffect(() => {
    if (!toast) return undefined
    const timer = window.setTimeout(() => setToast(null), 5000)
    return () => window.clearTimeout(timer)
  }, [toast])

  useEffect(() => {
    function handleKeys(event) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'o') {
        event.preventDefault()
        fileInputRef.current?.click()
      }
      if (event.key === 'Escape') setDialog(null)
    }
    window.addEventListener('keydown', handleKeys)
    return () => window.removeEventListener('keydown', handleKeys)
  }, [])

  const openPicker = () => fileInputRef.current?.click()

  async function addPhoto(file) {
    if (!file) return
    setToast(null)
    if (file.type !== 'image/png' && !file.name.toLowerCase().endsWith('.png')) {
      setToast({ variant: 'error', title: 'Нужен PNG-файл', detail: 'Поддерживаются изображения в формате PNG.' })
      return
    }
    if (file.size > MAX_FILE_SIZE) {
      setToast({ variant: 'error', title: 'Файл слишком большой', detail: 'Выбери PNG до 70 МиБ.' })
      return
    }

    const url = URL.createObjectURL(file)
    setIsUploading(true)
    try {
      const { width, height } = await getImageSize(url)
      const photo = {
        id: globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}`,
        file,
        name: file.name,
        size: file.size,
        width,
        height,
        url,
        result: null,
        createdAt: new Date(),
      }
      urlsRef.current.add(url)
      setPhotos((current) => [photo, ...current])
      setSelectedPhotoId(photo.id)
      setPage('editor')
      setZoom(100)
      setSplit(50)
      setFilter('Negative')
    } catch (error) {
      URL.revokeObjectURL(url)
      setToast({ variant: 'error', title: 'Не получилось открыть файл', detail: error.message })
    } finally {
      setIsUploading(false)
    }
  }

  function handleDrop(event) {
    event.preventDefault()
    setIsDragging(false)
    const file = event.dataTransfer.files?.[0]
    if (file) void addPhoto(file)
  }

  function handleDragOver(event) {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'copy'
    setIsDragging(true)
  }

  function handleDragLeave(event) {
    event.preventDefault()
    if (!event.currentTarget.contains(event.relatedTarget)) setIsDragging(false)
  }

  function selectPhoto(id) {
    const photo = photos.find((item) => item.id === id)
    setSelectedPhotoId(id)
    setPage('editor')
    setSplit(50)
    setZoom(100)
    setFilter(photo?.result?.filter ?? 'Negative')
    setSigma(photo?.result?.sigma ?? 2.5)
  }

  function clearHistory() {
    photos.forEach((photo) => {
      for (const url of [photo.url, photo.result?.url]) {
        if (url) {
          URL.revokeObjectURL(url)
          urlsRef.current.delete(url)
        }
      }
    })
    setPhotos([])
    setSelectedPhotoId(null)
  }

  function download(url, filename) {
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
  }

  function downloadOriginal() {
    if (selectedPhoto) download(selectedPhoto.url, selectedPhoto.name)
  }

  function downloadResult() {
    if (!selectedPhoto || !currentResult) return
    const baseName = selectedPhoto.name.replace(/\.png$/i, '')
    download(currentResult.url, `${baseName}-${currentResult.filter.toLowerCase()}.png`)
  }

  function handleAuthSuccess(userSession) {
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(userSession))
    setSession(userSession)
    setDialog(null)
  }

  function logout() {
    processingControllerRef.current?.abort()
    sessionStorage.removeItem(SESSION_KEY)
    setSession(null)
  }

  async function processPhoto() {
    if (!selectedPhoto || processingRef.current) return
    if (!session) {
      setDialog('auth')
      return
    }

    setToast(null)
    const photo = selectedPhoto
    const chosenFilter = filter
    const chosenSigma = sigma
    const token = session.token
    const controller = new AbortController()
    processingControllerRef.current = controller
    processingRef.current = true
    setProcessing({ photoId: photo.id, label: 'Отправка…', detail: 'Подготавливаем изображение' })

    try {
      const task = await createTask(photo.file, chosenFilter, chosenSigma, token)
      if (!task.task_id) throw new Error('Не удалось начать обработку. Попробуй ещё раз.')
      setProcessing({ photoId: photo.id, label: 'Обработка…', detail: 'Применяем выбранный эффект' })

      const deadline = Date.now() + TASK_TIMEOUT_MS
      let ready = false
      while (Date.now() < deadline) {
        await wait(900, controller.signal)
        const { status } = await getStatus(task.task_id, token, controller.signal)
        if (status === 'ready') {
          ready = true
          break
        }
        if (status === 'failed') throw new Error('Не удалось обработать изображение. Попробуй другой фильтр.')
        if (status !== 'in_progress') throw new Error('Не удалось завершить обработку. Попробуй ещё раз.')
      }

      if (!ready) throw new Error('Обработка занимает больше времени, чем обычно. Попробуй ещё раз.')
      setProcessing({ photoId: photo.id, label: 'Загрузка…', detail: 'Подготавливаем результат' })

      const blob = await getResult(task.task_id, token, controller.signal)
      if (!blob.size) throw new Error('Не удалось получить готовое изображение.')
      const url = URL.createObjectURL(blob)
      urlsRef.current.add(url)

      setPhotos((current) => current.map((item) => {
        if (item.id !== photo.id) return item
        if (item.result?.url) {
          const oldUrl = item.result.url
          window.setTimeout(() => {
            URL.revokeObjectURL(oldUrl)
            urlsRef.current.delete(oldUrl)
          }, 1500)
        }
        return { ...item, result: { url, filter: chosenFilter, sigma: chosenSigma, taskId: task.task_id } }
      }))
      setSplit(50)
      setCompare(true)
    } catch (error) {
      if (error.name === 'AbortError') return
      if (error.status === 401) {
        sessionStorage.removeItem(SESSION_KEY)
        setSession(null)
        setDialog('auth')
        setToast({ variant: 'error', title: 'Войди снова', detail: 'Пожалуйста, войди снова.' })
      } else if (error.status === 413) {
        setToast({ variant: 'error', title: 'Файл слишком большой', detail: 'Попробуй выбрать PNG меньшего размера.' })
      } else {
        setToast({ variant: 'error', title: 'Обработка не удалась', detail: error.message || 'Неизвестная ошибка.' })
      }
    } finally {
      processingRef.current = false
      if (processingControllerRef.current === controller) processingControllerRef.current = null
      setProcessing(null)
    }
  }

  return (
    <div className="app-shell">
      <Sidebar
        page={page}
        setPage={setPage}
        photos={photos}
        selectedPhotoId={selectedPhotoId}
        onSelectPhoto={selectPhoto}
        onUpload={openPicker}
        onClearHistory={clearHistory}
      />
      <main className="main-shell">
        <Topbar page={page} session={session} onAuth={() => setDialog('auth')} onLogout={logout} />
        {toast?.variant === 'error' && (
          <div className="editor-notice" role="alert">
            <AlertCircle size={18} />
            <div><strong>{toast.title}</strong><span>{toast.detail}</span></div>
            <button onClick={() => setToast(null)} aria-label="Скрыть сообщение"><X size={16} /></button>
          </div>
        )}
        {page === 'history' ? (
          <HistoryView photos={photos} onOpenPhoto={selectPhoto} onUpload={openPicker} onClearHistory={clearHistory} />
        ) : (
          <div className="workspace-grid">
            <EditorStage
              photo={selectedPhoto}
              filter={filter}
              sigma={sigma}
              processedUrl={currentResult?.url ?? null}
              zoom={zoom}
              setZoom={setZoom}
              compare={compare}
              setCompare={setCompare}
              split={split}
              setSplit={setSplit}
              isDragging={isDragging}
              onUpload={openPicker}
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onRemovePhoto={() => setSelectedPhotoId(null)}
              onDownloadOriginal={downloadOriginal}
              onDownloadResult={downloadResult}
            />
            <FilterPanel
              photo={selectedPhoto}
              filter={filter}
              onChangeFilter={setFilter}
              sigma={sigma}
              setSigma={setSigma}
              compare={compare}
              setCompare={setCompare}
              onProcess={processPhoto}
              processing={processingCurrent ?? (processing ? { label: 'Другая задача…', detail: 'Дождись завершения текущей обработки.' } : null)}
              isCurrentResult={Boolean(currentResult)}
            />
          </div>
        )}
      </main>

      <input
        ref={fileInputRef}
        className="hidden-file-input"
        type="file"
        accept=".png,image/png"
        onChange={(event) => {
          const file = event.target.files?.[0]
          if (file) void addPhoto(file)
          event.target.value = ''
        }}
      />

      {isUploading && <div className="upload-loading"><div className="loading-spinner"/><span>Открываем изображение…</span></div>}
      {dialog === 'auth' && <AuthDialog onClose={() => setDialog(null)} onSuccess={handleAuthSuccess} />}
    </div>
  )
}
