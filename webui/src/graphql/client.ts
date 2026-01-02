import { ApolloClient, InMemoryCache } from '@apollo/client/core'
import { setContext } from '@apollo/client/link/context'
import UploadHttpLink from 'apollo-upload-client/UploadHttpLink.mjs'
import { config } from '../config'

const uploadLink = new UploadHttpLink({
  uri: `${config.apiUrl}/graphql`,
})

const authLink = setContext((_, { headers }) => {
  const token = localStorage.getItem('flecto-access-token')
  return {
    headers: {
      ...headers,
      ...(token && { [config.authHeaderName]: `Bearer ${token}` }),
    },
  }
})

export const apolloClient = new ApolloClient({
  link: authLink.concat(uploadLink),
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
    },
  },
})
