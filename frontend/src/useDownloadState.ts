import { useEffect, useState } from 'react'
import type { Download } from './types'

interface QueueItem {
  song: { title: string; artist: string; album: string }
}

interface DownloadState {
  downloads: Download[]
  queue: QueueItem[]
  active: number
  queued: number
}

let state: DownloadState = { downloads: [], queue: [], active: 0, queued: 0 }
let listeners = new Set<() => void>()
let es: EventSource | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

function connect() {
  if (es) return
  es = new EventSource('/api/events')

  es.addEventListener('download-state', (e) => {
    try {
      state = JSON.parse(e.data)
      listeners.forEach(fn => fn())
    } catch { /* ignore */ }
  })

  es.onerror = () => {
    es?.close()
    es = null
    reconnectTimer = setTimeout(connect, 2000)
  }
}

function disconnect() {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  es?.close()
  es = null
}

export function getDownloadState(): DownloadState {
  return state
}

export function useDownloadState(): DownloadState {
  const [s, setS] = useState<DownloadState>(state)

  useEffect(() => {
    const update = () => setS({ ...state })
    listeners.add(update)
    connect()
    return () => {
      listeners.delete(update)
      if (listeners.size === 0) disconnect()
    }
  }, [])

  return s
}
