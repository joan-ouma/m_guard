export interface BikeDTO {
  id: string;
  user_id?: string;
  registration_number: string;
  label: string;
  make: string;
  model: string;
  year: number;
  color: string;
  imei: string;
  status: 'RUNNING' | 'STOPPED';
  ignition: boolean;
  speed_kmh: number;
  heading_deg: number;
  latitude: number;
  longitude: number;
  current_address: string;
  sub_county: string;
  county: string;
  battery_percent: number;
  signal_quality_csq: number;
  last_telemetry_at?: string;
  geofence_id?: string;
  driver_contact_id?: string;
<<<<<<< HEAD
=======
  rider_name?: string;
  rider_national_id?: string;
  rider_phone?: string;
>>>>>>> origin/main
  is_immobilized: boolean;
  created_at: string;
  updated_at: string;
}

export interface BikeListMeta {
  total: number;
  default_selected_bike_id?: string;
}

export interface BikeListResponse {
  data: BikeDTO[];
  meta: BikeListMeta;
}

export interface CreateBikePayload {
  registration_number: string;
  label?: string;
  make?: string;
  model?: string;
  year?: number;
  color?: string;
  imei: string;
  latitude?: number;
  longitude?: number;
  current_address?: string;
  sub_county?: string;
  county?: string;
  geofence_id?: string;
  driver_contact_id?: string;
<<<<<<< HEAD
=======
  rider_name?: string;
  rider_national_id?: string;
  rider_phone?: string;
>>>>>>> origin/main
}

export interface UpdateBikePayload {
  registration_number?: string;
  label?: string;
  make?: string;
  model?: string;
  year?: number;
  color?: string;
  current_address?: string;
  sub_county?: string;
  county?: string;
  status?: 'RUNNING' | 'STOPPED';
  geofence_id?: string;
  driver_contact_id?: string;
<<<<<<< HEAD
=======
  rider_name?: string;
  rider_national_id?: string;
  rider_phone?: string;
>>>>>>> origin/main
}

export interface BreadcrumbDTO {
  id: number;
  bike_id: string;
  latitude: number;
  longitude: number;
  speed_kmh: number;
  heading_deg: number;
  ignition: boolean;
  battery_percent: number;
  address?: string;
  geofence_breach: boolean;
  recorded_at: string;
<<<<<<< HEAD
=======
  event_type?: 'THEFT' | 'TAMPER' | 'DURESS' | 'ROUTINE';
>>>>>>> origin/main
}

export interface BikeEventDTO {
  id: number;
  bike_id: string;
  event_type: 'BIKE_TOUCHED' | 'GEOFENCE_APPROACHING' | 'ACCIDENT_DETECTED' | 'IGNITION_ON' | 'IGNITION_OFF' | string;
  latitude: number;
  longitude: number;
  recorded_at: string;
}

