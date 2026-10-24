import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, tap } from 'rxjs';
import { environment } from '../../../environments/environment';

export type TheftMode = 'violent_robbery' | 'theft_from_parking' | 'theft_unattended' | 'hijacking' | 'other';

export const THEFT_MODES: { value: TheftMode; label: string }[] = [
  { value: 'violent_robbery', label: 'Violent robbery' },
  { value: 'theft_from_parking', label: 'Theft from parking' },
  { value: 'theft_unattended', label: 'Stolen while unattended' },
  { value: 'hijacking', label: 'Hijacking' },
  { value: 'other', label: 'Other' },
];

export interface TheftIncident {
  id: string;
  bike_id?: string;
  mode: TheftMode;
  lost_at: string;
  location_lost: string;
  latitude: number | null;
  longitude: number | null;
  ob_number?: string;
  attachment_name?: string;
  created_at: string;
}

export interface CreateIncidentPayload {
  bike_id?: string;
  mode: TheftMode;
  lost_at: string;
  location_lost: string;
  latitude: number | null;
  longitude: number | null;
}

export type ContactCategory = 'emergency' | 'insurance';

export interface SavedContact {
  id: string;
  name: string;
  phone: string;
  category: ContactCategory;
  created_at: string;
}

export interface CreateContactPayload {
  name: string;
  phone: string;
  category: ContactCategory;
}

/** Talks to the backend for theft reports (/v1/incidents) and saved contacts (/v1/contacts). */
@Injectable({
  providedIn: 'root',
})
export class RecoveryService {
  private readonly http = inject(HttpClient);
  private readonly base = environment.apiBaseUrl;

  readonly contacts = signal<SavedContact[]>([]);
  readonly contactsLoading = signal<boolean>(false);

  loadContacts(): Observable<SavedContact[]> {
    this.contactsLoading.set(true);
    return this.http.get<{ data: SavedContact[] }>(`${this.base}/v1/contacts`).pipe(
      map((res) => res.data),
      tap({
        next: (list) => {
          this.contacts.set(list);
          this.contactsLoading.set(false);
        },
        error: () => this.contactsLoading.set(false),
      }),
    );
  }

  createIncident(payload: CreateIncidentPayload): Observable<TheftIncident> {
    return this.http
      .post<{ data: TheftIncident }>(`${this.base}/v1/incidents`, payload)
      .pipe(map((res) => res.data));
  }

  removeIncident(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/v1/incidents/${id}`);
  }

  /** Saves the OB number and, when given, the receipt file in one multipart request. */
  attachOb(incidentId: string, obNumber: string, file: File | null): Observable<void> {
    const form = new FormData();
    form.append('ob_number', obNumber);
    if (file) {
      form.append('attachment', file, file.name);
    }
    return this.http.put<void>(`${this.base}/v1/incidents/${incidentId}/ob`, form);
  }

  addContact(payload: CreateContactPayload): Observable<SavedContact> {
    return this.http.post<{ data: SavedContact }>(`${this.base}/v1/contacts`, payload).pipe(
      map((res) => res.data),
      tap((created) => this.contacts.update((list) => [created, ...list])),
    );
  }

  updateContact(id: string, payload: CreateContactPayload): Observable<SavedContact> {
    return this.http.put<{ data: SavedContact }>(`${this.base}/v1/contacts/${id}`, payload).pipe(
      map((res) => res.data),
      tap((updated) => this.contacts.update((list) => list.map((c) => (c.id === id ? updated : c)))),
    );
  }

  removeContact(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/v1/contacts/${id}`).pipe(
      tap(() => this.contacts.update((list) => list.filter((c) => c.id !== id))),
    );
  }
}
