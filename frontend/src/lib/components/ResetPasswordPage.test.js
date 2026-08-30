import { describe, it, expect, vi, afterEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/svelte"
import ResetPasswordPage from "./ResetPasswordPage.svelte"

vi.mock("$lib/api/registration.js", async () => {
  const actual = await vi.importActual("../api/registration.js")
  return { ...actual, resetPassword: vi.fn() }
})

import { resetPassword } from "$lib/api/registration.js"
import { LoginSessionError } from "$lib/api/loginSession.js"

describe("ResetPasswordPage", () => {
  afterEach(() => vi.clearAllMocks())

  it("rejects an empty password", async () => {
    render(ResetPasswordPage, { userId: "user-1", code: "CODE123" })

    await fireEvent.click(screen.getByRole("button", { name: /save new password/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/enter a new password/i)
    expect(resetPassword).not.toHaveBeenCalled()
  })

  it("resets the password and shows success", async () => {
    resetPassword.mockResolvedValueOnce({})
    render(ResetPasswordPage, { userId: "user-1", code: "CODE123" })

    await fireEvent.input(screen.getByPlaceholderText("New password"), { target: { value: "NewPass123!" } })
    await fireEvent.click(screen.getByRole("button", { name: /save new password/i }))

    expect(await screen.findByText(/password has been changed/i)).toBeInTheDocument()
    expect(resetPassword).toHaveBeenCalledWith("user-1", "CODE123", "NewPass123!")
  })

  it("shows an error for an invalid or expired reset code", async () => {
    resetPassword.mockRejectedValueOnce(new LoginSessionError("invalid", 401))
    render(ResetPasswordPage, { userId: "user-1", code: "BADCODE" })

    await fireEvent.input(screen.getByPlaceholderText("New password"), { target: { value: "NewPass123!" } })
    await fireEvent.click(screen.getByRole("button", { name: /save new password/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/invalid or has expired/i)
  })
})
