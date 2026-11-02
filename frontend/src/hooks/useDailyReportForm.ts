import { useState } from 'react'

import { useCreateDailyReportMutation } from './useCreateDailyReportMutation'
import { useReportDetail } from './useReportDetail'
import { useSaveDailyReportMutation } from './useSaveDailyReportMutation'
import { useSubmitDailyReportMutation } from './useSubmitDailyReportMutation'
import type { DailyReportPanelMode } from '../components/dashboard'
import type { DailyReportFormInput } from '../services'
import type { PowerStatus, SiteReportRow } from '../types'
import { toast } from '../utils'

export interface UseDailyReportFormArgs {
  site: SiteReportRow
  /** The reporting date, yyyy-mm-dd. */
  reportDate: string
  /** Present for Continue/Fix/View; absent for a fresh Report. */
  reportId?: string
  mode: DailyReportPanelMode
  /** Called after a successful save or submit -- e.g. to close a panel. */
  onSuccess?: () => void
}

// The backend doesn't persist the reporter's toggle as its own field -- it's
// derived from which interval state the day-level report actually built.
// Exported so useRequestReportCorrectionMutation can reuse it rather than
// re-deriving the same logic.
export function derivePowerStatus(intervals?: { state: PowerStatus }[]): PowerStatus | null {
  if (!intervals || intervals.length === 0) return null
  return intervals.some((iv) => iv.state === 'outage') ? 'outage' : 'served'
}

// Shared by DailyReportPanel (a SidePanel) and the Daily report page's inline
// pane -- both need the same status-derived toggle, validation and
// save/submit mutations, just different chrome around them.
export function useDailyReportForm({
  site,
  reportDate,
  reportId,
  mode,
  onSuccess,
}: UseDailyReportFormArgs) {
  const detailQuery = useReportDetail(reportId)
  const report = detailQuery.data?.report

  const [powerStatus, setPowerStatus] = useState<PowerStatus | null>(null)
  const [generatorHours, setGeneratorHours] = useState('')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState<'draft' | 'submit' | null>(null)

  // Initialises the form from the loaded report (edit/view) or blank
  // (create) the first time its data is available. Adjusting state during
  // render (not a useEffect) is this project's pattern for it -- SidePanel's
  // own open/close state does the same.
  const [initialized, setInitialized] = useState(false)
  if (!initialized && (reportId === undefined || report !== undefined)) {
    setInitialized(true)
    setPowerStatus(derivePowerStatus(report?.intervals))
    setGeneratorHours(report?.generatorHours != null ? String(report.generatorHours) : '')
    setNotes(report?.notes ?? '')
  }

  const createMutation = useCreateDailyReportMutation()
  const saveMutation = useSaveDailyReportMutation()
  const submitMutation = useSubmitDailyReportMutation()

  const readOnly = mode === 'view'

  function buildInput(): DailyReportFormInput | null {
    if (!powerStatus) {
      setError('Choose whether the site had power all day or had an outage.')
      return null
    }
    const trimmed = generatorHours.trim()
    const hours = trimmed === '' ? null : Number(trimmed)
    if (hours !== null && Number.isNaN(hours)) {
      setError('Generator runtime hours must be a number.')
      return null
    }
    if (hours !== null && (hours < 0 || hours > 24)) {
      setError('Generator runtime hours must be between 0 and 24.')
      return null
    }
    if (powerStatus === 'outage' && hours === null) {
      setError('Enter the generator runtime hours for a day with an outage.')
      return null
    }
    setError(null)
    return { powerStatus, generatorHours: hours, notes }
  }

  async function ensureReport(
    input: DailyReportFormInput
  ): Promise<{ id: string; rowVersion: number }> {
    if (report) {
      const saved = await saveMutation.mutateAsync({
        id: report.id,
        rowVersion: report.rowVersion,
        input,
      })
      return { id: saved.id, rowVersion: saved.rowVersion }
    }
    const created = await createMutation.mutateAsync({ siteId: site.siteId, reportDate, input })
    return { id: created.id, rowVersion: created.rowVersion }
  }

  async function handleSaveDraft() {
    const input = buildInput()
    if (!input) return
    setSubmitting('draft')
    try {
      await ensureReport(input)
      toast.success('Draft saved.')
      onSuccess?.()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not save this report.')
    } finally {
      setSubmitting(null)
    }
  }

  async function handleSubmit() {
    const input = buildInput()
    if (!input) return
    setSubmitting('submit')
    try {
      const { id } = await ensureReport(input)
      await submitMutation.mutateAsync(id)
      toast.success('Report submitted.')
      onSuccess?.()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not submit this report.')
    } finally {
      setSubmitting(null)
    }
  }

  return {
    detailQuery,
    report,
    powerStatus,
    setPowerStatus,
    generatorHours,
    setGeneratorHours,
    notes,
    setNotes,
    error,
    submitting,
    readOnly,
    handleSaveDraft,
    handleSubmit,
  }
}
