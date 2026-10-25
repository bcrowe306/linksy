<template>
  <div class="space-y-6">
    <!-- Checkout codes (Stripe) -->
    <section class="card overflow-hidden">
      <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-sm font-semibold text-fg">Checkout codes</h2>
          <p class="text-xs text-fg-muted">Discounts customers enter on the Billing page or at Stripe Checkout. A 100%-off code creates a free account without a card.</p>
        </div>
        <button class="btn-primary btn-sm" :disabled="!data?.billing_enabled" data-testid="new-checkout-code" @click="checkoutOpen = true"><Plus class="h-3.5 w-3.5" /> New code</button>
      </header>
      <p v-if="data && !data.billing_enabled" class="px-5 py-6 text-sm text-fg-muted">Stripe isn't configured, so checkout codes are unavailable.</p>
      <div v-else-if="!data" class="space-y-3 p-5"><div v-for="i in 3" :key="i" class="skeleton h-10" /></div>
      <EmptyState v-else-if="!data.stripe.length" :icon="TicketPercent" title="No checkout codes yet" description="Create one for a launch promotion, a partner or a free account." />
      <div v-else class="overflow-x-auto">
        <table class="w-full text-sm" data-testid="checkout-codes">
          <thead class="text-left text-xs text-fg-subtle">
            <tr class="border-b border-line">
              <th class="px-5 py-2.5 font-medium">Code</th>
              <th class="px-3 py-2.5 font-medium">Discount</th>
              <th class="px-3 py-2.5 text-right font-medium">Used</th>
              <th class="px-3 py-2.5 font-medium">Expires</th>
              <th class="px-5 py-2.5"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line">
            <tr v-for="c in data.stripe" :key="c.id">
              <td class="px-5 py-3">
                <span class="font-mono font-semibold text-fg">{{ c.code }}</span>
                <span v-if="!c.active" class="badge-neutral ml-2">Inactive</span>
                <span v-if="c.restrictions?.first_time_transaction" class="badge-neutral ml-2">New customers</span>
              </td>
              <td class="px-3 py-3 text-fg">{{ describe(c.coupon) }}</td>
              <td class="px-3 py-3 text-right tabular-nums text-fg">{{ c.times_redeemed }}<span v-if="c.max_redemptions" class="text-fg-subtle"> / {{ c.max_redemptions }}</span></td>
              <td class="px-3 py-3 text-fg-muted">{{ c.expires_at ? shortDate(c.expires_at * 1000) : 'Never' }}</td>
              <td class="px-5 py-3 text-right">
                <CopyButton :value="`${origin}/settings?tab=billing&code=${c.code}`" label="Copy checkout link" />
                <button class="btn-ghost btn-sm" @click="toggleCheckout(c)">{{ c.active ? 'Deactivate' : 'Activate' }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Trial codes (Lynqr) -->
    <section class="card overflow-hidden">
      <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-sm font-semibold text-fg">Trial codes</h2>
          <p class="text-xs text-fg-muted">Add free days to a workspace's trial. Each workspace can use a code once.</p>
        </div>
        <button class="btn-primary btn-sm" data-testid="new-trial-code" @click="trialOpen = true"><Plus class="h-3.5 w-3.5" /> New trial code</button>
      </header>
      <div v-if="!data" class="space-y-3 p-5"><div v-for="i in 2" :key="i" class="skeleton h-10" /></div>
      <EmptyState v-else-if="!data.trial.length" :icon="CalendarPlus" title="No trial codes yet" />
      <ul v-else class="divide-y divide-line" data-testid="trial-codes">
        <li v-for="c in data.trial" :key="c.code" class="flex flex-wrap items-center gap-3 px-5 py-3 text-sm">
          <span class="font-mono font-semibold text-fg">{{ c.code }}</span>
          <span class="text-fg">+{{ c.days }} days</span>
          <span v-if="!c.active" class="badge-neutral">Inactive</span>
          <span class="text-xs text-fg-muted">{{ c.redemptions }}{{ c.max_redemptions ? ` / ${c.max_redemptions}` : '' }} used · {{ c.expires_at ? `expires ${shortDate(c.expires_at)}` : 'no expiry' }}<span v-if="c.note"> · {{ c.note }}</span></span>
          <span class="flex-1" />
          <CopyButton :value="`${origin}/settings?tab=billing&code=${c.code}`" label="Copy link" />
          <button class="btn-ghost btn-sm" @click="toggleTrial(c)">{{ c.active ? 'Deactivate' : 'Activate' }}</button>
        </li>
      </ul>
    </section>

    <!-- New checkout code -->
    <AppModal :open="checkoutOpen" title="New checkout code" description="Created in Stripe as a coupon with a customer-facing promotion code." @close="checkoutOpen = false">
      <form class="space-y-4" @submit.prevent="createCheckout">
        <div>
          <label class="label" for="cc-code">Code</label>
          <input id="cc-code" v-model="cc.code" required maxlength="40" class="input font-mono uppercase" placeholder="LAUNCH50" data-testid="cc-code" />
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="label" for="cc-kind">Discount</label>
            <select id="cc-kind" v-model="cc.kind" class="input">
              <option value="percent">Percentage off</option>
              <option value="amount">Dollar amount off</option>
            </select>
          </div>
          <div>
            <label class="label" for="cc-value">{{ cc.kind === 'percent' ? 'Percent (1–100)' : 'Amount in USD' }}</label>
            <input id="cc-value" v-model.number="cc.value" type="number" :min="cc.kind === 'percent' ? 1 : 0.5" :max="cc.kind === 'percent' ? 100 : 60" step="any" required class="input" data-testid="cc-value" />
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="label" for="cc-duration">Applies to</label>
            <select id="cc-duration" v-model="cc.duration" class="input">
              <option value="once">The first payment</option>
              <option value="repeating">A number of months</option>
              <option value="forever">Every payment, forever</option>
            </select>
          </div>
          <div v-if="cc.duration === 'repeating'">
            <label class="label" for="cc-months">Months</label>
            <input id="cc-months" v-model.number="cc.months" type="number" min="1" max="36" required class="input" />
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="label" for="cc-max">Max uses <span class="font-normal text-fg-subtle">(optional)</span></label>
            <input id="cc-max" v-model.number="cc.max" type="number" min="1" class="input" placeholder="Unlimited" />
          </div>
          <div>
            <label class="label" for="cc-exp">Expires <span class="font-normal text-fg-subtle">(optional)</span></label>
            <input id="cc-exp" v-model="cc.expires" type="date" :min="today" class="input" />
          </div>
        </div>
        <label class="flex items-center gap-2 text-sm text-fg"><input v-model="cc.firstTime" type="checkbox" class="rounded" /> Only for customers who haven't paid before</label>
        <p class="rounded-lg bg-subtle px-3 py-2 text-xs text-fg-muted">Preview: <span class="font-medium text-fg">{{ preview }}</span></p>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn-ghost" @click="checkoutOpen = false">Cancel</button>
          <button class="btn-primary" :disabled="busy" data-testid="cc-create">Create code</button>
        </div>
      </form>
    </AppModal>

    <!-- New trial code -->
    <AppModal :open="trialOpen" title="New trial code" size="sm" @close="trialOpen = false">
      <form class="space-y-4" @submit.prevent="createTrial">
        <div>
          <label class="label" for="tc-code">Code</label>
          <input id="tc-code" v-model="tc.code" required maxlength="40" class="input font-mono uppercase" placeholder="EXTRA14" data-testid="tc-code" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="label" for="tc-days">Extra days</label>
            <input id="tc-days" v-model.number="tc.days" type="number" min="1" max="90" required class="input" />
          </div>
          <div>
            <label class="label" for="tc-max">Max uses</label>
            <input id="tc-max" v-model.number="tc.max" type="number" min="1" class="input" placeholder="Unlimited" />
          </div>
        </div>
        <div>
          <label class="label" for="tc-exp">Expires <span class="font-normal text-fg-subtle">(optional)</span></label>
          <input id="tc-exp" v-model="tc.expires" type="date" :min="today" class="input" />
        </div>
        <div>
          <label class="label" for="tc-note">Note <span class="font-normal text-fg-subtle">(optional)</span></label>
          <input id="tc-note" v-model="tc.note" maxlength="200" class="input" placeholder="Conference attendees" />
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn-ghost" @click="trialOpen = false">Cancel</button>
          <button class="btn-primary" :disabled="busy" data-testid="tc-create">Create code</button>
        </div>
      </form>
    </AppModal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, TicketPercent, CalendarPlus } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useToast } from '../../composables/useToast'
import { useAdminCall, shortDate } from '../../composables/useAdmin'
import AppModal from '../../components/ui/AppModal.vue'
import EmptyState from '../../components/ui/EmptyState.vue'
import CopyButton from '../../components/ui/CopyButton.vue'

const call = useAdminCall()
const toast = useToast()
const origin = window.location.origin
const today = new Date().toISOString().slice(0, 10)
const data = ref(null)
const busy = ref(false)
const checkoutOpen = ref(false)
const trialOpen = ref(false)
const cc = reactive({ code: '', kind: 'percent', value: 50, duration: 'repeating', months: 3, max: null, expires: '', firstTime: false })
const tc = reactive({ code: '', days: 14, max: null, expires: '', note: '' })

const load = async () => (data.value = await call(() => api.admin.codes()))
onMounted(load)

function describe(c) {
  const off = c.percent_off ? `${c.percent_off}% off` : `$${(c.amount_off / 100).toFixed(2)} off`
  if (c.percent_off === 100 && c.duration === 'forever') return 'Free forever'
  if (c.duration === 'forever') return `${off} forever`
  if (c.duration === 'repeating') return `${off} for ${c.duration_in_months} month${c.duration_in_months === 1 ? '' : 's'}`
  return `${off} the first payment`
}
const preview = computed(() =>
  describe({ percent_off: cc.kind === 'percent' ? cc.value : 0, amount_off: cc.kind === 'amount' ? Math.round((cc.value || 0) * 100) : 0, duration: cc.duration, duration_in_months: cc.months })
)
const endOfDay = (d) => (d ? new Date(`${d}T23:59:59`).toISOString() : null)

async function createCheckout() {
  busy.value = true
  try {
    await call(() =>
      api.admin.createCheckoutCode({
        code: cc.code,
        percent_off: cc.kind === 'percent' ? cc.value : 0,
        amount_off_cents: cc.kind === 'amount' ? Math.round(cc.value * 100) : 0,
        duration: cc.duration,
        duration_in_months: cc.duration === 'repeating' ? cc.months : 0,
        max_redemptions: cc.max || 0,
        expires_at: endOfDay(cc.expires),
        first_time_only: cc.firstTime
      })
    )
    toast.success(`Created ${cc.code.toUpperCase()}`)
    checkoutOpen.value = false
    Object.assign(cc, { code: '', max: null, expires: '' })
    await load()
  } finally {
    busy.value = false
  }
}

async function createTrial() {
  busy.value = true
  try {
    await call(() => api.admin.createTrialCode({ code: tc.code, days: tc.days, max_redemptions: tc.max || 0, expires_at: endOfDay(tc.expires), note: tc.note }))
    toast.success(`Created ${tc.code.toUpperCase()}`)
    trialOpen.value = false
    Object.assign(tc, { code: '', max: null, expires: '', note: '' })
    await load()
  } finally {
    busy.value = false
  }
}

async function toggleCheckout(c) {
  await call(() => api.admin.setCheckoutCode(c.id, !c.active, c.code))
  await load()
}
async function toggleTrial(c) {
  await call(() => api.admin.setTrialCode(c.code, !c.active))
  await load()
}
</script>
