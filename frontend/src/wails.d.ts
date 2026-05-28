// Wails v2 runtime type declarations
interface WailsRuntime {
  go: {
    main: {
      app: Record<string, (...args: any[]) => any>
    }
  }
}

declare global {
  interface Window extends WailsRuntime {}
}

export {}
