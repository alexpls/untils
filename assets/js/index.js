import htmx from "htmx.org"
import Alpine from "alpinejs"

window.htmx = htmx
window.Alpine = Alpine

Alpine.data("timezone", (initialTz) => ({
  tz: initialTz,
  init() {
    const formatter = new Intl.DateTimeFormat(undefined, {
      timeZone: this.tz,
      timeZoneName: "shortGeneric",
    })
    const parts = formatter.formatToParts()
    const prettyTz = parts.find((p) => p.type == "timeZoneName")?.value
    if (prettyTz) {
      this.tz = prettyTz
    }
  },
}))

Alpine.start()

// store timezone in cookie - server will persist if it differs from previous value
try {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  document.cookie = `tz=${encodeURIComponent(timezone)}; path=/`
} catch (e) {
  console.error("error getting timezone", e)
}
