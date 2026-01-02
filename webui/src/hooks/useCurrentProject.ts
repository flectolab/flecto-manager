import { useParams } from 'react-router-dom'
import { useQuery } from '@apollo/client/react'
import { GetProjectDocument, type GetProjectQuery } from '../generated/graphql'

export type CurrentProject = NonNullable<GetProjectQuery['project']>

export function useCurrentProject() {
  const { namespace, project } = useParams<{ namespace: string; project: string }>()

  const { data, loading, error, refetch } = useQuery(GetProjectDocument, {
    variables: {
      namespaceCode: namespace ?? '',
      projectCode: project ?? '',
    },
    skip: !namespace || !project,
  })

  return {
    project: data?.project ?? null,
    namespace: data?.project?.namespace ?? null,
    namespaceCode: namespace,
    projectCode: project,
    loading,
    error,
    refetch,
  }
}
