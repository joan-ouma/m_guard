import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterLinkActive } from '@angular/router';
import Swal from 'sweetalert2';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { ContactCategory, RecoveryService, SavedContact } from '../core/services/recovery.service';

type CategoryFilter = 'ALL' | ContactCategory;

const CATEGORY_LABELS: Record<ContactCategory, string> = {
  emergency: 'Emergency contact',
  insurance: 'Insurance broker',
};

@Component({
  selector: 'app-contacts',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive, MobileNavComponent],
  templateUrl: './contacts.component.html',
})
export class ContactsComponent implements OnInit {
  private readonly recoveryService = inject(RecoveryService);

  readonly categoryLabels = CATEGORY_LABELS;
  readonly pageSizes = [10, 25, 50];

  readonly filter = signal<CategoryFilter>('ALL');
  readonly pageSize = signal<number>(10);
  readonly page = signal<number>(1);

  // Add form
  readonly formName = signal<string>('');
  readonly formPhone = signal<string>('');
  readonly formCategory = signal<ContactCategory>('emergency');
  readonly formError = signal<string | null>(null);
  readonly saving = signal<boolean>(false);
  readonly addOpen = signal<boolean>(false);
  readonly editingId = signal<string | null>(null);

  openAdd(): void {
    this.resetForm();
    this.addOpen.set(true);
  }

  openEdit(c: SavedContact): void {
    this.resetForm();
    this.editingId.set(c.id);
    this.formName.set(c.name);
    this.formPhone.set(c.phone);
    this.formCategory.set(c.category);
    this.addOpen.set(true);
  }

  closeAdd(): void {
    if (this.saving()) return;
    this.addOpen.set(false);
  }

  private resetForm(): void {
    this.formError.set(null);
    this.editingId.set(null);
    this.formName.set('');
    this.formPhone.set('');
    this.formCategory.set('emergency');
  }

  ngOnInit(): void {
    this.recoveryService.loadContacts().subscribe({
      error: () => this.formError.set('Could not load your contacts. Check your connection and refresh.'),
    });
  }

  readonly filtered = computed<SavedContact[]>(() => {
    const f = this.filter();
    const list = this.recoveryService.contacts();
    return f === 'ALL' ? list : list.filter((c) => c.category === f);
  });

  readonly totalPages = computed<number>(() => Math.max(1, Math.ceil(this.filtered().length / this.pageSize())));
  readonly pageItems = computed<SavedContact[]>(() => {
    const start = (this.page() - 1) * this.pageSize();
    return this.filtered().slice(start, start + this.pageSize());
  });
  readonly hasPrev = computed(() => this.page() > 1);
  readonly hasNext = computed(() => this.page() < this.totalPages());

  setFilter(value: CategoryFilter): void {
    this.filter.set(value);
    this.page.set(1);
  }

  onPageSizeChange(event: Event): void {
    this.pageSize.set(Number((event.target as HTMLSelectElement).value));
    this.page.set(1);
  }

  prevPage(): void {
    if (this.hasPrev()) this.page.update((p) => p - 1);
  }

  nextPage(): void {
    if (this.hasNext()) this.page.update((p) => p + 1);
  }

  saveContact(): void {
    this.formError.set(null);
    const name = this.formName().trim();
    const phone = this.formPhone().trim();

    if (!name) {
      this.formError.set('Enter the name.');
      return;
    }
    if (!/^\+?[0-9\s-]{7,15}$/.test(phone)) {
      this.formError.set('Enter a valid phone number, for example 0712 345 678 or +254 712 345 678.');
      return;
    }

    const payload = { name, phone, category: this.formCategory() };
    const id = this.editingId();
    const request = id ? this.recoveryService.updateContact(id, payload) : this.recoveryService.addContact(payload);

    this.saving.set(true);
    request.subscribe({
      next: () => {
        this.saving.set(false);
        this.resetForm();
        this.page.set(1);
        this.addOpen.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.saving.set(false);
        this.formError.set(this.errorMessage(err));
      },
    });
  }

  private errorMessage(err: HttpErrorResponse): string {
    const errors = err?.error?.errors as Record<string, string[]> | undefined;
    if (errors) {
      const first = Object.values(errors).flat()[0];
      if (first) return first;
    }
    return err?.error?.detail || 'Something went wrong. Please try again.';
  }

  removeContact(contact: SavedContact): void {
    Swal.fire({
      title: `Remove ${contact.name}?`,
      text: 'This contact will no longer be listed.',
      icon: 'warning',
      showCancelButton: true,
      confirmButtonColor: '#e11d48',
      cancelButtonColor: '#71717a',
      confirmButtonText: 'Yes, remove',
      cancelButtonText: 'Cancel',
      reverseButtons: true,
    }).then((result) => {
      if (result.isConfirmed) {
        this.recoveryService.removeContact(contact.id).subscribe({
          next: () => this.page.update((p) => Math.min(p, this.totalPages())),
          error: (err: HttpErrorResponse) => this.formError.set(this.errorMessage(err)),
        });
      }
    });
  }

  categoryLabel(c: SavedContact): string {
    return CATEGORY_LABELS[c.category];
  }
}
