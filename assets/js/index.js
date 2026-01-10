import "./vendor/datastar.js"

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
  const seconds = Math.floor((now - date) / 1000)

  if (seconds < 0) {
    return "just now"
  }

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
    const count = Math.floor(seconds / interval.seconds)
    if (count >= 1) {
      return count === 1
        ? `${count} ${interval.label} ago`
        : `${count} ${interval.label}s ago`
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
