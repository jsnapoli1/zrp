# Real-Time Collaboration Features

ZRP now includes WebSocket-based real-time collaboration features that enable multiple users to work together seamlessly.

## Features

### 1. **User Presence Tracking**
- See who else is viewing or editing the same record
- Real-time presence indicators with avatars
- Distinguish between viewers and editors
- Automatic cleanup when users disconnect

### 2. **Real-Time Updates**
- Instant notifications when records are created, updated, or deleted
- Auto-refresh data when changes are detected
- Support for all major entities (Work Orders, ECOs, Inventory, etc.)
- Graceful degradation when WebSocket is unavailable

### 3. **Collaborative Workflows**

#### Work Orders
- See who else is viewing a work order
- Real-time updates when status changes
- Live notifications when materials are kitted
- Prevent conflicting edits

#### ECOs
- Show approvers currently viewing
- Real-time status updates
- Live approval notifications
- Implementation tracking

#### Inventory
- Prevent inventory conflicts
- Real-time stock updates
- Live transaction notifications
- Low stock alerts

## Technical Implementation

### Backend

**WebSocket Endpoint**: `/api/v1/ws`

The WebSocket server maintains:
- Connected clients registry
- User presence tracking per resource
- Pub/sub event broadcasting

**Events Broadcast**:
```json
{
  "type": "work_order_updated",
  "id": "WO-1234",
  "action": "update",
  "user_id": 5,
  "user": "jsmith"
}
```

**Presence Updates**:
```json
{
  "type": "presence_update",
  "user_id": 5,
  "user": "jsmith",
  "data": {
    "resource_type": "work_order",
    "resource_id": "WO-1234",
    "action": "editing",
    "timestamp": "2026-02-19T13:30:00Z"
  }
}
```

### Frontend Integration

#### 1. Enable Presence on a Page

```tsx
import { PresenceIndicator } from "@/components/PresenceIndicator";

function WorkOrderDetail({ id }: { id: string }) {
  return (
    <div>
      <div className="flex justify-between items-center">
        <h1>Work Order {id}</h1>
        <PresenceIndicator 
          resourceType="work_order" 
          resourceId={id}
          action="viewing"
        />
      </div>
      {/* ... rest of component */}
    </div>
  );
}
```

#### 2. Subscribe to Real-Time Updates

```tsx
import { useResourceUpdates } from "@/hooks/usePresence";

function WorkOrdersList() {
  const [workOrders, setWorkOrders] = useState([]);

  // Auto-refresh when updates happen
  useResourceUpdates("work_order", ({ action, id }) => {
    if (action === "update" || action === "create") {
      // Refetch data
      fetchWorkOrders();
    }
  });

  // ... rest of component
}
```

#### 3. Report Editing Activity

```tsx
import { usePresence } from "@/hooks/usePresence";

function WorkOrderEditor({ id }: { id: string }) {
  // Report that we're editing (not just viewing)
  usePresence("work_order", id, "editing");

  return (
    <form>
      {/* ... form fields */}
    </form>
  );
}
```

## API Endpoints

### Get Presence for a Resource

```
GET /api/v1/presence?resource_type=work_order&resource_id=WO-1234
```

Response:
```json
[
  {
    "user_id": 5,
    "username": "jsmith",
    "resource_type": "work_order",
    "resource_id": "WO-1234",
    "action": "viewing",
    "timestamp": "2026-02-19T13:30:00Z"
  }
]
```

## Event Types

### Resource Events
- `<resource>_created` - New record created
- `<resource>_updated` - Record modified
- `<resource>_deleted` - Record removed
- `<resource>_approved` - Record approved (ECOs, etc.)
- `<resource>_implemented` - Record implemented (ECOs)

### System Events
- `user_joined` - User connected to WebSocket
- `user_left` - User disconnected
- `presence_update` - User viewing/editing status changed

## Supported Resource Types

- `work_order` - Work Orders
- `eco` - Engineering Change Orders
- `inventory` - Inventory Items
- `part` - Parts (future)
- `vendor` - Vendors (future)
- `procurement` - Purchase Orders (future)

## Configuration

WebSocket connection is automatic and requires no configuration. Features include:

- **Auto-reconnect**: Reconnects automatically on disconnect (exponential backoff)
- **Keep-alive**: 30-second ping/pong to maintain connection
- **Graceful degradation**: UI works without WebSocket (no real-time features)
- **Session authentication**: Uses existing session cookie

## Testing

### Test with Multiple Sessions

1. Open ZRP in two browser windows
2. Log in as different users (or use incognito mode)
3. Navigate to the same Work Order
4. Observe presence indicators showing other users
5. Edit in one window and watch updates appear in the other

### Manual WebSocket Testing

```javascript
// In browser console
const ws = new WebSocket('ws://localhost:9000/api/v1/ws');

ws.onmessage = (e) => {
  console.log('Received:', JSON.parse(e.data));
};

// Send presence update
ws.send(JSON.stringify({
  type: 'presence',
  resource_type: 'work_order',
  resource_id: 'WO-1234',
  action: 'viewing'
}));
```

## Performance Considerations

- Presence data is stored in-memory (no database overhead)
- Events are broadcast to all connected clients (efficient for <100 concurrent users)
- Automatic cleanup of stale presence on disconnect
- Buffered message sending to prevent backpressure

## Future Enhancements

- [ ] Collaborative cursor positions
- [ ] Operational transforms for concurrent editing
- [ ] User activity timeline
- [ ] Conflict resolution UI
- [ ] Presence persistence across reconnects
- [ ] Room-based broadcasts (only notify relevant users)
- [ ] Typing indicators
- [ ] Lock records being edited

## Troubleshooting

### WebSocket Won't Connect

1. Check that backend is running on expected port
2. Verify session cookie is present
3. Check browser console for connection errors
4. Ensure firewall allows WebSocket connections

### Presence Not Updating

1. Verify WebSocket is connected (check `usePresence` hook)
2. Check that `resource_type` and `resource_id` match exactly
3. Verify presence endpoint returns data: `/api/v1/presence?resource_type=...&resource_id=...`

### Events Not Broadcasting

1. Check that handlers call `broadcast()` after mutations
2. Verify event type naming convention: `${resource}_${action}d`
3. Check backend logs for broadcast errors

## Security

- WebSocket endpoint requires authenticated session
- User context extracted from session cookie
- No privileged operations via WebSocket (broadcast only)
- Presence data includes only user ID and username (no sensitive data)
