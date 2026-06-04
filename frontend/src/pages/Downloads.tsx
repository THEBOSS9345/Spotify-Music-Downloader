import { useDownloadState } from '../useDownloadState'
import type { Download } from '../types'

const statusIcon: Record<string, string> = {
  pending: '⏳',
  searching: '🔍',
  downloading: '⬇️',
  converting: '🔄',
  complete: '✅',
  failed: '❌',
}

const statusLabel: Record<string, string> = {
  pending: 'Queued',
  searching: 'Searching...',
  downloading: 'Downloading...',
  converting: 'Converting...',
  complete: 'Done',
  failed: 'Failed',
}

export function Downloads() {
  const { downloads, queue, active, queued } = useDownloadState()

  const completed = downloads.filter(d => d.status === 'complete')
  const failed = downloads.filter(d => d.status === 'failed')
  const activeItems = downloads.filter(d => ['searching', 'downloading', 'converting'].includes(d.status))

  return (
    <div style={{ padding: 32, maxWidth: 860 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 28 }}>
        <h1 style={{ fontSize: 28, fontWeight: 700 }}>Downloads</h1>
        {(active > 0 || queued > 0) && (
          <span style={{
            fontSize: 13, background: '#1DB954', color: '#000', padding: '2px 10px',
            borderRadius: 500, fontWeight: 600,
          }}>
            {active + queued} in progress
          </span>
        )}
      </div>

      {(activeItems.length === 0 && queue.length === 0 && completed.length === 0 && failed.length === 0) ? (
        <div style={{ textAlign: 'center', padding: '80px 0', color: '#727272' }}>
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ marginBottom: 16, opacity: 0.4 }}>
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <p style={{ fontSize: 15 }}>No downloads yet</p>
          <p style={{ fontSize: 13, marginTop: 6 }}>Open a playlist and start downloading</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>

          {/* Active downloads */}
          {activeItems.map(d => (
            <DownloadRow key={d.id} d={d} />
          ))}

          {/* Queue section */}
          {queue.length > 0 && (
            <>
              <div style={{
                fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 1.2,
                color: '#727272', padding: '16px 4px 8px', borderTop: '1px solid #222', marginTop: 8,
              }}>
                Up Next ({queue.length})
              </div>
              {queue.map((q, i) => (
                <div key={i} style={{
                  display: 'flex', alignItems: 'center', gap: 14,
                  padding: '12px 14px', background: '#181818', borderRadius: 8,
                  opacity: 0.65,
                }}>
                  <span style={{ fontSize: 13, color: '#727272', minWidth: 20 }}>{i + 1}.</span>
                  <span style={{ width: 28, height: 28, borderRadius: 4, background: '#282828', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 }}>⏳</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{q.song.title}</div>
                    <div style={{ fontSize: 11, color: '#727272' }}>{q.song.artist}</div>
                  </div>
                </div>
              ))}
            </>
          )}

          {/* Completed + Failed */}
          {(completed.length > 0 || failed.length > 0) && (
            <div style={{
              fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 1.2,
              color: '#727272', padding: '16px 4px 8px', borderTop: '1px solid #222', marginTop: 8,
            }}>
              History ({completed.length + failed.length})
            </div>
          )}
          {completed.map(d => (
            <DownloadRow key={d.id} d={d} dimmed />
          ))}
          {failed.map(d => (
            <DownloadRow key={d.id} d={d} dimmed />
          ))}
        </div>
      )}
    </div>
  )
}

function DownloadRow({ d, dimmed }: { d: Download; dimmed?: boolean }) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 14,
      padding: '12px 14px', background: dimmed ? '#141414' : '#1e1e1e', borderRadius: 10,
      transition: 'background 200ms ease',
    }}>
      <span style={{ width: 28, height: 28, borderRadius: 4, background: '#282828', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 13 }}>
        {statusIcon[d.status] || '⏳'}
      </span>

      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontWeight: 600, fontSize: 14, marginBottom: 2,
          whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
          color: d.status === 'failed' ? '#e91429' : dimmed ? '#b3b3b3' : '#fff',
        }}>
          {d.song?.title || ''}
        </div>
        <div style={{ fontSize: 12, color: '#727272' }}>{d.song?.artist || ''}</div>
      </div>

      {['pending', 'searching', 'downloading', 'converting'].includes(d.status) && (
        <div style={{ width: 160 }}>
          <div style={{ height: 4, background: '#282828', borderRadius: 2, overflow: 'hidden' }}>
            <div style={{
              height: '100%', width: `${Math.max(d.progress, 5)}%`,
              background: d.status === 'failed' ? '#e91429' : '#1DB954',
              borderRadius: 2, transition: 'width 0.4s ease',
            }} />
          </div>
        </div>
      )}

      <div style={{
        fontSize: 12, fontWeight: 600, textAlign: 'right', minWidth: 80,
        color: d.status === 'failed' ? '#e91429' : d.status === 'complete' ? '#1DB954' : '#b3b3b3',
      }}>
        {statusLabel[d.status] || d.status}
      </div>
    </div>
  )
}
