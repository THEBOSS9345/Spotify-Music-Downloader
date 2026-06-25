export interface Playlist {
  id: string
  name: string
  description: string
  imageUrl: string
  trackCount: number
  owner: string
}

export interface Song {
  id: string
  title: string
  artist: string
  album: string
  duration: number
  albumArt: string
  trackNum: number
  playlistId: string
}

export interface User {
  id: string
  displayName: string
  avatarUrl: string
  email: string
}

export interface Download {
  id: string
  song: { title: string; artist: string; album: string }
  status: 'pending' | 'searching' | 'downloading' | 'converting' | 'complete' | 'failed'
  progress: number
  error: string
  createdAt?: number
}

export interface DownloadResponse {
  downloads: Download[]
  queue: { song: { title: string; artist: string; album: string } }[]
  active: number
  queued: number
}

export interface ImportResult {
  playlist: Playlist
  songs: Song[]
}

export interface Status {
  authenticated: boolean
}
