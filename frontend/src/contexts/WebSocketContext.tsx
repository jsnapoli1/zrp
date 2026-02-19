import { createContext, useContext, useEffect, useCallback, type ReactNode } from "react";
import { useWebSocket, type WSEvent, type WSStatus } from "../hooks/useWebSocket";

interface WebSocketContextValue {
  status: WSStatus;
  lastEvent: WSEvent | null;
  /** Subscribe to events of a specific type (or "*" for all). Returns unsubscribe fn. */
  subscribe: (eventType: string, callback: (evt: WSEvent) => void) => () => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const ws = useWebSocket();

  return (
    <WebSocketContext.Provider value={ws}>
      {children}
    </WebSocketContext.Provider>
  );
}

/** Access the shared WebSocket connection. */
export function useWS(): WebSocketContextValue {
  const ctx = useContext(WebSocketContext);
  if (!ctx) {
    throw new Error("useWS must be used within a WebSocketProvider");
  }
  return ctx;
}

/**
 * Convenience hook: call `onEvent` whenever an event matching any of the given types fires.
 * Automatically cleans up on unmount.
 */
export function useWSSubscription(
  eventTypes: string[],
  onEvent: (evt: WSEvent) => void
) {
  const { subscribe } = useWS();

  // Stable callback ref
  const callbackRef = useCallback(onEvent, [onEvent]);

  useEffect(() => {
    const unsubs = eventTypes.map((t) => subscribe(t, callbackRef));
    return () => unsubs.forEach((u) => u());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subscribe, callbackRef, ...eventTypes]);
}
