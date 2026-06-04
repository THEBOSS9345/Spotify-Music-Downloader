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
    <div style={{ position: 'fixed', bottom: 24, right: 24, zIndex: 9999, display: 'flex', flexDirection: 'column', gap: 8 }}>
      {toasts.map(t => (
        <div key={t.id} onClick={() => remove(t.id)} style={{
          padding: '12px 20px', borderRadius: 6, fontSize: 13, fontWeight: 500,
          boxShadow: '0 4px 20px rgba(0,0,0,0.5)', cursor: 'pointer',
          background: t.type === 'success' ? '#1DB954' : t.type === 'error' ? '#e91429' : '#242424',
          color: t.type === 'info' ? '#fff' : '#000',
          border: t.type === 'info' ? '1px solid #333' : 'none',
        }}>
          {t.message}
        </div>
      ))}
    </div>
  )
}
