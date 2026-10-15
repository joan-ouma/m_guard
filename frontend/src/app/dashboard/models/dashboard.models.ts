export interface NotificationItem {
  id: string;
  title: string;
  activity: string;
  bikeId?: string;
  time: string;
  read: boolean;
}

export interface FleetBike {
  id: string;
  plate: string;
  model: string;
  imei: string;
  status: 'RUNNING' | 'STOPPED';
  movementStatus: 'Moving' | 'Stationary';
  speedText: string;
  address: string;
  county: string;
  subCounty: string;
  lat: number;
  lng: number;
  hasGeofence: boolean;
  fenceCenterLat?: number;
  fenceCenterLng?: number;
  fenceName: string;
  fenceRadius: number;
  triggerCondition: string;
  email: string;
  lastSeenAt?: string;
  isImmobilized: boolean;
  riderName: string;
  riderNationalId: string;
  riderPhone: string;
  signalQuality?: number;
  batteryPercent?: number;
}
