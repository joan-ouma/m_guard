import { Component, computed, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { PwaService } from '../../../core/services/pwa.service';
import { NotificationService } from '../../../core/services/notification.service';

export type OnboardingStep = 'INSTALL' | 'NOTIFICATIONS';

@Component({
  selector: 'app-onboarding-modal',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './onboarding-modal.component.html',
})
export class OnboardingModalComponent {
  readonly pwa = inject(PwaService);
  readonly notificationService = inject(NotificationService);

  readonly isOpen = input<boolean>(false);
  readonly close = output<void>();

  readonly installedSuccess = signal<boolean>(false);
  readonly notificationsEnabled = signal<boolean>(false);

  readonly needsInstall = computed<boolean>(() => {
    return !this.pwa.isInstalled() && !this.installedSuccess();
  });

  readonly needsNotifications = computed<boolean>(() => {
    if (!this.notificationService.isSupported()) return false;
    const isGranted = this.notificationService.permission() === 'granted';
    const isSubscribed = this.notificationService.isSnsSubscribed();
    return !isGranted && !isSubscribed && !this.notificationsEnabled();
  });

  // Dynamically compute ONLY steps that are actually needed, with fallback so modal always shows
  readonly activeSteps = computed<OnboardingStep[]>(() => {
    const steps: OnboardingStep[] = [];
    if (this.needsInstall()) {
      steps.push('INSTALL');
    }
    if (this.needsNotifications()) {
      steps.push('NOTIFICATIONS');
    }
    return steps.length > 0 ? steps : ['INSTALL', 'NOTIFICATIONS'];
  });

  readonly activeStepIndex = signal<number>(0);

  readonly currentStep = computed<OnboardingStep | null>(() => {
    const steps = this.activeSteps();
    const idx = this.activeStepIndex();
    if (idx >= 0 && idx < steps.length) {
      return steps[idx];
    }
    return steps[0] || null;
  });

  readonly isLastStep = computed<boolean>(() => {
    return this.activeStepIndex() >= this.activeSteps().length - 1;
  });

  nextStep(): void {
    if (!this.isLastStep()) {
      this.activeStepIndex.update((i) => i + 1);
    } else {
      this.handleClose();
    }
  }

  handleClose(): void {
    this.close.emit();
  }

  async onInstallApp(): Promise<void> {
    const success = await this.pwa.installApp();
    if (success) {
      this.installedSuccess.set(true);
      if (!this.isLastStep()) {
        setTimeout(() => this.nextStep(), 600);
      } else {
        setTimeout(() => this.handleClose(), 1000);
      }
    }
  }

  async onAllowNotifications(): Promise<void> {
    const granted = await this.notificationService.registerWithSNS();
    if (granted) {
      this.notificationsEnabled.set(true);
      setTimeout(() => this.handleClose(), 800);
    }
  }
}
