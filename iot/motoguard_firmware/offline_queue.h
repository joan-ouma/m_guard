#pragma once
#include <Arduino.h>
#include <LittleFS.h>
#include "config.h"
#include "mqtt_client.h"

// Store-and-forward for alerts. An alert that cannot be published is appended to a
// LittleFS file (one JSON payload per line) and sent later, oldest first, once the
// modem is registered again. The file survives reboots.
namespace offline {

constexpr char QUEUE_PATH[]     = "/alerts.q";
constexpr char QUEUE_TMP_PATH[] = "/alerts.tmp";

inline bool     mounted = false;
inline uint16_t queued  = 0;

inline bool begin() {
  mounted = LittleFS.begin(true);
  if (!mounted) {
    Serial.println("[OFFLINE] LittleFS mount failed; unsent alerts will not be stored");
    return false;
  }

  queued = 0;
  File f = LittleFS.open(QUEUE_PATH, FILE_READ);
  if (f) {
    while (f.available()) {
      if (f.read() == '\n') queued++;
    }
    f.close();
  }
  Serial.printf("[OFFLINE] %u unsent alert(s) stored\n", queued);
  return true;
}

inline bool hasPending() { return queued > 0; }

// Copies the queue into a temp file without its first `skip` records, then swaps it in.
inline void rewriteWithout(uint16_t skip) {
  File src = LittleFS.open(QUEUE_PATH, FILE_READ);
  File dst = LittleFS.open(QUEUE_TMP_PATH, FILE_WRITE);
  if (!src || !dst) {
    src.close();
    dst.close();
    Serial.println("[OFFLINE] Could not rewrite queue");
    return;
  }

  uint16_t index = 0, kept = 0;
  while (src.available()) {
    String rec = src.readStringUntil('\n');
    if (rec.length() == 0) continue;
    if (index++ < skip) continue;
    dst.println(rec);
    kept++;
  }
  src.close();
  dst.close();

  LittleFS.remove(QUEUE_PATH);
  LittleFS.rename(QUEUE_TMP_PATH, QUEUE_PATH);
  queued = kept;
}

inline void enqueueAlert(const String &payload) {
  if (!mounted) return;

  if (queued >= OFFLINE_MAX_ALERTS) {
    rewriteWithout(1);   // full: the oldest alert makes room for the newest
  }

  String rec = payload;
  rec.replace('\n', ' ');
  rec.replace('\r', ' ');

  File f = LittleFS.open(QUEUE_PATH, FILE_APPEND);
  if (!f) {
    Serial.println("[OFFLINE] Could not open queue for writing");
    return;
  }
  f.println(rec);
  f.close();
  queued++;
  Serial.printf("[OFFLINE] Alert stored for later (%u pending)\n", queued);
}

// Publishes up to maxSend queued alerts, oldest first. Stops at the first failed
// publish so the rest keep their order. Returns how many were sent.
inline uint16_t flush(uint8_t maxSend) {
  if (!mounted || queued == 0) return 0;

  File f = LittleFS.open(QUEUE_PATH, FILE_READ);
  if (!f) return 0;

  uint16_t sent = 0;
  while (sent < maxSend && f.available()) {
    String rec = f.readStringUntil('\n');
    if (rec.length() == 0) continue;
    if (!mqtt::publish(ALERT_TOPIC, rec)) break;
    sent++;
  }
  f.close();

  if (sent > 0) rewriteWithout(sent);
  return sent;
}

}
