"use client"

import { useState, useRef, useCallback, useEffect } from "react"

type RecorderState = "idle" | "recording" | "uploading"

interface UseVoiceRecorderReturn {
  state: RecorderState
  duration: number
  startRecording: () => Promise<void>
  stopRecording: () => Promise<Blob | null>
  cancelRecording: () => void
  setUploading: (uploading: boolean) => void
}

export function useVoiceRecorder(): UseVoiceRecorderReturn {
  const [state, setState] = useState<RecorderState>("idle")
  const [duration, setDuration] = useState(0)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const chunksRef = useRef<Blob[]>([])
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const resolveStopRef = useRef<((blob: Blob | null) => void) | null>(null)

  const cleanup = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current)
      timerRef.current = null
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop())
      streamRef.current = null
    }
    mediaRecorderRef.current = null
    chunksRef.current = []
    setDuration(0)
  }, [])

  // Cleanup on unmount
  useEffect(() => cleanup, [cleanup])

  const startRecording = useCallback(async () => {
    chunksRef.current = []
    setDuration(0)

    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    streamRef.current = stream

    const recorder = new MediaRecorder(stream, {
      mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
        ? "audio/webm;codecs=opus"
        : "audio/webm",
    })
    mediaRecorderRef.current = recorder

    recorder.ondataavailable = (e) => {
      if (e.data.size > 0) chunksRef.current.push(e.data)
    }

    recorder.onstop = () => {
      const blob = new Blob(chunksRef.current, { type: recorder.mimeType })
      resolveStopRef.current?.(blob)
      resolveStopRef.current = null
    }

    recorder.start()
    setState("recording")

    timerRef.current = setInterval(() => {
      setDuration((prev) => prev + 1)
    }, 1000)
  }, [])

  const stopRecording = useCallback((): Promise<Blob | null> => {
    return new Promise((resolve) => {
      const recorder = mediaRecorderRef.current
      if (!recorder || recorder.state !== "recording") {
        cleanup()
        setState("idle")
        resolve(null)
        return
      }

      resolveStopRef.current = resolve
      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
      }
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((track) => track.stop())
        streamRef.current = null
      }
      recorder.stop()
      setState("idle")
    })
  }, [cleanup])

  const cancelRecording = useCallback(() => {
    const recorder = mediaRecorderRef.current
    if (recorder && recorder.state === "recording") {
      recorder.stop()
    }
    resolveStopRef.current = null
    cleanup()
    setState("idle")
  }, [cleanup])

  const setUploading = useCallback((uploading: boolean) => {
    setState(uploading ? "uploading" : "idle")
  }, [])

  return { state, duration, startRecording, stopRecording, cancelRecording, setUploading }
}
