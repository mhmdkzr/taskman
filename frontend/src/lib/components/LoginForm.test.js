import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/svelte"
import LoginForm from "./LoginForm.svelte"

vi.mock("$lib/api/loginSession.js", async () => {
  const actual = await vi.importActual("../api/loginSession.js")
  return {
    ...actual,
    createSession: vi.fn(),
    checkPassword: vi.fn(),
    checkTOTP: vi.fn(),
    finalize: vi.fn(),
  }
})

import { createSession, checkPassword, checkTOTP, finalize, LoginSessionError } from "$lib/api/loginSession.js"

describe("LoginForm", () => {
  beforeEach(() => {
    delete window.location
    window.location = { assign: vi.fn(), search: "" }
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it("starts on the login-name step and asks for a password after createSession succeeds", async () => {
    createSession.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false })
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))

    await waitFor(() => expect(screen.getByPlaceholderText("Password")).toBeInTheDocument())
    expect(createSession).toHaveBeenCalledWith("minnie@example.com")
  })

  it("shows an error and stays on the login-name step when the login name is empty", async () => {
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/enter your login name/i)
    expect(createSession).not.toHaveBeenCalled()
  })

  it("shows an error when the login name is rejected", async () => {
    createSession.mockRejectedValueOnce(new LoginSessionError("invalid credentials", 401))
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "nobody@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/couldn't find that account/i)
  })

  it("finalizes and redirects once the password is verified", async () => {
    createSession.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false })
    checkPassword.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: true })
    finalize.mockResolvedValueOnce({ redirectUrl: "/dashboard" })
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("Password")).toBeInTheDocument())

    await fireEvent.input(screen.getByPlaceholderText("Password"), { target: { value: "hunter2" } })
    await fireEvent.click(screen.getByRole("button", { name: /sign in/i }))

    await waitFor(() => expect(finalize).toHaveBeenCalledWith("V2_authreq"))
    expect(checkPassword).toHaveBeenCalledWith("hunter2")
    expect(window.location.assign).toHaveBeenCalledWith("/dashboard")
  })

  it("shows an error and does not finalize when the password is wrong", async () => {
    createSession.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false })
    checkPassword.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false })
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("Password")).toBeInTheDocument())

    await fireEvent.input(screen.getByPlaceholderText("Password"), { target: { value: "wrong" } })
    await fireEvent.click(screen.getByRole("button", { name: /sign in/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/incorrect password/i)
    expect(finalize).not.toHaveBeenCalled()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it("shows a TOTP step when the account has it enrolled, then finalizes", async () => {
    createSession.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false, needsTotp: true })
    checkPassword.mockResolvedValueOnce({
      loginName: "minnie@example.com",
      passwordVerified: true,
      needsTotp: true,
      totpVerified: false,
    })
    checkTOTP.mockResolvedValueOnce({
      loginName: "minnie@example.com",
      passwordVerified: true,
      needsTotp: true,
      totpVerified: true,
    })
    finalize.mockResolvedValueOnce({ redirectUrl: "/dashboard" })
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("Password")).toBeInTheDocument())

    await fireEvent.input(screen.getByPlaceholderText("Password"), { target: { value: "hunter2" } })
    await fireEvent.click(screen.getByRole("button", { name: /sign in/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("6-digit code")).toBeInTheDocument())
    expect(finalize).not.toHaveBeenCalled()

    await fireEvent.input(screen.getByPlaceholderText("6-digit code"), { target: { value: "123456" } })
    await fireEvent.click(screen.getByRole("button", { name: /verify/i }))

    await waitFor(() => expect(finalize).toHaveBeenCalledWith("V2_authreq"))
    expect(checkTOTP).toHaveBeenCalledWith("123456")
    expect(window.location.assign).toHaveBeenCalledWith("/dashboard")
  })

  it("shows an error and does not finalize when the TOTP code is wrong", async () => {
    createSession.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false, needsTotp: true })
    checkPassword.mockResolvedValueOnce({
      loginName: "minnie@example.com",
      passwordVerified: true,
      needsTotp: true,
      totpVerified: false,
    })
    checkTOTP.mockResolvedValueOnce({
      loginName: "minnie@example.com",
      passwordVerified: true,
      needsTotp: true,
      totpVerified: false,
    })
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("Password")).toBeInTheDocument())
    await fireEvent.input(screen.getByPlaceholderText("Password"), { target: { value: "hunter2" } })
    await fireEvent.click(screen.getByRole("button", { name: /sign in/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("6-digit code")).toBeInTheDocument())

    await fireEvent.input(screen.getByPlaceholderText("6-digit code"), { target: { value: "000000" } })
    await fireEvent.click(screen.getByRole("button", { name: /verify/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/incorrect code/i)
    expect(finalize).not.toHaveBeenCalled()
  })

  it("returns to the login-name step from 'use a different account'", async () => {
    createSession.mockResolvedValueOnce({ loginName: "minnie@example.com", passwordVerified: false })
    render(LoginForm, { authRequestId: "V2_authreq" })

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie@example.com" } })
    await fireEvent.click(screen.getByRole("button", { name: /continue/i }))
    await waitFor(() => expect(screen.getByPlaceholderText("Password")).toBeInTheDocument())

    await fireEvent.click(screen.getByRole("button", { name: /use a different account/i }))

    expect(screen.getByPlaceholderText("Login name")).toBeInTheDocument()
  })
})
