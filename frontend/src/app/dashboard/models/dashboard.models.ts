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
  fenceName: string;
  fenceRadius: number;
  triggerCondition: string;
  email: string;
  lastSeenAt?: string;
  isImmobilized: boolean;
  riderName: string;
  riderNationalId: string;
  riderPhone: string;
}

export const DEFAULT_BIKE: FleetBike = {
  id: '',
  plate: 'NO BIKE',
  model: 'No asset registered',
  imei: '',
  status: 'STOPPED',
  movementStatus: 'Stationary',
  speedText: '0.0 km/h • Ignition OFF',
  address: 'Nairobi CBD',
  county: 'Nairobi County',
  subCounty: 'Starehe',
  lat: -1.286389,
  lng: 36.817223,
  fenceName: 'Central Logistics Depot',
  fenceRadius: 2500,
  triggerCondition: 'Alert on Exit',
  email: '',
  lastSeenAt: new Date().toISOString(),
  isImmobilized: false,
  riderName: 'Unassigned',
  riderNationalId: 'N/A',
  riderPhone: 'N/A',
};
