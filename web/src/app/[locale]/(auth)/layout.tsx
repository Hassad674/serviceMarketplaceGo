// Soleil v2 auth layout — full-bleed wrapper. The previous header
// (brand + theme toggle) was lifted into the page-level designs because
// each Soleil auth screen owns its own visual chrome (W-01 has the
// AtelierMark inside the form column; W-02 will have its own top bar).
// Until W-02/W-03/W-04 land, the register / forgot-password /
// reset-password / invitation pages render their forms top-aligned
// inside this empty wrapper — their business logic is untouched.
export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return <main className="min-h-screen bg-background">{children}</main>
}
