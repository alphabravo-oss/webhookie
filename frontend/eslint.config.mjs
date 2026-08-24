import js from '@eslint/js';
import globals from 'globals';
import tseslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import pluginQuery from '@tanstack/eslint-plugin-query';

const config = [
  {
    ignores: ['dist/**', 'node_modules/**', 'src/routeTree.gen.ts'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  reactHooks.configs['recommended-latest'],
  jsxA11y.flatConfigs.recommended,
  ...pluginQuery.configs['flat/recommended'],
  {
    files: ['*.config.{js,cjs,mjs}'],
    languageOptions: {
      globals: { ...globals.node, ...globals.commonjs },
    },
  },
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
      'jsx-a11y/label-has-associated-control': 'warn',
      'jsx-a11y/no-autofocus': 'warn',
      'jsx-a11y/click-events-have-key-events': 'warn',
      'jsx-a11y/no-static-element-interactions': 'warn',
      'jsx-a11y/no-noninteractive-element-interactions': 'warn',
      'jsx-a11y/interactive-supports-focus': 'warn',
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@tanstack/react-router',
              message:
                'Do not import @tanstack/react-router directly. Use @/lib/navigation or @/lib/link instead.',
            },
          ],
        },
      ],
    },
  },
  {
    files: ['src/lib/navigation.ts', 'src/lib/link.tsx', 'src/routes/**', 'src/router.tsx', 'src/main.tsx'],
    rules: {
      'no-restricted-imports': 'off',
    },
  },
];

export default config;
