# Batch Operations Guide

## Overview

Batch operations allow users to perform actions on multiple records simultaneously, improving efficiency when managing large datasets. This feature is available across all major list views in ZRP.

## Features

- **Bulk Actions**: Delete, update status, assign, approve, etc.
- **Bulk Updates**: Edit multiple fields across selected records
- **Progress Indicators**: Real-time feedback for long-running operations
- **Confirmation Dialogs**: Safety checks for destructive actions
- **Transaction Support**: All-or-nothing updates (no partial failures in single operation)
- **Audit Trail**: All batch operations are logged in the audit system

## Architecture

### Frontend Components

#### 1. BatchSelectionProvider
Context provider that manages selection state across a page.

```tsx
import { BatchSelectionProvider } from '../contexts/BatchSelectionContext';

function MyPage() {
  return (
    <BatchSelectionProvider>
      <MyPageContent />
    </BatchSelectionProvider>
  );
}
```

#### 2. BatchCheckbox
Individual and master checkboxes for selecting items.

```tsx
import { BatchCheckbox, MasterBatchCheckbox } from '../components/BatchCheckbox';

// Master checkbox (select all)
<MasterBatchCheckbox allIds={items.map(i => i.id)} />

// Individual checkbox
<BatchCheckbox id={item.id} />
```

#### 3. BatchActionBar
Sticky action bar that appears when items are selected.

```tsx
import { BatchActionBar, type BatchAction } from '../components/BatchActionBar';

const actions: BatchAction[] = [
  {
    id: 'delete',
    label: 'Delete',
    icon: <Trash2 className="h-4 w-4" />,
    variant: 'destructive',
    requiresConfirmation: true,
    confirmationTitle: 'Delete Items?',
    confirmationMessage: 'This cannot be undone.',
    onExecute: async (ids) => {
      const result = await api.batchDelete(ids);
      return result; // { success: number, failed: number, errors?: string[] }
    },
  },
];

<BatchActionBar
  selectedCount={selectedCount}
  totalCount={totalItems}
  actions={actions}
  onClearSelection={clearSelection}
  selectedIds={Array.from(selectedItems)}
/>
```

#### 4. BulkEditDialog
Dialog for editing multiple fields across selected items.

```tsx
import { BulkEditDialog, type BulkEditField } from '../components/BulkEditDialog';

const fields: BulkEditField[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { value: 'open', label: 'Open' },
      { value: 'closed', label: 'Closed' },
    ],
  },
  {
    key: 'priority',
    label: 'Priority',
    type: 'text',
  },
];

<BulkEditDialog
  open={isOpen}
  onOpenChange={setIsOpen}
  fields={fields}
  selectedCount={selectedCount}
  onSubmit={handleBulkEdit}
  title="Bulk Edit Items"
/>
```

### Backend API

All batch operations follow a consistent pattern:

#### Batch Actions
```
POST /api/v1/{resource}/batch
Content-Type: application/json

{
  "ids": ["id1", "id2", "id3"],
  "action": "approve" | "delete" | "cancel" | etc.
}

Response:
{
  "data": {
    "success": 3,
    "failed": 0,
    "errors": []
  }
}
```

#### Batch Updates
```
POST /api/v1/{resource}/batch/update
Content-Type: application/json

{
  "ids": ["id1", "id2", "id3"],
  "updates": {
    "status": "completed",
    "priority": "high"
  }
}

Response:
{
  "data": {
    "success": 3,
    "failed": 0,
    "errors": []
  }
}
```

## Supported Operations by Entity

### Work Orders
**Batch Actions**:
- `complete`: Mark as completed
- `cancel`: Cancel work orders
- `delete`: Delete work orders

**Batch Updates**:
- `status`: open, in_progress, completed, cancelled
- `priority`: low, normal, high, urgent
- `due_date`: ISO date string

**API Client**:
```typescript
await api.batchWorkOrders(['wo-1', 'wo-2'], 'complete');
await api.bulkUpdateWorkOrders(['wo-1', 'wo-2'], { status: 'in_progress' });
```

### ECOs (Engineering Change Orders)
**Batch Actions**:
- `approve`: Approve ECOs
- `implement`: Mark as implemented
- `reject`: Reject ECOs
- `delete`: Delete ECOs

**Batch Updates**:
- `status`: draft, open, approved, implemented, rejected
- `priority`: low, medium, high, critical

**API Client**:
```typescript
await api.batchECOs(['eco-1', 'eco-2'], 'approve');
await api.batchUpdateECOs(['eco-1', 'eco-2'], { priority: 'high' });
```

### Parts
**Batch Actions**:
- `archive`: Archive parts
- `delete`: Delete parts

**Batch Updates**:
- `category`: Part category
- `status`: active, archived, obsolete
- `lifecycle`: development, production, end-of-life
- `min_stock`: Minimum stock level

**API Client**:
```typescript
await api.batchParts(['IPN-001', 'IPN-002'], 'archive');
await api.batchUpdateParts(['IPN-001', 'IPN-002'], { category: 'Resistors' });
```

### Purchase Orders
**Batch Actions**:
- `approve`: Approve POs
- `cancel`: Cancel POs
- `delete`: Delete POs

**API Client**:
```typescript
await api.batchPurchaseOrders(['po-1', 'po-2'], 'approve');
```

### Inventory
**Batch Actions**:
- `delete`: Delete inventory records

**Batch Updates**:
- `location`: Storage location
- `reorder_point`: Reorder threshold
- `reorder_qty`: Reorder quantity

**API Client**:
```typescript
await api.batchInventory(['IPN-001', 'IPN-002'], 'delete');
await api.bulkUpdateInventory(['IPN-001'], { location: 'A-12' });
```

## Implementation Guide

### Adding Batch Operations to a New Page

1. **Wrap page in BatchSelectionProvider**:
```tsx
export default function MyPage() {
  return (
    <BatchSelectionProvider>
      <MyPageContent />
    </BatchSelectionProvider>
  );
}
```

2. **Use batch selection hook**:
```tsx
function MyPageContent() {
  const { selectedItems, selectedCount, clearSelection } = useBatchSelection();
  // ...
}
```

3. **Add checkboxes to table**:
```tsx
<thead>
  <tr>
    <th><MasterBatchCheckbox allIds={items.map(i => i.id)} /></th>
    {/* other headers */}
  </tr>
</thead>
<tbody>
  {items.map(item => (
    <tr key={item.id}>
      <td><BatchCheckbox id={item.id} /></td>
      {/* other cells */}
    </tr>
  ))}
</tbody>
```

4. **Define batch actions**:
```tsx
const batchActions: BatchAction[] = [
  {
    id: 'delete',
    label: 'Delete',
    variant: 'destructive',
    requiresConfirmation: true,
    onExecute: async (ids) => {
      const result = await api.batchMyResource(ids, 'delete');
      await fetchData(); // Refresh after operation
      return result;
    },
  },
];
```

5. **Add BatchActionBar**:
```tsx
<BatchActionBar
  selectedCount={selectedCount}
  totalCount={items.length}
  actions={batchActions}
  onClearSelection={clearSelection}
  selectedIds={Array.from(selectedItems)}
/>
```

### Adding Backend Support

1. **Add batch action handler** in `handler_bulk.go`:
```go
func handleBulkMyResource(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"delete": true, "approve": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	
	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)
	
	for _, id := range req.IDs {
		// Validate item exists
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM my_table WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": not found")
			continue
		}
		
		// Execute action
		var err error
		switch req.Action {
		case "delete":
			createUndoEntry(user, "delete", "myresource", id)
			_, err = db.Exec("DELETE FROM my_table WHERE id=?", id)
		case "approve":
			_, err = db.Exec("UPDATE my_table SET status='approved' WHERE id=?", id)
		}
		
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "myresource", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	
	jsonResp(w, resp)
}
```

2. **Add batch update handler** in `handler_bulk_update.go`:
```go
var allowedMyResourceUpdateFields = map[string]bool{
	"status": true,
	"priority": true,
}

func handleBulkUpdateMyResource(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	
	// Validate fields
	for field := range req.Updates {
		if !allowedMyResourceUpdateFields[field] {
			jsonErr(w, "field not allowed: "+field, 400)
			return
		}
	}
	
	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)
	
	for _, id := range req.IDs {
		// Build and execute update
		setClauses := ""
		args := []interface{}{}
		for field, value := range req.Updates {
			if setClauses != "" {
				setClauses += ", "
			}
			setClauses += field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		
		_, err := db.Exec("UPDATE my_table SET "+setClauses+" WHERE id=?", args...)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_update", "myresource", id, fmt.Sprintf("Bulk update: %v", req.Updates))
		}
	}
	
	jsonResp(w, resp)
}
```

3. **Register routes** in `main.go`:
```go
case parts[0] == "myresource" && len(parts) == 2 && parts[1] == "batch" && r.Method == "POST":
	handleBulkMyResource(w, r)
case parts[0] == "myresource" && len(parts) == 3 && parts[1] == "batch" && parts[2] == "update" && r.Method == "POST":
	handleBulkUpdateMyResource(w, r)
```

4. **Add API client methods** in `frontend/src/lib/api.ts`:
```typescript
async batchMyResource(ids: string[], action: 'delete' | 'approve'): Promise<BatchResult> {
  return this.request('/myresource/batch', {
    method: 'POST',
    body: JSON.stringify({ ids, action }),
  });
}

async batchUpdateMyResource(ids: string[], updates: Record<string, string>): Promise<BatchResult> {
  return this.request('/myresource/batch/update', {
    method: 'POST',
    body: JSON.stringify({ ids, updates }),
  });
}
```

## Best Practices

1. **Always use confirmation dialogs for destructive actions**
2. **Provide clear feedback** with success/error toasts
3. **Refresh data after batch operations** complete
4. **Clear selection** after successful operations
5. **Log all batch operations** to the audit trail
6. **Validate permissions** on the backend for each item
7. **Use transactions** where possible for consistency
8. **Limit batch size** to prevent timeout (recommend max 1000 items)

## Performance Considerations

- Batch operations process items sequentially for better error tracking
- For very large batches (100+ items), progress indicators are shown
- Backend validates each item individually (no partial updates on validation failure)
- All audit logging happens in the same transaction as the update

## Security

- All batch operations require authentication
- Permission checks are performed for each item in the batch
- Audit logs capture the user, action, and all affected items
- Sensitive operations (delete, approve) require confirmation

## Testing

See `frontend/src/examples/` for complete working examples:
- `WorkOrdersWithBatch.tsx` - Full implementation with all batch features
- `ECOsWithBatch.tsx` - ECO-specific batch operations

## Troubleshooting

**Batch action not appearing**: Ensure you've wrapped the page in `BatchSelectionProvider`

**Items not selectable**: Check that checkboxes are using the correct IDs

**Backend errors**: Verify route registration in `main.go` and handler implementation

**Permission errors**: Check that user has appropriate permissions for the action
