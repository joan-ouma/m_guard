import { Component, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-mobile-nav',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive],
  template: `
    @if (open()) {
      <div
        (click)="close()"
        class="fixed inset-0 bg-zinc-900/50 backdrop-blur-sm z-50 lg:hidden"
      ></div>

      <aside
        class="fixed inset-y-0 left-0 w-72 max-w-[85vw] bg-white z-50 lg:hidden flex flex-col justify-between shadow-2xl border-r border-zinc-200 animate-in slide-in-from-left duration-200"
      >
        <div>
          <div class="h-16 px-6 flex items-center justify-between border-b border-zinc-100">
            <div class="flex items-center gap-3.5">
              <div class="w-10 h-10 rounded-full bg-zinc-900 flex items-center justify-center text-white shrink-0 shadow-xs">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
              </div>
              <span class="font-bold text-xl tracking-tight text-zinc-900">MGuard</span>
            </div>
            <button type="button" (click)="close()" class="w-9 h-9 rounded-xl flex items-center justify-center text-zinc-400 hover:text-zinc-900 hover:bg-zinc-100 transition-colors cursor-pointer">✕</button>
          </div>

          <nav class="p-4 space-y-1.5">
            <a routerLink="/" routerLinkActive="!bg-zinc-900 !text-white" [routerLinkActiveOptions]="{exact:true}" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
              </svg>
              Home
            </a>

            <a routerLink="/bikes" routerLinkActive="!bg-zinc-900 !text-white" [routerLinkActiveOptions]="{exact:true}" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 17a2 2 0 11-4 0 2 2 0 014 0zM19 17a2 2 0 11-4 0 2 2 0 014 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 16V6a1 1 0 00-1-1H4a1 1 0 00-1 1v10a1 1 0 001 1h1m8-1a1 1 0 01-1 1H9m4-1V8a1 1 0 011-1h2.586a1 1 0 01.707.293l3.414 3.414a1 1 0 01.293.707V16a1 1 0 01-1 1h-1m-6-1a1 1 0 001 1h1M5 17a2 2 0 104 0m-4 0a2 2 0 114 0m6 0a2 2 0 104 0m-4 0a2 2 0 114 0" />
              </svg>
              Bikes
            </a>

            <a routerLink="/history" routerLinkActive="!bg-zinc-900 !text-white" [routerLinkActiveOptions]="{exact:true}" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              History
            </a>

            <a routerLink="/" [fragment]="'geofence'" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
              </svg>
              Geofence
            </a>

            <a routerLink="/alerts" routerLinkActive="!bg-zinc-900 !text-white" [routerLinkActiveOptions]="{exact:true}" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
              </svg>
              Alerts
            </a>

            <a routerLink="/emergency" routerLinkActive="!bg-zinc-900 !text-white" [routerLinkActiveOptions]="{exact:true}" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
              </svg>
              Police / Help
            </a>

            <a routerLink="/settings" routerLinkActive="!bg-zinc-900 !text-white" [routerLinkActiveOptions]="{exact:true}" (click)="close()" class="min-h-[48px] px-4 py-3 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 transition-colors cursor-pointer">
              <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              Settings
            </a>
          </nav>
        </div>

        <div class="p-4 border-t border-zinc-100">
          <div class="flex items-center justify-between gap-3 p-3 rounded-2xl bg-zinc-50 border border-zinc-200/80">
            <div class="flex items-center gap-3 min-w-0">
              <div class="w-10 h-10 rounded-full bg-zinc-900 text-white font-bold text-sm flex items-center justify-center shrink-0">
                {{ getUserInitials() }}
              </div>
              <div class="min-w-0">
                <p class="text-sm font-bold text-zinc-900 truncate">{{ authService.currentUser()?.name || 'Rider' }}</p>
                <p class="text-xs text-zinc-500 truncate">{{ authService.currentUser()?.email }}</p>
              </div>
            </div>
            <button type="button" (click)="handleLogout()" title="Log out" class="w-10 h-10 rounded-xl flex items-center justify-center text-zinc-400 hover:text-rose-600 hover:bg-rose-50 transition-colors shrink-0 cursor-pointer">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
            </button>
          </div>
        </div>
      </aside>
    }

    <button
      type="button"
      (click)="toggle()"
      title="Open Menu"
      class="lg:hidden w-10 h-10 rounded-xl flex items-center justify-center text-zinc-700 hover:bg-zinc-100 transition-colors cursor-pointer shrink-0"
    >
      <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>
  `,
})
export class MobileNavComponent {
  readonly authService = inject(AuthService);
  readonly open = signal(false);

  toggle(): void {
    this.open.set(!this.open());
  }

  close(): void {
    this.open.set(false);
  }

  getUserInitials(): string {
    const name = this.authService.currentUser()?.name ?? '';
    return name.split(' ').map((w: string) => w[0]).join('').toUpperCase().slice(0, 2) || 'RD';
  }

  handleLogout(): void {
    this.authService.logout().subscribe();
  }
}
