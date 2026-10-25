<template>
  <div class="space-y-6">
    <div v-if="!b" class="skeleton h-40" />
    <template v-else>
      <!-- Current status -->
      <section class="card p-5 sm:p-6" data-testid="billing-status">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 class="flex items-center gap-2 text-sm font-semibold text-fg">
              Lynqr Pro <span :class="status.badge" data-testid="billing-state">{{ status.label }}</span>
            </h2>
            <p class="mt-1 text-sm text-fg-muted">{{ status.detail }}</p>
            <p v-if="b.discount_label" class="mt-2 inline-flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400">
              <BadgePercent class="h-3.5 w-3.5" /> {{ b.discount_code ? `${b.discount_code}: ` : '' }}{{ b.discount_label }}
            </p>
          </div>
          <button v-if="b.has_customer && b.can_manage" class="btn-secondary shrink-0" :disabled="busy" data-testid="billing-portal" @click="portal">
            <ExternalLink class="h-4 w-4" /> Manage billing
          </button>
        </div>
        <div v-if="b.state === 'trial'" class="mt-4">
          <div class="mb-1 flex justify-between text-xs text-fg-muted"><span>Free trial</span><span>{{ b.trial_days_left }} of 7 days left</span></div>
          <div class="h-2 overflow-hidden rounded-full bg-subtle"><div class="h-full rounded-full bg-brand-500" :style="{ width: `${(b.trial_days_left / 7) * 100}%` }" /></div>
        </div>
      </section>

      <!-- Plans -->
      <section v-if="showPlans" class="card p-5 sm:p-6" data-testid="billing-plans">
        <div class="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-sm font-semibold text-fg">Choose your plan</h2>
            <p class="text-xs text-fg-muted">One flat price per workspace: unlimited members, links, QR codes and analytics.</p>
          </div>
          <div class="inline-flex rounded-lg bg-subtle p-1 text-sm" role="radiogroup" aria-label="Billing interval">
            <button
              v-for="opt in intervals"
              :key="opt.id"
              role="radio"
              :aria-checked="interval === opt.id ? 'true' : 'false'"
              class="rounded-md px-3 py-1.5 font-medium transition"
              :class="interval === opt.id ? 'bg-surface text-fg shadow-sm' : 'text-fg-muted hover:text-fg'"
              :data-testid="`interval-${opt.id}`"
              @click="interval = opt.id"
            >
              {{ opt.label }}<span v-if="opt.id === 'year'" class="ml-1.5 text-xs text-emerald-600 dark:text-emerald-400">Save {{ savings }}%</span>
            </button>
          </div>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <div class="rounded-xl border border-line p-5">
            <p class="text-xs font-medium uppercase tracking-wide text-fg-subtle">Pro · {{ interval === 'year' ? 'billed annually' : 'billed monthly' }}</p>
            <p class="mt-2 flex items-baseline gap-2">
              <span v-if="discount && !discount.free && discount.discounted_cents !== discount.original_cents" class="text-lg text-fg-subtle line-through">{{ money(perMonth(discount.original_cents)) }}</span>
              <span class="text-3xl font-semibold tabular-nums text-fg" data-testid="plan-price">{{ discount?.free ? '$0' : money(perMonth(discount ? discount.discounted_cents : priceCents)) }}</span>
              <span class="text-sm text-fg-muted">/ month</span>
            </p>
            <p class="mt-1 text-xs text-fg-muted">
              <template v-if="interval === 'year'">{{ money(discount && !discount.free ? discount.discounted_cents : priceCents) }} billed once a year</template>
              <template v-else>Cancel any time</template>
            </p>
            <ul class="mt-4 space-y-1.5 text-sm text-fg">
              <li v-for="f in features" :key="f" class="flex items-center gap-2"><Check class="h-4 w-4 text-emerald-500" />{{ f }}</li>
            </ul>
          </div>

          <form class="flex flex-col gap-3" @submit.prevent="subscribe">
            <div>
              <label class="label" for="promo-code">Checkout code <Tip text="Have a promotion or partner code? It's applied at checkout. Some codes make Lynqr free." /></label>
              <div class="flex gap-2">
                <input id="promo-code" v-model="code" maxlength="64" autocomplete="off" placeholder="LAUNCH50" class="input uppercase" :disabled="!b.can_manage" data-testid="promo-code" @keydown.enter.prevent="applyCode" />
                <button type="button" class="btn-secondary" :disabled="!code.trim() || checking || !b.can_manage" data-testid="promo-apply" @click="applyCode">Apply</button>
              </div>
              <p v-if="promoError" class="mt-1.5 text-xs text-rose-500" data-testid="promo-error">{{ promoError }}</p>
              <p v-else-if="promo" class="mt-1.5 flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400" data-testid="promo-ok">
                <BadgePercent class="h-3.5 w-3.5" />{{ promo.code }}: {{ promo.description }}<template v-if="promo.free"> · no card needed</template>
              </p>
              <button v-if="promo?.kind === 'trial'" type="button" class="btn-secondary btn-sm mt-2" :disabled="busy" data-testid="redeem-trial" @click="redeem">
                <CalendarPlus class="h-3.5 w-3.5" /> Add {{ promo.trial_days }} days to my trial
              </button>
            </div>
            <div class="mt-auto space-y-2">
              <button class="btn-primary w-full justify-center" :disabled="busy || !b.can_manage" data-testid="subscribe">
                {{ discount?.free ? 'Activate for free' : 'Subscribe' }}
              </button>
              <p class="text-center text-xs text-fg-muted">
                <template v-if="!b.can_manage">Only the workspace owner can change the plan.</template>
                <template v-else-if="b.state === 'trial'">You won't be charged until your trial ends. Payments are handled securely by Stripe.</template>
                <template v-else>Payments are handled securely by Stripe.</template>
              </p>
            </div>
          </form>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BadgePercent, Check, ExternalLink, CalendarPlus } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../../composables/useToast'
import Tip from '../ui/Tip.vue'

const auth = useAuthStore()
const toast = useToast()
const route = useRoute()
const router = useRouter()

const b = ref(null)
const interval = ref('year')
const code = ref('')
const promo = ref(null)
const promoError = ref('')
const checking = ref(false)
const busy = ref(false)

const intervals = [
  { id: 'year', label: 'Annual' },
  { id: 'month', label: 'Monthly' }
]
const features = ['Unlimited short links & dynamic QR codes', 'Full analytics with live feed & exports', 'Team roles, audit log & API access', 'Password protection, UTM tags & documents']

const priceCents = computed(() => (interval.value === 'year' ? b.value.annual_cents : b.value.monthly_cents))
const savings = computed(() => (b.value ? Math.round((1 - b.value.annual_cents / (b.value.monthly_cents * 12)) * 100) : 0))
const perMonth = (cents) => (interval.value === 'year' ? cents / 12 : cents)
const money = (cents) => `$${(cents / 100).toFixed(cents % 100 === 0 ? 0 : 2)}`
const date = (d) => new Date(d).toLocaleDateString(undefined, { month: 'long', day: 'numeric', year: 'numeric' })

const discount = computed(() => (promo.value?.kind === 'discount' ? promo.value : null))
const showPlans = computed(() => ['trial', 'expired', 'canceled'].includes(b.value?.state))

const status = computed(() => {
  const s = b.value
  const plan = s.interval === 'year' ? 'annual' : 'monthly'
  switch (s.state) {
    case 'comped':
      return { label: 'Complimentary', badge: 'badge-success', detail: 'This workspace has free access to Lynqr Pro.' }
    case 'active':
      return {
        label: s.cancel_at_period_end ? 'Ending' : 'Active',
        badge: s.cancel_at_period_end ? 'badge-warning' : 'badge-success',
        detail: s.cancel_at_period_end
          ? `Your ${plan} plan ends on ${date(s.current_period_end)}. Renew any time from Manage billing.`
          : `Your ${plan} plan renews on ${s.current_period_end ? date(s.current_period_end) : 'schedule'}.`
      }
    case 'trialing':
      return { label: 'Subscribed', badge: 'badge-success', detail: `Your ${plan} plan starts on ${date(s.current_period_end || s.trial_ends_at)}. Nothing is charged before then.` }
    case 'past_due':
      return { label: 'Payment due', badge: 'badge-warning', detail: 'Your last payment failed. Update your card in Manage billing to keep the workspace active.' }
    case 'trial':
      return { label: 'Free trial', badge: 'badge-brand', detail: `Your free trial ends on ${date(s.trial_ends_at)}. Subscribe to keep your links and QR codes working.` }
    case 'canceled':
      return { label: 'Canceled', badge: 'badge-danger', detail: 'Your subscription has ended. The workspace is read-only and its links and QR codes are paused.' }
    default:
      return { label: 'Trial ended', badge: 'badge-danger', detail: 'Your trial has ended. The workspace is read-only and its links and QR codes are paused until you subscribe.' }
  }
})

async function load() {
  b.value = await api.billing()
}

async function applyCode() {
  const c = code.value.trim()
  if (!c) return
  checking.value = true
  promoError.value = ''
  try {
    promo.value = await api.previewPromo(c, interval.value)
  } catch (err) {
    promo.value = null
    promoError.value = err.message
  } finally {
    checking.value = false
  }
}

// Prices in the preview depend on the interval.
watch(interval, () => promo.value && applyCode())
watch(code, (c) => {
  if (promo.value && c.trim().toUpperCase() !== promo.value.code.toUpperCase()) promo.value = null
  promoError.value = ''
})

async function redeem() {
  busy.value = true
  try {
    b.value = await api.redeemTrialCode(promo.value.code)
    toast.success(`Added ${promo.value.trial_days} days to your trial`)
    code.value = ''
    promo.value = null
    await auth.refresh()
  } catch (err) {
    toast.error(err.message)
  } finally {
    busy.value = false
  }
}

async function subscribe() {
  busy.value = true
  try {
    // Trial codes are redeemed separately, never sent to Stripe.
    const { url } = await api.startCheckout(interval.value, discount.value ? code.value.trim() : '')
    window.location.href = url
  } catch (err) {
    toast.error(err.message)
    busy.value = false
  }
}

async function portal() {
  busy.value = true
  try {
    const { url } = await api.billingPortal()
    window.location.href = url
  } catch (err) {
    toast.error(err.message)
    busy.value = false
  }
}

onMounted(async () => {
  const { checkout, code: preset, ...rest } = route.query
  if (checkout || preset) router.replace({ query: rest })
  if (checkout === 'success') toast.success('Thanks! Your subscription is being activated.')
  await load()
  if (checkout === 'success') {
    // Stripe's webhook usually lands within seconds of the redirect.
    for (let i = 0; i < 5 && showPlans.value; i++) {
      await new Promise((r) => setTimeout(r, 1500))
      await load()
    }
    await auth.refresh()
  }
  if (preset && showPlans.value) {
    code.value = String(preset)
    applyCode()
  }
})
</script>
