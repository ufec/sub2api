import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn() }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    grok: { getCapabilities: vi.fn().mockResolvedValue({ password_auth_enabled: false }) },
  },
}))

import OAuthAuthorizationFlow from '../OAuthAuthorizationFlow.vue'

const mountFlow = (props: Record<string, unknown> = {}) =>
  mount(OAuthAuthorizationFlow, {
    props: {
      addMethod: 'oauth',
      platform: 'codebuddy',
      showCookieOption: false,
      showRefreshTokenOption: true,
      initialInputMethod: 'manual',
      ...props,
    },
    global: {
      stubs: {
        Icon: { template: '<i />' },
      },
    },
  })

const findVerifyButton = (wrapper: ReturnType<typeof mount>) =>
  wrapper.findAll('button').find((b) => b.text().includes('verifyAuthState'))

describe('OAuthAuthorizationFlow (codebuddy verify auth state)', () => {
  it('disables the verify button before an auth URL is generated', () => {
    const wrapper = mountFlow({ authUrl: '', initialOauthState: '' })
    const btn = findVerifyButton(wrapper)
    expect(btn).toBeDefined()
    expect(btn!.attributes('disabled')).toBeDefined()
  })

  it('enables the verify button once the state from the generated URL arrives via kebab-case binding', async () => {
    const wrapper = mountFlow({ authUrl: 'https://example.com/authorize?state=abc', initialOauthState: '' })
    const btn = findVerifyButton(wrapper)
    expect(btn!.attributes('disabled')).toBeDefined()

    await wrapper.setProps({ initialOauthState: 'abc' })
    expect(btn!.attributes('disabled')).toBeUndefined()
  })

  it('receives the state through the kebab-case attribute (initial-oauth-state)', () => {
    const wrapper = mountFlow({ initialOauthState: 'st-1' })
    expect(findVerifyButton(wrapper)!.attributes('disabled')).toBeUndefined()
  })

  it('emits verify-auth-state with the state when clicked', async () => {
    const wrapper = mountFlow({ initialOauthState: 'state-123' })
    const btn = findVerifyButton(wrapper)
    await btn!.trigger('click')
    expect(wrapper.emitted('verify-auth-state')).toEqual([['state-123']])
  })

  it('keeps the verify button clickable after a failed verification (loading resets)', async () => {
    const wrapper = mountFlow({ initialOauthState: 'state-123' })
    const btn = findVerifyButton(wrapper)
    await btn!.trigger('click')
    expect(wrapper.emitted('verify-auth-state')).toHaveLength(1)

    await wrapper.setProps({ loading: true })
    expect(btn!.attributes('disabled')).toBeDefined()
    await wrapper.setProps({ loading: false })
    expect(btn!.attributes('disabled')).toBeUndefined()
  })

  it('disables the verify button once the parent marked the state as verified', async () => {
    const wrapper = mountFlow({ initialOauthState: 'state-123' })
    expect(findVerifyButton(wrapper)!.attributes('disabled')).toBeUndefined()

    await wrapper.setProps({ stateVerified: true })
    expect(findVerifyButton(wrapper)!.attributes('disabled')).toBeDefined()
  })
})
