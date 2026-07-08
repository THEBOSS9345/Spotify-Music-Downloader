import { useState } from 'react'
import type { ReactNode } from 'react'
import { Sidebar } from './Sidebar'
import type { User } from '../types'

export function Layout({ user, onLogout, children }: { user: User | null; onLogout: () => void; children: ReactNode }) {
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <div style={{ height: '100vh', display: 'flex', overflow: 'hidden' }}>
      <div className={`sidebar-overlay${mobileOpen ? ' open' : ''}`} onClick={() => setMobileOpen(false)} />
      <div className={`sidebar-panel${mobileOpen ? ' open' : ''}`}>
        <Sidebar user={user} onLogout={onLogout} onMobileClose={() => setMobileOpen(false)} />
      </div>
      <div className="sidebar-desktop">
        <Sidebar user={user} onLogout={onLogout} />
      </div>
      <main style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, background: '#121212' }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '8px 16px',
          borderBottom: '1px solid var(--border)', minHeight: 52,
        }}>
          <button className="hamburger" onClick={() => setMobileOpen(true)} aria-label="Open menu">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
            </svg>
          </button>
          <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            SpotScoop
          </div>
        </div>
        <div className="main-scroll" style={{ flex: 1, overflowY: 'auto' }}>
          {children}
        </div>
      </main>
    </div>
  )
}
