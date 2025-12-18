import { Capacitor } from '@capacitor/core'
import { PushNotifications } from '@capacitor/push-notifications'

export async function requestNotificationPermission(): Promise<boolean> {
  if (!Capacitor.isNativePlatform()) return false

  const result = await PushNotifications.requestPermissions()
  if (result.receive === 'granted') {
    await PushNotifications.register()
    return true
  }
  return false
}

export function onNotificationReceived(callback: (notification: unknown) => void): void {
  if (!Capacitor.isNativePlatform()) return
  PushNotifications.addListener('pushNotificationReceived', callback)
}
