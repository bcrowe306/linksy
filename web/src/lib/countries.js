// Country names and ISO numeric codes (for the world map), loaded on demand.
let cache = null

export async function loadCountries() {
  if (cache) return cache
  const [lib, en] = await Promise.all([import('i18n-iso-countries'), import('i18n-iso-countries/langs/en.json')])
  const countries = lib.default || lib
  countries.registerLocale(en.default || en)
  cache = {
    name: (a2) => (!a2 || a2 === 'Unknown' ? 'Unknown' : countries.getName(a2, 'en', { select: 'alias' }) || countries.getName(a2, 'en') || a2),
    numeric: (a2) => countries.alpha2ToNumeric(a2),
    alpha2FromNumeric: (n) => countries.numericToAlpha2(String(n).padStart(3, '0'))
  }
  return cache
}

// flag turns an ISO alpha-2 code into its emoji flag.
export function flag(a2) {
  if (!a2 || a2.length !== 2) return '🌐'
  return String.fromCodePoint(...[...a2.toUpperCase()].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65))
}
