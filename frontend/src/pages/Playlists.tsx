import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { toast } from '../components/Toast'
import type { Playlist } from '../types'

export function Playlists() {
  const [playlists, setPlaylists] = useState<Playlist[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [search, setSearch] = useState('')
  const [importUrl, setImportUrl] = useState('')
  const [importing, setImporting] = useState(false)
  const q = search.toLowerCase()
  const filtered = playlists.filter(p => !q || p.name.toLowerCase().includes(q))
  const navigate = useNavigate()

  useEffect(() => {
    api.playlists().then(p => {
      setPlaylists(p || [])
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  const handleRefresh = useCallback(async () => {
    setRefreshing(true)
    try {
      await api.refreshPlaylists()
      const p = await api.playlists()
      setPlaylists(p || [])
      toast('Playlists refreshed', 'success')
    } catch {
      toast('Refresh failed', 'error')
    } finally {
      setRefreshing(false)
    }
  }, [])

  const handleImport = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!importUrl.trim()) return
    setImporting(true)
    try {
      const result = await api.importPlaylist(importUrl.trim())
      navigate(`/playlist/${result.playlist.id}`)
      toast(`Imported: ${result.playlist.name}`, 'success')
    } catch {
      toast('Import failed', 'error')
    } finally {
      setImporting(false)
      setImportUrl('')
    }
  }, [importUrl, navigate])

  if (loading) {
    return (
      <div className="fade-in" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', color: 'var(--text-subdued)' }}>
        <div className="spinner" />
        <p style={{ marginTop: 14, fontSize: 14 }}>Loading playlists...</p>
      </div>
    )
  }

  return (
    <div className="fade-in page" style={{ padding: 28, maxWidth: 1100 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 24 }}>
        <h1 style={{ fontSize: 26, fontWeight: 700, letterSpacing: '-0.02em' }}>Your Library</h1>
        <button onClick={handleRefresh} disabled={refreshing} className="pill pill-outline" style={{ marginLeft: 'auto' }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ marginRight: 6, animation: refreshing ? 'spin 1s linear infinite' : 'none' }}>
            <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
          </svg>
          {refreshing ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>
      <form onSubmit={handleImport} style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
        <input type="text" placeholder="Paste a Spotify playlist link or ID..." value={importUrl}
          onChange={e => setImportUrl(e.target.value)} className="search-input" style={{ flex: 1 }} />
        <button type="submit" disabled={importing || !importUrl.trim()} className="pill pill-outline" style={{ flexShrink: 0 }}>
          {importing ? 'Importing...' : 'Import'}
        </button>
      </form>
      <input type="text" placeholder="Filter playlists..." value={search} onChange={e => setSearch(e.target.value)}
        className="search-input" style={{ marginBottom: 20 }} />
      {!filtered.length && !loading ? (
        <p style={{ color: 'var(--text-subdued)', textAlign: 'center', padding: 40, fontSize: 14 }}>
          {search ? 'No matching playlists' : 'No playlists'}
        </p>
      ) : (
        <div className="stagger" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 16 }}>
        {filtered.map(p => (
          <div key={p.id} onClick={() => navigate(`/playlist/${p.id}`)} className="card" style={{ padding: 14 }}>
            {p.imageUrl ? (
              <img src={p.imageUrl} alt={p.name} style={{ width: '100%', aspectRatio: '1', borderRadius: 4, objectFit: 'cover', marginBottom: 10 }} />
            ) : (
              <div style={{ width: '100%', aspectRatio: '1', borderRadius: 4, background: 'var(--bg-hover)', marginBottom: 10, display: 'flex', alignItems: 'center', justifyContent: 'center' }} />
            )}
            <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{p.name}</div>
            <div style={{ fontSize: 11, color: 'var(--text-subdued)', marginTop: 2 }}>{p.trackCount} tracks</div>
          </div>
        ))}
      </div>
     )
    }
    </div>
  )
}
