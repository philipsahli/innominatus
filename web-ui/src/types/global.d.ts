// Global type declarations

declare global {
  interface Window {
    __addActivityEvent?: (event: any) => void;
  }
}

export {};
