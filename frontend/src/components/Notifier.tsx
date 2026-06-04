import { useRef, useEffect } from 'react'
import { useDownloadState, getDownloadState } from '../useDownloadState'
import { toast } from './Toast'

const NOTIFY_THROTTLE_MS = 4000

export function Notifier() {
  const known = useRef(new Map<string, string>())
  const pending = useRef<{ complete: number; failed: number }>({ complete: 0, failed: 0 })
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useDownloadState()

  useEffect(() => {
    const state = getDownloadState()
    let newComplete = 0
    let newFailed = 0

    for (const d of state.downloads) {
      const prev = known.current.get(d.id)
      if (!prev) {
        known.current.set(d.id, d.status)
        continue
      }
      if (prev === d.status) continue
      known.current.set(d.id, d.status)

      if (d.status === 'complete') newComplete++
      if (d.status === 'failed') newFailed++
    }

    if (newComplete === 0 && newFailed === 0) return

    const p = pending.current
    p.complete += newComplete
    p.failed += newFailed

    if (timer.current) clearTimeout(timer.current)

    timer.current = setTimeout(() => {
      const parts: string[] = []
      if (p.complete > 0) parts.push(`${p.complete} download${p.complete > 1 ? 's' : ''} finished`)
      if (p.failed > 0) parts.push(`${p.failed} download${p.failed > 1 ? 's' : ''} failed`)
      toast(parts.join(', '), p.failed > 0 && p.complete === 0 ? 'error' : 'success')
      p.complete = 0
      p.failed = 0
    }, NOTIFY_THROTTLE_MS)

    return () => {
      if (timer.current) clearTimeout(timer.current)
    }
  })

  return null
}
