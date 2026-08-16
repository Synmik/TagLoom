/// <reference types="vite/client" />

/// <reference types="./wails.d.ts" />

declare module "*.vue" {
  import type { DefineComponent } from "vue";
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  const component: DefineComponent<{}, {}, any>;
  export default component;
}
