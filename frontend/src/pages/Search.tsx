import { useEffect, useState, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { toast } from '../components/Toast'
import type { Song } from '../types'

const fmt = (s: number) => `${Math.floor(s / 60)}:${(s % 60).toString().padStart(2, '0')}`

export function Search() {
  const [params] = useSearchParams()
  const query = params.get('q') || ''
  const [results, setResults] = useState<Song[]>([])
  const [loading, setLoading] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [downloading, setDownloading] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (!query) return
    const ac = new AbortController()
    setLoading(true)
    api.search(query, ac.signal).then(s => {
      setResults(s || [])
      setLoading(false)
    }).catch(() => {})
    return () => ac.abort()
  }, [query])

  const toggle = useCallback((id: string) => setSelected(p => {
    const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n
  }), [])

  const startDownload = useCallback(async (list: Song[]) => {
    setDownloading(new Set(list.map(s => s.id)))
    try {
      await api.download(list)
      toast(`Downloading ${list.length} song${list.length > 1 ? 's' : ''}`, 'success')
    } catch { toast('Download failed', 'error') }
  }, [])

  const allSelected = results.length > 0 && selected.size === results.length

  return (
    <div className="fade-in page" style={{ padding: 28 }}>
      <h1 style={{ fontSize: 24, fontWeight: 700, marginBottom: 20, letterSpacing: '-0.02em' }}>
        {query ? `"${query}"` : 'Search'}
      </h1>

      {results.length > 0 && (
        <div style={{ display: 'flex', gap: 10, marginBottom: 20 }}>
          <button onClick={() => startDownload(results)} className="pill pill-primary">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Download All
          </button>
          <button onClick={() => startDownload(results.filter(s => selected.has(s.id)))}
            disabled={selected.size === 0} className="pill pill-outline">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Selected ({selected.size})
          </button>
        </div>
      )}

      {loading ? (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '40vh', color: 'var(--text-subdued)' }}>
          <div className="spinner" />
          <p style={{ marginTop: 14, fontSize: 14 }}>Searching...</p>
        </div>
      ) : !results.length ? (
        <p className="fade-in" style={{ color: 'var(--text-subdued)', textAlign: 'center', padding: 40, fontSize: 14 }}>
          {query ? 'No results' : 'Enter a search query'}
        </p>
      ) : (
        <div>
          <div style={{
            display: 'grid', gridTemplateColumns: '36px 1fr 1fr 1fr 64px 44px',
            padding: '6px 14px', fontSize: 10, fontWeight: 600, textTransform: 'uppercase',
            letterSpacing: 1.2, color: 'var(--text-subdued)', borderBottom: '1px solid var(--border)',
          }}>
            <button onClick={() => allSelected ? setSelected(new Set()) : setSelected(new Set(results.map(s => s.id)))}
              className="row-hover" style={{
              background: 'none', border: 'none', color: allSelected ? 'var(--accent)' : 'var(--text-subdued)',
              cursor: 'pointer', padding: 0, display: 'flex', alignItems: 'center', fontSize: 10, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 1.2,
            }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                {allSelected
                  ? <><rect x="3" y="3" width="18" height="18" rx="3"/><path d="M9 12l2 2 4-4"/></>
                  : <rect x="3" y="3" width="18" height="18" rx="3" fill="none"/>
                }
              </svg>
            </button>
            <div>Title</div>
            <div>Artist</div>
            <div className="hide-mobile">Album</div>
            <div className="hide-mobile" style={{ textAlign: 'right' }}>Time</div>
            <div />
          </div>
          {results.map((s, i) => {
            const isSelected = selected.has(s.id)
            return (
              <div key={s.id} onClick={() => toggle(s.id)} className="row-hover" style={{
                display: 'grid', gridTemplateColumns: '36px 1fr 1fr 1fr 64px 44px',
                padding: '6px 14px', alignItems: 'center', fontSize: 13, borderRadius: 4,
                opacity: downloading.has(s.id) ? 0.5 : 1,
                background: isSelected ? 'rgba(29,185,84,0.08)' : 'transparent',
                transition: 'background var(--transition), opacity var(--transition)',
                cursor: 'pointer',
                animation: 'slideUp 300ms ease both',
                animationDelay: `${Math.min(i * 30, 400)}ms`,
              }}>
                <div style={{ position: 'relative', width: 32, height: 32 }}>
                  {s.albumArt ? (
                    <img src={s.albumArt} alt="" style={{ width: 32, height: 32, borderRadius: 2, objectFit: 'cover' }} />
                  ) : (
                    <div style={{ width: 32, height: 32, borderRadius: 2, background: 'var(--bg-hover)' }} />
                  )}
                  {isSelected && (
                    <div style={{
                      position: 'absolute', inset: 0, borderRadius: 2,
                      background: 'rgba(0,0,0,0.55)',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                    }}>
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="var(--accent)" stroke="#000" strokeWidth="1">
                        <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                      </svg>
                    </div>
                  )}
                </div>
                <div style={{ fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', paddingLeft: 6 }}>
                  {s.title}
                </div>
                <div style={{ color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.artist}</div>
                <div className="hide-mobile" style={{ color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.album}</div>
                <div className="hide-mobile" style={{ color: 'var(--text-secondary)', fontSize: 12, textAlign: 'right' }}>{fmt(s.duration)}</div>
                <div style={{ display: 'flex', justifyContent: 'center' }}>
                  <button onClick={e => { e.stopPropagation(); startDownload([s]) }} disabled={downloading.has(s.id)} className="row-hover" style={{
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    width: 28, height: 28, padding: 0, border: 'none', background: 'none',
                    color: 'var(--text-subdued)', borderRadius: '50%', cursor: 'pointer',
                    transition: 'color var(--transition), background var(--transition)',
                  }}
                  onMouseEnter={e => { e.currentTarget.style.color = 'var(--text)'; e.currentTarget.style.background = 'var(--bg-hover)' }}
                  onMouseLeave={e => { e.currentTarget.style.color = 'var(--text-subdued)'; e.currentTarget.style.background = 'none' }}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                      <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
                    </svg>
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
