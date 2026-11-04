import { useMutation } from '@tanstack/react-query'

import {
  downloadCommissioningBacklogCsv,
  downloadDailySummariesCsv,
  downloadIncidentRegisterCsv,
  downloadMonthlyPerformanceCsv,
  downloadWeeklyExceptionsCsv,
} from '../services'
import type {
  CommissioningBacklogParams,
  ExportDateRange,
  MonthlyPerformanceParams,
  WeeklyExceptionsParams,
} from '../services'

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

export function useDownloadMonthlyPerformanceMutation() {
  return useMutation({
    mutationFn: (params: MonthlyPerformanceParams) => downloadMonthlyPerformanceCsv(params),
  })
}

export function useDownloadWeeklyExceptionsMutation() {
  return useMutation({
    mutationFn: (params: WeeklyExceptionsParams) => downloadWeeklyExceptionsCsv(params),
  })
}

export function useDownloadCommissioningBacklogMutation() {
  return useMutation({
    mutationFn: (params: CommissioningBacklogParams) => downloadCommissioningBacklogCsv(params),
  })
}
