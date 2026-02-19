import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// Mock the WebSocket hook globally so WebSocketProvider works in tests
vi.mock("../hooks/useWebSocket", () => ({
  useWebSocket: () => ({
    status: "connected" as const,
    lastEvent: null,
    subscribe: () => () => {},
  }),
}));

afterEach(() => {
  cleanup();
});

// Mock window.matchMedia
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Mock ResizeObserver
class ResizeObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}
window.ResizeObserver = ResizeObserverMock as any;

// Mock IntersectionObserver
class IntersectionObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}
window.IntersectionObserver = IntersectionObserverMock as any;

// Mock scrollIntoView
Element.prototype.scrollIntoView = vi.fn();

// Suppress console.error for expected test errors
const originalError = console.error;
console.error = (...args: any[]) => {
  if (
    typeof args[0] === "string" &&
    (args[0].includes("act(") || args[0].includes("Not implemented: HTMLFormElement"))
  ) {
    return;
  }
  originalError.call(console, ...args);
};
