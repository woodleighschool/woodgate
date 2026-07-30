// eslint.config.js
import path from "node:path";
import { fileURLToPath } from "node:url";

import js from "@eslint/js";
import { defineConfig } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

import { createTypeScriptImportResolver } from "eslint-import-resolver-typescript";
import { createNodeResolver, importX } from "eslint-plugin-import-x";
import jsxA11y from "eslint-plugin-jsx-a11y";
import react from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import sonarjs from "eslint-plugin-sonarjs";
import unicorn from "eslint-plugin-unicorn";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig(
  { ignores: ["dist", "build", "coverage", "node_modules", "src/api/openapi.ts"] },

  { files: ["**/*.{ts,tsx}"] },

  js.configs.recommended,

  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,

  react.configs.flat.recommended,
  react.configs.flat["jsx-runtime"],

  jsxA11y.flatConfigs.strict,

  importX.flatConfigs.recommended,
  importX.flatConfigs.typescript,

  sonarjs.configs.recommended,

  unicorn.configs.recommended,

  {
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        ...globals.browser,
        ...globals.es2021,
      },
      parserOptions: {
        tsconfigRootDir: __dirname,
        projectService: true,
      },
    },
    settings: {
      react: { version: "detect" },
      "import-x/resolver-next": [createTypeScriptImportResolver(), createNodeResolver()],
    },
    plugins: {
      "react-hooks": reactHooks,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,

      "no-console": "error",
      "no-debugger": "error",

      "@typescript-eslint/consistent-type-imports": ["error", { fixStyle: "inline-type-imports" }],
      "@typescript-eslint/no-floating-promises": ["error", { ignoreVoid: false, ignoreIIFE: false }],
      "@typescript-eslint/no-misused-promises": ["error", { checksVoidReturn: true }],
      "@typescript-eslint/explicit-function-return-type": [
        "error",
        { allowExpressions: false, allowTypedFunctionExpressions: false },
      ],
      "unicorn/filename-case": "off",
    },
  },
);
