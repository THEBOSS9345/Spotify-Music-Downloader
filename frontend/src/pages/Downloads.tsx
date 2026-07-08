import { useState, useCallback } from 'react'
import { useDownloadState } from '../useDownloadState'
import { api } from '../api'
import { toast } from '../components/Toast'
import type { Download } from '../types'

const icon: Record<string, string> = {
  pending: '⏳', searching: '🔍', downloading: '⬇️',
  converting: '🔄', complete: '✅', failed: '❌',
}

const label: Record<string, string> = {
  pending: 'Queued', searching: 'Searching...', downloading: 'Downloading...',
  converting: 'Converting...', complete: 'Done', failed: 'Failed',
}

type Tab = 'all' | 'queued' | 'active' | 'complete' | 'failed'

const TABS: { key: Tab; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'queued', label: 'Queued' },
  { key: 'active', label: 'Active' },
  { key: 'complete', label: 'Complete' },
  { key: 'failed', label: 'Failed' },
]

export function Downloads() {
  const { downloads, queue, active, queued } = useDownloadState()
  const [tab, setTab] = useState<Tab>('all')

  const handleRetry = useCallback(async (d: Download) => {
    try {
      await api.retry([d.id])
      toast(`Retrying: ${d.song?.title}`, 'success')
    } catch { toast('Retry failed', 'error') }
  }, [])

  const completed = downloads.filter(d => d.status === 'complete')
  const failed = downloads.filter(d => d.status === 'failed')
  const activeItems = downloads.filter(d => ['searching', 'downloading', 'converting'].includes(d.status))
  const pendingItems = downloads.filter(d => d.status === 'pending')
  const hasAny = activeItems.length + queue.length + completed.length + failed.length > 0

  const filteredDownloads = (() => {
    switch (tab) {
      case 'queued': return { items: pendingItems, queue: [...queue], showQueueTitle: false, showHistory: false, historyItems: [] as Download[] }
      case 'active': return { items: activeItems, queue: [], showQueueTitle: false, showHistory: false, historyItems: [] as Download[] }
      case 'complete': return { items: completed, queue: [], showQueueTitle: false, showHistory: false, historyItems: [] as Download[] }
      case 'failed': return { items: failed, queue: [], showQueueTitle: false, showHistory: false, historyItems: [] as Download[] }
      default: return {
        items: activeItems, queue: [...queue], showQueueTitle: true,
        showHistory: completed.length + failed.length > 0,
        historyItems: [...completed, ...failed] as Download[],
      }
    }
  })()

  return (
    <div className="fade-in page" style={{ padding: 28, maxWidth: 720 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <h1 style={{ fontSize: 24, fontWeight: 700, letterSpacing: '-0.02em' }}>Downloads</h1>
        {(active > 0 || queued > 0) && (
          <span style={{
            fontSize: 11, background: 'var(--accent)', color: '#000', padding: '2px 10px',
            borderRadius: 'var(--radius-full)', fontWeight: 600,
          }}>
            {active + queued} in progress
          </span>
        )}
      </div>

      {/* Tabs */}
      {hasAny && (
        <div style={{ display: 'flex', gap: 4, marginBottom: 20, borderBottom: '1px solid var(--border)' }}>
          {TABS.map(t => (
            <button key={t.key} onClick={() => setTab(t.key)} className="row-hover" style={{
              background: 'none', border: 'none', color: tab === t.key ? 'var(--text)' : 'var(--text-subdued)',
              fontSize: 13, fontWeight: tab === t.key ? 600 : 400, cursor: 'pointer', padding: '8px 14px',
              borderBottom: tab === t.key ? '2px solid var(--accent)' : '2px solid transparent',
              marginBottom: -1, transition: 'color var(--transition), border-color var(--transition)',
            }}>
              {t.label}
            </button>
          ))}
        </div>
      )}

      {!hasAny ? (
        <div className="fade-in" style={{ textAlign: 'center', padding: '80px 0', color: 'var(--text-subdued)' }}>
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ marginBottom: 14, opacity: 0.3 }}>
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <p style={{ fontSize: 14 }}>No downloads yet</p>
          <p style={{ fontSize: 12, marginTop: 4, color: 'var(--text-subdued)' }}>Open a playlist and start downloading</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {/* Active / filtered items */}
          {filteredDownloads.items.map((d, i) => (
            <DownloadRow key={d.id} d={d} delay={i * 30} />
          ))}

          {/* Queue */}
          {filteredDownloads.queue.length > 0 && (
            <>
              {filteredDownloads.showQueueTitle && (
                <div style={{
                  fontSize: 10, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 1.2,
                  color: 'var(--text-subdued)', padding: '14px 4px 6px', borderTop: '1px solid var(--border)', marginTop: 6,
                }}>
                  Up Next ({filteredDownloads.queue.length})
                </div>
              )}
              {filteredDownloads.queue.map((q, i) => (
                <div key={i} className="slide-up" style={{
                  display: 'flex', alignItems: 'center', gap: 12,
                  padding: '10px 14px', background: 'var(--bg)', borderRadius: 'var(--radius)',
                  opacity: 0.5, animationDelay: `${i * 20}ms`,
                }}>
                  <span style={{ fontSize: 13, color: 'var(--text-subdued)', minWidth: 18, textAlign: 'right' }}>{i + 1}.</span>
                  <span style={{ width: 24, height: 24, borderRadius: 4, background: 'var(--bg-hover)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12 }}>⏳</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{q.song.title}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-subdued)' }}>{q.song.artist}</div>
                  </div>
                </div>
              ))}
            </>
          )}

          {/* History */}
          {filteredDownloads.showHistory && (
            <div style={{
              fontSize: 10, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 1.2,
              color: 'var(--text-subdued)', padding: '14px 4px 6px', borderTop: '1px solid var(--border)', marginTop: 6,
            }}>
              History ({filteredDownloads.historyItems.length})
            </div>
          )}
          {filteredDownloads.historyItems.filter(d => d.status === 'complete').map((d, i) => (
            <DownloadRow key={d.id} d={d} dimmed delay={i * 20} />
          ))}
          {filteredDownloads.historyItems.filter(d => d.status === 'failed').map((d, i) => (
            <DownloadRow key={d.id} d={d} dimmed delay={i * 20} onRetry={handleRetry} />
          ))}
        </div>
      )}
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return ''
  const mb = bytes / 1024 / 1024
  return mb >= 1 ? `${mb.toFixed(1)} MB` : `${(bytes / 1024).toFixed(0)} KB`
}

function DownloadRow({ d, dimmed, onRetry, delay }: { d: Download; dimmed?: boolean; onRetry?: (d: Download) => void; delay?: number }) {
  const active = ['pending', 'searching', 'downloading', 'converting'].includes(d.status)
  const hasBytes = d.downloadedBytes > 0 && d.totalBytes > 0
  const pct = hasBytes ? Math.round(d.downloadedBytes / d.totalBytes * 100) : d.progress
  return (
    <div onClick={() => d.status === 'failed' && onRetry?.(d)} className="slide-up" style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '10px 14px', background: dimmed ? 'var(--bg)' : 'var(--bg-surface)',
      borderRadius: 'var(--radius)', cursor: d.status === 'failed' ? 'pointer' : undefined,
      transition: 'background var(--transition), opacity var(--transition)',
      opacity: active ? 1 : 0.65,
      animationDelay: `${delay || 0}ms`,
      border: d.status === 'failed' ? '1px solid rgba(233,20,41,0.15)' : '1px solid transparent',
    }}>
      <span style={{
        width: 24, height: 24, borderRadius: 4, background: 'var(--bg-hover)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12,
        flexShrink: 0,
      }}>
        {icon[d.status] || '⏳'}
      </span>

      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontWeight: 500, fontSize: 13, marginBottom: 1,
          whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
          color: d.status === 'failed' ? 'var(--error)' : 'var(--text)',
          transition: 'color var(--transition)',
        }}>
          {d.song?.title || ''}
        </div>
        <div style={{ fontSize: 11, color: 'var(--text-subdued)' }}>{d.song?.artist || ''}</div>
      </div>

      {(active && d.status !== 'pending') ? (
        <div style={{ width: 140, display: 'flex', flexDirection: 'column', gap: 3 }}>
          <div style={{ height: 6, background: 'var(--bg-hover)', borderRadius: 3, overflow: 'hidden' }}>
            <div style={{
              height: '100%', width: `${Math.max(pct, 2)}%`,
              background: d.status === 'converting' ? 'var(--accent)' : 'var(--accent)',
              borderRadius: 3,
              transition: 'width 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
            }} />
          </div>
          <div style={{ fontSize: 10, color: 'var(--text-subdued)', textAlign: 'right' }}>
            {d.status === 'downloading' && hasBytes
              ? `${formatBytes(d.downloadedBytes)} / ${formatBytes(d.totalBytes)}`
              : `${pct}%`}
          </div>
        </div>
      ) : (
        <div style={{
          fontSize: 11, fontWeight: 600, textAlign: 'right', minWidth: 64,
          color: d.status === 'failed' ? 'var(--error)' : d.status === 'complete' ? 'var(--accent)' : 'var(--text-secondary)',
          transition: 'color var(--transition)',
        }}>
          {label[d.status] || d.status}
        </div>
      )}
    </div>
  )
}
