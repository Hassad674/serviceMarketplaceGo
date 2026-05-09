import { describe, it, expect } from "vitest"
import {
  FILTERS_BY_PERSONA,
  resolveFilterVisibility,
} from "../search-filter-config"

describe("search-filter-config", () => {
  describe("FILTERS_BY_PERSONA", () => {
    it("freelance shows every filter", () => {
      const v = FILTERS_BY_PERSONA.freelance
      expect(v).toEqual({
        availability: true,
        pricing: true,
        location: true,
        workMode: true,
        languages: true,
        expertise: true,
        skills: true,
        rating: true,
      })
    })

    it("agency hides work mode but keeps everything else", () => {
      const v = FILTERS_BY_PERSONA.agency
      expect(v.workMode).toBe(false)
      expect(v.location).toBe(true)
      expect(v.skills).toBe(true)
      expect(v.pricing).toBe(true)
      expect(v.rating).toBe(true)
      expect(v.languages).toBe(true)
      expect(v.expertise).toBe(true)
    })

    it("referrer hides work mode + skills + pricing", () => {
      const v = FILTERS_BY_PERSONA.referrer
      expect(v.workMode).toBe(false)
      expect(v.skills).toBe(false)
      expect(v.pricing).toBe(false)
      // Location, languages, expertise, rating, availability stay.
      expect(v.location).toBe(true)
      expect(v.languages).toBe(true)
      expect(v.expertise).toBe(true)
      expect(v.rating).toBe(true)
      expect(v.availability).toBe(true)
    })
  })

  describe("resolveFilterVisibility", () => {
    it("returns the freelance map when persona is undefined", () => {
      expect(resolveFilterVisibility(undefined)).toEqual(
        FILTERS_BY_PERSONA.freelance,
      )
    })

    it("returns the persona-scoped map", () => {
      expect(resolveFilterVisibility("agency")).toBe(FILTERS_BY_PERSONA.agency)
      expect(resolveFilterVisibility("referrer")).toBe(
        FILTERS_BY_PERSONA.referrer,
      )
      expect(resolveFilterVisibility("freelance")).toBe(
        FILTERS_BY_PERSONA.freelance,
      )
    })
  })
})
