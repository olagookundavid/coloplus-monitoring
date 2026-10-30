import { Button, Modal } from '../ui'

export interface RecoveryCodesModalProps {
  open: boolean
  codes: string[]
  onContinue: () => void
  continuing?: boolean
}

// No `onClose`: these codes are shown exactly once, so the gate can't be
// dismissed without acknowledging them (frontend/CLAUDE.md §7).
export function RecoveryCodesModal({
  open,
  codes,
  onContinue,
  continuing = false,
}: RecoveryCodesModalProps) {
  return (
    <Modal
      open={open}
      title="Save your recovery codes"
      subtitle="Each code can be used once to sign in if you lose access to your authenticator app. Store them somewhere safe — they won't be shown again."
      footer={
        <Button fullWidth onClick={onContinue} loading={continuing}>
          Continue
        </Button>
      }
    >
      <div className="grid grid-cols-2 gap-2 rounded-lg border border-border bg-surface-subtle p-4 font-mono text-sm text-text">
        {codes.map((code) => (
          <span key={code}>{code}</span>
        ))}
      </div>
    </Modal>
  )
}
