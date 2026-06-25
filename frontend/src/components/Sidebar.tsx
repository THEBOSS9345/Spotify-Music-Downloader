import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { api } from '../api'
import { useDownloadState } from '../useDownloadState'
import type { Playlist, User } from '../types'

export function Sidebar({ user, onLogout }: { user: User | null; onLogout: () => void }) {
  const [playlists, setPlaylists] = useState<Playlist[]>([])
  const { active, queued } = useDownloadState()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    api.playlists().then(p => setPlaylists(p || [])).catch(() => {})
  }, [location.pathname])

  const handleSearch = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery.trim())}`)
    }
  }, [searchQuery, navigate])

  const nav = (to: string) => navigate(to)
  const activePath = (path: string) => location.pathname === path ? 'var(--bg-active)' : 'transparent'

  return (
    <aside style={{
      width: 'var(--sidebar)', background: 'var(--bg)', display: 'flex', flexDirection: 'column',
      flexShrink: 0, borderRight: '1px solid var(--border)', overflow: 'hidden',
    }}>
      {/* User */}
      <div style={{ padding: '16px 14px', borderBottom: '1px solid var(--border)' }}>
        {user && (
          <div onClick={() => nav('/downloads')} className="row-hover" style={{
            display: 'flex', alignItems: 'center', gap: 10, padding: '8px 10px', borderRadius: 'var(--radius)',
          }}>
            {user.avatarUrl ? (
              <img src={user.avatarUrl} alt="" style={{ width: 36, height: 36, borderRadius: '50%', objectFit: 'cover' }} />
            ) : (
              <div style={{ width: 36, height: 36, borderRadius: '50%', background: 'var(--bg-hover)' }} />
            )}
            <div style={{ minWidth: 0 }}>
              <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user.displayName || user.id}
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-subdued)', textTransform: 'uppercase', letterSpacing: 1 }}>Library</div>
            </div>
          </div>
        )}
      </div>

      {/* Search */}
      <form onSubmit={handleSearch} style={{ padding: '12px 14px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ position: 'relative' }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-subdued)" strokeWidth="2"
            style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}>
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <input type="text" placeholder="Search songs..." value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)} className="search-input" />
        </div>
      </form>

      {/* Playlists */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '6px 0' }}>
        <div style={{ padding: '6px 20px 4px' }}>
          <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: 1.5, color: 'var(--text-subdued)' }}>
            Playlists
          </div>
        </div>
        {playlists.map(p => (
          <div key={p.id} onClick={() => nav(`/playlist/${p.id}`)} className="row-hover" style={{
            display: 'flex', alignItems: 'center', gap: 10, padding: '6px 10px',
            margin: '0 6px', borderRadius: 'var(--radius-sm)',
            background: activePath(`/playlist/${p.id}`), transition: 'background var(--transition)',
          }}>
            {p.imageUrl ? (
              <img src={p.imageUrl} alt="" style={{ width: 40, height: 40, borderRadius: 4, objectFit: 'cover', flexShrink: 0 }} />
            ) : (
              <div style={{ width: 40, height: 40, borderRadius: 4, background: 'var(--bg-hover)', flexShrink: 0 }} />
            )}
            <div style={{ minWidth: 0, flex: 1 }}>
              <div style={{ fontSize: 13, fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{p.name}</div>
              <div style={{ fontSize: 11, color: 'var(--text-subdued)' }}>{p.trackCount} tracks</div>
            </div>
          </div>
        ))}
      </div>

      {/* Bottom actions */}
      <div style={{ padding: 6, borderTop: '1px solid var(--border)', display: 'flex', flexDirection: 'column', gap: 2 }}>
        <button onClick={() => nav('/downloads')} className="row-hover" style={{
          display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px',
          background: 'none', border: 'none', color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500,
          borderRadius: 'var(--radius-sm)', cursor: 'pointer', transition: 'color var(--transition)',
          width: '100', textAlign: 'left',
        }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <span style={{ flex: 1 }}>Downloads</span>
          {(active + queued) > 0 && (
            <span style={{
              background: 'var(--accent)', color: '#000', fontSize: 10, fontWeight: 700,
              borderRadius: 'var(--radius-full)', minWidth: 18, height: 18, display: 'flex',
              alignItems: 'center', justifyContent: 'center',
            }}>
              {(active + queued) > 99 ? '99+' : active + queued}
            </span>
          )}
        </button>
        <button onClick={onLogout} className="row-hover" style={{
          display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px',
          background: 'none', border: 'none', color: 'var(--text-subdued)', fontSize: 12,
          borderRadius: 'var(--radius-sm)', cursor: 'pointer', width: '100', textAlign: 'left',
        }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
            <polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
          </svg>
          Logout
        </button>
      </div>
    </aside>
  )
}
