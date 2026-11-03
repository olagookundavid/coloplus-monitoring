import { ArrowDownTrayIcon, ClipboardDocumentIcon } from '@heroicons/react/24/outline'

import { downloadTextFile, toast } from '../../utils'
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
  async function handleCopyAll() {
    try {
      await navigator.clipboard.writeText(codes.join('\n'))
      toast.success('Recovery codes copied.')
    } catch {
      toast.error('Could not copy to clipboard.')
    }
  }

  function handleDownload() {
    downloadTextFile('coloplus-recovery-codes.txt', codes.join('\n') + '\n', 'text/plain;charset=utf-8;')
  }

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
      <div className="flex flex-col gap-3">
        <div className="grid grid-cols-2 gap-2 rounded-lg border border-border bg-surface-subtle p-4 font-mono text-sm text-text">
          {codes.map((code) => (
            <span key={code}>{code}</span>
          ))}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            icon={ClipboardDocumentIcon}
            onClick={handleCopyAll}
            fullWidth
          >
            Copy all
          </Button>
          <Button
            variant="outline"
            size="sm"
            icon={ArrowDownTrayIcon}
            onClick={handleDownload}
            fullWidth
          >
            Download .txt
          </Button>
        </div>
      </div>
    </Modal>
  )
}
