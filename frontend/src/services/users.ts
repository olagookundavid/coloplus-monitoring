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

// GET /users and GET /users/:id both return this shape -- the list endpoint
// batch-loads every row's site assignments in one query server-side, so
// there's no separate per-user fetch needed to get them.
interface UserDetailDTO extends UserDTO {
  site_ids: string[]
}

function mapManagedUserDetail(dto: UserDetailDTO): ManagedUser {
  return {
    id: dto.id,
    email: dto.email,
    name: dto.full_name,
    role: dto.role as StaffRole,
    isActive: dto.is_active,
    invited: dto.invited,
    mfaEnabled: dto.mfa_enabled,
    lastLoginAt: dto.last_login_at,
    siteIds: dto.site_ids,
  }
}

export async function fetchUsers(): Promise<ManagedUser[]> {
  try {
    const { data } = await getApi().get<{ users: UserDetailDTO[] }>('/users')
    return data.users.map(mapManagedUserDetail)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load users.')
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
