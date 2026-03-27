"use client"

import { useState, useRef, useCallback, useEffect } from "react"

type RecorderState = "idle" | "recording" | "uploading"

/** Pick the best supported audio MIME type for the current browser. */
function selectAudioMimeType(): string | undefined {
  const candidates = [
    "audio/webm;codecs=opus",
    "audio/webm",
    "audio/mp4",       // Safari
    "audio/ogg;codecs=opus",
  ]
  for (const mime of candidates) {
    if (typeof MediaRecorder !== "undefined" && MediaRecorder.isTypeSupported(mime)) {
      return mime
    }
  }
  // Let the browser pick its default
  return undefined
}

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

    const mimeType = selectAudioMimeType()
    const recorderOptions: MediaRecorderOptions = {}
    if (mimeType) recorderOptions.mimeType = mimeType
    const recorder = new MediaRecorder(stream, recorderOptions)
    mediaRecorderRef.current = recorder

    recorder.ondataavailable = (e) => {
      if (e.data.size > 0) chunksRef.current.push(e.data)
    }

    recorder.onstop = () => {
      const blob = new Blob(chunksRef.current, { type: recorder.mimeType })
      resolveStopRef.current?.(blob)
      resolveStopRef.current = null
    }

    // Collect data every 250ms so chunks are available before stop()
    recorder.start(250)
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

      resolveStopRef.current = (blob) => {
        // Stop the microphone stream AFTER the recorder has finished
        // producing its final data chunk. Stopping the stream before
        // recorder.stop() can cause browsers to drop the last chunk.
        if (streamRef.current) {
          streamRef.current.getTracks().forEach((track) => track.stop())
          streamRef.current = null
        }
        resolve(blob)
      }

      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
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
