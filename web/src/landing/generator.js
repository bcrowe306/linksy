// Free QR code generator (/qr-code-generator): live preview and downloads.
// Everything is rendered by the server's QR engine; nothing is stored.

export function initGenerator(root) {
  const form = root.querySelector('[data-qr-form]')
  const preview = root.querySelector('[data-qr-preview]')
  const status = root.querySelector('[data-qr-status]')
  const type = root.dataset.type

  function payload() {
    const raw = {}
    form.querySelectorAll('[data-raw]').forEach((el) => {
      raw[el.dataset.raw] = el.type === 'checkbox' ? el.checked : el.value
    })
    const body = { type, target_url: form.querySelector('[data-target]')?.value || '', raw_data: JSON.stringify(raw) }
    form.querySelectorAll('[data-style]').forEach((el) => (body[el.dataset.style] = el.value))
    return body
  }

  async function request(format) {
    const res = await fetch(`/api/v1/public/qr?format=${format}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...payload(), size: 1024 })
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || 'Something went wrong. Please try again.')
    }
    return res
  }

  function setStatus(text, tone) {
    status.textContent = text
    status.className = `mt-3 min-h-[1.25rem] text-center text-xs ${tone === 'ok' ? 'text-emerald-600 dark:text-emerald-400' : tone === 'warn' ? 'text-amber-600 dark:text-amber-400' : 'text-rose-500'}`
  }

  let seq = 0
  async function render() {
    const mine = ++seq
    try {
      const data = await (await request('json')).json()
      if (mine !== seq) return
      preview.innerHTML = data.svg
      preview.classList.remove('opacity-40')
      if (data.check?.scannable) setStatus(`✓ Scans reliably · contrast ${data.check.contrast.toFixed(1)}:1`, 'ok')
      else setStatus(data.check?.warnings?.[0] || 'This design may not scan reliably. Try more contrast.', 'warn')
    } catch (err) {
      if (mine !== seq) return
      preview.classList.add('opacity-40')
      setStatus(err.message, 'error')
    }
  }

  let timer
  const schedule = () => {
    clearTimeout(timer)
    timer = setTimeout(render, 300)
  }
  form.addEventListener('input', schedule)
  form.addEventListener('change', schedule)
  form.addEventListener('submit', (e) => e.preventDefault())

  root.querySelectorAll('[data-download]').forEach((btn) =>
    btn.addEventListener('click', async () => {
      const format = btn.dataset.download
      btn.disabled = true
      try {
        const blob = await (await request(format)).blob()
        const a = document.createElement('a')
        a.href = URL.createObjectURL(blob)
        a.download = `lynqr-${type}-qr-code.${format}`
        document.body.appendChild(a)
        a.click()
        a.remove()
        setTimeout(() => URL.revokeObjectURL(a.href), 1000)
      } catch (err) {
        setStatus(err.message, 'error')
      } finally {
        btn.disabled = false
      }
    })
  )

  render()
}
