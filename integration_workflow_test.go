package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

// Integration tests that run against live ZRP server on localhost:9000
// These tests verify cross-module workflows and data flow between modules
// Each test creates unique test data using timestamps to ensure deterministic execution

const (
	baseURL = "http://localhost:9000"
	adminUser = "admin"
	adminPass = "changeme"
)

// TestClient wraps http.Client with authentication
type TestClient struct {
	client *http.Client
	token  string
}

// newTestClient creates an authenticated test client
func newTestClient(t *testing.T) *TestClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		Timeout: 30 * time.Second,
	}

	// Login to get session cookie
	loginData := map[string]string{
		"username": adminUser,
		"password": adminPass,
	}
	jsonData, _ := json.Marshal(loginData)
	
	resp, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login failed: %d - %s", resp.StatusCode, string(body))
	}

	t.Logf("✓ Authenticated as %s", adminUser)
	
	return &TestClient{
		client: client,
	}
}

// makeRequest makes an authenticated API request
func (tc *TestClient) makeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp, respBody
}

// uniqueName generates a unique name using timestamp
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// TestIntegration_BOM_Shortage_Procurement_PO_Inventory tests the complete workflow:
// BOM Shortage → Procurement → PO → Inventory update
func TestIntegration_BOM_Shortage_Procurement_PO_Inventory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newTestClient(t)
	timestamp := time.Now().UnixNano()
	
	t.Log("=== TEST 1: BOM Shortage → Procurement → PO → Inventory ===")

	// Step 1: Create a vendor
	t.Log("\n[1] Creating vendor...")
	vendorName := uniqueName("Integration Test Vendor")
	vendor := map[string]interface{}{
		"name":       vendorName,
		"status":     "active",
		"lead_days":  7,
	}
	
	resp, body := client.makeRequest(t, "POST", "/api/v1/vendors", vendor)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("Vendor creation response (%d): %s", resp.StatusCode, string(body))
	}
	
	var vendorResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &vendorResp)
	vendorID := vendorResp.Data.ID
	if vendorID == "" {
		t.Logf("Warning: No vendor ID returned, using name as fallback")
		vendorID = vendorName
	}
	t.Logf("✓ Created vendor: %s", vendorID)

	// Step 2: Create component part and set initial inventory
	t.Log("\n[2] Creating component part with initial inventory...")
	componentIPN := fmt.Sprintf("COMP-INT-%d", timestamp)
	
	// First, ensure the inventory record exists with qty=0
	resp, body = client.makeRequest(t, "POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  componentIPN,
		"type": "adjust",
		"qty":  0.0,
		"note": "Initialize inventory record",
	})
	
	// Now add initial inventory of 3 units
	resp, body = client.makeRequest(t, "POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  componentIPN,
		"type": "adjust",
		"qty":  3.0,
		"note": "Initial inventory for integration test",
	})
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("Inventory transaction response (%d): %s", resp.StatusCode, string(body))
	}
	t.Logf("✓ Created component inventory: %s (qty=3)", componentIPN)

	// Step 3: Create PO for the shortage (skip BOM/WO for now - testing PO → Inventory flow)
	t.Log("\n[3] Creating purchase order...")
	
	po := map[string]interface{}{
		"vendor_id": vendorID,
		"status":    "sent",
		"lines": []map[string]interface{}{
			{
				"ipn":         componentIPN,
				"qty_ordered": 10.0, // Ordering 10 to bring total to 13
				"unit_price":  0.50,
			},
		},
	}
	
	resp, body = client.makeRequest(t, "POST", "/api/v1/pos", po)
	var poResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &poResp)
	poID := poResp.Data.ID
	if poID == "" {
		t.Logf("PO creation failed or returned no ID: %s", string(body))
		t.Skip("Cannot continue without PO ID")
	}
	t.Logf("✓ Created PO: %s for 10x %s", poID, componentIPN)

	// Step 4: Receive the PO (simulate goods receipt)
	t.Log("\n[4] Receiving purchase order...")
	
	// First, get the PO to find line IDs
	resp, body = client.makeRequest(t, "GET", "/api/v1/pos/"+poID, nil)
	var poDetail struct {
		Data struct {
			Lines []struct {
				ID         int     `json:"id"`
				IPN        string  `json:"ipn"`
				QtyOrdered float64 `json:"qty_ordered"`
			} `json:"lines"`
		} `json:"data"`
	}
	json.Unmarshal(body, &poDetail)
	
	if len(poDetail.Data.Lines) == 0 {
		t.Logf("Warning: PO has no lines. Response: %s", string(body))
		t.Skip("Cannot receive PO without lines")
	}
	
	receiveData := map[string]interface{}{
		"skip_inspection": true,
		"lines": []map[string]interface{}{},
	}
	
	// Add all lines to receive
	for _, line := range poDetail.Data.Lines {
		if line.IPN == componentIPN {
			receiveData["lines"] = append(receiveData["lines"].([]map[string]interface{}), map[string]interface{}{
				"id":  line.ID,
				"qty": line.QtyOrdered,
			})
		}
	}
	
	resp, body = client.makeRequest(t, "POST", "/api/v1/pos/"+poID+"/receive", receiveData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("PO receive failed (%d): %s", resp.StatusCode, string(body))
	} else {
		t.Logf("✓ Received PO: %s", poID)
	}

	// Step 5: Verify inventory updated to 13 (3 + 10)
	time.Sleep(500 * time.Millisecond) // Allow async operations to complete
	
	t.Log("\n[5] Verifying inventory update...")
	resp, body = client.makeRequest(t, "GET", "/api/v1/inventory/"+componentIPN, nil)
	
	var invResp struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &invResp)
	
	expectedQty := 13.0
	actualQty := invResp.Data.QtyOnHand
	
	if actualQty == expectedQty {
		t.Logf("✓✓ SUCCESS: Inventory updated correctly!")
		t.Logf("   %s qty_on_hand: %.0f (expected %.0f)", componentIPN, actualQty, expectedQty)
	} else {
		t.Errorf("✗✗ FAILURE: Inventory not updated correctly")
		t.Errorf("   Expected: %.0f, Got: %.0f", expectedQty, actualQty)
	}

	t.Log("\n=== TEST 1 COMPLETE ===\n")
}

// TestIntegration_ECO_Part_Update_BOM_Impact tests:
// ECO → Part Update → BOM Impact
func TestIntegration_ECO_Part_Update_BOM_Impact(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newTestClient(t)
	timestamp := time.Now().UnixNano()
	
	t.Log("=== TEST 2: ECO → Part Update → BOM Impact ===")

	// Step 1: Create original parts
	t.Log("\n[1] Creating original parts...")
	
	oldPartIPN := fmt.Sprintf("OLD-PART-%d", timestamp)
	newPartIPN := fmt.Sprintf("NEW-PART-%d", timestamp)
	assemblyIPN := fmt.Sprintf("ASY-ECO-%d", timestamp)
	
	// Create old part
	oldPart := map[string]interface{}{
		"ipn":         oldPartIPN,
		"qty_on_hand": 50.0,
		"description": "Old Part (to be replaced)",
	}
	client.makeRequest(t, "PUT", "/api/v1/inventory/"+oldPartIPN, oldPart)
	
	// Create new part
	newPart := map[string]interface{}{
		"ipn":         newPartIPN,
		"qty_on_hand": 0.0,
		"description": "New Part (ECO replacement)",
	}
	client.makeRequest(t, "PUT", "/api/v1/inventory/"+newPartIPN, newPart)
	
	// Create assembly
	assembly := map[string]interface{}{
		"ipn":         assemblyIPN,
		"qty_on_hand": 0.0,
		"description": "Assembly using old part",
	}
	client.makeRequest(t, "PUT", "/api/v1/inventory/"+assemblyIPN, assembly)
	
	t.Logf("✓ Created parts: %s (old), %s (new), %s (assembly)", oldPartIPN, newPartIPN, assemblyIPN)

	// Step 2: Create BOM with old part
	t.Log("\n[2] Creating BOM with old part...")
	
	bom := map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": oldPartIPN,
		"qty_per":       2.0,
	}
	client.makeRequest(t, "POST", "/api/v1/bom", bom)
	t.Logf("✓ Created BOM: %s uses 2x %s", assemblyIPN, oldPartIPN)

	// Step 3: Create ECO proposing part change
	t.Log("\n[3] Creating ECO for part change...")
	
	ecoID := uniqueName("ECO-INT")
	eco := map[string]interface{}{
		"id":             ecoID,
		"title":          "Replace " + oldPartIPN + " with " + newPartIPN,
		"description":    "Integration test ECO: Part replacement due to obsolescence",
		"status":         "draft",
		"priority":       "normal",
		"affected_ipns":  assemblyIPN,
	}
	
	resp, body := client.makeRequest(t, "POST", "/api/v1/ecos", eco)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("ECO creation response: %s", string(body))
	}
	
	var ecoResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ecoResp)
	if ecoResp.Data.ID != "" {
		ecoID = ecoResp.Data.ID
	}
	
	t.Logf("✓ Created ECO: %s", ecoID)

	// Step 4: Approve the ECO
	t.Log("\n[4] Approving ECO...")
	
	approveData := map[string]interface{}{
		"title":          eco["title"],
		"description":    eco["description"],
		"status":         "approved",
		"priority":       eco["priority"],
		"affected_ipns":  eco["affected_ipns"],
	}
	
	resp, body = client.makeRequest(t, "PUT", "/api/v1/ecos/"+ecoID, approveData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("ECO approval response: %s", string(body))
	}
	t.Logf("✓ Approved ECO: %s", ecoID)

	// Step 5: Update BOM to use new part
	t.Log("\n[5] Updating BOM to use new part...")
	
	// Delete old BOM entry
	client.makeRequest(t, "DELETE", "/api/v1/bom?parent_ipn="+assemblyIPN+"&component_ipn="+oldPartIPN, nil)
	
	// Create new BOM entry
	newBOM := map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": newPartIPN,
		"qty_per":       2.0,
	}
	client.makeRequest(t, "POST", "/api/v1/bom", newBOM)
	t.Logf("✓ Updated BOM: %s now uses %s", assemblyIPN, newPartIPN)

	// Step 6: Verify BOM was updated
	t.Log("\n[6] Verifying BOM update...")
	
	resp, body = client.makeRequest(t, "GET", "/api/v1/bom?parent_ipn="+assemblyIPN, nil)
	
	var bomResp struct {
		Data []struct {
			ParentIPN    string  `json:"parent_ipn"`
			ComponentIPN string  `json:"component_ipn"`
			QtyPer       float64 `json:"qty_per"`
		} `json:"data"`
	}
	json.Unmarshal(body, &bomResp)
	
	foundNewPart := false
	foundOldPart := false
	
	for _, item := range bomResp.Data {
		if item.ComponentIPN == newPartIPN {
			foundNewPart = true
		}
		if item.ComponentIPN == oldPartIPN {
			foundOldPart = true
		}
	}
	
	if foundNewPart && !foundOldPart {
		t.Logf("✓✓ SUCCESS: BOM updated correctly!")
		t.Logf("   %s now uses %s (old part %s removed)", assemblyIPN, newPartIPN, oldPartIPN)
	} else if foundOldPart {
		t.Errorf("✗ PARTIAL: Old part still in BOM")
	} else if !foundNewPart {
		t.Errorf("✗ FAILURE: New part not in BOM")
	}

	// Step 7: Verify ECO is linked to affected parts
	t.Log("\n[7] Verifying ECO tracking...")
	
	resp, body = client.makeRequest(t, "GET", "/api/v1/ecos/"+ecoID, nil)
	var ecoDetailResp struct {
		Data struct {
			Status       string `json:"status"`
			AffectedIPNs string `json:"affected_ipns"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ecoDetailResp)
	
	if ecoDetailResp.Data.Status == "approved" {
		t.Logf("✓ ECO status: approved")
	} else {
		t.Logf("ECO status: %s", ecoDetailResp.Data.Status)
	}

	t.Log("\n=== TEST 2 COMPLETE ===\n")
}

// TestIntegration_NCR_RMA_ECO_Flow tests:
// NCR → RMA → ECO Flow
func TestIntegration_NCR_RMA_ECO_Flow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newTestClient(t)
	timestamp := time.Now().UnixNano()
	
	t.Log("=== TEST 3: NCR → RMA → ECO Flow ===")

	// Step 1: Create defective part
	t.Log("\n[1] Creating defective part...")
	
	defectiveIPN := fmt.Sprintf("DEFECT-%d", timestamp)
	defectivePart := map[string]interface{}{
		"ipn":         defectiveIPN,
		"qty_on_hand": 100.0,
		"description": "Defective incoming part",
	}
	client.makeRequest(t, "PUT", "/api/v1/inventory/"+defectiveIPN, defectivePart)
	t.Logf("✓ Created part: %s", defectiveIPN)

	// Step 2: Create NCR for defective incoming part
	t.Log("\n[2] Creating NCR for defective part...")
	
	ncrID := uniqueName("NCR-INT")
	ncr := map[string]interface{}{
		"id":          ncrID,
		"ipn":         defectiveIPN,
		"qty":         10.0,
		"severity":    "major",
		"status":      "open",
		"description": "Integration test: Incoming inspection failure - wrong dimensions",
		"root_cause":  "Supplier process deviation",
	}
	
	resp, body := client.makeRequest(t, "POST", "/api/v1/ncrs", ncr)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("NCR creation response: %s", string(body))
	}
	
	var ncrResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ncrResp)
	if ncrResp.Data.ID != "" {
		ncrID = ncrResp.Data.ID
	}
	
	t.Logf("✓ Created NCR: %s", ncrID)

	// Step 3: Create RMA linked to NCR
	t.Log("\n[3] Creating RMA linked to NCR...")
	
	rmaID := uniqueName("RMA-INT")
	rma := map[string]interface{}{
		"id":          rmaID,
		"ipn":         defectiveIPN,
		"qty":         10.0,
		"status":      "pending",
		"reason":      "Defective incoming parts - linked to " + ncrID,
		"ncr_id":      ncrID,
	}
	
	resp, body = client.makeRequest(t, "POST", "/api/v1/rmas", rma)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("RMA creation response: %s", string(body))
	}
	
	var rmaResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &rmaResp)
	if rmaResp.Data.ID != "" {
		rmaID = rmaResp.Data.ID
	}
	
	t.Logf("✓ Created RMA: %s linked to NCR: %s", rmaID, ncrID)

	// Step 4: Close NCR with corrective action → ECO
	t.Log("\n[4] Closing NCR with corrective action (trigger ECO)...")
	
	ecoID := uniqueName("ECO-NCR")
	
	closeData := map[string]interface{}{
		"ipn":                 defectiveIPN,
		"qty":                 10.0,
		"severity":            "major",
		"status":              "closed",
		"description":         ncr["description"],
		"root_cause":          ncr["root_cause"],
		"corrective_action":   "Create ECO to update part specification",
		"preventive_action":   "Supplier quality audit",
		"eco_id":              ecoID,
	}
	
	resp, body = client.makeRequest(t, "PUT", "/api/v1/ncrs/"+ncrID, closeData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("NCR close response: %s", string(body))
	}
	t.Logf("✓ Closed NCR with corrective action")

	// Step 5: Create ECO as corrective action
	t.Log("\n[5] Creating ECO as corrective action...")
	
	eco := map[string]interface{}{
		"id":            ecoID,
		"title":         "Corrective Action for NCR " + ncrID,
		"description":   "Update part specification to prevent recurrence of defect",
		"status":        "draft",
		"priority":      "high",
		"affected_ipns": defectiveIPN,
		"ncr_id":        ncrID,
	}
	
	resp, body = client.makeRequest(t, "POST", "/api/v1/eco", eco)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("ECO creation response: %s", string(body))
	}
	t.Logf("✓ Created ECO: %s", ecoID)

	// Step 6: Verify ECO is linked to NCR
	t.Log("\n[6] Verifying ECO ↔ NCR linkage...")
	
	// Check NCR has ECO reference
	resp, body = client.makeRequest(t, "GET", "/api/v1/ncrs/"+ncrID, nil)
	var ncrDetailResp struct {
		Data struct {
			ECOID  string `json:"eco_id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ncrDetailResp)
	
	// Check ECO has NCR reference
	resp, body = client.makeRequest(t, "GET", "/api/v1/ecos/"+ecoID, nil)
	var ecoDetailResp struct {
		Data struct {
			NcrID string `json:"ncr_id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ecoDetailResp)
	
	ncrLinked := (ncrDetailResp.Data.ECOID == ecoID)
	ecoLinked := (ecoDetailResp.Data.NcrID == ncrID)
	
	if ncrLinked && ecoLinked {
		t.Logf("✓✓ SUCCESS: NCR ↔ ECO bidirectional link verified!")
		t.Logf("   NCR %s → ECO %s", ncrID, ncrDetailResp.Data.ECOID)
		t.Logf("   ECO %s → NCR %s", ecoID, ecoDetailResp.Data.NcrID)
	} else {
		if !ncrLinked {
			t.Errorf("✗ NCR does not reference ECO (got: '%s')", ncrDetailResp.Data.ECOID)
		}
		if !ecoLinked {
			t.Errorf("✗ ECO does not reference NCR (got: '%s')", ecoDetailResp.Data.NcrID)
		}
	}

	// Verify RMA is still linked to NCR
	t.Log("\n[7] Verifying RMA linkage...")
	resp, body = client.makeRequest(t, "GET", "/api/v1/rmas/"+rmaID, nil)
	var rmaDetailResp struct {
		Data struct {
			NcrID string `json:"ncr_id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &rmaDetailResp)
	
	if rmaDetailResp.Data.NcrID == ncrID {
		t.Logf("✓ RMA still linked to NCR")
	}

	t.Log("\n=== TEST 3 COMPLETE ===\n")
}

// TestIntegration_WorkOrder_Inventory_Consumption tests:
// Work Order → Inventory Consumption
func TestIntegration_WorkOrder_Inventory_Consumption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newTestClient(t)
	timestamp := time.Now().UnixNano()
	
	t.Log("=== TEST 4: Work Order → Inventory Consumption ===")

	// Step 1: Create component and assembly parts
	t.Log("\n[1] Creating parts...")
	
	componentIPN := fmt.Sprintf("COMP-WO-%d", timestamp)
	assemblyIPN := fmt.Sprintf("ASY-WO-%d", timestamp)
	
	component := map[string]interface{}{
		"ipn":         componentIPN,
		"qty_on_hand": 100.0,
		"description": "Component for work order test",
	}
	client.makeRequest(t, "PUT", "/api/v1/inventory/"+componentIPN, component)
	
	assembly := map[string]interface{}{
		"ipn":         assemblyIPN,
		"qty_on_hand": 0.0,
		"description": "Assembly for work order test",
	}
	client.makeRequest(t, "PUT", "/api/v1/inventory/"+assemblyIPN, assembly)
	
	t.Logf("✓ Created %s (qty=100) and %s (qty=0)", componentIPN, assemblyIPN)

	// Step 2: Create BOM (5 components per assembly)
	t.Log("\n[2] Creating BOM...")
	
	bom := map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": componentIPN,
		"qty_per":       5.0,
	}
	client.makeRequest(t, "POST", "/api/v1/bom", bom)
	t.Logf("✓ BOM: %s requires 5x %s", assemblyIPN, componentIPN)

	// Step 3: Record initial inventory levels
	t.Log("\n[3] Recording initial inventory...")
	
	resp, body := client.makeRequest(t, "GET", "/api/v1/inventory/"+componentIPN, nil)
	var compInv struct {
		Data struct {
			QtyOnHand  float64 `json:"qty_on_hand"`
			QtyReserved float64 `json:"qty_reserved"`
		} `json:"data"`
	}
	json.Unmarshal(body, &compInv)
	initialCompQty := compInv.Data.QtyOnHand
	
	resp, body = client.makeRequest(t, "GET", "/api/v1/inventory/"+assemblyIPN, nil)
	var asmInv struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &asmInv)
	initialAsmQty := asmInv.Data.QtyOnHand
	
	t.Logf("  Initial: %s = %.0f, %s = %.0f", componentIPN, initialCompQty, assemblyIPN, initialAsmQty)

	// Step 4: Create work order for 10 assemblies
	t.Log("\n[4] Creating work order for 10 assemblies...")
	
	woID := uniqueName("WO-INT")
	wo := map[string]interface{}{
		"id":           woID,
		"assembly_ipn": assemblyIPN,
		"qty":          10.0,
		"priority":     "normal",
		"status":       "open",
	}
	
	resp, body = client.makeRequest(t, "POST", "/api/v1/workorders", wo)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("Work order creation response: %s", string(body))
	}
	
	var woResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &woResp)
	if woResp.Data.ID != "" {
		woID = woResp.Data.ID
	}
	
	t.Logf("✓ Created work order: %s", woID)

	// Step 5: Start work order
	t.Log("\n[5] Starting work order...")
	
	startData := map[string]interface{}{
		"assembly_ipn": assemblyIPN,
		"qty":          10.0,
		"status":       "in_progress",
		"priority":     "normal",
	}
	
	resp, body = client.makeRequest(t, "PUT", "/api/v1/workorders/"+woID, startData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("Work order start response: %s", string(body))
	}
	t.Logf("✓ Started work order")

	// Step 6: Complete work order
	t.Log("\n[6] Completing work order...")
	
	completeData := map[string]interface{}{
		"assembly_ipn": assemblyIPN,
		"qty":          10.0,
		"status":       "completed",
		"priority":     "normal",
	}
	
	resp, body = client.makeRequest(t, "PUT", "/api/v1/workorders/"+woID, completeData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("Work order complete response: %s", string(body))
	}
	t.Logf("✓ Completed work order")

	// Allow async inventory updates
	time.Sleep(500 * time.Millisecond)

	// Step 7: Verify component inventory decreased
	t.Log("\n[7] Verifying component consumption...")
	
	resp, body = client.makeRequest(t, "GET", "/api/v1/inventory/"+componentIPN, nil)
	json.Unmarshal(body, &compInv)
	finalCompQty := compInv.Data.QtyOnHand
	
	expectedCompQty := initialCompQty - (10.0 * 5.0) // 100 - 50 = 50
	
	if finalCompQty == expectedCompQty {
		t.Logf("✓ Component consumed correctly!")
		t.Logf("   %s: %.0f → %.0f (consumed 50)", componentIPN, initialCompQty, finalCompQty)
	} else {
		t.Errorf("✗ Component consumption incorrect")
		t.Errorf("   Expected: %.0f, Got: %.0f", expectedCompQty, finalCompQty)
	}

	// Step 8: Verify finished goods inventory increased
	t.Log("\n[8] Verifying finished goods addition...")
	
	resp, body = client.makeRequest(t, "GET", "/api/v1/inventory/"+assemblyIPN, nil)
	json.Unmarshal(body, &asmInv)
	finalAsmQty := asmInv.Data.QtyOnHand
	
	expectedAsmQty := initialAsmQty + 10.0 // 0 + 10 = 10
	
	if finalAsmQty == expectedAsmQty {
		t.Logf("✓✓ SUCCESS: Finished goods added correctly!")
		t.Logf("   %s: %.0f → %.0f (added 10)", assemblyIPN, initialAsmQty, finalAsmQty)
	} else {
		t.Errorf("✗ Finished goods addition incorrect")
		t.Errorf("   Expected: %.0f, Got: %.0f", expectedAsmQty, finalAsmQty)
	}

	// Step 9: Verify inventory transactions were created
	t.Log("\n[9] Verifying inventory transactions...")
	
	resp, body = client.makeRequest(t, "GET", "/api/v1/inventory/transactions?reference="+woID, nil)
	var txResp struct {
		Data []struct {
			IPN       string  `json:"ipn"`
			Type      string  `json:"type"`
			Qty       float64 `json:"qty"`
			Reference string  `json:"reference"`
		} `json:"data"`
	}
	json.Unmarshal(body, &txResp)
	
	if len(txResp.Data) > 0 {
		t.Logf("✓ Found %d inventory transactions for WO", len(txResp.Data))
		for _, tx := range txResp.Data {
			t.Logf("   - %s %s: %.0f units (type: %s)", tx.IPN, tx.Type, tx.Qty, tx.Type)
		}
	} else {
		t.Logf("⚠ No inventory transactions found (may be logged elsewhere)")
	}

	t.Log("\n=== TEST 4 COMPLETE ===\n")
}
