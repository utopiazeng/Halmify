import confetti from 'canvas-confetti'

const palettes = {
  light: ['#2563eb', '#16a34a', '#f59e0b', '#22d3ee'],
  milestone: ['#2563eb', '#16a34a', '#f59e0b', '#ec4899', '#22d3ee', '#8b5cf6'],
  win: ['#16a34a', '#f59e0b', '#f97316', '#ec4899', '#22d3ee', '#8b5cf6'],
  revenue: ['#16a34a', '#22c55e', '#facc15', '#f59e0b', '#06b6d4'],
  fulfillment: ['#0ea5e9', '#2563eb', '#22d3ee', '#16a34a', '#8b5cf6']
}

const profiles = {
  light: { particleCount: 72, spread: 78, startVelocity: 52, scalar: 0.78, extra: 28 },
  milestone: { particleCount: 120, spread: 92, startVelocity: 68, scalar: 1.02, extra: 55 },
  win: { particleCount: 165, spread: 105, startVelocity: 78, scalar: 1.18, extra: 82 },
  revenue: { particleCount: 145, spread: 96, startVelocity: 72, scalar: 1.08, extra: 70 },
  fulfillment: { particleCount: 125, spread: 90, startVelocity: 66, scalar: 1, extra: 55 }
}

export function playCelebration(type = 'milestone') {
  const profile = profiles[type] || profiles.milestone
  const colors = palettes[type] || palettes.milestone
  const bursts = [
    { delay: 0, origin: { x: 0.5, y: 1 } },
    { delay: 420, origin: { x: 0.22, y: 1 } },
    { delay: 840, origin: { x: 0.78, y: 1 } }
  ]
  bursts.forEach((burst) => {
    window.setTimeout(() => {
      confetti({
        particleCount: profile.particleCount,
        spread: profile.spread,
        startVelocity: profile.startVelocity,
        gravity: 0.82,
        ticks: 240,
        scalar: profile.scalar,
        origin: burst.origin,
        colors,
        disableForReducedMotion: true
      })
      confetti({
        particleCount: profile.extra,
        spread: 135,
        startVelocity: Math.max(42, profile.startVelocity - 24),
        gravity: 0.95,
        ticks: 220,
        scalar: Math.max(0.72, profile.scalar - 0.28),
        origin: burst.origin,
        colors,
        disableForReducedMotion: true
      })
    }, burst.delay)
  })
}
