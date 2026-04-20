const DEFAULT_ADVANCE_EVERY_MS = 3000
const DEFAULT_ANIMATION_MS = 850
const DEFAULT_VISIBLE_RADIUS = 3

class TextCarousel extends HTMLElement {
  connectedCallback() {
    if (this.initialized) {
      return
    }

    this.initialized = true
    this.reducedMotionQuery = window.matchMedia("(prefers-reduced-motion: reduce)")
    this.items = this.readItems()
    if (this.items.length === 0) {
      return
    }

    this.advanceEveryMs = this.readNumberAttribute(
      "advance-every",
      DEFAULT_ADVANCE_EVERY_MS,
    )
    this.animationMs = this.readNumberAttribute(
      "animation-duration",
      DEFAULT_ANIMATION_MS,
    )
    this.visibleRadius = Math.max(
      1,
      this.readNumberAttribute("visible-radius", DEFAULT_VISIBLE_RADIUS),
    )
    this.poolRadius = this.visibleRadius + 1
    this.activeIndex = 0
    this.animationFrame = 0
    this.animationStart = 0
    this.timeout = 0

    this.render()
    this.setAttribute("data-enhanced", "true")
    this.renderFrame(0)

    this.handleVisibilityChange = () => {
      if (document.hidden) {
        this.stop()
        return
      }
      this.start()
    }
    this.handleReducedMotionChange = () => {
      this.stop()
      this.renderFrame(0)
      this.start()
    }

    document.addEventListener("visibilitychange", this.handleVisibilityChange)
    this.reducedMotionQuery.addEventListener("change", this.handleReducedMotionChange)

    this.start()
  }

  disconnectedCallback() {
    this.stop()
    if (this.handleVisibilityChange) {
      document.removeEventListener("visibilitychange", this.handleVisibilityChange)
    }
    if (this.handleReducedMotionChange) {
      this.reducedMotionQuery?.removeEventListener(
        "change",
        this.handleReducedMotionChange,
      )
    }
  }

  readItems() {
    const sourceItems = this.querySelectorAll(".textcarousel-source li")
    return Array.from(sourceItems)
      .map((item) => item.textContent?.trim() ?? "")
      .filter(Boolean)
  }

  readNumberAttribute(name, fallback) {
    const value = Number.parseInt(this.getAttribute(name) ?? "", 10)
    if (Number.isNaN(value)) {
      return fallback
    }
    return value
  }

  render() {
    this.viewport = document.createElement("div")
    this.viewport.className = "textcarousel-viewport"

    this.track = document.createElement("div")
    this.track.className = "textcarousel-track"
    this.viewport.append(this.track)

    this.renderedItems = []

    for (let offset = -this.poolRadius; offset <= this.poolRadius; offset++) {
      const item = document.createElement("div")
      item.className = "textcarousel-item"
      item.setAttribute("aria-hidden", "true")
      this.track.append(item)
      this.renderedItems.push(item)
    }

    this.append(this.viewport)
  }

  start() {
    if (document.hidden || this.reducedMotionQuery.matches || this.items.length < 2) {
      return
    }

    this.stop()
    this.timeout = window.setTimeout(() => {
      this.animateToNext()
    }, this.advanceEveryMs)
  }

  stop() {
    if (this.timeout) {
      window.clearTimeout(this.timeout)
      this.timeout = 0
    }
    if (this.animationFrame) {
      window.cancelAnimationFrame(this.animationFrame)
      this.animationFrame = 0
    }
  }

  animateToNext() {
    this.stop()
    this.animationStart = performance.now()

    const tick = (now) => {
      const progress = clamp((now - this.animationStart) / this.animationMs, 0, 1)
      this.renderFrame(easeInOutCubic(progress))

      if (progress < 1) {
        this.animationFrame = window.requestAnimationFrame(tick)
        return
      }

      this.animationFrame = 0
      this.activeIndex = modulo(this.activeIndex + 1, this.items.length)
      this.renderFrame(0)
      this.start()
    }

    this.animationFrame = window.requestAnimationFrame(tick)
  }

  renderFrame(progress) {
    const emphasisDistance = this.visibleRadius + 0.05
    const opacityDistance = this.visibleRadius + 0.45

    for (const [index, item] of this.renderedItems.entries()) {
      const slotOffset = index - this.poolRadius
      const itemIndex = modulo(this.activeIndex + slotOffset, this.items.length)
      const visualOffset = slotOffset - progress
      const compressedOffset = compressOffset(visualOffset)
      const distance = Math.abs(visualOffset)
      const clampedDistance = Math.min(distance, this.poolRadius)
      const emphasis = Math.max(0, 1 - clampedDistance / emphasisDistance)
      const scale = 0.42 + Math.pow(emphasis, 1.45) * 0.9
      const opacity = Math.max(0, 1 - Math.pow(clampedDistance / opacityDistance, 2.35))
      const blur = Math.max(0, (clampedDistance - 0.18) * 1.4)
      const zIndex = Math.round((this.poolRadius + 2 - clampedDistance) * 10)

      item.textContent = this.items[itemIndex]
      item.style.setProperty("--textcarousel-offset", compressedOffset.toFixed(4))
      item.style.setProperty("--textcarousel-scale", scale.toFixed(4))
      item.style.setProperty("--textcarousel-opacity", opacity.toFixed(4))
      item.style.setProperty("--textcarousel-blur", `${blur.toFixed(3)}px`)
      item.style.zIndex = String(zIndex)
      item.toggleAttribute("data-active", distance < 0.5)
    }
  }
}

if (!customElements.get("text-carousel")) {
  customElements.define("text-carousel", TextCarousel)
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value))
}

function modulo(value, length) {
  return ((value % length) + length) % length
}

function easeInOutCubic(value) {
  if (value < 0.5) {
    return 4 * value * value * value
  }
  return 1 - Math.pow(-2 * value + 2, 3) / 2
}

function compressOffset(offset) {
  const direction = Math.sign(offset)
  const distance = Math.abs(offset)
  if (distance <= 1.5) {
    return offset
  }
  return direction * (1.5 + (distance - 1.5) * 0.72)
}
