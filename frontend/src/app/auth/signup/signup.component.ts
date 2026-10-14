import {
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild,
  computed,
  inject,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  AbstractControl,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  ValidationErrors,
  Validators,
} from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { AuthService } from '../../core/services/auth.service';
import { BikeService } from '../../core/services/bike.service';
import {
  AttachBikePayload,
  BikeDTO,
  TrackerStatusDTO,
} from '../../core/models/bike.models';

export type OnboardingStep = 'SCAN' | 'DETAILS' | 'PREVIEW' | 'ATTACHED';

export function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password');
  const confirmPassword = control.get('confirmPassword');

  if (password && confirmPassword && password.value !== confirmPassword.value) {
    confirmPassword.setErrors({ ...confirmPassword.errors, passwordMismatch: true });
    return { passwordMismatch: true };
  } else {
    if (confirmPassword?.hasError('passwordMismatch')) {
      const errors = { ...confirmPassword.errors };
      delete errors['passwordMismatch'];
      confirmPassword.setErrors(Object.keys(errors).length ? errors : null);
    }
  }
  return null;
}

@Component({
  selector: 'app-signup',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink],
  templateUrl: './signup.component.html',
})
export class SignupComponent implements OnInit, OnDestroy {
  private readonly fb = inject(FormBuilder);
  readonly authService = inject(AuthService);
  readonly bikeService = inject(BikeService);
  private readonly router = inject(Router);

  @ViewChild('scannerVideo') videoRef?: ElementRef<HTMLVideoElement>;

  // Wizard Step State
  readonly currentStep = signal<OnboardingStep>('SCAN');
  readonly activeInputTab = signal<'CAMERA' | 'MANUAL'>('CAMERA');

  // Computed Title & Subtitle matching the reference layout
  readonly stepTitle = computed<string>(() => {
    switch (this.currentStep()) {
      case 'SCAN':
        return 'Tracker Hardware';
      case 'DETAILS':
        return 'Account & Motorcycle';
      case 'PREVIEW':
        return 'Details Confirmation';
      case 'ATTACHED':
        return 'Tracker Activated';
    }
  });

  readonly stepSubtitle = computed<string>(() => {
    switch (this.currentStep()) {
      case 'SCAN':
        return 'Scan the QR code on your hardware unit or enter its 15-digit IMEI.';
      case 'DETAILS':
        return 'Fill in your credentials, motorcycle, and assigned rider details.';
      case 'PREVIEW':
        return 'Please review your details before sending for hardware activation.';
      case 'ATTACHED':
        return 'Your tracker has been verified and telemetry is active.';
    }
  });

  // Scanner & Hardware Verification State
  readonly imeiInput = signal<string>('');
  readonly isVerifying = signal<boolean>(false);
  readonly verifyError = signal<string | null>(null);
  readonly verifiedTracker = signal<TrackerStatusDTO | null>(null);

  // Rate Limiting & Cooldown State
  readonly cooldownSeconds = signal<number>(0);
  readonly isLockedOut = signal<boolean>(false);
  readonly attemptsUsed = signal<number>(0);
  readonly maxAttempts = signal<number>(5);
  private cooldownTimer?: ReturnType<typeof setInterval>;

  // Camera Scanner State
  readonly isCameraActive = signal<boolean>(false);
  readonly cameraError = signal<string | null>(null);
  private mediaStream: MediaStream | null = null;
  private scanInterval?: ReturnType<typeof setInterval>;

  // Form Controls Visibility
  readonly showPassword = signal<boolean>(false);
  readonly showConfirmPassword = signal<boolean>(false);

  // Submission & Attachment State
  readonly isSubmitting = signal<boolean>(false);
  readonly submissionStep = signal<string>('');
  readonly errorMessage = signal<string | null>(null);
  readonly attachedBikeData = signal<BikeDTO | null>(null);

  // Combined Form: User Account + Motorcycle + Rider
  readonly form: FormGroup = this.fb.group(
    {
      // Account Details
      name: ['', [Validators.required, Validators.minLength(2)]],
      email: ['', [Validators.required, Validators.email]],
      phoneNumber: ['', [Validators.required, Validators.pattern(/^(\+?[0-9\s-]{9,15})$/)]],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', [Validators.required]],

      // Motorcycle Details
      plate: ['', [Validators.required, Validators.pattern(/^[A-Z]{3,4}\s?[0-9]{3}[A-Z]$/i)]],
      make: ['', [Validators.required]],
      model: ['', [Validators.required, Validators.minLength(2)]],
      year: [
        '',
        [Validators.required, Validators.min(1990), Validators.max(2030)],
      ],
      color: ['', [Validators.required]],
      county: ['', [Validators.required]],
      subCounty: ['', [Validators.required]],

      // Rider Details
      isSelfRider: [false],
      riderName: ['', [Validators.required, Validators.minLength(2)]],
      riderNationalId: ['', [Validators.required, Validators.pattern(/^[0-9]{6,10}$/)]],
      riderPhone: ['', [Validators.required, Validators.pattern(/^(\+?[0-9\s-]{9,15})$/)]],
      riderEmail: ['', [Validators.required, Validators.email]],
    },
    { validators: passwordMatchValidator }
  );

  readonly stepIndex = computed<number>(() => {
    switch (this.currentStep()) {
      case 'SCAN':
        return 0;
      case 'DETAILS':
        return 1;
      case 'PREVIEW':
        return 2;
      case 'ATTACHED':
        return 3;
    }
  });

  ngOnInit(): void {
    if (this.currentStep() === 'SCAN' && this.activeInputTab() === 'CAMERA') {
      setTimeout(() => this.startCamera(), 100);
    }
  }

  ngOnDestroy(): void {
    this.stopCamera();
    this.clearCooldown();
  }

  togglePasswordVisibility(): void {
    this.showPassword.update((prev) => !prev);
  }

  toggleConfirmPasswordVisibility(): void {
    this.showConfirmPassword.update((prev) => !prev);
  }

  setTab(tab: 'CAMERA' | 'MANUAL'): void {
    this.activeInputTab.set(tab);
    this.verifyError.set(null);
    if (tab === 'CAMERA') {
      setTimeout(() => this.startCamera(), 100);
    } else {
      this.stopCamera();
    }
  }

  onImeiInputChange(event: Event): void {
    const raw = (event.target as HTMLInputElement).value;
    const clean = raw.replace(/\D/g, '').slice(0, 15);
    this.imeiInput.set(clean);
  }

  // --- Camera QR Scanning ---

  async startCamera(): Promise<void> {
    if (typeof window === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
      this.cameraError.set('Camera is not supported on this device. Please enter IMEI manually.');
      this.activeInputTab.set('MANUAL');
      return;
    }

    try {
      this.stopCamera();
      this.cameraError.set(null);

      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: { ideal: 'environment' } },
        audio: false,
      });

      this.mediaStream = stream;
      this.isCameraActive.set(true);

      if (this.videoRef?.nativeElement) {
        this.videoRef.nativeElement.srcObject = stream;
        await this.videoRef.nativeElement.play();
      }

      this.initBarcodeDetector();
    } catch (err: any) {
      this.isCameraActive.set(false);
      this.cameraError.set(
        'Camera permission was denied or camera is unavailable.'
      );
    }
  }

  stopCamera(): void {
    if (this.scanInterval) {
      clearInterval(this.scanInterval);
      this.scanInterval = undefined;
    }
    if (this.mediaStream) {
      this.mediaStream.getTracks().forEach((track) => track.stop());
      this.mediaStream = null;
    }
    if (this.videoRef?.nativeElement) {
      this.videoRef.nativeElement.srcObject = null;
    }
    this.isCameraActive.set(false);
  }

  private initBarcodeDetector(): void {
    if (typeof window === 'undefined') return;

    if (!('BarcodeDetector' in window)) {
      return;
    }

    try {
      const barcodeDetector = new (window as any).BarcodeDetector({
        formats: ['qr_code'],
      });

      this.scanInterval = setInterval(async () => {
        const video = this.videoRef?.nativeElement;
        if (!video || video.readyState < 2 || this.isVerifying()) return;

        try {
          const barcodes = await barcodeDetector.detect(video);
          if (barcodes && barcodes.length > 0) {
            const raw = barcodes[0].rawValue;
            if (raw) {
              this.handleScannedCode(raw);
            }
          }
        } catch {}
      }, 300);
    } catch (err) {
      console.warn('BarcodeDetector initialization skipped:', err);
    }
  }

  handleScannedCode(raw: string): void {
    if (this.cooldownSeconds() > 0 || this.isLockedOut() || this.isVerifying()) return;

    const extracted = this.extractIMEIFromRaw(raw);
    if (!extracted) {
      this.verifyError.set(
        `Scanned code does not contain a valid 15-digit tracker ID.`
      );
      return;
    }

    this.imeiInput.set(extracted);
    this.stopCamera();
    this.verifyTrackerCode(extracted);
  }

  private extractIMEIFromRaw(raw: string): string | null {
    if (!raw) return null;
    const clean = raw.trim();

    try {
      const parsed = JSON.parse(clean);
      if (parsed.imei && /^\d{15}$/.test(parsed.imei)) return parsed.imei;
      if (parsed.id && /^\d{15}$/.test(parsed.id)) return parsed.id;
      if (parsed.tracker_id && /^\d{15}$/.test(parsed.tracker_id)) return parsed.tracker_id;
    } catch {}

    const digitsOnly = clean.replace(/\D/g, '');
    if (digitsOnly.length === 15) return digitsOnly;

    const match = clean.match(/\b\d{15}\b/);
    if (match) return match[0];

    return null;
  }

  verifyTrackerCode(codeToVerify?: string): void {
    const imei = (codeToVerify || this.imeiInput()).trim();
    // Strictly refuse to make API calls if in cooldown, locked out, or already verifying
    if (this.cooldownSeconds() > 0 || this.isLockedOut() || this.isVerifying()) {
      return;
    }

    if (!/^\d{15}$/.test(imei)) {
      this.verifyError.set('Please enter a valid 15-digit IMEI.');
      return;
    }

    this.isVerifying.set(true);
    this.verifyError.set(null);

    this.bikeService.verifyTracker(imei).subscribe({
      next: (res) => {
        this.isVerifying.set(false);
        const tracker = res.data;

        if (tracker.status === 'CLAIMED' || tracker.is_claimed) {
          this.verifyError.set(
            `Tracker #${tracker.imei} is already registered.`
          );
          return;
        }

        // When tracker is found, clear errors & cooldown and transition to DETAILS
        this.verifiedTracker.set(tracker);
        this.stopCamera();
        this.verifyError.set(null);
        this.clearCooldown();

        this.currentStep.set('DETAILS');
      },
      error: (err: HttpErrorResponse) => {
        this.isVerifying.set(false);
        this.attemptsUsed.update((prev) => prev + 1);

        this.handleRateLimitHeaders(err);

        // Calculate exponential cooldown if not already started:
        // Attempt 1: 60s, Attempt 2: 120s, Attempt 3: 240s, Attempt 4: 480s
        if (this.cooldownSeconds() <= 0 && !this.isLockedOut()) {
          const expSeconds = 60 * Math.pow(2, Math.max(0, this.attemptsUsed() - 1));
          this.startCooldown(expSeconds);
        }

        if (err.status === 404) {
          this.verifyError.set(`Tracker with IMEI "${imei}" not found.`);
        } else {
          this.verifyError.set(err.error?.detail || 'Unable to verify tracker. Please check IMEI.');
        }
      },
    });
  }

  private handleRateLimitHeaders(err: HttpErrorResponse): void {
    const remainingHeader = err.headers.get('X-RateLimit-Remaining');
    const limitHeader = err.headers.get('X-RateLimit-Limit');

    if (limitHeader) {
      const parsedLimit = parseInt(limitHeader, 10);
      if (!isNaN(parsedLimit) && parsedLimit > 0) {
        this.maxAttempts.set(parsedLimit);
      }
    }

    if (remainingHeader) {
      const parsedRemaining = parseInt(remainingHeader, 10);
      if (!isNaN(parsedRemaining) && parsedRemaining >= 0) {
        this.attemptsUsed.set(Math.max(0, this.maxAttempts() - parsedRemaining));
      }
    }

    const lockedHeader = err.headers.get('X-RateLimit-Locked');
    if (lockedHeader === 'true' || this.attemptsUsed() >= this.maxAttempts()) {
      this.isLockedOut.set(true);
      return;
    }

    const retryAfter = err.headers.get('Retry-After');
    if (retryAfter) {
      const seconds = parseInt(retryAfter, 10);
      if (!isNaN(seconds) && seconds > 0) {
        this.startCooldown(seconds);
      }
    }
  }

  private startCooldown(seconds: number): void {
    this.clearCooldown();
    this.cooldownSeconds.set(seconds);

    this.cooldownTimer = setInterval(() => {
      const current = this.cooldownSeconds() - 1;
      if (current <= 0) {
        this.clearCooldown();
      } else {
        this.cooldownSeconds.set(current);
      }
    }, 1000);
  }

  private clearCooldown(): void {
    if (this.cooldownTimer) {
      clearInterval(this.cooldownTimer);
      this.cooldownTimer = undefined;
    }
    this.cooldownSeconds.set(0);
  }

  formatCooldown(): string {
    const s = this.cooldownSeconds();
    const mins = Math.floor(s / 60);
    const secs = s % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }

  // --- Step 2: Form & Rider Handling ---

  onToggleSelfRider(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.form.patchValue({ isSelfRider: checked });

    if (checked) {
      const val = this.form.value;
      this.form.patchValue({
        riderName: val.name || '',
        riderPhone: val.phoneNumber || '',
        riderEmail: val.email || '',
      });
    }
  }

  onAccountFieldChange(): void {
    if (this.form.get('isSelfRider')?.value) {
      const val = this.form.value;
      this.form.patchValue({
        riderName: val.name || '',
        riderPhone: val.phoneNumber || '',
        riderEmail: val.email || '',
      });
    }
  }

  isFieldInvalid(field: string): boolean {
    const control = this.form.get(field);
    if (!control) return false;
    return control.invalid && control.touched;
  }

  goToPreview(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      this.errorMessage.set('Please fill out all required fields before proceeding to preview.');
      return;
    }
    this.errorMessage.set(null);
    this.currentStep.set('PREVIEW');
  }

  backToStep(step: OnboardingStep): void {
    this.currentStep.set(step);
    if (step === 'SCAN' && this.activeInputTab() === 'CAMERA') {
      setTimeout(() => this.startCamera(), 100);
    } else {
      this.stopCamera();
    }
  }

  // --- Step 3 & 4: Account Creation + Attachment + Activation ---

  confirmAndComplete(): void {
    if (this.form.invalid || this.isSubmitting()) return;
    const tracker = this.verifiedTracker();
    if (!tracker) {
      this.currentStep.set('SCAN');
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);
    this.submissionStep.set('Creating owner account...');

    const val = this.form.value;

    // 1. Register User Account
    this.authService
      .register({
        name: val.name.trim(),
        email: val.email.trim(),
        phone_number: val.phoneNumber.trim(),
        password: val.password,
      })
      .subscribe({
        next: () => {
          this.submissionStep.set('Authenticating session...');

          // 2. Log in User to acquire JWT session cookies
          this.authService
            .login({
              email: val.email.trim(),
              password: val.password,
            })
            .subscribe({
              next: () => {
                this.submissionStep.set('Attaching tracker & configuring motorcycle...');

                // 3. Attach Motorcycle to Verified Tracker & Dispatch Rider Email
                const payload: AttachBikePayload = {
                  registration_number: val.plate.toUpperCase().trim(),
                  make: val.make.trim(),
                  model: val.model.trim(),
                  year: parseInt(val.year, 10) || new Date().getFullYear(),
                  color: val.color.trim(),
                  county: val.county.trim(),
                  sub_county: val.subCounty.trim(),
                  imei: tracker.imei,
                  rider_name: val.riderName.trim(),
                  rider_national_id: val.riderNationalId.trim(),
                  rider_phone: val.riderPhone.trim(),
                  rider_email: val.riderEmail.trim(),
                };

                this.bikeService.attachBike(payload).subscribe({
                  next: (bikeRes) => {
                    this.isSubmitting.set(false);
                    this.attachedBikeData.set(bikeRes.data);
                    this.currentStep.set('ATTACHED');
                  },
                  error: (err: HttpErrorResponse) => {
                    this.isSubmitting.set(false);
                    this.errorMessage.set(
                      err.error?.detail ||
                        'Account was created, but tracker pairing encountered an error. Please contact support.'
                    );
                  },
                });
              },
              error: (loginErr: HttpErrorResponse) => {
                this.isSubmitting.set(false);
                this.errorMessage.set(
                  loginErr.error?.detail ||
                    'Account created. Please log in with your credentials to finish tracker pairing.'
                );
                this.router.navigate(['/login']);
              },
            });
        },
        error: (regErr: HttpErrorResponse) => {
          this.isSubmitting.set(false);
          if (regErr.status === 409) {
            this.errorMessage.set('An account with this email address already exists. Please sign in instead.');
          } else {
            this.errorMessage.set(
              regErr.error?.detail ||
                regErr.error?.message ||
                'Unable to create account. Please check your information and try again.'
            );
          }
        },
      });
  }

  navigateToDashboard(): void {
    this.router.navigate(['/']);
  }
}
