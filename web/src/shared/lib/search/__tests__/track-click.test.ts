import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { buildTrackURL, trackSearchClick } from "../track-click"

describe("buildTrackURL", () => {
  it("encodes the query params", () => {
    const url = buildTrackURL("search 1", "doc 2", 3)
    expect(url).toContain("search_id=search+1")
    expect(url).toContain("doc_id=doc+2")
    expect(url).toContain("position=3")
  })
})

describe("trackSearchClick", () => {
  let beaconMock: ReturnType<typeof vi.fn>
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    beaconMock = vi.fn().mockReturnValue(true)
    fetchMock = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal("navigator", { ...navigator, sendBeacon: beaconMock })
    vi.stubGlobal("fetch", fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("calls sendBeacon when available", () => {
    trackSearchClick("s1", "d1", 0)
    expect(beaconMock).toHaveBeenCalledOnce()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("falls back to fetch when sendBeacon is missing", () => {
    vi.stubGlobal("navigator", { ...navigator, sendBeacon: undefined })
    trackSearchClick("s1", "d1", 0)
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it("skips invalid inputs", () => {
    trackSearchClick("", "d1", 0)
    trackSearchClick("s1", "", 0)
    trackSearchClick("s1", "d1", -1)
    expect(beaconMock).not.toHaveBeenCalled()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("swallows beacon errors", () => {
    beaconMock.mockImplementation(() => {
      throw new Error("beacon failed")
    })
    expect(() => trackSearchClick("s1", "d1", 0)).not.toThrow()
  })
})
