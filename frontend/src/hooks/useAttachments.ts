import { useQuery } from '@tanstack/react-query'

import { fetchAttachments } from '../services'
import type { AttachmentOwnerType } from '../types'

export const ATTACHMENTS_QUERY_KEY = ['attachments']

export function useAttachments(ownerType: AttachmentOwnerType, ownerId: string | undefined) {
  return useQuery({
    queryKey: [...ATTACHMENTS_QUERY_KEY, ownerType, ownerId],
    queryFn: () => fetchAttachments(ownerType, ownerId as string),
    enabled: Boolean(ownerId),
  })
}
