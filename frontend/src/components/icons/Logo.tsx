import logoFullRed from '../../assets/images/coloplus-red-1.png'
import logoMarkRed from '../../assets/images/coloplus-mark-red.png'
import logoFullWhite from '../../assets/images/coloplus-white-2.png'
import logoFullBlack from '../../assets/images/coloplus-black-1.png'

export type LogoVariant = 'full' | 'mark'
export type LogoTone = 'light' | 'dark' | 'black'

export interface LogoProps {
  /** full: wordmark + icon lockup. mark: icon only, for compact square slots. */
  variant?: LogoVariant
  /** Which background this sits on -- light: red mark/lockup. dark: white
      lockup. black: black lockup, the dashboard chrome and auth screen
      default. No icon-only black or white asset exists yet; dark/black +
      mark falls back to each tone's full lockup. */
  tone?: LogoTone
  className?: string
}

const SOURCES: Record<LogoTone, Record<LogoVariant, string>> = {
  light: { full: logoFullRed, mark: logoMarkRed },
  dark: { full: logoFullWhite, mark: logoFullWhite },
  black: { full: logoFullBlack, mark: logoFullBlack },
}

// The one place the Coloplus brand mark's image assets are referenced --
// every surface that needs a logo renders this, never its own <img>.
export function Logo({ variant = 'full', tone = 'light', className = '' }: LogoProps) {
  return <img src={SOURCES[tone][variant]} alt="Coloplus" className={`w-auto ${className}`} />
}
