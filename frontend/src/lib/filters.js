import { Aperture, FlipVertical2, Focus, Waves } from 'lucide-react'

export const FILTERS = [
  {
    id: 'Negative',
    title: 'Негатив',
    english: 'Negative',
    description: 'Инверсия цветов',
    icon: Aperture,
    color: 'blue',
    hasSigma: false,
  },
  {
    id: 'Blur',
    title: 'Размытие',
    english: 'Blur',
    description: 'Мягкий фокус',
    icon: Waves,
    color: 'blue',
    hasSigma: true,
  },
  {
    id: 'Sharpen',
    title: 'Резкость',
    english: 'Sharpen',
    description: 'Больше деталей',
    icon: Focus,
    color: 'orange',
    hasSigma: true,
  },
  {
    id: 'FlipX',
    title: 'Отражение X',
    english: 'FlipX',
    description: 'Перевернуть по вертикали',
    icon: FlipVertical2,
    color: 'mint',
    hasSigma: false,
  },
]

export function getPreviewStyle(filterId, sigma, zoom, pixelScale = 1) {
  const scale = zoom / 100
  const shared = { transform: `scale(${scale})`, transformOrigin: 'center' }

  switch (filterId) {
    case 'Negative':
      return { ...shared, filter: 'invert(1)' }
    case 'Blur':
      return { ...shared, filter: `blur(${(sigma * pixelScale).toFixed(3)}px)` }
    case 'Sharpen':
      return { ...shared, filter: `contrast(${(1 + sigma * 0.045).toFixed(2)}) saturate(${(1 + sigma * 0.02).toFixed(2)})` }
    case 'FlipX':
      return { ...shared, transform: `scale(${scale}) scaleY(-1)` }
    default:
      return shared
  }
}

export function formatFileSize(bytes) {
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`
  return `${(bytes / 1024 / 1024).toFixed(1)} МБ`
}
