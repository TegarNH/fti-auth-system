const API_BASE = 'http://localhost:8080'

export class ApiFetchError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiFetchError'
    this.status = status
  }
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE}${path}`
  const headers = new Headers(options.headers || {})

  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(url, {
    ...options,
    headers,
    credentials: 'include',
  })

  // Parse body
  let data: any = null
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    try {
      data = await response.json()
    } catch {
      data = null
    }
  }

  if (!response.ok) {
    const errorMsg = data?.error || data?.message || `Terjadi kesalahan (HTTP ${response.status})`
    throw new ApiFetchError(errorMsg, response.status)
  }

  return data as T
}
