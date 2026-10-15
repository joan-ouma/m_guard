import { Component, OnInit, OnDestroy, computed, inject, signal, ElementRef, viewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO, BikeListResponse } from '../core/models/bike.models';
import { EMERGENCY_CONTACTS, EmergencyContact } from './emergency.data';
import * as L from 'leaflet';

@Component({
  selector: 'app-emergency',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive, MobileNavComponent],
  templateUrl: './emergency.component.html',
})
export class EmergencyComponent implements OnInit, OnDestroy {
  private readonly bikeService = inject(BikeService);

  readonly contacts = signal<EmergencyContact[]>(EMERGENCY_CONTACTS);
  readonly searchQuery = signal<string>('');
  readonly selectedType = signal<'ALL' | 'police' | 'ambulance' | 'national'>('ALL');
  readonly selectedCounty = signal<string>('ALL');
  readonly activeView = signal<'split' | 'list' | 'map'>('list');
  readonly selectedContact = signal<EmergencyContact | null>(null);
  readonly copiedReport = signal<boolean>(false);

  // User bikes for location proximity and incident report
  readonly bikes = signal<BikeDTO[]>([]);
  readonly selectedBike = signal<BikeDTO | null>(null);

  // Available unique counties from directory
  readonly counties = computed<string[]>(() => {
    const set = new Set<string>();
    EMERGENCY_CONTACTS.forEach((c) => {
      if (c.county && c.county !== 'National') set.add(c.county);
    });
    return Array.from(set).sort();
  });

  // Filtered contacts list
  readonly filteredContacts = computed<EmergencyContact[]>(() => {
    const q = this.searchQuery().toLowerCase().trim();
    const type = this.selectedType();
    const county = this.selectedCounty();
    const bike = this.selectedBike();

    let list = this.contacts().filter((c) => {
      // Type filter
      if (type !== 'ALL' && c.type !== type) return false;
      // County filter
      if (county !== 'ALL' && c.county !== county && c.county !== 'National') return false;

      // Text search
      if (!q) return true;
      const haystack = `${c.name} ${c.county} ${c.subCounty} ${c.address} ${c.phone} ${c.altPhone || ''}`.toLowerCase();
      return haystack.includes(q);
    });

    // If bike location is known, sort by distance
    if (bike && bike.latitude && bike.longitude) {
      list = [...list].sort((a, b) => {
        // Keep national hotlines at top
        if (a.type === 'national' && b.type !== 'national') return -1;
        if (b.type === 'national' && a.type !== 'national') return 1;

        const distA = this.calculateDistance(bike.latitude, bike.longitude, a.latitude, a.longitude);
        const distB = this.calculateDistance(bike.latitude, bike.longitude, b.latitude, b.longitude);
        return distA - distB;
      });
    }

    return list;
  });

  // Leaflet map references
  private map: L.Map | null = null;
  private markersLayer: L.LayerGroup | null = null;
  private bikeMarker: L.Marker | null = null;

  ngOnInit(): void {
    // Load user bikes for nearest station calculation and report generator
    this.bikeService.loadBikes().subscribe({
      next: (res: BikeListResponse) => {
        const list = res.data || [];
        this.bikes.set(list);
        if (list.length > 0) {
          this.selectedBike.set(list[0]);
        }
      },
    });

    // Check screen width to default to split on desktop
    if (typeof window !== 'undefined' && window.innerWidth >= 1024) {
      this.activeView.set('split');
    }
  }

  ngOnDestroy(): void {
    if (this.map) {
      this.map.remove();
      this.map = null;
    }
  }

  setView(mode: 'split' | 'list' | 'map'): void {
    this.activeView.set(mode);
    if (mode === 'map' || mode === 'split') {
      setTimeout(() => this.initMapIfNeeded(), 100);
    }
  }

  setTypeFilter(type: 'ALL' | 'police' | 'ambulance' | 'national'): void {
    this.selectedType.set(type);
    this.updateMapMarkers();
  }

  setCountyFilter(county: string): void {
    this.selectedCounty.set(county);
    this.updateMapMarkers();
  }

  selectContact(c: EmergencyContact): void {
    this.selectedContact.set(c);

    // If on mobile list view, switch to map to show location
    if (this.activeView() === 'list' && typeof window !== 'undefined' && window.innerWidth < 1024) {
      this.activeView.set('map');
      setTimeout(() => {
        this.initMapIfNeeded();
        this.flyToContact(c);
      }, 150);
    } else {
      this.initMapIfNeeded();
      this.flyToContact(c);
    }
  }

  flyToContact(c: EmergencyContact): void {
    if (!this.map) return;
    this.map.flyTo([c.latitude, c.longitude], 15, { duration: 1 });
  }

  getDistanceToBike(c: EmergencyContact): string | null {
    const bike = this.selectedBike();
    if (!bike || !bike.latitude || !bike.longitude) return null;
    if (c.type === 'national' && c.county === 'National') return null;

    const km = this.calculateDistance(bike.latitude, bike.longitude, c.latitude, c.longitude);
    if (km < 1) {
      return `${Math.round(km * 1000)}m away`;
    }
    return `${km.toFixed(1)} km away`;
  }

  private calculateDistance(lat1: number, lon1: number, lat2: number, lon2: number): number {
    const R = 6371; // Earth radius in km
    const dLat = ((lat2 - lat1) * Math.PI) / 180;
    const dLon = ((lon2 - lon1) * Math.PI) / 180;
    const a =
      Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos((lat1 * Math.PI) / 180) *
        Math.cos((lat2 * Math.PI) / 180) *
        Math.sin(dLon / 2) *
        Math.sin(dLon / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
  }

  copyPoliceReport(): void {
    const bike = this.selectedBike();
    const now = new Date().toLocaleString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });

    let text = `🚨 MOTOGUARD EMERGENCY POLICE REPORT\n`;
    text += `Time: ${now}\n`;

    if (bike) {
      text += `Motorcycle: ${bike.registration_number} (${bike.model || 'Motorcycle'})\n`;
      text += `Color: ${bike.color || 'N/A'} | Year: ${bike.year || 'N/A'}\n`;
      text += `Tracker IMEI: ${bike.imei}\n`;
      if (bike.latitude && bike.longitude) {
        text += `Last GPS Coordinates: ${bike.latitude.toFixed(5)}, ${bike.longitude.toFixed(5)}\n`;
        text += `Google Maps Link: https://maps.google.com/?q=${bike.latitude},${bike.longitude}\n`;
      }
      if (bike.current_address) {
        text += `Last Known Landmark: ${bike.current_address}\n`;
      }
      text += `Engine Starter: ${bike.is_immobilized ? 'IMMOBILIZED (LOCKED)' : 'ACTIVE'}\n`;
    } else {
      text += `Motorcycle details not selected.\n`;
    }

    navigator.clipboard.writeText(text).then(() => {
      this.copiedReport.set(true);
      setTimeout(() => this.copiedReport.set(false), 3000);
    });
  }

  private initMapIfNeeded(): void {
    const el = document.getElementById('emergencyMap');
    if (!el) return;

    if (!this.map) {
      // Center default on Nairobi CBD or user bike
      const bike = this.selectedBike();
      const initialLat = bike && bike.latitude ? bike.latitude : -1.2864;
      const initialLng = bike && bike.longitude ? bike.longitude : 36.8172;

      this.map = L.map('emergencyMap', {
        zoomControl: true,
      }).setView([initialLat, initialLng], 13);

      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '© OpenStreetMap contributors',
      }).addTo(this.map);

      this.markersLayer = L.layerGroup().addTo(this.map);
    } else {
      this.map.invalidateSize();
    }

    this.updateMapMarkers();
  }

  private updateMapMarkers(): void {
    if (!this.map || !this.markersLayer) return;
    this.markersLayer.clearLayers();

    const bike = this.selectedBike();
    // Plot user bike if present
    if (bike && bike.latitude && bike.longitude) {
      const bikeIcon = L.divIcon({
        className: 'custom-bike-marker',
        html: `
          <div style="background:#09090b; color:white; width:34px; height:34px; border-radius:50%; display:flex; align-items:center; justify-content:center; box-shadow:0 4px 12px rgba(0,0,0,0.3); border:2px solid white; cursor:pointer;">
            <svg style="width:18px; height:18px;" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
            </svg>
          </div>
        `,
        iconSize: [34, 34],
        iconAnchor: [17, 34],
      });

      this.bikeMarker = L.marker([bike.latitude, bike.longitude], { icon: bikeIcon })
        .addTo(this.markersLayer)
        .bindPopup(`<strong>Your Motorcycle: ${bike.registration_number}</strong><br>${bike.current_address || 'Current Position'}`);
    }

    // Plot station markers
    const list = this.filteredContacts();
    list.forEach((c) => {
      if (!c.latitude || !c.longitude) return;

      const isPolice = c.type === 'police';
      const isAmbulance = c.type === 'ambulance';
      const bg = isPolice ? '#2563eb' : isAmbulance ? '#dc2626' : '#4f46e5';

      const icon = L.divIcon({
        className: 'station-marker',
        html: `
          <div style="background:${bg}; color:white; width:32px; height:32px; border-radius:10px; display:flex; align-items:center; justify-content:center; box-shadow:0 4px 10px rgba(0,0,0,0.25); border:2px solid white; cursor:pointer;">
            ${
              isPolice
                ? `<svg style="width:17px; height:17px;" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>`
                : `<svg style="width:17px; height:17px;" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>`
            }
          </div>
        `,
        iconSize: [32, 32],
        iconAnchor: [16, 16],
      });

      const dist = this.getDistanceToBike(c);
      const distHtml = dist ? `<div style="font-size:11px; color:#2563eb; font-weight:bold; margin-top:3px;">📍 ${dist}</div>` : '';

      const popupContent = `
        <div style="min-width:200px; font-family:sans-serif;">
          <h4 style="margin:0 0 4px 0; font-size:14px; font-weight:800; color:#09090b;">${c.name}</h4>
          <p style="margin:0; font-size:12px; color:#52525b;">${c.address}</p>
          ${distHtml}
          <div style="margin-top:10px; display:flex; gap:6px;">
            <a href="tel:${c.phone}" style="background:#09090b; color:white; padding:6px 12px; border-radius:8px; text-decoration:none; font-size:12px; font-weight:bold; display:inline-block;">Call: ${c.phone}</a>
          </div>
        </div>
      `;

      L.marker([c.latitude, c.longitude], { icon })
        .addTo(this.markersLayer!)
        .bindPopup(popupContent);
    });
  }
}
