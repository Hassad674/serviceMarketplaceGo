"use client"

import { useEffect, useState } from "react"

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8083"

type Word = { id: string; word: string; created_at: string }

export function TestDB() {
  const [backendStatus, setBackendStatus] = useState<"loading" | "ok" | "error">("loading")
  const [dbStatus, setDbStatus] = useState<"loading" | "ok" | "error">("loading")
  const [words, setWords] = useState<Word[]>([])
  const [input, setInput] = useState("")
  const [adding, setAdding] = useState(false)

  async function checkHealth() {
    try {
      const res = await fetch(`${API}/api/v1/test/health-check`)
      const data = await res.json()
      setBackendStatus(data.backend === "ok" ? "ok" : "error")
      setDbStatus(data.database === "ok" ? "ok" : "error")
    } catch {
      setBackendStatus("error")
      setDbStatus("error")
    }
  }

  async function fetchWords() {
    try {
      const res = await fetch(`${API}/api/v1/test/words`)
      const data = await res.json()
      setWords(data.words || [])
    } catch {
      // ignore
    }
  }

  async function addWord() {
    if (!input.trim()) return
    setAdding(true)
    try {
      await fetch(`${API}/api/v1/test/words`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ word: input.trim() }),
      })
      setInput("")
      await fetchWords()
    } catch {
      // ignore
    }
    setAdding(false)
  }

  useEffect(() => {
    checkHealth()
    fetchWords()
  }, [])

  const statusBadge = (status: "loading" | "ok" | "error") => {
    if (status === "loading") return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-300">Checking...</span>
    if (status === "ok") return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-500/20 text-green-800 dark:text-green-400">OK</span>
    return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 dark:bg-red-500/20 text-red-800 dark:text-red-400">ERROR</span>
  }

  return (
    <section className="w-full max-w-2xl mx-auto mt-16 p-8 border border-gray-200 dark:border-gray-800 rounded-lg bg-white dark:bg-gray-900">
      <h2 className="text-xl font-bold mb-6 text-gray-900 dark:text-white">Test Backend &amp; Database</h2>

      <div className="flex gap-8 mb-8">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-500 dark:text-gray-400">Backend:</span>
          {statusBadge(backendStatus)}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-500 dark:text-gray-400">Database:</span>
          {statusBadge(dbStatus)}
        </div>
      </div>

      <div className="mb-6">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addWord()}
            placeholder="Type a word and press Enter..."
            className="flex-1 h-10 px-4 rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
          />
          <button
            onClick={addWord}
            disabled={adding || !input.trim()}
            className="h-10 px-4 rounded-md bg-gray-900 dark:bg-white text-white dark:text-gray-900 text-sm font-medium hover:bg-gray-800 dark:hover:bg-gray-100 disabled:opacity-50 transition-all"
          >
            {adding ? "..." : "Add"}
          </button>
        </div>
      </div>

      {words.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">Saved words ({words.length})</h3>
          <div className="flex flex-wrap gap-2">
            {words.map((w) => (
              <span key={w.id} className="px-3 py-1 bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-200 rounded-full text-sm">
                {w.word}
              </span>
            ))}
          </div>
        </div>
      )}

      <p className="mt-6 text-xs text-gray-400 dark:text-gray-500">
        API: {API} — Debug tool for backend &amp; DB connectivity.
      </p>
    </section>
  )
}
