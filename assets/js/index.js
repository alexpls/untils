import "./vendor/datastar.js"
import "./schedule.js"
import "./textcarousel.js"

if (!("anchorName" in document.documentElement.style)) {
  import("@oddbird/css-anchor-positioning").catch((error) => {
    console.error("error loading anchor positioning polyfill", error)
  })
}

function timezoneName(tz) {
  const formatter = new Intl.DateTimeFormat(undefined, {
    timeZone: tz,
    timeZoneName: "shortGeneric",
  })
  const parts = formatter.formatToParts()
  const prettyTz = parts.find((p) => p.type == "timeZoneName")?.value
  if (!prettyTz) {
    return ""
  }
  return prettyTz
}
window.timezoneName = timezoneName

function timeAgo(isoTimestamp) {
  const date = new Date(isoTimestamp)
  const now = new Date()
  const diff = now - date
  const seconds = Math.floor(Math.abs(diff) / 1000)
  const isFuture = diff < 0

  const intervals = [
    { label: "year", seconds: 31536000 },
    { label: "month", seconds: 2592000 },
    { label: "week", seconds: 604800 },
    { label: "day", seconds: 86400 },
    { label: "hour", seconds: 3600 },
    { label: "minute", seconds: 60 },
    { label: "second", seconds: 1 },
  ]

  for (const interval of intervals) {
    const count = Math.round(seconds / interval.seconds)
    if (count >= 1) {
      const label = count === 1 ? interval.label : `${interval.label}s`
      return isFuture ? `in ${count} ${label}` : `${count} ${label} ago`
    }
  }

  return "just now"
}
window.timeAgo = timeAgo

// store timezone in cookie - server will persist if it differs from previous value
try {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  document.cookie = `tz=${encodeURIComponent(timezone)}; path=/`
} catch (e) {
  console.error("error getting timezone", e)
}
