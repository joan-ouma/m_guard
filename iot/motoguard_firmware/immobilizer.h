#pragma once
#include <Arduino.h>
#include <Preferences.h>
#include "config.h"
#include "mqtt_client.h"

namespace immobilizer {

inline Preferences prefs;
inline bool engaged = false;
inline unsigned long lastTs = 0;
inline bool     ackPending   = false;   // true until the broker confirms the status with a PUBACK
inline uint32_t ackLastTryMs = 0;

inline void apply() {
  pinMode(PIN_IMMOBILIZER, OUTPUT);
  digitalWrite(PIN_IMMOBILIZER, engaged ? IMMOBILIZER_ON_LEVEL : IMMOBILIZER_OFF_LEVEL);
}

inline void persist() {
  prefs.begin("immob", false);
  prefs.putBool("engaged", engaged);
  prefs.putBool("ackPend", ackPending);
  prefs.putULong("ackTs", lastTs);
  prefs.end();
}

// Publishes the state at QoS 1 and returns true only after the broker's PUBACK.
inline bool publishStatus() {
  if (!modem::isNetworkRegistered()) return false;

  String payload = String("{\"immobilized\":") + (engaged ? "true" : "false") + ",\"ts\":" + String(lastTs) + "}";
  uint16_t packetId = mqtt::nextPacketId++;
  if (packetId == 0) packetId = mqtt::nextPacketId++;

  if (!mqtt::publishQos1(IMMOB_STATUS_TOPIC, payload, packetId)) return false;
  return mqtt::waitPubAck(packetId, 5000);
}

// Retries the status until the broker confirms it. Runs from loop(), so a failed
// send never blocks the command handling.
inline void serviceAck() {
  if (!ackPending) return;
  if (ackLastTryMs && millis() - ackLastTryMs < IMMOB_ACK_RETRY_MS) return;
  ackLastTryMs = millis();

  if (publishStatus()) {
    ackPending = false;
    persist();
    Serial.println("[IMMOB] Status confirmed by broker");
  } else {
    Serial.println("[IMMOB] Status not confirmed yet, will retry");
  }
}

inline void begin() {
  // Set the relay off before anything else so a reset never cuts the fuel by accident.
  pinMode(PIN_IMMOBILIZER, OUTPUT);
  digitalWrite(PIN_IMMOBILIZER, IMMOBILIZER_OFF_LEVEL);

  prefs.begin("immob", true);
  engaged    = prefs.getBool("engaged", false);
  ackPending = prefs.getBool("ackPend", false);
  lastTs     = prefs.getULong("ackTs", 0);
  prefs.end();

  apply();
  Serial.println(engaged ? "[IMMOB] Restored: IMMOBILIZED" : "[IMMOB] Restored: mobilized");
}

inline void engage() {
  engaged = true;
  ackPending = true;
  apply();
  persist();
  Serial.println("[IMMOB] Engine IMMOBILIZED");
}

inline void release() {
  engaged = false;
  ackPending = true;
  apply();
  persist();
  Serial.println("[IMMOB] Engine MOBILIZED");
}

// Reads "action" from {"action":"IMMOBILIZE","timestamp":...}, or takes the bare word as plain text.
inline String parseCommand(const String &payload) {
  String cmd = payload;
  int key = payload.indexOf("\"action\"");
  if (key != -1) {
    int colon = payload.indexOf(':', key);
    int q1 = colon == -1 ? -1 : payload.indexOf('"', colon + 1);
    int q2 = q1 == -1 ? -1 : payload.indexOf('"', q1 + 1);
    cmd = (q2 == -1) ? String("") : payload.substring(q1 + 1, q2);
  }
  cmd.trim();
  cmd.toLowerCase();
  return cmd;
}

inline unsigned long parseTimestamp(const String &payload) {
  int key = payload.indexOf("\"timestamp\"");
  int colon = key == -1 ? -1 : payload.indexOf(':', key);
  return colon == -1 ? 0 : strtoul(payload.c_str() + colon + 1, nullptr, 10);
}

inline void handleCommand(const String &payload) {
  String action = parseCommand(payload);
  lastTs = parseTimestamp(payload);

  if (action == "immobilize") {
    engage();
  } else if (action == "reactivate") {
    release();
  } else {
    Serial.println("[IMMOB] Ignored unknown action: " + payload);
  }
}

}
