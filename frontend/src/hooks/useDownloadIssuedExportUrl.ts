import { useMutation } from '@tanstack/react-query'

import { fetchIssuedExportDownload } from '../services'

// Mirrors useDownloadAttachment.ts exactly: the link is a short-lived
// (2 minute) signed URL that forces Content-Disposition: attachment -- a
// plain top-level navigation triggers the save, no bytes fetched here.
export function useDownloadIssuedExportUrl() {
  return useMutation({
    mutationFn: (id: string) => fetchIssuedExportDownload(id),
    onSuccess: ({ url }) => {
      window.location.assign(url)
    },
  })
}
