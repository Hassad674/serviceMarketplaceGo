/**
 * Stripe Connect Embedded Components — Soleil v2 appearance.
 *
 * Visual port of the previous Rose theme to the Atelier ivoire/corail
 * palette so the Stripe-rendered widgets blend with the host app.
 *
 * IMPORTANT: this object is consumed by `loadConnectAndInitialize` and
 * only feeds Stripe's Appearance API. Behavior, callbacks, and the
 * iframe wiring are NOT affected.
 *
 * Reference: https://docs.stripe.com/connect/embedded-appearance-options
 */

const PALETTE = {
  // Soleil v2 brand & surfaces
  colorPrimary: "#e85d4a",            // corail (primary)
  colorBackground: "#ffffff",         // surface card white
  colorText: "#2a1f15",               // encre — primary text
  colorSecondaryText: "#7a6850",      // tabac — secondary text
  colorBorder: "#f0e6d8",             // sable clair — soft border
  colorDanger: "#c43a26",             // corail deep — destructive
  colorWarning: "#d4924a",            // ambre — warning
  colorSuccess: "#5a9670",            // sapin — success
  colorSecondaryLinkText: "#c43a26",  // hover/link emphasis

  fontFamily: "'Inter Tight', ui-sans-serif, system-ui, sans-serif",

  offsetBackgroundColor: "#fffbf5",   // ivoire — page bg (gives the warm halo around fields)
  formBackgroundColor: "#ffffff",
  formHighlightColorBorder: "#e85d4a",
  formAccentColor: "#e85d4a",

  // Primary CTAs — corail solid
  buttonPrimaryColorBackground: "#e85d4a",
  buttonPrimaryColorBorder: "#e85d4a",
  buttonPrimaryColorText: "#fffbf5",

  // Secondary buttons — ivoire surface, sable border, encre text
  buttonSecondaryColorBackground: "#ffffff",
  buttonSecondaryColorBorder: "#e0d3bc",
  buttonSecondaryColorText: "#2a1f15",

  actionPrimaryColorText: "#c43a26",
  actionSecondaryColorText: "#7a6850",

  // Badges — soft pastels matching the Soleil chip language
  badgeNeutralColorBackground: "#f0e6d8",
  badgeNeutralColorText: "#7a6850",
  badgeNeutralColorBorder: "#e0d3bc",
  badgeSuccessColorBackground: "#e8f2eb",
  badgeSuccessColorText: "#2f5b41",
  badgeSuccessColorBorder: "#cbe2d3",
  badgeWarningColorBackground: "#fbf0dc",
  badgeWarningColorText: "#7a4a14",
  badgeWarningColorBorder: "#f0e2bf",
  badgeDangerColorBackground: "#fde9e3",
  badgeDangerColorText: "#7a1f12",
  badgeDangerColorBorder: "#f5cdc3",
}

const TYPOGRAPHY = {
  fontSizeBase: "15px",
  headingXlFontSize: "28px",
  headingXlFontWeight: "600",
  headingLgFontSize: "22px",
  headingLgFontWeight: "600",
  headingMdFontSize: "17px",
  headingMdFontWeight: "600",
  headingSmFontSize: "14px",
  headingSmFontWeight: "600",
  headingXsFontSize: "13px",
  headingXsFontWeight: "600",
  bodyMdFontSize: "15px",
  bodyMdFontWeight: "400",
  bodySmFontSize: "13px",
  bodySmFontWeight: "400",
  labelMdFontSize: "13px",
  labelMdFontWeight: "600",
  labelSmFontSize: "12px",
  labelSmFontWeight: "500",
  buttonPrimaryFontWeight: "600",
  buttonPrimaryFontSize: "14px",
  buttonSecondaryFontWeight: "600",
  buttonSecondaryFontSize: "14px",
}

const LAYOUT_BALANCED = {
  spacingUnit: "12px",
  borderRadius: "14px",
  buttonBorderRadius: "999px",        // Soleil signature: full pills
  formBorderRadius: "12px",
  badgeBorderRadius: "999px",
  overlayBorderRadius: "20px",
  formBorderWidth: "1px",
  buttonBorderWidth: "1px",
}

export const ROSE_APPEARANCE = {
  variables: {
    ...PALETTE,
    ...TYPOGRAPHY,
    ...LAYOUT_BALANCED,
  },
} as const
