/// <reference types="vite/client" />

/// <reference types="./wails.d.ts" />

declare module '*.vue' {
    import type {DefineComponent} from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}
