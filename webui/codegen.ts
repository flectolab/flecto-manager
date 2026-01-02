import type { CodegenConfig } from '@graphql-codegen/cli'

const config: CodegenConfig = {
  schema: '../graph/schema/*.graphqls',
  documents: ['src/**/*.{ts,tsx,graphql}'],
  ignoreNoDocuments: true,
  generates: {
    './src/generated/': {
      preset: 'client',
      config: {
        useTypeImports: true,
        skipTypename: true,
        enumsAsTypes: true,
      },
    },
  },
}

export default config
