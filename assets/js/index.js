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

// store timezone in cookie - server will persist if it differs from previous value
try {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  document.cookie = `tz=${encodeURIComponent(timezone)}; path=/`
} catch (e) {
  console.error("error getting timezone", e)
}
