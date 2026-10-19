package service

import (
	"fmt"
	"strings"
)

type AlertMessage struct {
	Title string
	Body  string
}

type AlertTemplate struct {
	TitleTemplate string
	BodyTemplate  string
}

var alertTemplates = map[string]AlertTemplate{
	"GEOFENCE_BREACH": {
		TitleTemplate: "🚨 Geofence Breach Alert",
		BodyTemplate:  "Motorcycle %s has breached its geofence boundary.",
	},
	"CRASH": {
		TitleTemplate: "🚨 Crash Detected",
		BodyTemplate:  "Collision or crash detected on motorcycle %s.",
	},
	"ACCIDENT_DETECTED": {
		TitleTemplate: "🚨 Accident Detected",
		BodyTemplate:  "Accident or impact detected on motorcycle %s.",
	},
	"THEFT": {
		TitleTemplate: "🚨 Theft / Tamper Alert",
		BodyTemplate:  "Unauthorized movement or theft alert on motorcycle %s.",
	},
	"TAMPER": {
		TitleTemplate: "🚨 Tamper Detected",
		BodyTemplate:  "Tamper alert detected on motorcycle %s.",
	},
	"DURESS": {
		TitleTemplate: "🚨 Duress Emergency Alert",
		BodyTemplate:  "Duress or emergency trigger activated for motorcycle %s.",
	},
	"SUDDEN_STOP": {
		TitleTemplate: "⚠️ Sudden Stop Anomaly",
		BodyTemplate:  "Sudden deceleration or stop anomaly recorded for %s.",
	},
	"IMMOBILIZED": {
		TitleTemplate: "🔒 Engine Immobilized",
		BodyTemplate:  "Starter ignition has been remotely immobilized for %s.",
	},
	"REACTIVATED": {
		TitleTemplate: "⚡ Starter Reactivated",
		BodyTemplate:  "Starter ignition power has been restored for %s.",
	},
	"BIKE_TOUCHED": {
		TitleTemplate: "⚠️ Vibration / Movement",
		BodyTemplate:  "Motion vibration detected while parked on %s.",
	},
}

func RenderAlert(eventType, registration string) AlertMessage {
	key := strings.ToUpper(strings.TrimSpace(eventType))
	tpl, ok := alertTemplates[key]
	if ok {
		return AlertMessage{
			Title: tpl.TitleTemplate,
			Body:  fmt.Sprintf(tpl.BodyTemplate, registration),
		}
	}

	formattedType := strings.ReplaceAll(key, "_", " ")
	return AlertMessage{
		Title: fmt.Sprintf("🚨 %s", formattedType),
		Body:  fmt.Sprintf("Security event %s detected on %s.", formattedType, registration),
	}
}
