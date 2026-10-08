const brandName = (import.meta.env.VITE_BRAND_NAME || 'Halmify CRM').trim()
const brandTagline = (import.meta.env.VITE_BRAND_TAGLINE || 'AI-native revenue CRM').trim()

export const BRAND_NAME = brandName || 'Halmify CRM'
export const BRAND_TAGLINE = brandTagline || 'AI-native revenue CRM'
