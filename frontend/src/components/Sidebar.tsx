import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { api } from '../api'
import { useDownloadState } from '../useDownloadState'
import type { Playlist, User } from '../types'

export function Sidebar({ user, onLogout }: { user: User | null; onLogout: () => void }) {
  const [playlists, setPlaylists] = useState<Playlist[]>([])
  const { active, queued } = useDownloadState()
  const downloadCount = active + queued
  const navigate = useNavigate()
  const location = useLocation()
  const [searchQuery, setSearchQuery] = useState('')

  const handleSearch = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery.trim())}`)
    }
  }, [searchQuery, navigate])

  useEffect(() => {
    api.playlists().then(p => setPlaylists(p || [])).catch(() => {})
  }, [location.pathname])

  return (
    <aside style={{
      width: 280, background: '#000', display: 'flex', flexDirection: 'column',
      flexShrink: 0, borderRight: '1px solid #333', overflow: 'hidden',
    }}>
      <div style={{ padding: '20px 16px 12px', borderBottom: '1px solid #333' }}>
        {user && (
          <div onClick={() => navigate('/downloads')} style={{
            display: 'flex', alignItems: 'center', gap: 12, padding: '8px 12px',
            borderRadius: 6, cursor: 'pointer', transition: 'background 200ms ease',
          }}
          onMouseEnter={e => (e.currentTarget.style.background = '#282828')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
            {user.avatarUrl ? (
              <img src={user.avatarUrl} alt="" style={{ width: 42, height: 42, borderRadius: '50%', objectFit: 'cover' }} />
            ) : (
              <div style={{ width: 42, height: 42, borderRadius: '50%', background: '#282828' }} />
            )}
            <div>
              <div style={{ fontSize: 14, fontWeight: 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user.displayName || user.id}
              </div>
              <div style={{ fontSize: 11, color: '#b3b3b3', textTransform: 'uppercase', letterSpacing: 1 }}>My Library</div>
            </div>
          </div>
        )}
      </div>

      <form onSubmit={handleSearch} style={{ padding: '12px 16px', borderBottom: '1px solid #333' }}>
        <div style={{ position: 'relative' }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#727272" strokeWidth="2"
            style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)' }}>
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <input
            type="text" placeholder="Search songs..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            style={{
              width: '100%', padding: '10px 12px 10px 38px', border: '1px solid transparent',
              borderRadius: 500, background: '#282828', color: '#fff', fontSize: 13,
              outline: 'none', boxSizing: 'border-box', transition: 'border 200ms ease',
            }}
            onFocus={e => (e.target.style.border = '1px solid #fff')}
            onBlur={e => (e.target.style.border = '1px solid transparent')}
          />
        </div>
      </form>

      <div style={{ flex: 1, overflowY: 'auto', padding: '8px 0' }}>
        <div style={{ padding: '8px 20px 4px' }}>
          <h3 style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: 1.5, color: '#b3b3b3' }}>
            Your Playlists
          </h3>
        </div>
        {playlists.map(p => (
          <div key={p.id} onClick={() => navigate(`/playlist/${p.id}`)} style={{
            display: 'flex', alignItems: 'center', gap: 12, padding: '8px 12px',
            margin: '0 8px', borderRadius: 6, cursor: 'pointer',
            background: location.pathname === `/playlist/${p.id}` ? '#333' : 'transparent',
            transition: 'background 200ms ease',
          }}
          onMouseEnter={e => { if (location.pathname !== `/playlist/${p.id}`) e.currentTarget.style.background = '#282828' }}
          onMouseLeave={e => { if (location.pathname !== `/playlist/${p.id}`) e.currentTarget.style.background = 'transparent' }}>
            {p.imageUrl ? (
              <img src={p.imageUrl} alt="" style={{ width: 44, height: 44, borderRadius: 4, objectFit: 'cover' }} />
            ) : (
              <div style={{ width: 44, height: 44, borderRadius: 4, background: '#282828', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18 }}>🎵</div>
            )}
            <div style={{ minWidth: 0, flex: 1 }}>
              <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{p.name}</div>
              <div style={{ fontSize: 11, color: '#727272' }}>{p.trackCount} tracks</div>
            </div>
          </div>
        ))}
      </div>

      <div style={{ padding: 8, borderTop: '1px solid #333' }}>
        <button onClick={() => navigate('/downloads')} style={{
          display: 'flex', alignItems: 'center', gap: 10, width: '100%', padding: '10px 12px',
          background: 'none', border: 'none', color: '#b3b3b3', fontSize: 13, fontWeight: 600,
          borderRadius: 6, cursor: 'pointer', transition: 'all 200ms ease',
        }}
        onMouseEnter={e => { e.currentTarget.style.color = '#fff'; e.currentTarget.style.background = '#282828' }}
        onMouseLeave={e => { e.currentTarget.style.color = '#b3b3b3'; e.currentTarget.style.background = 'none' }}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <span style={{ flex: 1 }}>Downloads</span>
          {downloadCount > 0 && (
            <span style={{
              background: '#1DB954', color: '#000', fontSize: 10, fontWeight: 700,
              borderRadius: 500, minWidth: 18, height: 18, display: 'flex',
              alignItems: 'center', justifyContent: 'center',
            }}>
              {downloadCount > 99 ? '99+' : downloadCount}
            </span>
          )}
        </button>
        <button onClick={onLogout} style={{
          display: 'flex', alignItems: 'center', gap: 10, width: '100%', padding: '10px 12px',
          background: 'none', border: 'none', color: '#727272', fontSize: 12,
          borderRadius: 6, cursor: 'pointer', marginTop: 4,
        }}
        onMouseEnter={e => { e.currentTarget.style.color = '#b3b3b3'; e.currentTarget.style.background = '#282828' }}
        onMouseLeave={e => { e.currentTarget.style.color = '#727272'; e.currentTarget.style.background = 'none' }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
            <polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
          </svg>
          Logout
        </button>
      </div>
    </aside>
  )
}
