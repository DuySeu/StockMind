// The brand geometry, defined once. A geometric S built from two elliptical
// bowls whose top terminal breaks away into a rising stroke: the letter is the
// name, the break is the move. Mirrored byte-for-byte in public/favicon.svg and
// the raster exports — change one and the set has to be regenerated.
const MARK_PATH = "M27.83 3.78 L21.04 8.9 A6.4 4.8 0 1 0 15.5 16.1 A6.4 4.8 0 1 1 9.96 23.3";
const MARK_STROKE = 3.9;

interface LogoProps {
  className?: string;
}

// Render the bare S, taking its colour from the surrounding text
export function LogoMark({ className }: LogoProps) {
  return (
    <svg viewBox="0 0 32 32" fill="none" aria-hidden="true" className={className}>
      <path d={MARK_PATH} stroke="currentColor" strokeWidth={MARK_STROKE} strokeLinejoin="round" />
    </svg>
  );
}

// Render the S on its brand tile — the standalone app icon, for chrome and empty states
export function LogoTile({ className }: LogoProps) {
  return (
    <svg viewBox="0 0 32 32" fill="none" aria-hidden="true" className={className}>
      <rect width="32" height="32" rx="8" className="fill-primary" />
      <path
        d={MARK_PATH}
        className="stroke-primary-foreground"
        strokeWidth={MARK_STROKE}
        strokeLinejoin="round"
      />
    </svg>
  );
}
