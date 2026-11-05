--
-- PostgreSQL database dump
--

-- Dumped from database version 16.4
-- Dumped by pg_dump version 16.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

ALTER TABLE IF EXISTS ONLY public.tracker_assignments DROP CONSTRAINT IF EXISTS tracker_assignments_unassigned_by_fkey;
ALTER TABLE IF EXISTS ONLY public.tracker_assignments DROP CONSTRAINT IF EXISTS tracker_assignments_tracker_id_fkey;
ALTER TABLE IF EXISTS ONLY public.tracker_assignments DROP CONSTRAINT IF EXISTS tracker_assignments_bike_id_fkey;
ALTER TABLE IF EXISTS ONLY public.tracker_assignments DROP CONSTRAINT IF EXISTS tracker_assignments_assigned_by_fkey;
ALTER TABLE IF EXISTS ONLY public.sessions DROP CONSTRAINT IF EXISTS sessions_user_id_fkey;
ALTER TABLE IF EXISTS ONLY public.password_reset_tokens DROP CONSTRAINT IF EXISTS password_reset_tokens_user_id_fkey;
ALTER TABLE IF EXISTS ONLY public.geofences DROP CONSTRAINT IF EXISTS geofences_created_by_fkey;
ALTER TABLE IF EXISTS ONLY public.bikes DROP CONSTRAINT IF EXISTS bikes_user_id_fkey;
ALTER TABLE IF EXISTS ONLY public.bikes DROP CONSTRAINT IF EXISTS bikes_geofence_id_fkey;
ALTER TABLE IF EXISTS ONLY public.bikes DROP CONSTRAINT IF EXISTS bikes_driver_contact_id_fkey;
ALTER TABLE IF EXISTS ONLY public.bike_events DROP CONSTRAINT IF EXISTS bike_events_bike_id_fkey;
ALTER TABLE IF EXISTS ONLY public.bike_breadcrumbs DROP CONSTRAINT IF EXISTS bike_breadcrumbs_bike_id_fkey;
DROP INDEX IF EXISTS public.idx_unique_active_tracker;
DROP INDEX IF EXISTS public.idx_unique_active_bike;
DROP INDEX IF EXISTS public.idx_trackers_status;
DROP INDEX IF EXISTS public.idx_trackers_phone;
DROP INDEX IF EXISTS public.idx_trackers_last_online;
DROP INDEX IF EXISTS public.idx_sessions_user;
DROP INDEX IF EXISTS public.idx_sessions_refresh_token;
DROP INDEX IF EXISTS public.idx_pwd_resets_token;
DROP INDEX IF EXISTS public.idx_geofences_boundary;
DROP INDEX IF EXISTS public.idx_geofences_active_boundary;
DROP INDEX IF EXISTS public.idx_breadcrumbs_bike_time;
DROP INDEX IF EXISTS public.idx_breadcrumbs_bike_breaches;
DROP INDEX IF EXISTS public.idx_bikes_user_status;
DROP INDEX IF EXISTS public.idx_bikes_user_id;
DROP INDEX IF EXISTS public.idx_bikes_user_created;
DROP INDEX IF EXISTS public.idx_bikes_imei;
DROP INDEX IF EXISTS public.idx_bikes_geofence;
DROP INDEX IF EXISTS public.idx_bikes_driver;
DROP INDEX IF EXISTS public.idx_bike_events_bike_time;
DROP INDEX IF EXISTS public.idx_assignments_tracker;
DROP INDEX IF EXISTS public.idx_assignments_bike;
ALTER TABLE IF EXISTS ONLY public.users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE IF EXISTS ONLY public.users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE IF EXISTS ONLY public.trackers DROP CONSTRAINT IF EXISTS trackers_pkey;
ALTER TABLE IF EXISTS ONLY public.trackers DROP CONSTRAINT IF EXISTS trackers_imei_key;
ALTER TABLE IF EXISTS ONLY public.tracker_assignments DROP CONSTRAINT IF EXISTS tracker_assignments_pkey;
ALTER TABLE IF EXISTS ONLY public.sessions DROP CONSTRAINT IF EXISTS sessions_refresh_token_hash_key;
ALTER TABLE IF EXISTS ONLY public.sessions DROP CONSTRAINT IF EXISTS sessions_pkey;
ALTER TABLE IF EXISTS ONLY public.password_reset_tokens DROP CONSTRAINT IF EXISTS password_reset_tokens_token_hash_key;
ALTER TABLE IF EXISTS ONLY public.password_reset_tokens DROP CONSTRAINT IF EXISTS password_reset_tokens_pkey;
ALTER TABLE IF EXISTS ONLY public.geofences DROP CONSTRAINT IF EXISTS geofences_pkey;
ALTER TABLE IF EXISTS ONLY public.driver_contacts DROP CONSTRAINT IF EXISTS driver_contacts_pkey;
ALTER TABLE IF EXISTS ONLY public.bikes DROP CONSTRAINT IF EXISTS bikes_pkey;
ALTER TABLE IF EXISTS ONLY public.bikes DROP CONSTRAINT IF EXISTS bikes_imei_key;
ALTER TABLE IF EXISTS ONLY public.bike_events DROP CONSTRAINT IF EXISTS bike_events_pkey;
ALTER TABLE IF EXISTS ONLY public.bike_breadcrumbs DROP CONSTRAINT IF EXISTS bike_breadcrumbs_pkey;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.trackers;
DROP TABLE IF EXISTS public.tracker_assignments;
DROP TABLE IF EXISTS public.sessions;
DROP TABLE IF EXISTS public.password_reset_tokens;
DROP TABLE IF EXISTS public.geofences;
DROP TABLE IF EXISTS public.driver_contacts;
DROP TABLE IF EXISTS public.bikes;
DROP TABLE IF EXISTS public.bike_events;
DROP TABLE IF EXISTS public.bike_breadcrumbs;
DROP EXTENSION IF EXISTS postgis_topology;
DROP EXTENSION IF EXISTS postgis_tiger_geocoder;
DROP EXTENSION IF EXISTS postgis;
DROP EXTENSION IF EXISTS fuzzystrmatch;
DROP SCHEMA IF EXISTS topology;
DROP SCHEMA IF EXISTS tiger_data;
DROP SCHEMA IF EXISTS tiger;
--
-- Name: tiger; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA tiger;


ALTER SCHEMA tiger OWNER TO postgres;

--
-- Name: tiger_data; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA tiger_data;


ALTER SCHEMA tiger_data OWNER TO postgres;

--
-- Name: topology; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA topology;


ALTER SCHEMA topology OWNER TO postgres;

--
-- Name: SCHEMA topology; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON SCHEMA topology IS 'PostGIS Topology schema';


--
-- Name: fuzzystrmatch; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS fuzzystrmatch WITH SCHEMA public;


--
-- Name: EXTENSION fuzzystrmatch; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION fuzzystrmatch IS 'determine similarities and distance between strings';


--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;


--
-- Name: EXTENSION postgis; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION postgis IS 'PostGIS geometry and geography spatial types and functions';


--
-- Name: postgis_tiger_geocoder; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis_tiger_geocoder WITH SCHEMA tiger;


--
-- Name: EXTENSION postgis_tiger_geocoder; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION postgis_tiger_geocoder IS 'PostGIS tiger geocoder and reverse geocoder';


--
-- Name: postgis_topology; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis_topology WITH SCHEMA topology;


--
-- Name: EXTENSION postgis_topology; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION postgis_topology IS 'PostGIS topology spatial types and functions';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: bike_breadcrumbs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bike_breadcrumbs (
    id bigint NOT NULL,
    bike_id character varying(32) NOT NULL,
    latitude numeric(10,7) NOT NULL,
    longitude numeric(10,7) NOT NULL,
    heading_deg numeric(5,2) DEFAULT 0.0 NOT NULL,
    speed_kmh numeric(5,2) DEFAULT 0.0 NOT NULL,
    ignition boolean DEFAULT false NOT NULL,
    battery_percent integer DEFAULT 100 NOT NULL,
    address character varying(255) DEFAULT ''::character varying NOT NULL,
    geofence_breach boolean DEFAULT false NOT NULL,
    recorded_at timestamp with time zone NOT NULL
);


ALTER TABLE public.bike_breadcrumbs OWNER TO postgres;

--
-- Name: bike_breadcrumbs_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.bike_breadcrumbs ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.bike_breadcrumbs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: bike_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bike_events (
    id bigint NOT NULL,
    bike_id character varying(32) NOT NULL,
    event_type character varying(50) NOT NULL,
    latitude numeric(10,7) NOT NULL,
    longitude numeric(10,7) NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.bike_events OWNER TO postgres;

--
-- Name: bike_events_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.bike_events ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.bike_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: bikes; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bikes (
    id character varying(32) NOT NULL,
    registration_number character varying(32) NOT NULL,
    label character varying(64) DEFAULT ''::character varying NOT NULL,
    make character varying(64) DEFAULT ''::character varying NOT NULL,
    model character varying(64) DEFAULT ''::character varying NOT NULL,
    year smallint DEFAULT 2024 NOT NULL,
    color character varying(32) DEFAULT ''::character varying NOT NULL,
    imei character varying(32) NOT NULL,
    status character varying(16) DEFAULT 'STOPPED'::character varying NOT NULL,
    ignition boolean DEFAULT false NOT NULL,
    speed_kmh numeric(5,2) DEFAULT 0.0 NOT NULL,
    heading_deg numeric(5,2) DEFAULT 0.0 NOT NULL,
    latitude numeric(10,7) DEFAULT 0.0 NOT NULL,
    longitude numeric(10,7) DEFAULT 0.0 NOT NULL,
    current_address character varying(255) DEFAULT ''::character varying NOT NULL,
    sub_county character varying(100) DEFAULT ''::character varying NOT NULL,
    county character varying(100) DEFAULT ''::character varying NOT NULL,
    battery_percent integer DEFAULT 100 NOT NULL,
    signal_quality_csq integer DEFAULT 20 NOT NULL,
    last_telemetry_at timestamp with time zone,
    geofence_id character varying(32),
    driver_contact_id character varying(32),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_immobilized boolean DEFAULT false NOT NULL,
    user_id character varying(32) NOT NULL,
    CONSTRAINT bikes_status_check CHECK (((status)::text = ANY ((ARRAY['RUNNING'::character varying, 'STOPPED'::character varying])::text[])))
);


ALTER TABLE public.bikes OWNER TO postgres;

--
-- Name: driver_contacts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.driver_contacts (
    id character varying(32) NOT NULL,
    name character varying(150) NOT NULL,
    phone character varying(20) NOT NULL,
    id_number character varying(32) DEFAULT ''::character varying NOT NULL,
    license_number character varying(32) DEFAULT ''::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.driver_contacts OWNER TO postgres;

--
-- Name: geofences; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.geofences (
    id character varying(32) NOT NULL,
    created_by character varying(32) NOT NULL,
    name character varying(100) NOT NULL,
    description character varying(500) DEFAULT ''::character varying NOT NULL,
    boundary public.geometry(Polygon,4326),
    center_lat numeric(10,7),
    center_lng numeric(10,7),
    radius_meters numeric(10,2),
    alert_on_exit boolean DEFAULT true NOT NULL,
    alert_on_entry boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    status character varying(16) DEFAULT 'ACTIVE'::character varying NOT NULL,
    suspended_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT geofences_status_check CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'SUSPENDED'::character varying])::text[])))
);


ALTER TABLE public.geofences OWNER TO postgres;

--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.password_reset_tokens (
    id character varying(36) NOT NULL,
    user_id character varying(32) NOT NULL,
    token_hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.password_reset_tokens OWNER TO postgres;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sessions (
    id character varying(36) NOT NULL,
    user_id character varying(32) NOT NULL,
    refresh_token_hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    user_agent text DEFAULT ''::text NOT NULL,
    ip_address character varying(45) DEFAULT ''::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.sessions OWNER TO postgres;

--
-- Name: tracker_assignments; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tracker_assignments (
    id character varying(32) NOT NULL,
    tracker_id character varying(32) NOT NULL,
    bike_id character varying(32) NOT NULL,
    assigned_by character varying(32) NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    unassigned_by character varying(32),
    unassigned_at timestamp with time zone,
    unassign_reason character varying(100) DEFAULT ''::character varying NOT NULL,
    status character varying(16) DEFAULT 'ACTIVE'::character varying NOT NULL,
    notes text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tracker_assignments_status_check CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'ENDED'::character varying])::text[])))
);


ALTER TABLE public.tracker_assignments OWNER TO postgres;

--
-- Name: trackers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.trackers (
    id character varying(32) NOT NULL,
    imei character varying(32) NOT NULL,
    phone_number character varying(20) NOT NULL,
    sim_iccid character varying(32) DEFAULT ''::character varying NOT NULL,
    model character varying(64) DEFAULT ''::character varying NOT NULL,
    hardware_version character varying(32) DEFAULT ''::character varying NOT NULL,
    software_version character varying(32) DEFAULT ''::character varying NOT NULL,
    status character varying(16) DEFAULT 'AVAILABLE'::character varying NOT NULL,
    last_online_at timestamp with time zone,
    notes character varying(255) DEFAULT ''::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT trackers_status_check CHECK (((status)::text = ANY ((ARRAY['AVAILABLE'::character varying, 'ASSIGNED'::character varying, 'MAINTENANCE'::character varying, 'DECOMMISSIONED'::character varying])::text[])))
);


ALTER TABLE public.trackers OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id character varying(32) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash text NOT NULL,
    name character varying(150) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Data for Name: bike_breadcrumbs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.bike_breadcrumbs (id, bike_id, latitude, longitude, heading_deg, speed_kmh, ignition, battery_percent, address, geofence_breach, recorded_at) FROM stdin;
1	bik_55ccd33c4a694726a31b0b55	-1.2885000	36.8235000	0.00	0.00	f	100		f	2026-09-26 21:42:09.853938+00
2	bik_55ccd33c4a694726a31b0b55	-1.2915000	36.8285000	120.97	83.52	t	100		f	2026-09-26 21:42:37.794827+00
3	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8310000	120.97	33.80	t	100		f	2026-09-26 21:43:12.313046+00
4	bik_55ccd33c4a694726a31b0b55	-1.2890000	36.8260000	308.67	6.08	t	100		f	2026-09-26 21:50:13.965468+00
5	bik_55ccd33c4a694726a31b0b55	-1.2855000	36.8220000	311.19	0.00	f	100		f	2026-09-26 21:50:15.007132+00
6	bik_b8fd74db9c4341be851d4637	-1.2921000	36.8219000	90.00	42.80	t	98		f	2026-09-27 09:07:23.16976+00
7	bik_55ccd33c4a694726a31b0b55	-1.2833000	36.8167000	144.30	35.00	t	92	Kenyatta Ave, CBD	f	2026-09-25 07:30:00+00
8	bik_55ccd33c4a694726a31b0b55	-1.2841000	36.8172750	144.30	45.10	t	92	Kenyatta Ave, CBD	f	2026-09-25 07:30:43+00
9	bik_55ccd33c4a694726a31b0b55	-1.2849000	36.8178500	144.30	45.90	t	92	Haile Selassie Ave, CBD	f	2026-09-25 07:31:27+00
10	bik_55ccd33c4a694726a31b0b55	-1.2857000	36.8184250	144.30	36.70	t	92	Haile Selassie Ave, CBD	f	2026-09-25 07:32:11+00
11	bik_55ccd33c4a694726a31b0b55	-1.2865000	36.8190000	186.30	47.00	t	92	Haile Selassie Ave, CBD	f	2026-09-25 07:32:55+00
12	bik_55ccd33c4a694726a31b0b55	-1.2876250	36.8188750	186.30	42.20	t	92	Haile Selassie Ave, CBD	f	2026-09-25 07:33:38+00
13	bik_55ccd33c4a694726a31b0b55	-1.2887500	36.8187500	186.30	30.80	t	92	Uhuru Highway Roundabout	f	2026-09-25 07:34:22+00
14	bik_55ccd33c4a694726a31b0b55	-1.2898750	36.8186250	186.30	23.30	t	92	Uhuru Highway Roundabout	f	2026-09-25 07:35:06+00
15	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.8185000	228.40	36.70	t	92	Uhuru Highway Roundabout	f	2026-09-25 07:35:50+00
16	bik_55ccd33c4a694726a31b0b55	-1.2920000	36.8173750	228.40	25.90	t	92	Uhuru Highway Roundabout	f	2026-09-25 07:36:33+00
17	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8162500	228.40	23.50	t	92	Ragati Rd, Upper Hill	f	2026-09-25 07:37:17+00
18	bik_55ccd33c4a694726a31b0b55	-1.2940000	36.8151250	228.40	31.60	t	92	Ragati Rd, Upper Hill	f	2026-09-25 07:38:01+00
19	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.8140000	235.00	23.30	t	91	Ragati Rd, Upper Hill	f	2026-09-25 07:38:45+00
20	bik_55ccd33c4a694726a31b0b55	-1.2958750	36.8127500	235.00	26.50	t	91	Ragati Rd, Upper Hill	f	2026-09-25 07:39:28+00
21	bik_55ccd33c4a694726a31b0b55	-1.2967500	36.8115000	235.00	37.60	t	91	Hospital Rd, Upper Hill	f	2026-09-25 07:40:12+00
22	bik_55ccd33c4a694726a31b0b55	-1.2976250	36.8102500	235.00	46.30	t	91	Hospital Rd, Upper Hill	f	2026-09-25 07:40:56+00
23	bik_55ccd33c4a694726a31b0b55	-1.2985000	36.8090000	256.00	31.60	t	91	Hospital Rd, Upper Hill	f	2026-09-25 07:41:40+00
24	bik_55ccd33c4a694726a31b0b55	-1.2990000	36.8070000	256.00	42.90	t	91	Hospital Rd, Upper Hill	f	2026-09-25 07:42:23+00
25	bik_55ccd33c4a694726a31b0b55	-1.2995000	36.8050000	256.00	46.90	t	91	Elgon Rd, Upper Hill	f	2026-09-25 07:43:07+00
26	bik_55ccd33c4a694726a31b0b55	-1.3000000	36.8030000	256.00	39.90	t	91	Elgon Rd, Upper Hill	f	2026-09-25 07:43:51+00
27	bik_55ccd33c4a694726a31b0b55	-1.3005000	36.8010000	299.40	46.30	t	90	Elgon Rd, Upper Hill	f	2026-09-25 07:44:35+00
28	bik_55ccd33c4a694726a31b0b55	-1.2993750	36.7990000	299.40	44.60	t	90	Elgon Rd, Upper Hill	f	2026-09-25 07:45:18+00
29	bik_55ccd33c4a694726a31b0b55	-1.2982500	36.7970000	299.40	34.10	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:46:02+00
30	bik_55ccd33c4a694726a31b0b55	-1.2971250	36.7950000	299.40	24.40	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:46:46+00
31	bik_55ccd33c4a694726a31b0b55	-1.2960000	36.7930000	308.70	39.90	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:47:30+00
32	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.7917500	308.70	28.50	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:48:13+00
33	bik_55ccd33c4a694726a31b0b55	-1.2940000	36.7905000	308.70	23.00	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:48:57+00
34	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.7892500	308.70	28.60	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:49:41+00
35	bik_55ccd33c4a694726a31b0b55	-1.2920000	36.7880000	303.70	24.40	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:50:25+00
36	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.7865000	303.70	24.50	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:51:08+00
37	bik_55ccd33c4a694726a31b0b55	-1.2900000	36.7850000	303.70	34.20	t	90	Yaya Centre, Kilimani	f	2026-09-25 07:51:52+00
38	bik_55ccd33c4a694726a31b0b55	-1.2890000	36.7835000	303.70	44.60	t	90	Yaya Centre, Kilimani	f	2026-09-25 07:52:36+00
39	bik_55ccd33c4a694726a31b0b55	-1.2880000	36.7820000	303.70	28.60	t	89	Yaya Centre, Kilimani	f	2026-09-25 07:53:20+00
40	bik_55ccd33c4a694726a31b0b55	-1.2870000	36.7805000	303.70	40.00	t	89	Yaya Centre, Kilimani	f	2026-09-25 07:54:03+00
41	bik_55ccd33c4a694726a31b0b55	-1.2860000	36.7790000	303.70	46.90	t	89	Ring Rd Kilimani	f	2026-09-25 07:54:47+00
42	bik_55ccd33c4a694726a31b0b55	-1.2850000	36.7775000	303.70	42.80	t	89	Ring Rd Kilimani	f	2026-09-25 07:55:31+00
43	bik_55ccd33c4a694726a31b0b55	-1.2840000	36.7760000	50.20	44.60	t	89	Ring Rd Kilimani	f	2026-09-25 07:56:15+00
44	bik_55ccd33c4a694726a31b0b55	-1.2827500	36.7775000	50.20	46.20	t	89	Ring Rd Kilimani	f	2026-09-25 07:56:58+00
45	bik_55ccd33c4a694726a31b0b55	-1.2815000	36.7790000	50.20	37.50	t	89	Riverside Dr, Westlands	f	2026-09-25 07:57:42+00
46	bik_55ccd33c4a694726a31b0b55	-1.2802500	36.7805000	50.20	26.50	t	89	Riverside Dr, Westlands	f	2026-09-25 07:58:26+00
47	bik_55ccd33c4a694726a31b0b55	-1.2790000	36.7820000	64.40	42.80	t	88	Riverside Dr, Westlands	f	2026-09-25 07:59:10+00
48	bik_55ccd33c4a694726a31b0b55	-1.2762500	36.7877500	64.40	31.50	t	88	Riverside Dr, Westlands	f	2026-09-25 07:59:53+00
49	bik_55ccd33c4a694726a31b0b55	-1.2735000	36.7935000	64.40	23.50	t	88	Waiyaki Way, Westlands	f	2026-09-25 08:00:37+00
50	bik_55ccd33c4a694726a31b0b55	-1.2707500	36.7992500	64.40	26.00	t	88	Waiyaki Way, Westlands	f	2026-09-25 08:01:21+00
51	bik_55ccd33c4a694726a31b0b55	-1.2680000	36.8050000	54.50	26.50	t	88	Waiyaki Way, Westlands	f	2026-09-25 08:02:05+00
52	bik_55ccd33c4a694726a31b0b55	-1.2667500	36.8067500	54.50	23.30	t	88	Waiyaki Way, Westlands	f	2026-09-25 08:02:48+00
53	bik_55ccd33c4a694726a31b0b55	-1.2655000	36.8085000	54.50	30.90	t	88	Sarit Centre, Westlands	f	2026-09-25 08:03:32+00
54	bik_55ccd33c4a694726a31b0b55	-1.2642500	36.8102500	54.50	42.30	t	88	Sarit Centre, Westlands	f	2026-09-25 08:04:16+00
55	bik_55ccd33c4a694726a31b0b55	-1.2630000	36.8120000	54.50	0.00	f	88	Sarit Centre, Westlands	f	2026-09-25 08:05:00+00
56	bik_55ccd33c4a694726a31b0b55	-1.2630000	36.8120000	219.80	38.00	t	92	Sarit Centre, Westlands	f	2026-09-25 16:45:00+00
57	bik_55ccd33c4a694726a31b0b55	-1.2645000	36.8107500	219.80	48.10	t	92	Sarit Centre, Westlands	f	2026-09-25 16:46:01+00
58	bik_55ccd33c4a694726a31b0b55	-1.2660000	36.8095000	219.80	48.90	t	92	Westlands Roundabout	f	2026-09-25 16:47:02+00
59	bik_55ccd33c4a694726a31b0b55	-1.2675000	36.8082500	219.80	39.70	t	92	Westlands Roundabout	f	2026-09-25 16:48:04+00
60	bik_55ccd33c4a694726a31b0b55	-1.2690000	36.8070000	131.20	50.00	t	92	Westlands Roundabout	f	2026-09-25 16:49:05+00
61	bik_55ccd33c4a694726a31b0b55	-1.2707500	36.8090000	131.20	45.20	t	92	Westlands Roundabout	f	2026-09-25 16:50:06+00
62	bik_55ccd33c4a694726a31b0b55	-1.2725000	36.8110000	131.20	33.80	t	92	Museum Hill Interchange	f	2026-09-25 16:51:08+00
63	bik_55ccd33c4a694726a31b0b55	-1.2742500	36.8130000	131.20	26.30	t	92	Museum Hill Interchange	f	2026-09-25 16:52:09+00
64	bik_55ccd33c4a694726a31b0b55	-1.2760000	36.8150000	156.80	39.70	t	92	Museum Hill Interchange	f	2026-09-25 16:53:10+00
65	bik_55ccd33c4a694726a31b0b55	-1.2777500	36.8157500	156.80	28.90	t	92	Museum Hill Interchange	f	2026-09-25 16:54:12+00
66	bik_55ccd33c4a694726a31b0b55	-1.2795000	36.8165000	156.80	26.50	t	92	University Way, CBD	f	2026-09-25 16:55:13+00
67	bik_55ccd33c4a694726a31b0b55	-1.2812500	36.8172500	156.80	34.60	t	92	University Way, CBD	f	2026-09-25 16:56:14+00
68	bik_55ccd33c4a694726a31b0b55	-1.2830000	36.8180000	131.20	26.30	t	91	University Way, CBD	f	2026-09-25 16:57:16+00
69	bik_55ccd33c4a694726a31b0b55	-1.2838750	36.8190000	131.20	29.50	t	91	University Way, CBD	f	2026-09-25 16:58:17+00
70	bik_55ccd33c4a694726a31b0b55	-1.2847500	36.8200000	131.20	40.60	t	91	Moi Avenue, CBD	f	2026-09-25 16:59:19+00
71	bik_55ccd33c4a694726a31b0b55	-1.2856250	36.8210000	131.20	49.30	t	91	Moi Avenue, CBD	f	2026-09-25 17:00:20+00
72	bik_55ccd33c4a694726a31b0b55	-1.2865000	36.8220000	126.90	34.60	t	91	Moi Avenue, CBD	f	2026-09-25 17:01:21+00
73	bik_55ccd33c4a694726a31b0b55	-1.2876250	36.8235000	126.90	45.90	t	91	Moi Avenue, CBD	f	2026-09-25 17:02:23+00
74	bik_55ccd33c4a694726a31b0b55	-1.2887500	36.8250000	126.90	49.90	t	91	Railway Station, CBD	f	2026-09-25 17:03:24+00
75	bik_55ccd33c4a694726a31b0b55	-1.2898750	36.8265000	126.90	42.90	t	91	Railway Station, CBD	f	2026-09-25 17:04:25+00
76	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.8280000	143.10	49.30	t	90	Railway Station, CBD	f	2026-09-25 17:05:27+00
77	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8295000	143.10	47.60	t	90	Railway Station, CBD	f	2026-09-25 17:06:28+00
78	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.8310000	143.10	37.10	t	90	Bunyala Rd Roundabout	f	2026-09-25 17:07:29+00
79	bik_55ccd33c4a694726a31b0b55	-1.2970000	36.8325000	143.10	27.40	t	90	Bunyala Rd Roundabout	f	2026-09-25 17:08:31+00
80	bik_55ccd33c4a694726a31b0b55	-1.2990000	36.8340000	139.40	42.90	t	90	Bunyala Rd Roundabout	f	2026-09-25 17:09:32+00
81	bik_55ccd33c4a694726a31b0b55	-1.3007500	36.8355000	139.40	31.50	t	90	Bunyala Rd Roundabout	f	2026-09-25 17:10:34+00
82	bik_55ccd33c4a694726a31b0b55	-1.3025000	36.8370000	139.40	26.00	t	90	Nyayo National Stadium	f	2026-09-25 17:11:35+00
83	bik_55ccd33c4a694726a31b0b55	-1.3042500	36.8385000	139.40	31.60	t	90	Nyayo National Stadium	f	2026-09-25 17:12:36+00
84	bik_55ccd33c4a694726a31b0b55	-1.3060000	36.8400000	138.80	27.40	t	90	Nyayo National Stadium	f	2026-09-25 17:13:38+00
85	bik_55ccd33c4a694726a31b0b55	-1.3080000	36.8417500	138.80	27.50	t	90	Nyayo National Stadium	f	2026-09-25 17:14:39+00
86	bik_55ccd33c4a694726a31b0b55	-1.3100000	36.8435000	138.80	37.20	t	90	Bellevue, Mombasa Rd	f	2026-09-25 17:15:40+00
87	bik_55ccd33c4a694726a31b0b55	-1.3120000	36.8452500	138.80	47.60	t	90	Bellevue, Mombasa Rd	f	2026-09-25 17:16:42+00
88	bik_55ccd33c4a694726a31b0b55	-1.3140000	36.8470000	135.00	31.60	t	89	Bellevue, Mombasa Rd	f	2026-09-25 17:17:43+00
89	bik_55ccd33c4a694726a31b0b55	-1.3167500	36.8497500	135.00	43.00	t	89	Bellevue, Mombasa Rd	f	2026-09-25 17:18:44+00
90	bik_55ccd33c4a694726a31b0b55	-1.3195000	36.8525000	135.00	49.90	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-25 17:19:46+00
91	bik_55ccd33c4a694726a31b0b55	-1.3222500	36.8552500	135.00	45.80	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-25 17:20:47+00
92	bik_55ccd33c4a694726a31b0b55	-1.3250000	36.8580000	135.00	47.60	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-25 17:21:49+00
93	bik_55ccd33c4a694726a31b0b55	-1.3272500	36.8602500	135.00	49.20	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-25 17:22:50+00
94	bik_55ccd33c4a694726a31b0b55	-1.3295000	36.8625000	135.00	40.50	t	89	General Motors, Enterprise Rd	f	2026-09-25 17:23:51+00
95	bik_55ccd33c4a694726a31b0b55	-1.3317500	36.8647500	135.00	29.50	t	89	General Motors, Enterprise Rd	f	2026-09-25 17:24:53+00
96	bik_55ccd33c4a694726a31b0b55	-1.3340000	36.8670000	120.30	45.80	t	88	General Motors, Enterprise Rd	f	2026-09-25 17:25:54+00
97	bik_55ccd33c4a694726a31b0b55	-1.3357500	36.8700000	120.30	34.50	t	88	General Motors, Enterprise Rd	f	2026-09-25 17:26:55+00
98	bik_55ccd33c4a694726a31b0b55	-1.3375000	36.8730000	120.30	26.50	t	88	Cabanas, Mombasa Rd	f	2026-09-25 17:27:57+00
99	bik_55ccd33c4a694726a31b0b55	-1.3392500	36.8760000	120.30	29.00	t	88	Cabanas, Mombasa Rd	f	2026-09-25 17:28:58+00
100	bik_55ccd33c4a694726a31b0b55	-1.3410000	36.8790000	120.30	0.00	f	88	Cabanas, Mombasa Rd	f	2026-09-25 17:29:59+00
101	bik_55ccd33c4a694726a31b0b55	-1.2833000	36.8167000	144.30	41.00	t	92	Kenyatta Ave, CBD	f	2026-09-26 08:10:00+00
102	bik_55ccd33c4a694726a31b0b55	-1.2841000	36.8172750	144.30	51.10	t	92	Kenyatta Ave, CBD	f	2026-09-26 08:11:08+00
103	bik_55ccd33c4a694726a31b0b55	-1.2849000	36.8178500	144.30	51.90	t	92	Haile Selassie Ave, CBD	f	2026-09-26 08:12:17+00
104	bik_55ccd33c4a694726a31b0b55	-1.2857000	36.8184250	144.30	42.70	t	92	Haile Selassie Ave, CBD	f	2026-09-26 08:13:26+00
105	bik_55ccd33c4a694726a31b0b55	-1.2865000	36.8190000	186.30	53.00	t	92	Haile Selassie Ave, CBD	f	2026-09-26 08:14:35+00
106	bik_55ccd33c4a694726a31b0b55	-1.2876250	36.8188750	186.30	48.20	t	92	Haile Selassie Ave, CBD	f	2026-09-26 08:15:43+00
107	bik_55ccd33c4a694726a31b0b55	-1.2887500	36.8187500	186.30	36.80	t	92	Uhuru Highway Roundabout	f	2026-09-26 08:16:52+00
108	bik_55ccd33c4a694726a31b0b55	-1.2898750	36.8186250	186.30	29.30	t	92	Uhuru Highway Roundabout	f	2026-09-26 08:18:01+00
109	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.8185000	228.40	42.70	t	92	Uhuru Highway Roundabout	f	2026-09-26 08:19:10+00
110	bik_55ccd33c4a694726a31b0b55	-1.2920000	36.8173750	228.40	31.90	t	92	Uhuru Highway Roundabout	f	2026-09-26 08:20:18+00
111	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8162500	228.40	29.50	t	92	Ragati Rd, Upper Hill	f	2026-09-26 08:21:27+00
112	bik_55ccd33c4a694726a31b0b55	-1.2940000	36.8151250	228.40	37.60	t	92	Ragati Rd, Upper Hill	f	2026-09-26 08:22:36+00
113	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.8140000	235.00	29.30	t	91	Ragati Rd, Upper Hill	f	2026-09-26 08:23:45+00
114	bik_55ccd33c4a694726a31b0b55	-1.2958750	36.8127500	235.00	32.50	t	91	Ragati Rd, Upper Hill	f	2026-09-26 08:24:53+00
115	bik_55ccd33c4a694726a31b0b55	-1.2967500	36.8115000	235.00	43.60	t	91	Hospital Rd, Upper Hill	f	2026-09-26 08:26:02+00
116	bik_55ccd33c4a694726a31b0b55	-1.2976250	36.8102500	235.00	52.30	t	91	Hospital Rd, Upper Hill	f	2026-09-26 08:27:11+00
117	bik_55ccd33c4a694726a31b0b55	-1.2985000	36.8090000	256.00	37.60	t	91	Hospital Rd, Upper Hill	f	2026-09-26 08:28:20+00
118	bik_55ccd33c4a694726a31b0b55	-1.2990000	36.8070000	256.00	48.90	t	91	Hospital Rd, Upper Hill	f	2026-09-26 08:29:28+00
119	bik_55ccd33c4a694726a31b0b55	-1.2995000	36.8050000	256.00	52.90	t	91	Elgon Rd, Upper Hill	f	2026-09-26 08:30:37+00
120	bik_55ccd33c4a694726a31b0b55	-1.3000000	36.8030000	256.00	45.90	t	91	Elgon Rd, Upper Hill	f	2026-09-26 08:31:46+00
121	bik_55ccd33c4a694726a31b0b55	-1.3005000	36.8010000	299.40	52.30	t	90	Elgon Rd, Upper Hill	f	2026-09-26 08:32:55+00
122	bik_55ccd33c4a694726a31b0b55	-1.2993750	36.7990000	299.40	50.60	t	90	Elgon Rd, Upper Hill	f	2026-09-26 08:34:03+00
123	bik_55ccd33c4a694726a31b0b55	-1.2982500	36.7970000	299.40	40.10	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:35:12+00
124	bik_55ccd33c4a694726a31b0b55	-1.2971250	36.7950000	299.40	30.40	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:36:21+00
125	bik_55ccd33c4a694726a31b0b55	-1.2960000	36.7930000	308.70	45.90	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:37:30+00
126	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.7917500	308.70	34.50	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:38:38+00
127	bik_55ccd33c4a694726a31b0b55	-1.2940000	36.7905000	308.70	29.00	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:39:47+00
128	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.7892500	308.70	34.60	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:40:56+00
129	bik_55ccd33c4a694726a31b0b55	-1.2920000	36.7880000	303.70	30.40	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:42:05+00
130	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.7865000	303.70	30.50	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:43:13+00
131	bik_55ccd33c4a694726a31b0b55	-1.2900000	36.7850000	303.70	40.20	t	90	Yaya Centre, Kilimani	f	2026-09-26 08:44:22+00
132	bik_55ccd33c4a694726a31b0b55	-1.2890000	36.7835000	303.70	50.60	t	90	Yaya Centre, Kilimani	f	2026-09-26 08:45:31+00
133	bik_55ccd33c4a694726a31b0b55	-1.2880000	36.7820000	303.70	34.60	t	89	Yaya Centre, Kilimani	f	2026-09-26 08:46:40+00
134	bik_55ccd33c4a694726a31b0b55	-1.2870000	36.7805000	303.70	46.00	t	89	Yaya Centre, Kilimani	f	2026-09-26 08:47:48+00
135	bik_55ccd33c4a694726a31b0b55	-1.2860000	36.7790000	303.70	52.90	t	89	Ring Rd Kilimani	f	2026-09-26 08:48:57+00
136	bik_55ccd33c4a694726a31b0b55	-1.2850000	36.7775000	303.70	48.80	t	89	Ring Rd Kilimani	f	2026-09-26 08:50:06+00
137	bik_55ccd33c4a694726a31b0b55	-1.2840000	36.7760000	50.20	50.60	t	89	Ring Rd Kilimani	f	2026-09-26 08:51:15+00
138	bik_55ccd33c4a694726a31b0b55	-1.2827500	36.7775000	50.20	52.20	t	89	Ring Rd Kilimani	f	2026-09-26 08:52:23+00
139	bik_55ccd33c4a694726a31b0b55	-1.2815000	36.7790000	50.20	43.50	t	89	Riverside Dr, Westlands	f	2026-09-26 08:53:32+00
140	bik_55ccd33c4a694726a31b0b55	-1.2802500	36.7805000	50.20	32.50	t	89	Riverside Dr, Westlands	f	2026-09-26 08:54:41+00
141	bik_55ccd33c4a694726a31b0b55	-1.2790000	36.7820000	64.40	48.80	t	88	Riverside Dr, Westlands	f	2026-09-26 08:55:50+00
142	bik_55ccd33c4a694726a31b0b55	-1.2762500	36.7877500	64.40	37.50	t	88	Riverside Dr, Westlands	f	2026-09-26 08:56:58+00
143	bik_55ccd33c4a694726a31b0b55	-1.2735000	36.7935000	64.40	29.50	t	88	Waiyaki Way, Westlands	f	2026-09-26 08:58:07+00
144	bik_55ccd33c4a694726a31b0b55	-1.2707500	36.7992500	64.40	32.00	t	88	Waiyaki Way, Westlands	f	2026-09-26 08:59:16+00
145	bik_55ccd33c4a694726a31b0b55	-1.2680000	36.8050000	54.50	32.50	t	88	Waiyaki Way, Westlands	f	2026-09-26 09:00:25+00
146	bik_55ccd33c4a694726a31b0b55	-1.2667500	36.8067500	54.50	29.30	t	88	Waiyaki Way, Westlands	f	2026-09-26 09:01:33+00
147	bik_55ccd33c4a694726a31b0b55	-1.2655000	36.8085000	54.50	36.90	t	88	Sarit Centre, Westlands	f	2026-09-26 09:02:42+00
148	bik_55ccd33c4a694726a31b0b55	-1.2642500	36.8102500	54.50	48.30	t	88	Sarit Centre, Westlands	f	2026-09-26 09:03:51+00
149	bik_55ccd33c4a694726a31b0b55	-1.2630000	36.8120000	54.50	0.00	f	88	Sarit Centre, Westlands	f	2026-09-26 09:05:00+00
150	bik_55ccd33c4a694726a31b0b55	-1.2630000	36.8120000	219.80	44.00	t	92	Sarit Centre, Westlands	f	2026-09-26 14:20:00+00
151	bik_55ccd33c4a694726a31b0b55	-1.2645000	36.8107500	219.80	54.10	t	92	Sarit Centre, Westlands	f	2026-09-26 14:20:47+00
152	bik_55ccd33c4a694726a31b0b55	-1.2660000	36.8095000	219.80	54.90	t	92	Westlands Roundabout	f	2026-09-26 14:21:35+00
153	bik_55ccd33c4a694726a31b0b55	-1.2675000	36.8082500	219.80	45.70	t	92	Westlands Roundabout	f	2026-09-26 14:22:23+00
154	bik_55ccd33c4a694726a31b0b55	-1.2690000	36.8070000	131.20	56.00	t	92	Westlands Roundabout	f	2026-09-26 14:23:10+00
155	bik_55ccd33c4a694726a31b0b55	-1.2707500	36.8090000	131.20	51.20	t	92	Westlands Roundabout	f	2026-09-26 14:23:58+00
156	bik_55ccd33c4a694726a31b0b55	-1.2725000	36.8110000	131.20	39.80	t	92	Museum Hill Interchange	f	2026-09-26 14:24:46+00
157	bik_55ccd33c4a694726a31b0b55	-1.2742500	36.8130000	131.20	32.30	t	92	Museum Hill Interchange	f	2026-09-26 14:25:34+00
158	bik_55ccd33c4a694726a31b0b55	-1.2760000	36.8150000	156.80	45.70	t	92	Museum Hill Interchange	f	2026-09-26 14:26:21+00
159	bik_55ccd33c4a694726a31b0b55	-1.2777500	36.8157500	156.80	34.90	t	92	Museum Hill Interchange	f	2026-09-26 14:27:09+00
160	bik_55ccd33c4a694726a31b0b55	-1.2795000	36.8165000	156.80	32.50	t	92	University Way, CBD	f	2026-09-26 14:27:57+00
161	bik_55ccd33c4a694726a31b0b55	-1.2812500	36.8172500	156.80	40.60	t	92	University Way, CBD	f	2026-09-26 14:28:45+00
162	bik_55ccd33c4a694726a31b0b55	-1.2830000	36.8180000	131.20	32.30	t	91	University Way, CBD	f	2026-09-26 14:29:32+00
163	bik_55ccd33c4a694726a31b0b55	-1.2838750	36.8190000	131.20	35.50	t	91	University Way, CBD	f	2026-09-26 14:30:20+00
164	bik_55ccd33c4a694726a31b0b55	-1.2847500	36.8200000	131.20	46.60	t	91	Moi Avenue, CBD	f	2026-09-26 14:31:08+00
165	bik_55ccd33c4a694726a31b0b55	-1.2856250	36.8210000	131.20	55.30	t	91	Moi Avenue, CBD	f	2026-09-26 14:31:55+00
166	bik_55ccd33c4a694726a31b0b55	-1.2865000	36.8220000	126.90	40.60	t	91	Moi Avenue, CBD	f	2026-09-26 14:32:43+00
167	bik_55ccd33c4a694726a31b0b55	-1.2876250	36.8235000	126.90	51.90	t	91	Moi Avenue, CBD	f	2026-09-26 14:33:31+00
168	bik_55ccd33c4a694726a31b0b55	-1.2887500	36.8250000	126.90	55.90	t	91	Railway Station, CBD	f	2026-09-26 14:34:19+00
169	bik_55ccd33c4a694726a31b0b55	-1.2898750	36.8265000	126.90	48.90	t	91	Railway Station, CBD	f	2026-09-26 14:35:06+00
170	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.8280000	143.10	55.30	t	90	Railway Station, CBD	f	2026-09-26 14:35:54+00
171	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8295000	143.10	53.60	t	90	Railway Station, CBD	f	2026-09-26 14:36:42+00
172	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.8310000	143.10	43.10	t	90	Bunyala Rd Roundabout	f	2026-09-26 14:37:30+00
173	bik_55ccd33c4a694726a31b0b55	-1.2970000	36.8325000	143.10	33.40	t	90	Bunyala Rd Roundabout	f	2026-09-26 14:38:17+00
174	bik_55ccd33c4a694726a31b0b55	-1.2990000	36.8340000	139.40	48.90	t	90	Bunyala Rd Roundabout	f	2026-09-26 14:39:05+00
175	bik_55ccd33c4a694726a31b0b55	-1.3007500	36.8355000	139.40	37.50	t	90	Bunyala Rd Roundabout	f	2026-09-26 14:39:53+00
176	bik_55ccd33c4a694726a31b0b55	-1.3025000	36.8370000	139.40	32.00	t	90	Nyayo National Stadium	f	2026-09-26 14:40:40+00
177	bik_55ccd33c4a694726a31b0b55	-1.3042500	36.8385000	139.40	37.60	t	90	Nyayo National Stadium	f	2026-09-26 14:41:28+00
178	bik_55ccd33c4a694726a31b0b55	-1.3060000	36.8400000	138.80	33.40	t	90	Nyayo National Stadium	f	2026-09-26 14:42:16+00
179	bik_55ccd33c4a694726a31b0b55	-1.3080000	36.8417500	138.80	33.50	t	90	Nyayo National Stadium	f	2026-09-26 14:43:04+00
180	bik_55ccd33c4a694726a31b0b55	-1.3100000	36.8435000	138.80	43.20	t	90	Bellevue, Mombasa Rd	f	2026-09-26 14:43:51+00
181	bik_55ccd33c4a694726a31b0b55	-1.3120000	36.8452500	138.80	53.60	t	90	Bellevue, Mombasa Rd	f	2026-09-26 14:44:39+00
182	bik_55ccd33c4a694726a31b0b55	-1.3140000	36.8470000	135.00	37.60	t	89	Bellevue, Mombasa Rd	f	2026-09-26 14:45:27+00
183	bik_55ccd33c4a694726a31b0b55	-1.3167500	36.8497500	135.00	49.00	t	89	Bellevue, Mombasa Rd	f	2026-09-26 14:46:15+00
184	bik_55ccd33c4a694726a31b0b55	-1.3195000	36.8525000	135.00	55.90	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:47:02+00
185	bik_55ccd33c4a694726a31b0b55	-1.3222500	36.8552500	135.00	51.80	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:47:50+00
186	bik_55ccd33c4a694726a31b0b55	-1.3250000	36.8580000	135.00	53.60	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:48:38+00
187	bik_55ccd33c4a694726a31b0b55	-1.3272500	36.8602500	135.00	55.20	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:49:25+00
188	bik_55ccd33c4a694726a31b0b55	-1.3295000	36.8625000	135.00	46.50	t	89	General Motors, Enterprise Rd	f	2026-09-26 14:50:13+00
189	bik_55ccd33c4a694726a31b0b55	-1.3317500	36.8647500	135.00	35.50	t	89	General Motors, Enterprise Rd	f	2026-09-26 14:51:01+00
190	bik_55ccd33c4a694726a31b0b55	-1.3340000	36.8670000	120.30	51.80	t	88	General Motors, Enterprise Rd	f	2026-09-26 14:51:49+00
191	bik_55ccd33c4a694726a31b0b55	-1.3357500	36.8700000	120.30	40.50	t	88	General Motors, Enterprise Rd	f	2026-09-26 14:52:36+00
192	bik_55ccd33c4a694726a31b0b55	-1.3375000	36.8730000	120.30	32.50	t	88	Cabanas, Mombasa Rd	f	2026-09-26 14:53:24+00
193	bik_55ccd33c4a694726a31b0b55	-1.3392500	36.8760000	120.30	35.00	t	88	Cabanas, Mombasa Rd	f	2026-09-26 14:54:12+00
194	bik_55ccd33c4a694726a31b0b55	-1.3410000	36.8790000	120.30	0.00	f	88	Cabanas, Mombasa Rd	f	2026-09-26 14:55:00+00
195	bik_55ccd33c4a694726a31b0b55	-1.2833000	36.8167000	144.30	47.00	t	92	Kenyatta Ave, CBD	f	2026-09-27 07:50:00+00
196	bik_55ccd33c4a694726a31b0b55	-1.2841000	36.8172750	144.30	57.10	t	92	Kenyatta Ave, CBD	f	2026-09-27 07:50:56+00
197	bik_55ccd33c4a694726a31b0b55	-1.2849000	36.8178500	144.30	57.90	t	92	Haile Selassie Ave, CBD	f	2026-09-27 07:51:52+00
198	bik_55ccd33c4a694726a31b0b55	-1.2857000	36.8184250	144.30	48.70	t	92	Haile Selassie Ave, CBD	f	2026-09-27 07:52:48+00
199	bik_55ccd33c4a694726a31b0b55	-1.2865000	36.8190000	186.30	59.00	t	92	Haile Selassie Ave, CBD	f	2026-09-27 07:53:45+00
200	bik_55ccd33c4a694726a31b0b55	-1.2876250	36.8188750	186.30	54.20	t	92	Haile Selassie Ave, CBD	f	2026-09-27 07:54:41+00
201	bik_55ccd33c4a694726a31b0b55	-1.2887500	36.8187500	186.30	42.80	t	92	Uhuru Highway Roundabout	f	2026-09-27 07:55:37+00
202	bik_55ccd33c4a694726a31b0b55	-1.2898750	36.8186250	186.30	35.30	t	92	Uhuru Highway Roundabout	f	2026-09-27 07:56:33+00
203	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.8185000	228.40	48.70	t	92	Uhuru Highway Roundabout	f	2026-09-27 07:57:30+00
204	bik_55ccd33c4a694726a31b0b55	-1.2920000	36.8173750	228.40	37.90	t	92	Uhuru Highway Roundabout	f	2026-09-27 07:58:26+00
205	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8162500	228.40	35.50	t	92	Ragati Rd, Upper Hill	f	2026-09-27 07:59:22+00
206	bik_55ccd33c4a694726a31b0b55	-1.2940000	36.8151250	228.40	43.60	t	92	Ragati Rd, Upper Hill	f	2026-09-27 08:00:18+00
207	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.8140000	235.00	35.30	t	91	Ragati Rd, Upper Hill	f	2026-09-27 08:01:15+00
208	bik_55ccd33c4a694726a31b0b55	-1.2958750	36.8127500	235.00	38.50	t	91	Ragati Rd, Upper Hill	f	2026-09-27 08:02:11+00
209	bik_55ccd33c4a694726a31b0b55	-1.2967500	36.8115000	235.00	49.60	t	91	Hospital Rd, Upper Hill	f	2026-09-27 08:03:07+00
210	bik_55ccd33c4a694726a31b0b55	-1.2976250	36.8102500	235.00	58.30	t	91	Hospital Rd, Upper Hill	f	2026-09-27 08:04:03+00
211	bik_55ccd33c4a694726a31b0b55	-1.2985000	36.8090000	256.00	43.60	t	91	Hospital Rd, Upper Hill	f	2026-09-27 08:05:00+00
212	bik_55ccd33c4a694726a31b0b55	-1.2990000	36.8070000	256.00	54.90	t	91	Hospital Rd, Upper Hill	f	2026-09-27 08:05:56+00
213	bik_55ccd33c4a694726a31b0b55	-1.2995000	36.8050000	256.00	58.90	t	91	Elgon Rd, Upper Hill	f	2026-09-27 08:06:52+00
214	bik_55ccd33c4a694726a31b0b55	-1.3000000	36.8030000	256.00	51.90	t	91	Elgon Rd, Upper Hill	f	2026-09-27 08:07:48+00
215	bik_55ccd33c4a694726a31b0b55	-1.3005000	36.8010000	299.40	58.30	t	90	Elgon Rd, Upper Hill	f	2026-09-27 08:08:45+00
216	bik_55ccd33c4a694726a31b0b55	-1.2993750	36.7990000	299.40	56.60	t	90	Elgon Rd, Upper Hill	f	2026-09-27 08:09:41+00
217	bik_55ccd33c4a694726a31b0b55	-1.2982500	36.7970000	299.40	46.10	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:10:37+00
218	bik_55ccd33c4a694726a31b0b55	-1.2971250	36.7950000	299.40	36.40	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:11:33+00
219	bik_55ccd33c4a694726a31b0b55	-1.2960000	36.7930000	308.70	51.90	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:12:30+00
220	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.7917500	308.70	40.50	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:13:26+00
221	bik_55ccd33c4a694726a31b0b55	-1.2940000	36.7905000	308.70	35.00	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:14:22+00
222	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.7892500	308.70	40.60	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:15:18+00
223	bik_55ccd33c4a694726a31b0b55	-1.2920000	36.7880000	303.70	36.40	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:16:15+00
224	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.7865000	303.70	36.50	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:17:11+00
225	bik_55ccd33c4a694726a31b0b55	-1.2900000	36.7850000	303.70	46.20	t	90	Yaya Centre, Kilimani	f	2026-09-27 08:18:07+00
226	bik_55ccd33c4a694726a31b0b55	-1.2890000	36.7835000	303.70	56.60	t	90	Yaya Centre, Kilimani	f	2026-09-27 08:19:03+00
227	bik_55ccd33c4a694726a31b0b55	-1.2880000	36.7820000	303.70	40.60	t	89	Yaya Centre, Kilimani	f	2026-09-27 08:20:00+00
228	bik_55ccd33c4a694726a31b0b55	-1.2870000	36.7805000	303.70	52.00	t	89	Yaya Centre, Kilimani	f	2026-09-27 08:20:56+00
229	bik_55ccd33c4a694726a31b0b55	-1.2860000	36.7790000	303.70	58.90	t	89	Ring Rd Kilimani	f	2026-09-27 08:21:52+00
230	bik_55ccd33c4a694726a31b0b55	-1.2850000	36.7775000	303.70	54.80	t	89	Ring Rd Kilimani	f	2026-09-27 08:22:48+00
231	bik_55ccd33c4a694726a31b0b55	-1.2840000	36.7760000	50.20	56.60	t	89	Ring Rd Kilimani	f	2026-09-27 08:23:45+00
232	bik_55ccd33c4a694726a31b0b55	-1.2827500	36.7775000	50.20	58.20	t	89	Ring Rd Kilimani	f	2026-09-27 08:24:41+00
233	bik_55ccd33c4a694726a31b0b55	-1.2815000	36.7790000	50.20	49.50	t	89	Riverside Dr, Westlands	f	2026-09-27 08:25:37+00
234	bik_55ccd33c4a694726a31b0b55	-1.2802500	36.7805000	50.20	38.50	t	89	Riverside Dr, Westlands	f	2026-09-27 08:26:33+00
235	bik_55ccd33c4a694726a31b0b55	-1.2790000	36.7820000	64.40	54.80	t	88	Riverside Dr, Westlands	f	2026-09-27 08:27:30+00
236	bik_55ccd33c4a694726a31b0b55	-1.2762500	36.7877500	64.40	43.50	t	88	Riverside Dr, Westlands	f	2026-09-27 08:28:26+00
237	bik_55ccd33c4a694726a31b0b55	-1.2735000	36.7935000	64.40	35.50	t	88	Waiyaki Way, Westlands	f	2026-09-27 08:29:22+00
238	bik_55ccd33c4a694726a31b0b55	-1.2707500	36.7992500	64.40	38.00	t	88	Waiyaki Way, Westlands	f	2026-09-27 08:30:18+00
239	bik_55ccd33c4a694726a31b0b55	-1.2680000	36.8050000	54.50	38.50	t	88	Waiyaki Way, Westlands	f	2026-09-27 08:31:15+00
240	bik_55ccd33c4a694726a31b0b55	-1.2667500	36.8067500	54.50	35.30	t	88	Waiyaki Way, Westlands	f	2026-09-27 08:32:11+00
241	bik_55ccd33c4a694726a31b0b55	-1.2655000	36.8085000	54.50	42.90	t	88	Sarit Centre, Westlands	f	2026-09-27 08:33:07+00
242	bik_55ccd33c4a694726a31b0b55	-1.2642500	36.8102500	54.50	54.30	t	88	Sarit Centre, Westlands	f	2026-09-27 08:34:03+00
243	bik_55ccd33c4a694726a31b0b55	-1.2630000	36.8120000	54.50	0.00	f	88	Sarit Centre, Westlands	f	2026-09-27 08:35:00+00
244	bik_55ccd33c4a694726a31b0b55	-1.2630000	36.8120000	219.80	50.00	t	92	Sarit Centre, Westlands	f	2026-09-27 11:15:00+00
245	bik_55ccd33c4a694726a31b0b55	-1.2645000	36.8107500	219.80	60.10	t	92	Sarit Centre, Westlands	f	2026-09-27 11:16:15+00
246	bik_55ccd33c4a694726a31b0b55	-1.2660000	36.8095000	219.80	60.90	t	92	Westlands Roundabout	f	2026-09-27 11:17:30+00
247	bik_55ccd33c4a694726a31b0b55	-1.2675000	36.8082500	219.80	51.70	t	92	Westlands Roundabout	f	2026-09-27 11:18:45+00
248	bik_55ccd33c4a694726a31b0b55	-1.2690000	36.8070000	131.20	62.00	t	92	Westlands Roundabout	f	2026-09-27 11:20:00+00
249	bik_55ccd33c4a694726a31b0b55	-1.2707500	36.8090000	131.20	57.20	t	92	Westlands Roundabout	f	2026-09-27 11:21:15+00
250	bik_55ccd33c4a694726a31b0b55	-1.2725000	36.8110000	131.20	45.80	t	92	Museum Hill Interchange	f	2026-09-27 11:22:30+00
251	bik_55ccd33c4a694726a31b0b55	-1.2742500	36.8130000	131.20	38.30	t	92	Museum Hill Interchange	f	2026-09-27 11:23:45+00
252	bik_55ccd33c4a694726a31b0b55	-1.2760000	36.8150000	156.80	51.70	t	92	Museum Hill Interchange	f	2026-09-27 11:25:00+00
253	bik_55ccd33c4a694726a31b0b55	-1.2777500	36.8157500	156.80	40.90	t	92	Museum Hill Interchange	f	2026-09-27 11:26:15+00
254	bik_55ccd33c4a694726a31b0b55	-1.2795000	36.8165000	156.80	38.50	t	92	University Way, CBD	f	2026-09-27 11:27:30+00
255	bik_55ccd33c4a694726a31b0b55	-1.2812500	36.8172500	156.80	46.60	t	92	University Way, CBD	f	2026-09-27 11:28:45+00
256	bik_55ccd33c4a694726a31b0b55	-1.2830000	36.8180000	131.20	38.30	t	91	University Way, CBD	f	2026-09-27 11:30:00+00
257	bik_55ccd33c4a694726a31b0b55	-1.2838750	36.8190000	131.20	41.50	t	91	University Way, CBD	f	2026-09-27 11:31:15+00
258	bik_55ccd33c4a694726a31b0b55	-1.2847500	36.8200000	131.20	52.60	t	91	Moi Avenue, CBD	f	2026-09-27 11:32:30+00
259	bik_55ccd33c4a694726a31b0b55	-1.2856250	36.8210000	131.20	61.30	t	91	Moi Avenue, CBD	f	2026-09-27 11:33:45+00
260	bik_55ccd33c4a694726a31b0b55	-1.2865000	36.8220000	126.90	46.60	t	91	Moi Avenue, CBD	f	2026-09-27 11:35:00+00
261	bik_55ccd33c4a694726a31b0b55	-1.2876250	36.8235000	126.90	57.90	t	91	Moi Avenue, CBD	f	2026-09-27 11:36:15+00
262	bik_55ccd33c4a694726a31b0b55	-1.2887500	36.8250000	126.90	61.90	t	91	Railway Station, CBD	f	2026-09-27 11:37:30+00
263	bik_55ccd33c4a694726a31b0b55	-1.2898750	36.8265000	126.90	54.90	t	91	Railway Station, CBD	f	2026-09-27 11:38:45+00
264	bik_55ccd33c4a694726a31b0b55	-1.2910000	36.8280000	143.10	61.30	t	90	Railway Station, CBD	f	2026-09-27 11:40:00+00
265	bik_55ccd33c4a694726a31b0b55	-1.2930000	36.8295000	143.10	59.60	t	90	Railway Station, CBD	f	2026-09-27 11:41:15+00
266	bik_55ccd33c4a694726a31b0b55	-1.2950000	36.8310000	143.10	49.10	t	90	Bunyala Rd Roundabout	f	2026-09-27 11:42:30+00
267	bik_55ccd33c4a694726a31b0b55	-1.2970000	36.8325000	143.10	39.40	t	90	Bunyala Rd Roundabout	f	2026-09-27 11:43:45+00
268	bik_55ccd33c4a694726a31b0b55	-1.2990000	36.8340000	139.40	54.90	t	90	Bunyala Rd Roundabout	f	2026-09-27 11:45:00+00
269	bik_55ccd33c4a694726a31b0b55	-1.3007500	36.8355000	139.40	43.50	t	90	Bunyala Rd Roundabout	f	2026-09-27 11:46:15+00
270	bik_55ccd33c4a694726a31b0b55	-1.3025000	36.8370000	139.40	38.00	t	90	Nyayo National Stadium	f	2026-09-27 11:47:30+00
271	bik_55ccd33c4a694726a31b0b55	-1.3042500	36.8385000	139.40	43.60	t	90	Nyayo National Stadium	f	2026-09-27 11:48:45+00
272	bik_55ccd33c4a694726a31b0b55	-1.3060000	36.8400000	138.80	39.40	t	90	Nyayo National Stadium	f	2026-09-27 11:50:00+00
273	bik_55ccd33c4a694726a31b0b55	-1.3080000	36.8417500	138.80	39.50	t	90	Nyayo National Stadium	f	2026-09-27 11:51:15+00
274	bik_55ccd33c4a694726a31b0b55	-1.3100000	36.8435000	138.80	49.20	t	90	Bellevue, Mombasa Rd	f	2026-09-27 11:52:30+00
275	bik_55ccd33c4a694726a31b0b55	-1.3120000	36.8452500	138.80	59.60	t	90	Bellevue, Mombasa Rd	f	2026-09-27 11:53:45+00
276	bik_55ccd33c4a694726a31b0b55	-1.3140000	36.8470000	135.00	43.60	t	89	Bellevue, Mombasa Rd	f	2026-09-27 11:55:00+00
277	bik_55ccd33c4a694726a31b0b55	-1.3167500	36.8497500	135.00	55.00	t	89	Bellevue, Mombasa Rd	f	2026-09-27 11:56:15+00
278	bik_55ccd33c4a694726a31b0b55	-1.3195000	36.8525000	135.00	61.90	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-27 11:57:30+00
279	bik_55ccd33c4a694726a31b0b55	-1.3222500	36.8552500	135.00	57.80	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-27 11:58:45+00
280	bik_55ccd33c4a694726a31b0b55	-1.3250000	36.8580000	135.00	59.60	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-27 12:00:00+00
281	bik_55ccd33c4a694726a31b0b55	-1.3272500	36.8602500	135.00	61.20	t	89	Panari Sky Centre, Mombasa Rd	f	2026-09-27 12:01:15+00
282	bik_55ccd33c4a694726a31b0b55	-1.3295000	36.8625000	135.00	52.50	t	89	General Motors, Enterprise Rd	f	2026-09-27 12:02:30+00
283	bik_55ccd33c4a694726a31b0b55	-1.3317500	36.8647500	135.00	41.50	t	89	General Motors, Enterprise Rd	f	2026-09-27 12:03:45+00
284	bik_55ccd33c4a694726a31b0b55	-1.3340000	36.8670000	120.30	57.80	t	88	General Motors, Enterprise Rd	f	2026-09-27 12:05:00+00
285	bik_55ccd33c4a694726a31b0b55	-1.3357500	36.8700000	120.30	46.50	t	88	General Motors, Enterprise Rd	f	2026-09-27 12:06:15+00
286	bik_55ccd33c4a694726a31b0b55	-1.3375000	36.8730000	120.30	38.50	t	88	Cabanas, Mombasa Rd	f	2026-09-27 12:07:30+00
287	bik_55ccd33c4a694726a31b0b55	-1.3392500	36.8760000	120.30	41.00	t	88	Cabanas, Mombasa Rd	f	2026-09-27 12:08:45+00
288	bik_55ccd33c4a694726a31b0b55	-1.3410000	36.8790000	120.30	0.00	f	88	Cabanas, Mombasa Rd	f	2026-09-27 12:10:00+00
289	bik_b8fd74db9c4341be851d4637	-1.2630000	36.8120000	234.50	35.00	t	92	Sarit Centre, Westlands	f	2026-09-25 07:30:00+00
290	bik_b8fd74db9c4341be851d4637	-1.2642500	36.8102500	234.50	45.10	t	92	Sarit Centre, Westlands	f	2026-09-25 07:30:43+00
291	bik_b8fd74db9c4341be851d4637	-1.2655000	36.8085000	234.50	45.90	t	92	Waiyaki Way, Westlands	f	2026-09-25 07:31:27+00
292	bik_b8fd74db9c4341be851d4637	-1.2667500	36.8067500	234.50	36.70	t	92	Waiyaki Way, Westlands	f	2026-09-25 07:32:11+00
293	bik_b8fd74db9c4341be851d4637	-1.2680000	36.8050000	244.40	47.00	t	92	Waiyaki Way, Westlands	f	2026-09-25 07:32:55+00
294	bik_b8fd74db9c4341be851d4637	-1.2707500	36.7992500	244.40	42.20	t	92	Waiyaki Way, Westlands	f	2026-09-25 07:33:38+00
295	bik_b8fd74db9c4341be851d4637	-1.2735000	36.7935000	244.40	30.80	t	92	Riverside Dr, Westlands	f	2026-09-25 07:34:22+00
296	bik_b8fd74db9c4341be851d4637	-1.2762500	36.7877500	244.40	23.30	t	92	Riverside Dr, Westlands	f	2026-09-25 07:35:06+00
297	bik_b8fd74db9c4341be851d4637	-1.2790000	36.7820000	230.20	36.70	t	92	Riverside Dr, Westlands	f	2026-09-25 07:35:50+00
298	bik_b8fd74db9c4341be851d4637	-1.2802500	36.7805000	230.20	25.90	t	92	Riverside Dr, Westlands	f	2026-09-25 07:36:33+00
299	bik_b8fd74db9c4341be851d4637	-1.2815000	36.7790000	230.20	23.50	t	92	Ring Rd Kilimani	f	2026-09-25 07:37:17+00
300	bik_b8fd74db9c4341be851d4637	-1.2827500	36.7775000	230.20	31.60	t	92	Ring Rd Kilimani	f	2026-09-25 07:38:01+00
301	bik_b8fd74db9c4341be851d4637	-1.2840000	36.7760000	123.70	23.30	t	91	Ring Rd Kilimani	f	2026-09-25 07:38:45+00
302	bik_b8fd74db9c4341be851d4637	-1.2850000	36.7775000	123.70	26.50	t	91	Ring Rd Kilimani	f	2026-09-25 07:39:28+00
303	bik_b8fd74db9c4341be851d4637	-1.2860000	36.7790000	123.70	37.60	t	91	Yaya Centre, Kilimani	f	2026-09-25 07:40:12+00
304	bik_b8fd74db9c4341be851d4637	-1.2870000	36.7805000	123.70	46.30	t	91	Yaya Centre, Kilimani	f	2026-09-25 07:40:56+00
305	bik_b8fd74db9c4341be851d4637	-1.2880000	36.7820000	123.70	31.60	t	91	Yaya Centre, Kilimani	f	2026-09-25 07:41:40+00
306	bik_b8fd74db9c4341be851d4637	-1.2890000	36.7835000	123.70	42.90	t	91	Yaya Centre, Kilimani	f	2026-09-25 07:42:23+00
307	bik_b8fd74db9c4341be851d4637	-1.2900000	36.7850000	123.70	46.90	t	91	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:43:07+00
308	bik_b8fd74db9c4341be851d4637	-1.2910000	36.7865000	123.70	39.90	t	91	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:43:51+00
309	bik_b8fd74db9c4341be851d4637	-1.2920000	36.7880000	128.70	46.30	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:44:35+00
310	bik_b8fd74db9c4341be851d4637	-1.2930000	36.7892500	128.70	44.60	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-25 07:45:18+00
311	bik_b8fd74db9c4341be851d4637	-1.2940000	36.7905000	128.70	34.10	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:46:02+00
312	bik_b8fd74db9c4341be851d4637	-1.2950000	36.7917500	128.70	24.40	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:46:46+00
313	bik_b8fd74db9c4341be851d4637	-1.2960000	36.7930000	119.40	39.90	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:47:30+00
314	bik_b8fd74db9c4341be851d4637	-1.2971250	36.7950000	119.40	28.50	t	90	Ngong Rd / Adams Arcade	f	2026-09-25 07:48:13+00
315	bik_b8fd74db9c4341be851d4637	-1.2982500	36.7970000	119.40	23.00	t	90	Elgon Rd, Upper Hill	f	2026-09-25 07:48:57+00
316	bik_b8fd74db9c4341be851d4637	-1.2993750	36.7990000	119.40	28.60	t	90	Elgon Rd, Upper Hill	f	2026-09-25 07:49:41+00
317	bik_b8fd74db9c4341be851d4637	-1.3005000	36.8010000	76.00	24.40	t	90	Elgon Rd, Upper Hill	f	2026-09-25 07:50:25+00
318	bik_b8fd74db9c4341be851d4637	-1.3000000	36.8030000	76.00	24.50	t	90	Elgon Rd, Upper Hill	f	2026-09-25 07:51:08+00
319	bik_b8fd74db9c4341be851d4637	-1.2995000	36.8050000	76.00	34.20	t	90	Hospital Rd, Upper Hill	f	2026-09-25 07:51:52+00
320	bik_b8fd74db9c4341be851d4637	-1.2990000	36.8070000	76.00	44.60	t	90	Hospital Rd, Upper Hill	f	2026-09-25 07:52:36+00
321	bik_b8fd74db9c4341be851d4637	-1.2985000	36.8090000	55.00	28.60	t	89	Hospital Rd, Upper Hill	f	2026-09-25 07:53:20+00
322	bik_b8fd74db9c4341be851d4637	-1.2976250	36.8102500	55.00	40.00	t	89	Hospital Rd, Upper Hill	f	2026-09-25 07:54:03+00
323	bik_b8fd74db9c4341be851d4637	-1.2967500	36.8115000	55.00	46.90	t	89	Ragati Rd, Upper Hill	f	2026-09-25 07:54:47+00
324	bik_b8fd74db9c4341be851d4637	-1.2958750	36.8127500	55.00	42.80	t	89	Ragati Rd, Upper Hill	f	2026-09-25 07:55:31+00
325	bik_b8fd74db9c4341be851d4637	-1.2950000	36.8140000	48.40	44.60	t	89	Ragati Rd, Upper Hill	f	2026-09-25 07:56:15+00
326	bik_b8fd74db9c4341be851d4637	-1.2940000	36.8151250	48.40	46.20	t	89	Ragati Rd, Upper Hill	f	2026-09-25 07:56:58+00
327	bik_b8fd74db9c4341be851d4637	-1.2930000	36.8162500	48.40	37.50	t	89	Uhuru Highway Roundabout	f	2026-09-25 07:57:42+00
328	bik_b8fd74db9c4341be851d4637	-1.2920000	36.8173750	48.40	26.50	t	89	Uhuru Highway Roundabout	f	2026-09-25 07:58:26+00
329	bik_b8fd74db9c4341be851d4637	-1.2910000	36.8185000	6.30	42.80	t	88	Uhuru Highway Roundabout	f	2026-09-25 07:59:10+00
330	bik_b8fd74db9c4341be851d4637	-1.2898750	36.8186250	6.30	31.50	t	88	Uhuru Highway Roundabout	f	2026-09-25 07:59:53+00
331	bik_b8fd74db9c4341be851d4637	-1.2887500	36.8187500	6.30	23.50	t	88	Haile Selassie Ave, CBD	f	2026-09-25 08:00:37+00
332	bik_b8fd74db9c4341be851d4637	-1.2876250	36.8188750	6.30	26.00	t	88	Haile Selassie Ave, CBD	f	2026-09-25 08:01:21+00
333	bik_b8fd74db9c4341be851d4637	-1.2865000	36.8190000	324.30	26.50	t	88	Haile Selassie Ave, CBD	f	2026-09-25 08:02:05+00
334	bik_b8fd74db9c4341be851d4637	-1.2857000	36.8184250	324.30	23.30	t	88	Haile Selassie Ave, CBD	f	2026-09-25 08:02:48+00
335	bik_b8fd74db9c4341be851d4637	-1.2849000	36.8178500	324.30	30.90	t	88	Kenyatta Ave, CBD	f	2026-09-25 08:03:32+00
336	bik_b8fd74db9c4341be851d4637	-1.2841000	36.8172750	324.30	42.30	t	88	Kenyatta Ave, CBD	f	2026-09-25 08:04:16+00
337	bik_b8fd74db9c4341be851d4637	-1.2833000	36.8167000	324.30	0.00	f	88	Kenyatta Ave, CBD	f	2026-09-25 08:05:00+00
338	bik_b8fd74db9c4341be851d4637	-1.3410000	36.8790000	300.30	38.00	t	92	Cabanas, Mombasa Rd	f	2026-09-25 16:45:00+00
339	bik_b8fd74db9c4341be851d4637	-1.3392500	36.8760000	300.30	48.10	t	92	Cabanas, Mombasa Rd	f	2026-09-25 16:46:01+00
340	bik_b8fd74db9c4341be851d4637	-1.3375000	36.8730000	300.30	48.90	t	92	General Motors, Enterprise Rd	f	2026-09-25 16:47:02+00
341	bik_b8fd74db9c4341be851d4637	-1.3357500	36.8700000	300.30	39.70	t	92	General Motors, Enterprise Rd	f	2026-09-25 16:48:04+00
342	bik_b8fd74db9c4341be851d4637	-1.3340000	36.8670000	315.00	50.00	t	92	General Motors, Enterprise Rd	f	2026-09-25 16:49:05+00
343	bik_b8fd74db9c4341be851d4637	-1.3317500	36.8647500	315.00	45.20	t	92	General Motors, Enterprise Rd	f	2026-09-25 16:50:06+00
344	bik_b8fd74db9c4341be851d4637	-1.3295000	36.8625000	315.00	33.80	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-25 16:51:08+00
345	bik_b8fd74db9c4341be851d4637	-1.3272500	36.8602500	315.00	26.30	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-25 16:52:09+00
346	bik_b8fd74db9c4341be851d4637	-1.3250000	36.8580000	315.00	39.70	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-25 16:53:10+00
347	bik_b8fd74db9c4341be851d4637	-1.3222500	36.8552500	315.00	28.90	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-25 16:54:12+00
348	bik_b8fd74db9c4341be851d4637	-1.3195000	36.8525000	315.00	26.50	t	92	Bellevue, Mombasa Rd	f	2026-09-25 16:55:13+00
349	bik_b8fd74db9c4341be851d4637	-1.3167500	36.8497500	315.00	34.60	t	92	Bellevue, Mombasa Rd	f	2026-09-25 16:56:14+00
350	bik_b8fd74db9c4341be851d4637	-1.3140000	36.8470000	318.80	26.30	t	91	Bellevue, Mombasa Rd	f	2026-09-25 16:57:16+00
351	bik_b8fd74db9c4341be851d4637	-1.3120000	36.8452500	318.80	29.50	t	91	Bellevue, Mombasa Rd	f	2026-09-25 16:58:17+00
352	bik_b8fd74db9c4341be851d4637	-1.3100000	36.8435000	318.80	40.60	t	91	Nyayo National Stadium	f	2026-09-25 16:59:19+00
353	bik_b8fd74db9c4341be851d4637	-1.3080000	36.8417500	318.80	49.30	t	91	Nyayo National Stadium	f	2026-09-25 17:00:20+00
354	bik_b8fd74db9c4341be851d4637	-1.3060000	36.8400000	319.40	34.60	t	91	Nyayo National Stadium	f	2026-09-25 17:01:21+00
355	bik_b8fd74db9c4341be851d4637	-1.3042500	36.8385000	319.40	45.90	t	91	Nyayo National Stadium	f	2026-09-25 17:02:23+00
356	bik_b8fd74db9c4341be851d4637	-1.3025000	36.8370000	319.40	49.90	t	91	Bunyala Rd Roundabout	f	2026-09-25 17:03:24+00
357	bik_b8fd74db9c4341be851d4637	-1.3007500	36.8355000	319.40	42.90	t	91	Bunyala Rd Roundabout	f	2026-09-25 17:04:25+00
358	bik_b8fd74db9c4341be851d4637	-1.2990000	36.8340000	323.10	49.30	t	90	Bunyala Rd Roundabout	f	2026-09-25 17:05:27+00
359	bik_b8fd74db9c4341be851d4637	-1.2970000	36.8325000	323.10	47.60	t	90	Bunyala Rd Roundabout	f	2026-09-25 17:06:28+00
360	bik_b8fd74db9c4341be851d4637	-1.2950000	36.8310000	323.10	37.10	t	90	Railway Station, CBD	f	2026-09-25 17:07:29+00
361	bik_b8fd74db9c4341be851d4637	-1.2930000	36.8295000	323.10	27.40	t	90	Railway Station, CBD	f	2026-09-25 17:08:31+00
362	bik_b8fd74db9c4341be851d4637	-1.2910000	36.8280000	306.90	42.90	t	90	Railway Station, CBD	f	2026-09-25 17:09:32+00
363	bik_b8fd74db9c4341be851d4637	-1.2898750	36.8265000	306.90	31.50	t	90	Railway Station, CBD	f	2026-09-25 17:10:34+00
364	bik_b8fd74db9c4341be851d4637	-1.2887500	36.8250000	306.90	26.00	t	90	Moi Avenue, CBD	f	2026-09-25 17:11:35+00
365	bik_b8fd74db9c4341be851d4637	-1.2876250	36.8235000	306.90	31.60	t	90	Moi Avenue, CBD	f	2026-09-25 17:12:36+00
366	bik_b8fd74db9c4341be851d4637	-1.2865000	36.8220000	311.20	27.40	t	90	Moi Avenue, CBD	f	2026-09-25 17:13:38+00
367	bik_b8fd74db9c4341be851d4637	-1.2856250	36.8210000	311.20	27.50	t	90	Moi Avenue, CBD	f	2026-09-25 17:14:39+00
368	bik_b8fd74db9c4341be851d4637	-1.2847500	36.8200000	311.20	37.20	t	90	University Way, CBD	f	2026-09-25 17:15:40+00
369	bik_b8fd74db9c4341be851d4637	-1.2838750	36.8190000	311.20	47.60	t	90	University Way, CBD	f	2026-09-25 17:16:42+00
370	bik_b8fd74db9c4341be851d4637	-1.2830000	36.8180000	336.80	31.60	t	89	University Way, CBD	f	2026-09-25 17:17:43+00
371	bik_b8fd74db9c4341be851d4637	-1.2812500	36.8172500	336.80	43.00	t	89	University Way, CBD	f	2026-09-25 17:18:44+00
372	bik_b8fd74db9c4341be851d4637	-1.2795000	36.8165000	336.80	49.90	t	89	Museum Hill Interchange	f	2026-09-25 17:19:46+00
373	bik_b8fd74db9c4341be851d4637	-1.2777500	36.8157500	336.80	45.80	t	89	Museum Hill Interchange	f	2026-09-25 17:20:47+00
374	bik_b8fd74db9c4341be851d4637	-1.2760000	36.8150000	311.20	47.60	t	89	Museum Hill Interchange	f	2026-09-25 17:21:49+00
375	bik_b8fd74db9c4341be851d4637	-1.2742500	36.8130000	311.20	49.20	t	89	Museum Hill Interchange	f	2026-09-25 17:22:50+00
376	bik_b8fd74db9c4341be851d4637	-1.2725000	36.8110000	311.20	40.50	t	89	Westlands Roundabout	f	2026-09-25 17:23:51+00
377	bik_b8fd74db9c4341be851d4637	-1.2707500	36.8090000	311.20	29.50	t	89	Westlands Roundabout	f	2026-09-25 17:24:53+00
378	bik_b8fd74db9c4341be851d4637	-1.2690000	36.8070000	39.80	45.80	t	88	Westlands Roundabout	f	2026-09-25 17:25:54+00
379	bik_b8fd74db9c4341be851d4637	-1.2675000	36.8082500	39.80	34.50	t	88	Westlands Roundabout	f	2026-09-25 17:26:55+00
380	bik_b8fd74db9c4341be851d4637	-1.2660000	36.8095000	39.80	26.50	t	88	Sarit Centre, Westlands	f	2026-09-25 17:27:57+00
381	bik_b8fd74db9c4341be851d4637	-1.2645000	36.8107500	39.80	29.00	t	88	Sarit Centre, Westlands	f	2026-09-25 17:28:58+00
382	bik_b8fd74db9c4341be851d4637	-1.2630000	36.8120000	39.80	0.00	f	88	Sarit Centre, Westlands	f	2026-09-25 17:29:59+00
383	bik_b8fd74db9c4341be851d4637	-1.2630000	36.8120000	234.50	41.00	t	92	Sarit Centre, Westlands	f	2026-09-26 08:10:00+00
384	bik_b8fd74db9c4341be851d4637	-1.2642500	36.8102500	234.50	51.10	t	92	Sarit Centre, Westlands	f	2026-09-26 08:11:08+00
385	bik_b8fd74db9c4341be851d4637	-1.2655000	36.8085000	234.50	51.90	t	92	Waiyaki Way, Westlands	f	2026-09-26 08:12:17+00
386	bik_b8fd74db9c4341be851d4637	-1.2667500	36.8067500	234.50	42.70	t	92	Waiyaki Way, Westlands	f	2026-09-26 08:13:26+00
387	bik_b8fd74db9c4341be851d4637	-1.2680000	36.8050000	244.40	53.00	t	92	Waiyaki Way, Westlands	f	2026-09-26 08:14:35+00
388	bik_b8fd74db9c4341be851d4637	-1.2707500	36.7992500	244.40	48.20	t	92	Waiyaki Way, Westlands	f	2026-09-26 08:15:43+00
389	bik_b8fd74db9c4341be851d4637	-1.2735000	36.7935000	244.40	36.80	t	92	Riverside Dr, Westlands	f	2026-09-26 08:16:52+00
390	bik_b8fd74db9c4341be851d4637	-1.2762500	36.7877500	244.40	29.30	t	92	Riverside Dr, Westlands	f	2026-09-26 08:18:01+00
391	bik_b8fd74db9c4341be851d4637	-1.2790000	36.7820000	230.20	42.70	t	92	Riverside Dr, Westlands	f	2026-09-26 08:19:10+00
392	bik_b8fd74db9c4341be851d4637	-1.2802500	36.7805000	230.20	31.90	t	92	Riverside Dr, Westlands	f	2026-09-26 08:20:18+00
393	bik_b8fd74db9c4341be851d4637	-1.2815000	36.7790000	230.20	29.50	t	92	Ring Rd Kilimani	f	2026-09-26 08:21:27+00
394	bik_b8fd74db9c4341be851d4637	-1.2827500	36.7775000	230.20	37.60	t	92	Ring Rd Kilimani	f	2026-09-26 08:22:36+00
395	bik_b8fd74db9c4341be851d4637	-1.2840000	36.7760000	123.70	29.30	t	91	Ring Rd Kilimani	f	2026-09-26 08:23:45+00
396	bik_b8fd74db9c4341be851d4637	-1.2850000	36.7775000	123.70	32.50	t	91	Ring Rd Kilimani	f	2026-09-26 08:24:53+00
397	bik_b8fd74db9c4341be851d4637	-1.2860000	36.7790000	123.70	43.60	t	91	Yaya Centre, Kilimani	f	2026-09-26 08:26:02+00
398	bik_b8fd74db9c4341be851d4637	-1.2870000	36.7805000	123.70	52.30	t	91	Yaya Centre, Kilimani	f	2026-09-26 08:27:11+00
399	bik_b8fd74db9c4341be851d4637	-1.2880000	36.7820000	123.70	37.60	t	91	Yaya Centre, Kilimani	f	2026-09-26 08:28:20+00
400	bik_b8fd74db9c4341be851d4637	-1.2890000	36.7835000	123.70	48.90	t	91	Yaya Centre, Kilimani	f	2026-09-26 08:29:28+00
401	bik_b8fd74db9c4341be851d4637	-1.2900000	36.7850000	123.70	52.90	t	91	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:30:37+00
402	bik_b8fd74db9c4341be851d4637	-1.2910000	36.7865000	123.70	45.90	t	91	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:31:46+00
403	bik_b8fd74db9c4341be851d4637	-1.2920000	36.7880000	128.70	52.30	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:32:55+00
404	bik_b8fd74db9c4341be851d4637	-1.2930000	36.7892500	128.70	50.60	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-26 08:34:03+00
405	bik_b8fd74db9c4341be851d4637	-1.2940000	36.7905000	128.70	40.10	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:35:12+00
406	bik_b8fd74db9c4341be851d4637	-1.2950000	36.7917500	128.70	30.40	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:36:21+00
407	bik_b8fd74db9c4341be851d4637	-1.2960000	36.7930000	119.40	45.90	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:37:30+00
408	bik_b8fd74db9c4341be851d4637	-1.2971250	36.7950000	119.40	34.50	t	90	Ngong Rd / Adams Arcade	f	2026-09-26 08:38:38+00
409	bik_b8fd74db9c4341be851d4637	-1.2982500	36.7970000	119.40	29.00	t	90	Elgon Rd, Upper Hill	f	2026-09-26 08:39:47+00
410	bik_b8fd74db9c4341be851d4637	-1.2993750	36.7990000	119.40	34.60	t	90	Elgon Rd, Upper Hill	f	2026-09-26 08:40:56+00
411	bik_b8fd74db9c4341be851d4637	-1.3005000	36.8010000	76.00	30.40	t	90	Elgon Rd, Upper Hill	f	2026-09-26 08:42:05+00
412	bik_b8fd74db9c4341be851d4637	-1.3000000	36.8030000	76.00	30.50	t	90	Elgon Rd, Upper Hill	f	2026-09-26 08:43:13+00
413	bik_b8fd74db9c4341be851d4637	-1.2995000	36.8050000	76.00	40.20	t	90	Hospital Rd, Upper Hill	f	2026-09-26 08:44:22+00
414	bik_b8fd74db9c4341be851d4637	-1.2990000	36.8070000	76.00	50.60	t	90	Hospital Rd, Upper Hill	f	2026-09-26 08:45:31+00
415	bik_b8fd74db9c4341be851d4637	-1.2985000	36.8090000	55.00	34.60	t	89	Hospital Rd, Upper Hill	f	2026-09-26 08:46:40+00
416	bik_b8fd74db9c4341be851d4637	-1.2976250	36.8102500	55.00	46.00	t	89	Hospital Rd, Upper Hill	f	2026-09-26 08:47:48+00
417	bik_b8fd74db9c4341be851d4637	-1.2967500	36.8115000	55.00	52.90	t	89	Ragati Rd, Upper Hill	f	2026-09-26 08:48:57+00
418	bik_b8fd74db9c4341be851d4637	-1.2958750	36.8127500	55.00	48.80	t	89	Ragati Rd, Upper Hill	f	2026-09-26 08:50:06+00
419	bik_b8fd74db9c4341be851d4637	-1.2950000	36.8140000	48.40	50.60	t	89	Ragati Rd, Upper Hill	f	2026-09-26 08:51:15+00
420	bik_b8fd74db9c4341be851d4637	-1.2940000	36.8151250	48.40	52.20	t	89	Ragati Rd, Upper Hill	f	2026-09-26 08:52:23+00
421	bik_b8fd74db9c4341be851d4637	-1.2930000	36.8162500	48.40	43.50	t	89	Uhuru Highway Roundabout	f	2026-09-26 08:53:32+00
422	bik_b8fd74db9c4341be851d4637	-1.2920000	36.8173750	48.40	32.50	t	89	Uhuru Highway Roundabout	f	2026-09-26 08:54:41+00
423	bik_b8fd74db9c4341be851d4637	-1.2910000	36.8185000	6.30	48.80	t	88	Uhuru Highway Roundabout	f	2026-09-26 08:55:50+00
424	bik_b8fd74db9c4341be851d4637	-1.2898750	36.8186250	6.30	37.50	t	88	Uhuru Highway Roundabout	f	2026-09-26 08:56:58+00
425	bik_b8fd74db9c4341be851d4637	-1.2887500	36.8187500	6.30	29.50	t	88	Haile Selassie Ave, CBD	f	2026-09-26 08:58:07+00
426	bik_b8fd74db9c4341be851d4637	-1.2876250	36.8188750	6.30	32.00	t	88	Haile Selassie Ave, CBD	f	2026-09-26 08:59:16+00
427	bik_b8fd74db9c4341be851d4637	-1.2865000	36.8190000	324.30	32.50	t	88	Haile Selassie Ave, CBD	f	2026-09-26 09:00:25+00
428	bik_b8fd74db9c4341be851d4637	-1.2857000	36.8184250	324.30	29.30	t	88	Haile Selassie Ave, CBD	f	2026-09-26 09:01:33+00
429	bik_b8fd74db9c4341be851d4637	-1.2849000	36.8178500	324.30	36.90	t	88	Kenyatta Ave, CBD	f	2026-09-26 09:02:42+00
430	bik_b8fd74db9c4341be851d4637	-1.2841000	36.8172750	324.30	48.30	t	88	Kenyatta Ave, CBD	f	2026-09-26 09:03:51+00
431	bik_b8fd74db9c4341be851d4637	-1.2833000	36.8167000	324.30	0.00	f	88	Kenyatta Ave, CBD	f	2026-09-26 09:05:00+00
432	bik_b8fd74db9c4341be851d4637	-1.3410000	36.8790000	300.30	44.00	t	92	Cabanas, Mombasa Rd	f	2026-09-26 14:20:00+00
433	bik_b8fd74db9c4341be851d4637	-1.3392500	36.8760000	300.30	54.10	t	92	Cabanas, Mombasa Rd	f	2026-09-26 14:20:47+00
434	bik_b8fd74db9c4341be851d4637	-1.3375000	36.8730000	300.30	54.90	t	92	General Motors, Enterprise Rd	f	2026-09-26 14:21:35+00
435	bik_b8fd74db9c4341be851d4637	-1.3357500	36.8700000	300.30	45.70	t	92	General Motors, Enterprise Rd	f	2026-09-26 14:22:23+00
436	bik_b8fd74db9c4341be851d4637	-1.3340000	36.8670000	315.00	56.00	t	92	General Motors, Enterprise Rd	f	2026-09-26 14:23:10+00
437	bik_b8fd74db9c4341be851d4637	-1.3317500	36.8647500	315.00	51.20	t	92	General Motors, Enterprise Rd	f	2026-09-26 14:23:58+00
438	bik_b8fd74db9c4341be851d4637	-1.3295000	36.8625000	315.00	39.80	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:24:46+00
439	bik_b8fd74db9c4341be851d4637	-1.3272500	36.8602500	315.00	32.30	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:25:34+00
440	bik_b8fd74db9c4341be851d4637	-1.3250000	36.8580000	315.00	45.70	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:26:21+00
441	bik_b8fd74db9c4341be851d4637	-1.3222500	36.8552500	315.00	34.90	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-26 14:27:09+00
442	bik_b8fd74db9c4341be851d4637	-1.3195000	36.8525000	315.00	32.50	t	92	Bellevue, Mombasa Rd	f	2026-09-26 14:27:57+00
443	bik_b8fd74db9c4341be851d4637	-1.3167500	36.8497500	315.00	40.60	t	92	Bellevue, Mombasa Rd	f	2026-09-26 14:28:45+00
444	bik_b8fd74db9c4341be851d4637	-1.3140000	36.8470000	318.80	32.30	t	91	Bellevue, Mombasa Rd	f	2026-09-26 14:29:32+00
445	bik_b8fd74db9c4341be851d4637	-1.3120000	36.8452500	318.80	35.50	t	91	Bellevue, Mombasa Rd	f	2026-09-26 14:30:20+00
446	bik_b8fd74db9c4341be851d4637	-1.3100000	36.8435000	318.80	46.60	t	91	Nyayo National Stadium	f	2026-09-26 14:31:08+00
447	bik_b8fd74db9c4341be851d4637	-1.3080000	36.8417500	318.80	55.30	t	91	Nyayo National Stadium	f	2026-09-26 14:31:55+00
448	bik_b8fd74db9c4341be851d4637	-1.3060000	36.8400000	319.40	40.60	t	91	Nyayo National Stadium	f	2026-09-26 14:32:43+00
449	bik_b8fd74db9c4341be851d4637	-1.3042500	36.8385000	319.40	51.90	t	91	Nyayo National Stadium	f	2026-09-26 14:33:31+00
450	bik_b8fd74db9c4341be851d4637	-1.3025000	36.8370000	319.40	55.90	t	91	Bunyala Rd Roundabout	f	2026-09-26 14:34:19+00
451	bik_b8fd74db9c4341be851d4637	-1.3007500	36.8355000	319.40	48.90	t	91	Bunyala Rd Roundabout	f	2026-09-26 14:35:06+00
452	bik_b8fd74db9c4341be851d4637	-1.2990000	36.8340000	323.10	55.30	t	90	Bunyala Rd Roundabout	f	2026-09-26 14:35:54+00
453	bik_b8fd74db9c4341be851d4637	-1.2970000	36.8325000	323.10	53.60	t	90	Bunyala Rd Roundabout	f	2026-09-26 14:36:42+00
454	bik_b8fd74db9c4341be851d4637	-1.2950000	36.8310000	323.10	43.10	t	90	Railway Station, CBD	f	2026-09-26 14:37:30+00
455	bik_b8fd74db9c4341be851d4637	-1.2930000	36.8295000	323.10	33.40	t	90	Railway Station, CBD	f	2026-09-26 14:38:17+00
456	bik_b8fd74db9c4341be851d4637	-1.2910000	36.8280000	306.90	48.90	t	90	Railway Station, CBD	f	2026-09-26 14:39:05+00
457	bik_b8fd74db9c4341be851d4637	-1.2898750	36.8265000	306.90	37.50	t	90	Railway Station, CBD	f	2026-09-26 14:39:53+00
458	bik_b8fd74db9c4341be851d4637	-1.2887500	36.8250000	306.90	32.00	t	90	Moi Avenue, CBD	f	2026-09-26 14:40:40+00
459	bik_b8fd74db9c4341be851d4637	-1.2876250	36.8235000	306.90	37.60	t	90	Moi Avenue, CBD	f	2026-09-26 14:41:28+00
460	bik_b8fd74db9c4341be851d4637	-1.2865000	36.8220000	311.20	33.40	t	90	Moi Avenue, CBD	f	2026-09-26 14:42:16+00
461	bik_b8fd74db9c4341be851d4637	-1.2856250	36.8210000	311.20	33.50	t	90	Moi Avenue, CBD	f	2026-09-26 14:43:04+00
462	bik_b8fd74db9c4341be851d4637	-1.2847500	36.8200000	311.20	43.20	t	90	University Way, CBD	f	2026-09-26 14:43:51+00
463	bik_b8fd74db9c4341be851d4637	-1.2838750	36.8190000	311.20	53.60	t	90	University Way, CBD	f	2026-09-26 14:44:39+00
464	bik_b8fd74db9c4341be851d4637	-1.2830000	36.8180000	336.80	37.60	t	89	University Way, CBD	f	2026-09-26 14:45:27+00
465	bik_b8fd74db9c4341be851d4637	-1.2812500	36.8172500	336.80	49.00	t	89	University Way, CBD	f	2026-09-26 14:46:15+00
466	bik_b8fd74db9c4341be851d4637	-1.2795000	36.8165000	336.80	55.90	t	89	Museum Hill Interchange	f	2026-09-26 14:47:02+00
467	bik_b8fd74db9c4341be851d4637	-1.2777500	36.8157500	336.80	51.80	t	89	Museum Hill Interchange	f	2026-09-26 14:47:50+00
468	bik_b8fd74db9c4341be851d4637	-1.2760000	36.8150000	311.20	53.60	t	89	Museum Hill Interchange	f	2026-09-26 14:48:38+00
469	bik_b8fd74db9c4341be851d4637	-1.2742500	36.8130000	311.20	55.20	t	89	Museum Hill Interchange	f	2026-09-26 14:49:25+00
470	bik_b8fd74db9c4341be851d4637	-1.2725000	36.8110000	311.20	46.50	t	89	Westlands Roundabout	f	2026-09-26 14:50:13+00
471	bik_b8fd74db9c4341be851d4637	-1.2707500	36.8090000	311.20	35.50	t	89	Westlands Roundabout	f	2026-09-26 14:51:01+00
472	bik_b8fd74db9c4341be851d4637	-1.2690000	36.8070000	39.80	51.80	t	88	Westlands Roundabout	f	2026-09-26 14:51:49+00
473	bik_b8fd74db9c4341be851d4637	-1.2675000	36.8082500	39.80	40.50	t	88	Westlands Roundabout	f	2026-09-26 14:52:36+00
474	bik_b8fd74db9c4341be851d4637	-1.2660000	36.8095000	39.80	32.50	t	88	Sarit Centre, Westlands	f	2026-09-26 14:53:24+00
475	bik_b8fd74db9c4341be851d4637	-1.2645000	36.8107500	39.80	35.00	t	88	Sarit Centre, Westlands	f	2026-09-26 14:54:12+00
476	bik_b8fd74db9c4341be851d4637	-1.2630000	36.8120000	39.80	0.00	f	88	Sarit Centre, Westlands	f	2026-09-26 14:55:00+00
477	bik_b8fd74db9c4341be851d4637	-1.2630000	36.8120000	234.50	47.00	t	92	Sarit Centre, Westlands	f	2026-09-27 07:50:00+00
478	bik_b8fd74db9c4341be851d4637	-1.2642500	36.8102500	234.50	57.10	t	92	Sarit Centre, Westlands	f	2026-09-27 07:50:56+00
479	bik_b8fd74db9c4341be851d4637	-1.2655000	36.8085000	234.50	57.90	t	92	Waiyaki Way, Westlands	f	2026-09-27 07:51:52+00
480	bik_b8fd74db9c4341be851d4637	-1.2667500	36.8067500	234.50	48.70	t	92	Waiyaki Way, Westlands	f	2026-09-27 07:52:48+00
481	bik_b8fd74db9c4341be851d4637	-1.2680000	36.8050000	244.40	59.00	t	92	Waiyaki Way, Westlands	f	2026-09-27 07:53:45+00
482	bik_b8fd74db9c4341be851d4637	-1.2707500	36.7992500	244.40	54.20	t	92	Waiyaki Way, Westlands	f	2026-09-27 07:54:41+00
483	bik_b8fd74db9c4341be851d4637	-1.2735000	36.7935000	244.40	42.80	t	92	Riverside Dr, Westlands	f	2026-09-27 07:55:37+00
484	bik_b8fd74db9c4341be851d4637	-1.2762500	36.7877500	244.40	35.30	t	92	Riverside Dr, Westlands	f	2026-09-27 07:56:33+00
485	bik_b8fd74db9c4341be851d4637	-1.2790000	36.7820000	230.20	48.70	t	92	Riverside Dr, Westlands	f	2026-09-27 07:57:30+00
486	bik_b8fd74db9c4341be851d4637	-1.2802500	36.7805000	230.20	37.90	t	92	Riverside Dr, Westlands	f	2026-09-27 07:58:26+00
487	bik_b8fd74db9c4341be851d4637	-1.2815000	36.7790000	230.20	35.50	t	92	Ring Rd Kilimani	f	2026-09-27 07:59:22+00
488	bik_b8fd74db9c4341be851d4637	-1.2827500	36.7775000	230.20	43.60	t	92	Ring Rd Kilimani	f	2026-09-27 08:00:18+00
489	bik_b8fd74db9c4341be851d4637	-1.2840000	36.7760000	123.70	35.30	t	91	Ring Rd Kilimani	f	2026-09-27 08:01:15+00
490	bik_b8fd74db9c4341be851d4637	-1.2850000	36.7775000	123.70	38.50	t	91	Ring Rd Kilimani	f	2026-09-27 08:02:11+00
491	bik_b8fd74db9c4341be851d4637	-1.2860000	36.7790000	123.70	49.60	t	91	Yaya Centre, Kilimani	f	2026-09-27 08:03:07+00
492	bik_b8fd74db9c4341be851d4637	-1.2870000	36.7805000	123.70	58.30	t	91	Yaya Centre, Kilimani	f	2026-09-27 08:04:03+00
493	bik_b8fd74db9c4341be851d4637	-1.2880000	36.7820000	123.70	43.60	t	91	Yaya Centre, Kilimani	f	2026-09-27 08:05:00+00
494	bik_b8fd74db9c4341be851d4637	-1.2890000	36.7835000	123.70	54.90	t	91	Yaya Centre, Kilimani	f	2026-09-27 08:05:56+00
495	bik_b8fd74db9c4341be851d4637	-1.2900000	36.7850000	123.70	58.90	t	91	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:06:52+00
496	bik_b8fd74db9c4341be851d4637	-1.2910000	36.7865000	123.70	51.90	t	91	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:07:48+00
497	bik_b8fd74db9c4341be851d4637	-1.2920000	36.7880000	128.70	58.30	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:08:45+00
498	bik_b8fd74db9c4341be851d4637	-1.2930000	36.7892500	128.70	56.60	t	90	Argwings Kodhek Rd, Kilimani	f	2026-09-27 08:09:41+00
499	bik_b8fd74db9c4341be851d4637	-1.2940000	36.7905000	128.70	46.10	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:10:37+00
500	bik_b8fd74db9c4341be851d4637	-1.2950000	36.7917500	128.70	36.40	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:11:33+00
501	bik_b8fd74db9c4341be851d4637	-1.2960000	36.7930000	119.40	51.90	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:12:30+00
502	bik_b8fd74db9c4341be851d4637	-1.2971250	36.7950000	119.40	40.50	t	90	Ngong Rd / Adams Arcade	f	2026-09-27 08:13:26+00
503	bik_b8fd74db9c4341be851d4637	-1.2982500	36.7970000	119.40	35.00	t	90	Elgon Rd, Upper Hill	f	2026-09-27 08:14:22+00
504	bik_b8fd74db9c4341be851d4637	-1.2993750	36.7990000	119.40	40.60	t	90	Elgon Rd, Upper Hill	f	2026-09-27 08:15:18+00
505	bik_b8fd74db9c4341be851d4637	-1.3005000	36.8010000	76.00	36.40	t	90	Elgon Rd, Upper Hill	f	2026-09-27 08:16:15+00
506	bik_b8fd74db9c4341be851d4637	-1.3000000	36.8030000	76.00	36.50	t	90	Elgon Rd, Upper Hill	f	2026-09-27 08:17:11+00
507	bik_b8fd74db9c4341be851d4637	-1.2995000	36.8050000	76.00	46.20	t	90	Hospital Rd, Upper Hill	f	2026-09-27 08:18:07+00
508	bik_b8fd74db9c4341be851d4637	-1.2990000	36.8070000	76.00	56.60	t	90	Hospital Rd, Upper Hill	f	2026-09-27 08:19:03+00
509	bik_b8fd74db9c4341be851d4637	-1.2985000	36.8090000	55.00	40.60	t	89	Hospital Rd, Upper Hill	f	2026-09-27 08:20:00+00
510	bik_b8fd74db9c4341be851d4637	-1.2976250	36.8102500	55.00	52.00	t	89	Hospital Rd, Upper Hill	f	2026-09-27 08:20:56+00
511	bik_b8fd74db9c4341be851d4637	-1.2967500	36.8115000	55.00	58.90	t	89	Ragati Rd, Upper Hill	f	2026-09-27 08:21:52+00
512	bik_b8fd74db9c4341be851d4637	-1.2958750	36.8127500	55.00	54.80	t	89	Ragati Rd, Upper Hill	f	2026-09-27 08:22:48+00
513	bik_b8fd74db9c4341be851d4637	-1.2950000	36.8140000	48.40	56.60	t	89	Ragati Rd, Upper Hill	f	2026-09-27 08:23:45+00
514	bik_b8fd74db9c4341be851d4637	-1.2940000	36.8151250	48.40	58.20	t	89	Ragati Rd, Upper Hill	f	2026-09-27 08:24:41+00
515	bik_b8fd74db9c4341be851d4637	-1.2930000	36.8162500	48.40	49.50	t	89	Uhuru Highway Roundabout	f	2026-09-27 08:25:37+00
516	bik_b8fd74db9c4341be851d4637	-1.2920000	36.8173750	48.40	38.50	t	89	Uhuru Highway Roundabout	f	2026-09-27 08:26:33+00
517	bik_b8fd74db9c4341be851d4637	-1.2910000	36.8185000	6.30	54.80	t	88	Uhuru Highway Roundabout	f	2026-09-27 08:27:30+00
518	bik_b8fd74db9c4341be851d4637	-1.2898750	36.8186250	6.30	43.50	t	88	Uhuru Highway Roundabout	f	2026-09-27 08:28:26+00
519	bik_b8fd74db9c4341be851d4637	-1.2887500	36.8187500	6.30	35.50	t	88	Haile Selassie Ave, CBD	f	2026-09-27 08:29:22+00
520	bik_b8fd74db9c4341be851d4637	-1.2876250	36.8188750	6.30	38.00	t	88	Haile Selassie Ave, CBD	f	2026-09-27 08:30:18+00
521	bik_b8fd74db9c4341be851d4637	-1.2865000	36.8190000	324.30	38.50	t	88	Haile Selassie Ave, CBD	f	2026-09-27 08:31:15+00
522	bik_b8fd74db9c4341be851d4637	-1.2857000	36.8184250	324.30	35.30	t	88	Haile Selassie Ave, CBD	f	2026-09-27 08:32:11+00
523	bik_b8fd74db9c4341be851d4637	-1.2849000	36.8178500	324.30	42.90	t	88	Kenyatta Ave, CBD	f	2026-09-27 08:33:07+00
524	bik_b8fd74db9c4341be851d4637	-1.2841000	36.8172750	324.30	54.30	t	88	Kenyatta Ave, CBD	f	2026-09-27 08:34:03+00
525	bik_b8fd74db9c4341be851d4637	-1.2833000	36.8167000	324.30	0.00	f	88	Kenyatta Ave, CBD	f	2026-09-27 08:35:00+00
526	bik_b8fd74db9c4341be851d4637	-1.3410000	36.8790000	300.30	50.00	t	92	Cabanas, Mombasa Rd	f	2026-09-27 11:15:00+00
527	bik_b8fd74db9c4341be851d4637	-1.3392500	36.8760000	300.30	60.10	t	92	Cabanas, Mombasa Rd	f	2026-09-27 11:16:15+00
528	bik_b8fd74db9c4341be851d4637	-1.3375000	36.8730000	300.30	60.90	t	92	General Motors, Enterprise Rd	f	2026-09-27 11:17:30+00
529	bik_b8fd74db9c4341be851d4637	-1.3357500	36.8700000	300.30	51.70	t	92	General Motors, Enterprise Rd	f	2026-09-27 11:18:45+00
530	bik_b8fd74db9c4341be851d4637	-1.3340000	36.8670000	315.00	62.00	t	92	General Motors, Enterprise Rd	f	2026-09-27 11:20:00+00
531	bik_b8fd74db9c4341be851d4637	-1.3317500	36.8647500	315.00	57.20	t	92	General Motors, Enterprise Rd	f	2026-09-27 11:21:15+00
532	bik_b8fd74db9c4341be851d4637	-1.3295000	36.8625000	315.00	45.80	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-27 11:22:30+00
533	bik_b8fd74db9c4341be851d4637	-1.3272500	36.8602500	315.00	38.30	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-27 11:23:45+00
534	bik_b8fd74db9c4341be851d4637	-1.3250000	36.8580000	315.00	51.70	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-27 11:25:00+00
535	bik_b8fd74db9c4341be851d4637	-1.3222500	36.8552500	315.00	40.90	t	92	Panari Sky Centre, Mombasa Rd	f	2026-09-27 11:26:15+00
536	bik_b8fd74db9c4341be851d4637	-1.3195000	36.8525000	315.00	38.50	t	92	Bellevue, Mombasa Rd	f	2026-09-27 11:27:30+00
537	bik_b8fd74db9c4341be851d4637	-1.3167500	36.8497500	315.00	46.60	t	92	Bellevue, Mombasa Rd	f	2026-09-27 11:28:45+00
538	bik_b8fd74db9c4341be851d4637	-1.3140000	36.8470000	318.80	38.30	t	91	Bellevue, Mombasa Rd	f	2026-09-27 11:30:00+00
539	bik_b8fd74db9c4341be851d4637	-1.3120000	36.8452500	318.80	41.50	t	91	Bellevue, Mombasa Rd	f	2026-09-27 11:31:15+00
540	bik_b8fd74db9c4341be851d4637	-1.3100000	36.8435000	318.80	52.60	t	91	Nyayo National Stadium	f	2026-09-27 11:32:30+00
541	bik_b8fd74db9c4341be851d4637	-1.3080000	36.8417500	318.80	61.30	t	91	Nyayo National Stadium	f	2026-09-27 11:33:45+00
542	bik_b8fd74db9c4341be851d4637	-1.3060000	36.8400000	319.40	46.60	t	91	Nyayo National Stadium	f	2026-09-27 11:35:00+00
543	bik_b8fd74db9c4341be851d4637	-1.3042500	36.8385000	319.40	57.90	t	91	Nyayo National Stadium	f	2026-09-27 11:36:15+00
544	bik_b8fd74db9c4341be851d4637	-1.3025000	36.8370000	319.40	61.90	t	91	Bunyala Rd Roundabout	f	2026-09-27 11:37:30+00
545	bik_b8fd74db9c4341be851d4637	-1.3007500	36.8355000	319.40	54.90	t	91	Bunyala Rd Roundabout	f	2026-09-27 11:38:45+00
546	bik_b8fd74db9c4341be851d4637	-1.2990000	36.8340000	323.10	61.30	t	90	Bunyala Rd Roundabout	f	2026-09-27 11:40:00+00
547	bik_b8fd74db9c4341be851d4637	-1.2970000	36.8325000	323.10	59.60	t	90	Bunyala Rd Roundabout	f	2026-09-27 11:41:15+00
548	bik_b8fd74db9c4341be851d4637	-1.2950000	36.8310000	323.10	49.10	t	90	Railway Station, CBD	f	2026-09-27 11:42:30+00
549	bik_b8fd74db9c4341be851d4637	-1.2930000	36.8295000	323.10	39.40	t	90	Railway Station, CBD	f	2026-09-27 11:43:45+00
550	bik_b8fd74db9c4341be851d4637	-1.2910000	36.8280000	306.90	54.90	t	90	Railway Station, CBD	f	2026-09-27 11:45:00+00
551	bik_b8fd74db9c4341be851d4637	-1.2898750	36.8265000	306.90	43.50	t	90	Railway Station, CBD	f	2026-09-27 11:46:15+00
552	bik_b8fd74db9c4341be851d4637	-1.2887500	36.8250000	306.90	38.00	t	90	Moi Avenue, CBD	f	2026-09-27 11:47:30+00
553	bik_b8fd74db9c4341be851d4637	-1.2876250	36.8235000	306.90	43.60	t	90	Moi Avenue, CBD	f	2026-09-27 11:48:45+00
554	bik_b8fd74db9c4341be851d4637	-1.2865000	36.8220000	311.20	39.40	t	90	Moi Avenue, CBD	f	2026-09-27 11:50:00+00
555	bik_b8fd74db9c4341be851d4637	-1.2856250	36.8210000	311.20	39.50	t	90	Moi Avenue, CBD	f	2026-09-27 11:51:15+00
556	bik_b8fd74db9c4341be851d4637	-1.2847500	36.8200000	311.20	49.20	t	90	University Way, CBD	f	2026-09-27 11:52:30+00
557	bik_b8fd74db9c4341be851d4637	-1.2838750	36.8190000	311.20	59.60	t	90	University Way, CBD	f	2026-09-27 11:53:45+00
558	bik_b8fd74db9c4341be851d4637	-1.2830000	36.8180000	336.80	43.60	t	89	University Way, CBD	f	2026-09-27 11:55:00+00
559	bik_b8fd74db9c4341be851d4637	-1.2812500	36.8172500	336.80	55.00	t	89	University Way, CBD	f	2026-09-27 11:56:15+00
560	bik_b8fd74db9c4341be851d4637	-1.2795000	36.8165000	336.80	61.90	t	89	Museum Hill Interchange	f	2026-09-27 11:57:30+00
561	bik_b8fd74db9c4341be851d4637	-1.2777500	36.8157500	336.80	57.80	t	89	Museum Hill Interchange	f	2026-09-27 11:58:45+00
562	bik_b8fd74db9c4341be851d4637	-1.2760000	36.8150000	311.20	59.60	t	89	Museum Hill Interchange	f	2026-09-27 12:00:00+00
563	bik_b8fd74db9c4341be851d4637	-1.2742500	36.8130000	311.20	61.20	t	89	Museum Hill Interchange	f	2026-09-27 12:01:15+00
564	bik_b8fd74db9c4341be851d4637	-1.2725000	36.8110000	311.20	52.50	t	89	Westlands Roundabout	f	2026-09-27 12:02:30+00
565	bik_b8fd74db9c4341be851d4637	-1.2707500	36.8090000	311.20	41.50	t	89	Westlands Roundabout	f	2026-09-27 12:03:45+00
566	bik_b8fd74db9c4341be851d4637	-1.2690000	36.8070000	39.80	57.80	t	88	Westlands Roundabout	f	2026-09-27 12:05:00+00
567	bik_b8fd74db9c4341be851d4637	-1.2675000	36.8082500	39.80	46.50	t	88	Westlands Roundabout	f	2026-09-27 12:06:15+00
568	bik_b8fd74db9c4341be851d4637	-1.2660000	36.8095000	39.80	38.50	t	88	Sarit Centre, Westlands	f	2026-09-27 12:07:30+00
569	bik_b8fd74db9c4341be851d4637	-1.2645000	36.8107500	39.80	41.00	t	88	Sarit Centre, Westlands	f	2026-09-27 12:08:45+00
570	bik_b8fd74db9c4341be851d4637	-1.2630000	36.8120000	39.80	0.00	f	88	Sarit Centre, Westlands	f	2026-09-27 12:10:00+00
571	bik_55ccd33c4a694726a31b0b55	-1.2855000	36.8220000	0.00	0.00	f	95		f	2026-09-27 12:09:04.653937+00
572	bik_55ccd33c4a694726a31b0b55	-1.2855000	36.8220000	90.00	22.00	t	98		f	2026-09-27 16:23:43.635921+00
\.


--
-- Data for Name: bike_events; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.bike_events (id, bike_id, event_type, latitude, longitude, recorded_at) FROM stdin;
1	bik_55ccd33c4a694726a31b0b55	IGNITION_ON	-1.2855000	36.8220000	2026-09-27 11:30:00+00
2	bik_55ccd33c4a694726a31b0b55	GEOFENCE_APPROACHING	-1.3120000	36.8500000	2026-09-27 11:45:00+00
3	bik_55ccd33c4a694726a31b0b55	IGNITION_OFF	-1.3410000	36.8790000	2026-09-27 12:10:00+00
4	bik_55ccd33c4a694726a31b0b55	BIKE_TOUCHED	-1.3410000	36.8790000	2026-09-27 12:28:15+00
5	bik_b8fd74db9c4341be851d4637	IGNITION_ON	-1.2921000	36.8219000	2026-09-27 11:30:00+00
6	bik_b8fd74db9c4341be851d4637	GEOFENCE_APPROACHING	-1.2675000	36.8082500	2026-09-27 11:58:30+00
7	bik_b8fd74db9c4341be851d4637	IGNITION_OFF	-1.2630000	36.8120000	2026-09-27 12:10:00+00
8	bik_b8fd74db9c4341be851d4637	BIKE_TOUCHED	-1.2630000	36.8120000	2026-09-27 12:35:40+00
9	bik_55ccd33c4a694726a31b0b55	IGNITION_ON	-1.2855000	36.8220000	2026-09-26 09:15:00+00
10	bik_55ccd33c4a694726a31b0b55	BIKE_TOUCHED	-1.3000000	36.8300000	2026-09-26 13:42:00+00
11	bik_55ccd33c4a694726a31b0b55	IGNITION_OFF	-1.2855000	36.8220000	2026-09-26 17:30:00+00
12	bik_55ccd33c4a694726a31b0b55	BIKE_TOUCHED	-1.3410000	36.8790000	2026-09-27 11:20:00+00
13	bik_55ccd33c4a694726a31b0b55	ACCIDENT_DETECTED	-1.3340000	36.8670000	2026-09-27 12:05:30+00
14	bik_b8fd74db9c4341be851d4637	ACCIDENT_DETECTED	-1.2675000	36.8082500	2026-09-27 11:36:40+00
15	bik_55ccd33c4a694726a31b0b55	ACCIDENT_DETECTED	-1.2855000	36.8220000	2026-09-27 11:53:20+00
21	bik_b8fd74db9c4341be851d4637	IMMOBILIZED	-1.2921000	36.8219000	2026-09-28 18:37:41.510349+00
22	bik_b8fd74db9c4341be851d4637	REACTIVATED	-1.2921000	36.8219000	2026-09-28 18:43:05.446131+00
\.


--
-- Data for Name: bikes; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.bikes (id, registration_number, label, make, model, year, color, imei, status, ignition, speed_kmh, heading_deg, latitude, longitude, current_address, sub_county, county, battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id, created_at, updated_at, is_immobilized, user_id) FROM stdin;
bik_55ccd33c4a694726a31b0b55	KMEX 482B	Delivery Unit Alpha	Bajaj	Boxer 150	2024	Midnight Black	860123456789001	RUNNING	t	22.00	90.00	-1.2855000	36.8220000	Kenyatta Avenue, Nairobi CBD	Starehe	Nairobi	98	28	2026-09-27 16:23:43.635921+00	\N	\N	2026-09-26 20:53:55.824315+00	2026-09-28 18:28:58.010747+00	t	usr_01J8Z9X2A4B6C8D0E2F4G6H8
bik_b8fd74db9c4341be851d4637	KMDR 914Z	Dispatch Unit Bravo	TVS	HLX 125	2023	Crimson Red	860123456789002	RUNNING	t	42.80	90.00	-1.2921000	36.8219000	Haile Selassie Avenue, Nairobi	Starehe	Nairobi	98	28	2026-09-27 09:07:23.16976+00	\N	\N	2026-09-26 20:53:55.853731+00	2026-09-28 18:43:05.446131+00	f	usr_01J8Z9X2A4B6C8D0E2F4G6H8
\.


--
-- Data for Name: driver_contacts; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.driver_contacts (id, name, phone, id_number, license_number, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: geofences; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.geofences (id, created_by, name, description, boundary, center_lat, center_lng, radius_meters, alert_on_exit, alert_on_entry, active, created_at, updated_at, status, suspended_at, deleted_at) FROM stdin;
\.


--
-- Data for Name: password_reset_tokens; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at) FROM stdin;
\.


--
-- Data for Name: sessions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.sessions (id, user_id, refresh_token_hash, expires_at, revoked_at, user_agent, ip_address, created_at) FROM stdin;
34c5ece1-fafa-48bb-afab-fde907a89775	usr_01J8Z9X2A4B6C8D0E2F4G6H8	e6e092360662f4483940c8e2c923a0579d7c9e55f2b90a97a9dbe3679edd88e3	2026-10-03 19:18:05.709247+00	\N	curl/8.20.0	::1	2026-09-26 19:18:05.709247+00
675c93ab-84c6-4ec9-9a6a-c1a151911e9f	usr_01J8Z9X2A4B6C8D0E2F4G6H8	d8d2d68a94d017cb133ebe296d4435b878a96e7b6be1fb9d32ff538d2dcbb59a	2026-10-03 19:18:13.769874+00	\N	curl/8.20.0	::1	2026-09-26 19:18:13.769874+00
3b8fab76-90af-4bef-b753-698d39de2f5d	usr_01J8Z9X2A4B6C8D0E2F4G6H8	32bfef37e46ab6b969361e60a0af24b1855db6693a96d64e474663159e3d33fb	2026-10-03 19:18:32.207167+00	2026-09-26 19:18:32.216658+00	curl/8.20.0	::1	2026-09-26 19:18:32.207167+00
f225d0f5-b511-4016-a4e5-ba3cd852e849	usr_01J8Z9X2A4B6C8D0E2F4G6H8	8cbff63d87621bc68d375db7fca419946b7a2e2f716d97c689eecba920408b70	2026-10-03 19:18:32.216629+00	\N	curl/8.20.0	::1	2026-09-26 19:18:32.216629+00
a1408d35-28cf-4ff0-8c6c-a7d4c75a6d80	usr_01J8Z9X2A4B6C8D0E2F4G6H8	af489904fe75880e563342f26c48ea4d92af8cc8ec407cdcffd45994e98c8147	2026-10-03 19:23:27.207202+00	\N	Python-urllib/3.14	::1	2026-09-26 19:23:27.207202+00
a794b6f2-a147-4501-b6f4-c9b7e573b3c9	usr_01J8Z9X2A4B6C8D0E2F4G6H8	d7652606b227666a46caa98521a400f83c8a503db61d16a9ed127aca8c3d1d21	2026-10-03 20:53:20.903896+00	\N	curl/8.20.0	::1	2026-09-26 20:53:20.903896+00
fc23b5d6-7d9e-4525-b4fe-f4072ed29304	usr_01J8Z9X2A4B6C8D0E2F4G6H8	5ed095f5ba4fcadb6e9ce897def3f08eda089947adf89d7f1a917c2d0153014f	2026-10-03 20:53:25.471003+00	\N	curl/8.20.0	::1	2026-09-26 20:53:25.471003+00
f8907d86-e108-47b0-9ce7-2637c9c5448b	usr_01J8Z9X2A4B6C8D0E2F4G6H8	72a590306e27c2a632b57866bd3ec285875520beb5b324c42a0d8c2b77618067	2026-10-03 20:53:46.105048+00	\N	curl/8.20.0	::1	2026-09-26 20:53:46.105048+00
9a7decc0-6777-4fe9-a201-758cc75a26fa	usr_01J8Z9X2A4B6C8D0E2F4G6H8	436543c52e28c0fb66ad9d56ec6602452323db6a969724930bae9929db4e6aac	2026-10-03 20:53:55.814764+00	\N	curl/8.20.0	::1	2026-09-26 20:53:55.814764+00
e3bb339e-0a93-4a6e-b526-a9027ece2e11	usr_01J8Z9X2A4B6C8D0E2F4G6H8	4bf97f047176236ef6b8878bfbc74641e3d735f5ff41a1b1c6771dd9e98d626c	2026-10-03 20:56:13.858607+00	\N	curl/8.20.0	::1	2026-09-26 20:56:13.858607+00
bf853b09-e4df-4d7c-8d93-dba9d84d0cd5	usr_01J8Z9X2A4B6C8D0E2F4G6H8	6b8fe08723e144b07d78672933b2b13fc2dc3c19723628f9ec67acdd0c5df9d2	2026-10-03 19:24:42.131982+00	2026-09-26 21:08:33.622095+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-26 19:24:42.131982+00
d17a3c7c-976a-4e12-8715-d6b73cc5a0d1	usr_01J8Z9X2A4B6C8D0E2F4G6H8	d72ca73e5905cea428b4c0c2c07cb1d0ccb8b052810092ed7018b173958d178f	2026-10-03 21:12:31.605437+00	\N	curl/8.20.0	::1	2026-09-26 21:12:31.605437+00
b0b969df-9dfc-4d52-9cc9-2f287f07f900	usr_01J8Z9X2A4B6C8D0E2F4G6H8	5260d160fdedf170b42f48190376b15e6bc6a8c9b0e222b67f42ca1d13b2df0f	2026-10-03 21:43:20.653074+00	\N	curl/8.20.0	::1	2026-09-26 21:43:20.653074+00
1999e75f-7b96-4d03-b7a7-f8fd0cb762a8	usr_01J8Z9X2A4B6C8D0E2F4G6H8	2c23311f4d3ca94c5637a4d461441167bda7cee4290fec435575e88b0c0838c2	2026-10-03 21:08:33.622047+00	2026-09-26 21:44:50.969686+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-26 21:08:33.622047+00
4d3e53d7-fa01-4a46-83d7-33ebdd913671	usr_01J8Z9X2A4B6C8D0E2F4G6H8	018b7b034e4a5f200f121d15a5de3dd621f6bf224ca1fbe7985c2a786f10fc97	2026-10-03 21:47:50.115247+00	\N	curl/8.20.0	::1	2026-09-26 21:47:50.115247+00
764b0cdf-2430-4bf0-8dfd-183e86b8b09a	usr_01J8Z9X2A4B6C8D0E2F4G6H8	7b257c6b40a607465c3e5d885a2e2b3cf0d7b87ddfd2a183d85668ceaf931f32	2026-10-03 21:50:05.151782+00	\N	curl/8.20.0	::1	2026-09-26 21:50:05.151782+00
2a8450af-0ca0-48d9-9ba2-2f0a49e65b04	usr_01J8Z9X2A4B6C8D0E2F4G6H8	bf5b27fb72498ea970e45009e60511d9421f8b6eb564bc9601fdc9676037a487	2026-10-03 21:50:48.048846+00	\N	curl/8.20.0	::1	2026-09-26 21:50:48.048846+00
5bb442c7-55ab-453c-932d-b5e8b6e44e1d	usr_01J8Z9X2A4B6C8D0E2F4G6H8	5363d6d3097cc4cb8d7c3d9b337fee0a80545f4934ea7921ebb4efec64f0054e	2026-10-03 21:44:50.969648+00	2026-09-26 22:02:13.00576+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-26 21:44:50.969648+00
47f02137-7502-4592-aef1-289488411771	usr_01J8Z9X2A4B6C8D0E2F4G6H8	1c9d68c4e200aa1c15474bd358c4a0d2c3612c7eab2f88002b2dc8155f6e42c6	2026-10-03 22:02:13.005687+00	2026-09-27 05:51:47.883246+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-26 22:02:13.005687+00
57112cbf-eac9-4403-9b32-04e21ed0fe91	usr_01J8Z9X2A4B6C8D0E2F4G6H8	21ad2a27316472360193505a24e2542adea1e0c1fa7c4a1d8311c7462afcc23e	2026-10-04 05:51:47.883034+00	2026-09-27 06:48:20.478211+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 05:51:47.883034+00
29a0ef8d-44ba-4152-bea0-dab602000fde	usr_01J8Z9X2A4B6C8D0E2F4G6H8	3ce54e26db2b749a63e17ac99c9ec88586820077f5ce0955c72d706938e53ed6	2026-10-04 06:48:20.478156+00	2026-09-27 07:08:59.447158+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 06:48:20.478156+00
28631b91-8fd0-4724-a32b-99ebde88b97b	usr_01J8Z9X2A4B6C8D0E2F4G6H8	88451c739ca1416a91789f4176b822ee284eab145c66b1d92202578efbd5007d	2026-10-04 07:08:59.447113+00	2026-09-27 07:24:58.4833+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 07:08:59.447113+00
00ea55b9-8482-4981-b046-4b11f56f0f29	usr_01J8Z9X2A4B6C8D0E2F4G6H8	75d66ccae9b0c594802b68b640154ff6bf60e6bed5ae2a276c118528700b0e94	2026-10-04 07:24:58.483257+00	2026-09-27 07:47:30.576407+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 07:24:58.483257+00
fee945f2-e276-4e4e-9964-b7a162a7a48d	usr_01J8Z9X2A4B6C8D0E2F4G6H8	f7e0f9fb283347a1b8e61e032a73ad59447149aecfc0a3cf665b098b4a04ba83	2026-10-04 07:47:30.576349+00	2026-09-27 08:02:36.814994+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 07:47:30.576349+00
4556f01d-0674-4f44-ae08-bc076b15a94b	usr_01J8Z9X2A4B6C8D0E2F4G6H8	d735a10ab5e60f1ba231836aeb2a186d2840c7faf5bba7e5658858c7bb64e653	2026-10-04 08:02:36.814946+00	2026-09-27 08:19:39.736805+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 08:02:36.814946+00
31aa60aa-75ec-4616-8ad0-5e4b0b1ac6c7	usr_01J8Z9X2A4B6C8D0E2F4G6H8	f322ecc9278f4f6b581e8f0ad8cfce076d9478879d9f1d5b97ac60c88a32b140	2026-10-04 08:19:39.73675+00	2026-09-27 08:34:53.014611+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 08:19:39.73675+00
6616d63b-c12d-4156-ad91-01a455bce69f	usr_01J8Z9X2A4B6C8D0E2F4G6H8	295398b8d5d59640afbce9ea52e170fad826978cb3af4515dda62fea1132549c	2026-10-04 08:34:53.01456+00	2026-09-27 09:08:37.520189+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 08:34:53.01456+00
90076cd0-c756-48fb-a285-93bf97d56bd0	usr_01J8Z9X2A4B6C8D0E2F4G6H8	64e852742ae2f4dff2beae19d1520f76603050e16979cb672ffe259e1a4593a1	2026-10-04 09:08:37.520123+00	2026-09-27 09:24:06.187514+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 09:08:37.520123+00
e6a24f69-da20-48e8-98b6-51f7e1aa590e	usr_01J8Z9X2A4B6C8D0E2F4G6H8	9ff705229c68aaf207712a26234c6462cf3f93df0c7771d84dd1bf67897ae03a	2026-10-04 09:24:06.18746+00	2026-09-27 09:40:01.678818+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 09:24:06.18746+00
3fc5a9ec-f1b9-4633-a379-508788398786	usr_01J8Z9X2A4B6C8D0E2F4G6H8	45fd93e4a5666ab40c6e238456d3fbadb09acdb7e6b424bb74dc367a4c8dfbd8	2026-10-04 09:40:01.678766+00	2026-09-27 10:03:22.859247+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 09:40:01.678766+00
5ddc4e65-43e7-4aa1-a1e3-c44075c2dfad	usr_01J8Z9X2A4B6C8D0E2F4G6H8	b53085acdd30de6e6ff77e7a4ad53aaf0e1279916cf42ac9b3f85befe665458d	2026-10-04 10:24:30.907201+00	\N	curl/8.20.0	::1	2026-09-27 10:24:30.907201+00
33a55a8c-17bc-48ec-9892-d6f291bb3ab8	usr_01J8Z9X2A4B6C8D0E2F4G6H8	514edda080f51924d0111b58d6aa815e0cf7204df9445689eee02f3dc05d63ca	2026-10-04 10:03:22.859184+00	2026-09-27 10:27:59.358159+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 10:03:22.859184+00
c463978d-a4f6-4abd-9bb8-03bc9a6ec050	usr_01J8Z9X2A4B6C8D0E2F4G6H8	24e1616ab575dfe2e9905db41dc52a0e3d3320b20a82d1d79e3b423d3756322a	2026-10-04 10:27:59.358074+00	2026-09-27 10:49:12.485054+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 10:27:59.358074+00
662ab6c6-f5d2-4774-b9db-f8538fde02c8	usr_01J8Z9X2A4B6C8D0E2F4G6H8	b0222fa658d8bb793f05a87602dada0ccbdae181a8dd043acf5223dceb0d4ce7	2026-10-04 10:57:58.718584+00	\N	curl/8.20.0	::1	2026-09-27 10:57:58.718584+00
76f1f7cb-ecdb-4444-a81c-96c5a0546008	usr_01J8Z9X2A4B6C8D0E2F4G6H8	8bf7a0f8bb98a44b03942349828016d9bacc7e512aeee4bc903a32bed6aeaf69	2026-10-04 10:58:14.54304+00	\N	curl/8.20.0	::1	2026-09-27 10:58:14.54304+00
a7210b85-8cdf-4fe8-9512-124c6133317c	usr_01J8Z9X2A4B6C8D0E2F4G6H8	7ee5daf618d7f9a5306b7a7d795fa4601e972839f79cdd8d3beedfd33ff37d7b	2026-10-04 10:49:12.484989+00	2026-09-28 17:44:58.409463+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-27 10:49:12.484989+00
6a6ce3ce-4711-4251-8da3-f993b9fac795	usr_01J8Z9X2A4B6C8D0E2F4G6H8	db1a00b5eb4997fae4df716f20bf5f8f7ae342b04a26235303d70da042f709d5	2026-10-05 17:44:58.409313+00	\N	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-28 17:44:58.409313+00
2f1335a2-1ccc-4b13-acd3-7ab5a0fbf945	usr_01J8Z9X2A4B6C8D0E2F4G6H8	33091fe4456cd698f94206255ec591d06dfadca3b22752f42101b6bd8bc23dcf	2026-10-05 17:52:01.965406+00	\N	curl/8.20.0	::1	2026-09-28 17:52:01.965406+00
fb3f156c-8144-4d01-b0e4-2784d3752de5	usr_01J8Z9X2A4B6C8D0E2F4G6H8	dc81aeca0dd5a9c7d28226c79dfcf16398e3f3cd52e8d1fe2488134345e06f25	2026-10-05 18:01:35.648788+00	\N	curl/8.20.0	::1	2026-09-28 18:01:35.648788+00
4d37b865-8c7f-4696-946f-4526c09bd8f7	usr_01J8Z9X2A4B6C8D0E2F4G6H8	bfae47ab4a161c3a63bfc4a8dca9a099ce4902d87ef009e67e452b4397199830	2026-10-05 18:04:17.009063+00	\N	curl/8.20.0	::1	2026-09-28 18:04:17.009063+00
976ab0cc-6be7-445e-85e3-25726a87cccd	usr_01J8Z9X2A4B6C8D0E2F4G6H8	e1e72e76f38a3de7d7b3c8623f1f1ba757255bd29255482129cfdda906c6e817	2026-10-05 17:51:36.074853+00	2026-09-28 18:06:36.382534+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-28 17:51:36.074853+00
c71d8037-17a3-4212-b45d-eba642c92ffd	usr_01J8Z9X2A4B6C8D0E2F4G6H8	09cef68bb72df00235ce67a9faab541fb9471488c932864a7f0f6de2b3836c0d	2026-10-05 18:07:01.297405+00	\N	curl/8.20.0	::1	2026-09-28 18:07:01.297405+00
0a54b6b7-df9d-4429-ae24-6c980b2fbda7	usr_01J8Z9X2A4B6C8D0E2F4G6H8	d906ecd38653fa7fc1d313cc95577a714bfc01a0f27bf389892d1e2c1ff9bf9c	2026-10-05 18:08:23.360037+00	\N	curl/8.20.0	::1	2026-09-28 18:08:23.360037+00
8a3b7114-f07b-4a7b-8d4d-c58a672f32b7	usr_01J8Z9X2A4B6C8D0E2F4G6H8	e3026df5bcfbc9efaa80b02907f70f0cbe75747182eae6179b2a6b6547dcf959	2026-10-05 18:08:56.647893+00	\N	curl/8.20.0	::1	2026-09-28 18:08:56.647893+00
0358a6b8-a0c4-4e9c-97db-c852c5988a20	usr_01J8Z9X2A4B6C8D0E2F4G6H8	58c56023b076662a4f79e8b4bea5224ab4fefd3248591da5770d25dde5b4a33b	2026-10-05 18:09:00.716018+00	\N	curl/8.20.0	::1	2026-09-28 18:09:00.716018+00
fb44453e-4893-4b7c-a586-567095cdcb5e	usr_01J8Z9X2A4B6C8D0E2F4G6H8	fa36cb2f32ddfc03c491490d5ad00a3734c54a7772497ad80efb8a0483097d27	2026-10-05 18:09:09.446977+00	\N	curl/8.20.0	::1	2026-09-28 18:09:09.446977+00
1cab140b-4fb4-4e35-b8e4-9b965686f4bb	usr_01J8Z9X2A4B6C8D0E2F4G6H8	52416424211e252e2e65146dc1b56681f640643dfc25e1954a0de1fc0be778da	2026-10-05 18:10:06.5585+00	\N	curl/8.20.0	::1	2026-09-28 18:10:06.5585+00
74c2d684-f612-46db-9dd4-2ff846559732	usr_01J8Z9X2A4B6C8D0E2F4G6H8	35451d0ad2ec56e3683e8fe3ed4fd059be968a79a3eaf8cf02b3d7f7e73bc509	2026-10-05 18:06:36.382302+00	2026-09-28 18:22:13.332946+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-28 18:06:36.382302+00
1455ca5d-83b3-42b9-831a-2062aa490c1f	usr_01J8Z9X2A4B6C8D0E2F4G6H8	cca9b505d0f6fc182348b157d4025e395a8e6a416f53451499e82b8d6ed69cb8	2026-10-05 18:22:13.332816+00	2026-09-28 18:22:15.094985+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-28 18:22:13.332817+00
58952d21-fd5e-432d-a6e4-2bd10257e57e	usr_01J8Z9X2A4B6C8D0E2F4G6H8	c81f7a529aa583705b06fd74a7164bd977aa3cb98f3f0f3eb2676b5481ec716f	2026-10-05 18:28:44.089791+00	\N	curl/8.20.0	::1	2026-09-28 18:28:44.089791+00
e21fb386-bf95-4c6a-b39f-91cb7eed12b2	usr_01J8Z9X2A4B6C8D0E2F4G6H8	a53caaa044e91624a3c810a7703c40addd73ab987d87bb1054d705f727118ec1	2026-10-05 18:30:46.1167+00	\N	curl/8.20.0	::1	2026-09-28 18:30:46.1167+00
7b02f40a-a711-4e80-94aa-fe04f8b8dfa2	usr_01J8Z9X2A4B6C8D0E2F4G6H8	1a3d5109a04ea8d0a6999f7995103fd67e0de0cf48b6ddaca5fb90d886f6b13b	2026-10-05 18:31:31.783641+00	\N	curl/8.20.0	::1	2026-09-28 18:31:31.783641+00
da614aea-a26c-4abb-b62c-a51bb36c6f9e	usr_01J8Z9X2A4B6C8D0E2F4G6H8	2b396547fce993a3fd7c336e618da36595a28cd698d4585e12e742730baf00f0	2026-10-05 18:31:56.629988+00	\N	curl/8.20.0	::1	2026-09-28 18:31:56.629988+00
cfa0c9cd-decc-4f97-9bcf-4da2abc02870	usr_01J8Z9X2A4B6C8D0E2F4G6H8	858ba53a75c2a738c71cc30a26ff80011b2ad771a412e0b9538aeb77049751a0	2026-10-05 18:22:15.094804+00	2026-09-28 18:37:17.752081+00	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-28 18:22:15.094805+00
3dc96b79-8387-4acd-b16f-626e3d2a0e5f	usr_01J8Z9X2A4B6C8D0E2F4G6H8	a567cfa73027e7b438f83d4b10e2db6169956183a979ee4c4975d234e1c79e92	2026-10-05 18:37:17.751809+00	\N	Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0	::1	2026-09-28 18:37:17.751809+00
57e535b0-ebf5-4992-ac83-748be6a45514	usr_01J8Z9X2A4B6C8D0E2F4G6H8	c32e6b148de5ec033e406822d3ad80ec02a5eda9afeaa40430e2b2f9746e402f	2026-10-05 18:42:47.051354+00	\N	curl/8.20.0	::1	2026-09-28 18:42:47.051354+00
04b3737e-7d96-43d6-bac4-c8fb72d7b119	usr_01J8Z9X2A4B6C8D0E2F4G6H8	e274042372defa72c82a4d42de2f9ac9fdd75781a2f281a7243f1ac510884725	2026-10-05 18:43:05.424488+00	\N	curl/8.20.0	::1	2026-09-28 18:43:05.424488+00
e9972bb3-cf67-4a87-8ebb-9ee796ee5d23	usr_01J8Z9X2A4B6C8D0E2F4G6H8	ec32fc1a0437f356c92e964133651b04c073572962f2382d68a93de44c663a43	2026-10-05 18:43:20.679006+00	\N	curl/8.20.0	::1	2026-09-28 18:43:20.679006+00
\.


--
-- Data for Name: spatial_ref_sys; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.spatial_ref_sys (srid, auth_name, auth_srid, srtext, proj4text) FROM stdin;
\.


--
-- Data for Name: tracker_assignments; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.tracker_assignments (id, tracker_id, bike_id, assigned_by, assigned_at, unassigned_by, unassigned_at, unassign_reason, status, notes, created_at) FROM stdin;
\.


--
-- Data for Name: trackers; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.trackers (id, imei, phone_number, sim_iccid, model, hardware_version, software_version, status, last_online_at, notes, created_at, updated_at) FROM stdin;
trk_01J8Z9X2A4B6C8D0E2F4G6H1	860123456789001	+254712345678		Concox GT06N	v2.1	v1.0.4	ASSIGNED	2026-09-27 16:23:43.635921+00		2026-09-26 21:43:12.256468+00	2026-09-27 16:23:43.635921+00
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, email, password_hash, name, created_at, updated_at) FROM stdin;
usr_01J8Z9X2A4B6C8D0E2F4G6H8	admin@motoguard.co.ke	$argon2id$v=19$m=65536,t=3,p=4$47Mzowjrkp9e/rpLfpSrGQ$SIqy3zGFhxDwb68XyanQkr649zAJ8ySm4MHp3/4prXY	Brian Otieno	2026-09-26 19:16:04.122565+00	2026-09-26 19:16:04.122565+00
\.


--
-- Data for Name: geocode_settings; Type: TABLE DATA; Schema: tiger; Owner: postgres
--

COPY tiger.geocode_settings (name, setting, unit, category, short_desc) FROM stdin;
\.


--
-- Data for Name: pagc_gaz; Type: TABLE DATA; Schema: tiger; Owner: postgres
--

COPY tiger.pagc_gaz (id, seq, word, stdword, token, is_custom) FROM stdin;
\.


--
-- Data for Name: pagc_lex; Type: TABLE DATA; Schema: tiger; Owner: postgres
--

COPY tiger.pagc_lex (id, seq, word, stdword, token, is_custom) FROM stdin;
\.


--
-- Data for Name: pagc_rules; Type: TABLE DATA; Schema: tiger; Owner: postgres
--

COPY tiger.pagc_rules (id, rule, is_custom) FROM stdin;
\.


--
-- Data for Name: topology; Type: TABLE DATA; Schema: topology; Owner: postgres
--

COPY topology.topology (id, name, srid, "precision", hasz) FROM stdin;
\.


--
-- Data for Name: layer; Type: TABLE DATA; Schema: topology; Owner: postgres
--

COPY topology.layer (topology_id, layer_id, schema_name, table_name, feature_column, feature_type, level, child_id) FROM stdin;
\.


--
-- Name: bike_breadcrumbs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.bike_breadcrumbs_id_seq', 572, true);


--
-- Name: bike_events_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.bike_events_id_seq', 22, true);


--
-- Name: topology_id_seq; Type: SEQUENCE SET; Schema: topology; Owner: postgres
--

SELECT pg_catalog.setval('topology.topology_id_seq', 1, false);


--
-- Name: bike_breadcrumbs bike_breadcrumbs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bike_breadcrumbs
    ADD CONSTRAINT bike_breadcrumbs_pkey PRIMARY KEY (id);


--
-- Name: bike_events bike_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bike_events
    ADD CONSTRAINT bike_events_pkey PRIMARY KEY (id);


--
-- Name: bikes bikes_imei_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bikes
    ADD CONSTRAINT bikes_imei_key UNIQUE (imei);


--
-- Name: bikes bikes_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bikes
    ADD CONSTRAINT bikes_pkey PRIMARY KEY (id);


--
-- Name: driver_contacts driver_contacts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.driver_contacts
    ADD CONSTRAINT driver_contacts_pkey PRIMARY KEY (id);


--
-- Name: geofences geofences_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.geofences
    ADD CONSTRAINT geofences_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_token_hash_key UNIQUE (token_hash);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: sessions sessions_refresh_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_refresh_token_hash_key UNIQUE (refresh_token_hash);


--
-- Name: tracker_assignments tracker_assignments_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tracker_assignments
    ADD CONSTRAINT tracker_assignments_pkey PRIMARY KEY (id);


--
-- Name: trackers trackers_imei_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trackers
    ADD CONSTRAINT trackers_imei_key UNIQUE (imei);


--
-- Name: trackers trackers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trackers
    ADD CONSTRAINT trackers_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_assignments_bike; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_assignments_bike ON public.tracker_assignments USING btree (bike_id, assigned_at DESC);


--
-- Name: idx_assignments_tracker; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_assignments_tracker ON public.tracker_assignments USING btree (tracker_id, assigned_at DESC);


--
-- Name: idx_bike_events_bike_time; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bike_events_bike_time ON public.bike_events USING btree (bike_id, recorded_at DESC);


--
-- Name: idx_bikes_driver; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bikes_driver ON public.bikes USING btree (driver_contact_id) WHERE (driver_contact_id IS NOT NULL);


--
-- Name: idx_bikes_geofence; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bikes_geofence ON public.bikes USING btree (geofence_id) WHERE (geofence_id IS NOT NULL);


--
-- Name: idx_bikes_imei; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bikes_imei ON public.bikes USING btree (imei);


--
-- Name: idx_bikes_user_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bikes_user_created ON public.bikes USING btree (user_id, created_at DESC);


--
-- Name: idx_bikes_user_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bikes_user_id ON public.bikes USING btree (user_id);


--
-- Name: idx_bikes_user_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bikes_user_status ON public.bikes USING btree (user_id, status);


--
-- Name: idx_breadcrumbs_bike_breaches; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_breadcrumbs_bike_breaches ON public.bike_breadcrumbs USING btree (bike_id, recorded_at DESC) WHERE (geofence_breach = true);


--
-- Name: idx_breadcrumbs_bike_time; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_breadcrumbs_bike_time ON public.bike_breadcrumbs USING btree (bike_id, recorded_at DESC);


--
-- Name: idx_geofences_active_boundary; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_geofences_active_boundary ON public.geofences USING gist (boundary) WHERE (((status)::text = 'ACTIVE'::text) AND (deleted_at IS NULL));


--
-- Name: idx_geofences_boundary; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_geofences_boundary ON public.geofences USING gist (boundary);


--
-- Name: idx_pwd_resets_token; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_pwd_resets_token ON public.password_reset_tokens USING btree (token_hash);


--
-- Name: idx_sessions_refresh_token; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_sessions_refresh_token ON public.sessions USING btree (refresh_token_hash);


--
-- Name: idx_sessions_user; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_sessions_user ON public.sessions USING btree (user_id);


--
-- Name: idx_trackers_last_online; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_trackers_last_online ON public.trackers USING btree (last_online_at DESC);


--
-- Name: idx_trackers_phone; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_trackers_phone ON public.trackers USING btree (phone_number);


--
-- Name: idx_trackers_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_trackers_status ON public.trackers USING btree (status);


--
-- Name: idx_unique_active_bike; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_unique_active_bike ON public.tracker_assignments USING btree (bike_id) WHERE (unassigned_at IS NULL);


--
-- Name: idx_unique_active_tracker; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_unique_active_tracker ON public.tracker_assignments USING btree (tracker_id) WHERE (unassigned_at IS NULL);


--
-- Name: bike_breadcrumbs bike_breadcrumbs_bike_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bike_breadcrumbs
    ADD CONSTRAINT bike_breadcrumbs_bike_id_fkey FOREIGN KEY (bike_id) REFERENCES public.bikes(id) ON DELETE CASCADE;


--
-- Name: bike_events bike_events_bike_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bike_events
    ADD CONSTRAINT bike_events_bike_id_fkey FOREIGN KEY (bike_id) REFERENCES public.bikes(id) ON DELETE CASCADE;


--
-- Name: bikes bikes_driver_contact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bikes
    ADD CONSTRAINT bikes_driver_contact_id_fkey FOREIGN KEY (driver_contact_id) REFERENCES public.driver_contacts(id) ON DELETE SET NULL;


--
-- Name: bikes bikes_geofence_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bikes
    ADD CONSTRAINT bikes_geofence_id_fkey FOREIGN KEY (geofence_id) REFERENCES public.geofences(id) ON DELETE SET NULL;


--
-- Name: bikes bikes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bikes
    ADD CONSTRAINT bikes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: geofences geofences_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.geofences
    ADD CONSTRAINT geofences_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: password_reset_tokens password_reset_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: tracker_assignments tracker_assignments_assigned_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tracker_assignments
    ADD CONSTRAINT tracker_assignments_assigned_by_fkey FOREIGN KEY (assigned_by) REFERENCES public.users(id);


--
-- Name: tracker_assignments tracker_assignments_bike_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tracker_assignments
    ADD CONSTRAINT tracker_assignments_bike_id_fkey FOREIGN KEY (bike_id) REFERENCES public.bikes(id) ON DELETE CASCADE;


--
-- Name: tracker_assignments tracker_assignments_tracker_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tracker_assignments
    ADD CONSTRAINT tracker_assignments_tracker_id_fkey FOREIGN KEY (tracker_id) REFERENCES public.trackers(id) ON DELETE CASCADE;


--
-- Name: tracker_assignments tracker_assignments_unassigned_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tracker_assignments
    ADD CONSTRAINT tracker_assignments_unassigned_by_fkey FOREIGN KEY (unassigned_by) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

