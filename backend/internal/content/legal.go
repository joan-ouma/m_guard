// Package content holds the legal text the chatbot can read aloud to riders.
// It mirrors frontend/src/app/legal/legal.component.html; update both when the wording changes.
package content

// Clause is one numbered section of the Terms or the Privacy Policy.
type Clause struct {
	Title string
	Body  string
}

var Terms = []Clause{
	{Title: "Acceptance of Terms", Body: "By accessing or using the MGuard platform, web applications, or IoT tracking services, you agree to be bound by these Terms and Conditions. If you do not agree to these terms, you must not access or use the platform."},
	{Title: "Services Provided", Body: "MGuard provides motorcycle telemetry, real-time GPS tracking, geofence boundary monitoring, and remote vehicle immobilization commands. Service availability depends on GPS satellite reception, GSM cellular network coverage, and device power status."},
	{Title: "Hardware Trackers & Installation", Body: "Users are responsible for ensuring hardware trackers with certified 15-digit IMEIs are properly installed by qualified automotive technicians. MGuard is not responsible for vehicle battery discharge or electrical issues resulting from improper physical installation."},
	{Title: "Remote Immobilization Safety", Body: "Remote engine cut-off commands disengage motorcycle ignition or fuel delivery. You agree to use this feature only when the vehicle is stationary or moving at safe crawl speeds. You accept full responsibility for any accident or damage caused by triggering immobilization during high-speed transit."},
	{Title: "User Accounts & Rate Limiting", Body: "You are responsible for keeping your login credentials confidential. The platform enforces automated rate limiting to protect accounts and hardware trackers against unauthorized access and brute-force scans."},
	{Title: "Subscriptions & Data Charges", Body: "Riders do not pay subscription fees to use MGuard. Hardware trackers are supplied through our partnerships with motorcycle assembly companies, and the cellular connection that sends tracking data to MGuard servers is maintained as part of that service. If the connection is interrupted, tracking and remote control may be temporarily unavailable."},
	{Title: "Acceptable Use", Body: "You agree not to use MGuard for unlawful surveillance, harassment, or unauthorized tracking of individuals without consent. You may not reverse-engineer, decompile, or tamper with MGuard hardware firmware or API services."},
	{Title: "Limitation of Liability", Body: "MGuard is a tracking and telemetry management aid, not an insurance policy. MGuard is not liable for vehicle theft, recovery failures, road accidents, or indirect commercial losses. Vehicle owners must maintain appropriate commercial insurance."},
	{Title: "Termination", Body: "MGuard reserves the right to suspend or terminate accounts that violate these terms or engage in abusive activity. You may cancel your account at any time by contacting customer support."},
	{Title: "Governing Law", Body: "These terms are governed by and construed in accordance with the laws of the Republic of Kenya. Any unresolved legal disputes shall be settled through arbitration under Kenyan law."},
}

var Privacy = []Clause{
	{Title: "Overview", Body: "MGuard respects your privacy and is committed to protecting your personal information and vehicle telemetry data in compliance with the Kenya Data Protection Act, 2019 and relevant privacy regulations."},
	{Title: "Information We Collect", Body: "We collect account details (name, email address, phone number), vehicle information (license plate, motorcycle model), and device telemetry transmitted by installed hardware (GPS coordinates, speed, heading, ignition status, battery voltage, and 15-digit IMEI)."},
	{Title: "How We Use Your Data", Body: "Your information is used solely to provide real-time tracking, generate historical ride breadcrumbs, calculate geofence breach notifications, detect vehicle tampering, and maintain service security. We do not sell or rent your personal or location data to third parties."},
	{Title: "Data Sharing", Body: "We only share telemetry with essential service providers: cellular carriers for SIM connectivity, secure cloud hosting infrastructure, and OpenAI for the MGuard Assistant. In cases of motorcycle theft or legal investigation, location vectors may be shared with law enforcement authorities upon verified request.\n\nMGuard Assistant: when you use the assistant to check your motorcycle, the bike's last known location, address, ignition, speed, battery and signal status, and recent events are sent to OpenAI, a third-party AI service, so it can write the reply in your chosen language. MGuard keeps a record of your assistant questions and replies for 90 days so you can review them, and you can clear that history at any time. OpenAI processes the data it receives under its own API terms, and that processing may take place outside Kenya. Use of the assistant is optional."},
	{Title: "Data Retention", Body: "High-resolution location breadcrumbs are retained for 90 days for operational replay. Historical trip aggregates are kept for up to 12 months, after which they are permanently expunged or anonymized. Assistant history is kept for 90 days and can be cleared earlier from the assistant."},
	{Title: "Security Measures", Body: "We use industry-standard encryption protocols (TLS 1.3 in transit, salted bcrypt password hashes, and access-controlled databases) to safeguard your vehicle records from unauthorized access."},
	{Title: "Your Rights", Body: "Under the Kenya Data Protection Act, you have the right to access the data we hold about you, request corrections to erroneous records, and request the deletion of your account and associated tracking records upon discontinuing service."},
	{Title: "Contact Us", Body: "If you have questions about this Privacy Policy or wish to exercise your data rights, contact us at privacy@jullie.tech."},
}
