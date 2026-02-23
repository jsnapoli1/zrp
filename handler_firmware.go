package main

import (
	"net/http"
)

func handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().ListCampaigns(w, r)
}

func handleGetCampaign(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().GetCampaign(w, r, id)
}

func handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().CreateCampaign(w, r)
}

func handleUpdateCampaign(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().UpdateCampaign(w, r, id)
}

func handleLaunchCampaign(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().LaunchCampaign(w, r, id)
}

func handleCampaignProgress(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().CampaignProgress(w, r, id)
}

func handleCampaignStream(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().CampaignStream(w, r, id)
}

func handleMarkCampaignDevice(w http.ResponseWriter, r *http.Request, campaignID, serial string) {
	getFieldHandler().MarkCampaignDevice(w, r, campaignID, serial)
}

func handleCampaignDevices(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().CampaignDevices(w, r, id)
}
