import logoFullRed from '../../assets/images/coloplus-red-1.png'
import logoMarkRed from '../../assets/images/coloplus-mark-red.png'
import logoFullWhite from '../../assets/images/coloplus-white-2.png'

export type LogoVariant = 'full' | 'mark'
export type LogoTone = 'light' | 'dark'

export interface LogoProps {
  /** full: wordmark + icon lockup. mark: icon only, for compact square slots. */
  variant?: LogoVariant
  /** Which background this sits on -- light: red mark/lockup. dark: white lockup
      (no icon-only white asset exists yet; dark + mark falls back to the full
      white lockup). */
  tone?: LogoTone
  className?: string
}

const SOURCES: Record<LogoTone, Record<LogoVariant, string>> = {
  light: { full: logoFullRed, mark: logoMarkRed },
  dark: { full: logoFullWhite, mark: logoFullWhite },
}

// The one place the Coloplus brand mark's image assets are referenced --
// every surface that needs a logo renders this, never its own <img>.
export function Logo({ variant = 'full', tone = 'light', className = '' }: LogoProps) {
  return <img src={SOURCES[tone][variant]} alt="Coloplus" className={`w-auto ${className}`} />
}
