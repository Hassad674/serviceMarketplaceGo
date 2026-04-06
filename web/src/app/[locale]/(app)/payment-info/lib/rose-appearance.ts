/**
 * Stripe Connect Embedded Components — Rose theme.
 *
 * Exposes every Appearance API variable we have visibility into.
 * Reference: https://docs.stripe.com/connect/embedded-appearance-options
 *
 * Three variants are available — pick one at init time. Variant "balanced"
 * is used by default (matches the marketplace design system).
 */

const PALETTE = {
  colorPrimary: "#F43F5E",
  colorBackground: "#FFFFFF",
  colorText: "#0F172A",
  colorSecondaryText: "#475569",
  colorBorder: "#E2E8F0",
  colorDanger: "#EF4444",
  colorWarning: "#F59E0B",
  colorSuccess: "#22C55E",
  colorSecondaryLinkText: "#E11D48",

  fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif",

  offsetBackgroundColor: "#F8FAFC",
  formBackgroundColor: "#FFFFFF",
  formHighlightColorBorder: "#F43F5E",
  formAccentColor: "#F43F5E",

  buttonPrimaryColorBackground: "#F43F5E",
  buttonPrimaryColorBorder: "#F43F5E",
  buttonPrimaryColorText: "#FFFFFF",

  buttonSecondaryColorBackground: "#FFFFFF",
  buttonSecondaryColorBorder: "#E2E8F0",
  buttonSecondaryColorText: "#0F172A",

  actionPrimaryColorText: "#E11D48",
  actionSecondaryColorText: "#475569",

  badgeNeutralColorBackground: "#F1F5F9",
  badgeNeutralColorText: "#475569",
  badgeNeutralColorBorder: "#E2E8F0",
  badgeSuccessColorBackground: "#DCFCE7",
  badgeSuccessColorText: "#166534",
  badgeSuccessColorBorder: "#BBF7D0",
  badgeWarningColorBackground: "#FEF3C7",
  badgeWarningColorText: "#92400E",
  badgeWarningColorBorder: "#FDE68A",
  badgeDangerColorBackground: "#FEE2E2",
  badgeDangerColorText: "#991B1B",
  badgeDangerColorBorder: "#FECACA",
}

const TYPOGRAPHY = {
  fontSizeBase: "15px",
  headingXlFontSize: "28px",
  headingXlFontWeight: "800",
  headingLgFontSize: "22px",
  headingLgFontWeight: "700",
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
  borderRadius: "12px",
  buttonBorderRadius: "10px",
  formBorderRadius: "10px",
  badgeBorderRadius: "999px",
  overlayBorderRadius: "16px",
  formBorderWidth: "1.5px",
  buttonBorderWidth: "1px",
}

export const ROSE_APPEARANCE = {
  variables: {
    ...PALETTE,
    ...TYPOGRAPHY,
    ...LAYOUT_BALANCED,
  },
} as const
