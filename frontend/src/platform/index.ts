import { Capacitor } from '@capacitor/core'

export const platform = {
  isNative: Capacitor.isNativePlatform(),
  isWeb: !Capacitor.isNativePlatform(),
  isTauri: typeof window !== 'undefined' && '__TAURI__' in window,
}
