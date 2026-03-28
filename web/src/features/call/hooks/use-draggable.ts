"use client"

import { useState, useCallback, useRef } from "react"

interface Position {
  x: number
  y: number
}

interface DragHandlers {
  onPointerDown: (e: React.PointerEvent) => void
}

interface UseDraggableReturn {
  position: Position
  dragHandlers: DragHandlers
}

export function useDraggable(initial: Position = { x: 0, y: 0 }): UseDraggableReturn {
  const [position, setPosition] = useState<Position>(initial)
  const dragging = useRef(false)
  const offset = useRef<Position>({ x: 0, y: 0 })

  const onPointerMove = useCallback((e: PointerEvent) => {
    if (!dragging.current) return
    const rawX = e.clientX - offset.current.x
    const rawY = e.clientY - offset.current.y
    const maxX = window.innerWidth - 40
    const maxY = window.innerHeight - 40
    setPosition({
      x: Math.max(0, Math.min(rawX, maxX)),
      y: Math.max(0, Math.min(rawY, maxY)),
    })
  }, [])

  const onPointerUp = useCallback(() => {
    dragging.current = false
    document.removeEventListener("pointermove", onPointerMove)
    document.removeEventListener("pointerup", onPointerUp)
  }, [onPointerMove])

  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      dragging.current = true
      offset.current = {
        x: e.clientX - position.x,
        y: e.clientY - position.y,
      }
      document.addEventListener("pointermove", onPointerMove)
      document.addEventListener("pointerup", onPointerUp)
    },
    [position, onPointerMove, onPointerUp],
  )

  return { position, dragHandlers: { onPointerDown } }
}
