import { useMutation, useQueryClient } from '@tanstack/react-query'

import { derivePowerStatus } from './useDailyReportForm'
import { MY_REPORTS_QUERY_KEY } from './useMyReports'
import { saveDailyReportDraft, startReportCorrection, submitDailyReport } from '../services'
import type { PowerStatus } from '../types'

export type ReportCorrectionField = 'generator_hours' | 'power_status'

export interface RequestReportCorrectionInput {
  reportId: string
  field: ReportCorrectionField
  correctedValue: string
  reason: string
}

// Opens a correction revision, applies the one field the reporter changed,
// and submits it in one go -- three calls to the existing report endpoints
// (StartCorrection -> save -> submit), no new backend work. If the save or
// submit step fails after the revision was created, the orphaned draft
// still surfaces under "Needs your action" (utils/corrections.ts), so
// nothing is silently lost -- the mutation still invalidates on error so
// that recovery row appears immediately.
export function useRequestReportCorrectionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ reportId, field, correctedValue, reason }: RequestReportCorrectionInput) => {
      const revision = await startReportCorrection(reportId, reason)
      const powerStatus =
        field === 'power_status'
          ? (correctedValue as PowerStatus)
          : (derivePowerStatus(revision.intervals) ?? 'served')
      const generatorHours =
        field === 'generator_hours' ? Number(correctedValue) : (revision.generatorHours ?? null)
      const saved = await saveDailyReportDraft(revision.id, revision.rowVersion, {
        powerStatus,
        generatorHours,
        notes: revision.notes,
      })
      return submitDailyReport(saved.id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MY_REPORTS_QUERY_KEY })
    },
    onError: () => {
      queryClient.invalidateQueries({ queryKey: MY_REPORTS_QUERY_KEY })
    },
  })
}
