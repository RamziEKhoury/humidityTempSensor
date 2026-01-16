Below is a **clean, production-oriented `README.md`** you can use as the foundation for this project.
It assumes **ESP32 devices pushing data to a Go backend over HTTPS**.

---

# ESP32 Temperature & Humidity Monitoring System

This project implements a distributed temperature and humidity monitoring system using ESP32 devices deployed across multiple Wi-Fi networks and a centralized Go backend for data ingestion, storage, and monitoring.

The system is designed to work reliably across NATed networks, be secure by default, and scale from a handful of devices to hundreds.

---

## Architecture Overview

**Core design principle:**
ESP32 devices act as **clients** that periodically push sensor data to a **public Go backend**. Devices do not expose inbound endpoints.

### High-Level Components

* **ESP32 Devices**

  * Read temperature and humidity sensors
  * Authenticate and send data over HTTPS
  * Operate behind arbitrary Wi-Fi networks and NATs

* **Go Backend**

  * Exposes HTTPS ingestion endpoints
  * Authenticates devices
  * Persists readings and device state
  * Acts as the single source of truth

* **Persistence Layer**

  * Time-series or relational database for sensor readings
  * Device metadata storage (last seen, firmware version, status)

---

## Data Flow

1. ESP32 boots and connects to Wi-Fi
2. ESP32 reads temperature and humidity sensor
3. ESP32 sends an HTTPS POST request to the backend
4. Backend authenticates the device
5. Backend validates and stores the reading
6. Backend updates device `last_seen` timestamp
7. ESP32 sleeps or waits until the next interval

Heartbeats are typically **implicit**: a missing reading indicates an offline device.

---

## Communication Model

### Why ESP32 as Client (Push Model)

* Devices are behind NAT and consumer routers
* Inbound connections are unreliable or impossible
* Security is easier to manage centrally
* Lower power consumption
* Easier horizontal scaling

### Transport

* HTTPS (TLS)
* JSON payloads
* Outbound-only connections from devices

---

## API Design

### Ingest Sensor Readings

```
POST /api/v1/readings
```

**Headers**

```
X-Device-ID: <device_id>
X-Signature: <HMAC signature>
Content-Type: application/json
```

**Body**

```json
{
  "temperature": 23.8,
  "humidity": 48.2,
  "timestamp": 1700000000
}
```

### Optional Heartbeat Endpoint

```
POST /api/v1/heartbeat
```

This is optional. Most systems rely on sensor publishes to track device health.

---

## Authentication & Security

Each ESP32 device is provisioned with:

* `device_id`
* `device_secret`

Requests are authenticated using an HMAC signature calculated over the request body.

This prevents:

* Device spoofing
* Unauthorized data injection
* Replay attacks (when combined with timestamps)

TLS is mandatory for all device communication.

---

## Backend Responsibilities (Go)

* HTTP server (`net/http`, `chi`, or `gin`)
* Authentication middleware
* Payload validation
* Ingestion handlers
* Storage abstraction
* Device state tracking
* Logging and metrics

Suggested internal structure:

```
/cmd/server
/internal/http
/internal/auth
/internal/ingest
/internal/storage
/internal/devices
```

---

## Storage Model

### Devices

Stores device metadata and status.

Fields:

* device_id
* last_seen
* firmware_version
* created_at

### Readings

Stores time-series sensor data.

Fields:

* device_id
* temperature
* humidity
* timestamp

For higher scale, consider:

* TimescaleDB
* InfluxDB
* ClickHouse

---

## ESP32 Firmware Responsibilities

* Connect to Wi-Fi
* Read sensors at a fixed interval
* Build JSON payload
* Sign request with device secret
* POST data to backend
* Retry with backoff on failure
* Enter deep sleep if power constrained

Typical loop:

```
connect_wifi()
read_sensor()
send_https_request()
sleep()
```

---

## Failure Handling

* Network failures: exponential backoff
* Backend errors: retry with limits
* Device offline detection: missing data over time
* No inbound connections required to devices

---

## Scalability Notes

This architecture scales well for:

* Dozens to hundreds of devices
* Multiple Wi-Fi networks and ISPs
* Single or multi-instance Go backend

For advanced use cases (bidirectional control, real-time streaming), MQTT can be introduced later without changing the core data model.

---

## Non-Goals

* Device-to-device communication
* Direct device exposure to the public internet
* Inbound polling of ESP32 devices

---

## Future Extensions

* Device provisioning service
* OTA firmware updates
* Dashboard and alerting
* MQTT support
* Configuration push from backend

