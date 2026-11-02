import { Component, ElementRef, ViewChild, computed, inject, signal } from '@angular/core';
import { ChatOption, ChatResponse } from '../core/models/chat.models';
import { ChatService } from '../core/services/chat.service';

interface ChatLine {
  from: 'bot' | 'rider';
  text: string;
}

const ERROR_TEXT = 'Sorry, something went wrong. Please try again. / Samahani, kuna hitilafu. Jaribu tena.';

/** Floating guided assistant: the rider taps through a menu; the server answers from live bike data. */
@Component({
  selector: 'app-chat-widget',
  standalone: true,
  template: `
    <button
      type="button"
      class="fixed bottom-5 right-5 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-brand text-white shadow-lg transition-colors hover:bg-brand-strong focus:outline-none focus-visible:ring-2 focus-visible:ring-brand print:hidden"
      [attr.aria-expanded]="open()"
      aria-controls="mguard-chat-panel"
      aria-label="Open MGuard assistant"
      (click)="toggle()"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      </svg>
    </button>

    @if (open()) {
      <section
        id="mguard-chat-panel"
        class="fixed bottom-24 right-5 z-50 flex h-[min(34rem,70dvh)] w-[min(22rem,calc(100vw-2.5rem))] flex-col overflow-hidden rounded-2xl border border-line bg-surface shadow-2xl print:hidden"
        aria-label="MGuard assistant"
      >
        <header class="flex items-center justify-between border-b border-line px-4 py-3">
          <div>
            <p class="text-sm font-semibold text-ink">MGuard Assistant</p>
            <p class="text-xs text-muted">Live bike information · Taarifa za moja kwa moja</p>
          </div>
          <button
            type="button"
            class="rounded-md px-2 py-1 text-lg leading-none text-muted hover:bg-fill focus:outline-none focus-visible:ring-2 focus-visible:ring-brand"
            aria-label="Close assistant"
            (click)="open.set(false)"
          >
            ×
          </button>
        </header>

        <div #log role="log" aria-live="polite" class="flex-1 space-y-3 overflow-y-auto px-4 py-3">
          @if (earlier().length > 0) {
            <p class="text-center text-[11px] uppercase tracking-wide text-muted">Earlier · Awali</p>
            @for (line of earlier(); track $index) {
              <div class="flex opacity-75" [class.justify-end]="line.from === 'rider'">
                <p
                  class="max-w-[85%] whitespace-pre-line rounded-2xl px-3 py-2 text-sm"
                  [class.bg-brand]="line.from === 'rider'"
                  [class.text-white]="line.from === 'rider'"
                  [class.bg-fill]="line.from === 'bot'"
                  [class.text-ink]="line.from === 'bot'"
                >{{ line.text }}</p>
              </div>
            }
            <hr class="border-line" />
          }
          @for (line of lines(); track $index) {
            <div class="flex" [class.justify-end]="line.from === 'rider'">
              <p
                class="max-w-[85%] whitespace-pre-line rounded-2xl px-3 py-2 text-sm"
                [class.bg-brand]="line.from === 'rider'"
                [class.text-white]="line.from === 'rider'"
                [class.bg-fill]="line.from === 'bot'"
                [class.text-ink]="line.from === 'bot'"
              >{{ line.text }}</p>
            </div>
          }
          @if (busy()) {
            <p class="text-xs text-muted">…</p>
          }
        </div>

        <footer class="space-y-2 border-t border-line px-4 py-3">
          <div class="flex flex-col gap-2">
            @for (option of options(); track option.id) {
              <button
                type="button"
                class="rounded-xl border border-line bg-surface px-3 py-2 text-left text-sm font-medium text-ink transition-colors hover:border-brand hover:bg-brand-soft focus:outline-none focus-visible:ring-2 focus-visible:ring-brand disabled:opacity-50"
                [disabled]="busy()"
                (click)="choose(option)"
              >
                {{ option.label }}
              </button>
            }
          </div>
          <div class="flex justify-between text-xs">
            <button
              type="button"
              class="rounded-md px-2 py-1 font-medium text-muted hover:bg-fill disabled:opacity-40"
              [disabled]="!canGoBack() || busy()"
              (click)="back()"
            >
              ← Back / Rudi
            </button>
            <button
              type="button"
              class="rounded-md px-2 py-1 font-medium text-muted hover:bg-fill disabled:opacity-40"
              [disabled]="busy()"
              (click)="startOver()"
            >
              Start over / Anza upya
            </button>
          </div>
          @if (earlier().length > 0) {
            <button
              type="button"
              class="w-full rounded-md px-2 py-1 text-xs font-medium text-muted hover:bg-fill disabled:opacity-40"
              [disabled]="busy()"
              (click)="clearHistory()"
            >
              Clear history / Futa historia
            </button>
          }
        </footer>
      </section>
    }
  `,
})
export class ChatWidgetComponent {
  private readonly chat = inject(ChatService);

  @ViewChild('log') private logRef?: ElementRef<HTMLElement>;

  readonly open = signal(false);
  readonly busy = signal(false);
  /** Messages from earlier visits, loaded from the server. */
  readonly earlier = signal<ChatLine[]>([]);
  /** The current visit. */
  readonly lines = signal<ChatLine[]>([]);
  readonly options = signal<ChatOption[]>([]);
  /** Selections made so far, e.g. ["lang:en", "bike:…", "action:location"]. The server validates them. */
  readonly path = signal<string[]>([]);
  readonly canGoBack = computed(() => this.path().length > 0);

  toggle(): void {
    this.open.update((value) => !value);
    if (this.open()) {
      this.loadHistory();
      if (this.lines().length === 0) {
        this.startOver();
      }
    }
  }

  choose(option: ChatOption): void {
    if (this.busy()) {
      return;
    }
    this.lines.update((lines) => [...lines, { from: 'rider', text: option.label }]);
    this.send([...this.path(), option.id], option.label);
  }

  clearHistory(): void {
    if (this.busy()) {
      return;
    }
    this.chat.clearHistory().subscribe({
      next: () => this.earlier.set([]),
      error: () => this.lines.update((lines) => [...lines, { from: 'bot', text: ERROR_TEXT }]),
    });
  }

  back(): void {
    if (this.busy() || !this.canGoBack()) {
      return;
    }
    this.send(this.path().slice(0, -1));
  }

  startOver(): void {
    if (this.busy()) {
      return;
    }
    this.lines.set([]);
    this.send([]);
  }

  private loadHistory(): void {
    this.chat.history().subscribe({
      next: (items) =>
        this.earlier.set(items.map((item) => ({ from: item.role === 'rider' ? 'rider' : 'bot', text: item.text }))),
      error: () => this.earlier.set([]),
    });
  }

  private send(path: string[], label?: string): void {
    this.busy.set(true);
    this.chat.respond(path, label).subscribe({
      next: (res: ChatResponse) => {
        this.path.set(path);
        this.options.set(res.options ?? []);
        this.lines.update((lines) => [...lines, { from: 'bot', text: res.message }]);
        this.finish();
      },
      error: () => {
        this.options.set([]);
        this.lines.update((lines) => [...lines, { from: 'bot', text: ERROR_TEXT }]);
        this.finish();
      },
    });
  }

  private finish(): void {
    this.busy.set(false);
    setTimeout(() => {
      const el = this.logRef?.nativeElement;
      if (el) {
        el.scrollTop = el.scrollHeight;
      }
    });
  }
}
