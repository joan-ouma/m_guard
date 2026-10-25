import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [CommonModule],
  template: `
<div class="flex h-full w-full bg-surface font-sans text-ink overflow-hidden">

  <div class="flex-1 flex flex-col h-full overflow-hidden">

    <div class="flex-1 overflow-y-auto bg-canvas p-4 lg:p-8">
      <div class="max-w-4xl mx-auto w-full">
        <h1 class="text-2xl lg:text-3xl font-black tracking-tight text-ink mb-2">Settings &amp; Emergency Contacts</h1>
        <p class="text-muted mb-8 text-sm lg:text-base">Manage your Sacco details and emergency escalation contacts.</p>

        <div class="bg-surface rounded-2xl border border-line p-5 lg:p-8 mb-6">
          <h3 class="text-lg lg:text-xl font-bold text-ink mb-4 border-b border-line-soft pb-4">Emergency Contact</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 lg:gap-6">
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-2">Full Name</label>
              <input type="text" class="w-full bg-canvas border border-line rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-brand" placeholder="e.g. Jane Doe">
            </div>
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-2">Phone Number</label>
              <input type="text" class="w-full bg-canvas border border-line rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-brand" placeholder="e.g. +254 700 000000">
            </div>
          </div>
        </div>

        <div class="bg-surface rounded-2xl border border-line p-5 lg:p-8">
          <h3 class="text-lg lg:text-xl font-bold text-ink mb-4 border-b border-line-soft pb-4">Sacco Details</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 lg:gap-6">
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-2">Sacco Name</label>
              <input type="text" class="w-full bg-canvas border border-line rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-brand" placeholder="Enter Sacco name">
            </div>
            <div>
              <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-2">Sacco Leader Contact</label>
              <input type="text" class="w-full bg-canvas border border-line rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-brand" placeholder="Leader's phone number">
            </div>
          </div>
        </div>

        <div class="mt-6 flex justify-end">
          <button class="px-6 py-3 bg-success text-surface font-bold rounded-xl hover:bg-success-strong transition-colors cursor-pointer text-sm">Save Settings</button>
        </div>
      </div>
    </div>
  </div>
</div>
  `
})
export class SettingsComponent {}
