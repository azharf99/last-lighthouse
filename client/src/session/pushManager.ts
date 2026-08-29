/**
 * Push Manager Module for Web Push & FCM (ADR-007)
 */

function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  const buffer = new ArrayBuffer(rawData.length);
  const outputArray = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export function isPushSupported(): boolean {
  return typeof window !== 'undefined' && 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!isPushSupported()) return null;
  try {
    const reg = await navigator.serviceWorker.register('/sw.js', { scope: '/' });
    return reg;
  } catch (err) {
    console.warn('[PushManager] Failed to register service worker:', err);
    return null;
  }
}

export async function getPushPermissionState(): Promise<NotificationPermission> {
  if (!isPushSupported()) return 'denied';
  return Notification.permission;
}

export async function subscribeToPush(
  playerId: string,
  authToken: string,
  serverUrl = ''
): Promise<{ success: boolean; message: string }> {
  if (!isPushSupported()) {
    return { success: false, message: 'Browser tidak mendukung Web Push notification.' };
  }

  try {
    const permission = await Notification.requestPermission();
    if (permission !== 'granted') {
      return { success: false, message: 'Izin notifikasi ditolak oleh pengguna.' };
    }

    const registration = await navigator.serviceWorker.ready;

    // 1. Fetch VAPID public key
    const vapidRes = await fetch(`${serverUrl}/api/push/vapid-key`);
    if (!vapidRes.ok) {
      throw new Error('Gagal mengambil VAPID public key dari server');
    }
    const { publicKey } = await vapidRes.json();

    // 2. Subscribe via browser pushManager
    const applicationServerKey = urlBase64ToUint8Array(publicKey);
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey,
    });

    const subJson = subscription.toJSON();
    const p256dh = subJson.keys?.p256dh || '';
    const auth = subJson.keys?.auth || '';

    // 3. Send subscription to server
    const subRes = await fetch(`${serverUrl}/api/push/subscribe`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: authToken ? `Bearer ${authToken}` : '',
      },
      body: JSON.stringify({
        playerId,
        endpoint: subscription.endpoint,
        p256dh,
        auth,
        platform: 'web',
      }),
    });

    if (!subRes.ok) {
      throw new Error('Gagal mendaftarkan endpoint push ke server');
    }

    return { success: true, message: 'Notifikasi giliran berhasil diaktifkan!' };
  } catch (err: any) {
    console.error('[PushManager] Subscribe error:', err);
    return { success: false, message: err?.message || 'Terjadi kesalahan saat mengaktifkan notifikasi.' };
  }
}
