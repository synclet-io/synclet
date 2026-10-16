import antfu from '@antfu/eslint-config'

export default antfu(
  {
    vue: true,
    typescript: true,
    ignores: ['src/gen/**'],
  },
  {
    files: ['**/*.vue'],
    rules: {
      'vue/no-restricted-syntax': ['error', {
        selector: 'MemberExpression[property.name=\'message\'][object.name=/^(?:error|\\w+Error)$/]',
        message: 'Render errors with getErrorMessage() from @shared/lib/errorUtils; ConnectError.message carries a "[code]" prefix.',
      }],
    },
  },
)
