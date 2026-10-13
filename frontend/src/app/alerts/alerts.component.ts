import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';

@Component({
  selector: 'app-alerts',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, MobileNavComponent],
  template: `
<div class="flex h-screen w-screen bg-white font-sans text-zinc-900 overflow-hidden">
  <app-mobile-nav></app-mobile-nav>

  <aside class="hidden lg:flex w-72 bg-white border-r border-zinc-200 flex-col justify-between shrink-0 z-30">
    <div>
      <div class="h-20 px-6 flex items-center gap-3.5 border-b border-zinc-100">
        <div class="w-11 h-11 rounded-full bg-zinc-900 flex items-center justify-center text-white shrink-0 shadow-xs">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
        </div>
        <span class="font-bold text-xl tracking-tight text-zinc-900">MGuard</span>
      </div>
      <nav class="p-4 space-y-2">
        <a routerLink="/" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg> Home
        </a>
        <a routerLink="/bikes" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 17a2 2 0 11-4 0 2 2 0 014 0zM19 17a2 2 0 11-4 0 2 2 0 014 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M13 16V6a1 1 0 00-1-1H4a1 1 0 00-1 1v10a1 1 0 001 1h1m8-1a1 1 0 01-1 1H9m4-1V8a1 1 0 011-1h2.586a1 1 0 01.707.293l3.414 3.414a1 1 0 01.293.707V16a1 1 0 01-1 1h-1m-6-1a1 1 0 001 1h1M5 17a2 2 0 104 0m-4 0a2 2 0 114 0m6 0a2 2 0 104 0m-4 0a2 2 0 114 0" /></svg> Bikes
        </a>
        <a routerLink="/history" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg> History
        </a>
        <a routerLink="/alerts" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg> Alerts
        </a>
        <a routerLink="/settings" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg> Settings
        </a>
      </nav>
    </div>
  </aside>

  <div class="flex-1 flex flex-col h-full overflow-hidden">
    <header class="h-14 lg:hidden bg-white border-b border-zinc-200 px-4 flex items-center gap-3 shrink-0 z-20 relative">
      <span class="font-bold text-base text-zinc-900">Alerts</span>
    </header>

    <div class="flex-1 overflow-y-auto bg-zinc-50 p-4 lg:p-8">
      <div class="max-w-4xl mx-auto w-full">
        <h1 class="text-2xl lg:text-3xl font-black tracking-tight text-zinc-900 mb-2">Alerts &amp; Notifications</h1>
        <p class="text-zinc-500 mb-8 text-sm lg:text-base">View two-stage silent alerts, confirmed theft alerts, and history.</p>

        <div class="bg-white rounded-2xl border border-zinc-200 p-8 text-center">
          <div class="w-16 h-16 bg-rose-50 rounded-full flex items-center justify-center mx-auto mb-4 text-rose-500">
            <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          </div>
          <h3 class="text-lg font-bold text-zinc-900 mb-1">No Active Alerts</h3>
          <p class="text-zinc-500 text-sm">Quiet pre-alerts and escalation notifications will appear here.</p>
        </div>
      </div>
    </div>
  </div>
</div>
  `
})
export class AlertsComponent {}
