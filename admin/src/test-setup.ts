import { configure } from "@testing-library/react"

// The first RouterProvider load in each test file takes 1-2.5s cold (about
// 50ms afterwards), which blows RTL's 1s default for findBy*/waitFor. Stay
// under vitest's 5s test timeout so a real failure still reports its own message.
configure({ asyncUtilTimeout: 4000 })

// jsdom has no ResizeObserver; the virtual grid reads layout through one. Tests
// that need layout stub `getBoundingClientRect` and fire callbacks themselves.
// jsdom has no matchMedia; the theme provider reads the system colour scheme.
window.matchMedia ??= (query: string): MediaQueryList => ({
  matches: false,
  media: query,
  onchange: null,
  addListener() {},
  removeListener() {},
  addEventListener() {},
  removeEventListener() {},
  dispatchEvent: () => false,
})

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
}
