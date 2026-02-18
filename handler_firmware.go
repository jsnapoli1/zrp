package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

func handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,name,version,category,status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []FirmwareCampaign
	for rows.Next() {
		var f FirmwareCampaign
		var sa, ca sql.NullString
		rows.Scan(&f.ID, &f.Name, &f.Version, &f.Category, &f.Status, &f.TargetFilter, &f.Notes, &f.CreatedAt, &sa, &ca)
		f.StartedAt = sp(sa); f.CompletedAt = sp(ca)
		items = append(items, f)
	}
	if items == nil { items = []FirmwareCampaign{} }
	jsonResp(w, items)
}

func handleGetCampaign(w http.ResponseWriter, r *http.Request, id string) {
	var f FirmwareCampaign
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,name,version,category,status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns WHERE id=?", id).
		Scan(&f.ID, &f.Name, &f.Version, &f.Category, &f.Status, &f.TargetFilter, &f.Notes, &f.CreatedAt, &sa, &ca)
	if err != nil { jsonErr(w, "not found", 404); return }
	f.StartedAt = sp(sa); f.CompletedAt = sp(ca)
	jsonResp(w, f)
}

func handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var f FirmwareCampaign
	if err := decodeBody(r, &f); err != nil { jsonErr(w, "invalid body", 400); return }
	f.ID = nextID("FW", "firmware_campaigns", 3)
	if f.Status == "" { f.Status = "draft" }
	if f.Category == "" { f.Category = "public" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO firmware_campaigns (id,name,version,category,status,target_filter,notes,created_at) VALUES (?,?,?,?,?,?,?,?)",
		f.ID, f.Name, f.Version, f.Category, f.Status, f.TargetFilter, f.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	f.CreatedAt = now
	logAudit(db, getUsername(r), "created", "firmware", f.ID, "Created campaign "+f.ID+": "+f.Name)
	jsonResp(w, f)
}

func handleUpdateCampaign(w http.ResponseWriter, r *http.Request, id string) {
	var f FirmwareCampaign
	if err := decodeBody(r, &f); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE firmware_campaigns SET name=?,version=?,category=?,status=?,target_filter=?,notes=? WHERE id=?",
		f.Name, f.Version, f.Category, f.Status, f.TargetFilter, f.Notes, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "firmware", id, "Updated campaign "+id)
	handleGetCampaign(w, r, id)
}

func handleLaunchCampaign(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	// Get all active devices and add them to campaign
	rows, err := db.Query("SELECT serial_number FROM devices WHERE status='active'")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	count := 0
	for rows.Next() {
		var sn string
		rows.Scan(&sn)
		db.Exec("INSERT OR IGNORE INTO campaign_devices (campaign_id,serial_number,status) VALUES (?,?,?)", id, sn, "pending")
		count++
	}
	db.Exec("UPDATE firmware_campaigns SET status='active',started_at=? WHERE id=?", now, id)
	logAudit(db, getUsername(r), "launched", "firmware", id, fmt.Sprintf("Launched campaign %s to %d devices", id, count))
	jsonResp(w, map[string]interface{}{"launched": true, "devices_added": count})
}

func handleCampaignProgress(w http.ResponseWriter, r *http.Request, id string) {
	var pending, sent, updated, failed int
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='pending'", id).Scan(&pending)
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='sent'", id).Scan(&sent)
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='updated'", id).Scan(&updated)
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='failed'", id).Scan(&failed)
	total := pending + sent + updated + failed
	jsonResp(w, map[string]int{"total": total, "pending": pending, "sent": sent, "updated": updated, "failed": failed})
}

func handleCampaignStream(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	for {
		var pending, sent, updated, failed int
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='pending'", id).Scan(&pending)
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='sent'", id).Scan(&sent)
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='updated'", id).Scan(&updated)
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='failed'", id).Scan(&failed)
		total := pending + sent + updated + failed
		pct := 0
		if total > 0 {
			pct = (updated + failed) * 100 / total
		}
		fmt.Fprintf(w, "data: {\"pending\":%d,\"sent\":%d,\"updated\":%d,\"failed\":%d,\"total\":%d,\"pct\":%d}\n\n", pending, sent, updated, failed, total, pct)
		flusher.Flush()
		if total > 0 && (updated+failed) >= total {
			break
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func handleMarkCampaignDevice(w http.ResponseWriter, r *http.Request, campaignID, serial string) {
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeBody(r, &body); err != nil || (body.Status != "updated" && body.Status != "failed") {
		jsonErr(w, "status must be 'updated' or 'failed'", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec("UPDATE campaign_devices SET status=?,updated_at=? WHERE campaign_id=? AND serial_number=?", body.Status, now, campaignID, serial)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonErr(w, "device not found in campaign", 404)
		return
	}
	logAudit(db, getUsername(r), "marked_"+body.Status, "firmware", campaignID, fmt.Sprintf("Marked %s as %s in campaign %s", serial, body.Status, campaignID))
	jsonResp(w, map[string]string{"status": "ok"})
}

func handleCampaignDevices(w http.ResponseWriter, r *http.Request, id string) {
	rows, err := db.Query("SELECT campaign_id,serial_number,status,updated_at FROM campaign_devices WHERE campaign_id=?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []CampaignDevice
	for rows.Next() {
		var cd CampaignDevice
		var ua sql.NullString
		rows.Scan(&cd.CampaignID, &cd.SerialNumber, &cd.Status, &ua)
		cd.UpdatedAt = sp(ua)
		items = append(items, cd)
	}
	if items == nil {
		items = []CampaignDevice{}
	}
	jsonResp(w, items)
}
