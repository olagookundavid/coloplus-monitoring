import { useMutation, useQueryClient } from '@tanstack/react-query'

import { uploadAndRegisterAttachment } from '../services'
import type { UploadAttachmentInput } from '../services'
import { ATTACHMENTS_QUERY_KEY } from './useAttachments'

export function useUploadAttachmentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: UploadAttachmentInput) => uploadAndRegisterAttachment(input),
    onSuccess: (_attachment, input) => {
      queryClient.invalidateQueries({
        queryKey: [...ATTACHMENTS_QUERY_KEY, input.ownerType, input.ownerId],
      })
    },
  })
}
