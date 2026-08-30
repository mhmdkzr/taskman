// Test-only helper for provisioning throwaway ZITADEL users via the
// management API, so e2e specs never touch real accounts and clean up after
// themselves. Reads the admin-provisioner PAT from the shared
// zitadel-bootstrap Compose volume, the same one scripts/provision-zitadel-bff.sh
// uses — via the `app` container, since `zitadel`'s own image has no shell
// utilities to `cat` it with.
import { execSync } from 'node:child_process'

const ZITADEL_URL = 'http://localhost:8080'
const REPO_ROOT = new URL('../../../', import.meta.url)

let cachedToken

function adminToken() {
  if (!cachedToken) {
    cachedToken = execSync('docker compose exec -T app cat /zitadel/bootstrap/admin-provisioner.pat', {
      cwd: REPO_ROOT,
    })
      .toString()
      .trim()
  }
  return cachedToken
}

async function zitadelFetch(path, init) {
  const response = await fetch(`${ZITADEL_URL}${path}`, {
    ...init,
    headers: { Authorization: `Bearer ${adminToken()}`, 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!response.ok) {
    throw new Error(`${init?.method ?? 'GET'} ${path} returned ${response.status}: ${await response.text()}`)
  }
  return response.json()
}

/**
 * Creates a throwaway human user with a password already set, for e2e specs
 * to log in as. Returns the ZITADEL user ID (for deleteTestUser).
 * @param {{userName: string, password: string}} params
 */
export async function createTestUser({ userName, password }) {
  const body = await zitadelFetch('/management/v1/users/human/_import', {
    method: 'POST',
    body: JSON.stringify({
      userName,
      profile: { firstName: 'E2E', lastName: 'Test' },
      email: { email: `${userName}@example.com`, isEmailVerified: true },
      password,
      passwordChangeRequired: false,
    }),
  })
  return body.userId
}

/** Deletes a user created by createTestUser. */
export async function deleteTestUser(userId) {
  await zitadelFetch(`/management/v1/users/${userId}`, { method: 'DELETE' })
}
