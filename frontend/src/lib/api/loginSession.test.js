import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { createSession, checkPassword, finalize, LoginSessionError } from "./loginSession.js"

function jsonResponse(status, body) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }
}

describe("loginSession API client", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("createSession posts loginName and returns the decoded state", async () => {
    fetch.mockResolvedValueOnce(jsonResponse(201, { loginName: "minnie@example.com", passwordVerified: false }))

    const state = await createSession("minnie@example.com")

    expect(state).toEqual({ loginName: "minnie@example.com", passwordVerified: false })
    const [url, options] = fetch.mock.calls[0]
    expect(url).toBe("/auth/session")
    expect(options.method).toBe("POST")
    expect(options.credentials).toBe("same-origin")
    expect(JSON.parse(options.body)).toEqual({ loginName: "minnie@example.com" })
  })

  it("createSession throws LoginSessionError with the server message on failure", async () => {
    fetch.mockResolvedValueOnce(jsonResponse(401, { error: "invalid credentials" }))

    await expect(createSession("nobody@example.com")).rejects.toMatchObject({
      name: "LoginSessionError",
      message: "invalid credentials",
      status: 401,
    })
  })

  it("createSession rejects with LoginSessionError even without a JSON error body", async () => {
    fetch.mockResolvedValueOnce({ ok: false, status: 500, json: () => Promise.reject(new Error("no body")) })

    await expect(createSession("nobody@example.com")).rejects.toBeInstanceOf(LoginSessionError)
  })

  it("checkPassword PATCHes the password", async () => {
    fetch.mockResolvedValueOnce(jsonResponse(200, { loginName: "minnie@example.com", passwordVerified: true }))

    const state = await checkPassword("hunter2")

    expect(state.passwordVerified).toBe(true)
    const [url, options] = fetch.mock.calls[0]
    expect(url).toBe("/auth/session/password")
    expect(options.method).toBe("PATCH")
    expect(JSON.parse(options.body)).toEqual({ password: "hunter2" })
  })

  it("finalize posts the authRequestId and returns the redirect URL", async () => {
    fetch.mockResolvedValueOnce(jsonResponse(200, { redirectUrl: "/dashboard" }))

    const result = await finalize("V2_authreq")

    expect(result).toEqual({ redirectUrl: "/dashboard" })
    const [url, options] = fetch.mock.calls[0]
    expect(url).toBe("/auth/session/finalize")
    expect(JSON.parse(options.body)).toEqual({ authRequestId: "V2_authreq" })
  })
})
