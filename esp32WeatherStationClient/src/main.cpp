#include <Arduino.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <DHT.h>
#include "esp_sleep.h"
#include "time.h"

#define WIFI_SSID "bye bye"
#define WIFI_PASSWORD "hello"

#define SERVICE_URL "http://example.com/api/data"
#define DEVICE_ID "esp32-001"

#define DHTPIN 4 // remember to put 10k Ohm pull-up resistor. 
#define DHTTYPE DHT11


#define SLEEP_INTERVAL_SECONDS 300 // 5 minutes

DHT dht(DHTPIN, DHTTYPE);

// Connect WiFi
void connectWifi() {
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  Serial.print("Connecting to WiFi");
  uint8_t retries = 0;

  while (WiFi.status() != WL_CONNECTED && retries < 20) {
    delay(500);
    Serial.print(".");
    retries++;
  }
  Serial.println();
}

// Read sensor values
bool sensorReading(float &temperature, float &humidity) {
  humidity = dht.readHumidity();
  temperature = dht.readTemperature();

  if (isnan(humidity) || isnan(temperature)) {
    Serial.println("Failed to read from DHT sensor!");
    return false;
  }
  return true;
}


// Get ISO timestamp from NTP
String getTimestamp() {
  struct tm timeinfo;
  if (!getLocalTime(&timeinfo)) {
    return "1970-01-01T00:00:00Z"; // fallback
  }

  char buffer[30];
  strftime(buffer, sizeof(buffer), "%Y-%m-%dT%H:%M:%SZ", &timeinfo);
  return String(buffer);
}

// Send data to API
void sendReading(float temperature, float humidity, String timestamp) {
  if (WiFi.status() != WL_CONNECTED) {
    connectWifi();
    if (WiFi.status() != WL_CONNECTED) {
      Serial.println("WiFi not connected – cannot send data");
      return;
    }
  }

  HTTPClient http;
  http.begin(SERVICE_URL);
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-ID", DEVICE_ID);

  String payload = "{\"temperature\":" + String(temperature) +
                   ",\"humidity\":" + String(humidity) +
                   ",\"timestamp\":\"" + timestamp + "\"}";

  Serial.print("Sending JSON: ");
  Serial.println(payload);

  int code = http.POST(payload);
  Serial.print("Response code: ");
  Serial.println(code);

  http.end();
}

void setup() {
  Serial.begin(115200);
  delay(500);

  dht.begin();

  // Connect early for NTP time
  connectWifi();
  configTime(0, 0, "pool.ntp.org", "time.nist.gov");

  float t, h;
  if (!sensorReading(t, h)) {
    t = h = 0; // fallback
  }

  String timestamp = getTimestamp();

  Serial.println("Temp: " + String(t));
  Serial.println("Humidity: " + String(h));
  Serial.println("Timestamp: " + timestamp);

  sendReading(t, h,timestamp);

  // Sleep
  Serial.println("Going to deep sleep...");
  esp_sleep_enable_timer_wakeup((uint64_t)SLEEP_INTERVAL_SECONDS * 1000000ULL);
  esp_deep_sleep_start();
}

void loop() {
  // Won't run
}
