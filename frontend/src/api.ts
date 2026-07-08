import type { DownloadResponse, Playlist, Song, Status, User, ImportResult } from './types'

async function getJSON<T>(url: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal })
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`)
  return res.json()
}

async function postJSON<T>(url: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
    signal,
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`)
  return res.json()
}

export const api = {
  status: (signal?: AbortSignal) => getJSON<Status>('/api/status', signal),
  user: (signal?: AbortSignal) => getJSON<User>('/api/user', signal),
  playlists: (signal?: AbortSignal) => getJSON<Playlist[]>('/api/playlists', signal),
  playlistTracks: (id: string, signal?: AbortSignal) => getJSON<Song[]>(`/api/playlists/${id}/tracks`, signal),
  search: (q: string, signal?: AbortSignal) => getJSON<Song[]>(`/api/search?q=${encodeURIComponent(q)}`, signal),
  downloads: (signal?: AbortSignal) => getJSON<DownloadResponse>('/api/downloads', signal),
  login: (signal?: AbortSignal) => getJSON<{ url: string }>('/api/login', signal),
  download: (songs: Song[], signal?: AbortSignal) => postJSON<{ batchId: string }>('/api/download', songs, signal),
  retry: (ids: string[], signal?: AbortSignal) => postJSON<{ ok: boolean }>('/api/retry', ids, signal),
  refreshPlaylists: (signal?: AbortSignal) => postJSON<{ ok: boolean }>('/api/playlists/refresh', undefined, signal),
  refreshPlaylistTracks: (id: string, signal?: AbortSignal) => postJSON<{ ok: boolean }>(`/api/playlists/${id}/tracks/refresh`, undefined, signal),
  importPlaylist: (url: string, signal?: AbortSignal) => postJSON<ImportResult>('/api/playlists/import', { url }, signal),
  logout: (signal?: AbortSignal) => postJSON<void>('/api/logout', undefined, signal),
}
