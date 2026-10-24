<template>
  <div class="space-y-6">
    <!-- Top Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-gray-800 pb-4">
      <div class="flex items-center gap-3">
        <button
          @click="navigateBack"
          class="p-2 rounded-xl bg-gray-800 hover:bg-gray-700 text-gray-400 hover:text-white transition-colors"
          title="Back to QR Codes"
        >
          <ArrowLeft class="w-4 h-4" />
        </button>
        <div>
          <div class="flex items-center gap-2.5">
            <h1 class="text-xl font-bold text-white flex items-center gap-2">
              <span>{{ isEditing ? 'Edit QR Code' : 'QR Code Design Studio' }}</span>
              <span class="text-[10px] px-2 py-0.5 rounded-full bg-linksy-500/20 text-linksy-400 font-mono">PRO</span>
            </h1>
            <!-- Unsaved changes warning badge -->
            <span
              v-if="isDirty"
              class="text-[11px] px-2.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30 font-medium flex items-center gap-1"
            >
              <AlertTriangle class="w-3 h-3" />
              <span>Unsaved Changes</span>
            </span>
          </div>
          <p class="text-xs text-gray-400 mt-0.5">
            Full QR code spec customization: custom colors, gradients, module shapes, finder eyes, center logos, and frames.
          </p>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <button
          @click="saveQRCode"
          :disabled="isSaving"
          class="bg-linksy-600 hover:bg-linksy-500 text-white px-5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20 transition-all hover:scale-[1.02] active:scale-[0.98]"
        >
          <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
          <Save v-else class="w-4 h-4" />
          <span>{{ isSaving ? 'Saving...' : (isEditing ? 'Save Changes' : 'Create QR Code') }}</span>
        </button>
      </div>
    </div>

    <!-- Preset Style Templates Bar -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-4">
      <div class="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-2.5 flex items-center gap-1.5">
        <Sparkles class="w-3.5 h-3.5 text-amber-400" />
        <span>Quick Design Presets</span>
      </div>
      <div class="flex items-center gap-2.5 overflow-x-auto pb-1">
        <button
          v-for="preset in presets"
          :key="preset.name"
          @click="applyPreset(preset)"
          class="px-3 py-1.5 rounded-xl border text-xs font-medium whitespace-nowrap flex items-center gap-2 transition-all hover:border-linksy-500"
          :class="activePresetName === preset.name ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800/80 border-gray-700 text-gray-300'"
        >
          <span class="w-3.5 h-3.5 rounded-full border border-white/20 shrink-0" :style="{ background: preset.previewBg }"></span>
          <span>{{ preset.name }}</span>
        </button>
      </div>
    </div>

    <!-- Main Workspace: Left Controls (Tabs) & Right Live Canvas -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
      <!-- Left Controls (7 cols) -->
      <div class="lg:col-span-7 bg-gray-800/60 border border-gray-700/80 rounded-2xl overflow-hidden shadow-xl">
        <!-- Studio Category Navigation Tabs -->
        <div class="flex border-b border-gray-750 bg-gray-900/60 overflow-x-auto">
          <button
            v-for="tab in studioTabs"
            :key="tab.id"
            @click="activeStudioTab = tab.id"
            class="px-4 py-3 text-xs font-semibold flex items-center gap-2 border-b-2 whitespace-nowrap transition-all"
            :class="activeStudioTab === tab.id
              ? 'border-linksy-500 text-linksy-400 bg-gray-800/40'
              : 'border-transparent text-gray-400 hover:text-gray-200'"
          >
            <component :is="tab.icon" class="w-3.5 h-3.5" />
            <span>{{ tab.label }}</span>
          </button>
        </div>

        <div class="p-6 space-y-6">
          <!-- 1. CONTENT & PAYLOAD TAB -->
          <div v-show="activeStudioTab === 'content'" class="space-y-4">
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">QR Code Title *</label>
              <input
                v-model="qrData.title"
                type="text"
                placeholder="e.g. Summer Campaign or VIP Lounge Wi-Fi"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
            </div>

            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">QR Code Type</label>
              <select
                v-model="qrData.type"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              >
                <option value="website">Website / URL</option>
                <option value="document">Document (PDF / File)</option>
                <option value="vcard">Contact Card (vCard 3.0)</option>
                <option value="wifi">Wi-Fi Network</option>
                <option value="email">Email Pre-fill</option>
                <option value="sms">SMS Text Message</option>
                <option value="text">Plain Text</option>
              </select>
            </div>

            <!-- Website Target -->
            <div v-if="qrData.type === 'website'">
              <label class="block text-xs font-semibold text-gray-300 mb-1">Destination Target URL *</label>
              <input
                v-model="qrData.target_url"
                type="url"
                placeholder="https://example.com/promo"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
              <div class="mt-2 flex items-center gap-2">
                <input v-model="qrData.is_dynamic" type="checkbox" id="dynCheck" class="rounded bg-gray-800 border-gray-700 text-linksy-600 focus:ring-linksy-500" />
                <label for="dynCheck" class="text-xs text-gray-300 cursor-pointer">
                  Dynamic link routing (Edit destination URL anytime without reprinting)
                </label>
              </div>
            </div>

            <!-- Document Selection -->
            <div v-else-if="qrData.type === 'document'" class="space-y-3">
              <label class="block text-xs font-semibold text-gray-300">Choose Uploaded Document</label>
              <div v-if="store.documents.length > 0" class="max-h-40 overflow-y-auto space-y-1.5 border border-gray-700 rounded-xl p-2 bg-gray-800/40">
                <div
                  v-for="doc in store.documents"
                  :key="doc.id"
                  @click="selectDocument(doc)"
                  class="p-2 rounded-lg border cursor-pointer transition-all flex items-center justify-between text-xs"
                  :class="selectedDocId === doc.id ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800 border-gray-700 text-gray-300'"
                >
                  <div class="truncate max-w-[280px]">
                    <div class="font-medium truncate">{{ doc.filename }}</div>
                    <div class="text-[10px] text-gray-500">v{{ doc.version }} • {{ (doc.file_size / 1024 / 1024).toFixed(2) }} MB</div>
                  </div>
                  <Check v-if="selectedDocId === doc.id" class="w-4 h-4 text-linksy-400" />
                </div>
              </div>
              <div v-else class="text-xs text-gray-400">
                No documents uploaded yet. Upload documents in the Documents tab to link them here.
              </div>
            </div>

            <!-- Wi-Fi Network -->
            <div v-else-if="qrData.type === 'wifi'" class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-1">Network Name (SSID)</label>
                <input v-model="wifiData.ssid" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
              </div>
              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-1">Password</label>
                <input v-model="wifiData.password" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
              </div>
            </div>

            <!-- Contact Card vCard -->
            <div v-else-if="qrData.type === 'vcard'" class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-1">First Name</label>
                <input v-model="vcardData.first_name" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
              </div>
              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-1">Last Name</label>
                <input v-model="vcardData.last_name" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
              </div>
              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-1">Phone</label>
                <input v-model="vcardData.phone" type="tel" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
              </div>
              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-1">Email</label>
                <input v-model="vcardData.email" type="email" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
              </div>
            </div>

            <!-- Plain Text -->
            <div v-else-if="qrData.type === 'text'">
              <label class="block text-xs font-semibold text-gray-300 mb-1">Plain Text Content</label>
              <textarea v-model="qrData.raw_data" rows="3" class="w-full bg-gray-800 border border-gray-700 rounded-lg p-3 text-sm text-gray-100"></textarea>
            </div>
          </div>

          <!-- 2. COLORS & GRADIENTS TAB -->
          <div v-show="activeStudioTab === 'colors'" class="space-y-5">
            <!-- Fill Type: Solid vs Gradient -->
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-2">QR Pattern Fill Style</label>
              <div class="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  @click="style.gradient_type = 'none'"
                  class="py-2.5 px-4 rounded-xl border text-xs font-semibold flex items-center justify-center gap-2 transition-all"
                  :class="style.gradient_type === 'none' ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800/80 border-gray-700 text-gray-400'"
                >
                  <Palette class="w-4 h-4" />
                  <span>Solid Color</span>
                </button>
                <button
                  type="button"
                  @click="style.gradient_type = 'linear'"
                  class="py-2.5 px-4 rounded-xl border text-xs font-semibold flex items-center justify-center gap-2 transition-all"
                  :class="style.gradient_type === 'linear' ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800/80 border-gray-700 text-gray-400'"
                >
                  <SunMedium class="w-4 h-4" />
                  <span>Linear Gradient</span>
                </button>
              </div>
            </div>

            <!-- Solid Color Pickers -->
            <div v-if="style.gradient_type === 'none'" class="grid grid-cols-2 gap-4">
              <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                <label class="block text-xs font-semibold text-gray-300 mb-1.5">Pattern (Foreground)</label>
                <div class="flex items-center gap-3">
                  <input v-model="style.fg_color" type="color" class="w-10 h-10 rounded-lg cursor-pointer bg-transparent border-0" />
                  <input v-model="style.fg_color" type="text" class="w-24 font-mono text-xs bg-gray-900 px-2 py-1.5 rounded border border-gray-700 uppercase" />
                </div>
              </div>

              <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                <label class="block text-xs font-semibold text-gray-300 mb-1.5">Background</label>
                <div class="flex items-center gap-3">
                  <input v-model="style.bg_color" type="color" :disabled="style.is_transparent" class="w-10 h-10 rounded-lg cursor-pointer bg-transparent border-0 disabled:opacity-30" />
                  <input v-model="style.bg_color" type="text" :disabled="style.is_transparent" class="w-24 font-mono text-xs bg-gray-900 px-2 py-1.5 rounded border border-gray-700 uppercase disabled:opacity-30" />
                </div>
              </div>
            </div>

            <!-- Gradient Pickers -->
            <div v-else class="space-y-3">
              <div class="grid grid-cols-2 gap-4">
                <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                  <label class="block text-xs font-semibold text-gray-300 mb-1.5">Gradient Start</label>
                  <div class="flex items-center gap-3">
                    <input v-model="style.gradient_start" type="color" class="w-10 h-10 rounded-lg cursor-pointer bg-transparent border-0" />
                    <input v-model="style.gradient_start" type="text" class="w-24 font-mono text-xs bg-gray-900 px-2 py-1.5 rounded border border-gray-700 uppercase" />
                  </div>
                </div>

                <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                  <label class="block text-xs font-semibold text-gray-300 mb-1.5">Gradient End</label>
                  <div class="flex items-center gap-3">
                    <input v-model="style.gradient_end" type="color" class="w-10 h-10 rounded-lg cursor-pointer bg-transparent border-0" />
                    <input v-model="style.gradient_end" type="text" class="w-24 font-mono text-xs bg-gray-900 px-2 py-1.5 rounded border border-gray-700 uppercase" />
                  </div>
                </div>
              </div>
            </div>

            <!-- Transparent Toggle -->
            <div class="flex items-center justify-between p-3 bg-gray-800/40 rounded-xl border border-gray-750">
              <div>
                <span class="text-xs font-semibold text-gray-200 block">Transparent Background</span>
                <span class="text-[11px] text-gray-400">Export clean transparent PNG or SVG for dark mode or graphic design</span>
              </div>
              <input v-model="style.is_transparent" type="checkbox" class="w-4 h-4 rounded text-linksy-600 bg-gray-800 border-gray-700 focus:ring-linksy-500" />
            </div>

            <!-- Finder Eye Color Overrides -->
            <div class="border-t border-gray-750 pt-4 space-y-3">
              <span class="text-xs font-bold text-gray-300 block">Corner Finder Eye Colors</span>
              <div class="grid grid-cols-2 gap-4">
                <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                  <label class="block text-[11px] text-gray-400 mb-1">Outer Eye Frame</label>
                  <div class="flex items-center gap-2">
                    <input v-model="style.eye_color" type="color" class="w-8 h-8 rounded cursor-pointer bg-transparent border-0" />
                    <input v-model="style.eye_color" type="text" class="w-20 font-mono text-xs bg-gray-900 px-2 py-1 rounded border border-gray-700 uppercase" />
                  </div>
                </div>
                <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                  <label class="block text-[11px] text-gray-400 mb-1">Inner Eyeball</label>
                  <div class="flex items-center gap-2">
                    <input v-model="style.eye_inner_color" type="color" class="w-8 h-8 rounded cursor-pointer bg-transparent border-0" />
                    <input v-model="style.eye_inner_color" type="text" class="w-20 font-mono text-xs bg-gray-900 px-2 py-1 rounded border border-gray-700 uppercase" />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 3. SHAPES & SPEC PRECISION TAB -->
          <div v-show="activeStudioTab === 'shapes'" class="space-y-5">
            <!-- Dot / Module Shape -->
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-2">Module / Dot Shape</label>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                <button
                  v-for="shape in dotShapes"
                  :key="shape.id"
                  @click="style.dot_style = shape.id"
                  class="p-3 rounded-xl border text-xs font-medium flex flex-col items-center gap-2 transition-all"
                  :class="style.dot_style === shape.id ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800/80 border-gray-700 text-gray-400 hover:text-white'"
                >
                  <component :is="shape.icon" class="w-5 h-5 text-linksy-400" />
                  <span>{{ shape.name }}</span>
                </button>
              </div>
            </div>

            <!-- Finder Eye Shapes -->
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-2">Corner Finder Eye Shape</label>
              <div class="grid grid-cols-3 gap-2.5">
                <button
                  v-for="eye in eyeShapes"
                  :key="eye.id"
                  @click="style.eye_shape = eye.id"
                  class="py-2.5 px-3 rounded-xl border text-xs font-medium flex items-center justify-center gap-2 transition-all"
                  :class="style.eye_shape === eye.id ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800/80 border-gray-700 text-gray-400 hover:text-white'"
                >
                  <span>{{ eye.name }}</span>
                </button>
              </div>
            </div>

            <!-- Error Correction (QR Spec) -->
            <div class="border-t border-gray-750 pt-4 space-y-2">
              <div class="flex items-center justify-between">
                <label class="text-xs font-semibold text-gray-300">Error Correction Level (QR Spec)</label>
                <span class="text-[11px] text-linksy-400 font-mono">{{ style.error_correction }} ({{ getRecoveryPercent(style.error_correction) }})</span>
              </div>
              <div class="grid grid-cols-4 gap-2">
                <button
                  v-for="ec in ['L', 'M', 'Q', 'H']"
                  :key="ec"
                  @click="style.error_correction = ec"
                  class="py-2 rounded-lg border text-xs font-bold transition-all"
                  :class="style.error_correction === ec ? 'bg-linksy-600 text-white border-linksy-500' : 'bg-gray-800 border-gray-700 text-gray-400'"
                >
                  Level {{ ec }}
                </button>
              </div>
              <p class="text-[11px] text-gray-500 leading-relaxed">
                Level H (30% recovery) is recommended when adding logos or artwork in the center to maintain instant phone scannability.
              </p>
            </div>

            <!-- Quiet Zone Margin Slider -->
            <div class="border-t border-gray-750 pt-4 space-y-2">
              <div class="flex items-center justify-between text-xs">
                <span class="font-semibold text-gray-300">Quiet Zone (Margin)</span>
                <span class="font-mono text-gray-400">{{ style.margin }} modules</span>
              </div>
              <input v-model.number="style.margin" type="range" min="0" max="6" class="w-full accent-linksy-600" />
            </div>
          </div>

          <!-- 4. CENTER LOGO TAB -->
          <div v-show="activeStudioTab === 'logo'" class="space-y-5">
            <!-- Logo Upload -->
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-2">Upload Center Logo / Icon</label>
              <div
                @dragover.prevent
                @drop.prevent="handleLogoDrop"
                class="border-2 border-dashed border-gray-700 hover:border-linksy-500 rounded-xl p-5 flex flex-col items-center justify-center text-center cursor-pointer transition-colors"
                @click="$refs.logoInput.click()"
              >
                <input ref="logoInput" type="file" @change="handleLogoSelect" class="hidden" accept="image/*" />
                <div v-if="style.logo_base64" class="flex flex-col items-center gap-2">
                  <img :src="style.logo_base64" class="w-16 h-16 object-contain rounded-xl p-1 bg-white border border-gray-700" />
                  <span class="text-xs text-linksy-400 font-semibold">Click to replace logo</span>
                </div>
                <div v-else class="flex flex-col items-center">
                  <UploadCloud class="w-8 h-8 text-linksy-400 mb-1" />
                  <span class="text-xs font-semibold text-gray-200">Select or drop logo here</span>
                  <span class="text-[10px] text-gray-500 mt-0.5">PNG, SVG, or JPEG (transparent PNG recommended)</span>
                </div>
              </div>
              <button
                v-if="style.logo_base64"
                @click="style.logo_base64 = ''"
                type="button"
                class="mt-2 text-xs text-rose-400 hover:underline flex items-center gap-1"
              >
                <Trash2 class="w-3.5 h-3.5" />
                <span>Remove Logo</span>
              </button>
            </div>

            <!-- Logo Size & Badge Shapes -->
            <div v-if="style.logo_base64" class="space-y-4 border-t border-gray-750 pt-4">
              <div>
                <div class="flex items-center justify-between text-xs mb-1.5">
                  <span class="font-semibold text-gray-300">Logo Size Proportion</span>
                  <span class="font-mono text-gray-400">{{ Math.round((style.logo_size || 0.22) * 100) }}%</span>
                </div>
                <input v-model.number="style.logo_size" type="range" min="0.16" max="0.28" step="0.02" class="w-full accent-linksy-600" />
              </div>

              <div>
                <label class="block text-xs font-semibold text-gray-300 mb-2">Logo Background Clearance Badge</label>
                <div class="grid grid-cols-4 gap-2">
                  <button
                    v-for="shape in ['circle', 'rounded', 'square', 'none']"
                    :key="shape"
                    @click="style.logo_bg_shape = shape"
                    class="py-2 rounded-lg border text-xs font-medium capitalize transition-all"
                    :class="style.logo_bg_shape === shape ? 'bg-linksy-600 text-white border-linksy-500' : 'bg-gray-800 border-gray-700 text-gray-400'"
                  >
                    {{ shape }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 5. FRAMES, BORDERS & LABELS TAB -->
          <div v-show="activeStudioTab === 'frames'" class="space-y-5">
            <!-- Frame Style -->
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-2">Frame Style</label>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                <button
                  v-for="frame in frameStyles"
                  :key="frame.id"
                  @click="style.frame_style = frame.id"
                  class="py-2.5 px-3 rounded-xl border text-xs font-medium text-center transition-all"
                  :class="style.frame_style === frame.id ? 'bg-linksy-600/20 border-linksy-500 text-white' : 'bg-gray-800/80 border-gray-700 text-gray-400 hover:text-white'"
                >
                  {{ frame.name }}
                </button>
              </div>
            </div>

            <!-- Frame Label Text -->
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Frame Label Banner Text</label>
              <input
                v-model="style.label_text"
                type="text"
                placeholder="SCAN ME"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 font-bold uppercase tracking-wider focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
            </div>

            <!-- Border Thickness & Border Color -->
            <div class="grid grid-cols-2 gap-4">
              <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                <div class="flex justify-between text-xs mb-1">
                  <span class="font-semibold text-gray-300">Border Thickness</span>
                  <span class="font-mono text-gray-400">{{ style.border_width }}px</span>
                </div>
                <input v-model.number="style.border_width" type="range" min="0" max="16" class="w-full accent-linksy-600" />
              </div>

              <div class="bg-gray-800/80 p-3 rounded-xl border border-gray-700">
                <label class="block text-xs font-semibold text-gray-300 mb-1.5">Border Color</label>
                <div class="flex items-center gap-2">
                  <input v-model="style.border_color" type="color" class="w-8 h-8 rounded cursor-pointer bg-transparent border-0" />
                  <input v-model="style.border_color" type="text" class="w-20 font-mono text-xs bg-gray-900 px-2 py-1 rounded border border-gray-700 uppercase" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right Interactive Preview & Export Studio (5 cols) -->
      <div class="lg:col-span-5 sticky top-6 space-y-4">
        <div class="bg-gray-800/80 border border-gray-700/80 rounded-2xl p-6 flex flex-col items-center shadow-2xl">
          <div class="w-full flex items-center justify-between text-xs font-semibold text-gray-400 uppercase tracking-wider mb-5">
            <span class="flex items-center gap-1.5">
              <Eye class="w-4 h-4 text-linksy-400" />
              <span>Real-Time Canvas Preview</span>
            </span>
            <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 font-mono">Live</span>
          </div>

          <!-- The Rendered Preview Card -->
          <div
            class="relative flex flex-col items-center justify-center p-6 rounded-2xl transition-all duration-300 shadow-2xl"
            :style="{
              backgroundColor: style.is_transparent ? 'transparent' : (style.frame_bg_color || style.bg_color || '#ffffff'),
              border: style.border_width > 0 ? `${style.border_width}px solid ${style.border_color || style.fg_color || '#3b82f6'}` : 'none'
            }"
          >
            <!-- Label Banner Top -->
            <div
              v-if="style.label_text && style.frame_style === 'banner-top'"
              class="mb-3 text-sm font-extrabold tracking-wider uppercase text-center"
              :style="{ color: style.label_color || style.fg_color || '#000000' }"
            >
              {{ style.label_text }}
            </div>

            <!-- Center QR Canvas / SVG -->
            <div class="relative w-60 h-60 flex items-center justify-center">
              <div v-html="renderedSvg" class="w-full h-full flex items-center justify-center"></div>
            </div>

            <!-- Label Banner Bottom (Default) -->
            <div
              v-if="style.label_text && style.frame_style !== 'banner-top'"
              class="mt-3 text-sm font-extrabold tracking-wider uppercase text-center"
              :style="{ color: style.label_color || style.fg_color || '#000000' }"
            >
              {{ style.label_text }}
            </div>
          </div>

          <!-- Format Export Center -->
          <div class="mt-6 w-full space-y-2.5 pt-4 border-t border-gray-700/60">
            <span class="text-xs font-semibold text-gray-300 block text-center">Instant High-Res Exports</span>
            <div class="grid grid-cols-2 gap-2">
              <button
                @click="downloadFormat('png')"
                class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-xl text-xs font-semibold transition-colors flex items-center justify-center gap-1.5"
              >
                <Download class="w-3.5 h-3.5 text-linksy-400" />
                <span>PNG (3000px)</span>
              </button>
              <button
                @click="downloadFormat('jpeg')"
                class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-xl text-xs font-semibold transition-colors flex items-center justify-center gap-1.5"
              >
                <Download class="w-3.5 h-3.5 text-amber-400" />
                <span>JPEG (HD)</span>
              </button>
              <button
                @click="downloadFormat('pdf')"
                class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-xl text-xs font-semibold transition-colors flex items-center justify-center gap-1.5"
              >
                <FileText class="w-3.5 h-3.5 text-rose-400" />
                <span>Printable PDF</span>
              </button>
              <button
                @click="downloadFormat('svg')"
                class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-xl text-xs font-semibold transition-colors flex items-center justify-center gap-1.5"
              >
                <Download class="w-3.5 h-3.5 text-emerald-400" />
                <span>Vector SVG</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import {
  ArrowLeft,
  Save,
  Sparkles,
  Link2,
  Palette,
  Shapes,
  Image as ImageIcon,
  Frame,
  Check,
  Eye,
  Download,
  FileText,
  UploadCloud,
  Trash2,
  SunMedium,
  Square,
  Circle,
  Loader2,
  AlertTriangle
} from 'lucide-vue-next'
import { api } from '../api/client'
import { useLinksyStore } from '../stores/linksy'

const route = useRoute()
const router = useRouter()
const store = useLinksyStore()

const isEditing = computed(() => !!route.params.id)
const isSaving = ref(false)
const activeStudioTab = ref('content')
const activePresetName = ref('Midnight Indigo')

const studioTabs = [
  { id: 'content', label: '1. Content', icon: Link2 },
  { id: 'colors', label: '2. Colors & Gradients', icon: Palette },
  { id: 'shapes', label: '3. Dots & Eyes', icon: Shapes },
  { id: 'logo', label: '4. Center Logo', icon: ImageIcon },
  { id: 'frames', label: '5. Frame & Banner', icon: Frame },
]

const dotShapes = [
  { id: 'square', name: 'Square (Classic)', icon: Square },
  { id: 'rounded', name: 'Smooth Rounded', icon: Square },
  { id: 'dots', name: 'Circle Dots', icon: Circle },
  { id: 'classy', name: 'Classy Minimal', icon: Square },
]

const eyeShapes = [
  { id: 'square', name: 'Square' },
  { id: 'rounded', name: 'Rounded' },
  { id: 'circle', name: 'Circle' },
]

const frameStyles = [
  { id: 'banner-bottom', name: 'Bottom Banner' },
  { id: 'banner-top', name: 'Top Banner' },
  { id: 'box', name: 'Simple Box' },
  { id: 'none', name: 'No Frame' },
]

const presets = [
  {
    name: 'Midnight Indigo',
    previewBg: 'linear-gradient(135deg, #4338CA, #06B6D4)',
    style: {
      fg_color: '#4338CA',
      bg_color: '#FFFFFF',
      gradient_type: 'linear',
      gradient_start: '#4338CA',
      gradient_end: '#06B6D4',
      dot_style: 'rounded',
      eye_shape: 'rounded',
      eye_color: '#4338CA',
      eye_inner_color: '#06B6D4',
      border_width: 3,
      border_color: '#4338CA',
      label_text: 'SCAN ME',
      label_color: '#4338CA',
      frame_style: 'banner-bottom'
    }
  },
  {
    name: 'Emerald Luxury',
    previewBg: 'linear-gradient(135deg, #059669, #10B981)',
    style: {
      fg_color: '#064E3B',
      bg_color: '#F0FDF4',
      gradient_type: 'linear',
      gradient_start: '#064E3B',
      gradient_end: '#10B981',
      dot_style: 'dots',
      eye_shape: 'circle',
      eye_color: '#064E3B',
      eye_inner_color: '#10B981',
      border_width: 3,
      border_color: '#059669',
      label_text: 'EXPLORE',
      label_color: '#064E3B',
      frame_style: 'banner-bottom'
    }
  },
  {
    name: 'Neon Sunset',
    previewBg: 'linear-gradient(135deg, #DB2777, #F59E0B)',
    style: {
      fg_color: '#DB2777',
      bg_color: '#FFFFFF',
      gradient_type: 'linear',
      gradient_start: '#DB2777',
      gradient_end: '#F59E0B',
      dot_style: 'dots',
      eye_shape: 'rounded',
      eye_color: '#DB2777',
      eye_inner_color: '#F59E0B',
      border_width: 3,
      border_color: '#DB2777',
      label_text: 'VISIT US',
      label_color: '#DB2777',
      frame_style: 'banner-bottom'
    }
  },
  {
    name: 'Classic Monochrome',
    previewBg: '#000000',
    style: {
      fg_color: '#000000',
      bg_color: '#FFFFFF',
      gradient_type: 'none',
      dot_style: 'square',
      eye_shape: 'square',
      eye_color: '#000000',
      eye_inner_color: '#000000',
      border_width: 2,
      border_color: '#000000',
      label_text: 'SCAN ME',
      label_color: '#000000',
      frame_style: 'banner-bottom'
    }
  }
]

const qrData = ref({
  title: 'My Custom QR Code',
  type: 'website',
  target_url: 'https://linksy.dev',
  is_dynamic: true,
  raw_data: ''
})

const style = ref({
  fg_color: '#4338CA',
  bg_color: '#FFFFFF',
  is_transparent: false,
  gradient_type: 'linear',
  gradient_start: '#4338CA',
  gradient_end: '#06B6D4',
  gradient_angle: 135,
  dot_style: 'rounded',
  eye_shape: 'rounded',
  eye_color: '#4338CA',
  eye_inner_color: '#06B6D4',
  error_correction: 'H',
  margin: 2,
  logo_base64: '',
  logo_size: 0.22,
  logo_bg_shape: 'circle',
  logo_padding: 6,
  frame_style: 'banner-bottom',
  border_width: 3,
  border_color: '#4338CA',
  frame_bg_color: '#FFFFFF',
  label_text: 'SCAN ME',
  label_color: '#4338CA'
})

const wifiData = ref({ ssid: '', password: '', encryption: 'WPA' })
const vcardData = ref({ first_name: '', last_name: '', phone: '', email: '' })
const selectedDocId = ref('')

// Unsaved changes tracking (Requirement 2)
const baselineSnapshot = ref('')

function createSnapshot() {
  return JSON.stringify({
    qrData: qrData.value,
    style: style.value,
    wifiData: wifiData.value,
    vcardData: vcardData.value
  })
}

const isDirty = computed(() => {
  if (!baselineSnapshot.value) return false
  return createSnapshot() !== baselineSnapshot.value
})

function applyPreset(preset) {
  activePresetName.value = preset.name
  style.value = { ...style.value, ...preset.style }
}

function selectDocument(doc) {
  selectedDocId.value = doc.id
  qrData.value.target_url = `${window.location.origin}/api/v1/documents/files/${doc.stored_path}`
  if (!qrData.value.title || qrData.value.title === 'My Custom QR Code') {
    qrData.value.title = `Doc: ${doc.filename}`
  }
}

function handleLogoSelect(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (ev) => {
    style.value.logo_base64 = ev.target.result
    style.value.error_correction = 'H' // Ensure highest error recovery
  }
  reader.readAsDataURL(file)
}

function handleLogoDrop(e) {
  const file = e.dataTransfer.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (ev) => {
    style.value.logo_base64 = ev.target.result
    style.value.error_correction = 'H'
  }
  reader.readAsDataURL(file)
}

function getRecoveryPercent(level) {
  switch (level) {
    case 'L': return '7%'
    case 'M': return '15%'
    case 'Q': return '25%'
    case 'H': return '30%'
    default: return '30%'
  }
}

// Live client-side SVG renderer for immediate zero-lag canvas feedback
const renderedSvg = computed(() => {
  const target = qrData.value.target_url || 'https://linksy.dev'
  const fg = style.value.gradient_type === 'linear' ? 'url(#liveGrad)' : (style.value.fg_color || '#000000')
  const bg = style.value.is_transparent ? 'transparent' : (style.value.bg_color || '#ffffff')

  // Generate lightweight mock SVG modules for instant rendering
  const matrix = 25
  const cellSize = 8
  const total = matrix * cellSize

  let svg = `<svg viewBox="0 0 ${total} ${total}" width="100%" height="100%" xmlns="http://www.w3.org/2000/svg">`
  if (style.value.gradient_type === 'linear') {
    svg += `<defs><linearGradient id="liveGrad" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" stop-color="${style.value.gradient_start}"/><stop offset="100%" stop-color="${style.value.gradient_end}"/></linearGradient></defs>`
  }
  if (!style.value.is_transparent) {
    svg += `<rect width="${total}" height="${total}" fill="${bg}"/>`
  }

  // Draw simulated pattern modules
  for (let r = 0; r < matrix; r++) {
    for (let c = 0; c < matrix; c++) {
      // Finder patterns
      const isTopLeft = r < 7 && c < 7
      const isTopRight = r < 7 && c >= matrix - 7
      const isBottomLeft = r >= matrix - 7 && c < 7

      if (isTopLeft || isTopRight || isBottomLeft) {
        // Outer box or inner dot
        const x = c * cellSize
        const y = r * cellSize
        const ec = style.value.eye_color || style.value.fg_color || '#000000'
        if ((r === 0 || r === 6 || c === 0 || c === 6 || (r >= 2 && r <= 4 && c >= 2 && c <= 4)) && isTopLeft) {
          svg += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${ec}"/>`
        } else if ((r === 0 || r === 6 || c === matrix - 7 || c === matrix - 1 || (r >= 2 && r <= 4 && c >= matrix - 5 && c <= matrix - 3)) && isTopRight) {
          svg += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${ec}"/>`
        } else if ((r === matrix - 7 || r === matrix - 1 || c === 0 || c === 6 || (r >= matrix - 5 && r <= matrix - 3 && c >= 2 && c <= 4)) && isBottomLeft) {
          svg += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${ec}"/>`
        }
      } else {
        // Pseudo-hash fill for live visual feedback
        const fill = (r * 7 + c * 13 + target.length) % 3 !== 0
        if (fill) {
          const x = c * cellSize
          const y = r * cellSize
          if (style.value.dot_style === 'dots') {
            svg += `<circle cx="${x + cellSize / 2}" cy="${y + cellSize / 2}" r="${cellSize * 0.45}" fill="${fg}"/>`
          } else if (style.value.dot_style === 'rounded') {
            svg += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" rx="${cellSize * 0.3}" fill="${fg}"/>`
          } else {
            svg += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${fg}"/>`
          }
        }
      }
    }
  }

  // Draw Logo in Preview
  if (style.value.logo_base64) {
    const lSize = total * (style.value.logo_size || 0.22)
    const lx = (total - lSize) / 2
    const ly = (total - lSize) / 2
    if (style.value.logo_bg_shape === 'circle') {
      svg += `<circle cx="${total / 2}" cy="${total / 2}" r="${(lSize / 2) + 4}" fill="${bg}"/>`
    } else {
      svg += `<rect x="${lx - 4}" y="${ly - 4}" width="${lSize + 8}" height="${lSize + 8}" rx="6" fill="${bg}"/>`
    }
    svg += `<image x="${lx}" y="${ly}" width="${lSize}" height="${lSize}" href="${style.value.logo_base64}"/>`
  }

  svg += `</svg>`
  return svg
})

function navigateBack() {
  router.push('/qr-codes')
}

async function saveQRCode() {
  if (!qrData.value.title) {
    alert('Please enter a title for the QR code')
    return
  }

  isSaving.value = true
  try {
    let rawData = ''
    if (qrData.value.type === 'wifi') {
      rawData = JSON.stringify(wifiData.value)
    } else if (qrData.value.type === 'vcard') {
      rawData = JSON.stringify(vcardData.value)
    }

    if (isEditing.value) {
      await api.updateQRCode(route.params.id, {
        title: qrData.value.title,
        target_url: qrData.value.target_url,
        raw_data: rawData,
        style: style.value
      })
      alert('QR code updated successfully!')
    } else {
      await api.createQRCode({
        title: qrData.value.title,
        type: qrData.value.type,
        target_url: qrData.value.target_url,
        is_dynamic: qrData.value.is_dynamic,
        raw_data: rawData,
        style: style.value
      })
      alert('New custom QR code created successfully!')
    }

    // Reset baseline snapshot so navigation guard does not trigger
    baselineSnapshot.value = createSnapshot()
    await store.fetchQRCodes()
    router.push('/qr-codes')
  } catch (err) {
    alert(err.message || 'Failed saving QR code')
  } finally {
    isSaving.value = false
  }
}

function downloadFormat(fmt) {
  if (isEditing.value) {
    window.open(`/api/v1/qr/${route.params.id}/export?format=${fmt}&download=true`, '_blank')
    return
  }

  if (fmt === 'svg') {
    const blob = new Blob([renderedSvg.value], { type: 'image/svg+xml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `linksy-qr-${Date.now()}.svg`
    a.click()
  } else {
    if (confirm('Save this QR code to download high-resolution PNG, JPEG, or PDF?')) {
      saveQRCode()
    }
  }
}

// Intercept in-app navigation if unsaved changes exist (Requirement 2)
onBeforeRouteLeave((to, from, next) => {
  if (isDirty.value) {
    const answer = window.confirm(
      'You have unsaved changes in the QR Code Studio. If you leave now, your customizations will be lost.\n\nDo you want to discard your changes and leave?'
    )
    if (answer) {
      next()
    } else {
      next(false)
    }
  } else {
    next()
  }
})

// Intercept browser-level window close / page refresh
function handleBeforeUnload(e) {
  if (isDirty.value) {
    e.preventDefault()
    e.returnValue = ''
    return ''
  }
}

onMounted(async () => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  await store.fetchDocuments()

  if (isEditing.value) {
    try {
      const qr = await api.getQRCodes()
      const found = (qr.data || []).find(q => q.id === route.params.id)
      if (found) {
        qrData.value.title = found.title
        qrData.value.type = found.type
        qrData.value.target_url = found.payload
        qrData.value.is_dynamic = !!found.link_id
        if (found.style) {
          style.value = { ...style.value, ...found.style }
        }
      }
    } catch (e) {
      console.error(e)
    }
  }

  // Set baseline snapshot after initial data load
  baselineSnapshot.value = createSnapshot()
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>
