import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AuthService } from '../core/services/auth.service';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [CommonModule],
  template: `
<div class="flex h-full w-full bg-surface font-sans text-ink overflow-hidden">

  <div class="flex-1 flex flex-col h-full overflow-hidden">

    <div class="flex-1 overflow-y-auto bg-canvas p-4 lg:p-8">
      <div class="max-w-4xl mx-auto w-full">
        <h2 class="text-lg lg:text-xl font-bold tracking-tight text-ink mb-2">Emergency contacts</h2>
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

        <div class="mt-10 bg-surface rounded-2xl border border-brand-line p-5 lg:p-8">
          <h3 class="text-lg font-bold text-ink mb-2">Delete account</h3>
          <p class="text-sm text-muted mb-4">
            This permanently deletes your account, your bikes and their tracking history, your theft reports, your contacts, and your assistant history. This cannot be undone.
          </p>

          @if (!confirmingDelete()) {
            <button
              type="button"
              class="px-6 py-3 border border-brand text-brand font-bold rounded-xl hover:bg-brand-soft transition-colors cursor-pointer text-sm"
              (click)="startDelete()"
            >
              Delete my account
            </button>
          } @else {
            <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-2" for="delete-password">Enter your password to confirm</label>
            <input
              id="delete-password"
              type="password"
              autocomplete="current-password"
              class="w-full max-w-sm bg-canvas border border-line rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-brand mb-3"
              [value]="deletePassword()"
              (input)="deletePassword.set($any($event.target).value)"
            />
            @if (deleteError()) {
              <p class="text-sm text-brand mb-3" role="alert">{{ deleteError() }}</p>
            }
            <div class="flex flex-wrap gap-3">
              <button
                type="button"
                class="px-6 py-3 bg-brand text-surface font-bold rounded-xl hover:bg-brand-strong transition-colors cursor-pointer text-sm disabled:opacity-50"
                [disabled]="!deletePassword() || deleting()"
                (click)="confirmDelete()"
              >
                {{ deleting() ? 'Deleting…' : 'Permanently delete account' }}
              </button>
              <button
                type="button"
                class="px-6 py-3 text-ink-soft font-bold rounded-xl hover:bg-fill transition-colors cursor-pointer text-sm"
                [disabled]="deleting()"
                (click)="cancelDelete()"
              >
                Cancel
              </button>
            </div>
          }
        </div>
      </div>
    </div>
  </div>
</div>
  `
})
export class SettingsComponent {
  private readonly auth = inject(AuthService);

  readonly confirmingDelete = signal(false);
  readonly deletePassword = signal('');
  readonly deleting = signal(false);
  readonly deleteError = signal<string | null>(null);

  startDelete(): void {
    this.confirmingDelete.set(true);
    this.deleteError.set(null);
  }

  cancelDelete(): void {
    this.confirmingDelete.set(false);
    this.deletePassword.set('');
    this.deleteError.set(null);
  }

  confirmDelete(): void {
    if (!this.deletePassword() || this.deleting()) {
      return;
    }
    this.deleting.set(true);
    this.deleteError.set(null);
    this.auth.deleteAccount(this.deletePassword()).subscribe({
      error: (err) => {
        this.deleting.set(false);
        this.deleteError.set(err?.error?.detail ?? 'Could not delete your account. Please try again.');
      },
    });
  }
}
