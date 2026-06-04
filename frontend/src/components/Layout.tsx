import type { ReactNode } from 'react'
import { Sidebar } from './Sidebar'
import type { User } from '../types'

export function Layout({ user, onLogout, children }: { user: User | null; onLogout: () => void; children: ReactNode }) {
  return (
    <div style={{ height: '100vh', display: 'flex' }}>
      <Sidebar user={user} onLogout={onLogout} />
      <main style={{ flex: 1, overflowY: 'auto', background: '#121212' }}>
        {children}
      </main>
    </div>
  )
}
