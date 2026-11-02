export interface PasswordStrengthMeterProps {
  password: string
}

// Reflects the real backend policy (domain.ValidatePassword: 12-72 chars,
// at least one letter and one digit), plus a length bonus -- not an
// arbitrary cosmetic "strength" score.
function scorePassword(password: string): number {
  let score = 0
  if (password.length >= 12) score++
  if (/[A-Za-z]/.test(password) && /[0-9]/.test(password)) score++
  if (password.length >= 16) score++
  if (/[A-Za-z]/.test(password) && /[0-9]/.test(password) && /[^A-Za-z0-9]/.test(password)) score++
  return score
}

const SEGMENT_COUNT = 4

export function PasswordStrengthMeter({ password }: PasswordStrengthMeterProps) {
  const score = password ? scorePassword(password) : 0

  return (
    <div className="flex gap-1" role="presentation">
      {Array.from({ length: SEGMENT_COUNT }, (_, index) => (
        <span
          key={index}
          className={`h-1.5 flex-1 rounded-full ${index < score ? 'bg-primary' : 'bg-border'}`}
        />
      ))}
    </div>
  )
}
