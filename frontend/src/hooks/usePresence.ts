import { useEffect, useState } from "react";
import { useWS } from "../contexts/WebSocketContext";

export interface PresenceInfo {
  user_id: number;
  username: string;
  resource_type: string;
  resource_id: string | number;
  action: string; // "viewing" | "editing"
  timestamp: string;
}

/**
 * Track and report user presence on a resource
 */
export function usePresence(
  resourceType: string,
  resourceId: string | number,
  action: "viewing" | "editing" = "viewing"
) {
  const { status, subscribe } = useWS();
  const [presence, setPresence] = useState<PresenceInfo[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    setIsConnected(status === "connected");
  }, [status]);

  // Report our presence when mounted and connected
  useEffect(() => {
    if (status !== "connected" || !resourceType || !resourceId) return;

    // Send presence update
    const ws = (window as any).__wsConnection;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(
        JSON.stringify({
          type: "presence",
          resource_type: resourceType,
          resource_id: resourceId,
          action,
        })
      );
    }

    // Fetch initial presence
    fetch(
      `/api/v1/presence?resource_type=${resourceType}&resource_id=${resourceId}`
    )
      .then((res) => res.json())
      .then((data) => {
        if (Array.isArray(data)) {
          setPresence(data);
        }
      })
      .catch(() => {
        // Graceful degradation
        setPresence([]);
      });
  }, [status, resourceType, resourceId, action]);

  // Subscribe to presence updates
  useEffect(() => {
    const unsubscribe = subscribe("presence_update", (evt) => {
      if (evt.data) {
        const info = evt.data as PresenceInfo;
        // Only update if it's for our resource
        if (
          info.resource_type === resourceType &&
          String(info.resource_id) === String(resourceId)
        ) {
          setPresence((prev) => {
            const filtered = prev.filter((p) => p.user_id !== info.user_id);
            return [...filtered, info];
          });
        }
      }
    });

    return unsubscribe;
  }, [subscribe, resourceType, resourceId]);

  // Clean up stale presence (users who left)
  useEffect(() => {
    const unsubscribe = subscribe("user_left", (evt) => {
      if (evt.user_id) {
        setPresence((prev) => prev.filter((p) => p.user_id !== evt.user_id));
      }
    });

    return unsubscribe;
  }, [subscribe]);

  const otherUsers = presence.filter(
    (p) => p.user_id !== (window as any).__currentUserId
  );

  return {
    presence: otherUsers,
    isConnected,
  };
}

/**
 * Subscribe to resource updates (create/update/delete events)
 */
export function useResourceUpdates(
  resourceType: string,
  onUpdate: (event: { action: string; id: string | number }) => void
) {
  const { subscribe } = useWS();

  useEffect(() => {
    const eventTypes = [
      `${resourceType}_created`,
      `${resourceType}_updated`,
      `${resourceType}_deleted`,
      `${resourceType}_approved`,
      `${resourceType}_implemented`,
    ];

    const unsubs = eventTypes.map((type) =>
      subscribe(type, (evt) => {
        onUpdate({ action: evt.action, id: evt.id });
      })
    );

    return () => unsubs.forEach((u) => u());
  }, [subscribe, resourceType, onUpdate]);
}
