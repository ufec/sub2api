import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { generateAuthUrlMock, exchangeStateMock, updateAccountMock, clearErrorMock } = vi.hoisted(() => ({
  generateAuthUrlMock: vi.fn(),
  exchangeStateMock: vi.fn(),
  updateAccountMock: vi.fn(),
  clearErrorMock: vi.fn(),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      update: updateAccountMock,
      clearError: clearErrorMock,
    },
    codebuddy: {
      generateAuthUrl: generateAuthUrlMock,
      exchangeState: exchangeStateMock,
      refreshCodeBuddyToken: vi.fn(),
    },
    grok: { getCapabilities: vi.fn().mockResolvedValue({ password_auth_enabled: false }) },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import ReAuthAccountModal from '../ReAuthAccountModal.vue'
import type { Account } from '@/types'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

const codeBuddyAccount = {
  id: 5,
  platform: 'codebuddy',
  type: 'oauth',
  name: 'cb account',
  proxy_id: null,
  credentials: {},
} as unknown as Account

function mountModal(account: Account | null = codeBuddyAccount) {
  return mount(ReAuthAccountModal, {
    props: { show: true, account },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Icon: true,
      },
    },
  })
}

const findButtonByText = (wrapper: ReturnType<typeof mountModal>, text: string) =>
  wrapper.findAll('button').find((candidate) => candidate.text().includes(text))

describe('ReAuthAccountModal CodeBuddy flow', () => {
  beforeEach(() => {
    generateAuthUrlMock.mockReset().mockResolvedValue({
      auth_url: 'https://copilot.tencent.com/auth?state=st-1',
      session_id: 'sid-1',
      state: 'st-1',
    })
    exchangeStateMock.mockReset()
    updateAccountMock.mockReset().mockResolvedValue({ id: 5 })
    clearErrorMock.mockReset().mockResolvedValue({ id: 5 })
  })

  it('unlocks 完成授权 only after a successful 校验认证状态 and applies the stored token', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const verifyBtn = findButtonByText(wrapper, 'verifyAuthState')!
    const completeBtn = findButtonByText(wrapper, 'completeAuth')!

    // Initially and after generating the URL: complete stays disabled
    expect(verifyBtn.attributes('disabled')).toBeDefined()
    expect(completeBtn.attributes('disabled')).toBeDefined()

    await findButtonByText(wrapper, 'generateAuthUrl')!.trigger('click')
    await flushPromises()
    expect(generateAuthUrlMock).toHaveBeenCalledTimes(1)
    expect(verifyBtn.attributes('disabled')).toBeUndefined()
    expect(completeBtn.attributes('disabled')).toBeDefined()

    // Failed verification keeps verify clickable and complete disabled
    exchangeStateMock.mockRejectedValueOnce({ message: 'not authorized' })
    await verifyBtn.trigger('click')
    await flushPromises()
    expect(exchangeStateMock).toHaveBeenCalledTimes(1)
    expect(verifyBtn.attributes('disabled')).toBeUndefined()
    expect(completeBtn.attributes('disabled')).toBeDefined()

    // Successful verification stores the token and unlocks complete
    exchangeStateMock.mockResolvedValueOnce({ access_token: 'at-9', refresh_token: 'rt-9' })
    await verifyBtn.trigger('click')
    await flushPromises()
    expect(updateAccountMock).not.toHaveBeenCalled()
    expect(verifyBtn.attributes('disabled')).toBeDefined()
    expect(completeBtn.attributes('disabled')).toBeUndefined()

    // Complete applies the stored token without exchanging again
    await completeBtn.trigger('click')
    await flushPromises()
    expect(exchangeStateMock).toHaveBeenCalledTimes(2)
    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    const updatePayload = updateAccountMock.mock.calls[0][1]
    expect(updatePayload.type).toBe('oauth')
    expect(updatePayload.credentials.access_token).toBe('at-9')
    expect(updatePayload.credentials.refresh_token).toBe('rt-9')
    expect(wrapper.emitted('reauthorized')).toBeTruthy()
  })
})
