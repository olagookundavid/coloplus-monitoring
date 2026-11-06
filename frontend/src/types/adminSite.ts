// A minimal site reference -- mirrors dto.SiteRefDTO, the { id, name } shape
// GET /users sends per assigned site (beside the plain siteIds list).
export interface SiteRef {
  id: string
  name: string
}

// The ops-facing view of a site, real-backend-shaped — distinct from
// types/site.ts's `Site`, which is the customer portal's mock-backed shape
// (status/powerSource as display strings, no real DTO behind it yet).
export interface AdminSite {
  id: string
  airtelCode: string
  coloplusCode: string
  name: string
  siteType: string
  region: string
  state: string
}
