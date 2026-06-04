import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import type { Playlist } from '../types'

export function Playlists() {
  const [playlists, setPlaylists] = useState<Playlist[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    api.playlists().then(p => {
      setPlaylists(p || [])
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="loading-state" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', color: '#727272' }}>
        <div className="spinner" />
        <p style={{ marginTop: 16 }}>Loading playlists...</p>
      </div>
    )
  }

  return (
    <div style={{ padding: 32 }}>
      <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 24 }}>Your Library</h1>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 20 }}>
        {playlists.map(p => (
          <div key={p.id} onClick={() => navigate(`/playlist/${p.id}`)}
            style={{
              background: '#1e1e1e',
              borderRadius: 10,
              padding: 16,
              cursor: 'pointer',
              transition: 'background 200ms ease',
            }}
            onMouseEnter={e => (e.currentTarget.style.background = '#282828')}
            onMouseLeave={e => (e.currentTarget.style.background = '#1e1e1e')}>
            {p.imageUrl ? (
              <img src={p.imageUrl} alt={p.name} style={{ width: '100%', aspectRatio: '1', borderRadius: 6, objectFit: 'cover', marginBottom: 12 }} />
            ) : (
              <div style={{ width: '100%', aspectRatio: '1', borderRadius: 6, background: '#282828', marginBottom: 12, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 32 }}>🎵</div>
            )}
            <h3 style={{ fontSize: 14, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{p.name}</h3>
            <p style={{ fontSize: 12, color: '#727272', marginTop: 4 }}>{p.trackCount} tracks</p>
          </div>
        ))}
      </div>
    </div>
  )
}
