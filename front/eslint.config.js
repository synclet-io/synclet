import antfu from '@antfu/eslint-config'

export default antfu(
  {
    vue: true,
    typescript: true,
    ignores: ['src/gen/**'],
  },
  {
    files: ['**/*.vue'],
    ignores: [
      // P-10 template replacements scheduled in later EPIC-13 tasks.
      'src/pages/ConnectionLayout.vue',
      'src/pages/ConnectionSchemaChangesPage.vue',
      'src/pages/ConnectionsPage.vue',
      'src/pages/DestinationDetailPage.vue',
      'src/pages/DestinationsPage.vue',
      'src/pages/JobsPage.vue',
      'src/pages/SourceDetailPage.vue',
      'src/pages/SourcesPage.vue',
    ],
    rules: {
      'vue/no-restricted-syntax': ['error', {
        selector: 'MemberExpression[property.name=\'message\'][object.name=/^(?:error|\\w+Error)$/]',
        message: 'Render errors with getErrorMessage() from @shared/lib/errorUtils; ConnectError.message carries a "[code]" prefix.',
      }],
    },
  },
)
