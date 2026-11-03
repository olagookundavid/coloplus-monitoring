import { useMutation } from '@tanstack/react-query'

import { fetchDownloadUrl } from '../services'

// The download link is a direct, cross-origin, short-lived (2 minute)
// signed R2 URL that forces Content-Disposition: attachment -- a plain
// top-level navigation triggers the save, no need to fetch the bytes here.
export function useDownloadAttachment() {
  return useMutation({
    mutationFn: (id: string) => fetchDownloadUrl(id),
    onSuccess: ({ url }) => {
      window.location.assign(url)
    },
  })
}
