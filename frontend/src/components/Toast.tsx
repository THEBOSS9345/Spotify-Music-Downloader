import { useState, useCallback, useEffect, useRef } from 'react'

type ToastType = 'success' | 'error' | 'info'

interface ToastMessage {
  id: number
  message: string
  type: ToastType
}

let toastId = 0
let addToast: ((message: string, type: ToastType) => void) | null = null

export function toast(message: string, type: ToastType = 'info') {
  if (addToast) addToast(message, type)
}

export function Toast() {
  const [toasts, setToasts] = useState<ToastMessage[]>([])
  const timers = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map())

  const remove = useCallback((id: number) => {
    setToasts(prev => prev.filter(t => t.id !== id))
    const timer = timers.current.get(id)
    if (timer) { clearTimeout(timer); timers.current.delete(id) }
  }, [])

  useEffect(() => {
    addToast = (message: string, type: ToastType) => {
      const id = ++toastId
      setToasts(prev => [...prev, { id, message, type }])
      const timer = setTimeout(() => remove(id), 3000)
      timers.current.set(id, timer)
    }
    return () => { addToast = null }
  }, [remove])

  if (toasts.length === 0) return null

  return (
    <div style={{ position: 'fixed', bottom: 24, right: 24, zIndex: 9999, display: 'flex', flexDirection: 'column', gap: 8, pointerEvents: 'none' }}>
      {toasts.map(t => (
        <div key={t.id} onClick={() => remove(t.id)} className="slide-up" style={{
          padding: '10px 18px', borderRadius: 'var(--radius)', fontSize: 13, fontWeight: 500,
          boxShadow: 'var(--shadow)', cursor: 'pointer', pointerEvents: 'auto',
          background: t.type === 'success' ? 'var(--accent)' : t.type === 'error' ? 'var(--error)' : 'var(--bg-elevated)',
          color: t.type === 'info' ? 'var(--text)' : '#000',
          border: t.type === 'info' ? '1px solid var(--border)' : 'none',
          animation: 'slideUp 250ms ease both',
        }}>
          {t.message}
        </div>
      ))}
    </div>
  )
}
