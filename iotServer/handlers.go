package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

// Data structures for dashboard
type DeviceStatus struct {
	ID                string
	Location          string
	LastSeen          time.Time
	LastSeenFormatted string
	IsOnline          bool
	LastTemp          float64
	LastHumid         float64
}

type ChartData struct {
	Points   []ChartPoint
	Min      float64
	Max      float64
	LinePath string
	AreaPath string
}

type ChartPoint struct {
	X             float64
	Y             float64
	Value         float64
	Time          time.Time
	TimeFormatted string
}

var templates *template.Template

func InitTemplates() error {
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		return err
	}
	templates, err = templates.ParseGlob("templates/partials/*.html")
	return err
}

func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title": "IoT Dashboard",
	}

	if err := templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func HandleDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := getDevicesWithStatus()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Devices": devices,
	}

	if err := templates.ExecuteTemplate(w, "devices.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func HandleDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		HandleDeleteDevice(w, r)
		return
	}

	deviceID := strings.TrimPrefix(r.URL.Path, "/device/")
	if deviceID == "" {
		http.Error(w, "device id required", http.StatusBadRequest)
		return
	}

	device, err := getDeviceStatus(deviceID)
	if err != nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	tempChart, err := getChartData(deviceID, 1) // param_id 1 = temperature
	if err != nil {
		tempChart = ChartData{}
	}

	humidChart, err := getChartData(deviceID, 2) // param_id 2 = humidity
	if err != nil {
		humidChart = ChartData{}
	}

	data := map[string]interface{}{
		"Device":     device,
		"TempChart":  tempChart,
		"HumidChart": humidChart,
	}

	if err := templates.ExecuteTemplate(w, "device.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func HandleAddDeviceForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}
	if err := templates.ExecuteTemplate(w, "add_device.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderAddDeviceError(w, "Invalid form data")
		return
	}

	deviceID := strings.TrimSpace(r.FormValue("device_id"))
	location := strings.TrimSpace(r.FormValue("location"))

	if deviceID == "" || location == "" {
		renderAddDeviceError(w, "Device ID and location are required")
		return
	}

	if len(deviceID) > 64 {
		renderAddDeviceError(w, "Device ID must be 64 characters or less")
		return
	}

	// Check if device already exists
	var existingID string
	err := conn.QueryRow("SELECT id FROM devices WHERE id = ?", deviceID).Scan(&existingID)
	if err == nil {
		renderAddDeviceError(w, "A device with this ID already exists")
		return
	}
	if err != sql.ErrNoRows {
		renderAddDeviceError(w, "Database error")
		return
	}

	// Insert new device
	_, err = conn.Exec("INSERT INTO devices (id, location) VALUES (?, ?)", deviceID, location)
	if err != nil {
		renderAddDeviceError(w, "Failed to create device")
		return
	}

	// Return to device list
	devices, err := getDevicesWithStatus()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Devices": devices,
	}

	if err := templates.ExecuteTemplate(w, "devices.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func renderAddDeviceError(w http.ResponseWriter, errorMsg string) {
	data := map[string]interface{}{
		"Error": errorMsg,
	}
	templates.ExecuteTemplate(w, "add_device.html", data)
}

func HandleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deviceID := strings.TrimPrefix(r.URL.Path, "/device/")
	if deviceID == "" {
		http.Error(w, "device id required", http.StatusBadRequest)
		return
	}

	// Delete readings first (foreign key constraint)
	_, err := conn.Exec("DELETE FROM readings WHERE device_id = ?", deviceID)
	if err != nil {
		http.Error(w, "failed to delete readings", http.StatusInternalServerError)
		return
	}

	// Delete device
	_, err = conn.Exec("DELETE FROM devices WHERE id = ?", deviceID)
	if err != nil {
		http.Error(w, "failed to delete device", http.StatusInternalServerError)
		return
	}

	// Return updated device list
	devices, err := getDevicesWithStatus()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Devices": devices,
	}

	if err := templates.ExecuteTemplate(w, "devices.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func getDevicesWithStatus() ([]DeviceStatus, error) {
	rows, err := conn.Query(`
		SELECT d.id, d.location, MAX(r.received_at) as last_seen
		FROM devices d
		LEFT JOIN readings r ON d.id = r.device_id
		GROUP BY d.id, d.location
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []DeviceStatus
	for rows.Next() {
		var d DeviceStatus
		var lastSeen sql.NullTime
		if err := rows.Scan(&d.ID, &d.Location, &lastSeen); err != nil {
			return nil, err
		}

		if lastSeen.Valid {
			d.LastSeen = lastSeen.Time
			d.IsOnline = time.Since(lastSeen.Time) < 10*time.Minute
			d.LastSeenFormatted = formatTimeAgo(lastSeen.Time)
		} else {
			d.IsOnline = false
			d.LastSeenFormatted = "never"
		}

		// Get latest readings
		var temp, humid sql.NullFloat64
		conn.QueryRow(`
			SELECT value FROM readings
			WHERE device_id = ? AND param_id = 1
			ORDER BY device_timestamp DESC LIMIT 1
		`, d.ID).Scan(&temp)
		conn.QueryRow(`
			SELECT value FROM readings
			WHERE device_id = ? AND param_id = 2
			ORDER BY device_timestamp DESC LIMIT 1
		`, d.ID).Scan(&humid)

		if temp.Valid {
			d.LastTemp = temp.Float64
		}
		if humid.Valid {
			d.LastHumid = humid.Float64
		}

		devices = append(devices, d)
	}

	return devices, nil
}

func getDeviceStatus(deviceID string) (DeviceStatus, error) {
	var d DeviceStatus
	var lastSeen sql.NullTime

	err := conn.QueryRow(`
		SELECT d.id, d.location, MAX(r.received_at) as last_seen
		FROM devices d
		LEFT JOIN readings r ON d.id = r.device_id
		WHERE d.id = ?
		GROUP BY d.id, d.location
	`, deviceID).Scan(&d.ID, &d.Location, &lastSeen)
	if err != nil {
		return d, err
	}

	if lastSeen.Valid {
		d.LastSeen = lastSeen.Time
		d.IsOnline = time.Since(lastSeen.Time) < 10*time.Minute
		d.LastSeenFormatted = formatTimeAgo(lastSeen.Time)
	} else {
		d.IsOnline = false
		d.LastSeenFormatted = "never"
	}

	var temp, humid sql.NullFloat64
	conn.QueryRow(`
		SELECT value FROM readings
		WHERE device_id = ? AND param_id = 1
		ORDER BY device_timestamp DESC LIMIT 1
	`, deviceID).Scan(&temp)
	conn.QueryRow(`
		SELECT value FROM readings
		WHERE device_id = ? AND param_id = 2
		ORDER BY device_timestamp DESC LIMIT 1
	`, deviceID).Scan(&humid)

	if temp.Valid {
		d.LastTemp = temp.Float64
	}
	if humid.Valid {
		d.LastHumid = humid.Float64
	}

	return d, nil
}

func getChartData(deviceID string, paramID int) (ChartData, error) {
	rows, err := conn.Query(`
		SELECT value, device_timestamp
		FROM readings
		WHERE device_id = ? AND param_id = ?
		ORDER BY device_timestamp DESC
		LIMIT 24
	`, deviceID, paramID)
	if err != nil {
		return ChartData{}, err
	}
	defer rows.Close()

	var points []ChartPoint
	for rows.Next() {
		var p ChartPoint
		if err := rows.Scan(&p.Value, &p.Time); err != nil {
			return ChartData{}, err
		}
		p.TimeFormatted = p.Time.Format("Jan 2 15:04")
		points = append(points, p)
	}

	if len(points) == 0 {
		return ChartData{}, nil
	}

	// Reverse to chronological order
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	// Find min/max
	minVal, maxVal := points[0].Value, points[0].Value
	for _, p := range points {
		if p.Value < minVal {
			minVal = p.Value
		}
		if p.Value > maxVal {
			maxVal = p.Value
		}
	}

	// Add padding to range
	padding := (maxVal - minVal) * 0.1
	if padding == 0 {
		padding = 1
	}
	minVal -= padding
	maxVal += padding

	// Normalize points to SVG coordinates
	chartWidth := 400.0
	chartHeight := 120.0
	chartPadding := 10.0

	for i := range points {
		points[i].X = chartPadding + (float64(i)/float64(len(points)-1))*(chartWidth-2*chartPadding)
		normalizedY := (points[i].Value - minVal) / (maxVal - minVal)
		points[i].Y = chartHeight - chartPadding - normalizedY*(chartHeight-2*chartPadding)
	}

	// Generate SVG paths
	var linePath, areaPath strings.Builder

	linePath.WriteString(fmt.Sprintf("M%.1f,%.1f", points[0].X, points[0].Y))
	areaPath.WriteString(fmt.Sprintf("M%.1f,%.1f", points[0].X, chartHeight-chartPadding))
	areaPath.WriteString(fmt.Sprintf("L%.1f,%.1f", points[0].X, points[0].Y))

	for i := 1; i < len(points); i++ {
		linePath.WriteString(fmt.Sprintf("L%.1f,%.1f", points[i].X, points[i].Y))
		areaPath.WriteString(fmt.Sprintf("L%.1f,%.1f", points[i].X, points[i].Y))
	}

	areaPath.WriteString(fmt.Sprintf("L%.1f,%.1f", points[len(points)-1].X, chartHeight-chartPadding))
	areaPath.WriteString("Z")

	return ChartData{
		Points:   points,
		Min:      minVal + padding,
		Max:      maxVal - padding,
		LinePath: linePath.String(),
		AreaPath: areaPath.String(),
	}, nil
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

type Reading struct {
	ID              int64     `json:"id"`
	DeviceID        string    `json:"device_id"`
	ParamID         int       `json:"param_id"`
	Value           float64   `json:"value"`
	DeviceTimestamp time.Time `json:"device_timestamp"`
	ReceivedAt      time.Time `json:"received_at"`
	EntryHash       []byte    `json:"entry_hash"`
}

func PostWeatherListener(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	deviceID := r.Header.Get("X-Device-Id")
	if deviceID == "" {
		http.Error(w, "who are you?", http.StatusUnauthorized)
		return
	}

	var foundID string
	err := conn.QueryRow("SELECT id FROM devices WHERE id = ?", deviceID).Scan(&foundID)
	if err == sql.ErrNoRows {
		http.Error(w, "unknown device", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	var reading Reading
	if err := json.NewDecoder(r.Body).Decode(&reading); err != nil {
		http.Error(w, "not like that", http.StatusBadRequest)
		return
	}

	reading.DeviceID = deviceID
	reading.ReceivedAt = time.Now()

	hashInput := fmt.Sprintf("%s:%d:%f:%s", reading.DeviceID, reading.ParamID, reading.Value, reading.DeviceTimestamp.Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(hashInput))
	reading.EntryHash = hash[:]

	_, err = conn.Exec(
		`INSERT INTO readings (device_id, param_id, value, device_timestamp, received_at, entry_hash)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		reading.DeviceID,
		reading.ParamID,
		reading.Value,
		reading.DeviceTimestamp,
		reading.ReceivedAt,
		reading.EntryHash,
	)
	if err != nil {
		http.Error(w, "failed to save reading", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
