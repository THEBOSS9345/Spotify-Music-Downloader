import { useEffect, useState, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { toast } from '../components/Toast'
import type { Song } from '../types'

const formatDuration = (s: number) => `${Math.floor(s / 60)}:${(s % 60).toString().padStart(2, '0')}`

export function Search() {
  const [params] = useSearchParams()
  const query = params.get('q') || ''
  const [results, setResults] = useState<Song[]>([])
  const [loading, setLoading] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [downloading, setDownloading] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (!query) return
    setLoading(true)
    api.search(query).then(s => {
      setResults(s || [])
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [query])

  const toggle = useCallback((id: string) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const startDownload = useCallback(async (songsToDownload: Song[]) => {
    setDownloading(new Set(songsToDownload.map(s => s.id)))
    try {
      await api.download(songsToDownload)
      toast(`Downloading ${songsToDownload.length} song${songsToDownload.length > 1 ? 's' : ''}`, 'success')
    } catch {
      toast('Download failed', 'error')
    }
  }, [])

  return (
    <div style={{ padding: 32 }}>
      <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 24 }}>Search: "{query}"</h1>

      {results.length > 0 && (
        <div style={{ display: 'flex', gap: 12, marginBottom: 24 }}>
          <button onClick={() => startDownload(results)} style={{
            display: 'inline-flex', alignItems: 'center', gap: 8, padding: '12px 24px',
            background: '#1DB954', color: '#000', border: 'none', borderRadius: 500,
            fontSize: 14, fontWeight: 700, cursor: 'pointer',
          }}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Download All
          </button>
          <button onClick={() => startDownload(results.filter(s => selected.has(s.id)))}
            disabled={selected.size === 0}
            style={{
              display: 'inline-flex', alignItems: 'center', gap: 8, padding: '12px 24px',
              background: 'transparent', color: selected.size === 0 ? '#555' : '#fff',
              border: selected.size === 0 ? '1px solid #333' : '1px solid #fff',
              borderRadius: 500, fontSize: 14, fontWeight: 700, cursor: selected.size === 0 ? 'not-allowed' : 'pointer',
            }}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Download Selected ({selected.size})
          </button>
        </div>
      )}

      {loading ? (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '40vh', color: '#727272' }}>
          <div className="spinner" />
          <p style={{ marginTop: 16 }}>Searching...</p>
        </div>
      ) : results.length === 0 ? (
        <p style={{ color: '#727272', textAlign: 'center', padding: 40 }}>{query ? 'No results found' : 'Enter a search query'}</p>
      ) : (
        <div>
          <div style={{
            display: 'grid', gridTemplateColumns: '40px 40px 1fr 1fr 1fr 80px',
            padding: '8px 16px', fontSize: 11, fontWeight: 600, textTransform: 'uppercase',
            letterSpacing: 1, color: '#727272', borderBottom: '1px solid #333',
          }}>
            <div><input type="checkbox" checked={selected.size === results.length} onChange={() => {
              if (selected.size === results.length) setSelected(new Set())
              else setSelected(new Set(results.map(s => s.id)))
            }} style={{ accentColor: '#1DB954' }} /></div>
            <div>#</div>
            <div>Title</div>
            <div>Artist</div>
            <div>Album</div>
            <div style={{ textAlign: 'right' }}>Duration</div>
          </div>
          {results.map((s, i) => (
            <div key={s.id} style={{
              display: 'grid', gridTemplateColumns: '40px 40px 1fr 1fr 1fr 80px 50px',
              padding: '8px 16px', alignItems: 'center', borderRadius: 6,
              transition: 'background 200ms ease', fontSize: 14,
              opacity: downloading.has(s.id) ? 0.6 : 1,
            }}
            onMouseEnter={e => (e.currentTarget.style.background = '#282828')}
            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
              <div><input type="checkbox" checked={selected.has(s.id)} onChange={() => toggle(s.id)} disabled={downloading.has(s.id)} style={{ accentColor: '#1DB954' }} /></div>
              <div style={{ color: '#727272', fontSize: 13 }}>{i + 1}</div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, minWidth: 0 }}>
                {s.albumArt && <img src={s.albumArt} alt="" style={{ width: 36, height: 36, borderRadius: 4, objectFit: 'cover', flexShrink: 0 }} />}
                <span style={{ fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.title}</span>
              </div>
              <div style={{ color: '#b3b3b3', fontSize: 13, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.artist}</div>
              <div style={{ color: '#b3b3b3', fontSize: 13, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.album}</div>
              <div style={{ color: '#b3b3b3', fontSize: 13, textAlign: 'right' }}>{formatDuration(s.duration)}</div>
              <div style={{ display: 'flex', justifyContent: 'center' }}>
                <button onClick={() => startDownload([s])} disabled={downloading.has(s.id)} style={{
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  width: 32, height: 32, padding: 0, border: 'none', background: 'none',
                  color: '#727272', borderRadius: '50%', cursor: 'pointer',
                }}>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
