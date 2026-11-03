import { useMutation } from '@tanstack/react-query'

import { downloadDailySummariesCsv, downloadIncidentRegisterCsv } from '../services'
import type { ExportDateRange } from '../services'

export function useDownloadDailySummariesMutation() {
  return useMutation({
    mutationFn: (range: ExportDateRange) => downloadDailySummariesCsv(range),
  })
}

export function useDownloadIncidentRegisterMutation() {
  return useMutation({
    mutationFn: (range: ExportDateRange) => downloadIncidentRegisterCsv(range),
  })
}
