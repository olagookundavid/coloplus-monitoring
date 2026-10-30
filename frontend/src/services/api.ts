import axios from 'axios'

import { env } from '../config'

/**
 * The one axios instance for the whole app (architecture standard §4).
 * Not called by any service yet — auth.ts is mock-backed until the
 * backend ships a real auth feature (see docs/DECISIONS.md).
 */
export const api = axios.create({
  baseURL: env.apiBaseUrl,
  withCredentials: true,
})
