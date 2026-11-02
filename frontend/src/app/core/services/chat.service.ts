import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ChatHistoryItem, ChatResponse } from '../models/chat.models';

@Injectable({
  providedIn: 'root',
})
export class ChatService {
  private readonly http = inject(HttpClient);
  private readonly apiBase = environment.apiBaseUrl;

  /** Sends the rider's menu selections so far. `label` is the tapped option's text, stored with answers. */
  respond(path: string[], label?: string): Observable<ChatResponse> {
    return this.http
      .post<{ data: ChatResponse }>(`${this.apiBase}/v1/chat`, { path, label })
      .pipe(map((res) => res.data));
  }

  history(limit = 30): Observable<ChatHistoryItem[]> {
    return this.http
      .get<{ data: ChatHistoryItem[] }>(`${this.apiBase}/v1/chat/history`, { params: { limit } })
      .pipe(map((res) => res.data ?? []));
  }

  clearHistory(): Observable<void> {
    return this.http.delete<void>(`${this.apiBase}/v1/chat/history`);
  }
}
