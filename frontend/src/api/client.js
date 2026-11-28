const API_PREFIX = '/api'

export class ApiError extends Error {
  constructor(message, status = 0) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request(path, { token, signal, ...options } = {}) {
  let response
  try {
    response = await fetch(`${API_PREFIX}${path}`, {
      ...options,
      signal,
      headers: {
        ...(options.body ? { 'Content-Type': 'application/json' } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options.headers,
      },
    })
  } catch (error) {
    if (error.name === 'AbortError') throw error
    throw new ApiError('Сейчас не удаётся выполнить запрос. Проверь подключение и попробуй ещё раз.')
  }

  if (!response.ok) {
    const messages = {
      400: 'Проверь введённые данные и попробуй ещё раз.',
      401: 'Нужно войти в аккаунт.',
      404: 'Запрошенный результат больше недоступен.',
      409: 'Изображение ещё обрабатывается.',
      413: 'Изображение слишком большое. Выбери файл поменьше.',
      503: 'Обработка временно недоступна. Попробуй немного позже.',
    }
    throw new ApiError(messages[response.status] || 'Что-то пошло не так. Попробуй ещё раз.', response.status)
  }

  return response
}

export async function register(username, password) {
  await request('/register', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

export async function login(username, password) {
  const response = await request('/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  return response.json()
}

function fileToBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const data = String(reader.result ?? '')
      const comma = data.indexOf(',')
      if (comma < 0) {
        reject(new Error('Не удалось прочитать PNG как Base64.'))
        return
      }
      resolve(data.slice(comma + 1))
    }
    reader.onerror = () => reject(new Error('Не удалось прочитать файл.'))
    reader.readAsDataURL(file)
  })
}

export async function createTask(file, filter, sigma, token) {
  const image = await fileToBase64(file)
  const filterData = { name: filter }
  if (filter === 'Blur' || filter === 'Sharpen') {
    filterData.parameters = { sigma }
  }

  const response = await request('/task', {
    method: 'POST',
    token,
    body: JSON.stringify({ image, filter: filterData }),
  })
  return response.json()
}

export async function getStatus(taskId, token, signal) {
  const response = await request(`/status/${encodeURIComponent(taskId)}`, { token, signal })
  return response.json()
}

export async function getResult(taskId, token, signal) {
  const response = await request(`/result/${encodeURIComponent(taskId)}`, { token, signal })
  const contentType = response.headers.get('Content-Type') || ''
  if (!contentType.startsWith('image/png')) {
    throw new ApiError('Не удалось открыть готовое изображение.')
  }
  return response.blob()
}
