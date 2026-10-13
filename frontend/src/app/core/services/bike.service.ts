import { Injectable, inject, signal, effect } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import {
  BikeDTO,
  BikeEventDTO,
  BikeListResponse,
  BreadcrumbDTO,
  CreateBikePayload,
  PaginatedBreadcrumbsResponse,
  PaginatedEventsResponse,
  UpdateBikePayload,
} from '../models/bike.models';
import { AuthService } from './auth.service';
import { environment } from '../../../environments/environment';

@Injectable({
  providedIn: 'root',
})
export class BikeService {
  private readonly http = inject(HttpClient);
  private readonly authService = inject(AuthService);

  readonly bikes = signal<BikeDTO[]>([]);
  readonly defaultSelectedBikeId = signal<string | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<string | null>(null);
  readonly isConnected = signal<boolean>(false);
  readonly latestAlert = signal<any | null>(null);

  private eventSource: EventSource | null = null;

  constructor() {
    effect(() => {
      const token = this.authService.accessToken();
      if (token) {
        this.connectSSE();
      } else {
        this.disconnectSSE();
      }
    });
  }

  connectSSE(): void {
    if (typeof window === 'undefined') return;

    const token = this.authService.accessToken();
    if (!token) {
      this.disconnectSSE();
      return;
    }

    if (
      this.eventSource &&
      (this.eventSource.readyState === EventSource.OPEN ||
        this.eventSource.readyState === EventSource.CONNECTING)
    ) {
      return;
    }

    this.disconnectSSE();

    const streamUrl = `${environment.apiBaseUrl}/v1/bikes/stream`;
    const es = new EventSource(streamUrl, { withCredentials: true });
    this.eventSource = es;

    es.onopen = () => {
      this.isConnected.set(true);
    };

    es.addEventListener('bikes_init', (event: MessageEvent) => {
      try {
        const res: BikeListResponse = JSON.parse(event.data);
        if (res && Array.isArray(res.data)) {
          this.bikes.set(res.data);
          if (res.meta?.default_selected_bike_id) {
            this.defaultSelectedBikeId.set(res.meta.default_selected_bike_id);
          }
          this.isLoading.set(false);
        }
      } catch (err) {
        console.error('Failed to parse bikes_init event:', err);
      }
    });

    es.addEventListener('bike_update', (event: MessageEvent) => {
      try {
        const bike: BikeDTO = JSON.parse(event.data);
        if (bike && bike.id) {
          this.bikes.update((current) => {
            const index = current.findIndex((b) => b.id === bike.id);
            if (index >= 0) {
              const updated = [...current];
              updated[index] = bike;
              return updated;
            }
            return [bike, ...current];
          });
        }
      } catch (err) {
        console.error('Failed to parse bike_update event:', err);
      }
    });

    es.addEventListener('alert', (event: MessageEvent) => {
      try {
        const alert = JSON.parse(event.data);
        this.latestAlert.set(alert);
      } catch (err) {
        console.error('Failed to parse alert event:', err);
      }
    });

    es.onerror = () => {
      this.isConnected.set(false);
      if (es.readyState === EventSource.CLOSED) {
        this.disconnectSSE();
      }
    };
  }

  disconnectSSE(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
      this.isConnected.set(false);
    }
  }

  loadBikes(cursor?: string, limit = 25): Observable<BikeListResponse> {
    this.isLoading.set(true);
    this.error.set(null);

    let url = `${environment.apiBaseUrl}/v1/bikes?limit=${limit}`;
    if (cursor) {
      url += `&cursor=${encodeURIComponent(cursor)}`;
    }

    return this.http.get<BikeListResponse>(url).pipe(
      tap({
        next: (res) => {
          this.bikes.set(res.data);
          this.defaultSelectedBikeId.set(res.meta.default_selected_bike_id || null);
          this.isLoading.set(false);
        },
        error: (err) => {
          this.error.set(err?.error?.detail || 'Failed to load bikes');
          this.isLoading.set(false);
        },
      })
    );
  }

  getBike(id: string): Observable<{ data: BikeDTO }> {
    return this.http.get<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes/${id}`);
  }

  getBikeBreadcrumbs(
    id: string,
    limit = 25,
    date?: string,
    startDate?: string,
    endDate?: string,
    cursor?: string
  ): Observable<PaginatedBreadcrumbsResponse> {
    let url = `${environment.apiBaseUrl}/v1/bikes/${id}/breadcrumbs?limit=${limit}`;
    if (cursor) {
      url += `&cursor=${encodeURIComponent(cursor)}`;
    }
    if (date) {
      url += `&date=${encodeURIComponent(date)}`;
    } else if (startDate && endDate) {
      url += `&startDate=${encodeURIComponent(startDate)}&endDate=${encodeURIComponent(endDate)}`;
    }
    return this.http.get<PaginatedBreadcrumbsResponse>(url);
  }

  getBikeEvents(
    id: string,
    limit = 25,
    date?: string,
    startDate?: string,
    endDate?: string,
    cursor?: string
  ): Observable<PaginatedEventsResponse> {
    let url = `${environment.apiBaseUrl}/v1/bikes/${id}/events?limit=${limit}`;
    if (cursor) {
      url += `&cursor=${encodeURIComponent(cursor)}`;
    }
    if (date) {
      url += `&date=${encodeURIComponent(date)}`;
    } else if (startDate && endDate) {
      url += `&startDate=${encodeURIComponent(startDate)}&endDate=${encodeURIComponent(endDate)}`;
    }
    return this.http.get<PaginatedEventsResponse>(url);
  }

  createBike(payload: CreateBikePayload): Observable<{ data: BikeDTO }> {
    this.isLoading.set(true);
    return this.http.post<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes`, payload).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) => [res.data, ...current]);
          this.isLoading.set(false);
        },
        error: () => {
          this.isLoading.set(false);
        },
      })
    );
  }

  updateBike(id: string, payload: UpdateBikePayload): Observable<{ data: BikeDTO }> {
    return this.http.patch<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes/${id}`, payload).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) =>
            current.map((b) => (b.id === id ? res.data : b))
          );
        },
      })
    );
  }

  deleteBike(id: string): Observable<void> {
    return this.http.delete<void>(`${environment.apiBaseUrl}/v1/bikes/${id}`).pipe(
      tap({
        next: () => {
          this.bikes.update((current) => current.filter((b) => b.id !== id));
        },
      })
    );
  }

  toggleImmobilize(id: string, immobilize: boolean): Observable<{ data: BikeDTO }> {
    const action = immobilize ? 'IMMOBILIZE' : 'REACTIVATE';
    return this.http.post<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes/${id}/immobilize`, { action }).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) =>
            current.map((b) => (b.id === id ? res.data : b))
          );
        },
      })
    );
  }
}

