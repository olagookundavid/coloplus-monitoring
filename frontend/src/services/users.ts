import type { ManagedUser, StaffRole } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface UserDTO {
  id: string
  email: string
  full_name: string
  role: string
  is_active: boolean
  mfa_enabled: boolean
  invited: boolean
  last_login_at?: string
}

interface UserDetailDTO extends UserDTO {
  site_ids: string[]
}

function mapManagedUser(dto: UserDTO): ManagedUser {
  return {
    id: dto.id,
    email: dto.email,
    name: dto.full_name,
    role: dto.role as StaffRole,
    isActive: dto.is_active,
    invited: dto.invited,
    mfaEnabled: dto.mfa_enabled,
    lastLoginAt: dto.last_login_at,
  }
}

function mapManagedUserDetail(dto: UserDetailDTO): ManagedUser {
  return { ...mapManagedUser(dto), siteIds: dto.site_ids }
}

export async function fetchUsers(): Promise<ManagedUser[]> {
  try {
    const { data } = await getApi().get<{ users: UserDTO[] }>('/users')
    return data.users.map(mapManagedUser)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load users.')
  }
}

// The list endpoint doesn't return site_ids -- only this per-user call does.
export async function fetchUser(id: string): Promise<ManagedUser> {
  try {
    const { data } = await getApi().get<UserDetailDTO>(`/users/${id}`)
    return mapManagedUserDetail(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load that user.')
  }
}

export interface CreateUserRequest {
  email: string
  name: string
  role: StaffRole
}

// No password, no site_ids: leaving the password out is what triggers the
// real invite-email flow; sites are assigned afterward on the Site
// assignments page (PATCH /users/:id replaces the whole list anyway, so
// there's no granular "assign at invite time" call to make here).
export async function createUser(request: CreateUserRequest): Promise<ManagedUser> {
  try {
    const { data } = await getApi().post<UserDetailDTO>('/users', {
      email: request.email,
      full_name: request.name,
      role: request.role,
      site_ids: [],
    })
    return mapManagedUserDetail(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not invite that user.')
  }
}

// GET /users has no site_ids -- this fetches per-user detail for every
// non-admin user (admin can never hold a site assignment at all; the
// backend's checkSites rejects it) to get real site counts/ids. Fine at
// this scale: a handful of Coloplus/Airtel staff, not thousands.
export async function fetchUsersWithSites(): Promise<ManagedUser[]> {
  const all = await fetchUsers()
  return Promise.all(all.map((u) => (u.role === 'admin' ? u : fetchUser(u.id))))
}

// PATCH /users/:id replaces the whole site_ids list for that user -- there
// is no granular "assign this one site" call.
export async function updateUserSites(id: string, siteIds: string[]): Promise<ManagedUser> {
  try {
    const { data } = await getApi().patch<UserDetailDTO>(`/users/${id}`, { site_ids: siteIds })
    return mapManagedUserDetail(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not update that assignment.')
  }
}

export interface UpdateUserPatch {
  role?: StaffRole
  isActive?: boolean
  siteIds?: string[]
}

// Only the fields present are sent, matching the backend's "only the
// fields present are changed" PATCH semantics.
export async function updateUser(id: string, patch: UpdateUserPatch): Promise<ManagedUser> {
  try {
    const { data } = await getApi().patch<UserDetailDTO>(`/users/${id}`, {
      role: patch.role,
      is_active: patch.isActive,
      site_ids: patch.siteIds,
    })
    return mapManagedUserDetail(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not update that account.')
  }
}

export async function resetUserMfa(id: string): Promise<void> {
  try {
    await getApi().post(`/users/${id}/reset-mfa`, {})
  } catch (error) {
    throw mapAxiosError(error, "Could not reset that person's two-step verification.")
  }
}
