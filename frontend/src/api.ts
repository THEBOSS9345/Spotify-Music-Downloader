import type { DownloadResponse, Playlist, Song, Status, User, ImportResult } from './types'

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`)
  return res.json()
}

async function postJSON<T>(url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`)
  return res.json()
}

export const api = {
  status: () => getJSON<Status>('/api/status'),
  user: () => getJSON<User>('/api/user'),
  playlists: () => getJSON<Playlist[]>('/api/playlists'),
  playlistTracks: (id: string) => getJSON<Song[]>(`/api/playlists/${id}/tracks`),
  search: (q: string) => getJSON<Song[]>(`/api/search?q=${encodeURIComponent(q)}`),
  downloads: () => getJSON<DownloadResponse>('/api/downloads'),
  login: () => getJSON<{ url: string }>('/api/login'),
  download: (songs: Song[]) => postJSON<{ batchId: string }>('/api/download', songs),
  retry: (ids: string[]) => postJSON<{ ok: boolean }>('/api/retry', ids),
  refreshPlaylists: () => postJSON<{ ok: boolean }>('/api/playlists/refresh'),
  refreshPlaylistTracks: (id: string) => postJSON<{ ok: boolean }>(`/api/playlists/${id}/tracks/refresh`),
  importPlaylist: (url: string) => postJSON<ImportResult>('/api/playlists/import', { url }),
  logout: () => postJSON<void>('/api/logout'),
}
