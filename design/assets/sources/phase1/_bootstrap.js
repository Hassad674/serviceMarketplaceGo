// Defensive bootstrap — runs BEFORE any soleil-* dependent file.
// If soleil.jsx ever fails to load (network/transient), every dependent file still
// has a valid window.S / window.SI / window.Portrait to deref at module top-level
// instead of crashing the whole canvas with "cannot read property 'bg' of undefined".
// soleil.jsx then overwrites these with the real values when it loads cleanly.

(function () {
  if (window.S) return;
  window.S = {
    bg: '#fffbf5',
    surface: '#ffffff',
    border: '#f0e6d8',
    borderStrong: '#e0d3bc',
    text: '#2a1f15',
    textMute: '#7a6850',
    textSubtle: '#a89679',
    accent: '#e85d4a',
    accentSoft: '#fde9e3',
    accentDeep: '#c43a26',
    pink: '#f08aa8',
    pinkSoft: '#fde6ed',
    green: '#5a9670',
    greenSoft: '#e8f2eb',
    amber: '#d4924a',
    serif: 'Fraunces, Georgia, serif',
    sans: '"Inter Tight", system-ui, sans-serif',
    mono: '"Geist Mono", monospace',
  };
  // Stub Icons / SI / Portrait so module top-level destructuring doesn't crash.
  // If real ones load they'll overwrite. If not, components render quiet placeholders.
  if (!window.Icons) window.Icons = new Proxy({}, { get: () => () => null });
  if (!window.SI) window.SI = () => null;
  if (!window.Portrait) window.Portrait = () => null;
  if (!window.SSidebar) window.SSidebar = () => null;
  if (!window.STopbar) window.STopbar = () => null;
})();
