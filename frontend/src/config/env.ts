function requireEnv(key: string, value: string | undefined): string {
  if (!value) {
    throw new Error(`Missing required environment variable: ${key}`)
  }
  return value
}

export const env = {
  // A getter, not a plain property computed once at module load: nothing
  // in the app calls the real backend yet (every service is still
  // mock-backed, see docs/DECISIONS.md), so merely importing this module
  // must not throw just because the var isn't set in an environment that
  // never needed it. The check still runs — and still throws — the moment
  // something actually reads `env.apiBaseUrl`.
  get apiBaseUrl(): string {
    return requireEnv('VITE_API_BASE_URL', import.meta.env.VITE_API_BASE_URL)
  },
}
