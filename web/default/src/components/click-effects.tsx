import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

const CLICK_EFFECTS = ['ring', 'stars', 'heart', 'paw', 'confetti'] as const
const EFFECT_LIFETIME_MS = 1100
const CONFETTI_COLORS = ['#f472b6', '#8b7cf6', '#60a5fa', '#fbbf24', '#34d399']

type ClickEffect = (typeof CLICK_EFFECTS)[number]

function appendParticle(
  layer: HTMLDivElement,
  effect: ClickEffect,
  x: number,
  y: number,
  content?: string,
  variables?: Record<string, string>
) {
  const particle = document.createElement('i')
  particle.className = `click-effect click-effect--${effect}`
  particle.style.left = `${x}px`
  particle.style.top = `${y}px`
  particle.textContent = content ?? ''

  for (const [name, value] of Object.entries(variables ?? {})) {
    particle.style.setProperty(name, value)
  }

  const removeParticle = () => particle.remove()
  particle.addEventListener('animationend', removeParticle, { once: true })
  layer.appendChild(particle)
  window.setTimeout(removeParticle, EFFECT_LIFETIME_MS)
}

function createEffect(
  layer: HTMLDivElement,
  effect: ClickEffect,
  x: number,
  y: number
) {
  if (effect === 'ring') {
    appendParticle(layer, effect, x, y)
    return
  }

  if (effect === 'stars') {
    for (let index = 0; index < 5; index += 1) {
      const angle = (Math.PI * 2 * index) / 5 + Math.random()
      const distance = 26 + Math.random() * 26
      appendParticle(layer, effect, x, y, '✦', {
        '--click-effect-x': `${Math.cos(angle) * distance}px`,
        '--click-effect-y': `${Math.sin(angle) * distance}px`,
        'font-size': `${9 + Math.random() * 6}px`,
      })
    }
    return
  }

  if (effect === 'heart') {
    appendParticle(layer, effect, x, y, '💗')
    return
  }

  if (effect === 'paw') {
    appendParticle(layer, effect, x, y, '🐾', {
      '--click-effect-rotation': `${Math.random() * 50 - 25}deg`,
    })
    return
  }

  for (let index = 0; index < 6; index += 1) {
    const angle = Math.PI * 2 * Math.random()
    const distance = 30 + Math.random() * 34
    appendParticle(layer, effect, x, y, undefined, {
      '--click-effect-x': `${Math.cos(angle) * distance}px`,
      '--click-effect-y': `${Math.sin(angle) * distance + 30}px`,
      '--click-effect-rotation': `${Math.random() * 260 - 130}deg`,
      'background-color': CONFETTI_COLORS[index % CONFETTI_COLORS.length],
    })
  }
}

export function ClickEffects() {
  const layerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const layer = layerRef.current
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')

    if (!layer) return

    const handlePointerDown = (event: PointerEvent) => {
      if (event.button !== 0 || reducedMotion.matches) return

      const effect =
        CLICK_EFFECTS[Math.floor(Math.random() * CLICK_EFFECTS.length)]
      createEffect(layer, effect, event.clientX, event.clientY)
    }

    document.addEventListener('pointerdown', handlePointerDown, {
      passive: true,
    })

    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      layer.replaceChildren()
    }
  }, [])

  return createPortal(
    <div aria-hidden='true' className='click-effects-layer' ref={layerRef} />,
    document.body
  )
}
