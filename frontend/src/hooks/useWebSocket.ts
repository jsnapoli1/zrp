import { useEffect, useRef, useCallback, useState } from "react";

export interface WSEvent {
  type: string;   // e.g. "eco_updated", "part_created"
  id: string | number;
  action: string; // "create", "update", "delete"
}

export type WSStatus = "connecting" | "connected" | "disconnected";

interface UseWebSocketOptions {
  /** Auto-reconnect on disconnect (default: true) */
  reconnect?: boolean;
  /** Max reconnect delay in ms (default: 30000) */
  maxDelay?: number;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const { reconnect = true, maxDelay = 30000 } = options;
  const [status, setStatus] = useState<WSStatus>("disconnected");
  const [lastEvent, setLastEvent] = useState<WSEvent | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const retriesRef = useRef(0);
  const mountedRef = useRef(true);
  const listenersRef = useRef<Map<string, Set<(evt: WSEvent) => void>>>(new Map());

  const connect = useCallback(() => {
    if (!mountedRef.current) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${window.location.host}/api/v1/ws`;

    setStatus("connecting");
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      if (!mountedRef.current) { ws.close(); return; }
      setStatus("connected");
      retriesRef.current = 0;
    };

    ws.onmessage = (msg) => {
      try {
        const evt: WSEvent = JSON.parse(msg.data);
        setLastEvent(evt);

        // Notify type-specific listeners
        const typeListeners = listenersRef.current.get(evt.type);
        if (typeListeners) {
          typeListeners.forEach((fn) => fn(evt));
        }
        // Notify wildcard listeners
        const wildcard = listenersRef.current.get("*");
        if (wildcard) {
          wildcard.forEach((fn) => fn(evt));
        }
      } catch {
        // ignore non-JSON messages
      }
    };

    ws.onclose = () => {
      if (!mountedRef.current) return;
      setStatus("disconnected");
      if (reconnect) {
        const delay = Math.min(1000 * 2 ** retriesRef.current, maxDelay);
        retriesRef.current++;
        setTimeout(connect, delay);
      }
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [reconnect, maxDelay]);

  useEffect(() => {
    mountedRef.current = true;
    connect();
    return () => {
      mountedRef.current = false;
      wsRef.current?.close();
    };
  }, [connect]);

  /** Subscribe to a specific event type (or "*" for all). Returns unsubscribe fn. */
  const subscribe = useCallback(
    (eventType: string, callback: (evt: WSEvent) => void) => {
      if (!listenersRef.current.has(eventType)) {
        listenersRef.current.set(eventType, new Set());
      }
      listenersRef.current.get(eventType)!.add(callback);
      return () => {
        listenersRef.current.get(eventType)?.delete(callback);
      };
    },
    []
  );

  return { status, lastEvent, subscribe };
}
