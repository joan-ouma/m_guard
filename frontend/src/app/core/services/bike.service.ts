import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import {
  BikeDTO,
  BikeListResponse,
  BreadcrumbDTO,
  CreateBikePayload,
  UpdateBikePayload,
} from '../models/bike.models';

@Injectable({
  providedIn: 'root',
})
export class BikeService {
  private readonly http = inject(HttpClient);

  readonly bikes = signal<BikeDTO[]>([]);
  readonly defaultSelectedBikeId = signal<string | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  loadBikes(): Observable<BikeListResponse> {
    this.isLoading.set(true);
    this.error.set(null);

    return this.http.get<BikeListResponse>('/v1/bikes').pipe(
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
    return this.http.get<{ data: BikeDTO }>(`/v1/bikes/${id}`);
  }

  getBikeBreadcrumbs(id: string, limit = 200): Observable<{ data: BreadcrumbDTO[] }> {
    return this.http.get<{ data: BreadcrumbDTO[] }>(`/v1/bikes/${id}/breadcrumbs?limit=${limit}`);
  }

  createBike(payload: CreateBikePayload): Observable<{ data: BikeDTO }> {
    this.isLoading.set(true);
    return this.http.post<{ data: BikeDTO }>('/v1/bikes', payload).pipe(
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
    return this.http.patch<{ data: BikeDTO }>(`/v1/bikes/${id}`, payload).pipe(
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
    return this.http.delete<void>(`/v1/bikes/${id}`).pipe(
      tap({
        next: () => {
          this.bikes.update((current) => current.filter((b) => b.id !== id));
        },
      })
    );
  }
}

