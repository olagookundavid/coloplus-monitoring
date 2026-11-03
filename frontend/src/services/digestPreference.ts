import { getApi } from './api'
import { mapAxiosError } from './auth'
import type { DigestPreference } from '../types'

export async function fetchDigestPreference(): Promise<DigestPreference> {
  try {
    const { data } = await getApi().get<DigestPreference>('/digest/preference')
    return data
  } catch (error) {
    throw mapAxiosError(error, 'Could not load your weekly summary email preference.')
  }
}

export async function updateDigestPreference(weekly: boolean): Promise<DigestPreference> {
  try {
    const { data } = await getApi().put<DigestPreference>('/digest/preference', { weekly })
    return data
  } catch (error) {
    throw mapAxiosError(error, 'Could not update your weekly summary email preference.')
  }
}
