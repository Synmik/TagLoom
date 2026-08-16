import js from "@eslint/js";
import pluginVue from "eslint-plugin-vue";
import vueParser from "vue-eslint-parser";
import tseslint from "typescript-eslint";
import prettier from "eslint-config-prettier";

export default [
  // Ignore generated bindings, build output, and Wails glue code
  {
    ignores: ["dist/**", "wailsjs/**", "node_modules/**"],
  },

  // Base: recommended rules for modern JS
  js.configs.recommended,

  // Vue 3 recommended rules (includes vue-eslint-parser setup)
  ...pluginVue.configs["flat/recommended"],

  // TypeScript recommended rules
  ...tseslint.configs.recommended,

  {
    // Plain TypeScript files
    files: ["**/*.ts"],
    languageOptions: {
      parser: tseslint.parser,
    },
  },

  {
    // Vue SFCs: vue-eslint-parser on the outside, TS parser for <script>
    files: ["**/*.vue"],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },

  {
    // Shared rule tweaks for our TS/Vue source
    files: ["**/*.ts", "**/*.vue"],
    rules: {
      // Vue-specific tweaks for this project
      "vue/multi-word-component-names": "off", // App.vue is the root component
      // TS handles no-undef via type checking (vue-tsc)
      "no-undef": "off",
      // Prefer explicit types over any where reasonable, but allow any
      // in the Wails binding boundary until we tighten those types.
      "@typescript-eslint/no-explicit-any": "warn",
      // Unused vars: error in our code, allow _-prefixed intentionally unused
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
    },
  },

  // Prettier last: it turns off rules that would conflict with formatting
  prettier,
];
