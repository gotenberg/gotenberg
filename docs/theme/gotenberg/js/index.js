/**
 * Setup.
 */

const items = document.querySelectorAll('.Page')
const links = document.querySelectorAll('.Menu a')

/**
 * Check if `el` is out out of view.
 */

function isBelowScroll(el) {
  return el.getBoundingClientRect().bottom > 0
}

/**
 * Activate item `i`.
 */

function activateItem(i) {
  links.forEach(e => e.classList.remove('active'))
  links[i].classList.add('active')
}

/**
 * Activate the correct menu item for the
 * contents in the viewport.
 */

function activate() {
  let i = 0

  for (; i < items.length; i++) {
    if (isBelowScroll(items[i])) {
      break
    }
  }

  activateItem(i)
}

/**
 * Activate scroll spy thingy.
 */

window.addEventListener('scroll', e => activate())
