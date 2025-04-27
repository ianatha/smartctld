// Initialize your module before running:
//   go mod init smartctlservice

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/jaypipes/ghw"
)

// SmartResult holds the SMART data or an error for a device
// Data contains generic SMART attributes mapped to values
// Error is non-empty if the query failed for the device
// Device is the short disk name (e.g., "sda")
type SmartResult struct {
	Device string                 `json:"device"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

// listDrivesHandler returns SMART generic attributes for all physical drives
func listDrivesHandler(w http.ResponseWriter, r *http.Request) {
	blockInfo, err := ghw.Block()
	if err != nil {
		http.Error(w, "Failed to enumerate block devices: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var results []SmartResult
	for _, disk := range blockInfo.Disks {
		// Skip virtual (e.g., virtio) drives
		if disk.StorageController == ghw.StorageControllerVirtIO {
			continue
		}
		device := disk.Name
		path := "/dev/" + device
		dev, err := smart.Open(path)
		if err != nil {
			results = append(results, SmartResult{Device: device, Error: err.Error()})
			continue
		}
		defer dev.Close()

		attrs, err := dev.ReadGenericAttributes()
		if err != nil {
			results = append(results, SmartResult{Device: device, Error: err.Error()})
			continue
		}

		// Map generic attributes to JSON-friendly data
		data := map[string]interface{}{
			"temperature_celsius": attrs.Temperature,
			"read_blocks":         attrs.Read,
			"written_blocks":      attrs.Written,
			"power_on_hours":      attrs.PowerOnHours,
			"power_cycles":        attrs.PowerCycles,
		}
		results = append(results, SmartResult{Device: device, Data: data})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// driveHandler returns SMART generic attributes for a specific drive
func driveHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/drives/")
	blockInfo, err := ghw.Block()
	if err != nil {
		http.Error(w, "Failed to enumerate block devices: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var found bool
	for _, disk := range blockInfo.Disks {
		if disk.StorageController == ghw.StorageControllerVirtIO {
			continue
		}
		if disk.Name == name {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Unknown or virtual drive", http.StatusNotFound)
		return
	}
	path := "/dev/" + name
	dev, err := smart.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dev.Close()
	attrs, err := dev.ReadGenericAttributes()
	res := SmartResult{Device: name}
	if err != nil {
		res.Error = err.Error()
	} else {
		res.Data = map[string]interface{}{
			"temperature_celsius": attrs.Temperature,
			"read_blocks":         attrs.Read,
			"written_blocks":      attrs.Written,
			"power_on_hours":      attrs.PowerOnHours,
			"power_cycles":        attrs.PowerCycles,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func main() {
	// API endpoints
	http.HandleFunc("/drives", listDrivesHandler)
	http.HandleFunc("/drives/", driveHandler)

	log.Println("SMART API server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
