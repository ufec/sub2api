import { describe, expect, it } from 'vitest'

import zhAccounts from '../locales/zh/admin/accounts'
import enAccounts from '../locales/en/admin/accounts'
import zhOverview from '../locales/zh/admin/overview'
import enOverview from '../locales/en/admin/overview'

// 临时核查脚本:确保 CodeBuddy/WorkBuddy 相关 key 在 zh/en 均存在
const get = (o: Record<string, any>, path: string) =>
  path.split('.').reduce((a, k) => (a == null ? a : a[k]), o)

const acc = (mod: Record<string, any>, path: string) => get(mod.accounts, path)

describe('codebuddy i18n keys restored', () => {
  const keys = [
    'usageWindow.codebuddy.title',
    'usageWindow.codebuddy.remaining',
    'usageWindow.codebuddy.used',
    'usageWindow.codebuddy.total',
    'usageWindow.codebuddy.accountCount',
    'codebuddy.baseUrlHint',
    'codebuddy.apiKeyHint',
    'oauth.codebuddy.title',
    'oauth.codebuddy.followSteps',
    'oauth.codebuddy.step1GenerateUrl',
    'oauth.codebuddy.generateAuthUrl',
    'oauth.codebuddy.step2OpenUrl',
    'oauth.codebuddy.openUrlDesc',
    'oauth.codebuddy.importantNotice',
    'oauth.codebuddy.step3EnterCode',
    'oauth.codebuddy.authCodeDesc',
    'oauth.codebuddy.authCode',
    'oauth.codebuddy.authCodePlaceholder',
    'oauth.codebuddy.authCodeHint',
    'oauth.codebuddy.verifyAuthState',
    'oauth.codebuddy.verifyingAuthState',
    'oauth.codebuddy.verifyAuthStateHint',
    'oauth.codebuddy.refreshTokenAuth',
    'oauth.codebuddy.refreshTokenDesc',
    'oauth.codebuddy.refreshTokenPlaceholder',
    'oauth.codebuddy.validating',
    'oauth.codebuddy.validateAndCreate',
    'oauth.codebuddy.pleaseEnterRefreshToken',
    'oauth.codebuddy.refreshToken',
    'oauth.codebuddy.failedToGenerateUrl',
    'oauth.codebuddy.missingExchangeParams',
    'oauth.codebuddy.failedToExchangeState',
    'oauth.codebuddy.failedToValidateRT',
    'oauth.codebuddy.authFailed',
    'oauth.codebuddy.oauthOnlyHint',
    'oauth.codebuddy.errors.CODEBUDDY_OAUTH_SESSION_NOT_FOUND',
    'oauth.codebuddy.errors.CODEBUDDY_OAUTH_STATE_REQUIRED',
    'oauth.codebuddy.errors.CODEBUDDY_OAUTH_INVALID_STATE',
    'oauth.codebuddy.errors.CODEBUDDY_OAUTH_NO_REFRESH_TOKEN',
    'oauth.codebuddy.errors.CODEBUDDY_OAUTH_PROXY_NOT_AVAILABLE',
    'oauth.codebuddy.errors.CODEBUDDY_OAUTH_PROXY_NOT_FOUND'
  ]

  it('zh admin/accounts has all codebuddy keys', () => {
    for (const k of keys) {
      expect(acc(zhAccounts, k), `zh missing ${k}`).toBeTruthy()
    }
    expect(acc(zhAccounts, 'usageWindow.codebuddy.title')).toBe('CodeBuddy 额度')
  })

  it('en admin/accounts has all codebuddy keys', () => {
    for (const k of keys) {
      expect(acc(enAccounts, k), `en missing ${k}`).toBeTruthy()
    }
    expect(acc(enAccounts, 'usageWindow.codebuddy.title')).toBe('CodeBuddy Credits')
  })

  it('overview groups.platforms has codebuddy label', () => {
    expect(get(zhOverview, 'groups.platforms.codebuddy')).toBe('CodeBuddy')
    expect(get(enOverview, 'groups.platforms.codebuddy')).toBe('CodeBuddy')
  })
})
