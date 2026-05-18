import { mergePaths } from "./vendor/datastar.js"

const activeAnchorSignal = "_docs.activeAnchor"
const scrollspyAbort = Symbol("docsScrollspyAbort")

function hashID(hash) {
  return decodeURIComponent(hash.slice(1))
}

function linkTarget(link) {
  const url = new URL(link.href)
  if (url.pathname !== window.location.pathname || !url.hash) {
    return null
  }
  return document.getElementById(hashID(url.hash))
}

function sidebarLinks(sidebar) {
  const seenTargets = new Set()
  const links = []
  const targets = []

  for (const link of sidebar.querySelectorAll("a[href]")) {
    const target = linkTarget(link)
    if (!target) {
      continue
    }

    links.push(link)
    if (!seenTargets.has(target.id)) {
      seenTargets.add(target.id)
      targets.push(target)
    }
  }

  return { links, targets }
}

function currentTargetID(targets) {
  if (
    window.innerHeight + window.scrollY >=
    document.documentElement.scrollHeight - 2
  ) {
    return targets.at(-1).id
  }

  const activationLine = Math.min(window.innerHeight * 0.35, 220)
  return (
    targets.findLast(
      (target) => target.getBoundingClientRect().top <= activationLine,
    )?.id || targets[0].id
  )
}

function openParentDetails(link) {
  for (let parent = link.parentElement; parent; parent = parent.parentElement) {
    if (parent instanceof HTMLDetailsElement) {
      parent.open = true
    }
    if (
      parent instanceof HTMLUListElement &&
      parent.classList.contains("menu-dropdown")
    ) {
      parent.classList.add("menu-dropdown-show")

      const toggle = parent.previousElementSibling?.querySelector("button")
      toggle?.setAttribute("aria-expanded", "true")
      toggle?.querySelector("svg")?.classList.add("rotate-90")
    }
  }
}

function activate(links, targetID) {
  mergePaths([[activeAnchorSignal, targetID]])

  const activeLinks = links.filter((link) => hashID(link.hash) === targetID)
  for (const link of activeLinks) {
    openParentDetails(link)
  }

  activeLinks
    .find((link) => link.offsetParent !== null)
    ?.scrollIntoView({ block: "nearest", inline: "nearest" })
}

function initDocsSidebarScrollspy(sidebar) {
  sidebar[scrollspyAbort]?.abort()

  const { links, targets } = sidebarLinks(sidebar)
  if (targets.length === 0) {
    return
  }

  const abortController = new AbortController()
  sidebar[scrollspyAbort] = abortController

  let activeTargetID = ""
  let updateFrame = 0

  const update = () => {
    updateFrame = 0

    const targetID = currentTargetID(targets)
    if (targetID === activeTargetID) {
      return
    }

    activeTargetID = targetID
    activate(links, targetID)
  }

  const scheduleUpdate = () => {
    if (updateFrame !== 0) {
      return
    }

    updateFrame = window.requestAnimationFrame(update)
  }

  window.addEventListener("scroll", scheduleUpdate, {
    passive: true,
    signal: abortController.signal,
  })
  window.addEventListener("resize", scheduleUpdate, {
    signal: abortController.signal,
  })
  window.addEventListener("hashchange", scheduleUpdate, {
    signal: abortController.signal,
  })
  update()
}

function toggleDocsSidebarSection(toggle) {
  const submenu = toggle.parentElement?.nextElementSibling
  if (!(submenu instanceof HTMLUListElement)) {
    return
  }

  const open = submenu.classList.toggle("menu-dropdown-show")
  toggle.setAttribute("aria-expanded", open ? "true" : "false")
  toggle.querySelector("svg")?.classList.toggle("rotate-90", open)
}

window.initDocsSidebarScrollspy = initDocsSidebarScrollspy
window.toggleDocsSidebarSection = toggleDocsSidebarSection
