import { systemKeys } from '@shared/lib/queryKeys'
import { useQuery } from '@tanstack/vue-query'
import * as systemApi from './api'

export function useSystemInfo() {
  return useQuery({
    queryKey: systemKeys.info(),
    queryFn: () => systemApi.getSystemInfo(),
    staleTime: Infinity, // execution mode doesn't change at runtime
  })
}
