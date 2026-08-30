import { describe, it, expect, vi, afterEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/svelte"
import VerifyEmailPage from "./VerifyEmailPage.svelte"

vi.mock("$lib/api/registration.js", async () => {
  const actual = await vi.importActual("../api/registration.js")
  return {
    ...actual,
    verifyEmail: vi.fn(),
    startTOTPEnrollment: vi.fn(),
    confirmTOTPEnrollment: vi.fn(),
  }
})

import { verifyEmail, startTOTPEnrollment, confirmTOTPEnrollment } from "$lib/api/registration.js"

describe("VerifyEmailPage", () => {
  afterEach(() => vi.clearAllMocks())

  it("verifies on mount and offers to enable 2FA on success", async () => {
    verifyEmail.mockResolvedValueOnce({})
    render(VerifyEmailPage, { userId: "user-1", code: "CODE123" })

    expect(await screen.findByRole("button", { name: /enable two-factor/i })).toBeInTheDocument()
    expect(verifyEmail).toHaveBeenCalledWith("user-1", "CODE123")
  })

  it("shows an error for an invalid or expired code", async () => {
    verifyEmail.mockRejectedValueOnce(new Error("expired"))
    render(VerifyEmailPage, { userId: "user-1", code: "BADCODE" })

    expect(await screen.findByRole("alert")).toHaveTextContent(/invalid or has expired/i)
  })

  it("walks through enabling TOTP after verification", async () => {
    verifyEmail.mockResolvedValueOnce({})
    startTOTPEnrollment.mockResolvedValueOnce({ uri: "otpauth://totp/example", secret: "SECRET123" })
    confirmTOTPEnrollment.mockResolvedValueOnce({})
    render(VerifyEmailPage, { userId: "user-1", code: "CODE123" })

    await fireEvent.click(await screen.findByRole("button", { name: /enable two-factor/i }))
    expect(await screen.findByText("SECRET123")).toBeInTheDocument()

    await fireEvent.input(screen.getByPlaceholderText("6-digit code"), { target: { value: "123456" } })
    await fireEvent.click(screen.getByRole("button", { name: /confirm/i }))

    expect(await screen.findByText(/two-factor authentication is enabled/i)).toBeInTheDocument()
    expect(confirmTOTPEnrollment).toHaveBeenCalledWith("user-1", "123456")
  })
})
