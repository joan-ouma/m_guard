import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, MobileNavComponent],
  template: `
<div class="flex h-screen w-screen bg-white font-sans text-zinc-900 overflow-hidden">

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
        <a routerLink="/" [fragment]="'geofence'" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" /></svg> Geofence
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
    <header class="h-16 lg:hidden bg-white border-b border-zinc-200 px-4 flex items-center gap-3 shrink-0 z-20 relative">
      <app-mobile-nav></app-mobile-nav>
      <span class="font-bold text-base text-zinc-900">Settings</span>
    </header>

    <div class="flex-1 overflow-y-auto bg-zinc-50 p-4 lg:p-8">
      <div class="max-w-4xl mx-auto w-full">
        <h1 class="text-2xl lg:text-3xl font-black tracking-tight text-zinc-900 mb-2">Settings &amp; Emergency Contacts</h1>
        <p class="text-zinc-500 mb-8 text-sm lg:text-base">Manage your Sacco details and emergency escalation contacts.</p>

        <div class="bg-white rounded-2xl border border-zinc-200 p-5 lg:p-8 mb-6">
          <h3 class="text-lg lg:text-xl font-bold text-zinc-900 mb-4 border-b border-zinc-100 pb-4">Emergency Contact</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 lg:gap-6">
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-zinc-500 mb-2">Full Name</label>
              <input type="text" class="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-zinc-900" placeholder="e.g. Jane Doe">
            </div>
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-zinc-500 mb-2">Phone Number</label>
              <input type="text" class="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-zinc-900" placeholder="e.g. +254 700 000000">
            </div>
          </div>
        </div>

        <div class="bg-white rounded-2xl border border-zinc-200 p-5 lg:p-8">
          <h3 class="text-lg lg:text-xl font-bold text-zinc-900 mb-4 border-b border-zinc-100 pb-4">Sacco Details</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 lg:gap-6">
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-zinc-500 mb-2">Sacco Name</label>
              <input type="text" class="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-zinc-900" placeholder="Enter Sacco name">
            </div>
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-zinc-500 mb-2">Sacco Leader Contact</label>
              <input type="text" class="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-zinc-900" placeholder="Leader's phone number">
            </div>
          </div>
        </div>

        <div class="mt-6 flex justify-end">
          <button class="px-6 py-3 bg-emerald-600 text-white font-bold rounded-xl hover:bg-emerald-700 transition-colors cursor-pointer text-sm">Save Settings</button>
        </div>
      </div>
    </div>
  </div>
</div>
  `
})
export class SettingsComponent {}
