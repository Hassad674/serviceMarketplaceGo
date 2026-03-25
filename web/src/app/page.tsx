import Link from "next/link"
import { TestDB } from "./test-db"

export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col">
      {/* Navbar */}
      <header className="border-b border-gray-200 bg-white">
        <nav className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
          <span className="text-xl font-bold tracking-tight text-gray-900">
            Marketplace Service
          </span>
          <div className="flex items-center gap-4">
            <Link
              href="/login"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Sign In
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800"
            >
              Sign Up
            </Link>
          </div>
        </nav>
      </header>

      {/* Hero */}
      <section className="flex flex-1 flex-col items-center justify-center bg-gradient-to-b from-gray-50 to-white px-6 py-24 text-center">
        <h1 className="max-w-3xl text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">
          The B2B platform connecting agencies, freelancers and enterprises
        </h1>
        <p className="mt-6 max-w-2xl text-lg leading-relaxed text-gray-600">
          Find the best service providers, publish your projects and
          collaborate with confidence on a platform built for
          professionals.
        </p>
        <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
          <Link
            href="/register"
            className="rounded-lg bg-gray-900 px-6 py-3 text-sm font-semibold text-white shadow-sm hover:bg-gray-800"
          >
            Get started for free
          </Link>
          <Link
            href="/projects"
            className="rounded-lg border border-gray-300 bg-white px-6 py-3 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
          >
            Browse projects
          </Link>
        </div>
      </section>

      {/* Feature cards */}
      <section className="mx-auto grid max-w-7xl gap-8 px-6 py-20 sm:grid-cols-3">
        <div className="rounded-xl border border-gray-200 bg-white p-8">
          <h3 className="text-lg font-semibold text-gray-900">Agencies</h3>
          <p className="mt-2 text-sm leading-relaxed text-gray-600">
            Manage your team, land missions and grow your
            business.
          </p>
          <Link href="/agencies" className="mt-4 inline-block text-sm font-medium text-gray-900 underline underline-offset-4 hover:text-gray-700">
            Browse agencies
          </Link>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-8">
          <h3 className="text-lg font-semibold text-gray-900">Freelancers</h3>
          <p className="mt-2 text-sm leading-relaxed text-gray-600">
            Find missions matching your skills and manage your
            invoices.
          </p>
          <Link href="/freelances" className="mt-4 inline-block text-sm font-medium text-gray-900 underline underline-offset-4 hover:text-gray-700">
            Browse freelancers
          </Link>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-8">
          <h3 className="text-lg font-semibold text-gray-900">Enterprises</h3>
          <p className="mt-2 text-sm leading-relaxed text-gray-600">
            Publish your projects and find the ideal service providers in just a few
            clicks.
          </p>
          <Link href="/projects" className="mt-4 inline-block text-sm font-medium text-gray-900 underline underline-offset-4 hover:text-gray-700">
            Browse projects
          </Link>
        </div>
      </section>

      {/* Debug: Test DB connectivity */}
      <TestDB />

    </main>
  )
}
