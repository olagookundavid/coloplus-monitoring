// Shared "show the name, fall back to the id" convention for every response
// field that now carries a name beside an id (DECISIONS.md 070).
export function nameOrId(name: string | undefined, id: string): string {
  return name || id
}
