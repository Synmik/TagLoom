// Wails v2 runtime type declarations — matches generated bindings in wailsjs/go/app/App.js
interface WailsRuntime {
  go: {
    app: {
      App: Record<string, (...args: any[]) => any>;
    };
  };
}

declare global {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface Window extends WailsRuntime {}
}

export {};
