// Shared SVG icon set for pipepush. Icons inherit color via `currentColor` and
// size via `1em`, so the existing CSS (font-size on .ic / .node / .brand .bolt)
// drives their color and size with no per-icon sizing rules. Hand-rolled to fit
// the "signal console" look — no icon-library dependency.

import type { SVGProps } from "react";

type IconProps = SVGProps<SVGSVGElement>;

// Base for 24×24 line icons: stroke follows the current text color.
function Line({ children, ...p }: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      width="1em"
      height="1em"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      {...p}
    >
      {children}
    </svg>
  );
}

/* ---------- navigation / actions ---------- */

// Activity pulse — the "Runs" tab.
export function IconActivity(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M3 12h4l3-7 4 14 3-7h4" />
    </Line>
  );
}

// Four tiles — the "Projects" tab.
export function IconGrid(p: IconProps) {
  return (
    <Line {...p}>
      <rect x="4" y="4" width="7" height="7" rx="1.5" />
      <rect x="13" y="4" width="7" height="7" rx="1.5" />
      <rect x="4" y="13" width="7" height="7" rx="1.5" />
      <rect x="13" y="13" width="7" height="7" rx="1.5" />
    </Line>
  );
}

// Cog — the "Settings" tab.
export function IconGear(p: IconProps) {
  return (
    <Line {...p}>
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
    </Line>
  );
}

// Circular arrows — refresh. Add className="spin" while loading.
export function IconRefresh(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M3 12a9 9 0 0 1 15-6.7L21 8" />
      <path d="M21 3v5h-5" />
      <path d="M21 12a9 9 0 0 1-15 6.7L3 16" />
      <path d="M3 21v-5h5" />
    </Line>
  );
}

export function IconChevronLeft(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M15 18l-6-6 6-6" />
    </Line>
  );
}

export function IconPlus(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M12 5v14M5 12h14" />
    </Line>
  );
}

export function IconSearch(p: IconProps) {
  return (
    <Line {...p}>
      <circle cx="11" cy="11" r="7" />
      <path d="M21 21l-4.3-4.3" />
    </Line>
  );
}

/* ---------- security (Login) ---------- */

export function IconLock(p: IconProps) {
  return (
    <Line {...p}>
      <rect x="5" y="11" width="14" height="10" rx="2" />
      <path d="M8 11V7a4 4 0 0 1 8 0v4" />
    </Line>
  );
}

// Face ID glyph — corner brackets + face.
export function IconFaceId(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M4 8V6a2 2 0 0 1 2-2h2" />
      <path d="M16 4h2a2 2 0 0 1 2 2v2" />
      <path d="M20 16v2a2 2 0 0 1-2 2h-2" />
      <path d="M8 20H6a2 2 0 0 1-2-2v-2" />
      <path d="M9 9v1" />
      <path d="M15 9v1" />
      <path d="M12 9v3l-1 1" />
      <path d="M9 15a3.5 3.5 0 0 0 6 0" />
    </Line>
  );
}

/* ---------- status ---------- */

export function IconCheck(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M20 6L9 17l-5-5" />
    </Line>
  );
}

export function IconCross(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M18 6L6 18M6 6l12 12" />
    </Line>
  );
}

// Warning — triangle with an exclamation.
export function IconWarning(p: IconProps) {
  return (
    <Line {...p}>
      <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
      <path d="M12 9v4" />
      <path d="M12 17h.01" />
    </Line>
  );
}

// Cancelled — circle with a slash (⊘).
export function IconSlash(p: IconProps) {
  return (
    <Line {...p}>
      <circle cx="12" cy="12" r="9" />
      <path d="M5.6 5.6l12.8 12.8" />
    </Line>
  );
}

// Running — solid pulsing dot.
export function IconDot(p: IconProps) {
  return (
    <svg viewBox="0 0 24 24" width="1em" height="1em" fill="currentColor" aria-hidden="true" {...p}>
      <circle cx="12" cy="12" r="5" />
    </svg>
  );
}

// Skipped — hollow circle (○).
export function IconCircle(p: IconProps) {
  return (
    <Line {...p}>
      <circle cx="12" cy="12" r="7" />
    </Line>
  );
}

// Maps a run status to its icon (replaces the old dingbat statusGlyph map).
// Colour comes from the surrounding .node { color: var(--s) } via currentColor.
export function StatusIcon({ status, ...p }: { status: string } & IconProps) {
  switch (status) {
    case "success":
      return <IconCheck {...p} />;
    case "failure":
      return <IconCross {...p} />;
    case "warning":
      return <IconWarning {...p} />;
    case "cancelled":
      return <IconSlash {...p} />;
    case "running":
      return <IconDot {...p} />;
    case "skipped":
      return <IconCircle {...p} />;
    default:
      return <IconDot {...p} />;
  }
}

/* ---------- brand ---------- */

// Wordmark bolt — geometry from web/assets/icon.svg (the app icon), so the
// header mark matches the home-screen icon. Indigo gradient (8E6CF7 → 6A41E0,
// same as the app icon) plus the glow from `.brand .bolt` in styles.css.
// (Variant 2 of the logo options.)
export function IconBolt(p: IconProps) {
  return (
    <svg viewBox="170 84 172 344" height="1em" aria-hidden="true" {...p}>
      <defs>
        <linearGradient id="ppbolt" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#8e6cf7" />
          <stop offset="1" stopColor="#6a41e0" />
        </linearGradient>
      </defs>
      <path d="M312 96L184 304h78l-62 112 128-176h-78z" fill="url(#ppbolt)" />
    </svg>
  );
}
