import axios, { type AxiosInstance } from 'axios'

import { env } from '../config'

/**
 * The one axios instance for the whole app (architecture standard §4).
 * Not called by any service yet — auth.ts is mock-backed until the
 * backend ships a real auth feature (see docs/DECISIONS.md).
 *
 * Built lazily, on first use, not at module load: this module is reached
 * through the services/ barrel from code that's in the eager bundle
 * (Login.tsx -> useLoginMutation.ts -> '../services'), so constructing the
 * client at the top level would require VITE_API_BASE_URL in every
 * environment just to import this file — even one where nothing has
 * called the real backend yet.
 */
let instance: AxiosInstance | null = null

export function getApi(): AxiosInstance {
  if (!instance) {
    instance = axios.create({ baseURL: env.apiBaseUrl, withCredentials: true })
  }
  return instance
}
