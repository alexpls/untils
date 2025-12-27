;(function () {
  // Only keep things in the main content area (.mw-body)
  const mainContent = document.querySelector(".mw-body")
  if (mainContent) {
    document.body.innerHTML = ""
    document.body.appendChild(mainContent)
  }

  // Remove Wikipedia-specific elements
  const selectorsToRemove = [
    "#References",
    ".references",
    ".reflist",
    ".mw-editsection",
    ".navbox",
    ".sistersitebox",
    ".mw-jump-link",
    "#mw-navigation",
    "#mw-head",
    "#mw-panel",
    "#footer",
    ".mw-indicators",
    ".catlinks",
    "#siteSub",
    "#contentSub",
    ".mw-authority-control",
    "#External_links",
    "#See_also",
    "#Notes",
    "#Further_reading",
    ".infobox",
    ".sidebar",
    ".ambox",
    ".tmbox",
    ".ombox",
    ".mbox-small",
    ".metadata",
    ".hatnote",
    ".shortdescription",
  ]

  selectorsToRemove.forEach((selector) => {
    document.querySelectorAll(selector).forEach((el) => el.remove())
  })

  const sectionIds = [
    "Contents",
    "References",
    "External_links",
    "See_also",
    "Notes",
    "Further_reading",
  ]
  sectionIds.forEach((id) => {
    const heading = document.getElementById(id)
    if (heading) {
      // remove the heading and all following siblings until next heading
      let parent = heading.closest("h2, h3, h4")
      if (parent) {
        let sibling = parent.nextElementSibling
        while (sibling && !sibling.matches("h2")) {
          const next = sibling.nextElementSibling
          sibling.remove()
          sibling = next
        }
        parent.remove()
      }
    }
  })
})()
