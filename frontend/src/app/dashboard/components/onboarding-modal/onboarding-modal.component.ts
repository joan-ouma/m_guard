import {
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild,
  computed,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { BikeService } from '../../../core/services/bike.service';
import { AuthService } from '../../../core/services/auth.service';
import {
  AttachBikePayload,
  BikeDTO,
  TrackerStatusDTO,
} from '../../../core/models/bike.models';

export type OnboardingStep = 'SCAN' | 'DETAILS' | 'PREVIEW' | 'ATTACHED';

@Component({
  selector: 'app-onboarding-modal',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './onboarding-modal.component.html',
})
export class OnboardingModalComponent implements OnInit, OnDestroy {
  private readonly fb = inject(FormBuilder);
  readonly bikeService = inject(BikeService);
  readonly authService = inject(AuthService);

  readonly isOpen = input<boolean>(false);
  readonly close = output<void>();
  readonly bikeAttached = output<BikeDTO>();

  @ViewChild('scannerVideo') videoRef?: ElementRef<HTMLVideoElement>;
  @ViewChild('fileInput') fileInputRef?: ElementRef<HTMLInputElement>;

  // Wizard Step State
  readonly currentStep = signal<OnboardingStep>('SCAN');
  readonly activeInputTab = signal<'SCAN' | 'MANUAL'>('SCAN');

  // Scanner & Hardware Verification State
  readonly imeiInput = signal<string>('');
  readonly isVerifying = signal<boolean>(false);
  readonly verifyError = signal<string | null>(null);
  readonly verifiedTracker = signal<TrackerStatusDTO | null>(null);

  // Rate Limiting & Cooldown State
  readonly cooldownSeconds = signal<number>(0);
  readonly isLockedOut = signal<boolean>(false);
  private cooldownTimer?: ReturnType<typeof setInterval>;

  // Camera Scanner State
  readonly isCameraActive = signal<boolean>(false);
  readonly cameraError = signal<string | null>(null);
  private mediaStream: MediaStream | null = null;
  private scanInterval?: ReturnType<typeof setInterval>;

  // Attachment State
  readonly isSubmitting = signal<boolean>(false);
  readonly attachError = signal<string | null>(null);
  readonly attachedBikeData = signal<BikeDTO | null>(null);

  // Rider & Motorcycle Form
  readonly detailsForm: FormGroup = this.fb.group({
    plate: ['', [Validators.required, Validators.pattern(/^[A-Z]{3,4}\s?[0-9]{3}[A-Z]$/i)]],
    make: ['', [Validators.required]],
    model: ['', [Validators.required, Validators.minLength(2)]],
    year: ['', [Validators.required, Validators.min(1990), Validators.max(2030)]],
    color: ['', [Validators.required]],
    county: ['', [Validators.required]],
    subCounty: ['', [Validators.required]],
    isSelfRider: [false],
    riderName: ['', [Validators.required, Validators.minLength(2)]],
    riderNationalId: ['', [Validators.required, Validators.pattern(/^[0-9]{6,10}$/)]],
    riderPhone: ['', [Validators.required, Validators.pattern(/^(\+?[0-9\s-]{9,15})$/)]],
    riderEmail: ['', [Validators.required, Validators.email]],
  });

  // Step indicator compute
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

  constructor() {
    // Lifecycle effect when modal opens or closes
    effect(() => {
      const open = this.isOpen();
      if (open) {
        this.resetWizard();
        if (this.activeInputTab() === 'SCAN') {
          setTimeout(() => this.startCamera(), 200);
        }
      } else {
        this.stopCamera();
        this.clearCooldown();
      }
    });
  }

  ngOnInit(): void {}

  ngOnDestroy(): void {
    this.stopCamera();
    this.clearCooldown();
  }

  resetWizard(): void {
    this.currentStep.set('SCAN');
    this.activeInputTab.set('SCAN');
    this.imeiInput.set('');
    this.verifyError.set(null);
    this.verifiedTracker.set(null);
    this.attachError.set(null);
    this.attachedBikeData.set(null);
    this.isSubmitting.set(false);
    this.isVerifying.set(false);

    this.detailsForm.reset({
      plate: '',
      make: '',
      model: '',
      year: '',
      color: '',
      county: '',
      subCounty: '',
      isSelfRider: false,
      riderName: '',
      riderNationalId: '',
      riderPhone: '',
      riderEmail: '',
    });
  }

  handleClose(): void {
    this.stopCamera();
    this.close.emit();
  }

  setTab(tab: 'SCAN' | 'MANUAL'): void {
    this.activeInputTab.set(tab);
    this.verifyError.set(null);
    if (tab === 'SCAN') {
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
      this.cameraError.set('Camera is not supported on this browser. Please enter IMEI manually.');
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
      console.warn('Camera access denied or failed:', err);
      this.isCameraActive.set(false);
      this.cameraError.set('Camera permission denied or camera unavailable. Please enter IMEI manually.');
      this.activeInputTab.set('MANUAL');
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
      // Browser doesn't support native BarcodeDetector API
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

  // Handle uploaded QR image fallback
  async onFileUpload(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    if (!('BarcodeDetector' in window)) {
      this.verifyError.set('Your browser does not support image barcode detection. Please enter IMEI manually.');
      return;
    }

    try {
      const img = new Image();
      img.src = URL.createObjectURL(file);
      await img.decode();

      const barcodeDetector = new (window as any).BarcodeDetector({
        formats: ['qr_code'],
      });
      const barcodes = await barcodeDetector.detect(img);

      if (barcodes && barcodes.length > 0 && barcodes[0].rawValue) {
        this.handleScannedCode(barcodes[0].rawValue);
      } else {
        this.verifyError.set('No QR code detected in this photo. Please try another photo or enter IMEI.');
      }
    } catch (err) {
      this.verifyError.set('Could not read QR image. Please enter IMEI manually.');
    } finally {
      input.value = '';
    }
  }

  handleScannedCode(raw: string): void {
    const extracted = this.extractIMEIFromRaw(raw);
    if (!extracted) {
      this.verifyError.set(`Scanned code "${raw.slice(0, 30)}" does not contain a valid 15-digit tracker ID.`);
      return;
    }

    this.imeiInput.set(extracted);
    this.stopCamera();
    this.verifyTrackerCode(extracted);
  }

  private extractIMEIFromRaw(text: string): string | null {
    if (!text) return null;
    const trimmed = text.trim();

    // 1. Direct 15 digit IMEI
    const directMatch = trimmed.match(/\b\d{15}\b/);
    if (directMatch) return directMatch[0];

    // 2. Query param ?imei=... or &imei=...
    const urlMatch = trimmed.match(/[?&]imei=([0-9]{15})/i);
    if (urlMatch) return urlMatch[1];

    // 3. JSON payload e.g. {"imei":"860000000000001"}
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed.imei && /^\d{15}$/.test(parsed.imei)) {
        return parsed.imei;
      }
      if (parsed.id && /^\d{15}$/.test(parsed.id)) {
        return parsed.id;
      }
    } catch {}

    return null;
  }

  // --- Tracker Verification & Rate Limit Management ---

  verifyTrackerCode(imei?: string): void {
    const code = (imei || this.imeiInput()).trim();
    if (!code) {
      this.verifyError.set('Please enter a 15-digit tracker IMEI code.');
      return;
    }

    if (!/^\d{15}$/.test(code)) {
      this.verifyError.set('Invalid tracker code. IMEI must be exactly 15 numeric digits.');
      return;
    }

    if (this.cooldownSeconds() > 0) {
      this.verifyError.set(`Rate limited. Please wait ${this.cooldownSeconds()}s before next attempt.`);
      return;
    }

    if (this.isLockedOut()) {
      this.verifyError.set('Your IP address is locked out due to excessive failed attempts. Please contact support.');
      return;
    }

    this.isVerifying.set(true);
    this.verifyError.set(null);

    this.bikeService.verifyTracker(code).subscribe({
      next: (res) => {
        this.isVerifying.set(false);
        this.verifiedTracker.set(res.data);

        // Pre-fill default model in details form if available
        if (res.data?.model && !this.detailsForm.get('model')?.value) {
          this.detailsForm.patchValue({ model: 'Ace 125' });
        }

        // Advance to step 2 after brief success indicator
        setTimeout(() => {
          this.currentStep.set('DETAILS');
        }, 500);
      },
      error: (err: HttpErrorResponse) => {
        this.isVerifying.set(false);
        this.handleVerifyError(err);
      },
    });
  }

  private handleVerifyError(err: HttpErrorResponse): void {
    const status = err.status;
    const retryAfterHeader = err.headers?.get('Retry-After');
    const isLockedHeader = err.headers?.get('X-RateLimit-Locked') === 'true';

    // 1. Rate Limit 429
    if (status === 429) {
      const waitSec = retryAfterHeader ? parseInt(retryAfterHeader, 10) : 60;
      this.startCooldown(isNaN(waitSec) || waitSec <= 0 ? 60 : waitSec);
      if (isLockedHeader || err.error?.detail?.toLowerCase().includes('locked out')) {
        this.isLockedOut.set(true);
        this.verifyError.set('Access Blocked: IP locked out due to repeated abuse. Contact support.');
      } else {
        this.verifyError.set(err.error?.detail || `Rate limit exceeded. Please wait ${this.cooldownSeconds()}s before trying again.`);
      }
      return;
    }

    // 2. Already Claimed 409
    if (status === 409) {
      this.verifyError.set(err.error?.detail || 'This tracker is already claimed by another motorcycle.');
      return;
    }

    // 3. Not Found 404
    if (status === 404) {
      this.verifyError.set(err.error?.detail || 'Tracker hardware does not exist in MotoGuard database.');
      return;
    }

    // Generic error
    this.verifyError.set(err.error?.detail || 'Failed to verify tracker hardware. Please try again.');
  }

  private startCooldown(seconds: number): void {
    this.clearCooldown();
    this.cooldownSeconds.set(seconds);

    this.cooldownTimer = setInterval(() => {
      this.cooldownSeconds.update((curr) => {
        if (curr <= 1) {
          this.clearCooldown();
          return 0;
        }
        return curr - 1;
      });
    }, 1000);
  }

  private clearCooldown(): void {
    if (this.cooldownTimer) {
      clearInterval(this.cooldownTimer);
      this.cooldownTimer = undefined;
    }
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
    this.detailsForm.patchValue({ isSelfRider: checked });

    if (checked) {
      const user = this.authService.currentUser();
      if (user) {
        this.detailsForm.patchValue({
          riderName: user.name || '',
          riderNationalId: user.national_id || '',
          riderPhone: user.phone_number || '',
          riderEmail: user.email || '',
        });
      }
    }
  }

  goToPreview(): void {
    if (this.detailsForm.invalid) {
      this.detailsForm.markAllAsTouched();
      return;
    }
    this.attachError.set(null);
    this.currentStep.set('PREVIEW');
  }

  backToStep(step: OnboardingStep): void {
    this.currentStep.set(step);
    if (step === 'SCAN' && this.activeInputTab() === 'SCAN') {
      setTimeout(() => this.startCamera(), 100);
    }
  }

  // --- Step 3 & 4: Attachment & Email Dispatch ---

  confirmAndAttach(): void {
    if (this.detailsForm.invalid || this.isSubmitting()) return;
    const tracker = this.verifiedTracker();
    if (!tracker) {
      this.currentStep.set('SCAN');
      return;
    }

    this.isSubmitting.set(true);
    this.attachError.set(null);

    const val = this.detailsForm.value;
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
      next: (res) => {
        this.isSubmitting.set(false);
        this.attachedBikeData.set(res.data);
        this.currentStep.set('ATTACHED');
      },
      error: (err: HttpErrorResponse) => {
        this.isSubmitting.set(false);
        this.attachError.set(err.error?.detail || 'Failed to attach motorcycle to tracker. Please check details.');
      },
    });
  }

  onFinishTracking(): void {
    const bike = this.attachedBikeData();
    if (bike) {
      this.bikeAttached.emit(bike);
    }
    this.handleClose();
  }
}
