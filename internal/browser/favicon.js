function extractFavicon() {
  const preference = ["svg", "png", "ico"]

  const links = document.querySelectorAll('link[rel~="icon"]')

  for (const ext of preference) {
    for (const link of links) {
      const href = link.getAttribute("href")

      if (href && href.endsWith(`.${ext}`)) {
        const abs = new URL(href, document.baseURI)
        return abs.toString()
      }
    }
  }

  return ""
}

extractFavicon()
