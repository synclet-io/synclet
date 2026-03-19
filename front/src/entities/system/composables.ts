import { useQuery } from '@tanstack/vue-query'
import * as systemApi from './api'

export function useSystemInfo() {
  return useQuery({
    queryKey: ['system', 'info'],
    queryFn: () => systemApi.getSystemInfo(),
    staleTime: Infinity, // execution mode doesn't change at runtime
  })
}
