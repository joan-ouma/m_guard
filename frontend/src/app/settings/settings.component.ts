import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [CommonModule],
  template: `
<div class="flex h-full w-full bg-white font-sans text-zinc-900 overflow-hidden">

  <div class="flex-1 flex flex-col h-full overflow-hidden">

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
