// Package content holds the legal text the chatbot can read aloud to riders.
// It mirrors frontend/src/app/legal/legal.component.html; update both when the wording changes.
// The Swahili text is a draft translation and must be reviewed before it is relied on.
package content

// Clause is one numbered section of the Terms or the Privacy Policy, in English and Kiswahili.
type Clause struct {
	Title   string
	Body    string
	TitleSW string
	BodySW  string
}

var Terms = []Clause{
	{
		Title:   "Acceptance of Terms",
		Body:    "By accessing or using the MGuard platform, web applications, or IoT tracking services, you agree to be bound by these Terms and Conditions. If you do not agree to these terms, you must not access or use the platform.",
		TitleSW: "Kukubali Masharti",
		BodySW:  "Kwa kufikia au kutumia jukwaa la MGuard, programu za wavuti, au huduma za ufuatiliaji wa IoT, unakubali kufungwa na Masharti haya ya Matumizi. Usipokubaliana na masharti haya, usifikie wala kutumie jukwaa hili.",
	},
	{
		Title:   "Services Provided",
		Body:    "MGuard provides motorcycle telemetry, real-time GPS tracking, geofence boundary monitoring, and remote vehicle immobilization commands. Service availability depends on GPS satellite reception, GSM cellular network coverage, and device power status.",
		TitleSW: "Huduma Tunazotoa",
		BodySW:  "MGuard inatoa telemetri ya pikipiki, ufuatiliaji wa GPS wa wakati halisi, ufuatiliaji wa mipaka ya geofence, na amri za mbali za kuzima injini. Upatikanaji wa huduma unategemea mapokezi ya setilaiti ya GPS, upatikanaji wa mtandao wa simu wa GSM, na hali ya umeme wa kifaa.",
	},
	{
		Title:   "Hardware Trackers & Installation",
		Body:    "Users are responsible for ensuring hardware trackers with certified 15-digit IMEIs are properly installed by qualified automotive technicians. MGuard is not responsible for vehicle battery discharge or electrical issues resulting from improper physical installation.",
		TitleSW: "Vifaa vya Ufuatiliaji na Usakinishaji",
		BodySW:  "Watumiaji wanawajibika kuhakikisha vifaa vya ufuatiliaji vyenye IMEI halali za tarakimu 15 vimesakinishwa na mafundi wa magari waliohitimu. MGuard haihusiki na kuisha kwa betri ya gari au matatizo ya umeme yanayotokana na usakinishaji usio sahihi.",
	},
	{
		Title:   "Remote Immobilization Safety",
		Body:    "Remote engine cut-off commands disengage motorcycle ignition or fuel delivery. You agree to use this feature only when the vehicle is stationary or moving at safe crawl speeds. You accept full responsibility for any accident or damage caused by triggering immobilization during high-speed transit.",
		TitleSW: "Usalama wa Kuzima Injini kwa Mbali",
		BodySW:  "Amri za kuzima injini kwa mbali huzima swichi ya pikipiki au usambazaji wa mafuta. Unakubali kutumia kipengele hiki tu wakati pikipiki imesimama au inasonga kwa mwendo salama wa polepole. Unakubali kubeba jukumu kamili kwa ajali yoyote au uharibifu unaosababishwa na kuzima injini wakati wa safari ya mwendo kasi.",
	},
	{
		Title:   "User Accounts & Rate Limiting",
		Body:    "You are responsible for keeping your login credentials confidential. The platform enforces automated rate limiting to protect accounts and hardware trackers against unauthorized access and brute-force scans.",
		TitleSW: "Akaunti za Watumiaji na Kikomo cha Maombi",
		BodySW:  "Unawajibika kuweka siri nenosiri lako la kuingia. Jukwaa linatumia kikomo cha maombi kiotomatiki kulinda akaunti na vifaa vya ufuatiliaji dhidi ya ufikiaji usioidhinishwa na majaribio ya kubahatisha.",
	},
	{
		Title:   "Subscriptions & Data Charges",
		Body:    "Riders do not pay subscription fees to use MGuard. Hardware trackers are supplied through our partnerships with motorcycle assembly companies, and the cellular connection that sends tracking data to MGuard servers is maintained as part of that service. If the connection is interrupted, tracking and remote control may be temporarily unavailable.",
		TitleSW: "Usajili na Gharama za Data",
		BodySW:  "Watumiaji hawalipi ada za usajili ili kutumia MGuard. Vifaa vya ufuatiliaji vinatolewa kupitia ushirikiano wetu na kampuni za kutengeneza pikipiki, na muunganisho wa simu unaotuma data ya ufuatiliaji kwa seva za MGuard unadumishwa kama sehemu ya huduma hiyo. Muunganisho ukikatika, ufuatiliaji na udhibiti wa mbali vinaweza kusimama kwa muda.",
	},
	{
		Title:   "Acceptable Use",
		Body:    "You agree not to use MGuard for unlawful surveillance, harassment, or unauthorized tracking of individuals without consent. You may not reverse-engineer, decompile, or tamper with MGuard hardware firmware or API services.",
		TitleSW: "Matumizi Yanayokubalika",
		BodySW:  "Unakubali kutotumia MGuard kwa ufuatiliaji haramu, unyanyasaji, au ufuatiliaji usioidhinishwa wa watu bila ridhaa yao. Hutaruhusiwa kufanya uhandisi wa kinyume, kufungua, au kuharibu firmware ya vifaa vya MGuard au huduma za API.",
	},
	{
		Title:   "Limitation of Liability",
		Body:    "MGuard is a tracking and telemetry management aid, not an insurance policy. MGuard is not liable for vehicle theft, recovery failures, road accidents, or indirect commercial losses. Vehicle owners must maintain appropriate commercial insurance.",
		TitleSW: "Kikomo cha Dhima",
		BodySW:  "MGuard ni msaada wa ufuatiliaji na usimamizi wa telemetri, si mkataba wa bima. MGuard haiwajibiki kwa wizi wa gari, kushindwa kwa urejeshaji, ajali barabarani, au hasara zisizo za moja kwa moja za kibiashara. Wamiliki wa magari lazima wawe na bima ya kibiashara inayofaa.",
	},
	{
		Title:   "Termination",
		Body:    "MGuard reserves the right to suspend or terminate accounts that violate these terms or engage in abusive activity. You may cancel your account at any time from the app's settings, or by contacting customer support.",
		TitleSW: "Kusitishwa",
		BodySW:  "MGuard inahifadhi haki ya kusimamisha au kufunga akaunti zinazokiuka masharti haya au zinazojihusisha na shughuli za matusi. Unaweza kufuta akaunti yako wakati wowote kupitia mipangilio ya programu, au kwa kuwasiliana na huduma kwa wateja.",
	},
	{
		Title:   "Governing Law",
		Body:    "These terms are governed by and construed in accordance with the laws of the Republic of Kenya. Any unresolved legal disputes shall be settled through arbitration under Kenyan law.",
		TitleSW: "Sheria Inayotumika",
		BodySW:  "Masharti haya yanatawaliwa na kufasiriwa kulingana na sheria za Jamhuri ya Kenya. Mgogoro wowote wa kisheria ambao haujatatuliwa utasuluhishwa kwa upatanishi chini ya sheria za Kenya.",
	},
}

var Privacy = []Clause{
	{
		Title:   "Overview",
		Body:    "MGuard respects your privacy and is committed to protecting your personal information and vehicle telemetry data in compliance with the Kenya Data Protection Act, 2019 and relevant privacy regulations.",
		TitleSW: "Muhtasari",
		BodySW:  "MGuard inaheshimu faragha yako na imejitolea kulinda taarifa zako binafsi na data ya telemetri ya gari lako kwa mujibu wa Sheria ya Ulinzi wa Data ya Kenya, 2019 na kanuni nyingine husika za faragha.",
	},
	{
		Title:   "Information We Collect",
		Body:    "We collect account details (name, email address, phone number), vehicle information (license plate, motorcycle model), and device telemetry transmitted by installed hardware (GPS coordinates, speed, heading, ignition status, battery voltage, and 15-digit IMEI).",
		TitleSW: "Taarifa Tunazokusanya",
		BodySW:  "Tunakusanya taarifa za akaunti (jina, barua pepe, namba ya simu), taarifa za gari (namba ya usajili, aina ya pikipiki), na telemetri ya kifaa inayotumwa na vifaa vilivyosakinishwa (viwianishi vya GPS, kasi, mwelekeo, hali ya injini, volteji ya betri, na IMEI ya tarakimu 15).",
	},
	{
		Title:   "How We Use Your Data",
		Body:    "Your information is used solely to provide real-time tracking, generate historical ride breadcrumbs, calculate geofence breach notifications, detect vehicle tampering, and maintain service security. We do not sell or rent your personal or location data to third parties.",
		TitleSW: "Jinsi Tunavyotumia Data Yako",
		BodySW:  "Taarifa zako zinatumika tu kutoa ufuatiliaji wa wakati halisi, kutengeneza historia ya safari, kukokotoa arifa za kuvuka geofence, kugundua kuingiliwa kwa gari, na kudumisha usalama wa huduma. Hatuuzi wala kukodisha data yako binafsi au ya eneo kwa watu wengine.",
	},
	{
		Title:   "Data Sharing",
		Body:    "We only share telemetry with essential service providers: cellular carriers for SIM connectivity, secure cloud hosting infrastructure, and OpenAI for the MGuard Assistant. In cases of motorcycle theft or legal investigation, location vectors may be shared with law enforcement authorities upon verified request.\n\nMGuard Assistant: when you use the assistant to check your motorcycle, the bike's last known location, address, ignition, speed, battery and signal status, and recent events are sent to OpenAI, a third-party AI service, so it can write the reply in your chosen language. MGuard keeps a record of your assistant questions and replies for 90 days so you can review them, and you can clear that history at any time. OpenAI processes the data it receives under its own API terms, and that processing may take place outside Kenya. Use of the assistant is optional.",
		TitleSW: "Kushiriki Data",
		BodySW:  "Tunashiriki telemetri tu na watoa huduma muhimu: kampuni za simu kwa muunganisho wa SIM, na miundombinu salama ya uhifadhi wa wingu wa MGuard na OpenAI kwa Msaidizi wa MGuard. Katika kesi za wizi wa pikipiki au uchunguzi wa kisheria, taarifa za mahali zinaweza kushirikiwa na vyombo vya sheria kwa ombi lililothibitishwa.\n\nMsaidizi wa MGuard: unapotumia msaidizi kuangalia pikipiki yako, taarifa ya mwisho inayojulikana ya eneo, anwani, hali ya injini, kasi, betri na mawimbi, pamoja na matukio ya hivi karibuni, hutumwa kwa OpenAI, huduma ya tatu ya AI, ili iandike jibu kwa lugha uliyochagua. MGuard huhifadhi maswali na majibu yako ya msaidizi kwa siku 90 ili uweze kuyapitia, na unaweza kufuta historia hiyo wakati wowote. OpenAI huchakata data inayopokea chini ya masharti yake ya API, na uchakataji huo unaweza kufanyika nje ya Kenya. Matumizi ya msaidizi ni hiari.",
	},
	{
		Title:   "Data Retention",
		Body:    "High-resolution location breadcrumbs are retained for 90 days for operational replay. Historical trip aggregates are kept for up to 12 months, after which they are permanently expunged or anonymized. Assistant history is kept for 90 days and can be cleared earlier from the assistant. If you delete your account, your data and the tracking records linked to it are deleted.",
		TitleSW: "Muda wa Kuhifadhi Data",
		BodySW:  "Vipande vya mwendo wa juu vya eneo vinahifadhiwa kwa siku 90 kwa ajili ya kuonyesha upya safari. Muhtasari wa safari za kihistoria huhifadhiwa hadi miezi 12, baada ya hapo hufutwa kabisa au kufichwa utambulisho. Historia ya msaidizi huhifadhiwa kwa siku 90 na inaweza kufutwa mapema kutoka kwenye msaidizi. Ukifuta akaunti yako, data yako na rekodi za ufuatiliaji zinazohusiana nayo hufutwa.",
	},
	{
		Title:   "Security Measures",
		Body:    "We use industry-standard encryption protocols (TLS 1.3 in transit, salted bcrypt password hashes, and access-controlled databases) to safeguard your vehicle records from unauthorized access.",
		TitleSW: "Hatua za Usalama",
		BodySW:  "Tunatumia itifaki za usimbaji zinazotumika sana (TLS 1.3 wakati wa usafirishaji, heshi za nenosiri zilizotiwa chumvi za bcrypt, na hifadhidata zilizolindwa kwa udhibiti wa ufikiaji) kulinda rekodi za gari lako dhidi ya ufikiaji usioidhinishwa.",
	},
	{
		Title:   "Your Rights",
		Body:    "Under the Kenya Data Protection Act, you have the right to access the data we hold about you, request corrections to erroneous records, and request the deletion of your account and associated tracking records upon discontinuing service. You can delete your account yourself from the app's settings.",
		TitleSW: "Haki Zako",
		BodySW:  "Chini ya Sheria ya Ulinzi wa Data ya Kenya, una haki ya kupata data tunayoshikilia kuhusu wewe, kuomba marekebisho ya rekodi zisizo sahihi, na kuomba kufutwa kwa akaunti yako na rekodi za ufuatiliaji zinazohusiana nayo unapoacha kutumia huduma. Unaweza kufuta akaunti yako mwenyewe kutoka kwenye mipangilio ya programu.",
	},
	{
		Title:   "Contact Us",
		Body:    "If you have questions about this Privacy Policy or wish to exercise your data rights, contact us at privacy@jullie.tech.",
		TitleSW: "Wasiliana Nasi",
		BodySW:  "Ikiwa una maswali kuhusu Sera hii ya Faragha au unataka kutumia haki zako za data, wasiliana nasi kupitia privacy@jullie.tech.",
	},
}
