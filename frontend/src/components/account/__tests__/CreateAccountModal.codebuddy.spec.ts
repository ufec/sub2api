import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  generateAuthUrlMock,
  exchangeStateMock,
  createAccountMock,
  authIsSimpleMode,
} = vi.hoisted(() => ({
  generateAuthUrlMock: vi.fn(),
  exchangeStateMock: vi.fn(),
  createAccountMock: vi.fn(),
  authIsSimpleMode: { value: true },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return authIsSimpleMode.value
    },
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false }),
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({}),
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([]),
    },
    codebuddy: {
      generateAuthUrl: generateAuthUrlMock,
      exchangeState: exchangeStateMock,
      refreshCodeBuddyToken: vi.fn(),
    },
    grok: { getCapabilities: vi.fn().mockResolvedValue({ password_auth_enabled: false }) },
  },
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn().mockResolvedValue([]),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

const GroupSelectorStub = defineComponent({
  name: 'GroupSelector',
  props: { modelValue: { type: Array, default: () => [] }, groups: { type: Array, default: () => [] }, platform: String, mixedScheduling: Boolean },
  emits: ['update:modelValue'],
  template: '<div />',
})

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  props: { modelValue: { type: Array, default: () => [] }, platform: String },
  emits: ['update:modelValue', 'upstream-synced'],
  template: '<div />',
})

// Real OAuthAuthorizationFlow; only leaf presentational components stubbed.
function mountModal() {
  return mount(CreateAccountModal, {
    props: { show: true, proxies: [], groups: [] },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        Select: true,
        Icon: true,
        PlatformIcon: true,
        ProxySelector: true,
        ProxyAdBanner: true,
        GroupSelector: GroupSelectorStub,
        ModelWhitelistSelector: ModelWhitelistSelectorStub,
        QuotaLimitCard: true,
        CnBaseUrlPresets: true,
      },
    },
  })
}

async function selectButtonByText(wrapper: ReturnType<typeof mountModal>, text: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.text().includes(text))
  expect(button, `button containing text "${text}"`).toBeDefined()
  await button?.trigger('click')
}

async function openCodeBuddyStep2() {
  const wrapper = mountModal()
  await selectButtonByText(wrapper, 'CodeBuddy')
  await wrapper.get('form#create-account-form input[type="text"]').setValue('cb account')
  await wrapper.get('form#create-account-form').trigger('submit.prevent')
  await flushPromises()
  return wrapper
}

const findButtonByText = (wrapper: ReturnType<typeof mountModal>, text: string) =>
  wrapper.findAll('button').find((candidate) => candidate.text().includes(text))

describe('CreateAccountModal CodeBuddy OAuth flow', () => {
  beforeEach(() => {
    authIsSimpleMode.value = true
    generateAuthUrlMock.mockReset().mockResolvedValue({
      auth_url: 'https://copilot.tencent.com/auth?state=st-1',
      session_id: 'sid-1',
      state: 'st-1',
    })
    exchangeStateMock.mockReset()
    createAccountMock.mockReset().mockResolvedValue({ id: 7 })
  })

  it('starts with only 生成授权链接 clickable; 校验认证状态 and 完成授权 stay disabled', async () => {
    const wrapper = await openCodeBuddyStep2()

    expect(findButtonByText(wrapper, 'generateAuthUrl')).toBeDefined()
    expect(findButtonByText(wrapper, 'verifyAuthState')!.attributes('disabled')).toBeDefined()
    expect(findButtonByText(wrapper, 'completeAuth')!.attributes('disabled')).toBeDefined()
  })

  it('enables 校验认证状态 after the auth URL is generated; 完成授权 stays disabled', async () => {
    const wrapper = await openCodeBuddyStep2()

    await findButtonByText(wrapper, 'generateAuthUrl')!.trigger('click')
    await flushPromises()
    expect(generateAuthUrlMock).toHaveBeenCalledTimes(1)

    expect(findButtonByText(wrapper, 'verifyAuthState')!.attributes('disabled')).toBeUndefined()
    expect(findButtonByText(wrapper, 'completeAuth')!.attributes('disabled')).toBeDefined()
  })

  it('keeps 校验认证状态 retryable on failure and only unlocks 完成授权 after success', async () => {
    const wrapper = await openCodeBuddyStep2()
    await findButtonByText(wrapper, 'generateAuthUrl')!.trigger('click')
    await flushPromises()

    const verifyBtn = findButtonByText(wrapper, 'verifyAuthState')!
    const completeBtn = findButtonByText(wrapper, 'completeAuth')!

    // First verification fails (browser authorization not finished yet)
    exchangeStateMock.mockRejectedValueOnce({ message: 'not authorized' })
    await verifyBtn.trigger('click')
    await flushPromises()
    expect(exchangeStateMock).toHaveBeenCalledTimes(1)
    expect(verifyBtn.attributes('disabled')).toBeUndefined()
    expect(completeBtn.attributes('disabled')).toBeDefined()

    // Second verification succeeds → token stored, verify disabled, complete enabled
    exchangeStateMock.mockResolvedValueOnce({ access_token: 'at', refresh_token: 'rt' })
    await verifyBtn.trigger('click')
    await flushPromises()
    expect(exchangeStateMock).toHaveBeenCalledTimes(2)
    expect(createAccountMock).not.toHaveBeenCalled()
    expect(verifyBtn.attributes('disabled')).toBeDefined()
    expect(completeBtn.attributes('disabled')).toBeUndefined()
  })

  it('创建账号 on 完成授权 uses the token obtained during verification', async () => {
    const wrapper = await openCodeBuddyStep2()
    await findButtonByText(wrapper, 'generateAuthUrl')!.trigger('click')
    await flushPromises()

    exchangeStateMock.mockResolvedValueOnce({
      access_token: 'at-1',
      refresh_token: 'rt-1',
      nickname: 'Tencent User',
      uid: 'u-1',
    })
    await findButtonByText(wrapper, 'verifyAuthState')!.trigger('click')
    await flushPromises()

    await findButtonByText(wrapper, 'completeAuth')!.trigger('click')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    const payload = createAccountMock.mock.calls[0][0]
    expect(payload.platform).toBe('codebuddy')
    expect(payload.type).toBe('oauth')
    expect(payload.credentials.access_token).toBe('at-1')
    expect(payload.credentials.refresh_token).toBe('rt-1')
  })
})
