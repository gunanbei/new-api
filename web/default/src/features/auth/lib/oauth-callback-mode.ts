/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const OAUTH_BIND_FLOW_KEY_PREFIX = 'oauth_bind_flow:'

export type OAuthCallbackMode = 'login' | 'bind'

type OAuthModeStorage = {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
}

type OAuthStorageOwner = {
  readonly sessionStorage: OAuthModeStorage
}

type OAuthModeOpener = {
  closed: boolean
}

export function getOAuthSessionStorage(
  owner: OAuthStorageOwner | null | undefined
): OAuthModeStorage | null {
  try {
    return owner?.sessionStorage ?? null
  } catch {
    return null
  }
}

export function markOAuthBindFlow(
  storage: OAuthModeStorage | null | undefined,
  provider: string,
  state: string
): boolean {
  if (!storage || !provider || !state) return false

  try {
    const key = `${OAUTH_BIND_FLOW_KEY_PREFIX}${provider}`
    storage.setItem(key, state)
    return storage.getItem(key) === state
  } catch {
    return false
  }
}

export function resolveOAuthCallbackMode(
  provider: string,
  state: string,
  opener: OAuthModeOpener | null | undefined,
  storage: OAuthModeStorage | null | undefined
): OAuthCallbackMode {
  if (!opener || opener.closed || !storage || !state) return 'login'

  try {
    const markedState = storage.getItem(
      `${OAUTH_BIND_FLOW_KEY_PREFIX}${provider}`
    )
    return markedState === state ? 'bind' : 'login'
  } catch {
    return 'login'
  }
}
